package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/observability"
	"family-budget-service/internal/testhelpers"
	"family-budget-service/internal/version"
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
	webSession := webLoginAs(t, testServer, testServer.AuthUser)

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

	t.Run("BearerFromLoginAsAcceptedByAPI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// Тело пустое — ждём ошибку валидации, а не 401 и не отказ CSRF.
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	})

	t.Run("BearerIsIgnoredByProtectedWebRoute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		session.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusFound, rec.Code, "веб-маршрут принял bearer-токен вместо cookie")
	})

	t.Run("WebSessionIsAcceptedByProtectedWebRoute", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		webSession.Apply(req)
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

// TestSetupPage_RendersOnEmptyDatabase — критерий приёмки U-01: на чистой БД
// /setup отдаёт страницу, а не редирект, и её статика доступна.
// Раньше путь к статике был зашит относительно рабочего каталога процесса
// (`internal/web/static`), поэтому интеграционный сервер её вообще не отдавал
// и регрессию U-01 нечем было поймать.
func TestSetupPage_RendersOnEmptyDatabase(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("SetupPageRendered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/setup", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), "<form")
	})

	t.Run("StaticAssetsServedWithoutFamily", func(t *testing.T) {
		for _, asset := range []string{
			"/static/css/pico.min.css",
			"/static/css/custom.css",
			"/static/js/htmx.min.js",
		} {
			req := httptest.NewRequest(http.MethodGet, asset, nil)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "GET %s отдал %d", asset, rec.Code)
		}
	})

	t.Run("HealthIsPublic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var body observability.HealthStatus
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, version.String(), body.Version)
		assert.Equal(t, observability.HealthStatusHealthy, body.Status)
		assert.False(t, body.Timestamp.IsZero())
		assert.False(t, body.SetupComplete, "семьи ещё нет — setup_complete обязан быть false")
	})
}
