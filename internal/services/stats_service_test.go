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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

const statsRecentLimit = 10

type statsMocks struct {
	transactions *MockTransactionService
	budgets      *MockBudgetService
	categories   *MockCategoryService
	families     *MockFamilyService
	aggregates   *MockTransactionRepository
	holdings     *mockHoldingRepo
}

// totals объявляет агрегат за период; без него сводка за этот период не соберётся.
func (m *statsMocks) totals(from, to date.Date, rows ...transaction.CategoryTotal) {
	m.aggregates.On("GetTotalsByCategoryAndDateRange", mock.Anything, from, to).Return(rows, nil)
}

// monthly объявляет помесячный агрегат за период.
func (m *statsMocks) monthly(from, to date.Date, rows ...transaction.MonthTotal) {
	m.aggregates.On("GetTotalsByMonth", mock.Anything, from, to).Return(rows, nil)
}

func monthTotal(month string, txType transaction.Type, amount money.Minor, count int) transaction.MonthTotal {
	return transaction.MonthTotal{Month: month, Type: txType, AmountMinor: amount, Count: count}
}

func newStatsMocks() *statsMocks {
	return &statsMocks{
		transactions: new(MockTransactionService),
		budgets:      new(MockBudgetService),
		categories:   new(MockCategoryService),
		families:     new(MockFamilyService),
		aggregates:   new(MockTransactionRepository),
		holdings:     new(mockHoldingRepo),
	}
}

func newStatsService() (services.StatsService, *statsMocks) {
	m := newStatsMocks()
	m.families.On("GetFamily", mock.Anything).
		Return(&user.Family{Currency: "RUB", Timezone: "Europe/Moscow"}, nil).Maybe()

	return services.NewStatsService(m.transactions, m.budgets, m.categories, m.families, m.aggregates, m.holdings), m
}

// recentFilter матчит выборку последних транзакций (без дат).
func recentFilter() any {
	return mock.MatchedBy(func(f dto.TransactionFilterDTO) bool {
		return f.DateFrom == nil && f.Limit == statsRecentLimit
	})
}

func previousPeriod(from, to date.Date) (date.Date, date.Date) {
	previousTo := from.AddDays(-1)
	days := int(to.In(time.UTC).Sub(from.In(time.UTC)).Hours() / 24)

	return previousTo.AddDays(-days), previousTo
}

func catTotal(
	categoryID uuid.UUID,
	txType transaction.Type,
	amount money.Minor,
	count int,
) transaction.CategoryTotal {
	return transaction.CategoryTotal{CategoryID: categoryID, Type: txType, AmountMinor: amount, Count: count}
}

func statsTransaction(amount money.Minor, txType transaction.Type, categoryID uuid.UUID) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		CategoryID:  categoryID,
		AmountMinor: amount,
		Type:        txType,
		Description: "test",
		Date:        date.Today(time.UTC),
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
	from := date.New(2026, time.March, 1)
	to := date.New(2026, time.March, 31)
	prevFrom, prevTo := previousPeriod(from, to)

	foodID, salaryID := uuid.New(), uuid.New()
	m.totals(from, to,
		catTotal(salaryID, transaction.TypeIncome, 100_000, 1),
		catTotal(foodID, transaction.TypeExpense, 50_000, 2),
	)
	m.totals(prevFrom, prevTo,
		catTotal(salaryID, transaction.TypeIncome, 80_000, 1),
		catTotal(foodID, transaction.TypeExpense, 25_000, 1),
	)
	recent := []*transaction.Transaction{
		statsTransaction(100_000, transaction.TypeIncome, salaryID),
		statsTransaction(40_000, transaction.TypeExpense, foodID),
		statsTransaction(10_000, transaction.TypeExpense, foodID),
	}
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).Return(recent, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(42, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, foodID).Return(statsCategory(foodID, "Еда"), nil)
	m.categories.On("GetCategoryByID", mock.Anything, salaryID).Return(statsCategory(salaryID, "Зарплата"), nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	assert.Equal(t, money.Minor(100_000), summary.Current.IncomeMinor)
	assert.Equal(t, money.Minor(50_000), summary.Current.ExpensesMinor)
	assert.Equal(t, money.Minor(50_000), summary.Current.NetMinor)
	assert.Equal(t, 3, summary.Current.TransactionCount)
	assert.Equal(t, 42, summary.TransactionsTotal)

	assert.True(t, summary.HasPreviousData)
	assert.Equal(t, money.Minor(80_000), summary.Previous.IncomeMinor)
	assert.InDelta(t, 0.25, summary.IncomeDelta, 0.001)  // (1000-800)/800
	assert.InDelta(t, 1.0, summary.ExpensesDelta, 0.001) // (500-250)/250

	require.Len(t, summary.ExpenseCategories, 1)
	assert.Equal(t, "Еда", summary.ExpenseCategories[0].Name)
	assert.Equal(t, money.Minor(50_000), summary.ExpenseCategories[0].AmountMinor)
	assert.InDelta(t, 1.0, summary.ExpenseCategories[0].Share, 0.001)
	assert.Equal(t, 2, summary.ExpenseCategories[0].TransactionCount)

	require.Len(t, summary.IncomeCategories, 1)
	assert.Equal(t, "Зарплата", summary.IncomeCategories[0].Name)

	assert.Len(t, summary.Recent, 3)
	assert.Equal(t, "Еда", summary.Recent[1].CategoryName)
}

func TestStatsService_Summary_CategoriesSortedByAmount(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	smallID, bigID := uuid.New(), uuid.New()
	m.totals(from, to,
		catTotal(smallID, transaction.TypeExpense, 10_000, 1),
		catTotal(bigID, transaction.TypeExpense, 30_000, 1),
	)
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(2, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, smallID).Return(statsCategory(smallID, "Мелочь"), nil)
	m.categories.On("GetCategoryByID", mock.Anything, bigID).Return(statsCategory(bigID, "Крупное"), nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	require.Len(t, summary.ExpenseCategories, 2)
	assert.Equal(t, "Крупное", summary.ExpenseCategories[0].Name)
	assert.InDelta(t, 0.75, summary.ExpenseCategories[0].Share, 0.001)
	assert.Equal(t, "Мелочь", summary.ExpenseCategories[1].Name)
}

func TestStatsService_Summary_UnknownCategorySkipped(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	unknownID := uuid.New()
	m.totals(from, to, catTotal(unknownID, transaction.TypeExpense, 10_000, 1))
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{statsTransaction(10_000, transaction.TypeExpense, unknownID)}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(1, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, unknownID).Return(nil, errors.New("not found"))

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	assert.Empty(t, summary.ExpenseCategories)
	assert.Equal(t, money.Minor(10_000), summary.Current.ExpensesMinor)
	require.Len(t, summary.Recent, 1)
	assert.Empty(t, summary.Recent[0].CategoryName)
}

func TestStatsService_Summary_BudgetProgress(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	now := date.Today(time.UTC)
	budgets := []*budget.Budget{
		{
			ID: uuid.New(), Name: "Норма", AmountMinor: 100_000, SpentMinor: 10_000, IsActive: true,
			Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDays(10),
		},
		{
			ID: uuid.New(), Name: "На пределе", AmountMinor: 100_000, SpentMinor: 85_000, IsActive: true,
			CategoryID: &categoryID, Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDays(5),
		},
		{
			ID: uuid.New(), Name: "Превышен", AmountMinor: 100_000, SpentMinor: 120_000, IsActive: false,
			Period: budget.PeriodMonthly, StartDate: now, EndDate: now.AddDays(-1),
		},
	}

	m.totals(from, to)
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(0, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return(budgets, nil)
	m.categories.On("GetCategoryByID", mock.Anything, categoryID).Return(statsCategory(categoryID, "Еда"), nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
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
	assert.Equal(t, money.Minor(15_000), summary.Budgets[1].RemainingMinor)

	assert.Equal(t, "Норма", summary.Budgets[2].Name)
	assert.False(t, summary.Budgets[2].IsNearLimit)
	assert.Empty(t, summary.Budgets[2].CategoryName)
}

func TestStatsService_Summary_EmptyPeriod(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	m.totals(from, to)
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(0, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	assert.Zero(t, summary.Current.TransactionCount)
	assert.Zero(t, summary.Current.NetMinor)
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
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	m.totals(from, to, catTotal(categoryID, transaction.TypeIncome, 50_000, 1))
	m.totals(prevFrom, prevTo, catTotal(categoryID, transaction.TypeExpense, 20_000, 1))
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(2, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, categoryID).Return(statsCategory(categoryID, "Еда"), nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	assert.True(t, summary.HasPreviousData)
	assert.InDelta(t, 0.0, summary.IncomeDelta, 0.001)    // предыдущий доход = 0
	assert.InDelta(t, -1.0, summary.ExpensesDelta, 0.001) // (0-200)/200
}

func TestStatsService_Summary_InvalidPeriod(t *testing.T) {
	svc, _ := newStatsService()
	from := date.New(2026, time.March, 31)
	to := from.AddDays(-1)

	summary, err := svc.Summary(t.Context(), &from, &to)

	require.ErrorIs(t, err, services.ErrInvalidStatsPeriod)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_AggregateError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)

	m.aggregates.On("GetTotalsByCategoryAndDateRange", mock.Anything, from, to).
		Return(nil, errors.New("database error"))

	summary, err := svc.Summary(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_BudgetServiceError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)

	m.totals(from, to)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return(nil, errors.New("budget service down"))

	summary, err := svc.Summary(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

// Сбой выборки предыдущего периода — это сбой БД, а не «данных для сравнения нет».
func TestStatsService_Summary_PreviousPeriodError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	categoryID := uuid.New()
	m.totals(from, to, catTotal(categoryID, transaction.TypeIncome, 50_000, 1))
	m.aggregates.On("GetTotalsByCategoryAndDateRange", mock.Anything, prevFrom, prevTo).
		Return(nil, errors.New("database error"))
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(1, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_CountError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)

	m.totals(from, to)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).
		Return(0, errors.New("database error"))
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, summary)
}

func TestStatsService_Summary_CategoryCountsSplitByType(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.March, 1)
	to := from.AddDays(30)
	prevFrom, prevTo := previousPeriod(from, to)

	mixedID := uuid.New()
	m.totals(from, to,
		catTotal(mixedID, transaction.TypeExpense, 30_000, 2),
		catTotal(mixedID, transaction.TypeIncome, 5_000, 1),
	)
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(3, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, mock.Anything).Return([]*budget.Budget{}, nil)
	m.categories.On("GetCategoryByID", mock.Anything, mixedID).Return(statsCategory(mixedID, "Разное"), nil)

	summary, err := svc.Summary(t.Context(), &from, &to)
	require.NoError(t, err)

	require.Len(t, summary.ExpenseCategories, 1)
	require.Len(t, summary.IncomeCategories, 1)
	assert.Equal(t, 2, summary.ExpenseCategories[0].TransactionCount)
	assert.Equal(t, 1, summary.IncomeCategories[0].TransactionCount)
}

// TestStatsService_Summary_DefaultPeriodUsesFamilyTimezone — nil-границы означают текущий
// месяц в зоне семьи, а не в зоне процесса (A-06).
func TestStatsService_Summary_DefaultPeriodUsesFamilyTimezone(t *testing.T) {
	svc, m := newStatsService()

	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	today := date.Today(loc)
	from, _ := today.MonthBounds()
	prevFrom, prevTo := previousPeriod(from, today)

	m.totals(from, today)
	m.totals(prevFrom, prevTo)
	m.transactions.On("GetAllTransactions", mock.Anything, recentFilter()).
		Return([]*transaction.Transaction{}, nil)
	m.transactions.On("CountTransactions", mock.Anything, mock.Anything).Return(0, nil)
	m.budgets.On("GetActiveBudgets", mock.Anything, today).Return([]*budget.Budget{}, nil)

	summary, err := svc.Summary(t.Context(), nil, nil)
	require.NoError(t, err)

	assert.Equal(t, from, summary.From)
	assert.Equal(t, today, summary.To)
	// Пустой период отдаёт массивы, а не null: оба поля обязательны в контракте.
	assert.NotNil(t, summary.ExpenseCategories)
	assert.NotNil(t, summary.IncomeCategories)
	assert.Empty(t, summary.ExpenseCategories)
	assert.Empty(t, summary.IncomeCategories)
	m.families.AssertCalled(t, "GetFamily", mock.Anything)
	m.aggregates.AssertExpectations(t)
	m.budgets.AssertExpectations(t)
}

func TestStatsService_Summary_FamilyError(t *testing.T) {
	m := newStatsMocks()
	m.families.On("GetFamily", mock.Anything).Return(nil, errors.New("family repository down"))
	svc := services.NewStatsService(m.transactions, m.budgets, m.categories, m.families, m.aggregates, m.holdings)

	summary, err := svc.Summary(t.Context(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, summary)
}

// TestStatsService_Monthly_DefaultPeriod — без границ ряд покрывает двенадцать месяцев
// по сегодняшний в зоне семьи, месяцы без операций — нули.
func TestStatsService_Monthly_DefaultPeriod(t *testing.T) {
	svc, m := newStatsService()
	m.aggregates.On("GetTotalsByMonth", mock.Anything, mock.Anything, mock.Anything).
		Return([]transaction.MonthTotal{}, nil)

	monthly, err := svc.Monthly(t.Context(), nil, nil)
	require.NoError(t, err)

	moscow, locErr := time.LoadLocation("Europe/Moscow")
	require.NoError(t, locErr)
	today := date.Today(moscow)
	monthStart, _ := today.MonthBounds()

	require.Len(t, monthly.Months, 12)
	assert.Equal(t, monthStart.AddMonths(-11), monthly.From)
	assert.Equal(t, today, monthly.To)
	assert.Equal(t, monthStart.AddMonths(-11).MonthKey(), monthly.Months[0].Month)
	assert.Equal(t, today.MonthKey(), monthly.Months[11].Month)
	assert.Equal(t, money.Minor(0), monthly.Months[0].IncomeMinor)
	assert.Equal(t, 0, monthly.Months[0].TransactionCount)
}

// TestStatsService_Monthly_SingleMonth — интервал внутри одного месяца даёт одну корзину
// с обоими типами операций.
func TestStatsService_Monthly_SingleMonth(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.September, 5)
	to := date.New(2026, time.September, 13)

	m.monthly(from, to,
		monthTotal("2026-09", transaction.TypeExpense, 20_000, 3),
		monthTotal("2026-09", transaction.TypeIncome, 50_000, 1),
	)

	monthly, err := svc.Monthly(t.Context(), &from, &to)
	require.NoError(t, err)

	require.Len(t, monthly.Months, 1)
	assert.Equal(t, "2026-09", monthly.Months[0].Month)
	assert.Equal(t, money.Minor(50_000), monthly.Months[0].IncomeMinor)
	assert.Equal(t, money.Minor(20_000), monthly.Months[0].ExpensesMinor)
	assert.Equal(t, money.Minor(30_000), monthly.Months[0].NetMinor)
	assert.Equal(t, 4, monthly.Months[0].TransactionCount)
}

// TestStatsService_Monthly_SeveralMonths — корзина на каждый месяц периода, включая пустой в середине.
func TestStatsService_Monthly_SeveralMonths(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.July, 20)
	to := date.New(2026, time.September, 10)

	m.monthly(from, to,
		monthTotal("2026-07", transaction.TypeIncome, 70_000, 2),
		monthTotal("2026-09", transaction.TypeExpense, 15_000, 1),
	)

	monthly, err := svc.Monthly(t.Context(), &from, &to)
	require.NoError(t, err)

	require.Len(t, monthly.Months, 3)
	assert.Equal(t, []string{"2026-07", "2026-08", "2026-09"},
		[]string{monthly.Months[0].Month, monthly.Months[1].Month, monthly.Months[2].Month})
	assert.Equal(t, money.Minor(70_000), monthly.Months[0].NetMinor)
	assert.Equal(t, money.Minor(0), monthly.Months[1].NetMinor)
	assert.Equal(t, 0, monthly.Months[1].TransactionCount)
	assert.Equal(t, money.Minor(-15_000), monthly.Months[2].NetMinor)
}

func TestStatsService_Monthly_InvalidPeriod(t *testing.T) {
	svc, _ := newStatsService()
	from := date.New(2026, time.September, 30)
	to := from.AddDays(-1)

	monthly, err := svc.Monthly(t.Context(), &from, &to)

	require.ErrorIs(t, err, services.ErrInvalidStatsPeriod)
	assert.Nil(t, monthly)
}

// TestStatsService_Monthly_PeriodTooLong — потолок ряда: границы приходят от клиента,
// и период в тысячи лет иначе развернулся бы в корзину на каждый его месяц.
func TestStatsService_Monthly_PeriodTooLong(t *testing.T) {
	svc, _ := newStatsService()
	from := date.New(2026, time.January, 1)
	to := from.AddMonths(120)

	monthly, err := svc.Monthly(t.Context(), &from, &to)

	require.ErrorIs(t, err, services.ErrStatsPeriodTooLong)
	assert.Nil(t, monthly)
}

// TestStatsService_Monthly_MaxPeriod — ровно потолок ещё проходит.
func TestStatsService_Monthly_MaxPeriod(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.January, 1)
	to := from.AddMonths(119)
	m.monthly(from, to)

	monthly, err := svc.Monthly(t.Context(), &from, &to)

	require.NoError(t, err)
	assert.Len(t, monthly.Months, 120)
}

func TestStatsService_Monthly_FamilyError(t *testing.T) {
	m := newStatsMocks()
	m.families.On("GetFamily", mock.Anything).Return(nil, errors.New("family repository down"))
	svc := services.NewStatsService(m.transactions, m.budgets, m.categories, m.families, m.aggregates, m.holdings)

	monthly, err := svc.Monthly(t.Context(), nil, nil)

	require.Error(t, err)
	assert.Nil(t, monthly)
	m.aggregates.AssertNotCalled(t, "GetTotalsByMonth", mock.Anything, mock.Anything, mock.Anything)
}

func TestStatsService_Monthly_AggregateError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.September, 1)
	to := date.New(2026, time.September, 30)

	m.aggregates.On("GetTotalsByMonth", mock.Anything, from, to).Return(nil, errors.New("database error"))

	monthly, err := svc.Monthly(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, monthly)
}

// TestStatsService_NetWorth_DefaultPeriod — без границ ряд капитала берёт границы Monthly.
func TestStatsService_NetWorth_DefaultPeriod(t *testing.T) {
	svc, m := newStatsService()
	loc, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	today := date.Today(loc)
	monthStart, _ := today.MonthBounds()
	from := monthStart.AddMonths(-11)

	m.holdings.On("SeriesValues", mock.Anything, from, today).Return([]holding.SeriesRow{
		{HoldingID: uuid.New(), Side: holding.SideLiability, Date: from.AddDays(-1), ValueMinor: 700},
	}, nil)

	series, err := svc.NetWorth(t.Context(), nil, nil)

	require.NoError(t, err)
	assert.Equal(t, from, series.From)
	assert.Equal(t, today, series.To)
	require.Len(t, series.Months, 12)
	assert.Equal(t, money.Minor(-700), series.Months[11].NetMinor)
	m.holdings.AssertExpectations(t)
}

// TestStatsService_NetWorth_FutureTo — будущий конец отбивается раньше потолка периода.
func TestStatsService_NetWorth_FutureTo(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(1, time.January, 1)
	to := date.New(9999, time.December, 31)

	series, err := svc.NetWorth(t.Context(), &from, &to)

	require.ErrorIs(t, err, services.ErrStatsPeriodInFuture)
	assert.Nil(t, series)
	m.holdings.AssertNotCalled(t, "SeriesValues", mock.Anything, mock.Anything, mock.Anything)
}

func TestStatsService_NetWorth_PeriodTooLong(t *testing.T) {
	svc, _ := newStatsService()
	from := date.New(2000, time.January, 1)
	to := date.New(2020, time.January, 1)

	_, err := svc.NetWorth(t.Context(), &from, &to)

	require.ErrorIs(t, err, services.ErrStatsPeriodTooLong)
}

func TestStatsService_NetWorth_RepositoryError(t *testing.T) {
	svc, m := newStatsService()
	from := date.New(2026, time.January, 1)
	to := date.New(2026, time.January, 31)
	m.holdings.On("SeriesValues", mock.Anything, from, to).Return(nil, errors.New("database error"))

	series, err := svc.NetWorth(t.Context(), &from, &to)

	require.Error(t, err)
	assert.Nil(t, series)
}
