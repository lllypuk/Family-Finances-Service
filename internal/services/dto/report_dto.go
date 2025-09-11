package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/report"
)

// ReportRequestDTO contains basic report generation parameters
type ReportRequestDTO struct {
	Name      string         `json:"name"              validate:"required,min=1,max=100"`
	Type      report.Type    `json:"type"              validate:"required,oneof=expenses income budget cash_flow category_break"`
	Period    report.Period  `json:"period"            validate:"required,oneof=daily weekly monthly yearly custom"`
	FamilyID  uuid.UUID      `json:"family_id"         validate:"required"`
	UserID    uuid.UUID      `json:"user_id"           validate:"required"`
	StartDate time.Time      `json:"start_date"        validate:"required"`
	EndDate   time.Time      `json:"end_date"          validate:"required,gtfield=StartDate"`
	Filters   *ReportFilters `json:"filters,omitempty"`
}

// ReportFilters contains optional filters for report generation
type ReportFilters struct {
	CategoryIDs    []uuid.UUID `json:"category_ids,omitempty"`
	UserIDs        []uuid.UUID `json:"user_ids,omitempty"`
	MinAmount      *float64    `json:"min_amount,omitempty"   validate:"omitempty,min=0"`
	MaxAmount      *float64    `json:"max_amount,omitempty"   validate:"omitempty,min=0"`
	Description    string      `json:"description,omitempty"`
	IncludeSubcats bool        `json:"include_subcategories"`
}

// ExpenseReportDTO contains detailed expense report data
type ExpenseReportDTO struct {
	ID                uuid.UUID                  `json:"id"`
	Name              string                     `json:"name"`
	FamilyID          uuid.UUID                  `json:"family_id"`
	UserID            uuid.UUID                  `json:"user_id"`
	Period            report.Period              `json:"period"`
	StartDate         time.Time                  `json:"start_date"`
	EndDate           time.Time                  `json:"end_date"`
	TotalExpenses     float64                    `json:"total_expenses"`
	AverageDaily      float64                    `json:"average_daily"`
	CategoryBreakdown []CategoryBreakdownItemDTO `json:"category_breakdown"`
	DailyBreakdown    []DailyExpenseDTO          `json:"daily_breakdown"`
	TopExpenses       []TransactionSummaryDTO    `json:"top_expenses"`
	Trends            ExpenseTrendsDTO           `json:"trends"`
	Comparisons       ExpenseComparisonsDTO      `json:"comparisons"`
	GeneratedAt       time.Time                  `json:"generated_at"`
}

// IncomeReportDTO contains detailed income report data
type IncomeReportDTO struct {
	ID                uuid.UUID                  `json:"id"`
	Name              string                     `json:"name"`
	FamilyID          uuid.UUID                  `json:"family_id"`
	UserID            uuid.UUID                  `json:"user_id"`
	Period            report.Period              `json:"period"`
	StartDate         time.Time                  `json:"start_date"`
	EndDate           time.Time                  `json:"end_date"`
	TotalIncome       float64                    `json:"total_income"`
	AverageDaily      float64                    `json:"average_daily"`
	CategoryBreakdown []CategoryBreakdownItemDTO `json:"category_breakdown"`
	DailyBreakdown    []DailyIncomeDTO           `json:"daily_breakdown"`
	TopSources        []TransactionSummaryDTO    `json:"top_sources"`
	Trends            IncomeTrendsDTO            `json:"trends"`
	Comparisons       IncomeComparisonsDTO       `json:"comparisons"`
	GeneratedAt       time.Time                  `json:"generated_at"`
}

// BudgetComparisonDTO contains budget vs actual spending comparison
type BudgetComparisonDTO struct {
	ID            uuid.UUID                     `json:"id"`
	Name          string                        `json:"name"`
	FamilyID      uuid.UUID                     `json:"family_id"`
	UserID        uuid.UUID                     `json:"user_id"`
	Period        report.Period                 `json:"period"`
	StartDate     time.Time                     `json:"start_date"`
	EndDate       time.Time                     `json:"end_date"`
	TotalBudget   float64                       `json:"total_budget"`
	TotalSpent    float64                       `json:"total_spent"`
	TotalVariance float64                       `json:"total_variance"`
	Utilization   float64                       `json:"utilization_percentage"`
	Categories    []BudgetCategoryComparisonDTO `json:"categories"`
	Timeline      []BudgetTimelineDTO           `json:"timeline"`
	Alerts        []BudgetAlertReportDTO        `json:"alerts"`
	GeneratedAt   time.Time                     `json:"generated_at"`
}

// CashFlowReportDTO contains cash flow analysis
type CashFlowReportDTO struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	FamilyID       uuid.UUID              `json:"family_id"`
	UserID         uuid.UUID              `json:"user_id"`
	Period         report.Period          `json:"period"`
	StartDate      time.Time              `json:"start_date"`
	EndDate        time.Time              `json:"end_date"`
	OpeningBalance float64                `json:"opening_balance"`
	ClosingBalance float64                `json:"closing_balance"`
	NetCashFlow    float64                `json:"net_cash_flow"`
	TotalInflows   float64                `json:"total_inflows"`
	TotalOutflows  float64                `json:"total_outflows"`
	DailyFlow      []DailyCashFlowDTO     `json:"daily_flow"`
	WeeklyFlow     []WeeklyCashFlowDTO    `json:"weekly_flow"`
	MonthlyFlow    []MonthlyCashFlowDTO   `json:"monthly_flow"`
	Projections    CashFlowProjectionsDTO `json:"projections"`
	GeneratedAt    time.Time              `json:"generated_at"`
}

// CategoryBreakdownDTO contains category-based spending breakdown
type CategoryBreakdownDTO struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	FamilyID    uuid.UUID                    `json:"family_id"`
	UserID      uuid.UUID                    `json:"user_id"`
	Period      report.Period                `json:"period"`
	StartDate   time.Time                    `json:"start_date"`
	EndDate     time.Time                    `json:"end_date"`
	Categories  []CategoryAnalysisDTO        `json:"categories"`
	Hierarchy   []CategoryHierarchyReportDTO `json:"hierarchy"`
	Trends      CategoryTrendsDTO            `json:"trends"`
	Comparisons CategoryComparisonsDTO       `json:"comparisons"`
	GeneratedAt time.Time                    `json:"generated_at"`
}

// Supporting DTOs for detailed breakdown

type CategoryBreakdownItemDTO struct {
	CategoryID    uuid.UUID  `json:"category_id"`
	CategoryName  string     `json:"category_name"`
	CategoryType  string     `json:"category_type"`
	Amount        float64    `json:"amount"`
	Percentage    float64    `json:"percentage"`
	Count         int        `json:"transaction_count"`
	AverageAmount float64    `json:"average_amount"`
	ParentID      *uuid.UUID `json:"parent_id,omitempty"`
}

type DailyExpenseDTO struct {
	Date       time.Time `json:"date"`
	Amount     float64   `json:"amount"`
	Count      int       `json:"transaction_count"`
	Categories []string  `json:"top_categories"`
}

type DailyIncomeDTO struct {
	Date    time.Time `json:"date"`
	Amount  float64   `json:"amount"`
	Count   int       `json:"transaction_count"`
	Sources []string  `json:"top_sources"`
}

type DailyCashFlowDTO struct {
	Date    time.Time `json:"date"`
	Inflow  float64   `json:"inflow"`
	Outflow float64   `json:"outflow"`
	NetFlow float64   `json:"net_flow"`
	Balance float64   `json:"running_balance"`
}

type WeeklyCashFlowDTO struct {
	WeekStart time.Time `json:"week_start"`
	WeekEnd   time.Time `json:"week_end"`
	Inflow    float64   `json:"inflow"`
	Outflow   float64   `json:"outflow"`
	NetFlow   float64   `json:"net_flow"`
}

type MonthlyCashFlowDTO struct {
	Month   time.Time `json:"month"`
	Inflow  float64   `json:"inflow"`
	Outflow float64   `json:"outflow"`
	NetFlow float64   `json:"net_flow"`
}

type TransactionSummaryDTO struct {
	ID          uuid.UUID `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	UserName    string    `json:"user_name"`
}

type BudgetCategoryComparisonDTO struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	BudgetAmount float64   `json:"budget_amount"`
	ActualAmount float64   `json:"actual_amount"`
	Variance     float64   `json:"variance"`
	Utilization  float64   `json:"utilization_percentage"`
	Status       string    `json:"status"` // under_budget, over_budget, on_track
}

type BudgetTimelineDTO struct {
	Date         time.Time `json:"date"`
	PlannedSpent float64   `json:"planned_spent"`
	ActualSpent  float64   `json:"actual_spent"`
	Variance     float64   `json:"variance"`
}

type BudgetAlertReportDTO struct {
	Type       string    `json:"type"` // warning, critical, info
	CategoryID uuid.UUID `json:"category_id"`
	Category   string    `json:"category"`
	Message    string    `json:"message"`
	Threshold  float64   `json:"threshold"`
	Current    float64   `json:"current"`
	Severity   int       `json:"severity"` // 1-10
}

type CategoryAnalysisDTO struct {
	CategoryID       uuid.UUID             `json:"category_id"`
	CategoryName     string                `json:"category_name"`
	CategoryType     string                `json:"category_type"`
	TotalAmount      float64               `json:"total_amount"`
	Percentage       float64               `json:"percentage"`
	TransactionCount int                   `json:"transaction_count"`
	AverageAmount    float64               `json:"average_amount"`
	MinAmount        float64               `json:"min_amount"`
	MaxAmount        float64               `json:"max_amount"`
	Trend            string                `json:"trend"` // increasing, decreasing, stable
	TrendPercentage  float64               `json:"trend_percentage"`
	Subcategories    []CategoryAnalysisDTO `json:"subcategories,omitempty"`
	MonthlyBreakdown []MonthlyCategoryDTO  `json:"monthly_breakdown"`
}

type CategoryHierarchyReportDTO struct {
	CategoryID   uuid.UUID                    `json:"category_id"`
	CategoryName string                       `json:"category_name"`
	Level        int                          `json:"level"`
	Amount       float64                      `json:"amount"`
	Percentage   float64                      `json:"percentage"`
	Children     []CategoryHierarchyReportDTO `json:"children,omitempty"`
}

type MonthlyCategoryDTO struct {
	Month  time.Time `json:"month"`
	Amount float64   `json:"amount"`
	Count  int       `json:"transaction_count"`
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
	CategoryID     uuid.UUID        `json:"category_id"`
	CategoryName   string           `json:"category_name"`
	Trend          TrendAnalysisDTO `json:"trend"`
	CurrentAmount  float64          `json:"current_amount"`
	PreviousAmount float64          `json:"previous_amount"`
}

type SeasonalPatternDTO struct {
	Season      string  `json:"season"` // spring, summer, fall, winter
	Amount      float64 `json:"amount"`
	Percentage  float64 `json:"percentage"`
	Description string  `json:"description"`
}

type WeekdayPatternDTO struct {
	Weekday    string  `json:"weekday"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
	Count      int     `json:"transaction_count"`
}

type ForecastDTO struct {
	Date       time.Time `json:"date"`
	Amount     float64   `json:"predicted_amount"`
	Confidence float64   `json:"confidence"`
	Lower      float64   `json:"lower_bound"`
	Upper      float64   `json:"upper_bound"`
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
	CurrentAmount    float64 `json:"current_amount"`
	PreviousAmount   float64 `json:"previous_amount"`
	Difference       float64 `json:"difference"`
	PercentageChange float64 `json:"percentage_change"`
	Description      string  `json:"description"`
}

type BenchmarkComparisonDTO struct {
	UserAmount       float64 `json:"user_amount"`
	BenchmarkAmount  float64 `json:"benchmark_amount"`
	Difference       float64 `json:"difference"`
	PercentageChange float64 `json:"percentage_change"`
	Status           string  `json:"status"` // below, above, average
	Description      string  `json:"description"`
}

type CategoryComparisonDTO struct {
	CategoryID       uuid.UUID `json:"category_id"`
	CategoryName     string    `json:"category_name"`
	CurrentAmount    float64   `json:"current_amount"`
	PreviousAmount   float64   `json:"previous_amount"`
	Difference       float64   `json:"difference"`
	PercentageChange float64   `json:"percentage_change"`
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
	Period           string    `json:"period"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	ProjectedInflow  float64   `json:"projected_inflow"`
	ProjectedOutflow float64   `json:"projected_outflow"`
	ProjectedBalance float64   `json:"projected_balance"`
	Confidence       float64   `json:"confidence"`
	Assumptions      []string  `json:"assumptions"`
}

type ScenarioDTO struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Probability     float64 `json:"probability"`
	Impact          string  `json:"impact"` // positive, negative, neutral
	ProjectedChange float64 `json:"projected_change"`
}

type RecommendationDTO struct {
	Type        string  `json:"type"`     // saving, spending, investment
	Priority    string  `json:"priority"` // high, medium, low
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      float64 `json:"estimated_impact"`
	Effort      string  `json:"effort"` // easy, medium, hard
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
	FamilyID     uuid.UUID         `json:"family_id"         validate:"required"`
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
	FamilyID     uuid.UUID         `json:"family_id"`
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
