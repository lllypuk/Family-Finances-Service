package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
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
)

// statsService считает агрегаты поверх остальных сервисов, без прямого доступа к репозиториям.
type statsService struct {
	transactions TransactionService
	budgets      BudgetService
	categories   CategoryService
}

// NewStatsService создаёт сервис статистики
func NewStatsService(
	transactions TransactionService,
	budgets BudgetService,
	categories CategoryService,
) StatsService {
	return &statsService{
		transactions: transactions,
		budgets:      budgets,
		categories:   categories,
	}
}

// Summary собирает сводку за период [from, to] включительно
func (s *statsService) Summary(ctx context.Context, from, to time.Time) (*dto.StatsSummary, error) {
	if to.Before(from) {
		return nil, ErrInvalidStatsPeriod
	}

	current, err := s.transactionsBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}

	budgets, err := s.budgetProgress(ctx)
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

	totals := periodTotals(from, to, current)
	previousFrom, previousTo := previousPeriod(from, to)
	previous, hasPrevious := s.previousTotals(ctx, previousFrom, previousTo)
	expenseCategories, incomeCategories := s.categoryShares(ctx, current, totals)

	incomeDelta, expensesDelta := periodDeltas(totals, previous, hasPrevious)

	return &dto.StatsSummary{
		From:              from,
		To:                to,
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
	from, to time.Time,
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
func (s *statsService) previousTotals(ctx context.Context, from, to time.Time) (dto.PeriodTotals, bool) {
	if from.IsZero() || to.IsZero() {
		return dto.PeriodTotals{}, false
	}

	transactions, err := s.transactionsBetween(ctx, from, to)
	if err != nil || len(transactions) == 0 {
		return dto.PeriodTotals{From: from, To: to}, false
	}

	return periodTotals(from, to, transactions), true
}

func (s *statsService) budgetProgress(ctx context.Context) ([]dto.BudgetProgress, error) {
	now := time.Now()

	activeBudgets, err := s.budgets.GetActiveBudgets(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	progress := make([]dto.BudgetProgress, 0, len(activeBudgets))
	for _, b := range activeBudgets {
		progress = append(progress, s.budgetProgressItem(ctx, b, now))
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
	now time.Time,
) dto.BudgetProgress {
	utilization := 0.0
	if b.Amount > 0 {
		utilization = b.Spent / b.Amount
	}

	isOverBudget := utilization >= budgetOverLimitShare

	return dto.BudgetProgress{
		ID:            b.ID,
		Name:          b.Name,
		CategoryName:  s.categoryName(ctx, b.CategoryID),
		Amount:        b.Amount,
		Spent:         b.Spent,
		Remaining:     b.Amount - b.Spent,
		Utilization:   utilization,
		Period:        b.Period,
		StartDate:     b.StartDate,
		EndDate:       b.EndDate,
		DaysRemaining: max(int(b.EndDate.Sub(now).Hours()/statsHoursInDay), 0),
		IsActive:      b.IsActive,
		IsOverBudget:  isOverBudget,
		IsNearLimit:   utilization >= budgetNearLimitShare && !isOverBudget,
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
			Amount:       tx.Amount,
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
		income        float64
		expenses      float64
		incomeCount   int
		expensesCount int
	}

	var expenses, income []dto.CategoryShare
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
			b.income += tx.Amount
			b.incomeCount++
		case transaction.TypeExpense:
			b.expenses += tx.Amount
			b.expensesCount++
		}
	}

	for _, b := range buckets {
		if b.expenses > 0 {
			expenses = append(expenses, withAmount(b.share, b.expenses, b.expensesCount, totals.Expenses))
		}
		if b.income > 0 {
			income = append(income, withAmount(b.share, b.income, b.incomeCount, totals.Income))
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

func periodTotals(from, to time.Time, transactions []*transaction.Transaction) dto.PeriodTotals {
	totals := dto.PeriodTotals{
		From:             from,
		To:               to,
		TransactionCount: len(transactions),
	}

	for _, tx := range transactions {
		switch tx.Type {
		case transaction.TypeIncome:
			totals.Income += tx.Amount
		case transaction.TypeExpense:
			totals.Expenses += tx.Amount
		}
	}
	totals.Net = totals.Income - totals.Expenses

	return totals
}

// previousPeriod — период той же длины, вплотную перед [from, to].
func previousPeriod(from, to time.Time) (time.Time, time.Time) {
	previousTo := from.Add(-time.Second)
	return previousTo.Add(-to.Sub(from)), previousTo
}

func periodDeltas(current, previous dto.PeriodTotals, hasPrevious bool) (float64, float64) {
	if !hasPrevious {
		return 0, 0
	}

	var incomeDelta, expensesDelta float64
	if previous.Income > 0 {
		incomeDelta = (current.Income - previous.Income) / previous.Income
	}
	if previous.Expenses > 0 {
		expensesDelta = (current.Expenses - previous.Expenses) / previous.Expenses
	}
	return incomeDelta, expensesDelta
}

func withAmount(share dto.CategoryShare, amount float64, count int, total float64) dto.CategoryShare {
	share.Amount = amount
	share.TransactionCount = count
	if total > 0 {
		share.Share = amount / total
	}
	return share
}

func sortCategoryShares(shares []dto.CategoryShare) {
	slices.SortFunc(shares, func(a, b dto.CategoryShare) int {
		switch {
		case a.Amount > b.Amount:
			return -1
		case a.Amount < b.Amount:
			return 1
		default:
			return 0
		}
	})
}
