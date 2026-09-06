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
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

func listCategories(t *testing.T, testServer *testhelpers.TestServer) []handlers.CategoryResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	testServer.Auth(t).Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.CategoryResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	return response.Data
}

func TestCategoryHandler_Integration(t *testing.T) {
	t.Run("CreateCategory_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// First create a family
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		request := handlers.CreateCategoryRequest{
			Name:  "Food & Dining",
			Type:  "expense",
			Color: "#FF5733",
			Icon:  "utensils",
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
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
		assert.NotEqual(t, uuid.Nil, response.Data.ID)
		assert.True(t, response.Data.IsActive)
	})

	t.Run("GetCategories_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Create family and categories
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category1 := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		category1.Name = "Food"
		category2 := testhelpers.CreateTestCategory(family.ID, category.TypeIncome)
		category2.Name = "Salary"

		err = testServer.Repos.Category.Create(context.Background(), category1)
		require.NoError(t, err)
		err = testServer.Repos.Category.Create(context.Background(), category2)
		require.NoError(t, err)

		// Get categories via API with family_id query parameter
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories?family_id="+family.ID.String(), nil)
		testServer.Auth(t).Apply(req)
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
		testServer := testhelpers.SetupHTTPServer(t)

		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Get category via API
		req := httptest.NewRequest(http.MethodGet, "/api/v1/categories/"+category.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.CategoryResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, category.ID, response.Data.ID)
		assert.Equal(t, category.Name, response.Data.Name)
		assert.Equal(t, string(category.Type), response.Data.Type)
	})

	t.Run("UpdateCategory_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Update category
		newName := "Updated Food Category"
		newColor := "#28A745"
		request := handlers.UpdateCategoryRequest{
			Name:  &newName,
			Color: &newColor,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			"/api/v1/categories/"+category.ID.String(),
			bytes.NewBuffer(requestBodyBytes),
		)
		testServer.Auth(t).Apply(req)
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

	t.Run("CreateCategory_UnknownParent_422", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)

		missingParent := uuid.New()
		request := handlers.CreateCategoryRequest{
			Name:     "Orphan",
			Type:     "expense",
			ParentID: &missingParent,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("CreateCategory_DuplicateName_422", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)

		existing := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		existing.Name = "Food"
		require.NoError(t, testServer.Repos.Category.Create(t.Context(), existing))

		request := handlers.CreateCategoryRequest{Name: "Food", Type: "expense"}
		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	// categories.parent_id — ON DELETE SET NULL: удаляя родителя, сервис обязан взять список
	// подкатегорий заранее, иначе они остаются активными категориями верхнего уровня.
	t.Run("DeleteCategory_RemovesSubcategories", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		parent := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		parent.Name = "Transport"
		require.NoError(t, testServer.Repos.Category.Create(ctx, parent))

		child := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		child.Name = "Fuel"
		child.ParentID = &parent.ID
		require.NoError(t, testServer.Repos.Category.Create(ctx, child))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+parent.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)

		storedChild, err := testServer.Repos.Category.GetByID(ctx, child.ID)
		require.NoError(t, err)
		assert.False(t, storedChild.IsActive, "subcategory must be deactivated together with its parent")

		listReq := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
		testServer.Auth(t).Apply(listReq)
		listRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(listRec, listReq)

		require.Equal(t, http.StatusOK, listRec.Code)

		var listResponse handlers.APIResponse[[]handlers.CategoryResponse]
		require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResponse))
		assert.Empty(t, listResponse.Data)
	})

	// Категория с транзакциями раньше уходила в ветку Update, которая is_active не пишет:
	// ответ 204, а категория оставалась в списке.
	t.Run("DeleteCategory_UsedInTransactions", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		used := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		used.Name = "Groceries"
		require.NoError(t, testServer.Repos.Category.Create(ctx, used))
		testTransaction := testhelpers.CreateTestTransaction(
			testServer.AuthFamily.ID, testServer.AuthUser.ID, used.ID, transaction.TypeExpense,
		)
		require.NoError(t, testServer.Repos.Transaction.Create(ctx, testTransaction))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+used.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code, "тело: %s", rec.Body.String())

		stored, err := testServer.Repos.Category.GetByID(ctx, used.ID)
		require.NoError(t, err)
		assert.False(t, stored.IsActive, "категория с транзакциями обязана стать неактивной")

		assert.Empty(t, listCategories(t, testServer))
	})

	t.Run("DeleteCategory_ParentWithUsedSubcategory", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		parent := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		parent.Name = "Home"
		require.NoError(t, testServer.Repos.Category.Create(ctx, parent))

		child := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		child.Name = "Rent"
		child.ParentID = &parent.ID
		require.NoError(t, testServer.Repos.Category.Create(ctx, child))

		testTransaction := testhelpers.CreateTestTransaction(
			testServer.AuthFamily.ID, testServer.AuthUser.ID, child.ID, transaction.TypeExpense,
		)
		require.NoError(t, testServer.Repos.Transaction.Create(ctx, testTransaction))

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+parent.ID.String(), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code, "тело: %s", rec.Body.String())

		storedChild, err := testServer.Repos.Category.GetByID(ctx, child.ID)
		require.NoError(t, err)
		assert.False(t, storedChild.IsActive)

		storedParent, err := testServer.Repos.Category.GetByID(ctx, parent.ID)
		require.NoError(t, err)
		assert.False(t, storedParent.IsActive)

		assert.Empty(t, listCategories(t, testServer))
	})

	t.Run("DeleteCategory_Repeated_404", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		target := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		target.Name = "Travel"
		require.NoError(t, testServer.Repos.Category.Create(ctx, target))

		del := func() int {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+target.ID.String(), nil)
			testServer.Auth(t).Apply(req)
			rec := httptest.NewRecorder()
			testServer.Server.Echo().ServeHTTP(rec, req)
			return rec.Code
		}

		require.Equal(t, http.StatusNoContent, del())
		assert.Equal(t, http.StatusNotFound, del())
	})

	// Уникальность имени распространяется только на активные строки: у подкатегории parent_id
	// не NULL, поэтому без частичного индекса мягко удалённая строка навсегда занимала бы имя.
	t.Run("DeleteSubcategory_ThenRecreate", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		parent := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		parent.Name = "Home"
		require.NoError(t, testServer.Repos.Category.Create(ctx, parent))

		child := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		child.Name = "Rent"
		child.ParentID = &parent.ID
		require.NoError(t, testServer.Repos.Category.Create(ctx, child))

		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+child.ID.String(), nil)
		testServer.Auth(t).Apply(delReq)
		delRec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(delRec, delReq)
		require.Equal(t, http.StatusNoContent, delRec.Code)

		body, err := json.Marshal(handlers.CreateCategoryRequest{
			Name:     "Rent",
			Type:     "expense",
			Color:    "#FF5733",
			Icon:     "home",
			ParentID: &parent.ID,
		})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(body))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.APIResponse[handlers.CategoryResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "Rent", response.Data.Name)
		assert.NotEqual(t, child.ID, response.Data.ID)
	})

	// PUT по неактивной категории — 404: GetByID не фильтрует is_active, а WHERE is_active = 1
	// в репозитории не нашёл бы строку и дал 500.
	t.Run("UpdateCategory_AfterDelete_404", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)
		testServer.Auth(t)
		ctx := t.Context()

		target := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		target.Name = "Hobby"
		require.NoError(t, testServer.Repos.Category.Create(ctx, target))

		delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+target.ID.String(), nil)
		testServer.Auth(t).Apply(delReq)
		delRec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(delRec, delReq)
		require.Equal(t, http.StatusNoContent, delRec.Code)

		newName := "Hobby renamed"
		body, err := json.Marshal(handlers.UpdateCategoryRequest{Name: &newName})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/"+target.ID.String(), bytes.NewBuffer(body))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code, "тело: %s", rec.Body.String())
	})

	t.Run("DeleteCategory_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Create family and category
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		category := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), category)
		require.NoError(t, err)

		// Delete category (soft delete)
		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/v1/categories/"+category.ID.String()+"?family_id="+family.ID.String(),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// DELETE operations might return 200 or 204, both are valid
		assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNoContent)

		// Verify category is soft deleted by checking it's not returned in active categories
		req = httptest.NewRequest(http.MethodGet, "/api/v1/categories?family_id="+family.ID.String(), nil)
		testServer.Auth(t).Apply(req)
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
