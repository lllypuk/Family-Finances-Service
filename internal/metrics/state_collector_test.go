package metrics_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/metrics"
)

type fakeStateReader struct {
	sessions     int
	active       int
	inactive     int
	transactions int
	setup        bool
	err          error
}

func (r fakeStateReader) ActiveSessions(_ context.Context, _ time.Time) (int, error) {
	return r.sessions, r.err
}

func (r fakeStateReader) UsersByActive(_ context.Context) (int, int, error) {
	return r.active, r.inactive, r.err
}

func (r fakeStateReader) Transactions(_ context.Context) (int, error) {
	return r.transactions, r.err
}

func (r fakeStateReader) SetupComplete(_ context.Context) (bool, error) {
	return r.setup, r.err
}

func stateOutput(t *testing.T, reader metrics.StateReader, dbPath string) string {
	t.Helper()

	m := testMetrics(t)
	require.NoError(t, m.Register(metrics.NewStateCollector(reader, dbPath)))

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	return rec.Body.String()
}

func TestStateCollector_ReportsCountsAndSizes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "budget.db")
	require.NoError(t, os.WriteFile(dbPath, make([]byte, 4096), 0o600))

	body := stateOutput(t, fakeStateReader{
		sessions: 3, active: 2, inactive: 1, transactions: 42, setup: true,
	}, dbPath)

	assert.Contains(t, body, `ffs_db_size_bytes{file="main"} 4096`)
	assert.Contains(t, body, `ffs_db_size_bytes{file="wal"} 0`)
	assert.Contains(t, body, "ffs_sessions_active 3")
	assert.Contains(t, body, `ffs_users{active="true"} 2`)
	assert.Contains(t, body, `ffs_users{active="false"} 1`)
	assert.Contains(t, body, "ffs_transactions 42")
	assert.Contains(t, body, "ffs_setup_complete 1")
}

func TestStateCollector_MissingDatabaseFile(t *testing.T) {
	body := stateOutput(t, fakeStateReader{}, filepath.Join(t.TempDir(), "absent.db"))

	assert.Contains(t, body, `ffs_db_size_bytes{file="main"} 0`)
	assert.Contains(t, body, `ffs_db_size_bytes{file="wal"} 0`)
	assert.Contains(t, body, "ffs_setup_complete 0")
}

// Ошибка ридера гасит только его метрики: размер файлов и остальной скрейп остаются.
func TestStateCollector_ReaderErrorKeepsScrapeAlive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "budget.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("x"), 0o600))

	body := stateOutput(t, fakeStateReader{err: errors.New("database is locked")}, dbPath)

	assert.NotContains(t, body, "ffs_sessions_active")
	assert.NotContains(t, body, "ffs_users{")
	assert.NotContains(t, body, "ffs_transactions ")
	assert.NotContains(t, body, "ffs_setup_complete")
	assert.Contains(t, body, `ffs_db_size_bytes{file="main"} 1`)
	assert.Contains(t, body, "ffs_build_info")
}

// Запрос ридера идёт с дедлайном ScrapeTimeout: HTTP-таймаут promhttp SQL не прерывает.
// Ридер отвечает сразу — ждать здесь реальные пять секунд нечего.
func TestStateCollector_ReaderGetsScrapeDeadline(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "budget.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("x"), 0o600))

	reader := &deadlineStateReader{}
	body := stateOutput(t, reader, dbPath)

	require.True(t, reader.hadDeadline, "ридер получил контекст без дедлайна")
	assert.Positive(t, reader.left)
	assert.LessOrEqual(t, reader.left, metrics.ScrapeTimeout)
	assert.NotContains(t, body, "ffs_sessions_active")
	assert.Contains(t, body, "ffs_transactions 0")
}

// deadlineStateReader запоминает дедлайн первого запроса и отказывает, как отказал бы
// запрос, не уложившийся в него.
type deadlineStateReader struct {
	fakeStateReader

	hadDeadline bool
	left        time.Duration
}

func (r *deadlineStateReader) ActiveSessions(ctx context.Context, _ time.Time) (int, error) {
	deadline, ok := ctx.Deadline()
	r.hadDeadline = ok
	r.left = time.Until(deadline)
	return 0, context.DeadlineExceeded
}
