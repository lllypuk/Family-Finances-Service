package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
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

func (m *MockBudgetRepository) GetActiveBudgets(ctx context.Context, on date.Date) ([]*budget.Budget, error) {
	args := m.Called(ctx, on)
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
	startDate, endDate date.Date,
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
	startDate := date.New(2026, time.January, 1)
	endDate := date.New(2026, time.January, 31)
	foundBudget := &budget.Budget{
		ID:          budgetID,
		Name:        "Food",
		AmountMinor: 100_000,
		SpentMinor:  5_000,
		Period:      budget.PeriodMonthly,
		CategoryID:  &categoryID,
		StartDate:   startDate,
		EndDate:     endDate,
		IsActive:    true,
		CreatedAt:   startDate.In(time.UTC),
		UpdatedAt:   startDate.In(time.UTC),
	}

	mockBudgetRepo.On("GetByID", mock.Anything, budgetID).Return(foundBudget, nil).Once()
	mockTxRepo.On(
		"GetTotalByCategoryAndDateRange",
		mock.Anything,
		categoryID,
		startDate,
		endDate,
		mock.Anything,
	).Return(money.Minor(27_525), nil).Once()

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
	assert.Equal(t, money.Minor(27_525), response.Data.SpentMinor)
	assert.Equal(t, money.Minor(72_475), response.Data.RemainingMinor)

	mockTxRepo.AssertNotCalled(t, "GetTotalByCategory", mock.Anything, mock.Anything, mock.Anything)
	mockTxRepo.AssertNotCalled(t, "GetTotalByDateRange", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockBudgetRepo.AssertExpectations(t)
	mockTxRepo.AssertExpectations(t)
}

func TestBudgetHandler_GetBudgetByID_FamilyBudgetUsesDateRangeSpent(t *testing.T) {
	handler, mockBudgetRepo, mockTxRepo := setupBudgetHandler()

	budgetID := uuid.New()
	startDate := date.New(2026, time.February, 1)
	endDate := date.New(2026, time.February, 28)
	foundBudget := &budget.Budget{
		ID:          budgetID,
		Name:        "Family Budget",
		AmountMinor: 200_000,
		SpentMinor:  1_000,
		Period:      budget.PeriodMonthly,
		StartDate:   startDate,
		EndDate:     endDate,
		IsActive:    true,
		CreatedAt:   startDate.In(time.UTC),
		UpdatedAt:   startDate.In(time.UTC),
	}

	mockBudgetRepo.On("GetByID", mock.Anything, budgetID).Return(foundBudget, nil).Once()
	mockTxRepo.On(
		"GetTotalByDateRange",
		mock.Anything,
		startDate,
		endDate,
		mock.Anything,
	).Return(money.Minor(80_000), nil).Once()

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
	assert.Equal(t, money.Minor(80_000), response.Data.SpentMinor)
	assert.Equal(t, money.Minor(120_000), response.Data.RemainingMinor)

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

// stubBudgetService отдаёт заранее заданную ошибку из UpdateBudget; остальной интерфейс
// не вызывается и остаётся в встроенном nil.
type stubBudgetService struct {
	services.BudgetService

	err error
}

func (s stubBudgetService) UpdateBudget(
	_ context.Context,
	_ uuid.UUID,
	_ dto.UpdateBudgetDTO,
) (*budget.Budget, error) {
	return nil, s.err
}

func TestBudgetHandler_UpdateBudget_BusinessConflicts(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "overlap", err: services.ErrBudgetOverlapExists, code: handlers.ErrCodeBudgetOverlap},
		{name: "name_exists", err: services.ErrBudgetNameExists, code: handlers.ErrCodeBudgetNameExists},
		{name: "below_spent", err: services.ErrBudgetAlreadyExceeded, code: handlers.ErrCodeBudgetBelowSpent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewBudgetHandler(&handlers.Repositories{}, stubBudgetService{err: tt.err})

			budgetID := uuid.New()
			e := echo.New()
			req := httptest.NewRequest(
				http.MethodPut,
				"/budgets/"+budgetID.String(),
				strings.NewReader(`{"amount_minor":1000}`),
			)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(budgetID.String())

			require.NoError(t, handler.UpdateBudget(c))
			assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.code, response.Error.Code)
			assert.Empty(t, response.Error.Details)
		})
	}
}

// TestBudgetHandler_UpdateBudget_AmountTooLargeStays422 — отказ формы остаётся валидацией.
func TestBudgetHandler_UpdateBudget_AmountTooLargeStays422(t *testing.T) {
	handler := handlers.NewBudgetHandler(
		&handlers.Repositories{},
		stubBudgetService{err: services.ErrBudgetAmountTooLarge},
	)

	budgetID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		"/budgets/"+budgetID.String(),
		strings.NewReader(`{"amount_minor":1000}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	require.NoError(t, handler.UpdateBudget(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
}
