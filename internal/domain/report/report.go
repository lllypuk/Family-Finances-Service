package report

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

type Report struct {
	ID          uuid.UUID `json:"id"           bson:"_id"`
	Name        string    `json:"name"         bson:"name"`
	Type        Type      `json:"type"         bson:"type"`
	Period      Period    `json:"period"       bson:"period"`
	UserID      uuid.UUID `json:"user_id"      bson:"user_id"` // Кто создал отчет
	StartDate   date.Date `json:"start_date"   bson:"start_date"`
	EndDate     date.Date `json:"end_date"     bson:"end_date"`
	Data        Data      `json:"data"         bson:"data"`
	GeneratedAt time.Time `json:"generated_at" bson:"generated_at"`
}

type Type string

const (
	TypeExpenses      Type = "expenses"           // Отчет по расходам
	TypeIncome        Type = "income"             // Отчет по доходам
	TypeBudget        Type = "budget"             // Отчет по бюджету
	TypeCashFlow      Type = "cash_flow"          // Отчет по денежному потоку
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
	TotalIncomeMinor   money.Minor             `json:"total_income_minor"`
	TotalExpensesMinor money.Minor             `json:"total_expenses_minor"`
	NetIncomeMinor     money.Minor             `json:"net_income_minor"`
	CategoryBreakdown  []CategoryReportItem    `json:"category_breakdown"`
	DailyBreakdown     []DailyReportItem       `json:"daily_breakdown"`
	TopExpenses        []TransactionReportItem `json:"top_expenses"`
	BudgetComparison   []BudgetComparisonItem  `json:"budget_comparison"`
}

type CategoryReportItem struct {
	CategoryID   uuid.UUID   `json:"category_id"`
	CategoryName string      `json:"category_name"`
	AmountMinor  money.Minor `json:"amount_minor"`
	Percentage   float64     `json:"percentage"`
	Count        int         `json:"count"`
}

type DailyReportItem struct {
	Date          date.Date   `json:"date"`
	IncomeMinor   money.Minor `json:"income_minor"`
	ExpensesMinor money.Minor `json:"expenses_minor"`
	BalanceMinor  money.Minor `json:"balance_minor"`
}

type TransactionReportItem struct {
	ID          uuid.UUID   `json:"id"`
	AmountMinor money.Minor `json:"amount_minor"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Date        date.Date   `json:"date"`
}

type BudgetComparisonItem struct {
	BudgetID        uuid.UUID   `json:"budget_id"`
	BudgetName      string      `json:"budget_name"`
	PlannedMinor    money.Minor `json:"planned_minor"`
	ActualMinor     money.Minor `json:"actual_minor"`
	DifferenceMinor money.Minor `json:"difference_minor"`
	Percentage      float64     `json:"percentage"`
}

func NewReport(
	name string,
	reportType Type,
	period Period,
	userID uuid.UUID,
	startDate, endDate date.Date,
) *Report {
	return &Report{
		ID:          uuid.New(),
		Name:        name,
		Type:        reportType,
		Period:      period,
		UserID:      userID,
		StartDate:   startDate,
		EndDate:     endDate,
		Data:        Data{},
		GeneratedAt: time.Now(),
	}
}
