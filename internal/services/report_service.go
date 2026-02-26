package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
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
	reportTransactionQueryLimit = 1000 // Maximum transactions to query for reports
)

var ErrReportFeatureHiddenFromPublicAPI = errors.New("report feature hidden from public API until implemented")

type reportService struct {
	reportRepo      ReportRepository
	transactionRepo TransactionRepository
	budgetRepo      BudgetRepository
	categoryRepo    CategoryRepository
	userRepo        UserRepository

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
	totalAmount       float64
	averageDaily      float64
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
		ID:                uuid.New(),
		Name:              req.Name,
		UserID:            req.UserID,
		Period:            req.Period,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		TotalExpenses:     reportData.totalAmount,
		AverageDaily:      reportData.averageDaily,
		CategoryBreakdown: reportData.categoryBreakdown,
		DailyBreakdown:    reportData.dailyBreakdownExpense,
		TopExpenses:       reportData.topTransactions,
		Trends:            reportData.expenseTrends,
		Comparisons:       reportData.expenseComparisons,
		GeneratedAt:       time.Now(),
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
		TotalIncome:       reportData.totalAmount,
		AverageDaily:      reportData.averageDaily,
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
	// Calculate date range based on period
	startDate, endDate := s.calculatePeriodDates(period)

	// Get active budgets for the period
	budgets, err := s.budgetService.GetActiveBudgets(ctx, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	if len(budgets) == 0 {
		return &dto.BudgetComparisonDTO{
			ID:            uuid.New(),
			Name:          fmt.Sprintf("Budget Comparison - %s", period),
			Period:        period,
			StartDate:     startDate,
			EndDate:       endDate,
			TotalBudget:   0,
			TotalSpent:    0,
			TotalVariance: 0,
			Utilization:   0,
			Categories:    []dto.BudgetCategoryComparisonDTO{},
			Timeline:      []dto.BudgetTimelineDTO{},
			Alerts:        []dto.BudgetAlertReportDTO{},
			GeneratedAt:   time.Now(),
		}, nil
	}

	// Calculate totals
	totalBudget := 0.0
	for _, b := range budgets {
		totalBudget += b.Amount
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
	utilization := 0.0
	if totalBudget > 0 {
		utilization = (totalSpent / totalBudget) * percentageMultiplier
	}

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
		ID:            uuid.New(),
		Name:          fmt.Sprintf("Budget Comparison - %s", period),
		Period:        period,
		StartDate:     startDate,
		EndDate:       endDate,
		TotalBudget:   totalBudget,
		TotalSpent:    totalSpent,
		TotalVariance: totalVariance,
		Utilization:   utilization,
		Categories:    categoryComparisons,
		Timeline:      timeline,
		Alerts:        alerts,
		GeneratedAt:   time.Now(),
	}, nil
}

// GenerateCashFlowReport generates cash flow analysis report
func (s *reportService) GenerateCashFlowReport(
	ctx context.Context,
	from, to time.Time,
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
	openingBalance := 0.0
	closingBalance := openingBalance + netCashFlow

	// Generate daily cash flow
	dailyFlow := s.generateDailyCashFlow(allTransactions, openingBalance, from, to)

	// Generate weekly and monthly aggregations
	weeklyFlow := s.generateWeeklyCashFlow(dailyFlow)
	monthlyFlow := s.generateMonthlyCashFlow(dailyFlow)

	// Generate projections
	projections, err := s.generateCashFlowProjections(ctx, allTransactions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate projections: %w", err)
	}

	return &dto.CashFlowReportDTO{
		ID:             uuid.New(),
		Name:           fmt.Sprintf("Cash Flow Report - %s to %s", from.Format("2006-01-02"), to.Format("2006-01-02")),
		Period:         report.PeriodCustom,
		StartDate:      from,
		EndDate:        to,
		OpeningBalance: openingBalance,
		ClosingBalance: closingBalance,
		NetCashFlow:    netCashFlow,
		TotalInflows:   totalInflows,
		TotalOutflows:  totalOutflows,
		DailyFlow:      dailyFlow,
		WeeklyFlow:     weeklyFlow,
		MonthlyFlow:    monthlyFlow,
		Projections:    projections,
		GeneratedAt:    time.Now(),
	}, nil
}

// GenerateCategoryBreakdownReport generates detailed category analysis
func (s *reportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	startDate, endDate := s.calculatePeriodDates(period)

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

// SaveReport saves a generated report to the database
func (s *reportService) SaveReport(
	ctx context.Context,
	reportData any,
	reportType report.Type,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	// Convert reportData to report.Data format
	data, err := s.convertToReportData(reportData, reportType)
	if err != nil {
		return nil, fmt.Errorf("failed to convert report data: %w", err)
	}

	newReport := report.NewReport(
		req.Name,
		reportType,
		req.Period,
		req.UserID,
		req.StartDate,
		req.EndDate,
	)
	newReport.Data = data

	if createErr := s.reportRepo.Create(ctx, newReport); createErr != nil {
		return nil, fmt.Errorf("failed to save report: %w", createErr)
	}

	return newReport, nil
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
	case "csv":
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
	startDate, endDate time.Time,
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
		filter.AmountFrom = filters.MinAmount
		filter.AmountTo = filters.MaxAmount
		if filters.Description != "" {
			filter.Description = &filters.Description
		}
	}

	return s.transactionService.GetAllTransactions(ctx, filter)
}

func (s *reportService) calculateTotalAmount(transactions []*transaction.Transaction) float64 {
	total := 0.0
	for _, t := range transactions {
		total += t.Amount
	}
	return total
}

func (s *reportService) calculateAverageDaily(total float64, startDate, endDate time.Time) float64 {
	days := endDate.Sub(startDate).Hours() / hoursPerDay
	if days <= 0 {
		return 0
	}
	return total / days
}

func (s *reportService) generateCategoryBreakdown(
	ctx context.Context,
	transactions []*transaction.Transaction,
) []dto.CategoryBreakdownItemDTO {
	// Group transactions by category
	categoryTotals := make(map[uuid.UUID]float64)
	categoryCounts := make(map[uuid.UUID]int)

	total := 0.0
	for _, t := range transactions {
		if t.CategoryID != uuid.Nil {
			categoryTotals[t.CategoryID] += t.Amount
			categoryCounts[t.CategoryID]++
			total += t.Amount
		}
	}

	// Get category details
	var result []dto.CategoryBreakdownItemDTO
	for categoryID, amount := range categoryTotals {
		cat, err := s.categoryService.GetCategoryByID(ctx, categoryID)
		if err != nil {
			continue // Skip if category not found
		}

		percentage := 0.0
		if total > 0 {
			percentage = (amount / total) * percentageMultiplier
		}

		avgAmount := 0.0
		if count := categoryCounts[categoryID]; count > 0 {
			avgAmount = amount / float64(count)
		}

		result = append(result, dto.CategoryBreakdownItemDTO{
			CategoryID:    categoryID,
			CategoryName:  cat.Name,
			CategoryType:  string(cat.Type),
			Amount:        amount,
			Percentage:    percentage,
			Count:         categoryCounts[categoryID],
			AverageAmount: avgAmount,
			ParentID:      cat.ParentID,
		})
	}

	// Sort by amount descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Amount > result[j].Amount
	})

	return result
}

func (s *reportService) generateDailyExpenseBreakdown(transactions []*transaction.Transaction) []dto.DailyExpenseDTO {
	dailyMap := make(map[string]struct {
		amount     float64
		count      int
		categories map[string]int
	})

	for _, t := range transactions {
		day := t.Date.Format("2006-01-02")
		if _, exists := dailyMap[day]; !exists {
			dailyMap[day] = struct {
				amount     float64
				count      int
				categories map[string]int
			}{
				amount:     0,
				count:      0,
				categories: make(map[string]int),
			}
		}

		entry := dailyMap[day]
		entry.amount += t.Amount
		entry.count++
		// Note: Would need category name lookup for categories
		dailyMap[day] = entry
	}

	var result []dto.DailyExpenseDTO
	for dateStr, data := range dailyMap {
		date, _ := time.Parse("2006-01-02", dateStr)
		result = append(result, dto.DailyExpenseDTO{
			Date:   date,
			Amount: data.amount,
			Count:  data.count,
			// Categories would be populated with actual category names
		})
	}

	// Sort by date
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result
}

func (s *reportService) generateDailyIncomeBreakdown(transactions []*transaction.Transaction) []dto.DailyIncomeDTO {
	dailyMap := make(map[string]struct {
		amount  float64
		count   int
		sources map[string]int
	})

	for _, t := range transactions {
		day := t.Date.Format("2006-01-02")
		if _, exists := dailyMap[day]; !exists {
			dailyMap[day] = struct {
				amount  float64
				count   int
				sources map[string]int
			}{
				amount:  0,
				count:   0,
				sources: make(map[string]int),
			}
		}

		entry := dailyMap[day]
		entry.amount += t.Amount
		entry.count++
		dailyMap[day] = entry
	}

	var result []dto.DailyIncomeDTO
	for dateStr, data := range dailyMap {
		date, _ := time.Parse("2006-01-02", dateStr)
		result = append(result, dto.DailyIncomeDTO{
			Date:    date,
			Amount:  data.amount,
			Count:   data.count,
			Sources: []string{}, // Would be populated with actual source names
		})
	}

	// Sort by date
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
		return transactions[i].Amount > transactions[j].Amount
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
			Amount:      t.Amount,
			Description: t.Description,
			Category:    categoryName,
			Date:        t.Date,
			UserName:    userName,
		})
	}

	return result
}

func (s *reportService) calculatePeriodDates(period report.Period) (time.Time, time.Time) {
	now := time.Now()
	switch period {
	case report.PeriodDaily:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 0, 1)
	case report.PeriodWeekly:
		weekday := int(now.Weekday())
		start := now.AddDate(0, 0, -weekday).Truncate(hoursPerDay * time.Hour)
		return start, start.AddDate(0, 0, daysPerWeek)
	case report.PeriodMonthly:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 1, 0)
	case report.PeriodYearly:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(1, 0, 0)
	case report.PeriodCustom:
		// For custom periods, return current month as default
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 1, 0)
	default:
		// Default to current month
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(0, 1, 0)
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
	_, _ time.Time,
) (dto.ExpenseTrendsDTO, error) {
	// ROADMAP: implement sophisticated expense trend analysis.
	return dto.ExpenseTrendsDTO{}, nil
}

func (s *reportService) generateExpenseComparisons(
	_ context.Context,
	_ float64,
	_, _ time.Time,
) (dto.ExpenseComparisonsDTO, error) {
	// ROADMAP: compare expenses with previous periods.
	return dto.ExpenseComparisonsDTO{}, nil
}

func (s *reportService) generateIncomeTrends(
	_ context.Context,
	_, _ time.Time,
) (dto.IncomeTrendsDTO, error) {
	// ROADMAP: implement income trend analysis.
	return dto.IncomeTrendsDTO{}, nil
}

func (s *reportService) generateIncomeComparisons(
	_ context.Context,
	_ float64,
	_, _ time.Time,
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
	_ float64,
	_, _ time.Time,
) []dto.BudgetTimelineDTO {
	// ROADMAP: budget timeline generation.
	return []dto.BudgetTimelineDTO{}
}

func (s *reportService) generateBudgetAlerts(_ []dto.BudgetCategoryComparisonDTO) []dto.BudgetAlertReportDTO {
	// ROADMAP: budget alert generation.
	return []dto.BudgetAlertReportDTO{}
}

func (s *reportService) generateDailyCashFlow(
	_ []*transaction.Transaction,
	_ float64,
	_, _ time.Time,
) []dto.DailyCashFlowDTO {
	// ROADMAP: daily cash-flow calculation with running balance.
	return []dto.DailyCashFlowDTO{}
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
	_, _ time.Time,
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
	_, _ time.Time,
) (dto.CategoryTrendsDTO, error) {
	// ROADMAP: category trend analysis.
	return dto.CategoryTrendsDTO{}, nil
}

func (s *reportService) generateCategoryComparisons(
	_ context.Context,
	_ []dto.CategoryAnalysisDTO,
	_, _ time.Time,
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
			TotalExpenses:     expenseReport.TotalExpenses,
			CategoryBreakdown: convertCategoryBreakdownItemsToReportData(expenseReport.CategoryBreakdown),
			TopExpenses:       convertTransactionSummaryItemsToReportData(expenseReport.TopExpenses),
		}, nil

	case report.TypeIncome:
		incomeReport, ok := reportData.(*dto.IncomeReportDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.IncomeReportDTO, got %T", reportData)
		}
		// Persist top sources in TopExpenses generic field for unified rendering/storage.
		return report.Data{
			TotalIncome:       incomeReport.TotalIncome,
			CategoryBreakdown: convertCategoryBreakdownItemsToReportData(incomeReport.CategoryBreakdown),
			TopExpenses:       convertTransactionSummaryItemsToReportData(incomeReport.TopSources),
		}, nil

	case report.TypeBudget:
		budgetReport, ok := reportData.(*dto.BudgetComparisonDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.BudgetComparisonDTO, got %T", reportData)
		}
		return report.Data{
			TotalExpenses:    budgetReport.TotalSpent,
			BudgetComparison: convertBudgetComparisonItemsToReportData(budgetReport.Categories),
		}, nil

	case report.TypeCashFlow:
		cashFlowReport, ok := reportData.(*dto.CashFlowReportDTO)
		if !ok {
			return report.Data{}, fmt.Errorf("expected *dto.CashFlowReportDTO, got %T", reportData)
		}
		return report.Data{
			TotalIncome:    cashFlowReport.TotalInflows,
			TotalExpenses:  cashFlowReport.TotalOutflows,
			NetIncome:      cashFlowReport.NetCashFlow,
			DailyBreakdown: convertDailyCashFlowItemsToReportData(cashFlowReport.DailyFlow),
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
			Amount:       item.Amount,
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
			Amount:       item.TotalAmount,
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
			Amount:      item.Amount,
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
			BudgetID:   item.CategoryID, // Generic report.Data has no category-specific budget key.
			BudgetName: item.CategoryName,
			Planned:    item.BudgetAmount,
			Actual:     item.ActualAmount,
			Difference: item.Variance,
			Percentage: item.Utilization,
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
			Date:     item.Date,
			Income:   item.Inflow,
			Expenses: item.Outflow,
			Balance:  item.Balance,
		}
	}
	return result
}

func (s *reportService) exportToCSV(_ any, _ dto.ExportOptionsDTO) ([]byte, error) {
	// ROADMAP: CSV export implementation.
	return []byte{}, nil
}

func (s *reportService) exportToExcel(_ any, _ dto.ExportOptionsDTO) ([]byte, error) {
	// ROADMAP: Excel export implementation.
	return []byte{}, nil
}

func (s *reportService) exportToPDF(_ any, _ dto.ExportOptionsDTO) ([]byte, error) {
	// ROADMAP: PDF export implementation.
	return []byte{}, nil
}

// Hidden API stubs: methods remain on ReportService for interface compatibility,
// but callers must not expose them via public API until implemented.

func (s *reportService) ScheduleReport(_ context.Context, _ dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error) {
	// HIDDEN_API_STUB: scheduled report creation.
	return nil, reportFeatureHiddenStubError("scheduled report creation")
}

func (s *reportService) GetScheduledReports(_ context.Context) ([]*dto.ScheduledReportDTO, error) {
	// HIDDEN_API_STUB: scheduled report retrieval.
	return nil, reportFeatureHiddenStubError("scheduled report retrieval")
}

func (s *reportService) UpdateScheduledReport(
	_ context.Context,
	_ uuid.UUID,
	_ dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	// HIDDEN_API_STUB: scheduled report update.
	return nil, reportFeatureHiddenStubError("scheduled report update")
}

func (s *reportService) DeleteScheduledReport(_ context.Context, _ uuid.UUID) error {
	// HIDDEN_API_STUB: scheduled report deletion.
	return reportFeatureHiddenStubError("scheduled report deletion")
}

func (s *reportService) ExecuteScheduledReport(_ context.Context, _ uuid.UUID) error {
	// HIDDEN_API_STUB: scheduled report execution.
	return reportFeatureHiddenStubError("scheduled report execution")
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

func (s *reportService) GenerateSpendingForecast(_ context.Context, _ int) ([]dto.ForecastDTO, error) {
	// HIDDEN_API_STUB: spending forecast service entrypoint.
	return nil, reportFeatureHiddenStubError("spending forecast")
}

func (s *reportService) GenerateFinancialInsights(_ context.Context) ([]dto.RecommendationDTO, error) {
	// HIDDEN_API_STUB: financial insights service entrypoint.
	return nil, reportFeatureHiddenStubError("financial insights")
}

func (s *reportService) CalculateBenchmarks(_ context.Context) (*dto.BenchmarkComparisonDTO, error) {
	// HIDDEN_API_STUB: benchmarks service entrypoint.
	return nil, reportFeatureHiddenStubError("benchmark calculations")
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
