package budget

import (
	"time"

	"github.com/google/uuid"
)

type Budget struct {
	ID         uuid.UUID    `json:"id"          bson:"_id"`
	Name       string       `json:"name"        bson:"name"`
	Amount     float64      `json:"amount"      bson:"amount"` // Лимит бюджета
	Spent      float64      `json:"spent"       bson:"spent"`  // Потрачено
	Period     BudgetPeriod `json:"period"      bson:"period"`
	CategoryID *uuid.UUID   `json:"category_id" bson:"category_id,omitempty"` // Для конкретной категории
	FamilyID   uuid.UUID    `json:"family_id"   bson:"family_id"`
	StartDate  time.Time    `json:"start_date"  bson:"start_date"`
	EndDate    time.Time    `json:"end_date"    bson:"end_date"`
	IsActive   bool         `json:"is_active"   bson:"is_active"`
	CreatedAt  time.Time    `json:"created_at"  bson:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"  bson:"updated_at"`
}

type BudgetPeriod string

const (
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
	BudgetPeriodYearly  BudgetPeriod = "yearly"
	BudgetPeriodCustom  BudgetPeriod = "custom"
)

type BudgetAlert struct {
	ID          uuid.UUID  `json:"id"           bson:"_id"`
	BudgetID    uuid.UUID  `json:"budget_id"    bson:"budget_id"`
	Threshold   float64    `json:"threshold"    bson:"threshold"` // Процент (50, 80, 100)
	IsTriggered bool       `json:"is_triggered" bson:"is_triggered"`
	TriggeredAt *time.Time `json:"triggered_at" bson:"triggered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"   bson:"created_at"`
}

func NewBudget(
	name string,
	amount float64,
	period BudgetPeriod,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) *Budget {
	return &Budget{
		ID:        uuid.New(),
		Name:      name,
		Amount:    amount,
		Spent:     0,
		Period:    period,
		FamilyID:  familyID,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (b *Budget) GetRemainingAmount() float64 {
	return b.Amount - b.Spent
}

func (b *Budget) GetSpentPercentage() float64 {
	if b.Amount == 0 {
		return 0
	}
	return (b.Spent / b.Amount) * 100
}

func (b *Budget) IsOverBudget() bool {
	return b.Spent > b.Amount
}

func (b *Budget) UpdateSpent(amount float64) {
	b.Spent += amount
	b.UpdatedAt = time.Now()
}
