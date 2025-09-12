package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	webHandlers "family-budget-service/internal/web/handlers"
)

func TestBaseHandler_Creation(t *testing.T) {
	handler := webHandlers.NewBaseHandler(nil, nil)
	assert.NotNil(t, handler)
}

func TestBaseHandler_IsHTMXRequest(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "Regular request",
			headers:  map[string]string{},
			expected: false,
		},
		{
			name: "HTMX request",
			headers: map[string]string{
				"Hx-Request": "true",
			},
			expected: true,
		},
		{
			name: "HTMX request with false value",
			headers: map[string]string{
				"Hx-Request": "false",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			handler := webHandlers.NewBaseHandler(nil, nil)
			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			isHTMX := handler.IsHTMXRequest(c)

			// Assert
			assert.Equal(t, tt.expected, isHTMX)
		})
	}
}
