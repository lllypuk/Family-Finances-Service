//nolint:testpackage // serveAll и listeners не экспортируются: тест живёт в том же пакете.
package internal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// freeAddr занимает и сразу отпускает порт: слушатель теста должен подняться на
// известном адресе, иначе дождаться его готовности нечем.
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

// stopSignal — сервер на заданном адресе и канал, который закрывает его Shutdown.
func stopSignal(t *testing.T, addr string) (listener, <-chan struct{}) {
	t.Helper()

	stopped := make(chan struct{})
	srv := &http.Server{Addr: addr, ReadHeaderTimeout: time.Second}
	srv.RegisterOnShutdown(func() { close(stopped) })

	return metricsListener{Server: srv}, stopped
}

func waitListening(t *testing.T, addr string) {
	t.Helper()

	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "listener %s did not come up", addr)
}

func requireRefused(t *testing.T, addr string) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err == nil {
		_ = conn.Close()
	}
	require.Error(t, err, "listener %s is still accepting connections", addr)
}

// serveAll возвращается только после того, как горутины Start завершились: иначе
// shutdown() закрыл бы базу под живым скрейпом /metrics.
func TestServeAll_ContextCancelled_StopsAll(t *testing.T) {
	firstAddr, secondAddr := freeAddr(t), freeAddr(t)
	first, firstStopped := stopSignal(t, firstAddr)
	second, secondStopped := stopSignal(t, secondAddr)

	ctx, cancel := context.WithCancel(t.Context())
	type result struct {
		stopErrs []error
		err      error
	}
	done := make(chan result, 1)
	go func() {
		stopErrs, err := serveAll(ctx, first, second)
		done <- result{stopErrs: stopErrs, err: err}
	}()

	waitListening(t, firstAddr)
	waitListening(t, secondAddr)
	cancel()

	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.NoError(t, errors.Join(got.stopErrs...))
	case <-time.After(GracefulShutdownTimeout):
		t.Fatal("serveAll did not return")
	}

	requireClosed(t, firstStopped)
	requireClosed(t, secondStopped)
	requireRefused(t, firstAddr)
	requireRefused(t, secondAddr)
}

func TestServeAll_AddressInUse_ReturnsErrorAndStopsRest(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })

	firstAddr := freeAddr(t)
	first, firstStopped := stopSignal(t, firstAddr)
	second, _ := stopSignal(t, busy.Addr().String())

	_, serveErr := serveAll(t.Context(), first, second)

	require.Error(t, serveErr)
	requireClosed(t, firstStopped)
	requireRefused(t, firstAddr)
}

// Имена метрик — контракт с observability, а собирает их проводка NewApplication:
// коллекторы состояния, каталога копий и пула регистрируются только там.
func TestNewApplication_MetricsScrapeCoversContract(t *testing.T) {
	app := newTestApplication(t, filepath.Join(t.TempDir(), "budget.db"), freeAddr(t))

	require.Len(t, app.listeners, 2)
	srv, ok := app.listeners[1].(metricsListener)
	require.True(t, ok, "второй слушатель — служебный порт метрик")

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	for _, name := range []string{
		"ffs_build_info",
		"ffs_db_size_bytes",
		"ffs_sessions_active",
		"ffs_users",
		"ffs_transactions",
		"ffs_setup_complete",
		"ffs_backup_files",
		"ffs_backup_latest_file_timestamp_seconds",
		"go_sql_max_open_connections",
		"go_goroutines",
	} {
		assert.Contains(t, body, name)
	}
}

// Пустой METRICS_ADDR выключает метрики целиком: ни слушателя, ни каталога копий,
// созданного скрейпом.
func TestNewApplication_NoMetricsAddr_NoSecondListener(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "budget.db")
	app := newTestApplication(t, dbPath, "")

	assert.Len(t, app.listeners, 1)
	assert.NoDirExists(t, filepath.Join(filepath.Dir(dbPath), "backups"))
}

func newTestApplication(t *testing.T, dbPath, metricsAddr string) *Application {
	t.Helper()

	// migrationsDir резолвится от CWD процесса, а go test запускает пакет из его каталога.
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", dbPath)
	t.Setenv("METRICS_ADDR", metricsAddr)

	app, err := NewApplication()
	require.NoError(t, err)
	t.Cleanup(func() { _ = app.db.Close() })

	return app
}

func requireClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(GracefulShutdownTimeout):
		t.Fatal("listener was not shut down")
	}
}
