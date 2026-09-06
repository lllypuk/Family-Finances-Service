package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

func TestNewBudgetFilterDTO(t *testing.T) {
	filter := NewBudgetFilterDTO()

	assert.Equal(t, DefaultBudgetLimit, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestBudgetFilterDTO_ValidateDateRange_Valid(t *testing.T) {
	dateFrom := date.New(2024, time.January, 1)
	dateTo := date.New(2024, time.December, 31)

	filter := BudgetFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	assert.NoError(t, filter.ValidateDateRange())
}

func TestBudgetFilterDTO_ValidateDateRange_Invalid(t *testing.T) {
	dateFrom := date.New(2024, time.December, 31)
	dateTo := date.New(2024, time.January, 1)

	filter := BudgetFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	assert.Equal(t, ErrInvalidDateRange, filter.ValidateDateRange())
}

func TestBudgetFilterDTO_ValidateAmountRange_Valid(t *testing.T) {
	amountFrom := money.Minor(10_000)
	amountTo := money.Minor(100_000)

	filter := BudgetFilterDTO{
		AmountFromMinor: &amountFrom,
		AmountToMinor:   &amountTo,
	}

	assert.NoError(t, filter.ValidateAmountRange())
}

func TestBudgetFilterDTO_ValidateAmountRange_Invalid(t *testing.T) {
	amountFrom := money.Minor(100_000)
	amountTo := money.Minor(10_000)

	filter := BudgetFilterDTO{
		AmountFromMinor: &amountFrom,
		AmountToMinor:   &amountTo,
	}

	assert.Equal(t, ErrInvalidAmountRange, filter.ValidateAmountRange())
}

func TestCreateBudgetDTO_ValidatePeriod_Valid(t *testing.T) {
	dto := CreateBudgetDTO{
		Name:        "Monthly Budget",
		AmountMinor: 100_000,
		Period:      budget.PeriodMonthly,
		StartDate:   date.New(2024, time.January, 1),
		EndDate:     date.New(2024, time.January, 31),
	}

	assert.NoError(t, dto.ValidatePeriod())
}

func TestCreateBudgetDTO_ValidatePeriod_Invalid(t *testing.T) {
	dto := CreateBudgetDTO{
		Name:        "Invalid Budget",
		AmountMinor: 100_000,
		Period:      budget.PeriodMonthly,
		StartDate:   date.New(2024, time.January, 31),
		EndDate:     date.New(2024, time.January, 1),
	}

	assert.Equal(t, ErrInvalidBudgetPeriod, dto.ValidatePeriod())
}

func TestCreateBudgetDTO_ValidatePeriod_Equal(t *testing.T) {
	sameDate := date.New(2024, time.January, 15)

	dto := CreateBudgetDTO{
		Name:        "Equal Dates Budget",
		AmountMinor: 100_000,
		Period:      budget.PeriodMonthly,
		StartDate:   sameDate,
		EndDate:     sameDate,
	}

	assert.Equal(t, ErrInvalidBudgetPeriod, dto.ValidatePeriod())
}

func TestDetermineBudgetStatus(t *testing.T) {
	tests := []struct {
		name               string
		utilizationPercent float64
		expected           string
	}{
		{"healthy - under 70%", 50.0, BudgetStatusHealthy},
		{"warning - 80%", 80.0, BudgetStatusWarning},
		{"warning - 85%", 85.0, BudgetStatusWarning},
		{"critical - 90%", 90.0, BudgetStatusCritical},
		{"critical - 95%", 95.0, BudgetStatusCritical},
		{"exceeded - 100%", 100.0, BudgetStatusExceeded},
		{"exceeded - over 100%", 150.0, BudgetStatusExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetermineBudgetStatus(tt.utilizationPercent))
		})
	}
}

func TestCalculateDaysRemaining(t *testing.T) {
	today := date.Today(time.UTC)

	tests := []struct {
		name     string
		endDate  date.Date
		expected int
	}{
		{name: "future date", endDate: today.AddDays(10), expected: 10},
		{name: "today", endDate: today, expected: 0},
		{name: "past date", endDate: today.AddDays(-5), expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, CalculateDaysRemaining(tt.endDate))
		})
	}
}

func TestBudgetFilterDTO_ComplexFilter(t *testing.T) {
	categoryID := uuid.New()
	period := budget.PeriodMonthly
	isActive := true
	activeOn := date.New(2024, time.January, 15)
	isOverBudget := false

	filter := BudgetFilterDTO{
		CategoryID:   &categoryID,
		Period:       &period,
		IsActive:     &isActive,
		ActiveOn:     &activeOn,
		IsOverBudget: &isOverBudget,
		Limit:        50,
		Offset:       10,
	}

	assert.NotNil(t, filter.CategoryID)
	assert.NotNil(t, filter.Period)
	assert.NotNil(t, filter.IsActive)
	assert.Equal(t, 50, filter.Limit)
	assert.Equal(t, 10, filter.Offset)
}

func testTime() time.Time {
	return time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
}
