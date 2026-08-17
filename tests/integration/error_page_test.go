package integration_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// leakySecret — то, что раньше уезжало клиенту: обёрнутая ошибка сервиса с
// именем шаблона, полем структуры или деталями запроса к БД.
const leakySecret = "table budgets: no such column secret_internal_detail"

// customHTTPErrorHandler подставлял err.Error() любой не-*echo.HTTPError ошибки
// прямо в .ErrorMessage страницы ошибки и в текст HTMX-ответа. После того как
// ошибки исполнения шаблонов начали доходить сюда настоящими 500, наружу
// уезжали имена шаблонов, поля структур и обёрнутые ошибки репозиториев.
func TestErrorPage_DoesNotLeakInternalErrorText(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)

	// Маршрут, падающий обычной (не HTTP) ошибкой — так падает исполнение
	// шаблона или запрос к БД внутри хендлера.
	testServer.Server.Echo().GET("/__test_boom", func(_ echo.Context) error {
		return errors.New(leakySecret)
	})

	t.Run("HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/__test_boom", nil)
		auth.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), leakySecret,
			"страница ошибки показывает текст внутренней ошибки")
		assert.Contains(t, rec.Body.String(), "Внутренняя ошибка сервера")
	})

	t.Run("HTMX", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/__test_boom", nil)
		auth.Apply(req)
		req.Header.Set("Hx-Request", "true")
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), leakySecret,
			"HTMX-ответ показывает текст внутренней ошибки")
	})
}
