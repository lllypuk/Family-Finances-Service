package report

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID          uuid.UUID    `json:"id" bson:"_id"`
	Name        string       `json:"name" bson:"name"`
	Type        ReportType   `json:"type" bson:"type"`
	Period      ReportPeriod `json:"period" bson:"period"`
	FamilyID    uuid.UUID    `json:"family_id" bson:"family_id"`
	UserID      uuid.UUID    `json:"user_id" bson:"user_id"` // Кто создал отчет
	StartDate   time.Time    `json:"start_date" bson:"start_date"`
	EndDate     time.Time    `json:"end_date" bson:"end_date"`
	Data        ReportData   `json:"data" bson:"data"`
	GeneratedAt time.Time    `json:"generated_at" bson:"generated_at"`
}

type ReportType string

const (
	ReportTypeExpenses      ReportType = "expenses"       // Отчет по расходам
	ReportTypeIncome        ReportType = "income"         // Отчет по доходам
	ReportTypeBudget        ReportType = "budget"         // Отчет по бюджету
	ReportTypeCashFlow      ReportType = "cash_flow"      // Отчет по денежному потоку
	ReportTypeCategoryBreak ReportType = "category_break" // Разбивка по категориям
)

type ReportPeriod string

const (
	ReportPeriodDaily   ReportPeriod = "daily"
	ReportPeriodWeekly  ReportPeriod = "weekly"
	ReportPeriodMonthly ReportPeriod = "monthly"
	ReportPeriodYearly  ReportPeriod = "yearly"
	ReportPeriodCustom  ReportPeriod = "custom"
)

type ReportData struct {
	TotalIncome     float64                    `json:"total_income" bson:"total_income"`
	TotalExpenses   float64                    `json:"total_expenses" bson:"total_expenses"`
	NetIncome       float64                    `json:"net_income" bson:"net_income"`
	CategoryBreakdown []CategoryReportItem     `json:"category_breakdown" bson:"category_breakdown"`
	DailyBreakdown    []DailyReportItem        `json:"daily_breakdown" bson:"daily_breakdown"`
	TopExpenses       []TransactionReportItem  `json:"top_expenses" bson:"top_expenses"`
	BudgetComparison  []BudgetComparisonItem   `json:"budget_comparison" bson:"budget_comparison"`
}

type CategoryReportItem struct {
	CategoryID   uuid.UUID `json:"category_id" bson:"category_id"`
	CategoryName string    `json:"category_name" bson:"category_name"`
	Amount       float64   `json:"amount" bson:"amount"`
	Percentage   float64   `json:"percentage" bson:"percentage"`
	Count        int       `json:"count" bson:"count"`
}

type DailyReportItem struct {
	Date     time.Time `json:"date" bson:"date"`
	Income   float64   `json:"income" bson:"income"`
	Expenses float64   `json:"expenses" bson:"expenses"`
	Balance  float64   `json:"balance" bson:"balance"`
}

type TransactionReportItem struct {
	ID          uuid.UUID `json:"id" bson:"id"`
	Amount      float64   `json:"amount" bson:"amount"`
	Description string    `json:"description" bson:"description"`
	Category    string    `json:"category" bson:"category"`
	Date        time.Time `json:"date" bson:"date"`
}

type BudgetComparisonItem struct {
	BudgetID     uuid.UUID `json:"budget_id" bson:"budget_id"`
	BudgetName   string    `json:"budget_name" bson:"budget_name"`
	Planned      float64   `json:"planned" bson:"planned"`
	Actual       float64   `json:"actual" bson:"actual"`
	Difference   float64   `json:"difference" bson:"difference"`
	Percentage   float64   `json:"percentage" bson:"percentage"`
}

func NewReport(name string, reportType ReportType, period ReportPeriod, familyID, userID uuid.UUID, startDate, endDate time.Time) *Report {
	return &Report{
		ID:          uuid.New(),
		Name:        name,
		Type:        reportType,
		Period:      period,
		FamilyID:    familyID,
		UserID:      userID,
		StartDate:   startDate,
		EndDate:     endDate,
		Data:        ReportData{},
		GeneratedAt: time.Now(),
	}
}
