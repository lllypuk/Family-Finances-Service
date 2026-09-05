package services_test

import (
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
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

const statsRecentLimit = 10

type statsMocks struct {
	transactions *MockTransactionService
	budgets      *MockBudgetService
	categories   *MockCategoryService
}

func newStatsService() (services.StatsService, *statsMocks) {
	m := &statsMocks{
		transactions: new(MockTransactionService),
		budgets:      new(MockBudgetService),
		categories:   new(MockCategoryService),
	}
	return services.NewStatsService(m.transactions, m.budgets, m.categories), m
}

// periodFilter матчит выборку транзакций за конкретный период.
func periodFilter(from, to time.Time) any {
	return mock.MatchedBy(func(f dto.TransactionFilterDTO) bool {
		return f.DateFrom != nil && f.DateTo != nil && f.DateFrom.Equal(from) && f.DateTo.Equal(to)
	})
}

// recentFilter матчит выборку последних транзакций (без дат).
func recentFilter() any {
	return mock.MatchedBy(func(f dto.TransactionFilterDTO) bool {
		return f.DateFrom == nil && f.Limit == statsRecentLimit
	})
}

func previousPeriod(from, to time.Time) (time.Time, time.Time) {
	previousTo := from.Add(-time.Second)
	return previousTo.Add(-to.Sub(from)), previousTo
}

func statsTransaction(amount float64, txType transaction.Type, categoryID uuid.UUID) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		CategoryID:  categoryID,
		Amount:      amount,
		Type:        txType,
		Description: "test",
		Date:        time.Now(),
		CreatedAt:   time.Now(),
	}
}

func statsCategory(id uuid.UUID, name string) *category.Category {
	return &category.Category{
		ID:    id,
		Name:  name,
		Color: "#fff",
		Icon:  "icon",
	}
}

func TestStatsService_Summary_Success(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.March, 31, 23, 59, 59, 0, time.UTC)
	prevFrom, prevTo := previousPeriod(from, to)

	foodID, salaryID := uuid.New(), uuid.New()
	current := []*transaction.Transaction{
		statsTransaction(1000, transaction.TypeIncome, salaryID),
		statsTransaction(400, transaction.TypeExpense, foodID),
		statsTransaction(100, transaction.TypeExpense, foodID),
	}
	previous := []*transaction.Transaction{
		statsTransaction(800, transaction.TypeIncome, salaryID),
		statsTransaction(250, transaction.TypeExpense, foodID),
	}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).Return(current, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).Return(previous, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).Return(current, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(42, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, foodID).Return(statsCategory(foodID, "Еда"), nil)
	m.categories.On("GetCategoryByID", mock.Anything, salaryID).Return(statsCategory(salaryID, "Зарплата"), nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	assert.InDelta(t, 1000.0, summary.Current.Income, 0.001)
	assert.InDelta(t, 500.0, summary.Current.Expenses, 0.001)
	assert.InDelta(t, 500.0, summary.Current.Net, 0.001)
	assert.Equal(t, 3, summary.Current.TransactionCount)
	assert.Equal(t, 42, summary.TransactionsTotal)

	assert.True(t, summary.HasPreviousData)
	assert.InDelta(t, 800.0, summary.Previous.Income, 0.001)
	assert.InDelta(t, 0.25, summary.IncomeDelta, 0.001)  // (1000-800)/800
	assert.InDelta(t, 1.0, summary.ExpensesDelta, 0.001) // (500-250)/250

	require.Len(t, summary.ExpenseCategories, 1)
	assert.Equal(t, "Еда", summary.ExpenseCategories[0].Name)
	assert.InDelta(t, 500.0, summary.ExpenseCategories[0].Amount, 0.001)
	assert.InDelta(t, 1.0, summary.ExpenseCategories[0].Share, 0.001)
	assert.Equal(t, 2, summary.ExpenseCategories[0].TransactionCount)

	require.Len(t, summary.IncomeCategories, 1)
	assert.Equal(t, "Зарплата", summary.IncomeCategories[0].Name)

	assert.Len(t, summary.Recent, 3)
	assert.Equal(t, "Еда", summary.Recent[1].CategoryName)
}

func TestStatsService_Summary_CategoriesSortedByAmount(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	smallID, bigID := uuid.New(), uuid.New()
	current := []*transaction.Transaction{
		statsTransaction(100, transaction.TypeExpense, smallID),
		statsTransaction(300, transaction.TypeExpense, bigID),
	}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).Return(current, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(2, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, smallID).Return(statsCategory(smallID, "Мелочь"), nil)
	m.categories.On("GetCategoryByID", mock.Anything, bigID).Return(statsCategory(bigID, "Крупное"), nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	require.Len(t, summary.ExpenseCategories, 2)
	assert.Equal(t, "Крупное", summary.ExpenseCategories[0].Name)
	assert.InDelta(t, 0.75, summary.ExpenseCategories[0].Share, 0.001)
	assert.Equal(t, "Мелочь", summary.ExpenseCategories[1].Name)
}

func TestStatsService_Summary_UnknownCategorySkipped(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	unknownID := uuid.New()
	current := []*transaction.Transaction{statsTransaction(100, transaction.TypeExpense, unknownID)}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).Return(current, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).Return(current, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(1, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, unknownID).Return(nil, errors.New("not found"))

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	assert.Empty(t, summary.ExpenseCategories)
	assert.InDelta(t, 100.0, summary.Current.Expenses, 0.001)
	require.Len(t, summary.Recent, 1)
	assert.Empty(t, summary.Recent[0].CategoryName)
}

func TestStatsService_Summary_BudgetProgress(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	now := time.Now()
	budgets := []*budget.Budget{
		{
			ID: uuid.New(), Name: "Норма", Amount: 1000, Spent: 100, IsActive: true,
			Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDate(0, 0, 10),
		},
		{
			ID: uuid.New(), Name: "На пределе", Amount: 1000, Spent: 850, IsActive: true,
			CategoryID: &categoryID, Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDate(0, 0, 5),
		},
		{
			ID: uuid.New(), Name: "Превышен", Amount: 1000, Spent: 1200, IsActive: false,
			Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDate(0, 0, -1),
		},
	}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(0, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return(budgets, nil)
	m.categories.On("GetCategoryByID", mock.Anything, categoryID).Return(statsCategory(categoryID, "Еда"), nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	require.Len(t, summary.Budgets, 3)
	// Отсортированы по убыванию использования
	assert.Equal(t, "Превышен", summary.Budgets[0].Name)
	assert.True(t, summary.Budgets[0].IsOverBudget)
	assert.False(t, summary.Budgets[0].IsNearLimit)
	assert.False(t, summary.Budgets[0].IsActive)
	assert.Equal(t, 0, summary.Budgets[0].DaysRemaining)

	assert.Equal(t, "На пределе", summary.Budgets[1].Name)
	assert.True(t, summary.Budgets[1].IsNearLimit)
	assert.Equal(t, "Еда", summary.Budgets[1].CategoryName)
	assert.InDelta(t, 0.85, summary.Budgets[1].Utilization, 0.001)
	assert.InDelta(t, 150.0, summary.Budgets[1].Remaining, 0.001)

	assert.Equal(t, "Норма", summary.Budgets[2].Name)
	assert.False(t, summary.Budgets[2].IsNearLimit)
	assert.Empty(t, summary.Budgets[2].CategoryName)
}

func TestStatsService_Summary_EmptyPeriod(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(0, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	assert.Zero(t, summary.Current.TransactionCount)
	assert.InDelta(t, 0.0, summary.Current.Net, 0.001)
	assert.False(t, summary.HasPreviousData)
	assert.InDelta(t, 0.0, summary.IncomeDelta, 0.001)
	assert.InDelta(t, 0.0, summary.ExpensesDelta, 0.001)
	assert.Empty(t, summary.ExpenseCategories)
	assert.Empty(t, summary.Budgets)
	assert.Empty(t, summary.Recent)
}

// Деление на ноль: предыдущий период есть, но одна из сумм в нём нулевая.
func TestStatsService_Summary_ZeroPreviousAmount(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	current := []*transaction.Transaction{statsTransaction(500, transaction.TypeIncome, categoryID)}
	previous := []*transaction.Transaction{statsTransaction(200, transaction.TypeExpense, categoryID)}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).Return(current, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).Return(previous, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(2, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, categoryID).Return(statsCategory(categoryID, "Еда"), nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	assert.True(t, summary.HasPreviousData)
	assert.InDelta(t, 0.0, summary.IncomeDelta, 0.001)    // предыдущий доход = 0
	assert.InDelta(t, -1.0, summary.ExpensesDelta, 0.001) // (0-200)/200
}

func TestStatsService_Summary_InvalidPeriod(t *testing.T) {
	svc, _ := newStatsService()
	from := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	summary, err := svc.Summary(t.Context(), from, from.AddDate(0, 0, -1))

	require.ErrorIs(t, err, services.ErrInvalidStatsPeriod)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_TransactionServiceError(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).
		Return(nil, errors.New("database error"))

	summary, err := svc.Summary(t.Context(), from, to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_BudgetServiceError(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).
		Return([]*transaction.Transaction{}, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return(nil, errors.New("budget service down"))

	summary, err := svc.Summary(t.Context(), from, to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

// Ошибка выборки предыдущего периода не роняет сводку — данных для сравнения просто нет.
func TestStatsService_Summary_PreviousPeriodErrorIgnored(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	current := []*transaction.Transaction{statsTransaction(500, transaction.TypeIncome, categoryID)}

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).Return(current, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(prevFrom, prevTo)).
		Return(nil, errors.New("database error"))
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(1, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, categoryID).Return(statsCategory(categoryID, "Еда"), nil)

	summary, err := svc.Summary(t.Context(), from, to)
	require.NoError(t, err)

	assert.False(t, summary.HasPreviousData)
	assert.InDelta(t, 500.0, summary.Current.Income, 0.001)
}

func TestStatsService_Summary_CountError(t *testing.T) {
	svc, m := newStatsService()
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	m.transactions.On("GetAllTransactions", mock.Anything, periodFilter(from, to)).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).
		Return(0, errors.New("database error"))
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), from, to)

	require.Error(t, err)
	assert.Nil(t, summary)
}
