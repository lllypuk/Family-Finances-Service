package observability_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/router"
	"family-budget-service/internal/observability"
)

func TestRequestID_GeneratesAndKeeps(t *testing.T) {
	var seen string
	h := observability.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get(observability.RequestIDHeader)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Len(t, rec.Header().Get("X-Request-ID"), 32)
	assert.Equal(t, rec.Header().Get("X-Request-ID"), seen)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, "abc", rec.Header().Get("X-Request-ID"))
}

func TestRecover_PanicBecomesFailure(t *testing.T) {
	var panics int
	mw := observability.Recover(
		func(*http.Request, any, []byte) { panics++ },
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	)

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, 1, panics)
}

func TestRecover_StartedResponseIsNotOverwritten(t *testing.T) {
	failed := false
	mw := observability.Recover(
		func(*http.Request, any, []byte) {},
		func(http.ResponseWriter, *http.Request) { failed = true },
	)

	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		panic("late")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.False(t, failed)
	assert.Equal(t, http.StatusAccepted, rec.Code)
}

func TestRecover_AbortHandlerPropagates(t *testing.T) {
	mw := observability.Recover(func(*http.Request, any, []byte) {}, func(http.ResponseWriter, *http.Request) {})
	h := mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

func TestTimeout_DeadlineAndSkip(t *testing.T) {
	var hasDeadline bool
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, hasDeadline = r.Context().Deadline()
	})
	mw := observability.Timeout(time.Minute, func(r *http.Request) bool { return r.URL.Path == "/skip" })

	mw(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.True(t, hasDeadline)

	mw(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/skip", nil))
	assert.False(t, hasDeadline)
}

func TestRequestLogger_StatusAndLevel(t *testing.T) {
	cases := map[string]struct {
		status  int
		level   string
		message string
	}{
		"ok":       {http.StatusNoContent, "INFO", "HTTP request completed"},
		"rejected": {http.StatusUnauthorized, "WARN", "HTTP request rejected"},
		"failed":   {http.StatusServiceUnavailable, "ERROR", "HTTP request failed"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			mw := observability.RequestLogger(logger, func(*http.Request) string { return "203.0.113.1" })

			mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) })).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

			var entry map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), buf.String())
			assert.Equal(t, tc.level, entry["level"])
			assert.Equal(t, tc.message, entry["msg"])
			assert.InDelta(t, tc.status, entry["status"], 0)
			assert.Equal(t, "203.0.113.1", entry["remote_addr"])
		})
	}
}

// ResponseController доходит до сокета сквозь все обёртки цепочки; на рекордере он получил бы ErrNotSupported.
func TestChain_ResponseControllerReachesSocket(t *testing.T) {
	rt := router.New(func(http.ResponseWriter, *http.Request, error) {})
	var setErr error
	rt.Register(http.MethodGet, "/deadline", func(w http.ResponseWriter, _ *http.Request) error {
		rc := http.NewResponseController(w)
		setErr = errors.Join(
			rc.SetReadDeadline(time.Now().Add(time.Minute)),
			rc.SetWriteDeadline(time.Now().Add(time.Minute)),
		)

		return nil
	})

	var h http.Handler = rt
	h = observability.RequestLogger(slog.New(slog.DiscardHandler), func(*http.Request) string { return "" })(h)
	h = observability.Timeout(time.Minute, nil)(h)
	h = observability.Recover(func(*http.Request, any, []byte) {}, func(http.ResponseWriter, *http.Request) {})(h)
	h = observability.RequestID(h)

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/deadline")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, setErr)
}
