package services

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
)

// ErrInvalidStatsPeriod возвращается, когда конец периода раньше его начала.
var ErrInvalidStatsPeriod = errors.New("stats period end is before start")

// ErrStatsPeriodTooLong возвращается, когда период запрошен шире monthlyMaxMonths.
var ErrStatsPeriodTooLong = errors.New("stats period is too long")

// ErrStatsPeriodInFuture возвращается, когда конец ряда капитала позже сегодняшнего дня семьи.
var ErrStatsPeriodInFuture = errors.New("stats period end is later than today")

const (
	statsRecentLimit = 10
	// monthlyDefaultMonths — длина ряда по умолчанию в GET /stats/monthly.
	monthlyDefaultMonths = 12
	// monthlyMaxMonths — потолок ряда: на каждый месяц периода заводится корзина, а границы
	// приходят от клиента, так что 0001-01-01…9999-12-31 иначе выдаёт 120 000 корзин в JSON.
	monthlyMaxMonths = 120

	budgetNearLimitShare = 0.8
	budgetOverLimitShare = 1.0
)

// statsAggregates — суммы за период одним запросом; статистика берёт их из репозитория напрямую,
// потому что через сервис пришлось бы вычитывать все операции в память.
type statsAggregates interface {
	GetTotalsByCategoryAndDateRange(
		ctx context.Context,
		startDate, endDate date.Date,
	) ([]transaction.CategoryTotal, error)
	GetTotalsByMonth(ctx context.Context, startDate, endDate date.Date) ([]transaction.MonthTotal, error)
}

// holdingSeries — снимки позиций для ряда капитала.
type holdingSeries interface {
	SeriesValues(ctx context.Context, from, to date.Date) ([]holding.SeriesRow, error)
}

// statsService считает агрегаты поверх остальных сервисов; за суммами периода ходит в statsAggregates.
type statsService struct {
	transactions TransactionService
	budgets      BudgetService
	categories   CategoryService
	families     FamilyService
	aggregates   statsAggregates
	holdings     holdingSeries
}

// NewStatsService создаёт сервис статистики
func NewStatsService(
	transactions TransactionService,
	budgets BudgetService,
	categories CategoryService,
	families FamilyService,
	aggregates statsAggregates,
	holdings holdingSeries,
) StatsService {
	return &statsService{
		transactions: transactions,
		budgets:      budgets,
		categories:   categories,
		families:     families,
		aggregates:   aggregates,
		holdings:     holdings,
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

	totals, categoryTotals, err := s.totalsByCategory(ctx, start, end)
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

	// Фильтр по умолчанию, а не нулевой: нулевой Limit не проходит валидацию.
	total, err := s.transactions.CountTransactions(ctx, dto.NewTransactionFilterDTO())
	if err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	previousFrom, previousTo := previousPeriod(start, end)
	previous, _, err := s.totalsByCategory(ctx, previousFrom, previousTo)
	if err != nil {
		return nil, err
	}
	hasPrevious := previous.TransactionCount > 0
	expenseCategories, incomeCategories := s.categoryShares(ctx, categoryTotals, totals)

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

// Monthly собирает ряд по месяцам периода [from, to] включительно; nil-границы — двенадцать
// календарных месяцев по сегодняшний в часовом поясе семьи.
func (s *statsService) Monthly(ctx context.Context, from, to *date.Date) (*dto.StatsMonthly, error) {
	family, err := s.families.GetFamily(ctx)
	if err != nil {
		return nil, err
	}

	start, end, err := statsPeriod(date.Today(family.Location()), from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.aggregates.GetTotalsByMonth(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate transactions by month: %w", err)
	}

	return &dto.StatsMonthly{From: start, To: end, Months: monthBuckets(start, end, rows)}, nil
}

// NetWorth собирает ряд капитала за [from, to]; границы — как у Monthly, но to позже сегодня — отказ.
func (s *statsService) NetWorth(ctx context.Context, from, to *date.Date) (*dto.StatsNetWorth, error) {
	family, err := s.families.GetFamily(ctx)
	if err != nil {
		return nil, err
	}

	today := date.Today(family.Location())
	// До statsPeriod: иначе будущая граница пряталась бы за ErrStatsPeriodTooLong.
	if to != nil && to.After(today) {
		return nil, ErrStatsPeriodInFuture
	}

	start, end, err := statsPeriod(today, from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.holdings.SeriesValues(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to read holding values: %w", err)
	}

	return &dto.StatsNetWorth{From: start, To: end, Months: foldNetWorth(rows, start, end)}, nil
}

// totalsByCategory отдаёт итоги периода и строки агрегата, из которых они собраны.
func (s *statsService) totalsByCategory(
	ctx context.Context,
	from, to date.Date,
) (dto.PeriodTotals, []transaction.CategoryTotal, error) {
	rows, err := s.aggregates.GetTotalsByCategoryAndDateRange(ctx, from, to)
	if err != nil {
		return dto.PeriodTotals{}, nil, fmt.Errorf("failed to aggregate transactions for period: %w", err)
	}

	return periodTotals(from, to, rows), rows, nil
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
	utilization := b.GetSpentShare()

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
		DaysRemaining:  max(date.DaysBetween(today, b.EndDate), 0),
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

// categoryShares раскладывает строки агрегата по типу; категории без имени пропускаются.
func (s *statsService) categoryShares(
	ctx context.Context,
	rows []transaction.CategoryTotal,
	totals dto.PeriodTotals,
) ([]dto.CategoryShare, []dto.CategoryShare) {
	// Пустые, а не nil: в контракте оба поля — обязательные массивы (openapi StatsSummary).
	expenses := make([]dto.CategoryShare, 0)
	income := make([]dto.CategoryShare, 0)

	for _, row := range rows {
		if row.AmountMinor <= 0 {
			continue
		}

		category, err := s.categories.GetCategoryByID(ctx, row.CategoryID)
		if err != nil || category == nil {
			continue
		}
		share := dto.CategoryShare{
			CategoryID: row.CategoryID,
			Name:       category.Name,
			Color:      category.Color,
			Icon:       category.Icon,
		}

		switch row.Type {
		case transaction.TypeIncome:
			income = append(income, withAmount(share, row.AmountMinor, row.Count, totals.IncomeMinor))
		case transaction.TypeExpense:
			expenses = append(expenses, withAmount(share, row.AmountMinor, row.Count, totals.ExpensesMinor))
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

func periodTotals(from, to date.Date, rows []transaction.CategoryTotal) dto.PeriodTotals {
	totals := dto.PeriodTotals{From: from, To: to}

	for _, row := range rows {
		totals.TransactionCount += row.Count
		switch row.Type {
		case transaction.TypeIncome:
			totals.IncomeMinor += row.AmountMinor
		case transaction.TypeExpense:
			totals.ExpensesMinor += row.AmountMinor
		}
	}
	totals.NetMinor = totals.IncomeMinor - totals.ExpensesMinor

	return totals
}

// monthsBetween — число календарных месяцев, которые задевает период [from, to].
func monthsBetween(from, to date.Date) int {
	return (to.Year-from.Year)*12 + int(to.Month) - int(from.Month) + 1
}

// monthBuckets раскладывает строки агрегата по корзинам месяцев [from, to]; месяц без операций — нули.
// На месяц приходит 0–2 строки (по одной на тип) в непредсказуемом порядке, поэтому суммы накапливаются.
func monthBuckets(from, to date.Date, rows []transaction.MonthTotal) []dto.MonthTotals {
	// Пустой, а не nil: в контракте months — обязательный массив.
	months := make([]dto.MonthTotals, 0, monthlyDefaultMonths)
	index := make(map[string]int)

	last := date.Date{Year: to.Year, Month: to.Month, Day: 1}
	for m := (date.Date{Year: from.Year, Month: from.Month, Day: 1}); !m.After(last); m = m.AddMonths(1) {
		index[m.MonthKey()] = len(months)
		months = append(months, dto.MonthTotals{Month: m.MonthKey()})
	}

	for _, row := range rows {
		i, ok := index[row.Month]
		if !ok {
			continue
		}
		months[i].TransactionCount += row.Count
		switch row.Type {
		case transaction.TypeIncome:
			months[i].IncomeMinor += row.AmountMinor
		case transaction.TypeExpense:
			months[i].ExpensesMinor += row.AmountMinor
		}
	}

	for i := range months {
		months[i].NetMinor = months[i].IncomeMinor - months[i].ExpensesMinor
	}

	return months
}

// previousPeriod — период той же длины, вплотную перед [from, to].
func previousPeriod(from, to date.Date) (date.Date, date.Date) {
	previousTo := from.AddDays(-1)

	return previousTo.AddDays(-date.DaysBetween(from, to)), previousTo
}

func periodDeltas(current, previous dto.PeriodTotals, hasPrevious bool) (float64, float64) {
	if !hasPrevious {
		return 0, 0
	}

	// Share сам возвращает 0 при нулевой базе, поэтому отдельной проверки нет.
	incomeDelta := (current.IncomeMinor - previous.IncomeMinor).Share(previous.IncomeMinor)
	expensesDelta := (current.ExpensesMinor - previous.ExpensesMinor).Share(previous.ExpensesMinor)

	return incomeDelta, expensesDelta
}

func withAmount(share dto.CategoryShare, amount money.Minor, count int, total money.Minor) dto.CategoryShare {
	share.AmountMinor = amount
	share.TransactionCount = count
	share.Share = amount.Share(total)

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
