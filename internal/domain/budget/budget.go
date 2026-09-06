package budget

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

type Budget struct {
	ID          uuid.UUID   `json:"id"           bson:"_id"`
	Name        string      `json:"name"         bson:"name"`
	AmountMinor money.Minor `json:"amount_minor" bson:"amount_minor"` // Лимит бюджета
	SpentMinor  money.Minor `json:"spent_minor"  bson:"spent_minor"`  // Потрачено
	Period      Period      `json:"period"       bson:"period"`
	CategoryID  *uuid.UUID  `json:"category_id"  bson:"category_id,omitempty"` // Для конкретной категории
	StartDate   date.Date   `json:"start_date"   bson:"start_date"`
	EndDate     date.Date   `json:"end_date"     bson:"end_date"`
	IsActive    bool        `json:"is_active"    bson:"is_active"`
	CreatedAt   time.Time   `json:"created_at"   bson:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"   bson:"updated_at"`
}

type Period string

const (
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
	PeriodCustom  Period = "custom"
)

func NewBudget(
	name string,
	amountMinor money.Minor,
	period Period,
	startDate, endDate date.Date,
) *Budget {
	return &Budget{
		ID:          uuid.New(),
		Name:        name,
		AmountMinor: amountMinor,
		SpentMinor:  0,
		Period:      period,
		StartDate:   startDate,
		EndDate:     endDate,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (b *Budget) GetRemainingAmount() money.Minor {
	return b.AmountMinor - b.SpentMinor
}

// GetSpentPercentage возвращает долю потраченного в процентах; при нулевом лимите — 0.
func (b *Budget) GetSpentPercentage() float64 {
	return b.SpentMinor.Percent(b.AmountMinor)
}

func (b *Budget) IsOverBudget() bool {
	return b.SpentMinor > b.AmountMinor
}

func (b *Budget) UpdateSpent(amountMinor money.Minor) {
	b.SpentMinor += amountMinor
	b.UpdatedAt = time.Now()
}
