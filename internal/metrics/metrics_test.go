package metrics_test

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/metrics"
	"family-budget-service/internal/testhelpers"
)

// brokenCollector отдаёт метрику, которую Gather отвергает — имитация ошибки коллектора состояния.
type brokenCollector struct {
	desc *prometheus.Desc
}

func newBrokenCollector() *brokenCollector {
	return &brokenCollector{desc: prometheus.NewDesc("ffs_broken", "всегда ошибка", nil, nil)}
}

func (c *brokenCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *brokenCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.NewInvalidMetric(c.desc, errors.New("collector failed"))
}

func testMetrics(t *testing.T) *metrics.Metrics {
	t.Helper()
	return metrics.New("v1.2.3", slog.New(slog.DiscardHandler))
}

func gatheredValue(t *testing.T, m *metrics.Metrics, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := m.Gatherer().Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if !hasLabels(metric, labels) {
				continue
			}
			switch {
			case metric.GetCounter() != nil:
				return metric.GetCounter().GetValue()
			case metric.GetGauge() != nil:
				return metric.GetGauge().GetValue()
			case metric.GetHistogram() != nil:
				return metric.GetHistogram().GetSampleSum()
			}
		}
	}
	t.Fatalf("метрика %s с метками %v не найдена", name, labels)
	return 0
}

func hasLabels(metric *dto.Metric, labels map[string]string) bool {
	for name, want := range labels {
		found := false
		for _, pair := range metric.GetLabel() {
			if pair.GetName() == name && pair.GetValue() == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestMetrics_New_ExposesBuildInfo(t *testing.T) {
	m := testMetrics(t)

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_build_info", map[string]string{"version": "v1.2.3"}), 0)

	count, err := testutil.GatherAndCount(m.Gatherer(), "ffs_build_info")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMetrics_New_TwoInstancesDoNotConflict(t *testing.T) {
	first := testMetrics(t)
	second := metrics.New("v9.9.9", slog.New(slog.DiscardHandler))

	assert.InDelta(t, 1.0, gatheredValue(t, first, "ffs_build_info", map[string]string{"version": "v1.2.3"}), 0)
	assert.InDelta(t, 1.0, gatheredValue(t, second, "ffs_build_info", map[string]string{"version": "v9.9.9"}), 0)
}

func TestMetrics_Handler_ServesTextFormat(t *testing.T) {
	m := testMetrics(t)
	m.HTTP().ObserveRequest(http.MethodGet, "/health", http.StatusOK, 12*time.Millisecond)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, rec.Body.String(), "ffs_build_info")
	assert.Contains(t, rec.Body.String(), `ffs_http_requests_total{method="GET",route="/health",status="200"} 1`)
}

func TestMetrics_Handler_BrokenCollectorKeepsTwoHundred(t *testing.T) {
	m := testMetrics(t)
	require.NoError(t, m.Register(newBrokenCollector()))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ffs_build_info")
	assert.NotContains(t, rec.Body.String(), "ffs_broken")
}

func TestMetrics_HTTP_CountsRequestsAndInFlight(t *testing.T) {
	m := testMetrics(t)

	m.HTTP().IncInFlight()
	m.HTTP().ObserveRequest(http.MethodPost, "/api/v1/transactions", http.StatusCreated, 250*time.Millisecond)
	m.HTTP().ObservePanic()

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_in_flight", nil), 0)
	m.HTTP().DecInFlight()
	assert.InDelta(t, 0.0, gatheredValue(t, m, "ffs_http_requests_in_flight", nil), 0)

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_requests_total",
		map[string]string{"method": http.MethodPost, "status": "201"}), 0)
	assert.InDelta(t, 0.25, gatheredValue(t, m, "ffs_http_request_duration_seconds",
		map[string]string{"route": "/api/v1/transactions"}), 0.0001)
	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_http_panics_total", nil), 0)

	routes, err := testutil.GatherAndCount(m.Gatherer(), "ffs_http_requests_total")
	require.NoError(t, err)
	assert.Equal(t, 1, routes)
}

func TestMetrics_Login_CountsByOutcome(t *testing.T) {
	m := testMetrics(t)

	m.Login().ObserveLogin("ok")
	m.Login().ObserveLogin("invalid_credentials")
	m.Login().ObserveLogin("invalid_credentials")

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_login_attempts_total", map[string]string{"outcome": "ok"}), 0)
	assert.InDelta(t, 2.0,
		gatheredValue(t, m, "ffs_login_attempts_total", map[string]string{"outcome": "invalid_credentials"}), 0)
}

func TestMetrics_Backup_CountsOutcomeAndDuration(t *testing.T) {
	m := testMetrics(t)

	m.Backup().ObserveBackup("ok", 2*time.Second)

	assert.InDelta(t, 1.0, gatheredValue(t, m, "ffs_backups_total", map[string]string{"outcome": "ok"}), 0)
	assert.InDelta(t, 2.0, gatheredValue(t, m, "ffs_backup_duration_seconds", nil), 0.0001)
}

// db_name="budget" — контракт с observability, поэтому имя проверяется, а не только наличие go_sql_*.
func TestMetrics_NewDBCollector_LabelsPoolAsBudget(t *testing.T) {
	db := testhelpers.SetupSQLiteTestDB(t).DB
	m := testMetrics(t)
	require.NoError(t, m.Register(metrics.NewDBCollector(db)))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `go_sql_max_open_connections{db_name="budget"}`)
}
