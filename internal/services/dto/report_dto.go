package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
)

// ReportRequestDTO contains basic report generation parameters
type ReportRequestDTO struct {
	Name      string         `json:"name"              validate:"required,min=1,max=100"`
	Type      report.Type    `json:"type"              validate:"required,oneof=expenses income budget cash_flow category_break"`
	Period    report.Period  `json:"period"            validate:"required,oneof=daily weekly monthly yearly custom"`
	UserID    uuid.UUID      `json:"user_id"           validate:"required"`
	StartDate date.Date      `json:"start_date"        validate:"required"`
	EndDate   date.Date      `json:"end_date"          validate:"required"`
	Filters   *ReportFilters `json:"filters,omitempty"`
}

// ReportFilters contains optional filters for report generation
type ReportFilters struct {
	CategoryIDs    []uuid.UUID  `json:"category_ids,omitempty"`
	UserIDs        []uuid.UUID  `json:"user_ids,omitempty"`
	MinAmountMinor *money.Minor `json:"min_amount_minor,omitempty" validate:"omitempty,min=0"`
	MaxAmountMinor *money.Minor `json:"max_amount_minor,omitempty" validate:"omitempty,min=0"`
	Description    string       `json:"description,omitempty"`
	IncludeSubcats bool         `json:"include_subcategories"`
}

// ExpenseReportDTO contains detailed expense report data
type ExpenseReportDTO struct {
	ID                 uuid.UUID                  `json:"id"`
	Name               string                     `json:"name"`
	UserID             uuid.UUID                  `json:"user_id"`
	Period             report.Period              `json:"period"`
	StartDate          date.Date                  `json:"start_date"`
	EndDate            date.Date                  `json:"end_date"`
	TotalExpensesMinor money.Minor                `json:"total_expenses_minor"`
	AverageDailyMinor  money.Minor                `json:"average_daily_minor"`
	CategoryBreakdown  []CategoryBreakdownItemDTO `json:"category_breakdown"`
	DailyBreakdown     []DailyExpenseDTO          `json:"daily_breakdown"`
	TopExpenses        []TransactionSummaryDTO    `json:"top_expenses"`
	Trends             ExpenseTrendsDTO           `json:"trends"`
	Comparisons        ExpenseComparisonsDTO      `json:"comparisons"`
	GeneratedAt        time.Time                  `json:"generated_at"`
}

// IncomeReportDTO contains detailed income report data
type IncomeReportDTO struct {
	ID                uuid.UUID                  `json:"id"`
	Name              string                     `json:"name"`
	UserID            uuid.UUID                  `json:"user_id"`
	Period            report.Period              `json:"period"`
	StartDate         date.Date                  `json:"start_date"`
	EndDate           date.Date                  `json:"end_date"`
	TotalIncomeMinor  money.Minor                `json:"total_income_minor"`
	AverageDailyMinor money.Minor                `json:"average_daily_minor"`
	CategoryBreakdown []CategoryBreakdownItemDTO `json:"category_breakdown"`
	DailyBreakdown    []DailyIncomeDTO           `json:"daily_breakdown"`
	TopSources        []TransactionSummaryDTO    `json:"top_sources"`
	Trends            IncomeTrendsDTO            `json:"trends"`
	Comparisons       IncomeComparisonsDTO       `json:"comparisons"`
	GeneratedAt       time.Time                  `json:"generated_at"`
}

// BudgetComparisonDTO contains budget vs actual spending comparison
type BudgetComparisonDTO struct {
	ID                 uuid.UUID                     `json:"id"`
	Name               string                        `json:"name"`
	UserID             uuid.UUID                     `json:"user_id"`
	Period             report.Period                 `json:"period"`
	StartDate          date.Date                     `json:"start_date"`
	EndDate            date.Date                     `json:"end_date"`
	TotalBudgetMinor   money.Minor                   `json:"total_budget_minor"`
	TotalSpentMinor    money.Minor                   `json:"total_spent_minor"`
	TotalVarianceMinor money.Minor                   `json:"total_variance_minor"`
	Utilization        float64                       `json:"utilization_percentage"`
	Categories         []BudgetCategoryComparisonDTO `json:"categories"`
	Timeline           []BudgetTimelineDTO           `json:"timeline"`
	Alerts             []BudgetAlertReportDTO        `json:"alerts"`
	GeneratedAt        time.Time                     `json:"generated_at"`
}

// CashFlowReportDTO contains cash flow analysis
type CashFlowReportDTO struct {
	ID                  uuid.UUID              `json:"id"`
	Name                string                 `json:"name"`
	UserID              uuid.UUID              `json:"user_id"`
	Period              report.Period          `json:"period"`
	StartDate           date.Date              `json:"start_date"`
	EndDate             date.Date              `json:"end_date"`
	OpeningBalanceMinor money.Minor            `json:"opening_balance_minor"`
	ClosingBalanceMinor money.Minor            `json:"closing_balance_minor"`
	NetCashFlowMinor    money.Minor            `json:"net_cash_flow_minor"`
	TotalInflowsMinor   money.Minor            `json:"total_inflows_minor"`
	TotalOutflowsMinor  money.Minor            `json:"total_outflows_minor"`
	DailyFlow           []DailyCashFlowDTO     `json:"daily_flow"`
	WeeklyFlow          []WeeklyCashFlowDTO    `json:"weekly_flow"`
	MonthlyFlow         []MonthlyCashFlowDTO   `json:"monthly_flow"`
	Projections         CashFlowProjectionsDTO `json:"projections"`
	GeneratedAt         time.Time              `json:"generated_at"`
}

// CategoryBreakdownDTO contains category-based spending breakdown
type CategoryBreakdownDTO struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	UserID      uuid.UUID                    `json:"user_id"`
	Period      report.Period                `json:"period"`
	StartDate   date.Date                    `json:"start_date"`
	EndDate     date.Date                    `json:"end_date"`
	Categories  []CategoryAnalysisDTO        `json:"categories"`
	Hierarchy   []CategoryHierarchyReportDTO `json:"hierarchy"`
	Trends      CategoryTrendsDTO            `json:"trends"`
	Comparisons CategoryComparisonsDTO       `json:"comparisons"`
	GeneratedAt time.Time                    `json:"generated_at"`
}

// Supporting DTOs for detailed breakdown

type CategoryBreakdownItemDTO struct {
	CategoryID         uuid.UUID   `json:"category_id"`
	CategoryName       string      `json:"category_name"`
	CategoryType       string      `json:"category_type"`
	AmountMinor        money.Minor `json:"amount_minor"`
	Percentage         float64     `json:"percentage"`
	Count              int         `json:"transaction_count"`
	AverageAmountMinor money.Minor `json:"average_amount_minor"`
	ParentID           *uuid.UUID  `json:"parent_id,omitempty"`
}

type DailyExpenseDTO struct {
	Date        date.Date   `json:"date"`
	AmountMinor money.Minor `json:"amount_minor"`
	Count       int         `json:"transaction_count"`
	Categories  []string    `json:"top_categories"`
}

type DailyIncomeDTO struct {
	Date        date.Date   `json:"date"`
	AmountMinor money.Minor `json:"amount_minor"`
	Count       int         `json:"transaction_count"`
	Sources     []string    `json:"top_sources"`
}

type DailyCashFlowDTO struct {
	Date         date.Date   `json:"date"`
	InflowMinor  money.Minor `json:"inflow_minor"`
	OutflowMinor money.Minor `json:"outflow_minor"`
	NetFlowMinor money.Minor `json:"net_flow_minor"`
	BalanceMinor money.Minor `json:"running_balance_minor"`
}

type WeeklyCashFlowDTO struct {
	WeekStart    date.Date   `json:"week_start"`
	WeekEnd      date.Date   `json:"week_end"`
	InflowMinor  money.Minor `json:"inflow_minor"`
	OutflowMinor money.Minor `json:"outflow_minor"`
	NetFlowMinor money.Minor `json:"net_flow_minor"`
}

type MonthlyCashFlowDTO struct {
	Month        date.Date   `json:"month"`
	InflowMinor  money.Minor `json:"inflow_minor"`
	OutflowMinor money.Minor `json:"outflow_minor"`
	NetFlowMinor money.Minor `json:"net_flow_minor"`
}

type TransactionSummaryDTO struct {
	ID          uuid.UUID   `json:"id"`
	AmountMinor money.Minor `json:"amount_minor"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Date        date.Date   `json:"date"`
	UserName    string      `json:"user_name"`
}

type BudgetCategoryComparisonDTO struct {
	BudgetID          uuid.UUID   `json:"budget_id"`
	BudgetName        string      `json:"budget_name"`
	CategoryID        uuid.UUID   `json:"category_id"`
	CategoryName      string      `json:"category_name"`
	BudgetAmountMinor money.Minor `json:"budget_amount_minor"`
	ActualAmountMinor money.Minor `json:"actual_amount_minor"`
	VarianceMinor     money.Minor `json:"variance_minor"`
	Utilization       float64     `json:"utilization_percentage"`
	Status            string      `json:"status"` // under_budget, over_budget, on_track
}

type BudgetTimelineDTO struct {
	Date              date.Date   `json:"date"`
	PlannedSpentMinor money.Minor `json:"planned_spent_minor"`
	ActualSpentMinor  money.Minor `json:"actual_spent_minor"`
	VarianceMinor     money.Minor `json:"variance_minor"`
}

type BudgetAlertReportDTO struct {
	Type       string    `json:"type"` // warning, critical, info
	CategoryID uuid.UUID `json:"category_id"`
	Category   string    `json:"category"`
	Message    string    `json:"message"`
	Threshold  float64   `json:"threshold"` // процент утилизации бюджета
	Current    float64   `json:"current"`   // процент утилизации бюджета
	Severity   int       `json:"severity"`  // 1-10
}

type CategoryAnalysisDTO struct {
	CategoryID         uuid.UUID             `json:"category_id"`
	CategoryName       string                `json:"category_name"`
	CategoryType       string                `json:"category_type"`
	TotalAmountMinor   money.Minor           `json:"total_amount_minor"`
	Percentage         float64               `json:"percentage"`
	TransactionCount   int                   `json:"transaction_count"`
	AverageAmountMinor money.Minor           `json:"average_amount_minor"`
	MinAmountMinor     money.Minor           `json:"min_amount_minor"`
	MaxAmountMinor     money.Minor           `json:"max_amount_minor"`
	Trend              string                `json:"trend"` // increasing, decreasing, stable
	TrendPercentage    float64               `json:"trend_percentage"`
	Subcategories      []CategoryAnalysisDTO `json:"subcategories,omitempty"`
	MonthlyBreakdown   []MonthlyCategoryDTO  `json:"monthly_breakdown"`
}

type CategoryHierarchyReportDTO struct {
	CategoryID   uuid.UUID                    `json:"category_id"`
	CategoryName string                       `json:"category_name"`
	Level        int                          `json:"level"`
	AmountMinor  money.Minor                  `json:"amount_minor"`
	Percentage   float64                      `json:"percentage"`
	Children     []CategoryHierarchyReportDTO `json:"children,omitempty"`
}

type MonthlyCategoryDTO struct {
	Month       date.Date   `json:"month"`
	AmountMinor money.Minor `json:"amount_minor"`
	Count       int         `json:"transaction_count"`
}

// Trend Analysis DTOs

type ExpenseTrendsDTO struct {
	MonthlyTrend     TrendAnalysisDTO     `json:"monthly_trend"`
	CategoryTrends   []CategoryTrendDTO   `json:"category_trends"`
	SeasonalPatterns []SeasonalPatternDTO `json:"seasonal_patterns"`
	WeekdayPatterns  []WeekdayPatternDTO  `json:"weekday_patterns"`
	Forecasts        []ForecastDTO        `json:"forecasts"`
}

type IncomeTrendsDTO struct {
	MonthlyTrend     TrendAnalysisDTO     `json:"monthly_trend"`
	SourceTrends     []CategoryTrendDTO   `json:"source_trends"`
	SeasonalPatterns []SeasonalPatternDTO `json:"seasonal_patterns"`
	Forecasts        []ForecastDTO        `json:"forecasts"`
}

type CategoryTrendsDTO struct {
	TopGrowing       []CategoryTrendDTO   `json:"top_growing"`
	TopDeclining     []CategoryTrendDTO   `json:"top_declining"`
	MostVolatile     []CategoryTrendDTO   `json:"most_volatile"`
	SeasonalPatterns []SeasonalPatternDTO `json:"seasonal_patterns"`
}

type TrendAnalysisDTO struct {
	Direction   string  `json:"direction"` // increasing, decreasing, stable
	Percentage  float64 `json:"percentage"`
	Confidence  float64 `json:"confidence"` // 0-1
	Description string  `json:"description"`
}

type CategoryTrendDTO struct {
	CategoryID          uuid.UUID        `json:"category_id"`
	CategoryName        string           `json:"category_name"`
	Trend               TrendAnalysisDTO `json:"trend"`
	CurrentAmountMinor  money.Minor      `json:"current_amount_minor"`
	PreviousAmountMinor money.Minor      `json:"previous_amount_minor"`
}

type SeasonalPatternDTO struct {
	Season      string      `json:"season"` // spring, summer, fall, winter
	AmountMinor money.Minor `json:"amount_minor"`
	Percentage  float64     `json:"percentage"`
	Description string      `json:"description"`
}

type WeekdayPatternDTO struct {
	Weekday     string      `json:"weekday"`
	AmountMinor money.Minor `json:"amount_minor"`
	Percentage  float64     `json:"percentage"`
	Count       int         `json:"transaction_count"`
}

type ForecastDTO struct {
	Date        date.Date   `json:"date"`
	AmountMinor money.Minor `json:"predicted_amount_minor"`
	Confidence  float64     `json:"confidence"`
	LowerMinor  money.Minor `json:"lower_bound_minor"`
	UpperMinor  money.Minor `json:"upper_bound_minor"`
}

// Comparison DTOs

type ExpenseComparisonsDTO struct {
	PreviousPeriod  PeriodComparisonDTO    `json:"previous_period"`
	YearOverYear    PeriodComparisonDTO    `json:"year_over_year"`
	FamilyBenchmark BenchmarkComparisonDTO `json:"family_benchmark"`
}

type IncomeComparisonsDTO struct {
	PreviousPeriod  PeriodComparisonDTO    `json:"previous_period"`
	YearOverYear    PeriodComparisonDTO    `json:"year_over_year"`
	FamilyBenchmark BenchmarkComparisonDTO `json:"family_benchmark"`
}

type CategoryComparisonsDTO struct {
	PreviousPeriod []CategoryComparisonDTO `json:"previous_period"`
	YearOverYear   []CategoryComparisonDTO `json:"year_over_year"`
}

type PeriodComparisonDTO struct {
	CurrentAmountMinor  money.Minor `json:"current_amount_minor"`
	PreviousAmountMinor money.Minor `json:"previous_amount_minor"`
	DifferenceMinor     money.Minor `json:"difference_minor"`
	PercentageChange    float64     `json:"percentage_change"`
	Description         string      `json:"description"`
}

type BenchmarkComparisonDTO struct {
	UserAmountMinor      money.Minor `json:"user_amount_minor"`
	BenchmarkAmountMinor money.Minor `json:"benchmark_amount_minor"`
	DifferenceMinor      money.Minor `json:"difference_minor"`
	PercentageChange     float64     `json:"percentage_change"`
	Status               string      `json:"status"` // below, above, average
	Description          string      `json:"description"`
}

type CategoryComparisonDTO struct {
	CategoryID          uuid.UUID   `json:"category_id"`
	CategoryName        string      `json:"category_name"`
	CurrentAmountMinor  money.Minor `json:"current_amount_minor"`
	PreviousAmountMinor money.Minor `json:"previous_amount_minor"`
	DifferenceMinor     money.Minor `json:"difference_minor"`
	PercentageChange    float64     `json:"percentage_change"`
}

// Cash Flow Projections

type CashFlowProjectionsDTO struct {
	NextMonth       ProjectionDTO       `json:"next_month"`
	NextQuarter     ProjectionDTO       `json:"next_quarter"`
	NextYear        ProjectionDTO       `json:"next_year"`
	Scenarios       []ScenarioDTO       `json:"scenarios"`
	Recommendations []RecommendationDTO `json:"recommendations"`
}

type ProjectionDTO struct {
	Period                string      `json:"period"`
	StartDate             date.Date   `json:"start_date"`
	EndDate               date.Date   `json:"end_date"`
	ProjectedInflowMinor  money.Minor `json:"projected_inflow_minor"`
	ProjectedOutflowMinor money.Minor `json:"projected_outflow_minor"`
	ProjectedBalanceMinor money.Minor `json:"projected_balance_minor"`
	Confidence            float64     `json:"confidence"`
	Assumptions           []string    `json:"assumptions"`
}

type ScenarioDTO struct {
	Name                 string      `json:"name"`
	Description          string      `json:"description"`
	Probability          float64     `json:"probability"`
	Impact               string      `json:"impact"` // positive, negative, neutral
	ProjectedChangeMinor money.Minor `json:"projected_change_minor"`
}

type RecommendationDTO struct {
	Type        string      `json:"type"`     // saving, spending, investment
	Priority    string      `json:"priority"` // high, medium, low
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ImpactMinor money.Minor `json:"estimated_impact_minor"`
	Effort      string      `json:"effort"` // easy, medium, hard
}

// Export DTOs

type ExportRequestDTO struct {
	ReportID uuid.UUID        `json:"report_id" validate:"required"`
	Format   string           `json:"format"    validate:"required,oneof=pdf csv excel json"`
	Options  ExportOptionsDTO `json:"options"`
}

type ExportOptionsDTO struct {
	IncludeCharts  bool     `json:"include_charts"`
	IncludeDetails bool     `json:"include_details"`
	Sections       []string `json:"sections,omitempty"`
	Language       string   `json:"language"`
	Currency       string   `json:"currency"`
	DateFormat     string   `json:"date_format"`
}

// Scheduled Reports

type ScheduleReportDTO struct {
	Name         string            `json:"name"              validate:"required,min=1,max=100"`
	Type         report.Type       `json:"type"              validate:"required"`
	UserID       uuid.UUID         `json:"user_id"           validate:"required"`
	Schedule     ScheduleConfigDTO `json:"schedule"          validate:"required"`
	Filters      *ReportFilters    `json:"filters,omitempty"`
	ExportFormat string            `json:"export_format"     validate:"required,oneof=pdf csv excel"`
	Recipients   []string          `json:"recipients"        validate:"required,min=1,dive,email"`
	Active       bool              `json:"active"`
}

type ScheduledReportDTO struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Type         report.Type       `json:"type"`
	UserID       uuid.UUID         `json:"user_id"`
	Schedule     ScheduleConfigDTO `json:"schedule"`
	Filters      *ReportFilters    `json:"filters,omitempty"`
	ExportFormat string            `json:"export_format"`
	Recipients   []string          `json:"recipients"`
	Active       bool              `json:"active"`
	LastRun      *time.Time        `json:"last_run,omitempty"`
	NextRun      time.Time         `json:"next_run"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type ScheduleConfigDTO struct {
	Frequency  string `json:"frequency"              validate:"required,oneof=daily weekly monthly quarterly yearly"`
	DayOfWeek  *int   `json:"day_of_week,omitempty"  validate:"omitempty,min=0,max=6"`
	DayOfMonth *int   `json:"day_of_month,omitempty" validate:"omitempty,min=1,max=31"`
	Time       string `json:"time"                   validate:"required"` // HH:MM format
	Timezone   string `json:"timezone"               validate:"required"`
}
