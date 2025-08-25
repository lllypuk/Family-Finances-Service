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

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/testhelpers"
)

func TestFamilyHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateFamily_Success", func(t *testing.T) {
		request := handlers.CreateFamilyRequest{
			Name:     "Test Family",
			Currency: "USD",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.FamilyResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Name, response.Data.Name)
		assert.Equal(t, request.Currency, response.Data.Currency)
		assert.NotEqual(t, uuid.Nil, response.Data.ID)
	})

	t.Run("CreateFamily_InvalidRequest", func(t *testing.T) {
		request := handlers.CreateFamilyRequest{
			Name:     "", // Empty name should fail validation
			Currency: "USD",
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response handlers.APIResponse[any]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		t.Logf("Actual validation response: %+v", response)
		// Validation errors should be in the Errors field
		assert.NotEmpty(t, response.Errors)
		assert.Len(t, response.Errors, 1) // Should have one validation error for empty name
		assert.Equal(t, "VALIDATION_ERROR", response.Errors[0].Code)
	})

	t.Run("GetFamilyByID_Success", func(t *testing.T) {
		// First create a family
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		// Now get it via API
		req := httptest.NewRequest(http.MethodGet, "/api/v1/families/"+family.ID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.FamilyResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, family.ID, response.Data.ID)
		assert.Equal(t, family.Name, response.Data.Name)
		assert.Equal(t, family.Currency, response.Data.Currency)
	})

	t.Run("GetFamilyByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/families/"+nonExistentID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response handlers.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		t.Logf("Not found error response: %+v", response)
		assert.NotEmpty(t, response.Error.Code)
	})

	t.Run("GetFamilyByID_InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/families/invalid-uuid", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response handlers.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		t.Logf("Invalid UUID error response: %+v", response)
		t.Logf("Invalid UUID error code: %s", response.Error.Code)
		assert.NotEmpty(t, response.Error.Code)
	})

	t.Run("GetFamilyMembers_Success", func(t *testing.T) {
		// Create family and users
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user1 := testhelpers.CreateTestUser(family.ID)
		user1.Email = "user1@example.com"
		user2 := testhelpers.CreateTestUser(family.ID)
		user2.Email = "user2@example.com"

		err = testServer.Repos.User.Create(context.Background(), user1)
		require.NoError(t, err)
		err = testServer.Repos.User.Create(context.Background(), user2)
		require.NoError(t, err)

		// Get family members via API
		req := httptest.NewRequest(http.MethodGet, "/api/v1/families/"+family.ID.String()+"/members", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.UserResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2)

		userEmails := []string{response.Data[0].Email, response.Data[1].Email}
		assert.Contains(t, userEmails, "user1@example.com")
		assert.Contains(t, userEmails, "user2@example.com")
	})
}
