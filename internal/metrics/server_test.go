package metrics_test

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/metrics"
)

func TestNewServer_ServesOnlyMetrics(t *testing.T) {
	m := metrics.New("v0.5.0", slog.New(slog.DiscardHandler))
	srv := metrics.NewServer("127.0.0.1:0", m.Handler())
	require.Positive(t, srv.ReadHeaderTimeout)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { _ = srv.Close() })

	base := "http://" + listener.Addr().String()

	resp, err := http.Get(base + "/metrics")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "ffs_build_info")

	other, err := http.Get(base + "/health")
	require.NoError(t, err)
	require.NoError(t, other.Body.Close())
	assert.Equal(t, http.StatusNotFound, other.StatusCode)
}

func TestNewServer_BusyPortReturnsError(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })

	m := metrics.New("v0.5.0", slog.New(slog.DiscardHandler))
	srv := metrics.NewServer(busy.Addr().String(), m.Handler())
	t.Cleanup(func() { _ = srv.Close() })

	assert.Error(t, srv.ListenAndServe())
}
