package budget

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

// ErrNameExists — нарушение UNIQUE (family_id, name, start_date, end_date): бюджет
// с таким именем на этот период уже есть.
var ErrNameExists = errors.New("budget with this name already exists for this period")

// ErrIDExists — нарушение PRIMARY KEY budgets.id. Живой бюджет с этим id вернул бы 200
// по идемпотентности (A-07), поэтому сюда доходит только id мягко удалённого бюджета:
// его строка занимает ключ, но клиенту не видна.
var ErrIDExists = errors.New("budget with this id already exists")

type Budget struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	AmountMinor money.Minor `json:"amount_minor"` // Лимит бюджета
	SpentMinor  money.Minor `json:"spent_minor"`  // Потрачено
	Period      Period      `json:"period"`
	CategoryID  *uuid.UUID  `json:"category_id"` // Для конкретной категории
	StartDate   date.Date   `json:"start_date"`
	EndDate     date.Date   `json:"end_date"`
	IsActive    bool        `json:"is_active"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Period string

const (
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
	PeriodCustom  Period = "custom"
)

func (b *Budget) GetRemainingAmount() money.Minor {
	return b.AmountMinor - b.SpentMinor
}

// GetSpentPercentage возвращает долю потраченного в процентах; при нулевом лимите — 0.
func (b *Budget) GetSpentPercentage() float64 {
	return b.SpentMinor.Percent(b.AmountMinor)
}

// GetSpentShare — та же доля, что GetSpentPercentage, но в единице 0..1, в которой
// доли уходят в API.
func (b *Budget) GetSpentShare() float64 {
	return b.SpentMinor.Share(b.AmountMinor)
}

func (b *Budget) IsOverBudget() bool {
	return b.SpentMinor > b.AmountMinor
}

func (b *Budget) UpdateSpent(amountMinor money.Minor) {
	b.SpentMinor += amountMinor
	b.UpdatedAt = time.Now()
}
