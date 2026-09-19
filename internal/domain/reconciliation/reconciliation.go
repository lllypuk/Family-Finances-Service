// Package reconciliation — остатки счетов на конец месяца; сверка с операциями считается на чтении.
package reconciliation

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/money"
)

var (
	// ErrBalanceNotFound — у счёта нет остатка за месяц.
	ErrBalanceNotFound = errors.New("account balance not found")
	// ErrBalanceOutOfRange — |остаток| больше money.MaxAmount.
	ErrBalanceOutOfRange = errors.New("account balance is out of range")
)

// Balance — остаток счёта на конец месяца `YYYY-MM`; отрицательный у кредитки.
type Balance struct {
	AccountID    uuid.UUID
	Month        string
	BalanceMinor money.Minor
	UpdatedAt    time.Time
}

// ValidBalance — |x| ≤ money.MaxAmount.
func ValidBalance(balance money.Minor) bool {
	return balance >= -money.MaxAmount && balance <= money.MaxAmount
}
