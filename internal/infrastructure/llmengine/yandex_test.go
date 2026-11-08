package llmengine_test

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/infrastructure/llmengine"
)

type yandexRequest struct {
	Model          string          `json:"model"`
	MaxTokens      int             `json:"max_tokens"`
	Temperature    *float64        `json:"temperature"`
	ResponseFormat json.RawMessage `json:"response_format"`
	Messages       []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
}

func TestEngine_Yandex_Recognize(t *testing.T) {
	var (
		got     yandexRequest
		path    string
		headers http.Header
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, headers = r.URL.Path, r.Header.Clone()
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"model": "qwen3.6-35b-a3b",
			"choices": []map[string]any{{
				"message":       map[string]string{"role": "assistant", "content": fencedText},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 900, "completion_tokens": 300, "total_tokens": 1200},
		}))
	}))
	t.Cleanup(srv.Close)

	engine, err := llmengine.NewYandexAt(srv.URL, "b1gfolder", "secret-key", "qwen3.6-35b-a3b",
		5*time.Second, nil, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	result, err := engine.Recognize(t.Context(), testInput())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.EqualValues(t, 34990, result.Items[0].AmountMinor)

	assert.Equal(t, "/chat/completions", path)
	assert.Equal(t, "Api-Key secret-key", headers.Get("Authorization"))
	assert.Equal(t, "b1gfolder", headers.Get("Openai-Project"))
	assert.Equal(t, "gpt://b1gfolder/qwen3.6-35b-a3b", got.Model)
	assert.Equal(t, 32768, got.MaxTokens)
	require.NotNil(t, got.Temperature)
	assert.Zero(t, *got.Temperature)
	assert.Empty(t, got.ResponseFormat, "qwen3.6 с json_object отвечает мусором")
	require.Len(t, got.Messages, 3)
	assert.Contains(t, string(got.Messages[1].Content),
		"data:image/png;base64,"+base64.StdEncoding.EncodeToString([]byte("first-image-bytes")))
}

func TestEngine_NewYandex_RejectsModelWithoutImages(t *testing.T) {
	_, err := llmengine.NewYandex("b1gfolder", "key", "yandexgpt/latest", time.Minute, nil, slog.Default())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not accept images")
}
