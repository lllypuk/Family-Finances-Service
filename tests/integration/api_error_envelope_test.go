package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/testhelpers"
)

// Ошибки, не дошедшие до хендлера (404 роутера, 405, отказ middleware), тоже обязаны
// приходить в общем envelope: раньше клиент API получал на них HTML-страницу
// ошибки или строку plain text и не мог прочитать error.code (A-08).
func TestAPIErrorEnvelope_ErrorsOutsideHandlers(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	cases := []struct {
		name     string
		method   string
		target   string
		withAuth bool
		status   int
		code     string
	}{
		{"UnknownRoute", http.MethodGet, "/api/v1/does-not-exist", true, http.StatusNotFound, handlers.ErrCodeNotFound},
		{"UnknownRouteOutsideAPI", http.MethodGet, "/login", false, http.StatusNotFound, handlers.ErrCodeNotFound},
		// Внутри /api/v1 405 недостижим: catch-all группы перехватывает любой метод.
		{
			"MethodNotAllowed",
			http.MethodPost,
			"/health",
			false,
			http.StatusMethodNotAllowed,
			handlers.ErrCodeMethodNotAllowed,
		},
		{
			name: "WriteWithoutToken", method: http.MethodPost, target: "/api/v1/transactions", withAuth: false,
			status: http.StatusUnauthorized, code: handlers.ErrCodeUnauthorized,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(testCase.method, testCase.target, strings.NewReader(`{}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if testCase.withAuth {
				session.Apply(req)
			}
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			require.Equal(t, testCase.status, rec.Code, "тело: %s", rec.Body.String())
			assert.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, testCase.code, response.Error.Code)
			assert.NotEmpty(t, response.Error.Message)
			assert.NotEmpty(t, response.Meta.Version)
		})
	}
}
