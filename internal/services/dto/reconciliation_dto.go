package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/money"
)

// ReconciliationStats — сверка за месяц `YYYY-MM` для GET /stats/reconciliation.
type ReconciliationStats struct {
	Month           string              `json:"month"`
	UnassignedMinor money.Minor         `json:"unassigned_minor"`
	Accounts        []ReconciliationRow `json:"accounts"`
}

// ReconciliationRow — строка счёта; пока сверки нет, поля банка, разницы, заметки и времени — null.
type ReconciliationRow struct {
	Account          ReconciliationAccount `json:"account"`
	RecordedMinor    money.Minor           `json:"recorded_minor"`
	BankExpenseMinor *money.Minor          `json:"bank_expense_minor"`
	DiffMinor        *money.Minor          `json:"diff_minor"`
	Note             *string               `json:"note"`
	UpdatedAt        *time.Time            `json:"updated_at"`
}

// ReconciliationAccount — форма Account из openapi, чтобы клиент не заводил вторую модель счёта.
type ReconciliationAccount struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	IsArchived bool      `json:"is_archived"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
