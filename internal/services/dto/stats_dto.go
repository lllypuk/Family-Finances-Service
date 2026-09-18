package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
)

// StatsSummary — агрегаты за период для дашборда и GET /stats/summary.
// Доли (Delta, Share, Utilization) — именно доли: 0.12 = +12%; форматирование в процентах — на вызывающей стороне.
type StatsSummary struct {
	From              date.Date           `json:"from"`
	To                date.Date           `json:"to"`
	Current           PeriodTotals        `json:"current"`
	Previous          PeriodTotals        `json:"previous"`
	HasPreviousData   bool                `json:"has_previous_data"`
	IncomeDelta       float64             `json:"income_delta"`
	ExpensesDelta     float64             `json:"expenses_delta"`
	ExpenseCategories []CategoryShare     `json:"expense_categories"`
	IncomeCategories  []CategoryShare     `json:"income_categories"`
	Budgets           []BudgetProgress    `json:"budgets"`
	Recent            []RecentTransaction `json:"recent"`
	TransactionsTotal int                 `json:"transactions_total"`
}

// PeriodTotals — суммы за период [From, To].
type PeriodTotals struct {
	From             date.Date   `json:"from"`
	To               date.Date   `json:"to"`
	IncomeMinor      money.Minor `json:"income_minor"`
	ExpensesMinor    money.Minor `json:"expenses_minor"`
	NetMinor         money.Minor `json:"net_minor"`
	TransactionCount int         `json:"transaction_count"`
}

// StatsMonthly — помесячный ряд за период [From, To] для GET /stats/monthly.
type StatsMonthly struct {
	From   date.Date     `json:"from"`
	To     date.Date     `json:"to"`
	Months []MonthTotals `json:"months"`
}

// MonthTotals — итоги календарного месяца "YYYY-MM"; крайние месяцы покрывают только дни внутри периода.
type MonthTotals struct {
	Month            string      `json:"month"`
	IncomeMinor      money.Minor `json:"income_minor"`
	ExpensesMinor    money.Minor `json:"expenses_minor"`
	NetMinor         money.Minor `json:"net_minor"`
	TransactionCount int         `json:"transaction_count"`
}

// CategoryShare — сумма по категории и её доля в сумме периода.
type CategoryShare struct {
	CategoryID       uuid.UUID   `json:"category_id"`
	Name             string      `json:"name"`
	Color            string      `json:"color,omitempty"`
	Icon             string      `json:"icon,omitempty"`
	AmountMinor      money.Minor `json:"amount_minor"`
	TransactionCount int         `json:"transaction_count"`
	Share            float64     `json:"share"`
}

// BudgetProgress — состояние активного бюджета.
// CategoryName пуст для бюджета без категории.
type BudgetProgress struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	CategoryName   string        `json:"category_name,omitempty"`
	AmountMinor    money.Minor   `json:"amount_minor"`
	SpentMinor     money.Minor   `json:"spent_minor"`
	RemainingMinor money.Minor   `json:"remaining_minor"`
	Utilization    float64       `json:"utilization"`
	Period         budget.Period `json:"period"`
	StartDate      date.Date     `json:"start_date"`
	EndDate        date.Date     `json:"end_date"`
	DaysRemaining  int           `json:"days_remaining"`
	IsActive       bool          `json:"is_active"`
	IsOverBudget   bool          `json:"is_over_budget"`
	IsNearLimit    bool          `json:"is_near_limit"`
}

// RecentTransaction — последняя операция.
// CategoryName пуст, если категория не найдена.
type RecentTransaction struct {
	ID           uuid.UUID        `json:"id"`
	Description  string           `json:"description"`
	AmountMinor  money.Minor      `json:"amount_minor"`
	Type         transaction.Type `json:"type"`
	CategoryName string           `json:"category_name,omitempty"`
	Date         date.Date        `json:"date"`
	CreatedAt    time.Time        `json:"created_at"`
}

// StatsNetWorth — помесячный ряд капитала за период [From, To] для GET /stats/net-worth.
type StatsNetWorth struct {
	From   date.Date       `json:"from"`
	To     date.Date       `json:"to"`
	Months []NetWorthMonth `json:"months"`
}

// NetWorthMonth — активы и пассивы на конец месяца "YYYY-MM" (на To — для последнего); суммы
// нескольких позиций, поэтому могут превышать money.MaxAmount.
type NetWorthMonth struct {
	Month            string      `json:"month"`
	AssetsMinor      money.Minor `json:"assets_minor"`
	LiabilitiesMinor money.Minor `json:"liabilities_minor"`
	NetMinor         money.Minor `json:"net_minor"`
}
