package application

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
)

func TestHTTPServer_BuildNetHTTPServer_UsesConfiguredTimeouts(t *testing.T) {
	e := echo.New()
	s := &HTTPServer{
		echo: e,
		config: &Config{
			Host:         "127.0.0.1",
			Port:         "8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 7 * time.Second,
			IdleTimeout:  11 * time.Second,
		},
	}

	server := s.buildNetHTTPServer("127.0.0.1:8080")

	assert.Equal(t, "127.0.0.1:8080", server.Addr)
	assert.Equal(t, 5*time.Second, server.ReadTimeout)
	assert.Equal(t, 7*time.Second, server.WriteTimeout)
	assert.Equal(t, 11*time.Second, server.IdleTimeout)
	assert.Same(t, e, server.Handler)
}

// Ветки newAPIErrorHandler, недостижимые через роутер: произвольная ошибка → 500 без её текста,
// уже записанный ответ не перезаписывается, 403 и авторский текст HTTPError.
func TestAPIErrorHandler(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{
			"plain error hides its text", errors.New("sql: no rows in result set"),
			http.StatusInternalServerError, handlers.ErrCodeInternal, http.StatusText(http.StatusInternalServerError),
		},
		{
			"forbidden uses handler wording", echo.NewHTTPError(http.StatusForbidden),
			http.StatusForbidden, handlers.ErrCodeForbidden, handlers.ErrMessageForbidden,
		},
		{
			"author message is kept", echo.NewHTTPError(http.StatusNotFound, "session not found"),
			http.StatusNotFound, handlers.ErrCodeNotFound, "session not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			rec := httptest.NewRecorder()

			newAPIErrorHandler(slog.New(slog.DiscardHandler))(tc.err, e.NewContext(req, rec))

			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			var body handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
			assert.Equal(t, tc.code, body.Error.Code)
			assert.Equal(t, tc.message, body.Error.Message)
			assert.NotContains(t, rec.Body.String(), "sql:")
		})
	}

	t.Run("committed response is left alone", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		require.NoError(t, c.NoContent(http.StatusNoContent))

		newAPIErrorHandler(slog.New(slog.DiscardHandler))(errors.New("late failure"), c)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.String())
	})
}
