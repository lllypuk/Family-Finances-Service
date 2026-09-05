package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

func TestUserHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateUser_Success", func(t *testing.T) {
		// First create a family
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		request := handlers.CreateUserRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Role:      "admin",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(requestBody))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.UserResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Email, response.Data.Email)
		assert.Equal(t, request.FirstName, response.Data.FirstName)
		assert.Equal(t, request.LastName, response.Data.LastName)
		assert.Equal(t, request.Role, response.Data.Role)
		assert.NotEqual(t, uuid.Nil, response.Data.ID)
	})

	t.Run("CreateUser_ValidationError", func(t *testing.T) {
		request := handlers.CreateUserRequest{
			Email:     "invalid-email", // Invalid email format
			Password:  "123",           // Too short password
			FirstName: "",              // Empty first name
			LastName:  "Doe",
			Role:      "admin",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(requestBody))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		var response handlers.ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have validation error
		assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
		assert.Equal(t, "Validation failed", response.Error.Message)
		assert.NotEmpty(t, response.Error.Details) // Should contain validation details

		// Check for specific validation error codes
		t.Logf("Validation error details: %s", response.Error.Details)
	})

	t.Run("GetUserByID_Success", func(t *testing.T) {
		// Create family and user
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Get user via API
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.UserResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, user.ID, response.Data.ID)
		assert.Equal(t, user.Email, response.Data.Email)
		assert.Equal(t, user.FirstName, response.Data.FirstName)
		assert.Equal(t, user.LastName, response.Data.LastName)
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+nonExistentID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response handlers.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response.Error.Code)
		assert.NotEmpty(t, response.Error.Message)
	})

	t.Run("UpdateUser_Success", func(t *testing.T) {
		// Create family and user
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Update user
		newFirstName := "Jane"
		newLastName := "Smith"
		request := handlers.UpdateUserRequest{
			FirstName: &newFirstName,
			LastName:  &newLastName,
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+user.ID.String(), bytes.NewBuffer(requestBody))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.UserResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, newFirstName, response.Data.FirstName)
		assert.Equal(t, newLastName, response.Data.LastName)
		assert.Equal(t, user.Email, response.Data.Email) // Email should remain unchanged
	})

	t.Run("Deactivate_ViaPatch", func(t *testing.T) {
		// Пользователь заводится в ТОЙ ЖЕ семье, что и владелец сессии:
		// репозитории берут family_id как `SELECT id FROM families LIMIT 1`.
		testServer.Auth(t)

		user := testhelpers.CreateTestUser(testServer.AuthFamily.ID)
		err := testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+user.ID.String(),
			bytes.NewBufferString(`{"is_active":false}`))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		// Настоящего удаления нет: пользователь остаётся, но с is_active=false.
		req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec = httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.UserResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.False(t, response.Data.IsActive)
	})
}

// TestUsersAPI_ListAndPatch покрывает GET /api/v1/users и PATCH /api/v1/users/:id.
func TestUsersAPI_ListAndPatch(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)
	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	do := func(t *testing.T, method, path, body string, auth *testhelpers.AuthSession) *httptest.ResponseRecorder {
		t.Helper()

		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		auth.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	t.Run("List_Admin", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/users", "", adminAuth)
		require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.APIResponse[[]handlers.UserResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		emails := make([]string, 0, len(response.Data))
		for _, u := range response.Data {
			emails = append(emails, u.Email)
		}
		assert.Contains(t, emails, testServer.AuthUser.Email)
		assert.Contains(t, emails, member.Email)
	})

	t.Run("List_MemberForbidden", func(t *testing.T) {
		rec := do(t, http.MethodGet, "/api/v1/users", "", memberAuth)
		assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
	})

	// Понижение единственного админа оставило бы семью без администратора.
	t.Run("Patch_LastAdminConflict", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/users/"+testServer.AuthUser.ID.String(),
			`{"role":"member"}`, adminAuth)
		require.Equal(t, http.StatusConflict, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "LAST_ADMIN", response.Error.Code)
	})

	t.Run("Patch_PromotesMember", func(t *testing.T) {
		rec := do(t, http.MethodPatch, "/api/v1/users/"+member.ID.String(), `{"role":"admin"}`, adminAuth)
		require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.APIResponse[handlers.UserResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "admin", response.Data.Role)

		stored, err := testServer.Repos.User.GetByID(context.Background(), member.ID)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, stored.Role)
	})
}
