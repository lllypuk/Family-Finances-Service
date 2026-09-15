package metrics_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/metrics"
)

type fakeBackupLister struct {
	times []time.Time
	err   error
}

func (l fakeBackupLister) ListBackupTimes(_ context.Context) ([]time.Time, error) {
	return l.times, l.err
}

func collectorOutput(t *testing.T, lister metrics.BackupLister) string {
	t.Helper()

	m := testMetrics(t)
	require.NoError(t, m.Register(metrics.NewBackupDirCollector(lister)))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	return rec.Body.String()
}

func TestBackupDirCollector_EmptyDir(t *testing.T) {
	body := collectorOutput(t, fakeBackupLister{})

	assert.Contains(t, body, "ffs_backup_files 0")
	assert.Contains(t, body, "ffs_backup_latest_file_timestamp_seconds 0")
}

func TestBackupDirCollector_ReportsNewestFile(t *testing.T) {
	newest := time.Unix(1757900000, 0)
	body := collectorOutput(t, fakeBackupLister{times: []time.Time{
		newest.Add(-2 * time.Hour),
		newest,
		newest.Add(-24 * time.Hour),
	}})

	assert.Contains(t, body, "ffs_backup_files 3")
	assert.Contains(t, body, "ffs_backup_latest_file_timestamp_seconds 1.7579e+09")
}

// Ошибка чтения каталога гасит только свои метрики: остальной скрейп остаётся 200.
func TestBackupDirCollector_ErrorKeepsScrapeAlive(t *testing.T) {
	body := collectorOutput(t, fakeBackupLister{err: errors.New("permission denied")})

	assert.NotContains(t, body, "ffs_backup_files")
	assert.NotContains(t, body, "ffs_backup_latest_file_timestamp_seconds")
	assert.Contains(t, body, "ffs_build_info")
}
