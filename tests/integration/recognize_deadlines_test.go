package integration_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/recognize"
	"family-budget-service/internal/services"
	"family-budget-service/internal/testhelpers"
)

// Сроки сокращены в десятки раз: SERVER_READ/WRITE_TIMEOUT 15 с → 1 с, загрузка 60 с → 1 с.
const (
	deadlineServerTimeout = time.Second
	deadlineUploadTimeout = time.Second
	deadlineEngineBudget  = 3 * time.Second
)

// slowRecognizer отвечает через delay или раньше, если контекст отменён.
type slowRecognizer struct {
	delay    time.Duration
	started  chan struct{}
	canceled chan struct{}
}

func newSlowRecognizer(delay time.Duration) *slowRecognizer {
	return &slowRecognizer{delay: delay, started: make(chan struct{}), canceled: make(chan struct{})}
}

func (s *slowRecognizer) Recognize(ctx context.Context, _ recognize.Input) (recognize.Result, error) {
	close(s.started)

	select {
	case <-time.After(s.delay):
		return recognize.Result{Model: "slow"}, nil
	case <-ctx.Done():
		close(s.canceled)
		return recognize.Result{}, ctx.Err()
	}
}

func (s *slowRecognizer) Budget() time.Duration { return deadlineEngineBudget }

// startDeadlineServer поднимает Echo стенда на настоящем сокете: ResponseRecorder дедлайнов не знает.
func startDeadlineServer(t *testing.T, engine services.Recognizer) (*testhelpers.TestServer, *httptest.Server) {
	t.Helper()

	ts := testhelpers.SetupHTTPServer(t,
		testhelpers.WithRecognizer(engine),
		testhelpers.WithRecognizeUploadTimeout(deadlineUploadTimeout),
	)

	srv := httptest.NewUnstartedServer(ts.Server.Echo())
	srv.Config.ReadTimeout = deadlineServerTimeout
	srv.Config.WriteTimeout = deadlineServerTimeout
	srv.Start()
	t.Cleanup(srv.Close)

	return ts, srv
}

func recognizeRequest(
	ctx context.Context,
	t *testing.T,
	srv *httptest.Server,
	sess *testhelpers.AuthSession,
	body io.Reader,
	contentType string,
) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+recognizePath, body)
	require.NoError(t, err)
	req.Header.Set(echo.HeaderContentType, contentType)
	sess.Apply(req)

	return req
}

func TestRecognizeDeadlines_LateEngineAnswerArrives(t *testing.T) {
	engine := newSlowRecognizer(deadlineServerTimeout + deadlineUploadTimeout + 500*time.Millisecond)
	ts, srv := startDeadlineServer(t, engine)

	body, contentType := onePNG(t)
	resp, err := srv.Client().Do(recognizeRequest(t.Context(), t, srv, ts.Auth(t), bytes.NewReader(body), contentType))
	require.NoError(t, err)
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(payload))
	assert.Contains(t, string(payload), `"model":"slow"`)
}

func TestRecognizeDeadlines_SlowUploadRefused(t *testing.T) {
	engine := newSlowRecognizer(0)
	ts, srv := startDeadlineServer(t, engine)

	body, contentType := onePNG(t)
	stalled, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	go func() {
		// Половина тела и тишина: сервер не дождётся конца загрузки.
		_, _ = writer.Write(body[:len(body)/2])
	}()

	req := recognizeRequest(t.Context(), t, srv, ts.Auth(t), stalled, contentType)
	req.ContentLength = int64(len(body))

	started := time.Now()
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestTimeout, resp.StatusCode)
	assert.Less(t, time.Since(started), deadlineUploadTimeout+2*time.Second)

	select {
	case <-engine.started:
		t.Fatal("движок вызван без тела")
	default:
	}
}

func TestRecognizeDeadlines_ClientCancelReachesEngine(t *testing.T) {
	engine := newSlowRecognizer(time.Minute)
	ts, srv := startDeadlineServer(t, engine)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	body, contentType := onePNG(t)
	req := recognizeRequest(ctx, t, srv, ts.Auth(t), bytes.NewReader(body), contentType)
	done := make(chan error, 1)
	go func() {
		resp, err := srv.Client().Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		done <- err
	}()

	select {
	case <-engine.started:
	case <-time.After(5 * time.Second):
		t.Fatal("движок не вызван")
	}
	cancel()

	select {
	case <-engine.canceled:
	case <-time.After(deadlineServerTimeout + 2*time.Second):
		t.Fatal("отмена клиента не дошла до движка")
	}
	require.ErrorIs(t, <-done, context.Canceled)
}
