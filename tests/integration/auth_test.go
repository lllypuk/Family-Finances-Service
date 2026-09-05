package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/testhelpers"
)

const (
	bearerPassword    = "correct-horse-battery"
	bearerNewPassword = "battery-staple-2026"
)

// bearerRequest — запрос к API с bearer-токеном (пустой token — анонимный) и JSON-телом.
func bearerRequest(ts *testhelpers.TestServer, method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Buffer
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewBuffer(raw)
	} else {
		reader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)
	return rec
}

func createBearerMember(t *testing.T, ts *testhelpers.TestServer) *user.User {
	t.Helper()
	ts.Auth(t)

	// MinCost: проверяется путь, а не стоимость хеша (cost 12 — ~250 мс на фикстуру).
	hash, err := bcrypt.GenerateFromPassword([]byte(bearerPassword), bcrypt.MinCost)
	require.NoError(t, err)
	member := testhelpers.CreateTestUser(ts.AuthFamily.ID)
	member.Role = user.RoleMember
	member.Password = string(hash)
	require.NoError(t, ts.Repos.User.Create(t.Context(), member))
	return member
}

func loginBearer(t *testing.T, ts *testhelpers.TestServer, email, password, device string) handlers.LoginResponse {
	t.Helper()
	rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": email, "password": password, "device_name": device,
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body handlers.APIResponse[handlers.LoginResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body.Data.Token)
	return body.Data
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return body.Error.Code
}

// Полный цикл (план 03, задача 4): login → GET /me → смена пароля отзывает вторую сессию,
// текущая жива → logout → 401. Всё через реальный стек, без cookie и CSRF.
func TestAuth_BearerFullCycle(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)

	phone := loginBearer(t, ts, member.Email, bearerPassword, "Pixel 8")
	laptop := loginBearer(t, ts, member.Email, bearerPassword, "Laptop")
	assert.Equal(t, member.ID, phone.User.ID)
	assert.Equal(t, string(user.RoleMember), phone.User.Role)
	assert.False(t, phone.ExpiresAt.IsZero())

	t.Run("GET /me", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodGet, "/api/v1/me", phone.Token, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body handlers.APIResponse[handlers.UserResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, member.Email, body.Data.Email)
	})

	t.Run("PUT /me", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodPut, "/api/v1/me", phone.Token, map[string]string{"first_name": "Renamed"})
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		stored, err := ts.Repos.User.GetByID(t.Context(), member.ID)
		require.NoError(t, err)
		assert.Equal(t, "Renamed", stored.FirstName)
	})

	t.Run("sessions list marks current", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodGet, "/api/v1/auth/sessions", phone.Token, nil)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var body handlers.APIResponse[[]handlers.SessionResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data, 2)
		require.NotNil(t, body.Meta.Pagination)
		assert.Equal(t, 2, body.Meta.Pagination.Total)

		byDevice := map[string]handlers.SessionResponse{}
		for _, s := range body.Data {
			byDevice[s.DeviceName] = s
		}
		assert.True(t, byDevice["Pixel 8"].Current)
		assert.False(t, byDevice["Laptop"].Current)
	})

	t.Run("password change revokes the other session only", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodPut, "/api/v1/me/password", phone.Token, map[string]string{
			"current_password": bearerPassword, "new_password": bearerNewPassword,
		})
		require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

		assert.Equal(t, http.StatusUnauthorized,
			bearerRequest(ts, http.MethodGet, "/api/v1/me", laptop.Token, nil).Code, "вторая сессия отозвана")
		assert.Equal(t, http.StatusOK,
			bearerRequest(ts, http.MethodGet, "/api/v1/me", phone.Token, nil).Code, "текущая сессия жива")

		old := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": member.Email, "password": bearerPassword,
		})
		assert.Equal(t, http.StatusUnauthorized, old.Code, "старый пароль больше не подходит")
		loginBearer(t, ts, member.Email, bearerNewPassword, "Tablet")
	})

	t.Run("wrong current password is 401 and keeps sessions", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodPut, "/api/v1/me/password", phone.Token, map[string]string{
			"current_password": "not-the-password", "new_password": "another-password-1",
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "INVALID_CREDENTIALS", errorCode(t, rec))
		assert.Equal(t, http.StatusOK, bearerRequest(ts, http.MethodGet, "/api/v1/me", phone.Token, nil).Code)
	})

	t.Run("logout kills the token", func(t *testing.T) {
		require.Equal(t, http.StatusNoContent,
			bearerRequest(ts, http.MethodPost, "/api/v1/auth/logout", phone.Token, nil).Code)

		rec := bearerRequest(ts, http.MethodGet, "/api/v1/me", phone.Token, nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "UNAUTHORIZED", errorCode(t, rec))
	})
}

func TestAuth_Login_Errors(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)

	t.Run("wrong password and unknown email look the same", func(t *testing.T) {
		wrong := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": member.Email, "password": "wrong-password-1",
		})
		unknown := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "nobody@example.com", "password": "wrong-password-1",
		})

		assert.Equal(t, http.StatusUnauthorized, wrong.Code)
		assert.Equal(t, http.StatusUnauthorized, unknown.Code)
		assert.Equal(t, "INVALID_CREDENTIALS", errorCode(t, wrong))
		assert.Equal(t, errorCode(t, wrong), errorCode(t, unknown))
	})

	t.Run("validation is 422", func(t *testing.T) {
		rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": "not-an-email", "password": bearerPassword,
		})
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Equal(t, "VALIDATION_ERROR", errorCode(t, rec))
	})

	t.Run("protected routes without valid token are 401", func(t *testing.T) {
		for _, route := range []struct{ method, path, token string }{
			{http.MethodGet, "/api/v1/auth/sessions", ""},
			{http.MethodGet, "/api/v1/me", ""},
			{http.MethodPost, "/api/v1/auth/logout", "garbage"},
			{http.MethodGet, "/api/v1/auth/sessions", "garbage"},
			{http.MethodDelete, "/api/v1/auth/sessions/" + uuid.NewString(), "garbage"},
			{http.MethodGet, "/api/v1/me", "garbage"},
			{http.MethodPut, "/api/v1/me", "garbage"},
			{http.MethodPut, "/api/v1/me/password", "garbage"},
		} {
			rec := bearerRequest(ts, route.method, route.path, route.token, nil)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s: %s", route.method, route.path, rec.Body.String())
		}
	})

	t.Run("revoke unknown session is 404", func(t *testing.T) {
		login := loginBearer(t, ts, member.Email, bearerPassword, "Pixel 8")

		rec := bearerRequest(ts, http.MethodDelete, "/api/v1/auth/sessions/"+uuid.NewString(), login.Token, nil)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "SESSION_NOT_FOUND", errorCode(t, rec))
	})
}

// Лимитер на полном стеке: httptest даёт всем запросам один RemoteAddr, поэтому 11-я попытка — 429.
func TestAuth_Login_RateLimited(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)

	for range auth.IPLimit {
		rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"email": member.Email, "password": "wrong-password-1",
		})
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}

	rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": member.Email, "password": bearerPassword,
	})
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "RATE_LIMITED", errorCode(t, rec))
	assert.NotEmpty(t, rec.Header().Get(echo.HeaderRetryAfter))
}

// X-Forwarded-For принимается только от TRUSTED_PROXIES: без них подмена заголовка не даёт
// новых «IP» лимитеру, с ними — каждое значение считается отдельным клиентом.
func TestAuth_Login_XForwardedFor(t *testing.T) {
	spoofed := func(ts *testhelpers.TestServer, email string, i int) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(
			`{"email":"`+email+`","password":"wrong-password-1"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderXForwardedFor, "203.0.113."+strconv.Itoa(i+1))
		rec := httptest.NewRecorder()
		ts.Server.Echo().ServeHTTP(rec, req)
		return rec
	}

	t.Run("ignored without trusted proxies", func(t *testing.T) {
		ts := testhelpers.SetupHTTPServer(t)
		member := createBearerMember(t, ts)

		for i := range auth.IPLimit {
			require.Equal(t, http.StatusUnauthorized, spoofed(ts, member.Email, i).Code)
		}
		assert.Equal(t, http.StatusTooManyRequests, spoofed(ts, member.Email, auth.IPLimit).Code,
			"подменённый X-Forwarded-For не должен обходить лимит по IP")
	})

	t.Run("honoured from a trusted proxy", func(t *testing.T) {
		// httptest.NewRequest ставит RemoteAddr 192.0.2.1 — объявляем эту сеть доверенным прокси.
		ts := testhelpers.SetupHTTPServer(t, testhelpers.WithTrustedProxies(t, "192.0.2.0/24"))
		member := createBearerMember(t, ts)

		for i := range auth.IPLimit + 1 {
			assert.Equal(t, http.StatusUnauthorized, spoofed(ts, member.Email, i).Code,
				"каждый X-Forwarded-For — отдельный клиент")
		}
	})
}

// Истёкшая сессия на настоящей SQLite: RFC3339 туда-обратно, сравнение с now и удаление строки.
func TestAuth_ExpiredSession_IsRejectedAndDeleted(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)

	plain, hash := auth.GenerateToken()
	stale := auth.NewSession(member.ID, hash, "old phone", time.Now().Add(-auth.IdleTTL-time.Hour))
	require.NoError(t, ts.Repos.Session.Create(t.Context(), stale))
	require.False(t, stale.ExpiresAt.After(time.Now()))

	rec := bearerRequest(ts, http.MethodGet, "/api/v1/me", plain, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Equal(t, "UNAUTHORIZED", errorCode(t, rec))

	left, err := ts.Repos.Session.ListByUser(t.Context(), member.ID)
	require.NoError(t, err)
	assert.Empty(t, left, "истёкшая сессия удалена при первом же обращении")
}

// Адрес, который репозиторий отвергает как подозрительный («update» в локальной части),
// проходит тег email хендлера — ответ обязан быть 401, а не 500.
func TestAuth_Login_SuspiciousEmail_Is401(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	createBearerMember(t, ts)

	rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "updates@gmail.com", "password": bearerPassword,
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Equal(t, "INVALID_CREDENTIALS", errorCode(t, rec))
}

// До setup: /health отдаёт setup_complete=false, логин — 409 SETUP_REQUIRED, а не редирект на /setup.
func TestAuth_BeforeSetup(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)

	health := bearerRequest(ts, http.MethodGet, "/health", "", nil)
	require.Equal(t, http.StatusOK, health.Code)
	var status observability.HealthStatus
	require.NoError(t, json.Unmarshal(health.Body.Bytes(), &status))
	assert.False(t, status.SetupComplete)

	rec := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": "admin@example.com", "password": bearerPassword,
	})
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, "SETUP_REQUIRED", errorCode(t, rec))

	ts.Auth(t)
	health = bearerRequest(ts, http.MethodGet, "/health", "", nil)
	require.NoError(t, json.Unmarshal(health.Body.Bytes(), &status))
	assert.True(t, status.SetupComplete)
}
