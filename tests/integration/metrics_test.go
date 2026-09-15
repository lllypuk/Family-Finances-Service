package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// scrape снимает /metrics того же реестра, что считает запросы тестового сервера.
func scrape(t *testing.T, ts *testhelpers.TestServer) string {
	t.Helper()

	rec := httptest.NewRecorder()
	ts.Metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	return rec.Body.String()
}

func TestMetrics_ScrapeAfterRequests(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)

	loginBearer(t, ts, member.Email, bearerPassword, "metrics")
	wrong := bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": member.Email, "password": "definitely-not-the-password", "device_name": "metrics",
	})
	require.Equal(t, http.StatusUnauthorized, wrong.Code)

	require.Equal(t, http.StatusOK, doAuthedGET(t, ts, ts.Auth(t), "/api/v1/transactions"))

	body := scrape(t, ts)

	assert.Contains(t, body, `ffs_login_attempts_total{outcome="ok"} 1`)
	assert.Contains(t, body, `ffs_login_attempts_total{outcome="invalid_credentials"} 1`)
	assert.Contains(t, body,
		`ffs_http_requests_total{method="GET",route="/api/v1/transactions",status="200"} 1`)
	assert.Contains(t, body, "ffs_build_info{")
}

// /metrics живёт на отдельном слушателе, поэтому у Echo его нет ни для кого:
// RequireBearer висит только на группе /api/v1, значит это 404, а не 401.
func TestMetrics_NotServedByEcho(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)

	assert.Equal(t, http.StatusNotFound, doGET(t, ts, nil, "/metrics").Code)
	assert.Equal(t, http.StatusNotFound, doAuthedGET(t, ts, ts.Auth(t), "/metrics"))
}
