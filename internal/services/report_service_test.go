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
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Create test transactions with same category
	categoryID := uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			100.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			200.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 2),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			150.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 3),
		),
	}

	// Setup mock expectations for transaction service
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Mock category lookup (once since all transactions have same category)
	cat := createTestCategory(categoryID, "Test Category", category.TypeExpense)
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
	assert.Equal(t, req.UserID, result.UserID)
	assert.Equal(t, req.Period, result.Period)
	assert.InEpsilon(t, 450.0, result.TotalExpenses, 0.01) // 100 + 200 + 150
	assert.Positive(t, result.AverageDaily)
	assert.Len(t, result.CategoryBreakdown, 1) // All transactions have same category
	assert.Len(t, result.DailyBreakdown, 3)    // 3 different days
	assert.Len(t, result.TopExpenses, 3)       // All 3 transactions

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestReportService_GenerateExpenseReport_NoTransactions(t *testing.T) {
	service, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Empty Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Setup mock expectations - no transactions
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return([]*transaction.Transaction{}, nil)

	// Execute
	result, err := service.GenerateExpenseReport(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Zero(t, result.TotalExpenses)
	assert.Zero(t, result.AverageDaily)
	assert.Empty(t, result.CategoryBreakdown)
	assert.Empty(t, result.DailyBreakdown)
	assert.Empty(t, result.TopExpenses)

	mockTransactionService.AssertExpectations(t)
}

// Tests for GenerateIncomeReport
func TestReportService_GenerateIncomeReport(t *testing.T) {
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Income Report",
		Type:      report.TypeIncome,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Create test income transactions with same category
	categoryID := uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			5000.0,
			transaction.TypeIncome,
			startDate.AddDate(0, 0, 1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			1000.0,
			transaction.TypeIncome,
			startDate.AddDate(0, 0, 15),
		),
	}

	// Setup mock expectations
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Mock category lookup (once since all transactions have same category)
	cat := createTestCategory(categoryID, "Salary", category.TypeIncome)
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
	assert.InEpsilon(t, 6000.0, result.TotalIncome, 0.01) // 5000 + 1000
	assert.Positive(t, result.AverageDaily)

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// Tests for GenerateBudgetComparisonReport
func TestReportService_GenerateBudgetComparisonReport(t *testing.T) {
	service, _, _, mockTransactionService, mockBudgetService, _ := setupReportService()
	ctx := context.Background()

	categoryID := uuid.New()
	period := report.PeriodMonthly

	// Create test budget
	budgets := []*budget.Budget{
		createTestBudget(uuid.New(), 1000.0, categoryID),
	}

	// Create test expense transactions
	transactions := []*transaction.Transaction{
		createTestTransaction(uuid.New(), 300.0, transaction.TypeExpense, time.Now().AddDate(0, 0, -10)),
		createTestTransaction(uuid.New(), 250.0, transaction.TypeExpense, time.Now().AddDate(0, 0, -5)),
	}

	// Setup mock expectations
	mockBudgetService.On("GetActiveBudgets", ctx, mock.AnythingOfType("time.Time")).Return(budgets, nil)
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, period, result.Period)
	assert.InEpsilon(t, 1000.0, result.TotalBudget, 0.01)
	assert.InEpsilon(t, 550.0, result.TotalSpent, 0.01)    // 300 + 250
	assert.InEpsilon(t, 450.0, result.TotalVariance, 0.01) // 1000 - 550
	assert.InDelta(t, 55.0, result.Utilization, 0.01)      // (550/1000) * 100

	mockBudgetService.AssertExpectations(t)
	mockTransactionService.AssertExpectations(t)
}

func TestReportService_GenerateBudgetComparisonReport_NoBudgets(t *testing.T) {
	service, _, _, _, mockBudgetService, _ := setupReportService()
	ctx := context.Background()

	period := report.PeriodMonthly

	// Setup mock expectations - no budgets
	mockBudgetService.On("GetActiveBudgets", ctx, mock.AnythingOfType("time.Time")).
		Return([]*budget.Budget{}, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Zero(t, result.TotalBudget)
	assert.Zero(t, result.TotalSpent)
	assert.Zero(t, result.TotalVariance)
	assert.Zero(t, result.Utilization)

	mockBudgetService.AssertExpectations(t)
}

// Tests for GenerateCashFlowReport
func TestReportService_GenerateCashFlowReport(t *testing.T) {
	service, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()

	// Create mixed transactions
	transactions := []*transaction.Transaction{
		createTestTransaction(uuid.New(), 5000.0, transaction.TypeIncome, from.AddDate(0, 0, 1)),
		createTestTransaction(uuid.New(), 300.0, transaction.TypeExpense, from.AddDate(0, 0, 2)),
		createTestTransaction(uuid.New(), 200.0, transaction.TypeExpense, from.AddDate(0, 0, 3)),
		createTestTransaction(uuid.New(), 1000.0, transaction.TypeIncome, from.AddDate(0, 0, 15)),
	}

	// Setup mock expectations
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Execute
	result, err := service.GenerateCashFlowReport(ctx, from, to)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.InEpsilon(t, 6000.0, result.TotalInflows, 0.01) // 5000 + 1000
	assert.InEpsilon(t, 500.0, result.TotalOutflows, 0.01) // 300 + 200
	assert.InEpsilon(t, 5500.0, result.NetCashFlow, 0.01)  // 6000 - 500

	mockTransactionService.AssertExpectations(t)
}

// Tests for SaveReport
func TestReportService_SaveReport(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	reportData := &dto.ExpenseReportDTO{
		Name:          req.Name,
		TotalExpenses: 1000.0,
		CategoryBreakdown: []dto.CategoryBreakdownItemDTO{
			{
				CategoryID:   uuid.New(),
				CategoryName: "Groceries",
				Amount:       1000.0,
				Percentage:   100.0,
				Count:        2,
			},
		},
		TopExpenses: []dto.TransactionSummaryDTO{
			{
				ID:          uuid.New(),
				Amount:      700.0,
				Description: "Weekly grocery run",
				Category:    "Groceries",
				Date:        startDate.AddDate(0, 0, 1),
			},
		},
	}

	var createdReport *report.Report
	// Setup mock expectations
	mockReportRepo.On("Create", ctx, mock.AnythingOfType("*report.Report")).
		Run(func(args mock.Arguments) {
			createdReport = args.Get(1).(*report.Report)
		}).
		Return(nil)

	// Execute
	result, err := service.SaveReport(ctx, reportData, report.TypeExpenses, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.UserID, result.UserID)
	require.NotNil(t, createdReport)
	assert.InEpsilon(t, 1000.0, createdReport.Data.TotalExpenses, 0.01)
	require.Len(t, createdReport.Data.CategoryBreakdown, 1)
	assert.Equal(t, "Groceries", createdReport.Data.CategoryBreakdown[0].CategoryName)
	assert.Equal(t, 2, createdReport.Data.CategoryBreakdown[0].Count)
	require.Len(t, createdReport.Data.TopExpenses, 1)
	assert.Equal(t, "Weekly grocery run", createdReport.Data.TopExpenses[0].Description)

	mockReportRepo.AssertExpectations(t)
}

func TestReportService_SaveReport_ConvertsSupportedTypes(t *testing.T) {
	tests := []struct {
		name       string
		reportType report.Type
		reportData any
		assertData func(t *testing.T, rep *report.Report)
	}{
		{
			name:       "income",
			reportType: report.TypeIncome,
			reportData: &dto.IncomeReportDTO{
				TotalIncome: 2500.0,
				CategoryBreakdown: []dto.CategoryBreakdownItemDTO{
					{
						CategoryID:   uuid.New(),
						CategoryName: "Salary",
						Amount:       2500.0,
						Percentage:   100.0,
						Count:        1,
					},
				},
				TopSources: []dto.TransactionSummaryDTO{
					{
						ID:          uuid.New(),
						Amount:      2500.0,
						Description: "January salary",
						Category:    "Salary",
						Date:        time.Now(),
					},
				},
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.InEpsilon(t, 2500.0, rep.Data.TotalIncome, 0.01)
				require.Len(t, rep.Data.CategoryBreakdown, 1)
				require.Len(t, rep.Data.TopExpenses, 1)
			},
		},
		{
			name:       "budget",
			reportType: report.TypeBudget,
			reportData: &dto.BudgetComparisonDTO{
				TotalSpent: 900.0,
				Categories: []dto.BudgetCategoryComparisonDTO{
					{
						CategoryID:   uuid.New(),
						CategoryName: "Groceries",
						BudgetAmount: 1000.0,
						ActualAmount: 900.0,
						Variance:     100.0,
						Utilization:  90.0,
					},
				},
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.InEpsilon(t, 900.0, rep.Data.TotalExpenses, 0.01)
				require.Len(t, rep.Data.BudgetComparison, 1)
				assert.Equal(t, "Groceries", rep.Data.BudgetComparison[0].BudgetName)
			},
		},
		{
			name:       "cash_flow",
			reportType: report.TypeCashFlow,
			reportData: &dto.CashFlowReportDTO{
				TotalInflows:  3000.0,
				TotalOutflows: 1200.0,
				NetCashFlow:   1800.0,
				DailyFlow: []dto.DailyCashFlowDTO{
					{
						Date:    time.Now(),
						Inflow:  1000.0,
						Outflow: 300.0,
						Balance: 700.0,
					},
				},
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.InEpsilon(t, 3000.0, rep.Data.TotalIncome, 0.01)
				assert.InEpsilon(t, 1200.0, rep.Data.TotalExpenses, 0.01)
				assert.InEpsilon(t, 1800.0, rep.Data.NetIncome, 0.01)
				require.Len(t, rep.Data.DailyBreakdown, 1)
			},
		},
		{
			name:       "category_breakdown",
			reportType: report.TypeCategoryBreak,
			reportData: &dto.CategoryBreakdownDTO{
				Categories: []dto.CategoryAnalysisDTO{
					{
						CategoryID:       uuid.New(),
						CategoryName:     "Transport",
						TotalAmount:      450.0,
						Percentage:       45.0,
						TransactionCount: 3,
					},
				},
			},
			assertData: func(t *testing.T, rep *report.Report) {
				require.Len(t, rep.Data.CategoryBreakdown, 1)
				assert.Equal(t, "Transport", rep.Data.CategoryBreakdown[0].CategoryName)
				assert.Equal(t, 3, rep.Data.CategoryBreakdown[0].Count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockReportRepo, _, _, _, _ := setupReportService()
			ctx := context.Background()

			req := dto.ReportRequestDTO{
				Name:      "Converted " + tt.name,
				Type:      tt.reportType,
				Period:    report.PeriodMonthly,
				UserID:    uuid.New(),
				StartDate: time.Now().AddDate(0, 0, -30),
				EndDate:   time.Now(),
			}

			var createdReport *report.Report
			mockReportRepo.On("Create", ctx, mock.AnythingOfType("*report.Report")).
				Run(func(args mock.Arguments) {
					createdReport = args.Get(1).(*report.Report)
				}).
				Return(nil)

			result, err := service.SaveReport(ctx, tt.reportData, tt.reportType, req)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, createdReport)
			tt.assertData(t, createdReport)
			mockReportRepo.AssertExpectations(t)
		})
	}
}

// Tests for GetReportByID
func TestReportService_GetReportByID(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportID := uuid.New()
	userID := uuid.New()

	expectedReport := &report.Report{
		ID:     reportID,
		Name:   "Test Report",
		Type:   report.TypeExpenses,
		UserID: userID,
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

// Tests for GetReports
func TestReportService_GetReports(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	userID := uuid.New()

	expectedReports := []*report.Report{
		{
			ID:     uuid.New(),
			Name:   "Report 1",
			Type:   report.TypeExpenses,
			UserID: userID,
		},
		{
			ID:     uuid.New(),
			Name:   "Report 2",
			Type:   report.TypeIncome,
			UserID: userID,
		},
	}

	// Setup mock expectations
	mockReportRepo.On("GetAll", ctx).Return(expectedReports, nil)

	// Execute
	result, err := service.GetReports(ctx, nil)

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
	service, mockReportRepo, _, _, _, _ := setupReportService()
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

// Test for BUG-002: Validate that TransactionFilterDTO has valid Limit when generating reports
func TestReportService_GenerateExpenseReport_ValidatesFilterLimit(t *testing.T) {
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	req := dto.ReportRequestDTO{
		Name:      "Test Report - Filter Validation",
		Type:      report.TypeExpenses,
		Period:    report.PeriodWeekly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Create minimal test data
	categoryID := uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			50.0,
			transaction.TypeExpense,
			startDate.AddDate(0, 0, 1),
		),
	}

	// Setup mock - the key assertion is that GetAllTransactions is called with a valid filter
	mockTransactionService.On("GetAllTransactions", ctx, mock.MatchedBy(func(filter dto.TransactionFilterDTO) bool {
		// Verify that Limit is set and valid (not 0)
		return filter.Limit > 0 && filter.Limit <= 1000
	})).Return(transactions, nil)

	// Mock category and user lookups
	cat := createTestCategory(categoryID, "Test Category", category.TypeExpense)
	mockCategoryService.On("GetCategoryByID", ctx, categoryID).Return(cat, nil)

	testUser := &user.User{
		ID:        transactions[0].UserID,
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}
	mockUserRepo.On("GetByID", ctx, transactions[0].UserID).Return(testUser, nil)

	// Execute - should not fail with validation error
	result, err := service.GenerateExpenseReport(ctx, req)

	// Assert
	require.NoError(t, err, "Expected no validation error for TransactionFilterDTO.Limit")
	assert.NotNil(t, result)

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}
