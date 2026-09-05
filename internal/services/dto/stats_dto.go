package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/transaction"
)

// StatsSummary — агрегаты за период для дашборда и GET /stats/summary.
// Доли (Delta, Share, Utilization) — именно доли: 0.12 = +12%; форматирование в процентах — на вызывающей стороне.
type StatsSummary struct {
	From              time.Time           `json:"from"`
	To                time.Time           `json:"to"`
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
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
	Income           float64   `json:"income"`
	Expenses         float64   `json:"expenses"`
	Net              float64   `json:"net"`
	TransactionCount int       `json:"transaction_count"`
}

// CategoryShare — сумма по категории и её доля в сумме периода.
type CategoryShare struct {
	CategoryID       uuid.UUID `json:"category_id"`
	Name             string    `json:"name"`
	Color            string    `json:"color,omitempty"`
	Icon             string    `json:"icon,omitempty"`
	Amount           float64   `json:"amount"`
	TransactionCount int       `json:"transaction_count"`
	Share            float64   `json:"share"`
}

// BudgetProgress — состояние активного бюджета.
// CategoryName пуст для бюджета без категории.
type BudgetProgress struct {
	ID            uuid.UUID     `json:"id"`
	Name          string        `json:"name"`
	CategoryName  string        `json:"category_name,omitempty"`
	Amount        float64       `json:"amount"`
	Spent         float64       `json:"spent"`
	Remaining     float64       `json:"remaining"`
	Utilization   float64       `json:"utilization"`
	Period        budget.Period `json:"period"`
	StartDate     time.Time     `json:"start_date"`
	EndDate       time.Time     `json:"end_date"`
	DaysRemaining int           `json:"days_remaining"`
	IsActive      bool          `json:"is_active"`
	IsOverBudget  bool          `json:"is_over_budget"`
	IsNearLimit   bool          `json:"is_near_limit"`
}

// RecentTransaction — последняя операция.
// CategoryName пуст, если категория не найдена.
type RecentTransaction struct {
	ID           uuid.UUID        `json:"id"`
	Description  string           `json:"description"`
	Amount       float64          `json:"amount"`
	Type         transaction.Type `json:"type"`
	CategoryName string           `json:"category_name,omitempty"`
	Date         time.Time        `json:"date"`
	CreatedAt    time.Time        `json:"created_at"`
}
