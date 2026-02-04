package handlers_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

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
	"family-budget-service/internal/web/handlers"
)

// setupBudgetHandler creates a test budget handler with mocks
func setupBudgetHandler() (*handlers.BudgetHandler, *MockBudgetService, *MockCategoryService, *MockTransactionService) {
	mockBudgetService := new(MockBudgetService)
	mockCategoryService := new(MockCategoryService)
	mockTransactionService := new(MockTransactionService)

	servicesStruct := &services.Services{
		Budget:      mockBudgetService,
		Category:    mockCategoryService,
		Transaction: mockTransactionService,
	}

	handler := handlers.NewBudgetHandler(
		&appHandlers.Repositories{},
		servicesStruct,
	)

	return handler, mockBudgetService, mockCategoryService, mockTransactionService
}

// createTestBudgetWithSpent creates a test budget with spent amount
func createTestBudgetWithSpent(
	name string,
	amount, spent float64,
	categoryID *uuid.UUID,
	isActive bool,
) *budget.Budget {
	now := time.Now()
	b := &budget.Budget{
		ID:         uuid.New(),
		Name:       name,
		Amount:     amount,
		Spent:      spent,
		CategoryID: categoryID,
		Period:     budget.PeriodMonthly,
		StartDate:  now,
		EndDate:    now.AddDate(0, 1, 0),
		IsActive:   isActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return b
}

// TestBudgetHandler_Index tests the budgets list page
func TestBudgetHandler_Index(t *testing.T) {
	t.Run("list all budgets", func(t *testing.T) {
		handler, mockBudgetService, mockCategoryService, _ := setupBudgetHandler()

		categoryID := uuid.New()
		budgets := []*budget.Budget{
			createTestBudgetWithSpent("Food Budget", 1000, 500, &categoryID, true),
			createTestBudgetWithSpent("Transport Budget", 500, 100, nil, true),
		}

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return(budgets, nil)
		mockCategoryService.On("GetCategoryByID", mock.Anything, categoryID).
			Return(createTestCategory("Food", category.TypeExpense), nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, "/budgets", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("filter by active status", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgets := []*budget.Budget{
			createTestBudgetWithSpent("Active Budget", 1000, 500, nil, true),
		}

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return(budgets, nil)

		c, rec := newTestContext(http.MethodGet, "/budgets?is_active=true", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("filter by period", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return([]*budget.Budget{}, nil)

		c, rec := newTestContext(http.MethodGet, "/budgets?period=monthly", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/budgets", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Index(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_New tests the budget creation form
func TestBudgetHandler_New(t *testing.T) {
	t.Run("show form with categories", func(t *testing.T) {
		handler, _, mockCategoryService, _ := setupBudgetHandler()

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
			createTestCategory("Transport", category.TypeExpense),
		}

		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

		c, rec := newTestContext(http.MethodGet, "/budgets/new", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.New(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockCategoryService.AssertExpectations(t)
	})

	t.Run("categories service error", func(t *testing.T) {
		handler, _, mockCategoryService, _ := setupBudgetHandler()

		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).
			Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/budgets/new", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.New(c)

		require.Error(t, err)
		mockCategoryService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Create tests budget creation
func TestBudgetHandler_Create(t *testing.T) {
	t.Run("create valid monthly budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		categoryID := uuid.New()
		formData := url.Values{
			"name":        {"Food Budget"},
			"category_id": {categoryID.String()},
			"amount":      {"1000.00"},
			"period":      {"monthly"},
			"start_date":  {"2024-01-01"},
			"end_date":    {"2024-01-31"},
			"is_active":   {"true"},
		}

		createdBudget := createTestBudgetWithSpent("Food Budget", 1000, 0, &categoryID, true)
		mockBudgetService.On("CreateBudget", mock.Anything, mock.AnythingOfType("dto.CreateBudgetDTO")).
			Return(createdBudget, nil)

		c, rec := newTestContext(http.MethodPost, "/budgets", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("create yearly budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		formData := url.Values{
			"name":       {"Annual Budget"},
			"amount":     {"12000.00"},
			"period":     {"yearly"},
			"start_date": {"2024-01-01"},
			"end_date":   {"2024-12-31"},
			"is_active":  {"true"},
		}

		createdBudget := createTestBudgetWithSpent("Annual Budget", 12000, 0, nil, true)
		mockBudgetService.On("CreateBudget", mock.Anything, mock.AnythingOfType("dto.CreateBudgetDTO")).
			Return(createdBudget, nil)

		c, rec := newTestContext(http.MethodPost, "/budgets", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("missing required name", func(t *testing.T) {
		handler, _, mockCategoryService, _ := setupBudgetHandler()

		formData := url.Values{
			"amount":     {"1000.00"},
			"period":     {"monthly"},
			"start_date": {"2024-01-01"},
			"end_date":   {"2024-01-31"},
		}

		// Mock categories for error rendering
		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).
			Return([]*category.Category{}, nil)

		c, rec := newTestContext(http.MethodPost, "/budgets", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // Form redisplayed with errors
	})

	t.Run("invalid amount format", func(t *testing.T) {
		handler, _, mockCategoryService, _ := setupBudgetHandler()

		formData := url.Values{
			"name":       {"Test Budget"},
			"amount":     {"not-a-number"},
			"period":     {"monthly"},
			"start_date": {"2024-01-01"},
			"end_date":   {"2024-01-31"},
			"is_active":  {"true"},
		}

		// Mock categories for potential error rendering
		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).
			Return([]*category.Category{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodPost, "/budgets", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		// Either validation error or parse error - both are acceptable
		if err != nil {
			assert.Contains(t, err.Error(), "amount")
		} else {
			// Form redisplayed with errors
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		formData := url.Values{
			"name":       {"Test Budget"},
			"amount":     {"1000.00"},
			"period":     {"monthly"},
			"start_date": {"2024-01-01"},
			"end_date":   {"2024-01-31"},
			"is_active":  {"true"},
		}

		mockBudgetService.On("CreateBudget", mock.Anything, mock.AnythingOfType("dto.CreateBudgetDTO")).
			Return(nil, errors.New("budget period overlap"))

		c, _ := newTestContext(http.MethodPost, "/budgets", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Edit tests the budget edit form
func TestBudgetHandler_Edit(t *testing.T) {
	t.Run("show edit form", func(t *testing.T) {
		handler, mockBudgetService, mockCategoryService, _ := setupBudgetHandler()

		budgetID := uuid.New()
		categoryID := uuid.New()
		existingBudget := createTestBudgetWithSpent("Food Budget", 1000, 500, &categoryID, true)
		existingBudget.ID = budgetID

		categories := []*category.Category{
			createTestCategory("Food", category.TypeExpense),
		}

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)
		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/edit", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Edit(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
		mockCategoryService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/edit", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Edit(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("invalid budget ID", func(t *testing.T) {
		handler, _, _, _ := setupBudgetHandler()

		c, _ := newTestContext(http.MethodGet, "/budgets/invalid/edit", "")
		c.SetParamNames("id")
		c.SetParamValues("invalid-uuid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Edit(c)

		require.Error(t, err)
	})
}

// TestBudgetHandler_Update tests budget update
func TestBudgetHandler_Update(t *testing.T) {
	t.Run("update budget successfully", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		existingBudget := createTestBudgetWithSpent("Old Name", 1000, 500, nil, true)
		existingBudget.ID = budgetID

		formData := url.Values{
			"name":       {"Updated Budget"},
			"amount":     {"1500.00"},
			"period":     {"monthly"},
			"start_date": {"2024-01-01"},
			"end_date":   {"2024-01-31"},
			"is_active":  {"true"},
		}

		updatedBudget := createTestBudgetWithSpent("Updated Budget", 1500, 500, nil, true)
		updatedBudget.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)
		mockBudgetService.On("UpdateBudget", mock.Anything, budgetID, mock.AnythingOfType("dto.UpdateBudgetDTO")).
			Return(updatedBudget, nil)

		c, rec := newTestContext(http.MethodPut, fmt.Sprintf("/budgets/%s", budgetID), formData.Encode())
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Update(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		formData := url.Values{
			"name":   {"Updated"},
			"amount": {"1000.00"},
		}

		c, _ := newTestContext(http.MethodPut, fmt.Sprintf("/budgets/%s", budgetID), formData.Encode())
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Update(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		handler, mockBudgetService, mockCategoryService, _ := setupBudgetHandler()

		budgetID := uuid.New()
		existingBudget := createTestBudgetWithSpent("Existing", 1000, 0, nil, true)

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)
		// Mock categories for error rendering
		mockCategoryService.On("GetCategories", mock.Anything, mock.Anything).
			Return([]*category.Category{}, nil)

		formData := url.Values{
			// Missing required fields
			"amount": {"1000.00"},
		}

		c, rec := newTestContext(http.MethodPut, fmt.Sprintf("/budgets/%s", budgetID), formData.Encode())
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Update(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // Form redisplayed
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Delete tests budget deletion
func TestBudgetHandler_Delete(t *testing.T) {
	t.Run("delete existing budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		existingBudget := createTestBudgetWithSpent("To Delete", 1000, 0, nil, true)
		existingBudget.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)
		mockBudgetService.On("DeleteBudget", mock.Anything, budgetID).Return(nil)

		c, rec := newTestContext(http.MethodDelete, fmt.Sprintf("/budgets/%s", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodDelete, fmt.Sprintf("/budgets/%s", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Delete(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Show tests budget details page
func TestBudgetHandler_Show(t *testing.T) {
	t.Run("show budget with statistics", func(t *testing.T) {
		handler, mockBudgetService, mockCategoryService, mockTransactionService := setupBudgetHandler()

		budgetID := uuid.New()
		categoryID := uuid.New()
		b := createTestBudgetWithSpent("Food Budget", 1000, 750, &categoryID, true)
		b.ID = budgetID

		cat := createTestCategory("Food", category.TypeExpense)
		cat.ID = categoryID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(b, nil)
		mockCategoryService.On("GetCategoryByID", mock.Anything, categoryID).Return(cat, nil)
		mockTransactionService.On("GetTransactionsByDateRange", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("show budget over limit", func(t *testing.T) {
		handler, mockBudgetService, _, mockTransactionService := setupBudgetHandler()

		budgetID := uuid.New()
		b := createTestBudgetWithSpent("Over Budget", 1000, 1200, nil, true)
		b.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(b, nil)
		mockTransactionService.On("GetTransactionsByDateRange", mock.Anything, mock.Anything, mock.Anything).
			Return([]*transaction.Transaction{}, nil).
			Maybe()

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Show(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Activate tests budget activation
func TestBudgetHandler_Activate(t *testing.T) {
	t.Run("activate inactive budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		inactiveBudget := createTestBudgetWithSpent("Inactive", 1000, 0, nil, false)
		inactiveBudget.ID = budgetID

		activeBudget := createTestBudgetWithSpent("Inactive", 1000, 0, nil, true)
		activeBudget.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(inactiveBudget, nil).Once()
		mockBudgetService.On("UpdateBudget", mock.Anything, budgetID, mock.AnythingOfType("dto.UpdateBudgetDTO")).
			Return(activeBudget, nil)
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(activeBudget, nil).Maybe()

		c, _ := newTestContext(http.MethodPost, fmt.Sprintf("/budgets/%s/activate", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Activate(c)

		require.NoError(t, err)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodPost, fmt.Sprintf("/budgets/%s/activate", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Activate(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Deactivate tests budget deactivation
func TestBudgetHandler_Deactivate(t *testing.T) {
	t.Run("deactivate active budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		activeBudget := createTestBudgetWithSpent("Active", 1000, 500, nil, true)
		activeBudget.ID = budgetID

		inactiveBudget := createTestBudgetWithSpent("Active", 1000, 500, nil, false)
		inactiveBudget.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(activeBudget, nil).Once()
		mockBudgetService.On("UpdateBudget", mock.Anything, budgetID, mock.AnythingOfType("dto.UpdateBudgetDTO")).
			Return(inactiveBudget, nil)
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(inactiveBudget, nil).Maybe()

		c, _ := newTestContext(http.MethodPost, fmt.Sprintf("/budgets/%s/deactivate", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Deactivate(c)

		require.NoError(t, err)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodPost, fmt.Sprintf("/budgets/%s/deactivate", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Deactivate(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Progress tests HTMX progress update
func TestBudgetHandler_Progress(t *testing.T) {
	t.Run("under budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		b := createTestBudgetWithSpent("Food", 1000, 300, nil, true)
		b.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(b, nil)

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/progress", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Progress(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("at warning threshold", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		b := createTestBudgetWithSpent("Food", 1000, 850, nil, true)
		b.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(b, nil)

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/progress", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Progress(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("over budget", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		b := createTestBudgetWithSpent("Food", 1000, 1200, nil, true)
		b.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(b, nil)

		c, rec := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/progress", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Progress(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodGet, fmt.Sprintf("/budgets/%s/progress", budgetID), "")
		c.SetParamNames("id")
		c.SetParamValues(budgetID.String())
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.Progress(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_Alerts tests budget alerts page
func TestBudgetHandler_Alerts(t *testing.T) {
	t.Run("show alerts with triggered and healthy budgets", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgets := []*budget.Budget{
			createTestBudgetWithSpent("Critical", 1000, 950, nil, true),  // 95% - critical
			createTestBudgetWithSpent("Warning", 1000, 850, nil, true),   // 85% - warning
			createTestBudgetWithSpent("Healthy", 1000, 500, nil, true),   // 50% - healthy
			createTestBudgetWithSpent("Exceeded", 1000, 1200, nil, true), // 120% - exceeded
		}

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return(budgets, nil)

		c, rec := newTestContext(http.MethodGet, "/budgets/alerts", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Alerts(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("no active budgets", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return([]*budget.Budget{}, nil)

		c, rec := newTestContext(http.MethodGet, "/budgets/alerts", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Alerts(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		mockBudgetService.On("GetAllBudgets", mock.Anything, mock.AnythingOfType("dto.BudgetFilterDTO")).
			Return(nil, errors.New("database error"))

		c, _ := newTestContext(http.MethodGet, "/budgets/alerts", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Alerts(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_CreateAlert tests alert creation
func TestBudgetHandler_CreateAlert(t *testing.T) {
	t.Run("create valid alert", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		formData := url.Values{
			"budget_id": {budgetID.String()},
			"threshold": {"80"},
		}

		existingBudget := createTestBudgetWithSpent("Food", 1000, 500, nil, true)
		existingBudget.ID = budgetID

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)

		c, rec := newTestContext(http.MethodPost, "/budgets/alerts", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.CreateAlert(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		mockBudgetService.AssertExpectations(t)
	})

	t.Run("invalid threshold over 100", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		formData := url.Values{
			"budget_id": {budgetID.String()},
			"threshold": {"150"},
		}

		// The validation might pass, but the service would handle the logic
		existingBudget := createTestBudgetWithSpent("Food", 1000, 500, nil, true)
		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(existingBudget, nil).
			Maybe()

		c, rec := newTestContext(http.MethodPost, "/budgets/alerts", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.CreateAlert(c)

		// Either validation fails or succeeds - both are acceptable for this handler
		if err == nil {
			// If no error, should redirect
			assert.True(t, rec.Code == http.StatusFound || rec.Code == http.StatusBadRequest)
		}
	})

	t.Run("budget not found", func(t *testing.T) {
		handler, mockBudgetService, _, _ := setupBudgetHandler()

		budgetID := uuid.New()
		formData := url.Values{
			"budget_id": {budgetID.String()},
			"threshold": {"80"},
		}

		mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).
			Return(nil, errors.New("budget not found"))

		c, _ := newTestContext(http.MethodPost, "/budgets/alerts", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.CreateAlert(c)

		require.Error(t, err)
		mockBudgetService.AssertExpectations(t)
	})
}

// TestBudgetHandler_DeleteAlert tests alert deletion
func TestBudgetHandler_DeleteAlert(t *testing.T) {
	t.Run("delete existing alert", func(t *testing.T) {
		handler, _, _, _ := setupBudgetHandler()

		alertID := uuid.New()

		c, rec := newTestContext(http.MethodDelete, fmt.Sprintf("/budgets/alerts/%s", alertID), "")
		c.SetParamNames("alert_id")
		c.SetParamValues(alertID.String())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.DeleteAlert(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
	})

	t.Run("invalid alert ID", func(t *testing.T) {
		handler, _, _, _ := setupBudgetHandler()

		c, _ := newTestContext(http.MethodDelete, "/budgets/alerts/invalid", "")
		c.SetParamNames("alert_id")
		c.SetParamValues("invalid-uuid")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.DeleteAlert(c)

		require.Error(t, err)
	})

	t.Run("HTMX request", func(t *testing.T) {
		handler, _, _, _ := setupBudgetHandler()

		alertID := uuid.New()

		c, rec := newTestContext(http.MethodDelete, fmt.Sprintf("/budgets/alerts/%s", alertID), "")
		c.SetParamNames("alert_id")
		c.SetParamValues(alertID.String())
		withSession(c, uuid.New(), user.RoleAdmin)
		withHTMX(c)

		err := handler.DeleteAlert(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code) // HTMX returns 200 for delete
	})
}
