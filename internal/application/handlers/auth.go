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
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/auth"
)

// AuthService — часть auth.Service, нужная хендлерам /auth/* и /me/password.
type AuthService interface {
	Login(ctx context.Context, email, password, device string) (*auth.LoginResult, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID) ([]*auth.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword string, keepSessionID uuid.UUID) error
}

// AuthHandler — выдача и отзыв bearer-токенов.
type AuthHandler struct {
	authService AuthService
	limiter     *auth.RateLimiter
	logger      *slog.Logger
	validator   *validator.Validate
}

func NewAuthHandler(authService AuthService, limiter *auth.RateLimiter, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		limiter:     limiter,
		logger:      logger,
		validator:   newAPIValidator(),
	}
}

// Login — публичный роут: лимитер → пароль → токен. Ключ лимитера по email — после нормализации,
// иначе регистр букв обходил бы лимит.
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return HandleBindError(c)
	}
	if err := h.validator.Struct(&req); err != nil {
		return respondValidationErrors(c, err)
	}

	ctx := c.Request().Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))
	ip := c.RealIP()

	if retryAfter, ok := h.limiter.Allow(ip, email); !ok {
		h.logLoginFailure(ctx, email, ip, "rate_limited")
		c.Response().Header().Set(echo.HeaderRetryAfter, strconv.Itoa(int(retryAfter.Seconds())))
		return respondError(c, http.StatusTooManyRequests, ErrCodeRateLimited, ErrMessageRateLimited)
	}

	result, err := h.authService.Login(ctx, email, req.Password, req.DeviceName)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrSetupRequired):
			h.logLoginFailure(ctx, email, ip, "setup_required")
			return respondError(c, http.StatusConflict, ErrCodeSetupRequired, ErrMessageSetupRequired)
		case errors.Is(err, auth.ErrInvalidCredentials):
			h.logLoginFailure(ctx, email, ip, "invalid_credentials")
			return respondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, ErrMessageInvalidCredentials)
		default:
			h.logger.ErrorContext(ctx, "login failed", slog.String("email", email), slog.String("error", err.Error()))
			return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
		}
	}

	h.limiter.Reset(email)

	return respondAPI(c, http.StatusOK, LoginResponse{
		Token:     result.Token,
		ExpiresAt: result.Session.ExpiresAt,
		User:      toUserResponse(result.User),
	})
}

func (h *AuthHandler) logLoginFailure(ctx context.Context, email, ip, reason string) {
	h.logger.WarnContext(ctx, "login failed",
		slog.String("email", email), slog.String("ip", ip), slog.String("reason", reason))
}

// Logout отзывает сессию текущего токена; уже отозванная — тоже 204.
func (h *AuthHandler) Logout(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	if err = h.authService.Logout(c.Request().Context(), principal.SessionID); err != nil &&
		!errors.Is(err, auth.ErrSessionNotFound) {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return c.NoContent(http.StatusNoContent)
}

// ListSessions — сессии владельца токена, текущая помечена current.
func (h *AuthHandler) ListSessions(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	page, pageErr := parsePagination(c)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	sessions, err := h.authService.ListSessions(c.Request().Context(), principal.UserID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
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

	return respondList(c, response, page, len(sessions))
}

// RevokeSession удаляет свою сессию; чужая неотличима от несуществующей — 404.
func (h *AuthHandler) RevokeSession(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, "Invalid session ID format")
	}

	if err = h.authService.RevokeSession(c.Request().Context(), principal.UserID, id); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return HandleNotFoundError(c, "Session")
		}
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return c.NoContent(http.StatusNoContent)
}
