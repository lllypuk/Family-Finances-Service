package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/report"
)

func TestReportRequestDTO_AllFields(t *testing.T) {
	userID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	req := ReportRequestDTO{
		Name:      "Monthly Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: startDate,
		EndDate:   endDate,
		Filters: &ReportFilters{
			CategoryIDs: []uuid.UUID{uuid.New()},
			MinAmount:   new(10.0),
			MaxAmount:   new(1000.0),
		},
	}

	assert.Equal(t, "Monthly Expense Report", req.Name)
	assert.Equal(t, report.TypeExpenses, req.Type)
	assert.Equal(t, report.PeriodMonthly, req.Period)
	assert.NotNil(t, req.Filters)
}

func TestReportFilters_AllFields(t *testing.T) {
	categoryID1 := uuid.New()
	categoryID2 := uuid.New()
	userID := uuid.New()
	minAmount := 10.0
	maxAmount := 1000.0

	filters := ReportFilters{
		CategoryIDs:    []uuid.UUID{categoryID1, categoryID2},
		UserIDs:        []uuid.UUID{userID},
		MinAmount:      &minAmount,
		MaxAmount:      &maxAmount,
		Description:    "groceries",
		IncludeSubcats: true,
	}

	assert.Len(t, filters.CategoryIDs, 2)
	assert.Len(t, filters.UserIDs, 1)
	assert.NotNil(t, filters.MinAmount)
	assert.Equal(t, 10.0, *filters.MinAmount)
	assert.True(t, filters.IncludeSubcats)
}

func TestExpenseReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()

	reportDTO := ExpenseReportDTO{
		ID:            reportID,
		Name:          "January Expenses",
		UserID:        userID,
		Period:        "monthly",
		StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		TotalExpenses: 5000.00,
		AverageDaily:  161.29,
		GeneratedAt:   now,
	}

	assert.Equal(t, reportID, reportDTO.ID)
	assert.Equal(t, "January Expenses", reportDTO.Name)
	assert.Equal(t, 5000.00, reportDTO.TotalExpenses)
	assert.Equal(t, 161.29, reportDTO.AverageDaily)
}

func TestIncomeReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()

	reportDTO := IncomeReportDTO{
		ID:           reportID,
		Name:         "January Income",
		UserID:       userID,
		Period:       "monthly",
		StartDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		TotalIncome:  8000.00,
		AverageDaily: 258.06,
		GeneratedAt:  now,
	}

	assert.Equal(t, reportID, reportDTO.ID)
	assert.Equal(t, "January Income", reportDTO.Name)
	assert.Equal(t, 8000.00, reportDTO.TotalIncome)
}

func TestBudgetComparisonDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()

	comparison := BudgetComparisonDTO{
		ID:            reportID,
		Name:          "Budget vs Actual",
		UserID:        userID,
		Period:        "monthly",
		StartDate:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		TotalBudget:   10000.00,
		TotalSpent:    8500.00,
		TotalVariance: 1500.00,
		Utilization:   85.0,
		GeneratedAt:   now,
	}

	assert.Equal(t, reportID, comparison.ID)
	assert.Equal(t, 10000.00, comparison.TotalBudget)
	assert.Equal(t, 8500.00, comparison.TotalSpent)
	assert.Equal(t, 85.0, comparison.Utilization)
}

func TestCashFlowReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()

	cashFlow := CashFlowReportDTO{
		ID:             reportID,
		Name:           "January Cash Flow",
		UserID:         userID,
		Period:         "monthly",
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		OpeningBalance: 5000.00,
		ClosingBalance: 8000.00,
		NetCashFlow:    3000.00,
		TotalInflows:   10000.00,
		TotalOutflows:  7000.00,
		GeneratedAt:    now,
	}

	assert.Equal(t, reportID, cashFlow.ID)
	assert.Equal(t, 5000.00, cashFlow.OpeningBalance)
	assert.Equal(t, 8000.00, cashFlow.ClosingBalance)
	assert.Equal(t, 3000.00, cashFlow.NetCashFlow)
}

func TestCategoryBreakdownDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()

	breakdown := CategoryBreakdownDTO{
		ID:          reportID,
		Name:        "Category Analysis",
		UserID:      userID,
		Period:      "monthly",
		StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		GeneratedAt: now,
	}

	assert.Equal(t, reportID, breakdown.ID)
	assert.Equal(t, "Category Analysis", breakdown.Name)
}

func TestCategoryBreakdownItemDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	parentID := uuid.New()

	item := CategoryBreakdownItemDTO{
		CategoryID:    categoryID,
		CategoryName:  "Food",
		CategoryType:  "expense",
		Amount:        1500.00,
		Percentage:    30.0,
		Count:         45,
		AverageAmount: 33.33,
		ParentID:      &parentID,
	}

	assert.Equal(t, categoryID, item.CategoryID)
	assert.Equal(t, "Food", item.CategoryName)
	assert.Equal(t, 1500.00, item.Amount)
	assert.Equal(t, 30.0, item.Percentage)
	assert.Equal(t, 45, item.Count)
	assert.NotNil(t, item.ParentID)
}

func TestDailyExpenseDTO_AllFields(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	daily := DailyExpenseDTO{
		Date:       date,
		Amount:     150.00,
		Count:      5,
		Categories: []string{"Food", "Transport"},
	}

	assert.Equal(t, date, daily.Date)
	assert.Equal(t, 150.00, daily.Amount)
	assert.Equal(t, 5, daily.Count)
	assert.Len(t, daily.Categories, 2)
}

func TestDailyIncomeDTO_AllFields(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	daily := DailyIncomeDTO{
		Date:    date,
		Amount:  2500.00,
		Count:   2,
		Sources: []string{"Salary", "Freelance"},
	}

	assert.Equal(t, date, daily.Date)
	assert.Equal(t, 2500.00, daily.Amount)
	assert.Len(t, daily.Sources, 2)
}

func TestDailyCashFlowDTO_AllFields(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	daily := DailyCashFlowDTO{
		Date:    date,
		Inflow:  2500.00,
		Outflow: 150.00,
		NetFlow: 2350.00,
		Balance: 10000.00,
	}

	assert.Equal(t, date, daily.Date)
	assert.Equal(t, 2500.00, daily.Inflow)
	assert.Equal(t, 150.00, daily.Outflow)
	assert.Equal(t, 2350.00, daily.NetFlow)
	assert.Equal(t, 10000.00, daily.Balance)
}

func TestWeeklyCashFlowDTO_AllFields(t *testing.T) {
	weekStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	weekEnd := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)

	weekly := WeeklyCashFlowDTO{
		WeekStart: weekStart,
		WeekEnd:   weekEnd,
		Inflow:    5000.00,
		Outflow:   2000.00,
		NetFlow:   3000.00,
	}

	assert.Equal(t, weekStart, weekly.WeekStart)
	assert.Equal(t, weekEnd, weekly.WeekEnd)
	assert.Equal(t, 3000.00, weekly.NetFlow)
}

func TestMonthlyCashFlowDTO_AllFields(t *testing.T) {
	month := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	monthly := MonthlyCashFlowDTO{
		Month:   month,
		Inflow:  20000.00,
		Outflow: 15000.00,
		NetFlow: 5000.00,
	}

	assert.Equal(t, month, monthly.Month)
	assert.Equal(t, 20000.00, monthly.Inflow)
	assert.Equal(t, 5000.00, monthly.NetFlow)
}

func TestTransactionSummaryDTO_AllFields(t *testing.T) {
	txID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	summary := TransactionSummaryDTO{
		ID:          txID,
		Amount:      150.00,
		Description: "Groceries",
		Category:    "Food",
		Date:        date,
		UserName:    "John Doe",
	}

	assert.Equal(t, txID, summary.ID)
	assert.Equal(t, 150.00, summary.Amount)
	assert.Equal(t, "Groceries", summary.Description)
	assert.Equal(t, "John Doe", summary.UserName)
}

func TestBudgetCategoryComparisonDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()

	comparison := BudgetCategoryComparisonDTO{
		CategoryID:   categoryID,
		CategoryName: "Food",
		BudgetAmount: 1000.00,
		ActualAmount: 850.00,
		Variance:     150.00,
		Utilization:  85.0,
		Status:       "on_track",
	}

	assert.Equal(t, categoryID, comparison.CategoryID)
	assert.Equal(t, "Food", comparison.CategoryName)
	assert.Equal(t, 1000.00, comparison.BudgetAmount)
	assert.Equal(t, 850.00, comparison.ActualAmount)
	assert.Equal(t, "on_track", comparison.Status)
}

func TestBudgetTimelineDTO_AllFields(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	timeline := BudgetTimelineDTO{
		Date:         date,
		PlannedSpent: 500.00,
		ActualSpent:  450.00,
		Variance:     50.00,
	}

	assert.Equal(t, date, timeline.Date)
	assert.Equal(t, 500.00, timeline.PlannedSpent)
	assert.Equal(t, 450.00, timeline.ActualSpent)
	assert.Equal(t, 50.00, timeline.Variance)
}

func TestBudgetAlertReportDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()

	alert := BudgetAlertReportDTO{
		Type:       "warning",
		CategoryID: categoryID,
		Category:   "Food",
		Message:    "Budget approaching limit",
		Threshold:  80.0,
		Current:    85.0,
		Severity:   7,
	}

	assert.Equal(t, "warning", alert.Type)
	assert.Equal(t, categoryID, alert.CategoryID)
	assert.Equal(t, "Food", alert.Category)
	assert.Equal(t, 7, alert.Severity)
}

func TestCategoryAnalysisDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()

	analysis := CategoryAnalysisDTO{
		CategoryID:       categoryID,
		CategoryName:     "Food",
		CategoryType:     "expense",
		TotalAmount:      1500.00,
		Percentage:       30.0,
		TransactionCount: 45,
		AverageAmount:    33.33,
		MinAmount:        5.00,
		MaxAmount:        200.00,
		Trend:            "increasing",
		TrendPercentage:  10.0,
	}

	assert.Equal(t, categoryID, analysis.CategoryID)
	assert.Equal(t, "Food", analysis.CategoryName)
	assert.Equal(t, 1500.00, analysis.TotalAmount)
	assert.Equal(t, 45, analysis.TransactionCount)
	assert.Equal(t, "increasing", analysis.Trend)
}

func TestTrendAnalysisDTO_AllFields(t *testing.T) {
	trend := TrendAnalysisDTO{
		Direction:   "increasing",
		Percentage:  15.5,
		Confidence:  0.85,
		Description: "Spending is increasing by 15.5%",
	}

	assert.Equal(t, "increasing", trend.Direction)
	assert.Equal(t, 15.5, trend.Percentage)
	assert.Equal(t, 0.85, trend.Confidence)
}

func TestCategoryTrendDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()

	categoryTrend := CategoryTrendDTO{
		CategoryID:   categoryID,
		CategoryName: "Food",
		Trend: TrendAnalysisDTO{
			Direction:  "increasing",
			Percentage: 10.0,
			Confidence: 0.8,
		},
		CurrentAmount:  1500.00,
		PreviousAmount: 1364.00,
	}

	assert.Equal(t, categoryID, categoryTrend.CategoryID)
	assert.Equal(t, "Food", categoryTrend.CategoryName)
	assert.Equal(t, 1500.00, categoryTrend.CurrentAmount)
}

func TestSeasonalPatternDTO_AllFields(t *testing.T) {
	pattern := SeasonalPatternDTO{
		Season:      "winter",
		Amount:      5000.00,
		Percentage:  28.0,
		Description: "Higher spending in winter",
	}

	assert.Equal(t, "winter", pattern.Season)
	assert.Equal(t, 5000.00, pattern.Amount)
	assert.Equal(t, 28.0, pattern.Percentage)
}

func TestWeekdayPatternDTO_AllFields(t *testing.T) {
	pattern := WeekdayPatternDTO{
		Weekday:    "Saturday",
		Amount:     800.00,
		Percentage: 20.0,
		Count:      25,
	}

	assert.Equal(t, "Saturday", pattern.Weekday)
	assert.Equal(t, 800.00, pattern.Amount)
	assert.Equal(t, 25, pattern.Count)
}

func TestForecastDTO_AllFields(t *testing.T) {
	date := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	forecast := ForecastDTO{
		Date:       date,
		Amount:     1500.00,
		Confidence: 0.85,
		Lower:      1200.00,
		Upper:      1800.00,
	}

	assert.Equal(t, date, forecast.Date)
	assert.Equal(t, 1500.00, forecast.Amount)
	assert.Equal(t, 0.85, forecast.Confidence)
	assert.Equal(t, 1200.00, forecast.Lower)
	assert.Equal(t, 1800.00, forecast.Upper)
}

func TestPeriodComparisonDTO_AllFields(t *testing.T) {
	comparison := PeriodComparisonDTO{
		CurrentAmount:    5000.00,
		PreviousAmount:   4500.00,
		Difference:       500.00,
		PercentageChange: 11.11,
		Description:      "Spending increased by 11.11%",
	}

	assert.Equal(t, 5000.00, comparison.CurrentAmount)
	assert.Equal(t, 4500.00, comparison.PreviousAmount)
	assert.Equal(t, 500.00, comparison.Difference)
	assert.Equal(t, 11.11, comparison.PercentageChange)
}

func TestBenchmarkComparisonDTO_AllFields(t *testing.T) {
	comparison := BenchmarkComparisonDTO{
		UserAmount:       5000.00,
		BenchmarkAmount:  4500.00,
		Difference:       500.00,
		PercentageChange: 11.11,
		Status:           "above",
		Description:      "Above family average",
	}

	assert.Equal(t, 5000.00, comparison.UserAmount)
	assert.Equal(t, "above", comparison.Status)
}

func TestProjectionDTO_AllFields(t *testing.T) {
	startDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)

	projection := ProjectionDTO{
		Period:           "next_month",
		StartDate:        startDate,
		EndDate:          endDate,
		ProjectedInflow:  8000.00,
		ProjectedOutflow: 6000.00,
		ProjectedBalance: 12000.00,
		Confidence:       0.80,
		Assumptions:      []string{"Stable income", "Normal spending"},
	}

	assert.Equal(t, "next_month", projection.Period)
	assert.Equal(t, 8000.00, projection.ProjectedInflow)
	assert.Equal(t, 0.80, projection.Confidence)
	assert.Len(t, projection.Assumptions, 2)
}

func TestScenarioDTO_AllFields(t *testing.T) {
	scenario := ScenarioDTO{
		Name:            "Emergency Expense",
		Description:     "Unexpected car repair",
		Probability:     0.15,
		Impact:          "negative",
		ProjectedChange: -1500.00,
	}

	assert.Equal(t, "Emergency Expense", scenario.Name)
	assert.Equal(t, 0.15, scenario.Probability)
	assert.Equal(t, "negative", scenario.Impact)
	assert.Equal(t, -1500.00, scenario.ProjectedChange)
}

func TestRecommendationDTO_AllFields(t *testing.T) {
	recommendation := RecommendationDTO{
		Type:        "saving",
		Priority:    "high",
		Title:       "Reduce dining out",
		Description: "You can save $200/month by reducing dining out",
		Impact:      200.00,
		Effort:      "easy",
	}

	assert.Equal(t, "saving", recommendation.Type)
	assert.Equal(t, "high", recommendation.Priority)
	assert.Equal(t, 200.00, recommendation.Impact)
	assert.Equal(t, "easy", recommendation.Effort)
}

func TestExportRequestDTO_AllFields(t *testing.T) {
	reportID := uuid.New()

	export := ExportRequestDTO{
		ReportID: reportID,
		Format:   "pdf",
		Options: ExportOptionsDTO{
			IncludeCharts:  true,
			IncludeDetails: true,
			Sections:       []string{"summary", "details"},
			Language:       "en",
			Currency:       "USD",
			DateFormat:     "2006-01-02",
		},
	}

	assert.Equal(t, reportID, export.ReportID)
	assert.Equal(t, "pdf", export.Format)
	assert.True(t, export.Options.IncludeCharts)
	assert.Len(t, export.Options.Sections, 2)
}

func TestScheduleReportDTO_AllFields(t *testing.T) {
	userID := uuid.New()

	schedule := ScheduleReportDTO{
		Name:   "Monthly Report",
		Type:   report.TypeExpenses,
		UserID: userID,
		Schedule: ScheduleConfigDTO{
			Frequency:  "monthly",
			DayOfMonth: new(1),
			Time:       "09:00",
			Timezone:   "UTC",
		},
		ExportFormat: "pdf",
		Recipients:   []string{"admin@example.com"},
		Active:       true,
	}

	assert.Equal(t, "Monthly Report", schedule.Name)
	assert.Equal(t, report.TypeExpenses, schedule.Type)
	assert.Equal(t, "pdf", schedule.ExportFormat)
	assert.Len(t, schedule.Recipients, 1)
	assert.True(t, schedule.Active)
}

func TestScheduledReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()
	userID := uuid.New()
	lastRun := now.Add(-24 * time.Hour)
	nextRun := now.Add(24 * time.Hour)

	scheduled := ScheduledReportDTO{
		ID:     reportID,
		Name:   "Monthly Report",
		Type:   report.TypeExpenses,
		UserID: userID,
		Schedule: ScheduleConfigDTO{
			Frequency: "monthly",
			Time:      "09:00",
			Timezone:  "UTC",
		},
		ExportFormat: "pdf",
		Recipients:   []string{"admin@example.com"},
		Active:       true,
		LastRun:      &lastRun,
		NextRun:      nextRun,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, reportID, scheduled.ID)
	assert.NotNil(t, scheduled.LastRun)
	assert.Equal(t, nextRun, scheduled.NextRun)
	assert.True(t, scheduled.Active)
}
