package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
)

const (
	fakeLoginEmail    = "member@example.com"
	fakeLoginPassword = "correct-horse-battery"
	fakeLoginToken    = "issued-token"
	fakeDevice        = "Pixel 8"
)

// fakeAuthService — auth.Service без БД: пароль сверяется со строкой, сессии лежат в срезе.
type fakeAuthService struct {
	user       *user.User
	session    *auth.Session
	sessions   []*auth.Session
	loginErr   error
	logoutErr  error
	listErr    error
	revokeErr  error
	changeErr  error
	revoked    []uuid.UUID
	loggedOut  []uuid.UUID
	changeCall *changePasswordCall
	adminErr   error
	adminSet   *adminSetPasswordCall
}

type adminSetPasswordCall struct {
	userID uuid.UUID
	next   string
}

type changePasswordCall struct {
	userID  uuid.UUID
	current string
	next    string
	keep    uuid.UUID
}

func newFakeAuthService() *fakeAuthService {
	u := &user.User{
		ID:        uuid.New(),
		Email:     fakeLoginEmail,
		FirstName: "Jane",
		LastName:  "Doe",
		Role:      user.RoleMember,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	now := time.Now()
	return &fakeAuthService{
		user:    u,
		session: auth.NewSession(u.ID, auth.HashToken(fakeLoginToken), fakeDevice, now),
	}
}

func (f *fakeAuthService) Login(_ context.Context, email, password, device string) (*auth.LoginResult, error) {
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	if email != f.user.Email || password != fakeLoginPassword {
		return nil, auth.ErrInvalidCredentials
	}
	f.session.DeviceName = device
	return &auth.LoginResult{Token: fakeLoginToken, Session: f.session, User: f.user}, nil
}

func (f *fakeAuthService) Logout(_ context.Context, sessionID uuid.UUID) error {
	f.loggedOut = append(f.loggedOut, sessionID)
	return f.logoutErr
}

func (f *fakeAuthService) ListSessions(_ context.Context, userID uuid.UUID) ([]*auth.Session, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*auth.Session
	for _, s := range f.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeAuthService) RevokeSession(_ context.Context, userID, sessionID uuid.UUID) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	for _, s := range f.sessions {
		if s.ID == sessionID && s.UserID == userID {
			f.revoked = append(f.revoked, sessionID)
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (f *fakeAuthService) ChangePassword(
	_ context.Context, userID uuid.UUID, current, next string, keep uuid.UUID,
) error {
	f.changeCall = &changePasswordCall{userID: userID, current: current, next: next, keep: keep}
	return f.changeErr
}

func (f *fakeAuthService) AdminSetPassword(_ context.Context, userID uuid.UUID, next string) error {
	f.adminSet = &adminSetPasswordCall{userID: userID, next: next}
	return f.adminErr
}

func newAuthHandler(svc *fakeAuthService) *handlers.AuthHandler {
	return handlers.NewAuthHandler(svc, auth.NewRateLimiter(nil), slog.New(slog.DiscardHandler))
}

// principalContext — контекст запроса с principal, как его кладёт auth.RequireBearer.
func principalContext(method, path string, body string, p *auth.Principal) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if p != nil {
		c.Set(auth.ContextKey, p)
	}
	return c, rec
}

func principalFor(svc *fakeAuthService) *auth.Principal {
	return &auth.Principal{
		SessionID: svc.session.ID,
		UserID:    svc.user.ID,
		Email:     svc.user.Email,
		Role:      svc.user.Role,
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) handlers.ErrorResponse {
	t.Helper()
	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body
}

func loginBody(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `","device_name":"` + fakeDevice + `"}`
}

// loginFrom — POST /auth/login с указанного RemoteAddr; лимитер по IP считает именно его.
func loginFrom(t *testing.T, h *handlers.AuthHandler, ip, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(email, password), nil)
	c.Request().RemoteAddr = ip + ":4242"
	require.NoError(t, h.Login(c))
	return rec
}

func ipN(n int) string {
	return "10.0." + strconv.Itoa(n/256) + "." + strconv.Itoa(n%256)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := newFakeAuthService()
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(fakeLoginEmail, fakeLoginPassword), nil)

	require.NoError(t, newAuthHandler(svc).Login(c))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body handlers.APIResponse[handlers.LoginResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, fakeLoginToken, body.Data.Token)
	assert.Equal(t, svc.user.ID, body.Data.User.ID)
	assert.Equal(t, string(user.RoleMember), body.Data.User.Role)
	assert.True(t, body.Data.ExpiresAt.Equal(svc.session.ExpiresAt))
	assert.Equal(t, fakeDevice, svc.session.DeviceName)
}

func TestAuthHandler_Login_EmailNormalized(t *testing.T) {
	svc := newFakeAuthService()
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login",
		loginBody("Member@Example.COM", fakeLoginPassword), nil)

	require.NoError(t, newAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	svc := newFakeAuthService()
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(fakeLoginEmail, "wrong-password"), nil)

	require.NoError(t, newAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_CREDENTIALS", decodeError(t, rec).Error.Code)
}

func TestAuthHandler_Login_SetupRequired(t *testing.T) {
	svc := newFakeAuthService()
	svc.loginErr = auth.ErrSetupRequired
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(fakeLoginEmail, fakeLoginPassword), nil)

	require.NoError(t, newAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "SETUP_REQUIRED", decodeError(t, rec).Error.Code)
}

func TestAuthHandler_Login_StorageFailure(t *testing.T) {
	svc := newFakeAuthService()
	svc.loginErr = errors.New("db down")
	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(fakeLoginEmail, fakeLoginPassword), nil)

	require.NoError(t, newAuthHandler(svc).Login(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec).Error.Code)
}

func TestAuthHandler_Login_Validation(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		code  int
		field string
	}{
		{name: "broken json", body: `{`, code: http.StatusBadRequest},
		{name: "bad email", body: loginBody("not-an-email", fakeLoginPassword), code: http.StatusUnprocessableEntity,
			field: "email"},
		{name: "short password is a credential check, not policy", body: loginBody(fakeLoginEmail, "short"),
			code: http.StatusUnauthorized},
		{name: "long device", body: `{"email":"` + fakeLoginEmail + `","password":"` + fakeLoginPassword +
			`","device_name":"` + strings.Repeat("x", 65) + `"}`, code: http.StatusUnprocessableEntity,
			field: "device_name"},
		{name: "device of 64 is accepted", body: `{"email":"` + fakeLoginEmail + `","password":"` +
			fakeLoginPassword + `","device_name":"` + strings.Repeat("x", 64) + `"}`, code: http.StatusOK},
		{name: "password of 72 bytes passes validation", body: loginBody(fakeLoginEmail, strings.Repeat("p", 72)),
			code: http.StatusUnauthorized},
		{name: "password of 73 bytes", body: loginBody(fakeLoginEmail, strings.Repeat("p", 73)),
			code: http.StatusUnprocessableEntity, field: "password"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newFakeAuthService()
			c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", tc.body, nil)

			require.NoError(t, newAuthHandler(svc).Login(c))

			assert.Equal(t, tc.code, rec.Code, rec.Body.String())
			if tc.field != "" {
				body := decodeError(t, rec)
				require.Len(t, body.Error.Details, 1)
				assert.Equal(t, tc.field, body.Error.Details[0].Field)
			}
		})
	}
}

// Лимитер: 11-я попытка с того же IP — 429 с Retry-After на всё окно.
func TestAuthHandler_Login_RateLimited(t *testing.T) {
	svc := newFakeAuthService()
	handler := newAuthHandler(svc)

	for range auth.IPLimit {
		c, rec := principalContext(
			http.MethodPost,
			"/api/v1/auth/login",
			loginBody(fakeLoginEmail, "wrong-password"),
			nil,
		)
		require.NoError(t, handler.Login(c))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}

	c, rec := principalContext(http.MethodPost, "/api/v1/auth/login", loginBody(fakeLoginEmail, fakeLoginPassword), nil)
	require.NoError(t, handler.Login(c))

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "RATE_LIMITED", decodeError(t, rec).Error.Code)
	assert.Equal(t, strconv.Itoa(int(auth.IPWindow.Seconds())), rec.Header().Get(echo.HeaderRetryAfter))
}

// Успешный вход сбрасывает счётчик email (иначе владелец, перебравший пароли с разных
// устройств, остался бы заблокирован на час), но не счётчик IP.
func TestAuthHandler_Login_SuccessResetsEmailCounterOnly(t *testing.T) {
	svc := newFakeAuthService()
	handler := newAuthHandler(svc)

	for i := range auth.EmailLimit - 1 {
		require.Equal(t, http.StatusUnauthorized,
			loginFrom(t, handler, ipN(i), fakeLoginEmail, "wrong-password").Code)
	}
	require.Equal(t, http.StatusOK, loginFrom(t, handler, ipN(100), fakeLoginEmail, fakeLoginPassword).Code)

	rec := loginFrom(t, handler, ipN(101), fakeLoginEmail, "wrong-password")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "после успеха счётчик email пуст")

	const sameIP = "192.0.2.50"
	for range auth.IPLimit - 1 {
		loginFrom(t, handler, sameIP, "other@example.com", "wrong-password")
	}
	require.Equal(t, http.StatusOK, loginFrom(t, handler, sameIP, fakeLoginEmail, fakeLoginPassword).Code)
	rec = loginFrom(t, handler, sameIP, fakeLoginEmail, fakeLoginPassword)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "успех не сбрасывает счётчик IP")
}

// Ключ лимитера по email — после приведения к нижнему регистру, иначе регистр обходил бы лимит.
func TestAuthHandler_Login_EmailLimitIgnoresCase(t *testing.T) {
	svc := newFakeAuthService()
	handler := newAuthHandler(svc)
	variants := []string{"Member@Example.COM", "member@example.com"}

	for i := range auth.EmailLimit {
		require.Equal(t, http.StatusUnauthorized,
			loginFrom(t, handler, ipN(i), variants[i%2], "wrong-password").Code)
	}

	rec := loginFrom(t, handler, ipN(200), "MEMBER@example.com", fakeLoginPassword)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("current session removed", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPost, "/api/v1/auth/logout", "", principalFor(svc))

		require.NoError(t, newAuthHandler(svc).Logout(c))

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, []uuid.UUID{svc.session.ID}, svc.loggedOut)
	})

	t.Run("already revoked is still 204", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.logoutErr = auth.ErrSessionNotFound
		c, rec := principalContext(http.MethodPost, "/api/v1/auth/logout", "", principalFor(svc))

		require.NoError(t, newAuthHandler(svc).Logout(c))

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("no principal", func(t *testing.T) {
		svc := newFakeAuthService()
		c, rec := principalContext(http.MethodPost, "/api/v1/auth/logout", "", nil)

		require.NoError(t, newAuthHandler(svc).Logout(c))

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("storage failure is 500", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.logoutErr = errors.New("db down")
		c, rec := principalContext(http.MethodPost, "/api/v1/auth/logout", "", principalFor(svc))

		require.NoError(t, newAuthHandler(svc).Logout(c))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec).Error.Code)
	})
}

func TestAuthHandler_ListSessions(t *testing.T) {
	svc := newFakeAuthService()
	other := auth.NewSession(svc.user.ID, "other-hash", "Laptop", time.Now().Add(-time.Hour))
	foreign := auth.NewSession(uuid.New(), "foreign-hash", "Stranger", time.Now())
	svc.sessions = []*auth.Session{svc.session, other, foreign}

	c, rec := principalContext(http.MethodGet, "/api/v1/auth/sessions", "", principalFor(svc))
	require.NoError(t, newAuthHandler(svc).ListSessions(c))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body handlers.APIResponse[[]handlers.SessionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 2, "чужая сессия не видна")
	require.NotNil(t, body.Meta.Pagination)
	assert.Equal(t, 2, body.Meta.Pagination.Total)

	current := map[uuid.UUID]bool{}
	for _, s := range body.Data {
		current[s.ID] = s.Current
	}
	assert.True(t, current[svc.session.ID])
	assert.False(t, current[other.ID])
}

func TestAuthHandler_ListSessions_BadLimit(t *testing.T) {
	svc := newFakeAuthService()
	c, rec := principalContext(http.MethodGet, "/api/v1/auth/sessions?limit=0", "", principalFor(svc))

	require.NoError(t, newAuthHandler(svc).ListSessions(c))

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAuthHandler_ListSessions_StorageFailure(t *testing.T) {
	svc := newFakeAuthService()
	svc.listErr = errors.New("db down")
	c, rec := principalContext(http.MethodGet, "/api/v1/auth/sessions", "", principalFor(svc))

	require.NoError(t, newAuthHandler(svc).ListSessions(c))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec).Error.Code)
}

func TestAuthHandler_RevokeSession(t *testing.T) {
	svc := newFakeAuthService()
	other := auth.NewSession(svc.user.ID, "other-hash", "Laptop", time.Now())
	foreign := auth.NewSession(uuid.New(), "foreign-hash", "Stranger", time.Now())
	svc.sessions = []*auth.Session{svc.session, other, foreign}
	handler := newAuthHandler(svc)

	revoke := func(id string) *httptest.ResponseRecorder {
		c, rec := principalContext(http.MethodDelete, "/api/v1/auth/sessions/"+id, "", principalFor(svc))
		c.SetParamNames("id")
		c.SetParamValues(id)
		require.NoError(t, handler.RevokeSession(c))
		return rec
	}

	t.Run("own session", func(t *testing.T) {
		assert.Equal(t, http.StatusNoContent, revoke(other.ID.String()).Code)
		assert.Equal(t, []uuid.UUID{other.ID}, svc.revoked)
	})

	t.Run("foreign session is 404", func(t *testing.T) {
		rec := revoke(foreign.ID.String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "SESSION_NOT_FOUND", decodeError(t, rec).Error.Code)
	})

	t.Run("bad id", func(t *testing.T) {
		rec := revoke("not-a-uuid")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "INVALID_ID", decodeError(t, rec).Error.Code)
	})

	t.Run("storage failure is 500", func(t *testing.T) {
		svc.revokeErr = errors.New("db down")
		rec := revoke(other.ID.String())
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "INTERNAL_ERROR", decodeError(t, rec).Error.Code)
	})
}

// Ответ логина не должен зависеть от того, есть ли тело: пустой JSON — 422, не 500.
func TestAuthHandler_Login_EmptyBody(t *testing.T) {
	svc := newFakeAuthService()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	require.NoError(t, newAuthHandler(svc).Login(e.NewContext(req, rec)))

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
