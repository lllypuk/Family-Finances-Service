package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

// Tests for GenerateExpenseReport
func TestReportService_GenerateExpenseReport(t *testing.T) {
	service, _, _, _, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		FamilyID:  familyID,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Create test transactions with same category
	categoryID := uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			familyID,
			categoryID,
			100.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			familyID,
			categoryID,
			200.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 2),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			familyID,
			categoryID,
			150.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 3),
		),
	}

	// Setup mock expectations for transaction service
	mockTransactionService.On("GetTransactionsByFamily", ctx, familyID, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Mock category lookup (once since all transactions have same category)
	cat := createTestCategory(categoryID, familyID, "Test Category", category.TypeExpense)
	mockCategoryService.On("GetCategoryByID", ctx, categoryID).Return(cat, nil)

	// Mock user lookup for getTopTransactions
	for _, tx := range transactions {
		testUser := &user.User{
			ID:        tx.UserID,
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
		}
		mockUserRepo.On("GetByID", ctx, tx.UserID).Return(testUser, nil)
	}

	// Execute
	result, err := service.GenerateExpenseReport(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.FamilyID, result.FamilyID)
	assert.Equal(t, req.UserID, result.UserID)
	assert.Equal(t, req.Period, result.Period)
	assert.Equal(t, 450.0, result.TotalExpenses) // 100 + 200 + 150
	assert.Positive(t, result.AverageDaily)
	assert.Len(t, result.CategoryBreakdown, 1) // All transactions have same category
	assert.Len(t, result.DailyBreakdown, 3)    // 3 different days
	assert.Len(t, result.TopExpenses, 3)       // All 3 transactions

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestReportService_GenerateExpenseReport_NoTransactions(t *testing.T) {
	service, _, _, _, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Empty Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		FamilyID:  familyID,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Setup mock expectations - no transactions
	mockTransactionService.On("GetTransactionsByFamily", ctx, familyID, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return([]*transaction.Transaction{}, nil)

	// Execute
	result, err := service.GenerateExpenseReport(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.TotalExpenses)
	assert.Equal(t, 0.0, result.AverageDaily)
	assert.Empty(t, result.CategoryBreakdown)
	assert.Empty(t, result.DailyBreakdown)
	assert.Empty(t, result.TopExpenses)

	mockTransactionService.AssertExpectations(t)
}

// Tests for GenerateIncomeReport
func TestReportService_GenerateIncomeReport(t *testing.T) {
	service, _, _, _, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Income Report",
		Type:      report.TypeIncome,
		Period:    report.PeriodMonthly,
		FamilyID:  familyID,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Create test income transactions with same category
	categoryID := uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			familyID,
			categoryID,
			5000.0,
			transaction.TypeIncome,
			startDate.AddDate(0, 0, 1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			familyID,
			categoryID,
			1000.0,
			transaction.TypeIncome,
			startDate.AddDate(0, 0, 15),
		),
	}

	// Setup mock expectations
	mockTransactionService.On("GetTransactionsByFamily", ctx, familyID, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Mock category lookup (once since all transactions have same category)
	cat := createTestCategory(categoryID, familyID, "Salary", category.TypeIncome)
	mockCategoryService.On("GetCategoryByID", ctx, categoryID).Return(cat, nil)

	// Mock user lookup for getTopTransactions
	for _, tx := range transactions {
		testUser := &user.User{
			ID:        tx.UserID,
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
		}
		mockUserRepo.On("GetByID", ctx, tx.UserID).Return(testUser, nil)
	}

	// Execute
	result, err := service.GenerateIncomeReport(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, 6000.0, result.TotalIncome) // 5000 + 1000
	assert.Positive(t, result.AverageDaily)

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// Tests for GenerateBudgetComparisonReport
func TestReportService_GenerateBudgetComparisonReport(t *testing.T) {
	service, _, _, _, _, _, mockTransactionService, mockBudgetService, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	categoryID := uuid.New()
	period := report.PeriodMonthly

	// Create test budget
	budgets := []*budget.Budget{
		createTestBudget(uuid.New(), familyID, 1000.0, categoryID),
	}

	// Create test expense transactions
	transactions := []*transaction.Transaction{
		createTestTransaction(uuid.New(), familyID, 300.0, transaction.TypeExpense, time.Now().AddDate(0, 0, -10)),
		createTestTransaction(uuid.New(), familyID, 250.0, transaction.TypeExpense, time.Now().AddDate(0, 0, -5)),
	}

	// Setup mock expectations
	mockBudgetService.On("GetActiveBudgets", ctx, familyID, mock.AnythingOfType("time.Time")).Return(budgets, nil)
	mockTransactionService.On("GetTransactionsByFamily", ctx, familyID, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, familyID, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, familyID, result.FamilyID)
	assert.Equal(t, period, result.Period)
	assert.Equal(t, 1000.0, result.TotalBudget)
	assert.Equal(t, 550.0, result.TotalSpent)         // 300 + 250
	assert.Equal(t, 450.0, result.TotalVariance)      // 1000 - 550
	assert.InDelta(t, 55.0, result.Utilization, 0.01) // (550/1000) * 100

	mockBudgetService.AssertExpectations(t)
	mockTransactionService.AssertExpectations(t)
}

func TestReportService_GenerateBudgetComparisonReport_NoBudgets(t *testing.T) {
	service, _, _, _, _, _, _, mockBudgetService, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	period := report.PeriodMonthly

	// Setup mock expectations - no budgets
	mockBudgetService.On("GetActiveBudgets", ctx, familyID, mock.AnythingOfType("time.Time")).
		Return([]*budget.Budget{}, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, familyID, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.TotalBudget)
	assert.Equal(t, 0.0, result.TotalSpent)
	assert.Equal(t, 0.0, result.TotalVariance)
	assert.Equal(t, 0.0, result.Utilization)

	mockBudgetService.AssertExpectations(t)
}

// Tests for GenerateCashFlowReport
func TestReportService_GenerateCashFlowReport(t *testing.T) {
	service, _, _, _, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()

	// Create mixed transactions
	transactions := []*transaction.Transaction{
		createTestTransaction(uuid.New(), familyID, 5000.0, transaction.TypeIncome, from.AddDate(0, 0, 1)),
		createTestTransaction(uuid.New(), familyID, 300.0, transaction.TypeExpense, from.AddDate(0, 0, 2)),
		createTestTransaction(uuid.New(), familyID, 200.0, transaction.TypeExpense, from.AddDate(0, 0, 3)),
		createTestTransaction(uuid.New(), familyID, 1000.0, transaction.TypeIncome, from.AddDate(0, 0, 15)),
	}

	// Setup mock expectations
	mockTransactionService.On("GetTransactionsByFamily", ctx, familyID, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Execute
	result, err := service.GenerateCashFlowReport(ctx, familyID, from, to)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, familyID, result.FamilyID)
	assert.Equal(t, 6000.0, result.TotalInflows) // 5000 + 1000
	assert.Equal(t, 500.0, result.TotalOutflows) // 300 + 200
	assert.Equal(t, 5500.0, result.NetCashFlow)  // 6000 - 500

	mockTransactionService.AssertExpectations(t)
}

// Tests for SaveReport
func TestReportService_SaveReport(t *testing.T) {
	service, mockReportRepo, _, _, _, _, _, _, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		FamilyID:  familyID,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	reportData := &dto.ExpenseReportDTO{
		Name:          req.Name,
		TotalExpenses: 1000.0,
	}

	// Setup mock expectations
	mockReportRepo.On("Create", ctx, mock.AnythingOfType("*report.Report")).Return(nil)

	// Execute
	result, err := service.SaveReport(ctx, reportData, report.TypeExpenses, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.FamilyID, result.FamilyID)
	assert.Equal(t, req.UserID, result.UserID)

	mockReportRepo.AssertExpectations(t)
}

// Tests for GetReportByID
func TestReportService_GetReportByID(t *testing.T) {
	service, mockReportRepo, _, _, _, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportID := uuid.New()
	familyID := uuid.New()
	userID := uuid.New()

	expectedReport := &report.Report{
		ID:       reportID,
		Name:     "Test Report",
		Type:     report.TypeExpenses,
		FamilyID: familyID,
		UserID:   userID,
	}

	// Setup mock expectations
	mockReportRepo.On("GetByID", ctx, reportID).Return(expectedReport, nil)

	// Execute
	result, err := service.GetReportByID(ctx, reportID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedReport.ID, result.ID)
	assert.Equal(t, expectedReport.Name, result.Name)
	assert.Equal(t, expectedReport.Type, result.Type)

	mockReportRepo.AssertExpectations(t)
}

// Tests for GetReportsByFamily
func TestReportService_GetReportsByFamily(t *testing.T) {
	service, mockReportRepo, _, _, _, _, _, _, _ := setupReportService()
	ctx := context.Background()

	familyID := uuid.New()
	userID := uuid.New()

	expectedReports := []*report.Report{
		{
			ID:       uuid.New(),
			Name:     "Report 1",
			Type:     report.TypeExpenses,
			FamilyID: familyID,
			UserID:   userID,
		},
		{
			ID:       uuid.New(),
			Name:     "Report 2",
			Type:     report.TypeIncome,
			FamilyID: familyID,
			UserID:   userID,
		},
	}

	// Setup mock expectations
	mockReportRepo.On("GetByFamilyID", ctx, familyID).Return(expectedReports, nil)

	// Execute
	result, err := service.GetReportsByFamily(ctx, familyID, nil)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedReports[0].ID, result[0].ID)
	assert.Equal(t, expectedReports[1].ID, result[1].ID)

	mockReportRepo.AssertExpectations(t)
}

// Tests for DeleteReport
func TestReportService_DeleteReport(t *testing.T) {
	service, mockReportRepo, _, _, _, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportID := uuid.New()

	// Setup mock expectations
	mockReportRepo.On("Delete", ctx, reportID).Return(nil)

	// Execute
	err := service.DeleteReport(ctx, reportID)

	// Assert
	require.NoError(t, err)

	mockReportRepo.AssertExpectations(t)
}
