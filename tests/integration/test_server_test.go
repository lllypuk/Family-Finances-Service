package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// TestSetupHTTPServer_WebLayerRegistered — тест-страж для testhelpers.SetupHTTPServer.
//
// Раньше путь к шаблонам резолвился относительно cwd процесса, поэтому при
// `go test ./tests/integration/` NewWebServer падал, ошибка молча глоталась, и
// интеграционные тесты гоняли приложение БЕЗ SessionStore, CSRFProtection и
// веб-маршрутов (GET /login отдавал 404, POST без токена — 400 вместо 403).
// Именно поэтому дыры S-01 и U-02 прошли мимо тестов.
func TestSetupHTTPServer_WebLayerRegistered(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	require.NoError(t, testServer.Server.WebServerInitError(),
		"web layer must be initialized in the test server")

	// Семья обязана существовать, иначе RequireSetup уводит любой путь на /setup.
	session := testServer.Auth(t)

	t.Run("LoginRouteRegistered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.NotEqual(t, http.StatusNotFound, rec.Code, "web routes are not registered")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("CSRFMiddlewareRejectsWriteWithoutToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code, "CSRF middleware is not registered")
	})

	t.Run("CSRFMiddlewareAcceptsTokenFromLoginAs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// Тело пустое — ждём ошибку валидации, но не отказ CSRF.
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("LoginAsSessionIsAcceptedByProtectedWebRoute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		session.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.NotEqual(t, http.StatusFound, rec.Code, "session cookie was rejected, redirected to /login")
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("SetupRedirectsWhenFamilyExists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/setup", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

// TestSetupHTTPServer_RequireSetupRedirectsWithoutFamily проверяет вторую половину
// контракта: на пустой БД любой путь уводит на /setup.
func TestSetupHTTPServer_RequireSetupRedirectsWithoutFamily(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/setup", rec.Header().Get("Location"))
}
