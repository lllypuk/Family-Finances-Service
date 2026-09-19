package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/money"
)

// ReconciliationStats — остатки против операций за месяц `YYYY-MM` для GET /stats/reconciliation.
// Суммы остатков null, пока свой край заполнен не у всех счетов; gap — пока не заполнен любой.
type ReconciliationStats struct {
	Month        string              `json:"month"`
	OpeningMinor *money.Minor        `json:"opening_minor"`
	ClosingMinor *money.Minor        `json:"closing_minor"`
	IncomeMinor  money.Minor         `json:"income_minor"`
	ExpenseMinor money.Minor         `json:"expense_minor"`
	GapMinor     *money.Minor        `json:"gap_minor"`
	Complete     bool                `json:"complete"`
	Accounts     []ReconciliationRow `json:"accounts"`
}

// ReconciliationRow — остатки счёта на начало и конец месяца; updated_at — у остатка на конец.
type ReconciliationRow struct {
	Account      ReconciliationAccount `json:"account"`
	OpeningMinor *money.Minor          `json:"opening_minor"`
	ClosingMinor *money.Minor          `json:"closing_minor"`
	UpdatedAt    *time.Time            `json:"updated_at"`
}

// ReconciliationAccount — форма Account из openapi, чтобы клиент не заводил вторую модель счёта.
type ReconciliationAccount struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	IsArchived bool      `json:"is_archived"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
