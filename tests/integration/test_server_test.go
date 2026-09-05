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

// TestSetupHTTPServer_BearerAccepted — тест-страж для testhelpers.SetupHTTPServer:
// токен LoginAs принимает настоящий RequireBearer, а не тестовая подмена.
func TestSetupHTTPServer_BearerAccepted(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	t.Run("BearerFromLoginAsAcceptedByAPI", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// Тело пустое — ждём ошибку валидации, а не 401.
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	})

	t.Run("WriteWithoutTokenIs401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	})
}

// TestSetupHTTPServer_EmptyDatabase — до CLI `setup` сервис отвечает на /health
// и честно сообщает setup_complete: false; веб-редиректов на /setup больше нет.
func TestSetupHTTPServer_EmptyDatabase(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

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

	t.Run("FormerWebRoutesAre404JSON", func(t *testing.T) {
		for _, path := range []string{"/", "/setup", "/login", "/static/css/pico.min.css"} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code, "GET %s отдал %d", path, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json", path)
		}
	})
}
