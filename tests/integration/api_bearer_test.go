package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Bearer-путь /api/v1 до удаления cookie (план 03, задача 3): токен из auth.Service принимается
// той же группой, что и cookie-сессия; мусор и отозванный токен — 401 в общем envelope.
func TestAPIBearer_TokenFromAuthService(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	ts.Auth(t)

	const password = "correct-horse-battery"
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)
	member := testhelpers.CreateTestUser(ts.AuthFamily.ID)
	member.Role = user.RoleMember
	member.Password = hash
	require.NoError(t, ts.Repos.User.Create(t.Context(), member))

	token, _, err := ts.Services.Auth.Login(t.Context(), member.Email, password, "integration")
	require.NoError(t, err)

	get := func(authorization string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
		if authorization != "" {
			req.Header.Set(echo.HeaderAuthorization, authorization)
		}
		rec := httptest.NewRecorder()
		ts.Server.Echo().ServeHTTP(rec, req)
		return rec
	}

	t.Run("valid token", func(t *testing.T) {
		rec := get("Bearer " + token)
		assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	})

	t.Run("garbage token", func(t *testing.T) {
		rec := get("Bearer garbage")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		errBody, ok := body["error"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "UNAUTHORIZED", errBody["code"])
	})

	t.Run("role gate applies to bearer principal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
		rec := httptest.NewRecorder()
		ts.Server.Echo().ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code, "member не допущен к /users")
	})

	t.Run("revoked token", func(t *testing.T) {
		principal, authErr := ts.Services.Auth.Authenticate(t.Context(), token)
		require.NoError(t, authErr)
		require.NoError(t, ts.Services.Auth.Logout(t.Context(), principal.SessionID))

		rec := get("Bearer " + token)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
