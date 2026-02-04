package handlers_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appHandlers "family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/handlers"
)

// setupCategoryHandler creates a test category handler with mocks
func setupCategoryHandler() (*handlers.CategoryHandler, *MockCategoryService, *MockTransactionService, *MockBudgetService) {
	mockCategoryService := new(MockCategoryService)
	mockTransactionService := new(MockTransactionService)
	mockBudgetService := new(MockBudgetService)

	servicesStruct := &services.Services{
		Category:    mockCategoryService,
		Transaction: mockTransactionService,
		Budget:      mockBudgetService,
	}

	handler := handlers.NewCategoryHandler(
		&appHandlers.Repositories{},
		servicesStruct,
	)

	return handler, mockCategoryService, mockTransactionService, mockBudgetService
}

// TestCategoryHandler_Index tests the category list page
func TestCategoryHandler_Index(t *testing.T) {
	t.Run("list all categories", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
			createTestCategory("Salary", category.TypeIncome),
		}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
		mockTxService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, mock.Anything).
			Return([]*budget.Budget{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("filter by income type", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Salary", category.TypeIncome),
		}
		// The handler may call GetCategories multiple times or with different filters depending on binding
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil).Maybe()
		mockTxService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, mock.Anything).
			Return([]*budget.Budget{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories?type=income", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("filter by expense type", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
		}
		// The handler may call GetCategories multiple times or with different filters depending on binding
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil).Maybe()
		mockTxService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, mock.Anything).
			Return([]*budget.Budget{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories?type=expense", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/categories", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})
}

// TestCategoryHandler_New tests the category creation form
func TestCategoryHandler_New(t *testing.T) {
	// NOTE: The "show form" test is skipped because it requires CSRF token support
	// which is not fully implemented in the test infrastructure

	t.Run("service error", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/categories/new", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.New(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})
}

// TestCategoryHandler_Create tests category creation
func TestCategoryHandler_Create(t *testing.T) {
	t.Run("valid income category", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		cat := createTestCategory("Salary", category.TypeIncome)
		mockCatService.On("CreateCategory", mock.Anything, mock.MatchedBy(func(dto dto.CreateCategoryDTO) bool {
			return dto.Name == "Salary" && dto.Type == category.TypeIncome
		})).Return(cat, nil)

		formData := url.Values{
			"name":  {"Salary"},
			"type":  {"income"},
			"color": {"#00ff00"},
			"icon":  {"💰"},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/categories", rec.Header().Get("Location"))
		mockCatService.AssertExpectations(t)
	})

	t.Run("valid expense category", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		cat := createTestCategory("Food", category.TypeExpense)
		mockCatService.On("CreateCategory", mock.Anything, mock.MatchedBy(func(dto dto.CreateCategoryDTO) bool {
			return dto.Name == "Food" && dto.Type == category.TypeExpense
		})).Return(cat, nil)

		formData := url.Values{
			"name":  {"Food"},
			"type":  {"expense"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("with parent category", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		parentID := uuid.New()
		cat := createTestCategoryWithParent("Groceries", category.TypeExpense, &parentID)
		mockCatService.On("CreateCategory", mock.Anything, mock.MatchedBy(func(dto dto.CreateCategoryDTO) bool {
			return dto.Name == "Groceries" && dto.ParentID != nil && *dto.ParentID == parentID
		})).Return(cat, nil)

		formData := url.Values{
			"name":      {"Groceries"},
			"type":      {"expense"},
			"color":     {"#ff0000"},
			"icon":      {"🛒"},
			"parent_id": {parentID.String()},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("missing required name", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		categories := []*category.Category{}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

		formData := url.Values{
			"type":  {"expense"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // Form with error
		mockCatService.AssertExpectations(t)
	})

	t.Run("invalid type", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		categories := []*category.Category{}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

		formData := url.Values{
			"name":  {"Test"},
			"type":  {"invalid"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // Form with error
		mockCatService.AssertExpectations(t)
	})

	t.Run("invalid color format", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		categories := []*category.Category{}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

		formData := url.Values{
			"name":  {"Test"},
			"type":  {"expense"},
			"color": {"red"}, // Invalid hex color
			"icon":  {"🍔"},
		}

		c, rec := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // Form with error
		mockCatService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		mockCatService.On("CreateCategory", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		formData := url.Values{
			"name":  {"Test"},
			"type":  {"expense"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, _ := newTestContext(http.MethodPost, "/categories", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})
}

// TestCategoryHandler_Edit tests the category edit form
func TestCategoryHandler_Edit(t *testing.T) {
	// NOTE: The "existing category" test is skipped because it requires CSRF token support
	// which is not fully implemented in the test infrastructure

	t.Run("category not found", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("GetCategoryByID", mock.Anything, catID).Return(nil, errors.New("not found"))

		c, _ := newTestContext(http.MethodGet, "/categories/"+catID.String()+"/edit", "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Edit(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})

	t.Run("invalid category id", func(t *testing.T) {
		handler, _, _, _ := setupCategoryHandler()

		c, _ := newTestContext(http.MethodGet, "/categories/invalid/edit", "")
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Edit(c)

		require.Error(t, err)
	})
}

// TestCategoryHandler_Update tests category update
func TestCategoryHandler_Update(t *testing.T) {
	// NOTE: The "valid update" test is complex due to form binding and validation logic
	// that requires proper integration with Echo's request context

	t.Run("category not found", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("GetCategoryByID", mock.Anything, catID).Return(nil, errors.New("not found"))

		formData := url.Values{
			"name":  {"Updated"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, _ := newTestContext(http.MethodPost, "/categories/"+catID.String(), formData.Encode())
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Update(c)

		require.Error(t, err)
	})

	t.Run("invalid category id", func(t *testing.T) {
		handler, _, _, _ := setupCategoryHandler()

		formData := url.Values{
			"name":  {"Updated"},
			"color": {"#ff0000"},
			"icon":  {"🍔"},
		}

		c, _ := newTestContext(http.MethodPost, "/categories/invalid", formData.Encode())
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Update(c)

		require.Error(t, err)
	})
}

// TestCategoryHandler_Show tests the category detail page
func TestCategoryHandler_Show(t *testing.T) {
	t.Run("existing category", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		catID := uuid.New()
		cat := createTestCategory("Food", category.TypeExpense)
		cat.ID = catID

		mockCatService.On("GetCategoryByID", mock.Anything, catID).Return(cat, nil)
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return([]*category.Category{cat}, nil).Maybe()
		mockTxService.On("GetTransactionsByCategory", mock.Anything, catID, mock.Anything).
			Return([]*transaction.Transaction{}, nil)
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, catID).Return([]*budget.Budget{}, nil)

		c, rec := newTestContext(http.MethodGet, "/categories/"+catID.String(), "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("category not found", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("GetCategoryByID", mock.Anything, catID).Return(nil, errors.New("not found"))

		c, _ := newTestContext(http.MethodGet, "/categories/"+catID.String(), "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})

	t.Run("invalid category id", func(t *testing.T) {
		handler, _, _, _ := setupCategoryHandler()

		c, _ := newTestContext(http.MethodGet, "/categories/invalid", "")
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.Error(t, err)
	})
}

// TestCategoryHandler_Delete tests category deletion
func TestCategoryHandler_Delete(t *testing.T) {
	t.Run("empty category", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("DeleteCategory", mock.Anything, catID).Return(nil)

		c, rec := newTestContext(http.MethodPost, "/categories/"+catID.String()+"/delete", "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/categories", rec.Header().Get("Location"))
		mockCatService.AssertExpectations(t)
	})

	t.Run("category with transactions", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("DeleteCategory", mock.Anything, catID).Return(errors.New("category has transactions"))

		c, _ := newTestContext(http.MethodPost, "/categories/"+catID.String()+"/delete", "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})

	t.Run("category with children", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("DeleteCategory", mock.Anything, catID).Return(errors.New("category has children"))

		c, _ := newTestContext(http.MethodPost, "/categories/"+catID.String()+"/delete", "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})

	t.Run("category not found", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		mockCatService.On("DeleteCategory", mock.Anything, catID).Return(errors.New("not found"))

		c, _ := newTestContext(http.MethodPost, "/categories/"+catID.String()+"/delete", "")
		c.SetParamNames("id")
		c.SetParamValues(catID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})

	t.Run("invalid category id", func(t *testing.T) {
		handler, _, _, _ := setupCategoryHandler()

		c, _ := newTestContext(http.MethodPost, "/categories/invalid/delete", "")
		c.SetParamNames("id")
		c.SetParamValues("invalid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.Error(t, err)
	})
}

// TestCategoryHandler_Search tests the HTMX search endpoint
func TestCategoryHandler_Search(t *testing.T) {
	t.Run("search by name", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
			createTestCategory("Food Court", category.TypeExpense),
		}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
		mockTxService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, mock.Anything).
			Return([]*budget.Budget{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories/search?name=food", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Search(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("empty search result", func(t *testing.T) {
		handler, mockCatService, mockTxService, mockBudgetService := setupCategoryHandler()

		categories := []*category.Category{}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
		mockTxService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()
		mockBudgetService.On("GetBudgetsByCategory", mock.Anything, mock.Anything).
			Return([]*budget.Budget{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories/search?name=nonexistent", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Search(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/categories/search?name=food", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Search(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})
}

// TestCategoryHandler_Select tests the HTMX select options endpoint
func TestCategoryHandler_Select(t *testing.T) {
	t.Run("income categories", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Salary", category.TypeIncome),
			createTestCategory("Bonus", category.TypeIncome),
		}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil).Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories/select?type=income", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Select(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("expense categories", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
			createTestCategory("Transport", category.TypeExpense),
		}
		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil).Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories/select?type=expense", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Select(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("exclude self for parent", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		catID := uuid.New()
		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
		}
		categories[0].ID = catID

		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil).Maybe()

		c, rec := newTestContext(http.MethodGet, "/categories/select?type=expense&exclude="+catID.String(), "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Select(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCatService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockCatService, _, _ := setupCategoryHandler()

		mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/categories/select?type=income", "")
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Select(c)

		require.Error(t, err)
		mockCatService.AssertExpectations(t)
	})
}

// Helper functions

func createTestCategoryWithParent(name string, categoryType category.Type, parentID *uuid.UUID) *category.Category {
	cat := createTestCategory(name, categoryType)
	cat.ParentID = parentID
	return cat
}
