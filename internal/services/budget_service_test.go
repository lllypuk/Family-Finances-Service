package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Mock repositories for BudgetService
type MockBudgetRepositoryForService struct {
	mock.Mock
}

func (m *MockBudgetRepositoryForService) Create(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepositoryForService) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) GetAll(ctx context.Context) ([]*budget.Budget, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) GetActiveBudgets(
	ctx context.Context,
	on date.Date,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, on)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) Update(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepositoryForService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBudgetRepositoryForService) GetByCategory(
	ctx context.Context,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) GetByPeriod(
	ctx context.Context,
	startDate, endDate date.Date,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

type MockTransactionRepositoryForBudgets struct {
	mock.Mock
}

func (m *MockTransactionRepositoryForBudgets) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	txType transaction.Type,
) (money.Minor, error) {
	args := m.Called(ctx, categoryID, txType)
	return args.Get(0).(money.Minor), args.Error(1)
}

func (m *MockTransactionRepositoryForBudgets) GetTotalByDateRange(
	ctx context.Context,
	startDate, endDate date.Date,
	txType transaction.Type,
) (money.Minor, error) {
	args := m.Called(ctx, startDate, endDate, txType)
	return args.Get(0).(money.Minor), args.Error(1)
}

func (m *MockTransactionRepositoryForBudgets) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate date.Date,
	txType transaction.Type,
) (money.Minor, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, txType)
	return args.Get(0).(money.Minor), args.Error(1)
}

// Test fixtures
func setupBudgetService(t *testing.T) (
	*services.BudgetServiceImpl,
	*MockBudgetRepositoryForService,
	*MockTransactionRepositoryForBudgets,
) {
	t.Helper()

	budgetRepo := &MockBudgetRepositoryForService{}
	txRepo := &MockTransactionRepositoryForBudgets{}

	service := services.NewBudgetService(budgetRepo, txRepo)

	return service, budgetRepo, txRepo
}

func createTestBudgetForService() *budget.Budget {
	return &budget.Budget{
		ID:          uuid.New(),
		Name:        "Test Budget",
		AmountMinor: 100_000,
		SpentMinor:  30_000,
		Period:      budget.PeriodMonthly,
		CategoryID:  func() *uuid.UUID { id := uuid.New(); return &id }(),
		StartDate:   date.Today(time.UTC),
		EndDate:     date.Today(time.UTC).AddDays(30),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestBudgetDTO() dto.CreateBudgetDTO {
	categoryID := uuid.New()
	return dto.CreateBudgetDTO{
		Name:        "Test Budget",
		AmountMinor: 100_000,
		Period:      budget.PeriodMonthly,
		CategoryID:  &categoryID,
		StartDate:   date.Today(time.UTC),
		EndDate:     date.Today(time.UTC).AddDays(30),
	}
}

// Test CreateBudget
func TestBudgetService_CreateBudget_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()

	// Setup expectations
	budgetRepo.On(
		"GetByPeriod",
		ctx,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
	).Return([]*budget.Budget{}, nil)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)
	budgetRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(createTestBudgetForService(), nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(15000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.CreateBudget(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.AmountMinor, result.AmountMinor)
	assert.Equal(t, req.Period, result.Period)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_CreateBudget_InvalidPeriod(t *testing.T) {
	service, _, _ := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()
	req.EndDate = req.StartDate.AddDays(-1) // End before start

	// Execute
	result, err := service.CreateBudget(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "budget end date must be after start date")
}

func TestBudgetService_CreateBudget_PeriodOverlap(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()

	existingBudget := createTestBudgetForService()
	existingBudget.CategoryID = req.CategoryID
	existingBudget.StartDate = req.StartDate.AddDays(-5) // Overlapping period
	existingBudget.EndDate = req.StartDate.AddDays(5)

	// Setup expectations
	budgetRepo.On(
		"GetByPeriod",
		ctx,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
	).Return([]*budget.Budget{existingBudget}, nil)

	// Execute
	result, err := service.CreateBudget(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "budget period overlaps")

	budgetRepo.AssertExpectations(t)
}

// Test GetBudgetByID
func TestBudgetService_GetBudgetByID_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil) // Different from budget.Spent (300.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.GetBudgetByID(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testBudget.ID, result.ID)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_GetBudgetByID_NotFound(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	budgetID := uuid.New()

	// Setup expectations
	budgetRepo.On("GetByID", ctx, budgetID).Return(nil, errors.New("not found"))

	// Execute
	result, err := service.GetBudgetByID(ctx, budgetID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	budgetRepo.AssertExpectations(t)
}

// Test UpdateBudget
func TestBudgetService_UpdateBudget_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	newAmount := money.Minor(150_000)
	newName := "Updated Budget"

	req := dto.UpdateBudgetDTO{
		Name:        &newName,
		AmountMinor: &newAmount,
	}

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil) // Different from budget.Spent (300.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).
		Return(nil).Times(2) // Once for recalc, once for update

	// Execute
	result, err := service.UpdateBudget(ctx, testBudget.ID, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	assert.Equal(t, newAmount, result.AmountMinor)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_UpdateBudget_AmountLessThanSpent(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	testBudget.SpentMinor = 50000

	newAmount := money.Minor(40_000) // Less than spent amount

	req := dto.UpdateBudgetDTO{
		AmountMinor: &newAmount,
	}

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(45000), nil) // Different from testBudget.Spent (500.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil) // For recalc

	// Execute
	result, err := service.UpdateBudget(ctx, testBudget.ID, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "new amount")
	assert.Contains(t, err.Error(), "is less than spent")

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test GetActiveBudgets
func TestBudgetService_GetActiveBudgets_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	on := date.Today(time.UTC)

	activeBudget := createTestBudgetForService()
	activeBudget.StartDate = on.AddDays(-5)
	activeBudget.EndDate = on.AddDays(5)

	// Отбор по датам делает запрос репозитория; сервис только пересчитывает spent.
	budgetRepo.On("GetActiveBudgets", ctx, mock.AnythingOfType("date.Date")).
		Return([]*budget.Budget{activeBudget}, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*activeBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(20000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.GetActiveBudgets(ctx, on)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, activeBudget.ID, result[0].ID)
	assert.Equal(t, money.Minor(20000), result[0].SpentMinor)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test CheckBudgetLimits
func TestBudgetService_CheckBudgetLimits_WithinLimit(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	categoryID := uuid.New()
	amount := money.Minor(10_000)

	testBudget := createTestBudgetForService()
	testBudget.CategoryID = &categoryID
	testBudget.SpentMinor = 30000
	testBudget.AmountMinor = 100000

	// Setup expectations
	budgetRepo.On("GetByCategory", ctx, &categoryID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		categoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, categoryID, amount, date.Today(time.UTC))

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_CheckBudgetLimits_ExceedsLimit(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	categoryID := uuid.New()
	amount := money.Minor(80_000) // Would exceed limit (300 + 800 > 1000)

	testBudget := createTestBudgetForService()
	testBudget.CategoryID = &categoryID
	testBudget.SpentMinor = 30000
	testBudget.AmountMinor = 100000

	// Setup expectations
	budgetRepo.On("GetByCategory", ctx, &categoryID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		categoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, categoryID, amount, date.Today(time.UTC))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient budget funds")

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test GetBudgetStatus
func TestBudgetService_GetBudgetStatus_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	testBudget.SpentMinor = 80000 // 80% utilization
	testBudget.AmountMinor = 100000

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.GetBudgetStatus(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testBudget.ID, result.BudgetID)
	assert.Equal(t, testBudget.Name, result.Name)
	assert.Equal(t, money.Minor(25_000), result.SpentAmountMinor)
	assert.Equal(t, money.Minor(75_000), result.RemainingAmountMinor)
	assert.InDelta(t, 25.0, result.UtilizationPercent, 0.1)
	assert.Equal(t, 30, result.DaysTotal)
	// 100000 на 30 дней — 3333 с округлением половины от нуля, не 3333.33 и не 3334.
	assert.Equal(t, money.Minor(3_333), result.DailyBudgetMinor)
	assert.False(t, result.IsNearLimit)
	assert.False(t, result.IsOverBudget)
	assert.Equal(t, dto.BudgetStatusHealthy, result.Status)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test CalculateBudgetUtilization
func TestBudgetService_CalculateBudgetUtilization_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	testBudget.SpentMinor = 70000
	testBudget.AmountMinor = 100000
	testBudget.StartDate = date.Today(time.UTC).AddDays(-10) // 10 days ago

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(25000), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.CalculateBudgetUtilization(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testBudget.ID, result.BudgetID)
	assert.Equal(t, string(testBudget.Period), result.Period)
	assert.InDelta(t, 25.0, result.UtilizationPercent, 0.1)
	// 25000 копеек за 10 прошедших дней — ровно 2500 в день, без «просто больше нуля».
	assert.Equal(t, money.Minor(2_500), result.SpendingVelocityMinor)
	assert.NotEmpty(t, result.Recommendations)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test UpdateBudgetSpent
func TestBudgetService_UpdateBudgetSpent_Success(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	amount := money.Minor(5_000)

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.UpdateBudgetSpent(ctx, testBudget.ID, amount)

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
}

// Test DeleteBudget
func TestBudgetService_DeleteBudget_Success(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	budgetRepo.On("Delete", ctx, testBudget.ID).Return(nil)

	// Execute
	err := service.DeleteBudget(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
}

// Test GetAllBudgets
func TestBudgetService_GetAllBudgets_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	filter := dto.NewBudgetFilterDTO()

	testBudgets := []*budget.Budget{createTestBudgetForService(), createTestBudgetForService()}

	// Setup expectations
	budgetRepo.On("GetAll", ctx).Return(testBudgets, nil)
	for _, b := range testBudgets {
		txRepo.On(
			"GetTotalByCategoryAndDateRange",
			ctx,
			*b.CategoryID,
			mock.AnythingOfType("date.Date"),
			mock.AnythingOfType("date.Date"),
			transaction.TypeExpense,
		).Return(money.Minor(20000), nil)
		budgetRepo.On("Update", ctx, mock.MatchedBy(func(budget *budget.Budget) bool {
			return budget.ID == b.ID
		})).Return(nil)
	}

	// Execute
	result, err := service.GetAllBudgets(ctx, filter)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 2)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test RecalculateBudgetSpent
func TestBudgetService_RecalculateBudgetSpent_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	actualSpent := money.Minor(45_000)

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(actualSpent, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.RecalculateBudgetSpent(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test CheckBudgetLimits with no budgets
func TestBudgetService_CheckBudgetLimits_NoBudgets(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	categoryID := uuid.New()
	amount := money.Minor(100_000)

	// Setup expectations - no budgets found
	budgetRepo.On("GetByCategory", ctx, &categoryID).Return([]*budget.Budget{}, nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, categoryID, amount, date.Today(time.UTC))

	// Assert
	require.NoError(t, err) // No budgets means no limit

	budgetRepo.AssertExpectations(t)
}

// TestBudgetService_UpdateBudget_EndDateBeforeStoredStart — обновление одной границы
// проверяется по уже применённым значениям, иначе ошибка всплывала бы CHECK-ом в БД.
func TestBudgetService_UpdateBudget_EndDateBeforeStoredStart(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(0), nil).Maybe()
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil).Maybe()

	earlierEnd := testBudget.StartDate.AddDays(-1)
	result, err := service.UpdateBudget(ctx, testBudget.ID, dto.UpdateBudgetDTO{EndDate: &earlierEnd})

	require.ErrorIs(t, err, dto.ErrInvalidBudgetPeriod)
	assert.Nil(t, result)
}

// Границы периода включительные: общий день — пересечение.
func TestBudgetService_CreateBudget_PeriodOverlapSharedBoundaryDay(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()

	existingBudget := createTestBudgetForService()
	existingBudget.CategoryID = req.CategoryID
	existingBudget.StartDate = req.EndDate // общий день с концом нового периода
	existingBudget.EndDate = req.EndDate.AddDays(10)

	budgetRepo.On(
		"GetByPeriod",
		ctx,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
	).Return([]*budget.Budget{existingBudget}, nil)

	result, err := service.CreateBudget(ctx, req)

	require.ErrorIs(t, err, services.ErrBudgetOverlapExists)
	assert.Nil(t, result)

	budgetRepo.AssertExpectations(t)
}

func TestBudgetService_CreateBudget_PeriodsTouchWithoutSharedDay(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()

	existingBudget := createTestBudgetForService()
	existingBudget.CategoryID = req.CategoryID
	existingBudget.StartDate = req.EndDate.AddDays(1) // сосед начинается на следующий день
	existingBudget.EndDate = req.EndDate.AddDays(10)

	budgetRepo.On(
		"GetByPeriod",
		ctx,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
	).Return([]*budget.Budget{existingBudget}, nil)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)
	budgetRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(createTestBudgetForService(), nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(0), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil).Maybe()

	result, err := service.CreateBudget(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)

	budgetRepo.AssertExpectations(t)
}

func TestBudgetService_CreateBudget_SharedDayOtherCategory(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()

	existingBudget := createTestBudgetForService()
	otherCategory := uuid.New()
	existingBudget.CategoryID = &otherCategory
	existingBudget.StartDate = req.EndDate
	existingBudget.EndDate = req.EndDate.AddDays(10)

	budgetRepo.On(
		"GetByPeriod",
		ctx,
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
	).Return([]*budget.Budget{existingBudget}, nil)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)
	budgetRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(createTestBudgetForService(), nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(0), nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil).Maybe()

	result, err := service.CreateBudget(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)

	budgetRepo.AssertExpectations(t)
}

// Пара с общим днём, созданная до включительной проверки: переименование проходит, сдвиг
// конца упирается в соседа, сдвиг начала на день за него — проходит.
func TestBudgetService_UpdateBudget_LegacySharedDayPair(t *testing.T) {
	legacyPair := func(t *testing.T) (*services.BudgetServiceImpl, *budget.Budget) {
		t.Helper()

		service, budgetRepo, txRepo := setupBudgetService(t)

		neighbour := createTestBudgetForService()
		second := createTestBudgetForService()
		second.Name = "Second"
		second.CategoryID = neighbour.CategoryID
		second.StartDate = neighbour.EndDate // общий день
		second.EndDate = neighbour.EndDate.AddDays(20)

		budgetRepo.On("GetByID", mock.Anything, second.ID).Return(second, nil)
		budgetRepo.On("GetByPeriod", mock.Anything,
			mock.AnythingOfType("date.Date"), mock.AnythingOfType("date.Date")).
			Return([]*budget.Budget{neighbour, second}, nil).Maybe()
		txRepo.On("GetTotalByCategoryAndDateRange", mock.Anything,
			*second.CategoryID, mock.AnythingOfType("date.Date"), mock.AnythingOfType("date.Date"),
			transaction.TypeExpense).Return(money.Minor(0), nil)
		budgetRepo.On("Update", mock.Anything, mock.AnythingOfType("*budget.Budget")).Return(nil).Maybe()

		return service, second
	}

	t.Run("rename passes", func(t *testing.T) {
		service, second := legacyPair(t)
		newName := "Renamed"

		result, err := service.UpdateBudget(context.Background(), second.ID, dto.UpdateBudgetDTO{Name: &newName})

		require.NoError(t, err)
		assert.Equal(t, newName, result.Name)
	})

	t.Run("end date shift hits neighbour", func(t *testing.T) {
		service, second := legacyPair(t)
		newEnd := second.EndDate.AddDays(5)

		result, err := service.UpdateBudget(context.Background(), second.ID, dto.UpdateBudgetDTO{EndDate: &newEnd})

		require.ErrorIs(t, err, services.ErrBudgetOverlapExists)
		assert.Nil(t, result)
	})

	t.Run("start date shift past neighbour passes", func(t *testing.T) {
		service, second := legacyPair(t)
		newStart := second.StartDate.AddDays(1)

		result, err := service.UpdateBudget(context.Background(), second.ID, dto.UpdateBudgetDTO{StartDate: &newStart})

		require.NoError(t, err)
		assert.Equal(t, newStart, result.StartDate)
	})
}
