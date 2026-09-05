package observability_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/observability"
)

// Ошибка из middleware (401 от RequireBearer) ещё не записана в ответ, когда логгер
// смотрит на него: статус берётся из *echo.HTTPError, уровень — по классу статуса.
func TestLoggingMiddleware_StatusAndLevel(t *testing.T) {
	cases := map[string]struct {
		handler   echo.HandlerFunc
		status    int
		level     string
		message   string
		hasError  bool
		wantError bool
	}{
		"ok": {
			handler: func(c echo.Context) error { return c.NoContent(http.StatusNoContent) },
			status:  http.StatusNoContent, level: "INFO", message: "HTTP request completed",
		},
		"401 from middleware": {
			handler: func(echo.Context) error { return echo.NewHTTPError(http.StatusUnauthorized) },
			status:  http.StatusUnauthorized, level: "WARN", message: "HTTP request rejected", hasError: true,
			wantError: true,
		},
		"plain error": {
			handler: func(echo.Context) error { return errors.New("boom") },
			status:  http.StatusInternalServerError, level: "ERROR", message: "HTTP request failed", hasError: true,
			wantError: true,
		},
		"503 http error": {
			handler: func(echo.Context) error { return echo.NewHTTPError(http.StatusServiceUnavailable) },
			status:  http.StatusServiceUnavailable, level: "ERROR", message: "HTTP request failed", hasError: true,
			wantError: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			rec := httptest.NewRecorder()

			err := observability.LoggingMiddleware(logger)(tc.handler)(e.NewContext(req, rec))

			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			var entry map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &entry), buf.String())
			assert.Equal(t, tc.level, entry["level"])
			assert.Equal(t, tc.message, entry["msg"])
			assert.InDelta(t, tc.status, entry["status"], 0)
			_, hasErr := entry["error"]
			assert.Equal(t, tc.hasError, hasErr)
			assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
		})
	}
}
