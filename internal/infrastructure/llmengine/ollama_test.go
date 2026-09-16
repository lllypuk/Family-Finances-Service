package llmengine_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/infrastructure/llmengine"
	"family-budget-service/internal/recognize"
)

const (
	testModel  = "gemma4:31b-cloud"
	fencedText = "```json\n{\"items\":[{\"source\":1,\"amount\":\"349.90\",\"currency\":\"₽\",\"type\":\"expense\"," +
		"\"date\":\"2026-09-14\",\"date_text\":\"14 сентября 2026\",\"year_present\":true," +
		"\"description\":\"Пятёрочка\",\"category\":null}],\"incomplete\":false}\n```"
)

type ollamaRequest struct {
	Model    string `json:"model"`
	Format   string `json:"format"`
	Messages []struct {
		Role    string   `json:"role"`
		Content string   `json:"content"`
		Images  [][]byte `json:"images"`
	} `json:"messages"`
	Options struct {
		Temperature *float64 `json:"temperature"`
	} `json:"options"`
}

func chatAnswer(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"model":             "gemma4:31b",
		"message":           map[string]string{"role": "assistant", "content": content},
		"done":              true,
		"done_reason":       "stop",
		"prompt_eval_count": 519,
		"eval_count":        202,
	}))
}

func newEngine(t *testing.T, handler http.HandlerFunc) (*llmengine.Engine, *atomic.Int32, *bytes.Buffer) {
	t.Helper()

	var calls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	var logs bytes.Buffer

	engine := llmengine.New(srv.URL, testModel, 5*time.Second, nil, slog.New(slog.NewJSONHandler(&logs, nil)))
	engine.SetPause(0)

	return engine, &calls, &logs
}

func testInput() recognize.Input {
	return recognize.Input{
		Images: []recognize.Image{
			{MIME: "image/png", Data: []byte("first-image-bytes")},
			{MIME: "image/jpeg", Data: []byte("second-image-bytes")},
		},
		Today:    date.New(2026, time.September, 16),
		Currency: "RUB",
	}
}

func TestEngine_Recognize_FencedAnswer(t *testing.T) {
	var got ollamaRequest

	engine, calls, logs := newEngine(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &got))
		chatAnswer(t, w, fencedText)
	})

	result, err := engine.Recognize(t.Context(), testInput())
	require.NoError(t, err)

	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "gemma4:31b", result.Model)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Пятёрочка", result.Items[0].Description)
	assert.EqualValues(t, 34990, result.Items[0].AmountMinor)

	assert.Equal(t, testModel, got.Model)
	assert.Equal(t, "json", got.Format)
	require.NotNil(t, got.Options.Temperature)
	assert.Zero(t, *got.Options.Temperature)
	require.Len(t, got.Messages, 3)
	assert.Equal(t, "system", got.Messages[0].Role)
	assert.Empty(t, got.Messages[0].Images)
	assert.Equal(t, "user", got.Messages[1].Role)
	assert.Equal(t, "Картинка 1", got.Messages[1].Content)
	assert.Equal(t, [][]byte{[]byte("first-image-bytes")}, got.Messages[1].Images)
	assert.Equal(t, "Картинка 2", got.Messages[2].Content)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
	assert.Equal(t, "ok", entry["outcome"])
	assert.Equal(t, "receipt.screenshot", entry["task"])
	assert.InDelta(t, 1, entry["items"], 0)
	assert.InDelta(t, 519, entry["input_tokens"], 0)
	assert.NotContains(t, logs.String(), "Пятёрочка")
	assert.NotContains(t, logs.String(), base64.StdEncoding.EncodeToString([]byte("first-image-bytes")))
}

func TestEngine_Recognize_BadAnswerIsNotRetried(t *testing.T) {
	engine, calls, logs := newEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		chatAnswer(t, w, `{"items": {}}`)
	})

	_, err := engine.Recognize(t.Context(), testInput())
	require.ErrorIs(t, err, recognize.ErrBadAnswer)
	require.NotErrorIs(t, err, recognize.ErrUnavailable)
	assert.Equal(t, int32(1), calls.Load())
	assert.Contains(t, logs.String(), `"outcome":"bad_answer"`)
}

func TestEngine_Recognize_TooManyRequests(t *testing.T) {
	engine, calls, _ := newEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := engine.Recognize(t.Context(), testInput())
	require.ErrorIs(t, err, recognize.ErrUnavailable)

	var unavailable *recognize.UnavailableError
	require.ErrorAs(t, err, &unavailable)
	assert.Equal(t, 30*time.Second, unavailable.RetryAfter)
	assert.Equal(t, int32(1), calls.Load(), "просьба дольше MaxRetryAfter не повторяется")
}

func TestEngine_Recognize_Unauthorized(t *testing.T) {
	engine, calls, _ := newEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := engine.Recognize(t.Context(), testInput())
	require.ErrorIs(t, err, recognize.ErrUnavailable)

	var unavailable *recognize.UnavailableError
	require.ErrorAs(t, err, &unavailable)
	assert.Zero(t, unavailable.RetryAfter)
	assert.Equal(t, int32(1), calls.Load())
}

func TestEngine_Recognize_BadRequestIsNotUnavailable(t *testing.T) {
	engine, calls, _ := newEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := engine.Recognize(t.Context(), testInput())
	require.Error(t, err)
	require.NotErrorIs(t, err, recognize.ErrUnavailable)
	require.NotErrorIs(t, err, recognize.ErrBadAnswer)
	assert.Equal(t, int32(1), calls.Load())
}

func TestEngine_Recognize_ConnectionDropped(t *testing.T) {
	engine, calls, _ := newEngine(t, func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !assert.True(t, ok) {
			return
		}

		conn, _, err := hijacker.Hijack()
		if assert.NoError(t, err) {
			_ = conn.Close()
		}
	})

	_, err := engine.Recognize(t.Context(), testInput())
	require.ErrorIs(t, err, recognize.ErrUnavailable)
	assert.Equal(t, int32(2), calls.Load(), "обрыв повторяется один раз")
}

func TestEngine_Recognize_ContextCancelled(t *testing.T) {
	started := make(chan struct{}, 1)

	engine, calls, _ := newEngine(t, func(_ http.ResponseWriter, r *http.Request) {
		// Без дочитанного тела сервер не следит за соединением и контекст запроса не отменится.
		_, _ = io.Copy(io.Discard, r.Body)
		started <- struct{}{}
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-started
		cancel()
	}()

	_, err := engine.Recognize(ctx, testInput())
	require.ErrorIs(t, err, recognize.ErrUnavailable)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), calls.Load())
}

func TestEngine_Budget(t *testing.T) {
	engine := llmengine.New("http://127.0.0.1:0", testModel, 60*time.Second, nil, slog.Default())

	assert.Equal(t, 130*time.Second, engine.Budget())
}
