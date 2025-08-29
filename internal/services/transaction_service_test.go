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
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Mock repositories
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, txType)
	return args.Get(0).(float64), args.Error(1)
}

type MockBudgetRepository struct {
	mock.Mock
}

func (m *MockBudgetRepository) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) Update(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

type MockCategoryRepositoryForTransactions struct {
	mock.Mock
}

func (m *MockCategoryRepositoryForTransactions) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

type MockUserRepositoryForTransactions struct {
	mock.Mock
}

func (m *MockUserRepositoryForTransactions) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

// Test fixtures
func setupTransactionService(t *testing.T) (
	*services.TransactionServiceImpl,
	*MockTransactionRepository,
	*MockBudgetRepository,
	*MockCategoryRepositoryForTransactions,
	*MockUserRepositoryForTransactions,
) {
	t.Helper()

	txRepo := &MockTransactionRepository{}
	budgetRepo := &MockBudgetRepository{}
	categoryRepo := &MockCategoryRepositoryForTransactions{}
	userRepo := &MockUserRepositoryForTransactions{}

	service := services.NewTransactionService(txRepo, budgetRepo, categoryRepo, userRepo)

	return service, txRepo, budgetRepo, categoryRepo, userRepo
}

func createTestTransaction() *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      100.50,
		Type:        transaction.TypeExpense,
		Description: "Test expense",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		FamilyID:    uuid.New(),
		Date:        time.Now(),
		Tags:        []string{"test"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func createTestUser(familyID uuid.UUID) *user.User {
	return &user.User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		FamilyID: familyID,
		Role:     user.RoleMember,
	}
}

func createTestCategory(familyID uuid.UUID) *category.Category {
	return &category.Category{
		ID:       uuid.New(),
		Name:     "Test Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
	}
}

func createTestBudget(familyID, categoryID uuid.UUID) *budget.Budget {
	return &budget.Budget{
		ID:         uuid.New(),
		Name:       "Test Budget",
		Amount:     500.00,
		Spent:      100.00,
		CategoryID: &categoryID,
		FamilyID:   familyID,
		IsActive:   true,
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
	}
}

// Test CreateTransaction
func TestTransactionService_CreateTransaction_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, userRepo := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      100.50,
		Type:        transaction.TypeExpense,
		Description: "Test expense",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
		Tags:        []string{"test"},
	}

	testUser := createTestUser(familyID)
	testUser.ID = userID

	testCategory := createTestCategory(familyID)
	testCategory.ID = categoryID

	testBudget := createTestBudget(familyID, categoryID)

	// Setup expectations
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	categoryRepo.On("GetByID", ctx, categoryID).Return(testCategory, nil)
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.InDelta(t, req.Amount, result.Amount, 0.01)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.Description, result.Description)
	assert.Equal(t, req.CategoryID, result.CategoryID)
	assert.Equal(t, req.UserID, result.UserID)
	assert.Equal(t, req.FamilyID, result.FamilyID)

	// Verify all expectations were met
	txRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestTransactionService_CreateTransaction_UserNotInFamily(t *testing.T) {
	service, _, _, _, userRepo := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      100.50,
		Type:        transaction.TypeExpense,
		Description: "Test expense",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
	}

	// User belongs to different family
	testUser := createTestUser(uuid.New())
	testUser.ID = userID

	// Setup expectations
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user does not belong to the specified family")

	userRepo.AssertExpectations(t)
}

func TestTransactionService_CreateTransaction_CategoryNotInFamily(t *testing.T) {
	service, _, _, categoryRepo, userRepo := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      100.50,
		Type:        transaction.TypeExpense,
		Description: "Test expense",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
	}

	testUser := createTestUser(familyID)
	testUser.ID = userID

	// Category belongs to different family
	testCategory := createTestCategory(uuid.New())
	testCategory.ID = categoryID

	// Setup expectations
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	categoryRepo.On("GetByID", ctx, categoryID).Return(testCategory, nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "category does not belong to the specified family")

	userRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
}

func TestTransactionService_CreateTransaction_ExceedsBudgetLimit(t *testing.T) {
	service, _, budgetRepo, categoryRepo, userRepo := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      500.00, // This exceeds the remaining budget (500 - 100 = 400)
		Type:        transaction.TypeExpense,
		Description: "Large expense",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
	}

	testUser := createTestUser(familyID)
	testUser.ID = userID

	testCategory := createTestCategory(familyID)
	testCategory.ID = categoryID

	testBudget := createTestBudget(familyID, categoryID)
	testBudget.Spent = 100.00 // Already spent 100 out of 500 budget

	// Setup expectations
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	categoryRepo.On("GetByID", ctx, categoryID).Return(testCategory, nil)
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{testBudget}, nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "transaction would exceed budget limit")

	userRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
}

func TestTransactionService_CreateTransaction_IncomeNoLimitCheck(t *testing.T) {
	service, txRepo, _, categoryRepo, userRepo := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      1000.00, // Large amount but it's income
		Type:        transaction.TypeIncome,
		Description: "Salary",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
	}

	testUser := createTestUser(familyID)
	testUser.ID = userID

	testCategory := createTestCategory(familyID)
	testCategory.ID = categoryID
	testCategory.Type = category.TypeIncome

	// Setup expectations - no budget check for income
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	categoryRepo.On("GetByID", ctx, categoryID).Return(testCategory, nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*transaction.Transaction")).Return(nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.InDelta(t, req.Amount, result.Amount, 0.01)
	assert.Equal(t, transaction.TypeIncome, result.Type)

	userRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test GetTransactionByID
func TestTransactionService_GetTransactionByID_Success(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	testTx := createTestTransaction()

	// Setup expectations
	txRepo.On("GetByID", ctx, testTx.ID).Return(testTx, nil)

	// Execute
	result, err := service.GetTransactionByID(ctx, testTx.ID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testTx, result)

	txRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionByID_NotFound(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	txID := uuid.New()

	// Setup expectations
	txRepo.On("GetByID", ctx, txID).Return(nil, errors.New("not found"))

	// Execute
	result, err := service.GetTransactionByID(ctx, txID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	txRepo.AssertExpectations(t)
}

// Test GetTransactionsByFamily
func TestTransactionService_GetTransactionsByFamily_Success(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	filter := dto.NewTransactionFilterDTO()
	filter.FamilyID = familyID

	testTxs := []*transaction.Transaction{createTestTransaction(), createTestTransaction()}

	// Setup expectations
	txRepo.On("GetByFilter", ctx, mock.AnythingOfType("transaction.Filter")).Return(testTxs, nil)

	// Execute
	result, err := service.GetTransactionsByFamily(ctx, familyID, filter)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testTxs, result)

	txRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionsByFamily_InvalidDateRange(t *testing.T) {
	service, _, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	filter := dto.NewTransactionFilterDTO()
	filter.FamilyID = familyID
	now := time.Now()
	filter.DateFrom = &now
	yesterday := now.AddDate(0, 0, -1)
	filter.DateTo = &yesterday // DateTo is before DateFrom

	// Execute
	result, err := service.GetTransactionsByFamily(ctx, familyID, filter)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "date_to must be after date_from")
}

// Test UpdateTransaction
func TestTransactionService_UpdateTransaction_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	newCategoryID := uuid.New()

	testTx := createTestTransaction()
	testTx.FamilyID = familyID
	testTx.CategoryID = categoryID

	newAmount := 150.75
	newDescription := "Updated expense"

	req := dto.UpdateTransactionDTO{
		Amount:      &newAmount,
		Description: &newDescription,
		CategoryID:  &newCategoryID,
	}

	testCategory := createTestCategory(familyID)
	testCategory.ID = newCategoryID

	testBudget := createTestBudget(familyID, newCategoryID)

	// Setup expectations
	txRepo.On("GetByID", ctx, testTx.ID).Return(testTx, nil)
	categoryRepo.On("GetByID", ctx, newCategoryID).Return(testCategory, nil)
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{testBudget}, nil)
	txRepo.On("Update", ctx, mock.AnythingOfType("*transaction.Transaction")).Return(nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).
		Return(nil)
		// Only one update call for the new category

	// Execute
	result, err := service.UpdateTransaction(ctx, testTx.ID, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.InDelta(t, newAmount, result.Amount, 0.01)
	assert.Equal(t, newDescription, result.Description)
	assert.Equal(t, newCategoryID, result.CategoryID)

	txRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
}

// Test DeleteTransaction
func TestTransactionService_DeleteTransaction_Success(t *testing.T) {
	service, txRepo, budgetRepo, _, _ := setupTransactionService(t)
	ctx := context.Background()

	testTx := createTestTransaction()
	testBudget := createTestBudget(testTx.FamilyID, testTx.CategoryID)

	// Setup expectations
	txRepo.On("GetByID", ctx, testTx.ID).Return(testTx, nil)
	txRepo.On("Delete", ctx, testTx.ID).Return(nil)
	budgetRepo.On("GetActiveBudgets", ctx, testTx.FamilyID).Return([]*budget.Budget{testBudget}, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.DeleteTransaction(ctx, testTx.ID)

	// Assert
	require.NoError(t, err)

	txRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
}

// Test BulkCategorizeTransactions
func TestTransactionService_BulkCategorizeTransactions_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	oldCategoryID := uuid.New()
	newCategoryID := uuid.New()

	tx1 := createTestTransaction()
	tx1.FamilyID = familyID
	tx1.CategoryID = oldCategoryID

	tx2 := createTestTransaction()
	tx2.FamilyID = familyID
	tx2.CategoryID = oldCategoryID

	transactionIDs := []uuid.UUID{tx1.ID, tx2.ID}

	testCategory := createTestCategory(familyID)
	testCategory.ID = newCategoryID

	oldBudget := createTestBudget(familyID, oldCategoryID)
	newBudget := createTestBudget(familyID, newCategoryID)

	// Setup expectations
	txRepo.On("GetByID", ctx, tx1.ID).Return(tx1, nil)
	txRepo.On("GetByID", ctx, tx2.ID).Return(tx2, nil)
	categoryRepo.On("GetByID", ctx, newCategoryID).Return(testCategory, nil).Times(2)
	txRepo.On("Update", ctx, mock.AnythingOfType("*transaction.Transaction")).Return(nil).Times(2)
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{oldBudget, newBudget}, nil).Times(4)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).
		Return(nil).
		Times(4)
		// Remove from old and add to new for both transactions

	// Execute
	err := service.BulkCategorizeTransactions(ctx, transactionIDs, newCategoryID)

	// Assert
	require.NoError(t, err)

	txRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
}

func TestTransactionService_BulkCategorizeTransactions_EmptyList(t *testing.T) {
	service, _, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	// Execute
	err := service.BulkCategorizeTransactions(ctx, []uuid.UUID{}, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transaction IDs provided")
}

// Test GetTransactionsByDateRange
func TestTransactionService_GetTransactionsByDateRange_Success(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	from := time.Now().AddDate(0, 0, -7)
	to := time.Now()

	testTxs := []*transaction.Transaction{createTestTransaction()}

	// Setup expectations
	txRepo.On("GetByFilter", ctx, mock.AnythingOfType("transaction.Filter")).Return(testTxs, nil)

	// Execute
	result, err := service.GetTransactionsByDateRange(ctx, familyID, from, to)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testTxs, result)

	txRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionsByDateRange_InvalidRange(t *testing.T) {
	service, _, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	from := time.Now()
	to := time.Now().AddDate(0, 0, -7) // to is before from

	// Execute
	result, err := service.GetTransactionsByDateRange(ctx, familyID, from, to)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "date_to must be after date_from")
}

// Test ValidateTransactionLimits
func TestTransactionService_ValidateTransactionLimits_WithinBudget(t *testing.T) {
	service, _, budgetRepo, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 200.0 // Within budget (500 - 100 = 400 remaining)

	testBudget := createTestBudget(familyID, categoryID)
	testBudget.Spent = 100.0

	// Setup expectations
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{testBudget}, nil)

	// Execute
	err := service.ValidateTransactionLimits(ctx, familyID, categoryID, amount, transaction.TypeExpense)

	// Assert
	require.NoError(t, err)

	budgetRepo.AssertExpectations(t)
}

func TestTransactionService_ValidateTransactionLimits_ExceedsBudget(t *testing.T) {
	service, _, budgetRepo, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 500.0 // Exceeds budget (500 - 100 = 400 remaining)

	testBudget := createTestBudget(familyID, categoryID)
	testBudget.Spent = 100.0

	// Setup expectations
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{testBudget}, nil)

	// Execute
	err := service.ValidateTransactionLimits(ctx, familyID, categoryID, amount, transaction.TypeExpense)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction would exceed budget limit")

	budgetRepo.AssertExpectations(t)
}

func TestTransactionService_ValidateTransactionLimits_IncomeTransaction(t *testing.T) {
	service, _, _, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 1000.0

	// Execute - income transactions should not check budget limits
	err := service.ValidateTransactionLimits(ctx, familyID, categoryID, amount, transaction.TypeIncome)

	// Assert
	require.NoError(t, err)
}

func TestTransactionService_ValidateTransactionLimits_NoBudget(t *testing.T) {
	service, _, budgetRepo, _, _ := setupTransactionService(t)
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 1000.0

	// Setup expectations - no budget found
	budgetRepo.On("GetActiveBudgets", ctx, familyID).Return([]*budget.Budget{}, nil)

	// Execute
	err := service.ValidateTransactionLimits(ctx, familyID, categoryID, amount, transaction.TypeExpense)

	// Assert
	require.NoError(t, err) // No budget means no limit

	budgetRepo.AssertExpectations(t)
}
