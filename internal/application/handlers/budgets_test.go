package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
)

type MockBudgetRepository struct {
	mock.Mock
}

func (m *MockBudgetRepository) Create(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetAll(ctx context.Context) ([]*budget.Budget, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetActiveBudgets(ctx context.Context) ([]*budget.Budget, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) Update(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBudgetRepository) GetByCategory(ctx context.Context, categoryID *uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetByPeriod(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func setupBudgetHandler() (*handlers.BudgetHandler, *MockBudgetRepository, *MockTransactionRepository) {
	mockBudgetRepo := &MockBudgetRepository{}
	mockTxRepo := &MockTransactionRepository{}
	repositories := &handlers.Repositories{
		Budget:      mockBudgetRepo,
		Transaction: mockTxRepo,
	}
	handler := handlers.NewBudgetHandler(repositories)
	return handler, mockBudgetRepo, mockTxRepo
}

func TestBudgetHandler_GetBudgetByID_UsesCategoryDateRangeSpent(t *testing.T) {
	handler, mockBudgetRepo, mockTxRepo := setupBudgetHandler()

	budgetID := uuid.New()
	categoryID := uuid.New()
	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	foundBudget := &budget.Budget{
		ID:         budgetID,
		Name:       "Food",
		Amount:     1000,
		Spent:      50,
		Period:     budget.PeriodMonthly,
		CategoryID: &categoryID,
		StartDate:  startDate,
		EndDate:    endDate,
		IsActive:   true,
		CreatedAt:  startDate,
		UpdatedAt:  startDate,
	}

	mockBudgetRepo.On("GetByID", mock.Anything, budgetID).Return(foundBudget, nil).Once()
	mockTxRepo.On(
		"GetTotalByCategoryAndDateRange",
		mock.Anything,
		categoryID,
		startDate,
		endDate,
		mock.Anything,
	).Return(275.25, nil).Once()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets/"+budgetID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	err := handler.GetBudgetByID(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.InDelta(t, 275.25, response.Data.Spent, 0.001)
	assert.InDelta(t, 724.75, response.Data.Remaining, 0.001)

	mockTxRepo.AssertNotCalled(t, "GetTotalByCategory", mock.Anything, mock.Anything, mock.Anything)
	mockTxRepo.AssertNotCalled(t, "GetTotalByDateRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockBudgetRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestBudgetHandler_GetBudgetByID_FamilyBudgetUsesDateRangeSpent(t *testing.T) {
	handler, mockBudgetRepo, mockTxRepo := setupBudgetHandler()

	budgetID := uuid.New()
	startDate := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)
	foundBudget := &budget.Budget{
		ID:        budgetID,
		Name:      "Family Budget",
		Amount:    2000,
		Spent:     10,
		Period:    budget.PeriodMonthly,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  true,
		CreatedAt: startDate,
		UpdatedAt: startDate,
	}

	mockBudgetRepo.On("GetByID", mock.Anything, budgetID).Return(foundBudget, nil).Once()
	mockTxRepo.On(
		"GetTotalByDateRange",
		mock.Anything,
		startDate,
		endDate,
		mock.Anything,
	).Return(800.0, nil).Once()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets/"+budgetID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	err := handler.GetBudgetByID(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.InDelta(t, 800.0, response.Data.Spent, 0.001)
	assert.InDelta(t, 1200.0, response.Data.Remaining, 0.001)

	mockTxRepo.AssertNotCalled(t, "GetTotalByCategory", mock.Anything, mock.Anything, mock.Anything)
	mockTxRepo.AssertNotCalled(
		t,
		"GetTotalByCategoryAndDateRange",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
	mockBudgetRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}
