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
	"family-budget-service/internal/services/dto"
)

// Test helpers and mocks are now in test_helpers_test.go

// Test CreateTransaction
func TestTransactionService_CreateTransaction_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, userRepo := setupTransactionService()
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

	testCategory := createTestCategory(categoryID, familyID, "Test Category", category.TypeExpense)

	testBudget := createTestBudget(uuid.New(), familyID, 500.00, categoryID)

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

func TestTransactionService_CreateTransaction_UserNotFound(t *testing.T) {
	service, _, _, _, userRepo := setupTransactionService()
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

	// Setup expectations - user not found
	userRepo.On("GetByID", ctx, userID).Return((*user.User)(nil), errors.New("user not found"))

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)

	userRepo.AssertExpectations(t)
}

func TestTransactionService_CreateTransaction_ExceedsBudget(t *testing.T) {
	service, _, budgetRepo, categoryRepo, userRepo := setupTransactionService()
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

	testCategory := createTestCategory(categoryID, familyID, "Test Category", category.TypeExpense)

	testBudget := createTestBudget(uuid.New(), familyID, 500.00, categoryID)
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
	service, txRepo, _, categoryRepo, userRepo := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()

	req := dto.CreateTransactionDTO{
		Amount:      300.0, // Income transaction
		Type:        transaction.TypeIncome,
		Description: "Test income",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        time.Now(),
		Tags:        []string{"test"},
	}

	testUser := createTestUser(familyID)
	testUser.ID = userID

	testCategory := createTestCategory(categoryID, familyID, "Test Category", category.TypeIncome)

	// Setup expectations - no budget check for income
	userRepo.On("GetByID", ctx, userID).Return(testUser, nil)
	categoryRepo.On("GetByID", ctx, categoryID).Return(testCategory, nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*transaction.Transaction")).Return(nil)

	// Execute
	result, err := service.CreateTransaction(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.InEpsilon(t, req.Amount, result.Amount, 0.01)
	assert.Equal(t, req.Type, result.Type)

	userRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
	txRepo.AssertExpectations(t)
}

// Test GetTransactionByID
func TestTransactionService_GetTransactionByID_Success(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService()
	ctx := context.Background()

	testTx := createTestTransaction(uuid.New(), uuid.New(), 100.50, transaction.TypeExpense, time.Now())

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
	service, txRepo, _, _, _ := setupTransactionService()
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
	service, txRepo, _, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	filter := dto.NewTransactionFilterDTO()
	filter.FamilyID = familyID

	testTxs := []*transaction.Transaction{
		createTestTransaction(uuid.New(), familyID, 100.0, transaction.TypeExpense, time.Now()),
		createTestTransaction(uuid.New(), familyID, 200.0, transaction.TypeExpense, time.Now()),
	}

	// Setup expectations
	txRepo.On("GetByFilter", ctx, mock.AnythingOfType("transaction.Filter")).Return(testTxs, nil)

	// Execute
	result, err := service.GetTransactionsByFamily(ctx, familyID, filter)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testTxs, result)
	assert.Len(t, result, 2)

	txRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionsByFamily_EmptyResult(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	filter := dto.NewTransactionFilterDTO()
	filter.FamilyID = familyID

	// Setup expectations - empty result
	txRepo.On("GetByFilter", ctx, mock.AnythingOfType("transaction.Filter")).Return([]*transaction.Transaction{}, nil)

	// Execute
	result, err := service.GetTransactionsByFamily(ctx, familyID, filter)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result)

	txRepo.AssertExpectations(t)
}

// Test UpdateTransaction
func TestTransactionService_UpdateTransaction_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	newCategoryID := uuid.New()

	testTx := createTestTransaction(uuid.New(), familyID, 100.50, transaction.TypeExpense, time.Now())
	testTx.FamilyID = familyID
	testTx.CategoryID = categoryID

	newAmount := 150.75
	newDescription := "Updated expense"

	req := dto.UpdateTransactionDTO{
		Amount:      &newAmount,
		Description: &newDescription,
		CategoryID:  &newCategoryID,
	}

	testCategory := createTestCategory(newCategoryID, familyID, "Test Category", category.TypeExpense)

	testBudget := createTestBudget(uuid.New(), familyID, 500.00, newCategoryID)

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
	service, txRepo, budgetRepo, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	existingTx := createTestTransaction(uuid.New(), familyID, 100.50, transaction.TypeExpense, time.Now())
	existingTx.CategoryID = categoryID
	testBudget := createTestBudget(uuid.New(), familyID, 500.00, categoryID)

	// Setup expectations
	txRepo.On("GetByID", ctx, existingTx.ID).Return(existingTx, nil)
	txRepo.On("Delete", ctx, existingTx.ID, existingTx.FamilyID).Return(nil)
	budgetRepo.On("GetActiveBudgets", ctx, existingTx.FamilyID).Return([]*budget.Budget{testBudget}, nil)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)

	// Execute
	err := service.DeleteTransaction(ctx, existingTx.ID, existingTx.FamilyID)

	// Assert
	require.NoError(t, err)

	txRepo.AssertExpectations(t)
	budgetRepo.AssertExpectations(t)
}

// Test BulkCategorizeTransactions
func TestTransactionService_BulkCategorizeTransactions_Success(t *testing.T) {
	service, txRepo, budgetRepo, categoryRepo, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	oldCategoryID := uuid.New()
	newCategoryID := uuid.New()

	tx1 := createTestTransaction(uuid.New(), familyID, 100.50, transaction.TypeExpense, time.Now())
	tx1.CategoryID = oldCategoryID

	tx2 := createTestTransaction(uuid.New(), familyID, 200.00, transaction.TypeExpense, time.Now())
	tx2.CategoryID = oldCategoryID

	transactionIDs := []uuid.UUID{tx1.ID, tx2.ID}

	testCategory := createTestCategory(newCategoryID, familyID, "Test Category", category.TypeExpense)

	oldBudget := createTestBudget(uuid.New(), familyID, 500.00, oldCategoryID)
	newBudget := createTestBudget(uuid.New(), familyID, 300.00, newCategoryID)

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
	service, _, _, _, _ := setupTransactionService()
	ctx := context.Background()

	// Execute
	err := service.BulkCategorizeTransactions(ctx, []uuid.UUID{}, uuid.New())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no transaction IDs provided")
}

// Test GetTransactionsByDateRange
func TestTransactionService_GetTransactionsByDateRange_Success(t *testing.T) {
	service, txRepo, _, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	from := time.Now().AddDate(0, 0, -7)
	to := time.Now()

	testTxs := []*transaction.Transaction{
		createTestTransaction(uuid.New(), familyID, 100.0, transaction.TypeExpense, time.Now()),
	}

	// Setup expectations
	txRepo.On("GetByFilter", ctx, mock.AnythingOfType("transaction.Filter")).Return(testTxs, nil)

	// Execute
	result, err := service.GetTransactionsByDateRange(ctx, familyID, from, to)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testTxs, result)
	assert.Len(t, result, 1)

	txRepo.AssertExpectations(t)
}

func TestTransactionService_GetTransactionsByDateRange_EmptyResult(t *testing.T) {
	service, _, _, _, _ := setupTransactionService()
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
	service, _, budgetRepo, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 200.0 // Within budget (500 - 100 = 400 remaining)

	testBudget := createTestBudget(uuid.New(), familyID, 500.00, categoryID)
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
	service, _, budgetRepo, _, _ := setupTransactionService()
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	amount := 500.0 // Exceeds budget (500 - 100 = 400 remaining)

	testBudget := createTestBudget(uuid.New(), familyID, 500.00, categoryID)
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
	service, _, _, _, _ := setupTransactionService()
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
	service, _, budgetRepo, _, _ := setupTransactionService()
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
