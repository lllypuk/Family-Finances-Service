package metrics_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/router"
	"family-budget-service/internal/metrics"
)

func passHTTP(next router.Handler) router.Handler { return next }

func okHandler(w http.ResponseWriter, _ *http.Request) error {
	w.WriteHeader(http.StatusOK)

	return nil
}

// routerWithMetrics повторяет раскладку маршрутов сервера: корневой fallback,
// публичный маршрут и группа со своим отвергающим middleware.
func routerWithMetrics(m *metrics.Metrics, groupMW router.Middleware) http.Handler {
	rt := router.New(func(w http.ResponseWriter, _ *http.Request, err error) {
		status := http.StatusInternalServerError
		var he *router.HTTPError
		if errors.As(err, &he) {
			status = he.Status
		}
		w.WriteHeader(status)
	})
	rt.Register(http.MethodGet, "/health", okHandler)
	api := rt.Group("/api/v1", groupMW)
	api.Register(http.MethodGet, "/transactions/:id", okHandler)

	return metrics.HTTPMiddleware(m.HTTP())(rt)
}

func TestHTTPMiddleware_RouteIsTemplate(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, passHTTP)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total", map[string]string{
		"method": http.MethodGet, "route": "/api/v1/transactions/:id", "status": "200",
	}), 0)
}

func TestHTTPMiddleware_UnknownRoutesUseFallbackTemplates(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, passHTTP)

	for _, path := range []string{"/api/v1/nope", "/healthz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, rec.Code, path)
	}

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"route": "/api/v1/*", "status": "404"}), 0)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"route": "/*", "status": "404"}), 0)
}

func TestHTTPMiddleware_WrongMethodKeepsFallbackLabel(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, passHTTP)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/health", nil))

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"route": "/*", "status": "404"}), 0)
}

func TestHTTPMiddleware_StatusFromMiddlewareError(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, func(router.Handler) router.Handler {
		return func(http.ResponseWriter, *http.Request) error {
			return &router.HTTPError{Status: http.StatusUnauthorized}
		}
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total", map[string]string{
		"route": "/api/v1/transactions/:id", "status": "401",
	}), 0)
}

func TestHTTPMiddleware_PlainErrorIsFiveHundred(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, func(router.Handler) router.Handler {
		return func(http.ResponseWriter, *http.Request) error { return errors.New("boom") }
	})

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/transactions/42", nil))

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"status": "500"}), 0)
}

func TestHTTPMiddleware_NonStandardMethodIsOther(t *testing.T) {
	m := testMetrics(t)
	h := routerWithMetrics(m, passHTTP)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PROPFIND", "/healthz", nil))

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"method": "OTHER", "route": "/*"}), 0)
}

func TestHTTPMiddleware_InFlightRisesAndReturns(t *testing.T) {
	m := testMetrics(t)
	var inside float64
	h := metrics.HTTPMiddleware(m.HTTP())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inside = gatheredValue(t, m, "ffs_http_requests_in_flight", nil)
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	assert.InDelta(t, 1.0, inside, 0)
	assert.InDelta(t, 0.0, gatheredValue(t, m, "ffs_http_requests_in_flight", nil), 0)
}
