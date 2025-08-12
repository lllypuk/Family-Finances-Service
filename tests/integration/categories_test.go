package integration

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

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/handlers"
	"family-budget-service/internal/testhelpers"
)

func TestCategoryHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateCategory_Success", func(t *testing.T) {
		// First create a family
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		request := handlers.CreateCategoryRequest{
			Name:     "Food & Dining",
			Type:     "expense",
			Color:    "#FF5733",
			Icon:     "utensils",
			FamilyID: family.ID,
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Name, response.Data.Name)
		assert.Equal(t, request.Type, response.Data.Type)
		assert.Equal(t, request.Color, response.Data.Color)
		assert.Equal(t, request.Icon, response.Data.Icon)
		assert.Equal(t, request.FamilyID, response.Data.FamilyID)
		assert.NotEqual(t, uuid.Nil, response.Data.ID)
		assert.True(t, response.Data.IsActive)
	})

	t.Run("GetCategories_Success", func(t *testing.T) {
		// Create family and categories
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category1 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		category1.Name = "Food"
		category2 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeIncome)
		category2.Name = "Salary"

		err = testServer.Repos.Category.Create(context.Background(), category1)
		require.NoError(t, err)
		err = testServer.Repos.Category.Create(context.Background(), category2)
		require.NoError(t, err)

		// Get categories via API with family_id query parameter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories?family_id="+family.ID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2)

		categoryNames := []string{response.Data[0].Name, response.Data[1].Name}
		assert.Contains(t, categoryNames, "Food")
		assert.Contains(t, categoryNames, "Salary")
	})

	t.Run("GetCategoryByID_Success", func(t *testing.T) {
		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Get category via API
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories/"+category.ID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, category.ID, response.Data.ID)
		assert.Equal(t, category.Name, response.Data.Name)
		assert.Equal(t, string(category.Type), response.Data.Type)
		assert.Equal(t, category.FamilyID, response.Data.FamilyID)
	})

	t.Run("UpdateCategory_Success", func(t *testing.T) {
		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Update category
		newName := "Updated Food Category"
		newColor := "#28A745"
		request := handlers.UpdateCategoryRequest{
			Name:  &newName,
			Color: &newColor,
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			"/api/v1/categories/"+category.ID.String(),
			bytes.NewBuffer(requestBody),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, newName, response.Data.Name)
		assert.Equal(t, newColor, response.Data.Color)
		assert.Equal(t, string(category.Type), response.Data.Type) // Type should remain unchanged
	})

	t.Run("DeleteCategory_Success", func(t *testing.T) {
		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Delete category (soft delete)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+category.ID.String(), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// DELETE operations might return 200 or 204, both are valid
		assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNoContent)

		// Verify category is soft deleted by checking it's not returned in active categories
		req = httptest.NewRequest(http.MethodGet, "/api/v1/categories?family_id="+family.ID.String(), nil)
		rec = httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should be empty since the category is soft deleted
		assert.Empty(t, response.Data)
	})
}
