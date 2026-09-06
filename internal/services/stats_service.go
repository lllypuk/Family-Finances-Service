package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
)

// ErrInvalidStatsPeriod возвращается, когда конец периода раньше его начала.
var ErrInvalidStatsPeriod = errors.New("stats period end is before start")

const (
	// statsTransactionLimit — размер страницы выборки транзакций за период.
	statsTransactionLimit = 1000
	// statsMaxTransactions — потолок суммируемых за период операций: страховка от
	// бесконечного цикла, если источник данных не уважает offset.
	statsMaxTransactions = 20000
	statsRecentLimit     = 10

	budgetNearLimitShare = 0.8
	budgetOverLimitShare = 1.0

	statsHoursInDay = 24

	// percentToShare переводит проценты из money.Percent в долю 0..1, в которой
	// dto держит Share и Utilization.
	percentToShare = 100.0
)

// statsService считает агрегаты поверх остальных сервисов, без прямого доступа к репозиториям.
type statsService struct {
	transactions TransactionService
	budgets      BudgetService
	categories   CategoryService
	families     FamilyService
}

// NewStatsService создаёт сервис статистики
func NewStatsService(
	transactions TransactionService,
	budgets BudgetService,
	categories CategoryService,
	families FamilyService,
) StatsService {
	return &statsService{
		transactions: transactions,
		budgets:      budgets,
		categories:   categories,
		families:     families,
	}
}

// Summary собирает сводку за период [from, to] включительно; nil-границы — текущий месяц
// по часовому поясу семьи (A-06).
func (s *statsService) Summary(ctx context.Context, from, to *date.Date) (*dto.StatsSummary, error) {
	family, err := s.families.GetFamily(ctx)
	if err != nil {
		return nil, err
	}

	today := date.Today(family.Location())
	start, _ := today.MonthBounds()
	end := today
	if from != nil {
		start = *from
	}
	if to != nil {
		end = *to
	}
	if end.Before(start) {
		return nil, ErrInvalidStatsPeriod
	}

	current, err := s.transactionsBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	budgets, err := s.budgetProgress(ctx, today)
	if err != nil {
		return nil, err
	}

	recent, err := s.recentTransactions(ctx)
	if err != nil {
		return nil, err
	}

	// Limit обязателен для валидации фильтра, на подсчёт он не влияет.
	total, err := s.transactions.CountTransactions(ctx, dto.TransactionFilterDTO{Limit: statsTransactionLimit})
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	totals := periodTotals(start, end, current)
	previousFrom, previousTo := previousPeriod(start, end)
	previous, hasPrevious := s.previousTotals(ctx, previousFrom, previousTo)
	expenseCategories, incomeCategories := s.categoryShares(ctx, current, totals)

	incomeDelta, expensesDelta := periodDeltas(totals, previous, hasPrevious)

	return &dto.StatsSummary{
		From:              start,
		To:                end,
		Current:           totals,
		Previous:          previous,
		HasPreviousData:   hasPrevious,
		IncomeDelta:       incomeDelta,
		ExpensesDelta:     expensesDelta,
		ExpenseCategories: expenseCategories,
		IncomeCategories:  incomeCategories,
		Budgets:           budgets,
		Recent:            recent,
		TransactionsTotal: total,
	}, nil
}

func (s *statsService) transactionsBetween(
	ctx context.Context,
	from, to date.Date,
) ([]*transaction.Transaction, error) {
	// Постранично: суммы периода должны сходиться, а не обрываться на первой странице.
	var all []*transaction.Transaction
	for offset := 0; ; offset += statsTransactionLimit {
		filter := dto.TransactionFilterDTO{
			DateFrom: &from,
			DateTo:   &to,
			Limit:    statsTransactionLimit,
			Offset:   offset,
		}

		page, err := s.transactions.GetAllTransactions(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions for period: %w", err)
		}

		all = append(all, page...)
		if len(page) < statsTransactionLimit || len(all) >= statsMaxTransactions {
			return all, nil
		}
	}
}

// previousTotals возвращает суммы за предыдущий период; ошибка выборки означает «данных нет».
func (s *statsService) previousTotals(ctx context.Context, from, to date.Date) (dto.PeriodTotals, bool) {
	if from.IsZero() || to.IsZero() {
		return dto.PeriodTotals{From: from, To: to}, false
	}

	transactions, err := s.transactionsBetween(ctx, from, to)
	if err != nil || len(transactions) == 0 {
		return dto.PeriodTotals{From: from, To: to}, false
	}

	return periodTotals(from, to, transactions), true
}

func (s *statsService) budgetProgress(ctx context.Context, today date.Date) ([]dto.BudgetProgress, error) {
	activeBudgets, err := s.budgets.GetActiveBudgets(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	progress := make([]dto.BudgetProgress, 0, len(activeBudgets))
	for _, b := range activeBudgets {
		progress = append(progress, s.budgetProgressItem(ctx, b, today))
	}

	slices.SortFunc(progress, func(a, b dto.BudgetProgress) int {
		switch {
		case a.Utilization > b.Utilization:
			return -1
		case a.Utilization < b.Utilization:
			return 1
		default:
			return 0
		}
	})

	return progress, nil
}

func (s *statsService) budgetProgressItem(
	ctx context.Context,
	b *budget.Budget,
	today date.Date,
) dto.BudgetProgress {
	utilization := b.SpentMinor.Percent(b.AmountMinor) / percentToShare

	isOverBudget := utilization >= budgetOverLimitShare

	return dto.BudgetProgress{
		ID:             b.ID,
		Name:           b.Name,
		CategoryName:   s.categoryName(ctx, b.CategoryID),
		AmountMinor:    b.AmountMinor,
		SpentMinor:     b.SpentMinor,
		RemainingMinor: b.GetRemainingAmount(),
		Utilization:    utilization,
		Period:         b.Period,
		StartDate:      b.StartDate,
		EndDate:        b.EndDate,
		DaysRemaining:  max(daysBetween(today, b.EndDate), 0),
		IsActive:       b.IsActive,
		IsOverBudget:   isOverBudget,
		IsNearLimit:    utilization >= budgetNearLimitShare && !isOverBudget,
	}
}

func (s *statsService) recentTransactions(ctx context.Context) ([]dto.RecentTransaction, error) {
	transactions, err := s.transactions.GetAllTransactions(ctx, dto.TransactionFilterDTO{Limit: statsRecentLimit})
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	recent := make([]dto.RecentTransaction, 0, len(transactions))
	for _, tx := range transactions {
		recent = append(recent, dto.RecentTransaction{
			ID:           tx.ID,
			Description:  tx.Description,
			AmountMinor:  tx.AmountMinor,
			Type:         tx.Type,
			CategoryName: s.categoryName(ctx, &tx.CategoryID),
			Date:         tx.Date,
			CreatedAt:    tx.CreatedAt,
		})
	}

	return recent, nil
}

// categoryShares группирует транзакции периода по категориям; категории без имени пропускаются.
func (s *statsService) categoryShares(
	ctx context.Context,
	transactions []*transaction.Transaction,
	totals dto.PeriodTotals,
) ([]dto.CategoryShare, []dto.CategoryShare) {
	type bucket struct {
		share         dto.CategoryShare
		income        money.Minor
		expenses      money.Minor
		incomeCount   int
		expensesCount int
	}

	// Пустые, а не nil: в контракте оба поля — обязательные массивы (openapi StatsSummary).
	expenses := make([]dto.CategoryShare, 0)
	income := make([]dto.CategoryShare, 0)
	buckets := make(map[uuid.UUID]*bucket)
	for _, tx := range transactions {
		b, ok := buckets[tx.CategoryID]
		if !ok {
			category, err := s.categories.GetCategoryByID(ctx, tx.CategoryID)
			if err != nil || category == nil {
				continue
			}
			b = &bucket{share: dto.CategoryShare{
				CategoryID: tx.CategoryID,
				Name:       category.Name,
				Color:      category.Color,
				Icon:       category.Icon,
			}}
			buckets[tx.CategoryID] = b
		}

		switch tx.Type {
		case transaction.TypeIncome:
			b.income += tx.AmountMinor
			b.incomeCount++
		case transaction.TypeExpense:
			b.expenses += tx.AmountMinor
			b.expensesCount++
		}
	}

	for _, b := range buckets {
		if b.expenses > 0 {
			expenses = append(expenses, withAmount(b.share, b.expenses, b.expensesCount, totals.ExpensesMinor))
		}
		if b.income > 0 {
			income = append(income, withAmount(b.share, b.income, b.incomeCount, totals.IncomeMinor))
		}
	}

	sortCategoryShares(expenses)
	sortCategoryShares(income)

	return expenses, income
}

// categoryName возвращает имя категории или пустую строку, если её нет или она недоступна.
func (s *statsService) categoryName(ctx context.Context, categoryID *uuid.UUID) string {
	if categoryID == nil {
		return ""
	}

	category, err := s.categories.GetCategoryByID(ctx, *categoryID)
	if err != nil || category == nil {
		return ""
	}
	return category.Name
}

func periodTotals(from, to date.Date, transactions []*transaction.Transaction) dto.PeriodTotals {
	totals := dto.PeriodTotals{
		From:             from,
		To:               to,
		TransactionCount: len(transactions),
	}

	for _, tx := range transactions {
		switch tx.Type {
		case transaction.TypeIncome:
			totals.IncomeMinor += tx.AmountMinor
		case transaction.TypeExpense:
			totals.ExpensesMinor += tx.AmountMinor
		}
	}
	totals.NetMinor = totals.IncomeMinor - totals.ExpensesMinor

	return totals
}

// previousPeriod — период той же длины, вплотную перед [from, to].
func previousPeriod(from, to date.Date) (date.Date, date.Date) {
	previousTo := from.AddDays(-1)

	return previousTo.AddDays(-daysBetween(from, to)), previousTo
}

// daysBetween — число суток от from до to; отрицательное, если to раньше from.
func daysBetween(from, to date.Date) int {
	return int(to.In(time.UTC).Sub(from.In(time.UTC)).Hours() / statsHoursInDay)
}

func periodDeltas(current, previous dto.PeriodTotals, hasPrevious bool) (float64, float64) {
	if !hasPrevious {
		return 0, 0
	}

	// Percent сам возвращает 0 при нулевой базе, поэтому отдельной проверки нет.
	incomeDelta := (current.IncomeMinor - previous.IncomeMinor).Percent(previous.IncomeMinor) / percentToShare
	expensesDelta := (current.ExpensesMinor - previous.ExpensesMinor).Percent(previous.ExpensesMinor) / percentToShare

	return incomeDelta, expensesDelta
}

func withAmount(share dto.CategoryShare, amount money.Minor, count int, total money.Minor) dto.CategoryShare {
	share.AmountMinor = amount
	share.TransactionCount = count
	share.Share = amount.Percent(total) / percentToShare

	return share
}

func sortCategoryShares(shares []dto.CategoryShare) {
	slices.SortFunc(shares, func(a, b dto.CategoryShare) int {
		switch {
		case a.AmountMinor > b.AmountMinor:
			return -1
		case a.AmountMinor < b.AmountMinor:
			return 1
		default:
			return 0
		}
	})
}
