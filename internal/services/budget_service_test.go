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

func (m *MockBudgetRepositoryForService) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) GetActiveBudgets(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
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

func (m *MockBudgetRepositoryForService) GetByFamilyAndCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryForService) GetByPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
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
) (float64, error) {
	args := m.Called(ctx, categoryID, txType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepositoryForBudgets) GetTotalByFamilyAndDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, familyID, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepositoryForBudgets) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
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
		ID:         uuid.New(),
		Name:       "Test Budget",
		Amount:     1000.00,
		Spent:      300.00,
		Period:     budget.PeriodMonthly,
		CategoryID: func() *uuid.UUID { id := uuid.New(); return &id }(),
		FamilyID:   uuid.New(),
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func createTestBudgetDTO() dto.CreateBudgetDTO {
	familyID := uuid.New()
	categoryID := uuid.New()
	return dto.CreateBudgetDTO{
		Name:       "Test Budget",
		Amount:     1000.00,
		Period:     budget.PeriodMonthly,
		CategoryID: &categoryID,
		FamilyID:   familyID,
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
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
		req.FamilyID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
	).Return([]*budget.Budget{}, nil)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)
	budgetRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(createTestBudgetForService(), nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(150.0, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.CreateBudget(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.InDelta(t, req.Amount, result.Amount, 0.01)
	assert.Equal(t, req.Period, result.Period)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_CreateBudget_InvalidPeriod(t *testing.T) {
	service, _, _ := setupBudgetService(t)
	ctx := context.Background()

	req := createTestBudgetDTO()
	req.EndDate = req.StartDate.AddDate(0, 0, -1) // End before start

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
	existingBudget.StartDate = req.StartDate.AddDate(0, 0, -5) // Overlapping period
	existingBudget.EndDate = req.StartDate.AddDate(0, 0, 5)

	// Setup expectations
	budgetRepo.On(
		"GetByPeriod",
		ctx,
		req.FamilyID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
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
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil) // Different from budget.Spent (300.0) to trigger update
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
	newAmount := 1500.0
	newName := "Updated Budget"

	req := dto.UpdateBudgetDTO{
		Name:   &newName,
		Amount: &newAmount,
	}

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil) // Different from budget.Spent (300.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).
		Return(nil).Times(2) // Once for recalc, once for update

	// Execute
	result, err := service.UpdateBudget(ctx, testBudget.ID, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	assert.InDelta(t, newAmount, result.Amount, 0.01)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_UpdateBudget_AmountLessThanSpent(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	testBudget.Spent = 500.0

	newAmount := 400.0 // Less than spent amount

	req := dto.UpdateBudgetDTO{
		Amount: &newAmount,
	}

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(450.0, nil) // Different from testBudget.Spent (500.0) to trigger update
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

	familyID := uuid.New()
	date := time.Now()

	activeBudget := createTestBudgetForService()
	activeBudget.FamilyID = familyID
	activeBudget.StartDate = date.AddDate(0, 0, -5)
	activeBudget.EndDate = date.AddDate(0, 0, 5)

	inactiveBudget := createTestBudgetForService()
	inactiveBudget.FamilyID = familyID
	inactiveBudget.StartDate = date.AddDate(0, 0, -20)
	inactiveBudget.EndDate = date.AddDate(0, 0, -10) // Expired

	allBudgets := []*budget.Budget{activeBudget, inactiveBudget}

	// Setup expectations
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return(allBudgets, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*activeBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(200.0, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.GetActiveBudgets(ctx, familyID, date)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1) // Only active budget should be returned
	assert.Equal(t, activeBudget.ID, result[0].ID)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test CheckBudgetLimits
func TestBudgetService_CheckBudgetLimits_WithinLimit(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 100.0

	testBudget := createTestBudgetForService()
	testBudget.FamilyID = familyID
	testBudget.CategoryID = &categoryID
	testBudget.Spent = 300.0
	testBudget.Amount = 1000.0

	// Setup expectations
	budgetRepo.On("GetByFamilyAndCategory", ctx, familyID, &categoryID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		categoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, familyID, categoryID, amount)

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

func TestBudgetService_CheckBudgetLimits_ExceedsLimit(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 800.0 // Would exceed limit (300 + 800 > 1000)

	testBudget := createTestBudgetForService()
	testBudget.FamilyID = familyID
	testBudget.CategoryID = &categoryID
	testBudget.Spent = 300.0
	testBudget.Amount = 1000.0

	// Setup expectations
	budgetRepo.On("GetByFamilyAndCategory", ctx, familyID, &categoryID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		categoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, familyID, categoryID, amount)

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
	testBudget.Spent = 800.0 // 80% utilization
	testBudget.Amount = 1000.0

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil) // Different from budget.Spent (300.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.GetBudgetStatus(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testBudget.ID, result.BudgetID)
	assert.Equal(t, testBudget.Name, result.Name)
	assert.InDelta(t, 250.0, result.SpentAmount, 0.01)
	assert.InDelta(t, 750.0, result.RemainingAmount, 0.01)
	assert.InDelta(t, 25.0, result.UtilizationPercent, 0.1)
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
	testBudget.Spent = 700.0
	testBudget.Amount = 1000.0
	testBudget.StartDate = time.Now().AddDate(0, 0, -10) // 10 days ago

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
		transaction.TypeExpense,
	).Return(250.0, nil) // Different from budget.Spent (800.0) to trigger update
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.CalculateBudgetUtilization(ctx, testBudget.ID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testBudget.ID, result.BudgetID)
	assert.Equal(t, string(testBudget.Period), result.Period)
	assert.InDelta(t, 25.0, result.UtilizationPercent, 0.1)
	assert.Greater(t, result.SpendingVelocity, 0.0)
	assert.NotEmpty(t, result.Recommendations)

	budgetRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test UpdateBudgetSpent
func TestBudgetService_UpdateBudgetSpent_Success(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)
	ctx := context.Background()

	testBudget := createTestBudgetForService()
	amount := 50.0

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

// Test GetBudgetsByFamily
func TestBudgetService_GetBudgetsByFamily_Success(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	familyID := uuid.New()
	filter := dto.NewBudgetFilterDTO()
	filter.FamilyID = familyID

	testBudgets := []*budget.Budget{createTestBudgetForService(), createTestBudgetForService()}
	for _, b := range testBudgets {
		b.FamilyID = familyID
	}

	// Setup expectations
	budgetRepo.On("GetByFamilyID", ctx, familyID).Return(testBudgets, nil)
	for _, b := range testBudgets {
		txRepo.On(
			"GetTotalByCategoryAndDateRange",
			ctx,
			*b.CategoryID,
			mock.AnythingOfType("time.Time"),
			mock.AnythingOfType("time.Time"),
			transaction.TypeExpense,
		).Return(200.0, nil)
		budgetRepo.On("Update", ctx, mock.MatchedBy(func(budget *budget.Budget) bool {
			return budget.ID == b.ID
		})).Return(nil)
	}

	// Execute
	result, err := service.GetBudgetsByFamily(ctx, familyID, filter)

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
	actualSpent := 450.0

	// Setup expectations
	budgetRepo.On("GetByID", ctx, testBudget.ID).Return(testBudget, nil)
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		ctx,
		*testBudget.CategoryID,
		mock.AnythingOfType("time.Time"),
		mock.AnythingOfType("time.Time"),
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

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 1000.0

	// Setup expectations - no budgets found
	budgetRepo.On("GetByFamilyAndCategory", ctx, familyID, &categoryID).Return([]*budget.Budget{}, nil)

	// Execute
	err := service.CheckBudgetLimits(ctx, familyID, categoryID, amount)

	// Assert
	require.NoError(t, err) // No budgets means no limit

	budgetRepo.AssertExpectations(t)
}
