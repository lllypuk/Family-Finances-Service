// Package reconciliation — цифра банка «траты за месяц» по счёту; записанное считается на чтении.
package reconciliation

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/money"
)

var (
	ErrNotFound = errors.New("reconciliation not found")
	// ErrAmountOutOfRange — сумма банка вне 0…money.MaxAmount.
	ErrAmountOutOfRange = errors.New("bank expense amount is out of range")
)

// Reconciliation — сверка счёта за месяц `YYYY-MM`.
type Reconciliation struct {
	AccountID        uuid.UUID
	Month            string
	BankExpenseMinor money.Minor
	Note             string
	UpdatedAt        time.Time
}

// ValidAmount — 0 законен: по счёту в месяце могло не быть трат.
func ValidAmount(amount money.Minor) bool {
	return amount >= 0 && amount <= money.MaxAmount
}
