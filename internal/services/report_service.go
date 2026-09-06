package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"

	"family-budget-service/internal/services/dto"
)

// Constants for report generation
const (
	topTransactionsLimit        = 10
	percentageMultiplier        = 100.0
	hoursPerDay                 = 24
	daysPerWeek                 = 7
	lastDayOfDecember           = 31
	reportTransactionQueryLimit = 1000 // Maximum transactions to query for reports

	exportFormatCSV = "csv"
)

var (
	ErrReportFeatureHiddenFromPublicAPI = errors.New("report feature hidden from public API until implemented")
	ErrUnsupportedReportType            = errors.New("unsupported report type")
	ErrUnsupportedCSVData               = errors.New("csv export supports report data only")
	ErrReportNotFound                   = errors.New("report not found")
)

type reportService struct {
	reportRepo      ReportRepository
	transactionRepo TransactionRepository
	budgetRepo      BudgetRepository
	categoryRepo    CategoryRepository
	userRepo        UserRepository
	familyRepo      FamilyRepository

	// Service dependencies for complex calculations
	transactionService TransactionService
	budgetService      BudgetService
	categoryService    CategoryService
}

// NewReportService creates a new report service instance
func NewReportService(
	reportRepo ReportRepository,
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepository,
	categoryRepo CategoryRepository,
	userRepo UserRepository,
	familyRepo FamilyRepository,
	transactionService TransactionService,
	budgetService BudgetService,
	categoryService CategoryService,
) ReportService {
	return &reportService{
		reportRepo:         reportRepo,
		transactionRepo:    transactionRepo,
		budgetRepo:         budgetRepo,
		categoryRepo:       categoryRepo,
		userRepo:           userRepo,
		familyRepo:         familyRepo,
		transactionService: transactionService,
		budgetService:      budgetService,
		categoryService:    categoryService,
	}
}

func reportFeatureHiddenStubError(feature string) error {
	return fmt.Errorf("%w: %s", ErrReportFeatureHiddenFromPublicAPI, feature)
}

// transactionReportData contains common data for transaction reports
type transactionReportData struct {
	transactions      []*transaction.Transaction
	totalAmount       money.Minor
	averageDaily      money.Minor
	categoryBreakdown []dto.CategoryBreakdownItemDTO
	topTransactions   []dto.TransactionSummaryDTO
}

// generateTransactionReport generates common transaction report data
func (s *reportService) generateTransactionReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
	transactionType transaction.Type,
) (*transactionReportData, error) {
	// Get transactions for the period
	transactions, err := s.getTransactionsForPeriod(
		ctx,
		req.StartDate,
		req.EndDate,
		transactionType,
		req.Filters,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s transactions: %w", transactionType, err)
	}

	// Calculate basic metrics
	totalAmount := s.calculateTotalAmount(transactions)
	averageDaily := s.calculateAverageDaily(totalAmount, req.StartDate, req.EndDate)

	// Generate category breakdown
	categoryBreakdown := s.generateCategoryBreakdown(ctx, transactions)

	// Get top transactions
	topTransactions := s.getTopTransactions(ctx, transactions, topTransactionsLimit)

	return &transactionReportData{
		transactions:      transactions,
		totalAmount:       totalAmount,
		averageDaily:      averageDaily,
		categoryBreakdown: categoryBreakdown,
		topTransactions:   topTransactions,
	}, nil
}

// completeTransactionReportData contains full transaction report data
type completeTransactionReportData struct {
	*transactionReportData

	dailyBreakdownExpense []dto.DailyExpenseDTO
	dailyBreakdownIncome  []dto.DailyIncomeDTO
	expenseTrends         dto.ExpenseTrendsDTO
	incomeTrends          dto.IncomeTrendsDTO
	expenseComparisons    dto.ExpenseComparisonsDTO
	incomeComparisons     dto.IncomeComparisonsDTO
}

// generateTransactionReportComplete generates complete transaction report with all components
func (s *reportService) generateTransactionReportComplete(
	ctx context.Context,
	req dto.ReportRequestDTO,
	transactionType transaction.Type,
) (*completeTransactionReportData, error) {
	// Get base transaction data
	baseData, err := s.generateTransactionReport(ctx, req, transactionType)
	if err != nil {
		return nil, err
	}

	result := &completeTransactionReportData{
		transactionReportData: baseData,
	}

	// Generate type-specific components
	if transactionType == transaction.TypeExpense {
		if err = s.generateExpenseSpecificData(ctx, req, baseData, result); err != nil {
			return nil, err
		}
	} else {
		if err = s.generateIncomeSpecificData(ctx, req, baseData, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// GenerateExpenseReport generates a comprehensive expense analysis report
func (s *reportService) GenerateExpenseReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.ExpenseReportDTO, error) {
	reportData, err := s.generateTransactionReportComplete(ctx, req, transaction.TypeExpense)
	if err != nil {
		return nil, err
	}

	return &dto.ExpenseReportDTO{
		ID:                 uuid.New(),
		Name:               req.Name,
		UserID:             req.UserID,
		Period:             req.Period,
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		TotalExpensesMinor: reportData.totalAmount,
		AverageDailyMinor:  reportData.averageDaily,
		CategoryBreakdown:  reportData.categoryBreakdown,
		DailyBreakdown:     reportData.dailyBreakdownExpense,
		TopExpenses:        reportData.topTransactions,
		Trends:             reportData.expenseTrends,
		Comparisons:        reportData.expenseComparisons,
		GeneratedAt:        time.Now(),
	}, nil
}

// GenerateIncomeReport generates a comprehensive income analysis report
func (s *reportService) GenerateIncomeReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.IncomeReportDTO, error) {
	reportData, err := s.generateTransactionReportComplete(ctx, req, transaction.TypeIncome)
	if err != nil {
		return nil, err
	}

	return &dto.IncomeReportDTO{
		ID:                uuid.New(),
		Name:              req.Name,
		UserID:            req.UserID,
		Period:            req.Period,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		TotalIncomeMinor:  reportData.totalAmount,
		AverageDailyMinor: reportData.averageDaily,
		CategoryBreakdown: reportData.categoryBreakdown,
		DailyBreakdown:    reportData.dailyBreakdownIncome,
		TopSources:        reportData.topTransactions,
		Trends:            reportData.incomeTrends,
		Comparisons:       reportData.incomeComparisons,
		GeneratedAt:       time.Now(),
	}, nil
}

// GenerateBudgetComparisonReport generates budget vs actual spending comparison
func (s *reportService) GenerateBudgetComparisonReport(
	ctx context.Context,
	period report.Period,
) (*dto.BudgetComparisonDTO, error) {
	startDate, endDate := s.calculatePeriodDates(ctx, period)

	return s.budgetComparisonReport(ctx, period, startDate, endDate)
}

func (s *reportService) budgetComparisonReport(
	ctx context.Context,
	period report.Period,
	startDate, endDate date.Date,
) (*dto.BudgetComparisonDTO, error) {
	// Get active budgets for the period
	budgets, err := s.budgetService.GetActiveBudgets(ctx, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	if len(budgets) == 0 {
		return &dto.BudgetComparisonDTO{
			ID:          uuid.New(),
			Name:        fmt.Sprintf("Budget Comparison - %s", period),
			Period:      period,
			StartDate:   startDate,
			EndDate:     endDate,
			Categories:  []dto.BudgetCategoryComparisonDTO{},
			Timeline:    []dto.BudgetTimelineDTO{},
			Alerts:      []dto.BudgetAlertReportDTO{},
			GeneratedAt: time.Now(),
		}, nil
	}

	// Calculate totals
	var totalBudget money.Minor
	for _, b := range budgets {
		totalBudget += b.AmountMinor
	}

	// Get actual spending for the same period
	expenseTransactions, err := s.getTransactionsForPeriod(
		ctx,
		startDate,
		endDate,
		transaction.TypeExpense,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get expense transactions: %w", err)
	}

	totalSpent := s.calculateTotalAmount(expenseTransactions)
	totalVariance := totalBudget - totalSpent
	utilization := totalSpent.Percent(totalBudget)

	// Generate category comparisons
	categoryComparisons, err := s.generateBudgetCategoryComparisons(ctx, budgets, expenseTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate category comparisons: %w", err)
	}

	// Generate timeline
	timeline := s.generateBudgetTimeline(expenseTransactions, totalBudget, startDate, endDate)

	// Generate alerts
	alerts := s.generateBudgetAlerts(categoryComparisons)

	return &dto.BudgetComparisonDTO{
		ID:                 uuid.New(),
		Name:               fmt.Sprintf("Budget Comparison - %s", period),
		Period:             period,
		StartDate:          startDate,
		EndDate:            endDate,
		TotalBudgetMinor:   totalBudget,
		TotalSpentMinor:    totalSpent,
		TotalVarianceMinor: totalVariance,
		Utilization:        utilization,
		Categories:         categoryComparisons,
		Timeline:           timeline,
		Alerts:             alerts,
		GeneratedAt:        time.Now(),
	}, nil
}

// GenerateCashFlowReport generates cash flow analysis report
func (s *reportService) GenerateCashFlowReport(
	ctx context.Context,
	from, to date.Date,
) (*dto.CashFlowReportDTO, error) {
	// Get all transactions for the period
	allTransactions, err := s.getTransactionsForPeriod(ctx, from, to, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Separate income and expenses
	incomeTransactions := s.filterTransactionsByType(allTransactions, transaction.TypeIncome)
	expenseTransactions := s.filterTransactionsByType(allTransactions, transaction.TypeExpense)

	totalInflows := s.calculateTotalAmount(incomeTransactions)
	totalOutflows := s.calculateTotalAmount(expenseTransactions)
	netCashFlow := totalInflows - totalOutflows

	// ROADMAP: opening balance must be calculated from prior periods or stored snapshots.
	var openingBalance money.Minor
	closingBalance := openingBalance + netCashFlow

	// Generate daily cash flow
	dailyFlow := s.generateDailyCashFlow(allTransactions, openingBalance)

	// Generate weekly and monthly aggregations
	weeklyFlow := s.generateWeeklyCashFlow(dailyFlow)
	monthlyFlow := s.generateMonthlyCashFlow(dailyFlow)

	// Generate projections
	projections, err := s.generateCashFlowProjections(ctx, allTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate projections: %w", err)
	}

	return &dto.CashFlowReportDTO{
		ID:                  uuid.New(),
		Name:                fmt.Sprintf("Cash Flow Report - %s to %s", from, to),
		Period:              report.PeriodCustom,
		StartDate:           from,
		EndDate:             to,
		OpeningBalanceMinor: openingBalance,
		ClosingBalanceMinor: closingBalance,
		NetCashFlowMinor:    netCashFlow,
		TotalInflowsMinor:   totalInflows,
		TotalOutflowsMinor:  totalOutflows,
		DailyFlow:           dailyFlow,
		WeeklyFlow:          weeklyFlow,
		MonthlyFlow:         monthlyFlow,
		Projections:         projections,
		GeneratedAt:         time.Now(),
	}, nil
}

// GenerateCategoryBreakdownReport generates detailed category analysis
func (s *reportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	startDate, endDate := s.calculatePeriodDates(ctx, period)

	return s.categoryBreakdownReport(ctx, period, startDate, endDate)
}

func (s *reportService) categoryBreakdownReport(
	ctx context.Context,
	period report.Period,
	startDate, endDate date.Date,
) (*dto.CategoryBreakdownDTO, error) {
	// Get all transactions for the period
	transactions, err := s.getTransactionsForPeriod(ctx, startDate, endDate, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Get category hierarchy
	categories, err := s.categoryService.GetCategoryHierarchy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get category hierarchy: %w", err)
	}

	// Generate detailed category analysis
	categoryAnalysis, err := s.generateDetailedCategoryAnalysis(ctx, transactions, categories, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate category analysis: %w", err)
	}

	// Generate category hierarchy with amounts
	hierarchy := s.generateCategoryHierarchy(categoryAnalysis, categories)

	// Generate category trends
	trends, err := s.generateCategoryTrends(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate category trends: %w", err)
	}

	// Generate category comparisons
	comparisons, err := s.generateCategoryComparisons(ctx, categoryAnalysis, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate category comparisons: %w", err)
	}

	return &dto.CategoryBreakdownDTO{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("Category Breakdown - %s", period),
		Period:      period,
		StartDate:   startDate,
		EndDate:     endDate,
		Categories:  categoryAnalysis,
		Hierarchy:   hierarchy,
		Trends:      trends,
		Comparisons: comparisons,
		GeneratedAt: time.Now(),
	}, nil
}

// GenerateReport строит отчёт запрошенного типа: генерация данных плюс конвертация в report.Data.
//
// Отчёт не сохраняется: HTMX-предпросмотр показывает его, не записывая в БД.
// Неизвестный тип — ErrUnsupportedReportType.
func (s *reportService) GenerateReport(ctx context.Context, req dto.ReportRequestDTO) (*report.Report, error) {
	reportData, err := s.generateReportData(ctx, req)
	if err != nil {
		return nil, err
	}

	data, err := s.convertToReportData(reportData, req.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to convert report data: %w", err)
	}

	newReport := report.NewReport(
		req.Name, req.Type, req.Period, req.UserID,
		req.StartDate, req.EndDate,
	)
	newReport.Data = data

	return newReport, nil
}

func (s *reportService) generateReportData(ctx context.Context, req dto.ReportRequestDTO) (any, error) {
	switch req.Type {
	case report.TypeExpenses:
		return s.GenerateExpenseReport(ctx, req)
	case report.TypeIncome:
		return s.GenerateIncomeReport(ctx, req)
	case report.TypeBudget:
		return s.budgetComparisonReport(ctx, req.Period, req.StartDate, req.EndDate)
	case report.TypeCashFlow:
		return s.GenerateCashFlowReport(ctx, req.StartDate, req.EndDate)
	case report.TypeCategoryBreak:
		return s.categoryBreakdownReport(ctx, req.Period, req.StartDate, req.EndDate)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedReportType, req.Type)
	}
}

// SaveReport сохраняет сгенерированный отчёт.
func (s *reportService) SaveReport(ctx context.Context, reportEntity *report.Report) error {
	if err := s.reportRepo.Create(ctx, reportEntity); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

// GetReportByID retrieves a report by its ID
func (s *reportService) GetReportByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	return s.reportRepo.GetByID(ctx, id)
}

// GetReports retrieves all reports (single family model)
func (s *reportService) GetReports(
	ctx context.Context,
	_ *report.Type,
) ([]*report.Report, error) {
	return s.reportRepo.GetAll(ctx)
}

// GetReportsByUserID retrieves reports for a specific user (single family model).
func (s *reportService) GetReportsByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	return s.reportRepo.GetByUserID(ctx, userID)
}

// DeleteReport deletes a report by its ID
func (s *reportService) DeleteReport(ctx context.Context, id uuid.UUID) error {
	if _, err := s.reportRepo.GetByID(ctx, id); err != nil {
		return ErrReportNotFound
	}

	return s.reportRepo.Delete(ctx, id)
}

// ExportReport exports a saved report in the specified format
func (s *reportService) ExportReport(
	ctx context.Context,
	reportID uuid.UUID,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	reportEntity, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	// Тип отчёта известен только здесь: ExportReportData получает голые данные,
	// а колонки CSV зависят от типа.
	if strings.EqualFold(format, exportFormatCSV) {
		return reportToCSV(reportEntity.Data, reportEntity.Type, s.currency(ctx))
	}

	return s.ExportReportData(ctx, reportEntity.Data, format, options)
}

// ExportReportData exports report data in the specified format
func (s *reportService) ExportReportData(
	_ context.Context,
	reportData any,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return json.Marshal(reportData)
	case exportFormatCSV:
		return s.exportToCSV(reportData, options)
	case "excel":
		return s.exportToExcel(reportData, options)
	case "pdf":
		return s.exportToPDF(reportData, options)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Helper methods for report generation

func (s *reportService) getTransactionsForPeriod(
	ctx context.Context,
	startDate, endDate date.Date,
	transactionType transaction.Type,
	filters *dto.ReportFilters,
) ([]*transaction.Transaction, error) {
	// Build filter for transaction service
	filter := dto.TransactionFilterDTO{
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    reportTransactionQueryLimit,
	}

	if transactionType != "" {
		filter.Type = &transactionType
	}

	if filters != nil {
		if len(filters.CategoryIDs) > 0 {
			filter.CategoryID = &filters.CategoryIDs[0] // Take first category for now
		}
		if len(filters.UserIDs) > 0 {
			filter.UserID = &filters.UserIDs[0] // Take first user for now
		}
		filter.AmountFromMinor = filters.MinAmountMinor
		filter.AmountToMinor = filters.MaxAmountMinor
		if filters.Description != "" {
			filter.Description = &filters.Description
		}
	}

	return s.transactionService.GetAllTransactions(ctx, filter)
}

func (s *reportService) calculateTotalAmount(transactions []*transaction.Transaction) money.Minor {
	var total money.Minor
	for _, t := range transactions {
		total += t.AmountMinor
	}
	return total
}

// calculateAverageDaily делит сумму на число суток периода включительно; деление —
// money.DivRound, чтобы копейка не терялась по-разному в каждом вызове.
func (s *reportService) calculateAverageDaily(total money.Minor, startDate, endDate date.Date) money.Minor {
	days := daysBetween(startDate, endDate) + 1
	if days <= 0 {
		return 0
	}
	return total.DivRound(int64(days))
}

func (s *reportService) generateCategoryBreakdown(
	ctx context.Context,
	transactions []*transaction.Transaction,
) []dto.CategoryBreakdownItemDTO {
	// Group transactions by category
	categoryTotals := make(map[uuid.UUID]money.Minor)
	categoryCounts := make(map[uuid.UUID]int)

	var total money.Minor
	for _, t := range transactions {
		if t.CategoryID != uuid.Nil {
			categoryTotals[t.CategoryID] += t.AmountMinor
			categoryCounts[t.CategoryID]++
			total += t.AmountMinor
		}
	}

	// Get category details
	var result []dto.CategoryBreakdownItemDTO
	for categoryID, amount := range categoryTotals {
		cat, err := s.categoryService.GetCategoryByID(ctx, categoryID)
		if err != nil {
			continue // Skip if category not found
		}

		var avgAmount money.Minor
		if count := categoryCounts[categoryID]; count > 0 {
			avgAmount = amount.DivRound(int64(count))
		}

		result = append(result, dto.CategoryBreakdownItemDTO{
			CategoryID:         categoryID,
			CategoryName:       cat.Name,
			CategoryType:       string(cat.Type),
			AmountMinor:        amount,
			Percentage:         amount.Percent(total),
			Count:              categoryCounts[categoryID],
			AverageAmountMinor: avgAmount,
			ParentID:           cat.ParentID,
		})
	}

	// Sort by amount descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].AmountMinor > result[j].AmountMinor
	})

	return result
}

func (s *reportService) generateDailyExpenseBreakdown(transactions []*transaction.Transaction) []dto.DailyExpenseDTO {
	daily := aggregateByDay(transactions)

	result := make([]dto.DailyExpenseDTO, 0, len(daily))
	for day, data := range daily {
		result = append(result, dto.DailyExpenseDTO{
			Date:        day,
			AmountMinor: data.amount,
			Count:       data.count,
			// Categories would be populated with actual category names
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result
}

// dayTotal — сумма и число операций одного календарного дня.
type dayTotal struct {
	amount money.Minor
	count  int
}

func aggregateByDay(transactions []*transaction.Transaction) map[date.Date]dayTotal {
	daily := make(map[date.Date]dayTotal, len(transactions))
	for _, t := range transactions {
		entry := daily[t.Date]
		entry.amount += t.AmountMinor
		entry.count++
		daily[t.Date] = entry
	}

	return daily
}

func (s *reportService) generateDailyIncomeBreakdown(transactions []*transaction.Transaction) []dto.DailyIncomeDTO {
	daily := aggregateByDay(transactions)

	result := make([]dto.DailyIncomeDTO, 0, len(daily))
	for day, data := range daily {
		result = append(result, dto.DailyIncomeDTO{
			Date:        day,
			AmountMinor: data.amount,
			Count:       data.count,
			Sources:     []string{}, // Would be populated with actual source names
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result
}

func (s *reportService) getTopTransactions(
	ctx context.Context,
	transactions []*transaction.Transaction,
	limit int,
) []dto.TransactionSummaryDTO {
	// Sort by amount descending
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].AmountMinor > transactions[j].AmountMinor
	})

	var result []dto.TransactionSummaryDTO
	for i, t := range transactions {
		if i >= limit {
			break
		}

		categoryName := "Unknown"
		if t.CategoryID != uuid.Nil {
			if cat, err := s.categoryService.GetCategoryByID(ctx, t.CategoryID); err == nil {
				categoryName = cat.Name
			}
		}

		userName := "Unknown"
		if user, err := s.userRepo.GetByID(ctx, t.UserID); err == nil {
			userName = user.FirstName + " " + user.LastName
		}

		result = append(result, dto.TransactionSummaryDTO{
			ID:          t.ID,
			AmountMinor: t.AmountMinor,
			Description: t.Description,
			Category:    categoryName,
			Date:        t.Date,
			UserName:    userName,
		})
	}

	return result
}

// calculatePeriodDates возвращает включительные границы периода в зоне семьи;
// при недоступной семье — UTC, отчёт важнее точной полуночи.
func (s *reportService) calculatePeriodDates(ctx context.Context, period report.Period) (date.Date, date.Date) {
	loc := time.UTC
	if family, err := s.familyRepo.Get(ctx); err == nil && family != nil {
		loc = family.Location()
	}

	today := date.Today(loc)
	firstOfMonth, lastOfMonth := today.MonthBounds()

	switch period {
	case report.PeriodDaily:
		return today, today
	case report.PeriodWeekly:
		start := today.AddDays(-int(today.In(loc).Weekday()))
		return start, start.AddDays(daysPerWeek - 1)
	case report.PeriodMonthly, report.PeriodCustom:
		return firstOfMonth, lastOfMonth
	case report.PeriodYearly:
		return date.New(today.Year, time.January, 1), date.New(today.Year, time.December, lastDayOfDecember)
	default:
		return firstOfMonth, lastOfMonth
	}
}

func (s *reportService) filterTransactionsByType(
	transactions []*transaction.Transaction,
	transactionType transaction.Type,
) []*transaction.Transaction {
	var result []*transaction.Transaction
	for _, t := range transactions {
		if t.Type == transactionType {
			result = append(result, t)
		}
	}
	return result
}

// ROADMAP placeholders for advanced calculations used by implemented report flows.
// They intentionally return zero-values to keep current report generation stable while
// signaling missing depth in analytics/report details.

func (s *reportService) generateExpenseTrends(
	_ context.Context,
	_, _ date.Date,
) (dto.ExpenseTrendsDTO, error) {
	// ROADMAP: implement sophisticated expense trend analysis.
	return dto.ExpenseTrendsDTO{}, nil
}

func (s *reportService) generateExpenseComparisons(
	_ context.Context,
	_ money.Minor,
	_, _ date.Date,
) (dto.ExpenseComparisonsDTO, error) {
	// ROADMAP: compare expenses with previous periods.
	return dto.ExpenseComparisonsDTO{}, nil
}

func (s *reportService) generateIncomeTrends(
	_ context.Context,
	_, _ date.Date,
) (dto.IncomeTrendsDTO, error) {
	// ROADMAP: implement income trend analysis.
	return dto.IncomeTrendsDTO{}, nil
}

func (s *reportService) generateIncomeComparisons(
	_ context.Context,
	_ money.Minor,
	_, _ date.Date,
) (dto.IncomeComparisonsDTO, error) {
	// ROADMAP: compare income with previous periods.
	return dto.IncomeComparisonsDTO{}, nil
}

func (s *reportService) generateBudgetCategoryComparisons(
	_ context.Context,
	_ []*budget.Budget,
	_ []*transaction.Transaction,
) ([]dto.BudgetCategoryComparisonDTO, error) {
	// ROADMAP: budget-vs-actual comparison by category.
	return []dto.BudgetCategoryComparisonDTO{}, nil
}

func (s *reportService) generateBudgetTimeline(
	_ []*transaction.Transaction,
	_ money.Minor,
	_, _ date.Date,
) []dto.BudgetTimelineDTO {
	// ROADMAP: budget timeline generation.
	return []dto.BudgetTimelineDTO{}
}

func (s *reportService) generateBudgetAlerts(_ []dto.BudgetCategoryComparisonDTO) []dto.BudgetAlertReportDTO {
	// ROADMAP: budget alert generation.
	return []dto.BudgetAlertReportDTO{}
}

// generateDailyCashFlow сводит операции по календарным дням периода; Balance —
// нарастающий итог от openingBalance. Дни без операций пропускаются.
func (s *reportService) generateDailyCashFlow(
	transactions []*transaction.Transaction,
	openingBalance money.Minor,
) []dto.DailyCashFlowDTO {
	byDay := make(map[date.Date]*dto.DailyCashFlowDTO)
	for _, tx := range transactions {
		item, ok := byDay[tx.Date]
		if !ok {
			item = &dto.DailyCashFlowDTO{Date: tx.Date}
			byDay[tx.Date] = item
		}

		switch tx.Type {
		case transaction.TypeIncome:
			item.InflowMinor += tx.AmountMinor
		case transaction.TypeExpense:
			item.OutflowMinor += tx.AmountMinor
		}
	}

	days := make([]dto.DailyCashFlowDTO, 0, len(byDay))
	for _, item := range byDay {
		days = append(days, *item)
	}
	slices.SortFunc(days, func(a, b dto.DailyCashFlowDTO) int {
		return a.Date.In(time.UTC).Compare(b.Date.In(time.UTC))
	})

	balance := openingBalance
	for i := range days {
		days[i].NetFlowMinor = days[i].InflowMinor - days[i].OutflowMinor
		balance += days[i].NetFlowMinor
		days[i].BalanceMinor = balance
	}

	return days
}

func (s *reportService) generateWeeklyCashFlow(_ []dto.DailyCashFlowDTO) []dto.WeeklyCashFlowDTO {
	// ROADMAP: weekly cash-flow aggregation.
	return []dto.WeeklyCashFlowDTO{}
}

func (s *reportService) generateMonthlyCashFlow(_ []dto.DailyCashFlowDTO) []dto.MonthlyCashFlowDTO {
	// ROADMAP: monthly cash-flow aggregation.
	return []dto.MonthlyCashFlowDTO{}
}

func (s *reportService) generateCashFlowProjections(
	_ context.Context,
	_ []*transaction.Transaction,
) (dto.CashFlowProjectionsDTO, error) {
	// ROADMAP: cash-flow projections.
	return dto.CashFlowProjectionsDTO{}, nil
}

func (s *reportService) generateDetailedCategoryAnalysis(
	_ context.Context,
	_ []*transaction.Transaction,
	_ []*category.Category,
	_, _ date.Date,
) ([]dto.CategoryAnalysisDTO, error) {
	// ROADMAP: detailed category analysis.
	return []dto.CategoryAnalysisDTO{}, nil
}

func (s *reportService) generateCategoryHierarchy(
	_ []dto.CategoryAnalysisDTO,
	_ []*category.Category,
) []dto.CategoryHierarchyReportDTO {
	// ROADMAP: category hierarchy report generation.
	return []dto.CategoryHierarchyReportDTO{}
}

func (s *reportService) generateCategoryTrends(
	_ context.Context,
	_, _ date.Date,
) (dto.CategoryTrendsDTO, error) {
	// ROADMAP: category trend analysis.
	return dto.CategoryTrendsDTO{}, nil
}

func (s *reportService) generateCategoryComparisons(
	_ context.Context,
	_ []dto.CategoryAnalysisDTO,
	_, _ date.Date,
) (dto.CategoryComparisonsDTO, error) {
	// ROADMAP: category comparison analysis.
	return dto.CategoryComparisonsDTO{}, nil
}

func (s *reportService) convertToReportData(reportData any, reportType report.Type) (report.Data, error) {
	switch reportType {
	case report.TypeExpenses:
		expenseReport, ok := reportData.(*dto.ExpenseReportDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.ExpenseReportDTO, got %T", reportData)
		}
		return report.Data{
			TotalExpensesMinor: expenseReport.TotalExpensesMinor,
			CategoryBreakdown:  convertCategoryBreakdownItemsToReportData(expenseReport.CategoryBreakdown),
			TopExpenses:        convertTransactionSummaryItemsToReportData(expenseReport.TopExpenses),
		}, nil

	case report.TypeIncome:
		incomeReport, ok := reportData.(*dto.IncomeReportDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.IncomeReportDTO, got %T", reportData)
		}
		// Persist top sources in TopExpenses generic field for unified rendering/storage.
		return report.Data{
			TotalIncomeMinor:  incomeReport.TotalIncomeMinor,
			CategoryBreakdown: convertCategoryBreakdownItemsToReportData(incomeReport.CategoryBreakdown),
			TopExpenses:       convertTransactionSummaryItemsToReportData(incomeReport.TopSources),
		}, nil

	case report.TypeBudget:
		budgetReport, ok := reportData.(*dto.BudgetComparisonDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.BudgetComparisonDTO, got %T", reportData)
		}
		return report.Data{
			TotalExpensesMinor: budgetReport.TotalSpentMinor,
			BudgetComparison:   convertBudgetComparisonItemsToReportData(budgetReport.Categories),
		}, nil

	case report.TypeCashFlow:
		cashFlowReport, ok := reportData.(*dto.CashFlowReportDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.CashFlowReportDTO, got %T", reportData)
		}
		return report.Data{
			TotalIncomeMinor:   cashFlowReport.TotalInflowsMinor,
			TotalExpensesMinor: cashFlowReport.TotalOutflowsMinor,
			NetIncomeMinor:     cashFlowReport.NetCashFlowMinor,
			DailyBreakdown:     convertDailyCashFlowItemsToReportData(cashFlowReport.DailyFlow),
		}, nil

	case report.TypeCategoryBreak:
		categoryReport, ok := reportData.(*dto.CategoryBreakdownDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.CategoryBreakdownDTO, got %T", reportData)
		}
		return report.Data{
			CategoryBreakdown: convertCategoryAnalysisItemsToReportData(categoryReport.Categories),
		}, nil

	default:
		return report.Data{}, fmt.Errorf("unsupported report type: %s", reportType)
	}
}

func convertCategoryBreakdownItemsToReportData(items []dto.CategoryBreakdownItemDTO) []report.CategoryReportItem {
	if len(items) == 0 {
		return []report.CategoryReportItem{}
	}

	result := make([]report.CategoryReportItem, len(items))
	for i, item := range items {
		result[i] = report.CategoryReportItem{
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			AmountMinor:  item.AmountMinor,
			Percentage:   item.Percentage,
			Count:        item.Count,
		}
	}
	return result
}

func convertCategoryAnalysisItemsToReportData(items []dto.CategoryAnalysisDTO) []report.CategoryReportItem {
	if len(items) == 0 {
		return []report.CategoryReportItem{}
	}

	result := make([]report.CategoryReportItem, len(items))
	for i, item := range items {
		result[i] = report.CategoryReportItem{
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			AmountMinor:  item.TotalAmountMinor,
			Percentage:   item.Percentage,
			Count:        item.TransactionCount,
		}
	}
	return result
}

func convertTransactionSummaryItemsToReportData(items []dto.TransactionSummaryDTO) []report.TransactionReportItem {
	if len(items) == 0 {
		return []report.TransactionReportItem{}
	}

	result := make([]report.TransactionReportItem, len(items))
	for i, item := range items {
		result[i] = report.TransactionReportItem{
			ID:          item.ID,
			AmountMinor: item.AmountMinor,
			Description: item.Description,
			Category:    item.Category,
			Date:        item.Date,
		}
	}
	return result
}

func convertBudgetComparisonItemsToReportData(items []dto.BudgetCategoryComparisonDTO) []report.BudgetComparisonItem {
	if len(items) == 0 {
		return []report.BudgetComparisonItem{}
	}

	result := make([]report.BudgetComparisonItem, len(items))
	for i, item := range items {
		result[i] = report.BudgetComparisonItem{
			BudgetID:        item.CategoryID, // Generic report.Data has no category-specific budget key.
			BudgetName:      item.CategoryName,
			PlannedMinor:    item.BudgetAmountMinor,
			ActualMinor:     item.ActualAmountMinor,
			DifferenceMinor: item.VarianceMinor,
			Percentage:      item.Utilization,
		}
	}
	return result
}

func convertDailyCashFlowItemsToReportData(items []dto.DailyCashFlowDTO) []report.DailyReportItem {
	if len(items) == 0 {
		return []report.DailyReportItem{}
	}

	result := make([]report.DailyReportItem, len(items))
	for i, item := range items {
		result[i] = report.DailyReportItem{
			Date:          item.Date,
			IncomeMinor:   item.InflowMinor,
			ExpensesMinor: item.OutflowMinor,
			BalanceMinor:  item.BalanceMinor,
		}
	}
	return result
}

// exportToCSV принимает только report.Data: для прочих структур набор колонок неизвестен.
func (s *reportService) exportToCSV(reportData any, options dto.ExportOptionsDTO) ([]byte, error) {
	data, ok := reportData.(report.Data)
	if !ok {
		return nil, ErrUnsupportedCSVData
	}

	return reportToCSV(data, "", options.Currency)
}

// currency — код валюты семьи для колонки CSV; при недоступной семье пусто.
func (s *reportService) currency(ctx context.Context) string {
	family, err := s.familyRepo.Get(ctx)
	if err != nil || family == nil {
		return ""
	}

	return family.Currency
}

func (s *reportService) exportToExcel(_ any, _ dto.ExportOptionsDTO) ([]byte, error) {
	// ROADMAP: Excel export implementation.
	return []byte{}, nil
}

func (s *reportService) exportToPDF(_ any, _ dto.ExportOptionsDTO) ([]byte, error) {
	// ROADMAP: PDF export implementation.
	return []byte{}, nil
}

// Hidden API stubs for advanced analytics endpoints.

func (s *reportService) GenerateTrendAnalysis(
	_ context.Context,
	_ *uuid.UUID,
	_ report.Period,
) (*dto.TrendAnalysisDTO, error) {
	// HIDDEN_API_STUB: trend analysis service entrypoint.
	return nil, reportFeatureHiddenStubError("trend analysis")
}

func (s *reportService) GenerateFinancialInsights(_ context.Context) ([]dto.RecommendationDTO, error) {
	// HIDDEN_API_STUB: financial insights service entrypoint.
	return nil, reportFeatureHiddenStubError("financial insights")
}

// generateExpenseSpecificData generates expense-specific report components
func (s *reportService) generateExpenseSpecificData(
	ctx context.Context,
	req dto.ReportRequestDTO,
	baseData *transactionReportData,
	result *completeTransactionReportData,
) error {
	result.dailyBreakdownExpense = s.generateDailyExpenseBreakdown(baseData.transactions)

	var err error
	result.expenseTrends, err = s.generateExpenseTrends(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return fmt.Errorf("failed to generate expense trends: %w", err)
	}

	result.expenseComparisons, err = s.generateExpenseComparisons(
		ctx,
		baseData.totalAmount,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to generate expense comparisons: %w", err)
	}

	return nil
}

// generateIncomeSpecificData generates income-specific report components
func (s *reportService) generateIncomeSpecificData(
	ctx context.Context,
	req dto.ReportRequestDTO,
	baseData *transactionReportData,
	result *completeTransactionReportData,
) error {
	result.dailyBreakdownIncome = s.generateDailyIncomeBreakdown(baseData.transactions)

	var err error
	result.incomeTrends, err = s.generateIncomeTrends(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return fmt.Errorf("failed to generate income trends: %w", err)
	}

	result.incomeComparisons, err = s.generateIncomeComparisons(
		ctx,
		baseData.totalAmount,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		return fmt.Errorf("failed to generate income comparisons: %w", err)
	}

	return nil
}
