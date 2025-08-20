package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/handlers"
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
			FamilyID:  family.ID,
			Role:      "admin",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(requestBody))
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
		assert.Equal(t, request.FamilyID, response.Data.FamilyID)
		assert.Equal(t, request.Role, response.Data.Role)
		assert.NotEqual(t, uuid.Nil, response.Data.ID)
	})

	t.Run("CreateUser_ValidationError", func(t *testing.T) {
		request := handlers.CreateUserRequest{
			Email:     "invalid-email", // Invalid email format
			Password:  "123",           // Too short password
			FirstName: "",              // Empty first name
			LastName:  "Doe",
			FamilyID:  uuid.New(),
			Role:      "admin",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response handlers.APIResponse[any]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have multiple validation errors
		assert.NotEmpty(t, response.Errors)
		assert.GreaterOrEqual(t, len(response.Errors), 3) // At least 3 validation errors

		// Check for specific validation error codes
		errorCodes := make([]string, len(response.Errors))
		for i, e := range response.Errors {
			errorCodes[i] = e.Code
		}
		for _, code := range errorCodes {
			assert.Equal(t, "VALIDATION_ERROR", code)
		}
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
		assert.Equal(t, user.FamilyID, response.Data.FamilyID)
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+nonExistentID.String(), nil)
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

	t.Run("DeleteUser_Success", func(t *testing.T) {
		// Create family and user
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Delete user
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+user.ID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// DELETE operations might return 200 or 204, both are valid
		assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNoContent)

		// Verify user is deleted by trying to get it
		req = httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
		rec = httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
