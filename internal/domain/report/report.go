package report

import (
	"time"

	"github.com/google/uuid"
)

type Report struct {
	ID          uuid.UUID `json:"id"           bson:"_id"`
	Name        string    `json:"name"         bson:"name"`
	Type        Type      `json:"type"         bson:"type"`
	Period      Period    `json:"period"       bson:"period"`
	FamilyID    uuid.UUID `json:"family_id"    bson:"family_id"`
	UserID      uuid.UUID `json:"user_id"      bson:"user_id"` // Кто создал отчет
	StartDate   time.Time `json:"start_date"   bson:"start_date"`
	EndDate     time.Time `json:"end_date"     bson:"end_date"`
	Data        Data      `json:"data"         bson:"data"`
	GeneratedAt time.Time `json:"generated_at" bson:"generated_at"`
}

type Type string

const (
	TypeExpenses      Type = "expenses"       // Отчет по расходам
	TypeIncome        Type = "income"         // Отчет по доходам
	TypeBudget        Type = "budget"         // Отчет по бюджету
	TypeCashFlow      Type = "cash_flow"      // Отчет по денежному потоку
	TypeCategoryBreak Type = "category_breakdown" // Разбивка по категориям
)

type Period string

const (
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
	PeriodCustom  Period = "custom"
)

type Data struct {
	TotalIncome       float64                 `json:"total_income"       bson:"total_income"`
	TotalExpenses     float64                 `json:"total_expenses"     bson:"total_expenses"`
	NetIncome         float64                 `json:"net_income"         bson:"net_income"`
	CategoryBreakdown []CategoryReportItem    `json:"category_breakdown" bson:"category_breakdown"`
	DailyBreakdown    []DailyReportItem       `json:"daily_breakdown"    bson:"daily_breakdown"`
	TopExpenses       []TransactionReportItem `json:"top_expenses"       bson:"top_expenses"`
	BudgetComparison  []BudgetComparisonItem  `json:"budget_comparison"  bson:"budget_comparison"`
}

type CategoryReportItem struct {
	CategoryID   uuid.UUID `json:"category_id"   bson:"category_id"`
	CategoryName string    `json:"category_name" bson:"category_name"`
	Amount       float64   `json:"amount"        bson:"amount"`
	Percentage   float64   `json:"percentage"    bson:"percentage"`
	Count        int       `json:"count"         bson:"count"`
}

type DailyReportItem struct {
	Date     time.Time `json:"date"     bson:"date"`
	Income   float64   `json:"income"   bson:"income"`
	Expenses float64   `json:"expenses" bson:"expenses"`
	Balance  float64   `json:"balance"  bson:"balance"`
}

type TransactionReportItem struct {
	ID          uuid.UUID `json:"id"          bson:"id"`
	Amount      float64   `json:"amount"      bson:"amount"`
	Description string    `json:"description" bson:"description"`
	Category    string    `json:"category"    bson:"category"`
	Date        time.Time `json:"date"        bson:"date"`
}

type BudgetComparisonItem struct {
	BudgetID   uuid.UUID `json:"budget_id"   bson:"budget_id"`
	BudgetName string    `json:"budget_name" bson:"budget_name"`
	Planned    float64   `json:"planned"     bson:"planned"`
	Actual     float64   `json:"actual"      bson:"actual"`
	Difference float64   `json:"difference"  bson:"difference"`
	Percentage float64   `json:"percentage"  bson:"percentage"`
}

func NewReport(
	name string,
	reportType Type,
	period Period,
	familyID, userID uuid.UUID,
	startDate, endDate time.Time,
) *Report {
	return &Report{
		ID:          uuid.New(),
		Name:        name,
		Type:        reportType,
		Period:      period,
		FamilyID:    familyID,
		UserID:      userID,
		StartDate:   startDate,
		EndDate:     endDate,
		Data:        Data{},
		GeneratedAt: time.Now(),
	}
}

// GetFamilyID returns the family ID of the report
func (r *Report) GetFamilyID() uuid.UUID {
	return r.FamilyID
}
