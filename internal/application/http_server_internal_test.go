package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/application/router"
)

func TestHTTPServer_BuildNetHTTPServer_UsesConfiguredTimeouts(t *testing.T) {
	s := &HTTPServer{
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
	assert.Same(t, s, server.Handler)
}

func TestAPIErrorHandler_Cases(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{"Bare401", &router.HTTPError{Status: http.StatusUnauthorized}, 401, handlers.ErrCodeUnauthorized,
			handlers.ErrMessageUnauthorized},
		{
			"Explicit",
			&router.HTTPError{Status: http.StatusTeapot, Code: "TEAPOT", Message: "short"},
			418,
			"TEAPOT",
			"short",
		},
		{"Wrapped", fmt.Errorf("x: %w", &router.HTTPError{Status: http.StatusNotFound}), 404, handlers.ErrCodeNotFound,
			"Not Found"},
		{"Forbidden", &router.HTTPError{Status: http.StatusForbidden}, 403, handlers.ErrCodeForbidden,
			handlers.ErrMessageForbidden},
		{"Deadline", fmt.Errorf("query: %w", context.DeadlineExceeded), 503, handlers.ErrorCodeForStatus(503),
			"Service Unavailable"},
		{"UpstreamDeadline", fmt.Errorf("upstream: %w", context.DeadlineExceeded), 500, handlers.ErrCodeInternal,
			"Internal Server Error"},
		{"Plain", errors.New("secret detail"), 500, handlers.ErrCodeInternal, "Internal Server Error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rt := router.New(newAPIErrorHandler(slog.New(slog.DiscardHandler)))
			rt.Register(http.MethodGet, "/x", func(http.ResponseWriter, *http.Request) error { return tt.err })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if strings.HasPrefix(tt.name, "Deadline") {
				ctx, cancel := context.WithTimeout(req.Context(), time.Hour)
				defer cancel()
				req = req.WithContext(ctx)
			}
			rt.ServeHTTP(rec, req)

			assert.Equal(t, tt.status, rec.Code)
			var resp handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tt.code, resp.Error.Code)
			assert.Equal(t, tt.message, resp.Error.Message)
		})
	}
}

func TestAPIErrorHandler_ErrorAfterWrite_KeepsResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	rt := router.New(newAPIErrorHandler(slog.New(slog.DiscardHandler)))
	rt.Register(http.MethodGet, "/x", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusAccepted)

		return errors.New("late")
	})

	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestAPIErrorHandler_ServerHTTPError_LogsCause(t *testing.T) {
	var buf strings.Builder
	rec := httptest.NewRecorder()
	rt := router.New(newAPIErrorHandler(slog.New(slog.NewTextHandler(&buf, nil))))
	rt.Register(http.MethodGet, "/x", func(http.ResponseWriter, *http.Request) error {
		return &router.HTTPError{Status: http.StatusInternalServerError, Err: errors.New("db down")}
	})

	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "db down")
	assert.Contains(t, buf.String(), "db down")
}
