package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/auth"
)

// AuthService — часть auth.Service, нужная хендлерам /auth/* и /me/password.
type AuthService interface {
	Login(ctx context.Context, email, password, device string) (*auth.LoginResult, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID) ([]*auth.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword string, keepSessionID uuid.UUID) error
	AdminSetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error
}

// LoginObserver считает попытки входа по исходу; интерфейс объявлен здесь,
// чтобы handlers не зависел от пакета метрик.
type LoginObserver interface {
	ObserveLogin(outcome string)
}

// NopLoginObserver — заглушка для сборок без реестра метрик.
type NopLoginObserver struct{}

func (NopLoginObserver) ObserveLogin(string) {}

// Исходы входа — значения метки outcome у ffs_login_attempts_total.
const (
	loginOutcomeOK                 = "ok"
	loginOutcomeInvalidCredentials = "invalid_credentials" //nolint:gosec // G101 — значение метки, не секрет
	loginOutcomeRateLimited        = "rate_limited"
	loginOutcomeSetupRequired      = "setup_required"
	loginOutcomeInvalidRequest     = "invalid_request"
	loginOutcomeError              = "error"
)

// AuthHandler — выдача и отзыв bearer-токенов.
type AuthHandler struct {
	authService AuthService
	limiter     *auth.RateLimiter
	logger      *slog.Logger
	validator   *validator.Validate
	observer    LoginObserver
	clientIP    func(*http.Request) string
}

func NewAuthHandler(
	authService AuthService,
	limiter *auth.RateLimiter,
	logger *slog.Logger,
	observer LoginObserver,
	clientIP func(*http.Request) string,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		limiter:     limiter,
		logger:      logger,
		validator:   newAPIValidator(),
		observer:    observer,
		clientIP:    clientIP,
	}
}

// Login — публичный роут: лимитер → пароль → токен. Ключ лимитера по email — после нормализации,
// иначе регистр букв обходил бы лимит.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		h.observer.ObserveLogin(loginOutcomeInvalidRequest)
		return respondBindError(w, err)
	}
	if err := h.validator.Struct(&req); err != nil {
		h.observer.ObserveLogin(loginOutcomeInvalidRequest)
		return respondValidationErrors(w, err)
	}

	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))
	ip := h.clientIP(r)

	if retryAfter, ok := h.limiter.Allow(ip, email); !ok {
		h.logLoginFailure(ctx, email, ip, loginOutcomeRateLimited)
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		return respondError(w, http.StatusTooManyRequests, ErrCodeRateLimited, ErrMessageRateLimited)
	}

	result, err := h.authService.Login(ctx, email, req.Password, req.DeviceName)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrSetupRequired):
			h.logLoginFailure(ctx, email, ip, loginOutcomeSetupRequired)
			return respondError(w, http.StatusConflict, ErrCodeSetupRequired, ErrMessageSetupRequired)
		case errors.Is(err, auth.ErrInvalidCredentials):
			h.logLoginFailure(ctx, email, ip, loginOutcomeInvalidCredentials)
			return respondError(w, http.StatusUnauthorized, ErrCodeInvalidCredentials, ErrMessageInvalidCredentials)
		default:
			h.observer.ObserveLogin(loginOutcomeError)
			h.logger.ErrorContext(ctx, "login error",
				slog.String("email", email), slog.String("ip", ip), slog.String("error", err.Error()))
			return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
		}
	}

	h.limiter.Reset(email)
	h.observer.ObserveLogin(loginOutcomeOK)

	return respondAPI(w, http.StatusOK, LoginResponse{
		Token:     result.Token,
		ExpiresAt: result.Session.ExpiresAt,
		User:      toUserResponse(result.User),
	})
}

// logLoginFailure пишет лог и считает исход: reason — это и значение метки outcome.
func (h *AuthHandler) logLoginFailure(ctx context.Context, email, ip, reason string) {
	h.observer.ObserveLogin(reason)
	h.logger.WarnContext(ctx, "login failed",
		slog.String("email", email), slog.String("ip", ip), slog.String("reason", reason))
}

// Logout отзывает сессию текущего токена; уже отозванная — тоже 204.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	if err = h.authService.Logout(r.Context(), principal.SessionID); err != nil &&
		!errors.Is(err, auth.ErrSessionNotFound) {
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return noContent(w)
}

// ListSessions — сессии владельца токена, текущая помечена current.
func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	sessions, err := h.authService.ListSessions(r.Context(), principal.UserID)
	if err != nil {
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	response := make([]SessionResponse, 0, page.Limit)
	for _, s := range pageSlice(sessions, page) {
		response = append(response, SessionResponse{
			ID:         s.ID,
			DeviceName: s.DeviceName,
			CreatedAt:  s.CreatedAt,
			LastUsedAt: s.LastUsedAt,
			ExpiresAt:  s.ExpiresAt,
			Current:    s.ID == principal.SessionID,
		})
	}

	return respondList(w, response, page, len(sessions))
}

// RevokeSession удаляет свою сессию; чужая неотличима от несуществующей — 404.
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, "Invalid session ID format")
	}

	if err = h.authService.RevokeSession(r.Context(), principal.UserID, id); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return handleNotFoundError(w, "Session")
		}
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return noContent(w)
}
