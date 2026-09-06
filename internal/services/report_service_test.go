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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Tests for GenerateExpenseReport
func TestReportService_GenerateExpenseReport(t *testing.T) {
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := date.Today(time.UTC).AddDays(-30)
	endDate := date.Today(time.UTC)

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
			10000,
			transaction.TypeExpense,
			startDate.AddDays(1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			20000,
			transaction.TypeExpense,
			startDate.AddDays(2),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			15000,
			transaction.TypeExpense,
			startDate.AddDays(3),
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
	assert.Equal(t, money.Minor(45_000), result.TotalExpensesMinor) // 100 + 200 + 150
	assert.Positive(t, result.AverageDailyMinor)
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
	startDate := date.Today(time.UTC).AddDays(-30)
	endDate := date.Today(time.UTC)

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
	assert.Zero(t, result.TotalExpensesMinor)
	assert.Zero(t, result.AverageDailyMinor)
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
	startDate := date.Today(time.UTC).AddDays(-30)
	endDate := date.Today(time.UTC)

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
			500000,
			transaction.TypeIncome,
			startDate.AddDays(1),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			100000,
			transaction.TypeIncome,
			startDate.AddDays(15),
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
	assert.Equal(t, money.Minor(600_000), result.TotalIncomeMinor) // 5000 + 1000
	assert.Positive(t, result.AverageDailyMinor)

	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// Tests for GenerateBudgetComparisonReport
func TestReportService_GenerateBudgetComparisonReport(t *testing.T) {
	service, _, _, mockTransactionService, mockBudgetService, mockCategoryService := setupReportService()
	ctx := context.Background()

	categoryID := uuid.New()
	period := report.PeriodMonthly

	// Create test budget
	budgets := []*budget.Budget{
		createTestBudget(uuid.New(), 100000, categoryID),
	}

	// Create test expense transactions
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(), categoryID, 30000, transaction.TypeExpense, date.Today(time.UTC).AddDays(-10)),
		createTestTransactionWithCategory(
			uuid.New(), categoryID, 25000, transaction.TypeExpense, date.Today(time.UTC).AddDays(-5)),
	}

	// Setup mock expectations
	mockBudgetService.On("GetActiveBudgets", ctx, mock.AnythingOfType("date.Date")).Return(budgets, nil)
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)
	mockCategoryService.On("GetCategoryByID", ctx, categoryID).
		Return(&category.Category{ID: categoryID, Name: "Groceries", Type: category.TypeExpense}, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, period, result.Period)
	assert.Equal(t, money.Minor(100_000), result.TotalBudgetMinor)
	assert.Equal(t, money.Minor(55_000), result.TotalSpentMinor)    // 300 + 250
	assert.Equal(t, money.Minor(45_000), result.TotalVarianceMinor) // 1000 - 550
	assert.InDelta(t, 55.0, result.Utilization, 0.01)               // (550/1000) * 100

	require.Len(t, result.Categories, 1)
	comparison := result.Categories[0]
	assert.Equal(t, categoryID, comparison.CategoryID)
	assert.Equal(t, "Groceries", comparison.CategoryName)
	assert.Equal(t, money.Minor(100_000), comparison.BudgetAmountMinor)
	assert.Equal(t, money.Minor(55_000), comparison.ActualAmountMinor)
	assert.Equal(t, money.Minor(45_000), comparison.VarianceMinor)
	assert.InDelta(t, 55.0, comparison.Utilization, 0.01)
	assert.Equal(t, "under_budget", comparison.Status)

	mockBudgetService.AssertExpectations(t)
	mockTransactionService.AssertExpectations(t)
	mockCategoryService.AssertExpectations(t)
}

func TestReportService_GenerateBudgetComparisonReport_NoBudgets(t *testing.T) {
	service, _, _, _, mockBudgetService, _ := setupReportService()
	ctx := context.Background()

	period := report.PeriodMonthly

	// Setup mock expectations - no budgets
	mockBudgetService.On("GetActiveBudgets", ctx, mock.AnythingOfType("date.Date")).
		Return([]*budget.Budget{}, nil)

	// Execute
	result, err := service.GenerateBudgetComparisonReport(ctx, period)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Zero(t, result.TotalBudgetMinor)
	assert.Zero(t, result.TotalSpentMinor)
	assert.Zero(t, result.TotalVarianceMinor)
	assert.Zero(t, result.Utilization)

	mockBudgetService.AssertExpectations(t)
}

// Tests for GenerateCashFlowReport
func TestReportService_GenerateCashFlowReport(t *testing.T) {
	service, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	from := date.Today(time.UTC).AddDays(-30)
	to := date.Today(time.UTC)

	// Create mixed transactions
	transactions := []*transaction.Transaction{
		createTestTransaction(uuid.New(), 500000, transaction.TypeIncome, from.AddDays(1)),
		createTestTransaction(uuid.New(), 30000, transaction.TypeExpense, from.AddDays(2)),
		createTestTransaction(uuid.New(), 20000, transaction.TypeExpense, from.AddDays(3)),
		createTestTransaction(uuid.New(), 100000, transaction.TypeIncome, from.AddDays(15)),
	}

	// Setup mock expectations
	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)

	// Execute
	result, err := service.GenerateCashFlowReport(ctx, from, to)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, money.Minor(600_000), result.TotalInflowsMinor) // 5000 + 1000
	assert.Equal(t, money.Minor(50_000), result.TotalOutflowsMinor) // 300 + 200
	assert.Equal(t, money.Minor(550_000), result.NetCashFlowMinor)  // 6000 - 500

	mockTransactionService.AssertExpectations(t)
}

// Tests for SaveReport
func TestReportService_SaveReport(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportEntity := report.NewReport(
		"Test Report",
		report.TypeExpenses,
		report.PeriodMonthly,
		uuid.New(),
		date.Today(time.UTC).AddDays(-30),
		date.Today(time.UTC),
	)
	reportEntity.Data.TotalExpensesMinor = 100_000

	mockReportRepo.On("Create", ctx, reportEntity).Return(nil)

	err := service.SaveReport(ctx, reportEntity)

	require.NoError(t, err)
	mockReportRepo.AssertExpectations(t)
}

func TestReportService_SaveReport_RepositoryError(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportEntity := report.NewReport(
		"Test Report",
		report.TypeExpenses,
		report.PeriodMonthly,
		uuid.New(),
		date.Today(time.UTC).AddDays(-30),
		date.Today(time.UTC),
	)

	mockReportRepo.On("Create", ctx, reportEntity).Return(errors.New("database error"))

	err := service.SaveReport(ctx, reportEntity)

	require.ErrorContains(t, err, "database error")
	mockReportRepo.AssertExpectations(t)
}

// reportServiceMocks собирает моки, нужные генераторам отчётов всех типов.
type reportServiceMocks struct {
	userRepo    *MockUserRepository
	transaction *MockTransactionService
	budget      *MockBudgetService
	category    *MockCategoryService
}

// Tests for GenerateReport
func TestReportService_GenerateReport(t *testing.T) {
	categoryID := uuid.New()
	expenses := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			30000,
			transaction.TypeExpense,
			date.Today(time.UTC).AddDays(-3),
		),
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			20000,
			transaction.TypeExpense,
			date.Today(time.UTC).AddDays(-2),
		),
	}
	incomes := []*transaction.Transaction{
		createTestTransactionWithCategory(
			uuid.New(),
			categoryID,
			250000,
			transaction.TypeIncome,
			date.Today(time.UTC).AddDays(-5),
		),
	}
	mixed := []*transaction.Transaction{
		createTestTransaction(uuid.New(), 300000, transaction.TypeIncome, date.Today(time.UTC).AddDays(-6)),
		createTestTransaction(uuid.New(), 120000, transaction.TypeExpense, date.Today(time.UTC).AddDays(-4)),
	}

	tests := []struct {
		name       string
		reportType report.Type
		setup      func(ctx context.Context, m reportServiceMocks)
		assertData func(t *testing.T, rep *report.Report)
	}{
		{
			name:       "expenses",
			reportType: report.TypeExpenses,
			setup: func(ctx context.Context, m reportServiceMocks) {
				expectTransactions(ctx, m.transaction, expenses)
				expectCategory(ctx, m.category, categoryID, "Groceries", category.TypeExpense)
				expectUsers(ctx, m.userRepo, expenses)
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.Equal(t, money.Minor(50000), rep.Data.TotalExpensesMinor)
				require.Len(t, rep.Data.CategoryBreakdown, 1)
				assert.Equal(t, "Groceries", rep.Data.CategoryBreakdown[0].CategoryName)
				assert.Len(t, rep.Data.TopExpenses, 2)
			},
		},
		{
			name:       "income",
			reportType: report.TypeIncome,
			setup: func(ctx context.Context, m reportServiceMocks) {
				expectTransactions(ctx, m.transaction, incomes)
				expectCategory(ctx, m.category, categoryID, "Salary", category.TypeIncome)
				expectUsers(ctx, m.userRepo, incomes)
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.Equal(t, money.Minor(250000), rep.Data.TotalIncomeMinor)
				require.Len(t, rep.Data.CategoryBreakdown, 1)
				assert.Equal(t, "Salary", rep.Data.CategoryBreakdown[0].CategoryName)
			},
		},
		{
			name:       "budget",
			reportType: report.TypeBudget,
			setup: func(ctx context.Context, m reportServiceMocks) {
				budgets := []*budget.Budget{createTestBudget(uuid.New(), 100000, categoryID)}
				m.budget.On("GetActiveBudgets", ctx, mock.AnythingOfType("date.Date")).Return(budgets, nil)
				expectTransactions(ctx, m.transaction, expenses)
				expectCategory(ctx, m.category, categoryID, "Groceries", category.TypeExpense)
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.Equal(t, money.Minor(50000), rep.Data.TotalExpensesMinor)
				require.Len(t, rep.Data.BudgetComparison, 1)
				comparison := rep.Data.BudgetComparison[0]
				assert.Equal(t, "Groceries", comparison.BudgetName)
				assert.Equal(t, money.Minor(100_000), comparison.PlannedMinor)
				assert.Equal(t, money.Minor(50_000), comparison.ActualMinor)
				assert.Equal(t, money.Minor(50_000), comparison.DifferenceMinor)
				assert.InDelta(t, 50.0, comparison.Percentage, 0.01)
			},
		},
		{
			name:       "cash_flow",
			reportType: report.TypeCashFlow,
			setup: func(ctx context.Context, m reportServiceMocks) {
				expectTransactions(ctx, m.transaction, mixed)
			},
			assertData: func(t *testing.T, rep *report.Report) {
				assert.Equal(t, money.Minor(300000), rep.Data.TotalIncomeMinor)
				assert.Equal(t, money.Minor(120000), rep.Data.TotalExpensesMinor)
				assert.Equal(t, money.Minor(180_000), rep.Data.NetIncomeMinor)

				// Дни идут по возрастанию, Balance — нарастающий итог.
				require.Len(t, rep.Data.DailyBreakdown, 2)
				first, second := rep.Data.DailyBreakdown[0], rep.Data.DailyBreakdown[1]
				assert.True(t, first.Date.Before(second.Date))
				assert.Equal(t, money.Minor(300_000), first.IncomeMinor)
				assert.Equal(t, money.Minor(300_000), first.BalanceMinor)
				assert.Equal(t, money.Minor(120_000), second.ExpensesMinor)
				assert.Equal(t, money.Minor(180_000), second.BalanceMinor)
			},
		},
		{
			name:       "category_breakdown",
			reportType: report.TypeCategoryBreak,
			setup: func(ctx context.Context, m reportServiceMocks) {
				expectTransactions(ctx, m.transaction, expenses)
				expectCategory(ctx, m.category, categoryID, "Groceries", category.TypeExpense)
				m.category.On("GetCategoryHierarchy", ctx).Return([]*category.Category{}, nil)
			},
			assertData: func(t *testing.T, rep *report.Report) {
				require.Len(t, rep.Data.CategoryBreakdown, 1)
				item := rep.Data.CategoryBreakdown[0]
				assert.Equal(t, "Groceries", item.CategoryName)
				assert.Equal(t, money.Minor(50_000), item.AmountMinor)
				assert.Equal(t, 2, item.Count)
				assert.InDelta(t, 100.0, item.Percentage, 0.01)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockReportRepo, mockUserRepo, mockTxService, mockBudgetService, mockCategoryService := setupReportService()
			ctx := context.Background()

			tt.setup(ctx, reportServiceMocks{
				userRepo:    mockUserRepo,
				transaction: mockTxService,
				budget:      mockBudgetService,
				category:    mockCategoryService,
			})

			req := dto.ReportRequestDTO{
				Name:      "Generated " + tt.name,
				Type:      tt.reportType,
				Period:    report.PeriodMonthly,
				UserID:    uuid.New(),
				StartDate: date.Today(time.UTC).AddDays(-30),
				EndDate:   date.Today(time.UTC),
			}

			result, err := service.GenerateReport(ctx, req)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, req.Name, result.Name)
			assert.Equal(t, tt.reportType, result.Type)
			assert.Equal(t, req.UserID, result.UserID)
			tt.assertData(t, result)
			assert.Equal(t, req.StartDate, result.StartDate)
			assert.Equal(t, req.EndDate, result.EndDate)

			// Генерация не сохраняет отчёт — это делает SaveReport.
			mockReportRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestReportService_GenerateReport_UnsupportedType(t *testing.T) {
	service, _, _, _, _, _ := setupReportService()
	ctx := context.Background()

	req := dto.ReportRequestDTO{
		Name:      "Unknown",
		Type:      report.Type("unknown"),
		Period:    report.PeriodMonthly,
		UserID:    uuid.New(),
		StartDate: date.Today(time.UTC).AddDays(-30),
		EndDate:   date.Today(time.UTC),
	}

	result, err := service.GenerateReport(ctx, req)

	require.ErrorIs(t, err, services.ErrUnsupportedReportType)
	assert.Nil(t, result)
}

func TestReportService_GenerateReport_GenerationError(t *testing.T) {
	service, _, _, mockTransactionService, _, _ := setupReportService()
	ctx := context.Background()

	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(nil, errors.New("database error"))

	req := dto.ReportRequestDTO{
		Name:      "Broken",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    uuid.New(),
		StartDate: date.Today(time.UTC).AddDays(-30),
		EndDate:   date.Today(time.UTC),
	}

	result, err := service.GenerateReport(ctx, req)

	require.ErrorContains(t, err, "database error")
	assert.Nil(t, result)
	mockTransactionService.AssertExpectations(t)
}

func expectTransactions(ctx context.Context, m *MockTransactionService, transactions []*transaction.Transaction) {
	m.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).Return(transactions, nil)
}

func expectCategory(
	ctx context.Context,
	m *MockCategoryService,
	id uuid.UUID,
	name string,
	categoryType category.Type,
) {
	m.On("GetCategoryByID", ctx, id).Return(createTestCategory(id, name, categoryType), nil)
}

func expectUsers(ctx context.Context, m *MockUserRepository, transactions []*transaction.Transaction) {
	for _, tx := range transactions {
		m.On("GetByID", ctx, tx.UserID).Return(&user.User{
			ID:        tx.UserID,
			FirstName: "Test",
			LastName:  "User",
			Email:     "test@example.com",
		}, nil)
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
	mockReportRepo.On("GetByID", ctx, reportID).Return(&report.Report{ID: reportID}, nil)
	mockReportRepo.On("Delete", ctx, reportID).Return(nil)

	// Execute
	err := service.DeleteReport(ctx, reportID)

	// Assert
	require.NoError(t, err)

	mockReportRepo.AssertExpectations(t)
}

func TestReportService_DeleteReport_NotFound(t *testing.T) {
	service, mockReportRepo, _, _, _, _ := setupReportService()
	ctx := context.Background()

	reportID := uuid.New()
	mockReportRepo.On("GetByID", ctx, reportID).Return(nil, errors.New("not found"))

	err := service.DeleteReport(ctx, reportID)

	require.ErrorIs(t, err, services.ErrReportNotFound)
	mockReportRepo.AssertExpectations(t)
}

// Test for BUG-002: Validate that TransactionFilterDTO has valid Limit when generating reports
func TestReportService_GenerateExpenseReport_ValidatesFilterLimit(t *testing.T) {
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	userID := uuid.New()
	startDate := date.Today(time.UTC).AddDays(-7)
	endDate := date.Today(time.UTC)

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
			5000,
			transaction.TypeExpense,
			startDate.AddDays(1),
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

// Копейки не должны схлопываться в ноль: доля 1 из 3 — это 33.33%, а не 0,
// а среднее за период округляется half-up (3 копейки за 2 дня → 2).
func TestReportService_GenerateExpenseReport_SmallAmounts(t *testing.T) {
	service, _, mockUserRepo, mockTransactionService, _, mockCategoryService := setupReportService()
	ctx := context.Background()

	startDate := date.Today(time.UTC).AddDays(-1)
	endDate := date.Today(time.UTC)

	cheapID, pricyID := uuid.New(), uuid.New()
	transactions := []*transaction.Transaction{
		createTestTransactionWithCategory(uuid.New(), cheapID, 1, transaction.TypeExpense, startDate),
		createTestTransactionWithCategory(uuid.New(), pricyID, 2, transaction.TypeExpense, endDate),
	}

	mockTransactionService.On("GetAllTransactions", ctx, mock.AnythingOfType("dto.TransactionFilterDTO")).
		Return(transactions, nil)
	mockCategoryService.On("GetCategoryByID", ctx, cheapID).
		Return(createTestCategory(cheapID, "Мелочь", category.TypeExpense), nil)
	mockCategoryService.On("GetCategoryByID", ctx, pricyID).
		Return(createTestCategory(pricyID, "Крупное", category.TypeExpense), nil)
	mockUserRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).
		Return(&user.User{FirstName: "Test", LastName: "User"}, nil)

	result, err := service.GenerateExpenseReport(ctx, dto.ReportRequestDTO{
		Name:      "Копейки",
		Type:      report.TypeExpenses,
		Period:    report.PeriodCustom,
		UserID:    uuid.New(),
		StartDate: startDate,
		EndDate:   endDate,
	})
	require.NoError(t, err)

	assert.Equal(t, money.Minor(3), result.TotalExpensesMinor)
	assert.Equal(t, money.Minor(2), result.AverageDailyMinor) // 3/2 = 1.5 → 2

	require.Len(t, result.CategoryBreakdown, 2)
	assert.InDelta(t, 66.67, result.CategoryBreakdown[0].Percentage, 0.01)
	assert.InDelta(t, 33.33, result.CategoryBreakdown[1].Percentage, 0.01)
}

// TestReportService_PeriodBoundsPerPeriod — границы каждого периода считаются в зоне семьи;
// неделя начинается с воскресенья, год — с 1 января.
func TestReportService_PeriodBoundsPerPeriod(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	today := date.Today(loc)
	firstOfMonth, lastOfMonth := today.MonthBounds()
	weekStart := today.AddDays(-int(today.In(loc).Weekday()))

	tests := []struct {
		name   string
		period report.Period
		from   date.Date
		to     date.Date
	}{
		{"daily", report.PeriodDaily, today, today},
		{"weekly", report.PeriodWeekly, weekStart, weekStart.AddDays(6)},
		{"monthly", report.PeriodMonthly, firstOfMonth, lastOfMonth},
		{
			"yearly",
			report.PeriodYearly,
			date.New(today.Year, time.January, 1),
			date.New(today.Year, time.December, 31),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _, _, _, mockBudgetService, _ := setupReportService()
			ctx := context.Background()
			mockBudgetService.On("GetActiveBudgets", ctx, tt.from).Return([]*budget.Budget{}, nil)

			result, reportErr := service.GenerateBudgetComparisonReport(ctx, tt.period)

			require.NoError(t, reportErr)
			assert.Equal(t, tt.from, result.StartDate)
			assert.Equal(t, tt.to, result.EndDate)
			mockBudgetService.AssertExpectations(t)
		})
	}
}
