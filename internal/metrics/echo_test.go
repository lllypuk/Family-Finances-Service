package metrics_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/metrics"
)

// echoWithMetrics повторяет раскладку маршрутов сервера: корневой catch-all,
// публичный маршрут и группа со своим отвергающим middleware.
func echoWithMetrics(m *metrics.Metrics, groupMW echo.MiddlewareFunc) *echo.Echo {
	e := echo.New()
	e.Use(metrics.EchoMiddleware(m.HTTP()))
	e.RouteNotFound("/*", echo.NotFoundHandler)
	e.GET("/health", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	api := e.Group("/api/v1", groupMW)
	api.GET("/transactions/:id", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	return e
}

func passthrough(next echo.HandlerFunc) echo.HandlerFunc {
	return next
}

func TestEchoMiddleware_RouteIsTemplate(t *testing.T) {
	m := testMetrics(t)
	e := echoWithMetrics(m, passthrough)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total", map[string]string{
		"method": http.MethodGet, "route": "/api/v1/transactions/:id", "status": "200",
	}), 0)
}

func TestEchoMiddleware_UnknownRoutesUseCatchAllTemplates(t *testing.T) {
	m := testMetrics(t)
	e := echoWithMetrics(m, passthrough)

	for _, path := range []string{"/api/v1/nope", "/healthz"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, rec.Code, path)
	}

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"route": "/api/v1/*", "status": "404"}), 0)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"route": "/*", "status": "404"}), 0)
}

// Ошибку middleware обработчик ошибок пишет уже после наблюдателя — статус берётся из неё.
func TestEchoMiddleware_StatusFromMiddlewareError(t *testing.T) {
	m := testMetrics(t)
	e := echoWithMetrics(m, func(echo.HandlerFunc) echo.HandlerFunc {
		return func(echo.Context) error { return echo.NewHTTPError(http.StatusUnauthorized) }
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total", map[string]string{
		"route": "/api/v1/transactions/:id", "status": "401",
	}), 0)
}

func TestEchoMiddleware_PlainErrorIsFiveHundred(t *testing.T) {
	m := testMetrics(t)
	e := echoWithMetrics(m, func(echo.HandlerFunc) echo.HandlerFunc {
		return func(echo.Context) error { return errors.New("boom") }
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"status": "500"}), 0)
}

func TestEchoMiddleware_NonStandardMethodIsOther(t *testing.T) {
	m := testMetrics(t)
	e := echoWithMetrics(m, passthrough)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest("PROPFIND", "/healthz", nil))

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"method": "OTHER", "route": "/*"}), 0)
}

func TestEchoMiddleware_InFlightRisesAndReturns(t *testing.T) {
	m := testMetrics(t)
	e := echo.New()
	e.Use(metrics.EchoMiddleware(m.HTTP()))

	var inside float64
	e.GET("/health", func(c echo.Context) error {
		inside = gatheredValue(t, m, "ffs_http_requests_in_flight", nil)
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.InDelta(t, 1.0, inside, 0)
	assert.InDelta(t, 0.0, gatheredValue(t, m, "ffs_http_requests_in_flight", nil), 0)
}
