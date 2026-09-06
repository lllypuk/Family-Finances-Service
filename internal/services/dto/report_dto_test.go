package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
)

func TestReportRequestDTO_AllFields(t *testing.T) {
	userID := uuid.New()

	req := ReportRequestDTO{
		Name:      "Monthly Expense Report",
		Type:      report.TypeExpenses,
		Period:    report.PeriodMonthly,
		UserID:    userID,
		StartDate: date.New(2024, time.January, 1),
		EndDate:   date.New(2024, time.January, 31),
		Filters: &ReportFilters{
			CategoryIDs:    []uuid.UUID{uuid.New()},
			MinAmountMinor: new(money.Minor(1_000)),
			MaxAmountMinor: new(money.Minor(100_000)),
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
	minAmount := money.Minor(1_000)
	maxAmount := money.Minor(100_000)

	filters := ReportFilters{
		CategoryIDs:    []uuid.UUID{categoryID1, categoryID2},
		UserIDs:        []uuid.UUID{userID},
		MinAmountMinor: &minAmount,
		MaxAmountMinor: &maxAmount,
		Description:    "groceries",
		IncludeSubcats: true,
	}

	assert.Len(t, filters.CategoryIDs, 2)
	assert.Len(t, filters.UserIDs, 1)
	assert.NotNil(t, filters.MinAmountMinor)
	assert.Equal(t, money.Minor(1_000), *filters.MinAmountMinor)
	assert.True(t, filters.IncludeSubcats)
}

func TestExpenseReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()

	reportDTO := ExpenseReportDTO{
		ID:                 reportID,
		Name:               "January Expenses",
		UserID:             uuid.New(),
		Period:             "monthly",
		StartDate:          date.New(2024, time.January, 1),
		EndDate:            date.New(2024, time.January, 31),
		TotalExpensesMinor: 500_000,
		AverageDailyMinor:  16_129,
		GeneratedAt:        now,
	}

	assert.Equal(t, reportID, reportDTO.ID)
	assert.Equal(t, "January Expenses", reportDTO.Name)
	assert.Equal(t, money.Minor(500_000), reportDTO.TotalExpensesMinor)
	assert.Equal(t, money.Minor(16_129), reportDTO.AverageDailyMinor)
}

func TestIncomeReportDTO_AllFields(t *testing.T) {
	now := time.Now()
	reportID := uuid.New()

	reportDTO := IncomeReportDTO{
		ID:                reportID,
		Name:              "January Income",
		UserID:            uuid.New(),
		Period:            "monthly",
		StartDate:         date.New(2024, time.January, 1),
		EndDate:           date.New(2024, time.January, 31),
		TotalIncomeMinor:  800_000,
		AverageDailyMinor: 25_806,
		GeneratedAt:       now,
	}

	assert.Equal(t, reportID, reportDTO.ID)
	assert.Equal(t, "January Income", reportDTO.Name)
	assert.Equal(t, money.Minor(800_000), reportDTO.TotalIncomeMinor)
}

func TestBudgetComparisonDTO_AllFields(t *testing.T) {
	reportID := uuid.New()

	comparison := BudgetComparisonDTO{
		ID:                 reportID,
		Name:               "Budget vs Actual",
		UserID:             uuid.New(),
		Period:             "monthly",
		StartDate:          date.New(2024, time.January, 1),
		EndDate:            date.New(2024, time.January, 31),
		TotalBudgetMinor:   1_000_000,
		TotalSpentMinor:    850_000,
		TotalVarianceMinor: 150_000,
		Utilization:        85.0,
		GeneratedAt:        time.Now(),
	}

	assert.Equal(t, reportID, comparison.ID)
	assert.Equal(t, money.Minor(1_000_000), comparison.TotalBudgetMinor)
	assert.Equal(t, money.Minor(850_000), comparison.TotalSpentMinor)
	assert.InDelta(t, 85.0, comparison.Utilization, 0.001)
}

func TestCashFlowReportDTO_AllFields(t *testing.T) {
	reportID := uuid.New()

	cashFlow := CashFlowReportDTO{
		ID:                  reportID,
		Name:                "January Cash Flow",
		UserID:              uuid.New(),
		Period:              "monthly",
		StartDate:           date.New(2024, time.January, 1),
		EndDate:             date.New(2024, time.January, 31),
		OpeningBalanceMinor: 500_000,
		ClosingBalanceMinor: 800_000,
		NetCashFlowMinor:    300_000,
		TotalInflowsMinor:   1_000_000,
		TotalOutflowsMinor:  700_000,
		GeneratedAt:         time.Now(),
	}

	assert.Equal(t, reportID, cashFlow.ID)
	assert.Equal(t, money.Minor(500_000), cashFlow.OpeningBalanceMinor)
	assert.Equal(t, money.Minor(800_000), cashFlow.ClosingBalanceMinor)
	assert.Equal(t, money.Minor(300_000), cashFlow.NetCashFlowMinor)
}

func TestCategoryBreakdownDTO_AllFields(t *testing.T) {
	reportID := uuid.New()

	breakdown := CategoryBreakdownDTO{
		ID:          reportID,
		Name:        "Category Analysis",
		UserID:      uuid.New(),
		Period:      "monthly",
		StartDate:   date.New(2024, time.January, 1),
		EndDate:     date.New(2024, time.January, 31),
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, reportID, breakdown.ID)
	assert.Equal(t, "Category Analysis", breakdown.Name)
}

func TestCategoryBreakdownItemDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	parentID := uuid.New()

	item := CategoryBreakdownItemDTO{
		CategoryID:         categoryID,
		CategoryName:       "Food",
		CategoryType:       "expense",
		AmountMinor:        150_000,
		Percentage:         30.0,
		Count:              45,
		AverageAmountMinor: 3_333,
		ParentID:           &parentID,
	}

	assert.Equal(t, categoryID, item.CategoryID)
	assert.Equal(t, "Food", item.CategoryName)
	assert.Equal(t, money.Minor(150_000), item.AmountMinor)
	assert.InDelta(t, 30.0, item.Percentage, 0.001)
	assert.Equal(t, 45, item.Count)
	assert.NotNil(t, item.ParentID)
}

func TestDailyExpenseDTO_AllFields(t *testing.T) {
	day := date.New(2024, time.January, 15)

	daily := DailyExpenseDTO{
		Date:        day,
		AmountMinor: 15_000,
		Count:       5,
		Categories:  []string{"Food", "Transport"},
	}

	assert.Equal(t, day, daily.Date)
	assert.Equal(t, money.Minor(15_000), daily.AmountMinor)
	assert.Equal(t, 5, daily.Count)
	assert.Len(t, daily.Categories, 2)
}

func TestDailyIncomeDTO_AllFields(t *testing.T) {
	day := date.New(2024, time.January, 15)

	daily := DailyIncomeDTO{
		Date:        day,
		AmountMinor: 250_000,
		Count:       2,
		Sources:     []string{"Salary", "Freelance"},
	}

	assert.Equal(t, day, daily.Date)
	assert.Equal(t, money.Minor(250_000), daily.AmountMinor)
	assert.Len(t, daily.Sources, 2)
}

func TestDailyCashFlowDTO_AllFields(t *testing.T) {
	day := date.New(2024, time.January, 15)

	daily := DailyCashFlowDTO{
		Date:         day,
		InflowMinor:  250_000,
		OutflowMinor: 15_000,
		NetFlowMinor: 235_000,
		BalanceMinor: 1_000_000,
	}

	assert.Equal(t, day, daily.Date)
	assert.Equal(t, money.Minor(250_000), daily.InflowMinor)
	assert.Equal(t, money.Minor(15_000), daily.OutflowMinor)
	assert.Equal(t, money.Minor(235_000), daily.NetFlowMinor)
	assert.Equal(t, money.Minor(1_000_000), daily.BalanceMinor)
}

func TestWeeklyCashFlowDTO_AllFields(t *testing.T) {
	weekStart := date.New(2024, time.January, 1)
	weekEnd := date.New(2024, time.January, 7)

	weekly := WeeklyCashFlowDTO{
		WeekStart:    weekStart,
		WeekEnd:      weekEnd,
		InflowMinor:  500_000,
		OutflowMinor: 200_000,
		NetFlowMinor: 300_000,
	}

	assert.Equal(t, weekStart, weekly.WeekStart)
	assert.Equal(t, weekEnd, weekly.WeekEnd)
	assert.Equal(t, money.Minor(300_000), weekly.NetFlowMinor)
}

func TestMonthlyCashFlowDTO_AllFields(t *testing.T) {
	month := date.New(2024, time.January, 1)

	monthly := MonthlyCashFlowDTO{
		Month:        month,
		InflowMinor:  2_000_000,
		OutflowMinor: 1_500_000,
		NetFlowMinor: 500_000,
	}

	assert.Equal(t, month, monthly.Month)
	assert.Equal(t, money.Minor(2_000_000), monthly.InflowMinor)
	assert.Equal(t, money.Minor(500_000), monthly.NetFlowMinor)
}

func TestTransactionSummaryDTO_AllFields(t *testing.T) {
	txID := uuid.New()
	day := date.New(2024, time.January, 15)

	summary := TransactionSummaryDTO{
		ID:          txID,
		AmountMinor: 15_000,
		Description: "Groceries",
		Category:    "Food",
		Date:        day,
		UserName:    "John Doe",
	}

	assert.Equal(t, txID, summary.ID)
	assert.Equal(t, money.Minor(15_000), summary.AmountMinor)
	assert.Equal(t, "Groceries", summary.Description)
	assert.Equal(t, "John Doe", summary.UserName)
}

func TestBudgetCategoryComparisonDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()

	comparison := BudgetCategoryComparisonDTO{
		CategoryID:        categoryID,
		CategoryName:      "Food",
		BudgetAmountMinor: 100_000,
		ActualAmountMinor: 85_000,
		VarianceMinor:     15_000,
		Utilization:       85.0,
		Status:            "on_track",
	}

	assert.Equal(t, categoryID, comparison.CategoryID)
	assert.Equal(t, "Food", comparison.CategoryName)
	assert.Equal(t, money.Minor(100_000), comparison.BudgetAmountMinor)
	assert.Equal(t, money.Minor(85_000), comparison.ActualAmountMinor)
	assert.Equal(t, "on_track", comparison.Status)
}

func TestBudgetTimelineDTO_AllFields(t *testing.T) {
	day := date.New(2024, time.January, 15)

	timeline := BudgetTimelineDTO{
		Date:              day,
		PlannedSpentMinor: 50_000,
		ActualSpentMinor:  45_000,
		VarianceMinor:     5_000,
	}

	assert.Equal(t, day, timeline.Date)
	assert.Equal(t, money.Minor(50_000), timeline.PlannedSpentMinor)
	assert.Equal(t, money.Minor(45_000), timeline.ActualSpentMinor)
	assert.Equal(t, money.Minor(5_000), timeline.VarianceMinor)
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
		CategoryID:         categoryID,
		CategoryName:       "Food",
		CategoryType:       "expense",
		TotalAmountMinor:   150_000,
		Percentage:         30.0,
		TransactionCount:   45,
		AverageAmountMinor: 3_333,
		MinAmountMinor:     500,
		MaxAmountMinor:     20_000,
		Trend:              "increasing",
		TrendPercentage:    10.0,
	}

	assert.Equal(t, categoryID, analysis.CategoryID)
	assert.Equal(t, "Food", analysis.CategoryName)
	assert.Equal(t, money.Minor(150_000), analysis.TotalAmountMinor)
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
	assert.InDelta(t, 15.5, trend.Percentage, 0.001)
	assert.InDelta(t, 0.85, trend.Confidence, 0.001)
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
		CurrentAmountMinor:  150_000,
		PreviousAmountMinor: 136_400,
	}

	assert.Equal(t, categoryID, categoryTrend.CategoryID)
	assert.Equal(t, "Food", categoryTrend.CategoryName)
	assert.Equal(t, money.Minor(150_000), categoryTrend.CurrentAmountMinor)
}

func TestSeasonalPatternDTO_AllFields(t *testing.T) {
	pattern := SeasonalPatternDTO{
		Season:      "winter",
		AmountMinor: 500_000,
		Percentage:  28.0,
		Description: "Higher spending in winter",
	}

	assert.Equal(t, "winter", pattern.Season)
	assert.Equal(t, money.Minor(500_000), pattern.AmountMinor)
	assert.InDelta(t, 28.0, pattern.Percentage, 0.001)
}

func TestWeekdayPatternDTO_AllFields(t *testing.T) {
	pattern := WeekdayPatternDTO{
		Weekday:     "Saturday",
		AmountMinor: 80_000,
		Percentage:  20.0,
		Count:       25,
	}

	assert.Equal(t, "Saturday", pattern.Weekday)
	assert.Equal(t, money.Minor(80_000), pattern.AmountMinor)
	assert.Equal(t, 25, pattern.Count)
}

func TestForecastDTO_AllFields(t *testing.T) {
	day := date.New(2024, time.February, 1)

	forecast := ForecastDTO{
		Date:        day,
		AmountMinor: 150_000,
		Confidence:  0.85,
		LowerMinor:  120_000,
		UpperMinor:  180_000,
	}

	assert.Equal(t, day, forecast.Date)
	assert.Equal(t, money.Minor(150_000), forecast.AmountMinor)
	assert.InDelta(t, 0.85, forecast.Confidence, 0.001)
	assert.Equal(t, money.Minor(120_000), forecast.LowerMinor)
	assert.Equal(t, money.Minor(180_000), forecast.UpperMinor)
}

func TestPeriodComparisonDTO_AllFields(t *testing.T) {
	comparison := PeriodComparisonDTO{
		CurrentAmountMinor:  500_000,
		PreviousAmountMinor: 450_000,
		DifferenceMinor:     50_000,
		PercentageChange:    11.11,
		Description:         "Spending increased by 11.11%",
	}

	assert.Equal(t, money.Minor(500_000), comparison.CurrentAmountMinor)
	assert.Equal(t, money.Minor(450_000), comparison.PreviousAmountMinor)
	assert.Equal(t, money.Minor(50_000), comparison.DifferenceMinor)
	assert.InDelta(t, 11.11, comparison.PercentageChange, 0.001)
}

func TestBenchmarkComparisonDTO_AllFields(t *testing.T) {
	comparison := BenchmarkComparisonDTO{
		UserAmountMinor:      500_000,
		BenchmarkAmountMinor: 450_000,
		DifferenceMinor:      50_000,
		PercentageChange:     11.11,
		Status:               "above",
		Description:          "Above family average",
	}

	assert.Equal(t, money.Minor(500_000), comparison.UserAmountMinor)
	assert.Equal(t, "above", comparison.Status)
}

func TestProjectionDTO_AllFields(t *testing.T) {
	projection := ProjectionDTO{
		Period:                "next_month",
		StartDate:             date.New(2024, time.February, 1),
		EndDate:               date.New(2024, time.February, 29),
		ProjectedInflowMinor:  800_000,
		ProjectedOutflowMinor: 600_000,
		ProjectedBalanceMinor: 1_200_000,
		Confidence:            0.80,
		Assumptions:           []string{"Stable income", "Normal spending"},
	}

	assert.Equal(t, "next_month", projection.Period)
	assert.Equal(t, money.Minor(800_000), projection.ProjectedInflowMinor)
	assert.InDelta(t, 0.80, projection.Confidence, 0.001)
	assert.Len(t, projection.Assumptions, 2)
}

func TestScenarioDTO_AllFields(t *testing.T) {
	scenario := ScenarioDTO{
		Name:                 "Emergency Expense",
		Description:          "Unexpected car repair",
		Probability:          0.15,
		Impact:               "negative",
		ProjectedChangeMinor: -150_000,
	}

	assert.Equal(t, "Emergency Expense", scenario.Name)
	assert.InDelta(t, 0.15, scenario.Probability, 0.001)
	assert.Equal(t, "negative", scenario.Impact)
	assert.Equal(t, money.Minor(-150_000), scenario.ProjectedChangeMinor)
}

func TestRecommendationDTO_AllFields(t *testing.T) {
	recommendation := RecommendationDTO{
		Type:        "saving",
		Priority:    "high",
		Title:       "Reduce dining out",
		Description: "You can save $200/month by reducing dining out",
		ImpactMinor: 20_000,
		Effort:      "easy",
	}

	assert.Equal(t, "saving", recommendation.Type)
	assert.Equal(t, "high", recommendation.Priority)
	assert.Equal(t, money.Minor(20_000), recommendation.ImpactMinor)
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
	lastRun := now.Add(-24 * time.Hour)
	nextRun := now.Add(24 * time.Hour)

	scheduled := ScheduledReportDTO{
		ID:     reportID,
		Name:   "Monthly Report",
		Type:   report.TypeExpenses,
		UserID: uuid.New(),
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
