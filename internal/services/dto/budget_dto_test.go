package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/budget"
)

func TestNewBudgetFilterDTO(t *testing.T) {
	filter := NewBudgetFilterDTO()

	assert.Equal(t, DefaultBudgetLimit, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
	assert.NotNil(t, filter.SortBy)
	assert.Equal(t, "created_at", *filter.SortBy)
	assert.NotNil(t, filter.SortOrder)
	assert.Equal(t, "desc", *filter.SortOrder)
}

func TestBudgetFilterDTO_ValidateDateRange_Valid(t *testing.T) {
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	filter := BudgetFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	err := filter.ValidateDateRange()
	assert.NoError(t, err)
}

func TestBudgetFilterDTO_ValidateDateRange_Invalid(t *testing.T) {
	dateFrom := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	filter := BudgetFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	err := filter.ValidateDateRange()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidDateRange, err)
}

func TestBudgetFilterDTO_ValidateAmountRange_Valid(t *testing.T) {
	amountFrom := 100.0
	amountTo := 1000.0

	filter := BudgetFilterDTO{
		AmountFrom: &amountFrom,
		AmountTo:   &amountTo,
	}

	err := filter.ValidateAmountRange()
	assert.NoError(t, err)
}

func TestBudgetFilterDTO_ValidateAmountRange_Invalid(t *testing.T) {
	amountFrom := 1000.0
	amountTo := 100.0

	filter := BudgetFilterDTO{
		AmountFrom: &amountFrom,
		AmountTo:   &amountTo,
	}

	err := filter.ValidateAmountRange()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAmountRange, err)
}

func TestCreateBudgetDTO_ValidatePeriod_Valid(t *testing.T) {
	dto := CreateBudgetDTO{
		Name:      "Monthly Budget",
		Amount:    1000.00,
		Period:    budget.PeriodMonthly,
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
	}

	err := dto.ValidatePeriod()
	assert.NoError(t, err)
}

func TestCreateBudgetDTO_ValidatePeriod_Invalid(t *testing.T) {
	dto := CreateBudgetDTO{
		Name:      "Invalid Budget",
		Amount:    1000.00,
		Period:    budget.PeriodMonthly,
		StartDate: time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	err := dto.ValidatePeriod()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidBudgetPeriod, err)
}

func TestCreateBudgetDTO_ValidatePeriod_Equal(t *testing.T) {
	sameDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	dto := CreateBudgetDTO{
		Name:      "Equal Dates Budget",
		Amount:    1000.00,
		Period:    budget.PeriodMonthly,
		StartDate: sameDate,
		EndDate:   sameDate,
	}

	err := dto.ValidatePeriod()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidBudgetPeriod, err)
}

func TestUpdateBudgetDTO_ValidatePeriod_Valid(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	dto := UpdateBudgetDTO{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	err := dto.ValidatePeriod()
	assert.NoError(t, err)
}

func TestUpdateBudgetDTO_ValidatePeriod_Invalid(t *testing.T) {
	startDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	dto := UpdateBudgetDTO{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	err := dto.ValidatePeriod()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidBudgetPeriod, err)
}

func TestUpdateBudgetDTO_ValidatePeriod_OnlyStartDate(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	dto := UpdateBudgetDTO{
		StartDate: &startDate,
	}

	err := dto.ValidatePeriod()
	assert.NoError(t, err)
}

func TestCalculateUtilizationPercent(t *testing.T) {
	tests := []struct {
		name     string
		spent    float64
		amount   float64
		expected float64
	}{
		{
			name:     "50% utilized",
			spent:    500.00,
			amount:   1000.00,
			expected: 50.0,
		},
		{
			name:     "100% utilized",
			spent:    1000.00,
			amount:   1000.00,
			expected: 100.0,
		},
		{
			name:     "over budget",
			spent:    1500.00,
			amount:   1000.00,
			expected: 150.0,
		},
		{
			name:     "zero amount",
			spent:    100.00,
			amount:   0.00,
			expected: 0.0,
		},
		{
			name:     "zero spent",
			spent:    0.00,
			amount:   1000.00,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateUtilizationPercent(tt.spent, tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineBudgetStatus(t *testing.T) {
	tests := []struct {
		name               string
		utilizationPercent float64
		expected           string
	}{
		{
			name:               "healthy - under 70%",
			utilizationPercent: 50.0,
			expected:           BudgetStatusHealthy,
		},
		{
			name:               "warning - 80%",
			utilizationPercent: 80.0,
			expected:           BudgetStatusWarning,
		},
		{
			name:               "warning - 85%",
			utilizationPercent: 85.0,
			expected:           BudgetStatusWarning,
		},
		{
			name:               "critical - 90%",
			utilizationPercent: 90.0,
			expected:           BudgetStatusCritical,
		},
		{
			name:               "critical - 95%",
			utilizationPercent: 95.0,
			expected:           BudgetStatusCritical,
		},
		{
			name:               "exceeded - 100%",
			utilizationPercent: 100.0,
			expected:           BudgetStatusExceeded,
		},
		{
			name:               "exceeded - over 100%",
			utilizationPercent: 150.0,
			expected:           BudgetStatusExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineBudgetStatus(tt.utilizationPercent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateDaysRemaining(t *testing.T) {
	tests := []struct {
		name        string
		endDate     time.Time
		expectRange struct{ min, max int }
	}{
		{
			name:        "future date",
			endDate:     time.Now().Add(10 * 24 * time.Hour),
			expectRange: struct{ min, max int }{9, 10},
		},
		{
			name:        "past date",
			endDate:     time.Now().Add(-5 * 24 * time.Hour),
			expectRange: struct{ min, max int }{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDaysRemaining(tt.endDate)
			assert.GreaterOrEqual(t, result, tt.expectRange.min)
			assert.LessOrEqual(t, result, tt.expectRange.max)
		})
	}
}

func TestCreateBudgetDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	dto := CreateBudgetDTO{
		Name:       "Monthly Food Budget",
		Amount:     1000.00,
		Period:     budget.PeriodMonthly,
		CategoryID: &categoryID,
		StartDate:  startDate,
		EndDate:    endDate,
	}

	assert.Equal(t, "Monthly Food Budget", dto.Name)
	assert.Equal(t, 1000.00, dto.Amount)
	assert.Equal(t, budget.PeriodMonthly, dto.Period)
	assert.NotNil(t, dto.CategoryID)
	assert.Equal(t, categoryID, *dto.CategoryID)
}

func TestUpdateBudgetDTO_AllFields(t *testing.T) {
	name := "Updated Budget"
	amount := 1500.00
	startDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	isActive := false

	dto := UpdateBudgetDTO{
		Name:      &name,
		Amount:    &amount,
		StartDate: &startDate,
		EndDate:   &endDate,
		IsActive:  &isActive,
	}

	assert.NotNil(t, dto.Name)
	assert.Equal(t, "Updated Budget", *dto.Name)
	assert.NotNil(t, dto.Amount)
	assert.Equal(t, 1500.00, *dto.Amount)
	assert.NotNil(t, dto.IsActive)
	assert.False(t, *dto.IsActive)
}

func TestBudgetResponseDTO_AllFields(t *testing.T) {
	now := time.Now()
	budgetID := uuid.New()
	categoryID := uuid.New()

	response := BudgetResponseDTO{
		ID:                 budgetID,
		Name:               "Monthly Budget",
		Amount:             1000.00,
		Spent:              500.00,
		Remaining:          500.00,
		Period:             "monthly",
		CategoryID:         &categoryID,
		StartDate:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:            time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		IsActive:           true,
		CreatedAt:          now,
		UpdatedAt:          now,
		UtilizationPercent: 50.0,
		DaysRemaining:      15,
		IsOverBudget:       false,
		IsNearLimit:        false,
	}

	assert.Equal(t, budgetID, response.ID)
	assert.Equal(t, 1000.00, response.Amount)
	assert.Equal(t, 500.00, response.Spent)
	assert.Equal(t, 500.00, response.Remaining)
	assert.Equal(t, 50.0, response.UtilizationPercent)
	assert.False(t, response.IsOverBudget)
}

func TestBudgetStatusDTO_AllFields(t *testing.T) {
	budgetID := uuid.New()

	status := BudgetStatusDTO{
		BudgetID:           budgetID,
		Name:               "Food Budget",
		TotalAmount:        1000.00,
		SpentAmount:        800.00,
		RemainingAmount:    200.00,
		UtilizationPercent: 80.0,
		DaysTotal:          30,
		DaysElapsed:        20,
		DaysRemaining:      10,
		IsOverBudget:       false,
		IsNearLimit:        true,
		IsCriticalLimit:    false,
		DailyBudget:        33.33,
		DailySpent:         40.00,
		ProjectedOverrun:   100.00,
		Status:             BudgetStatusWarning,
	}

	assert.Equal(t, budgetID, status.BudgetID)
	assert.Equal(t, 80.0, status.UtilizationPercent)
	assert.True(t, status.IsNearLimit)
	assert.Equal(t, BudgetStatusWarning, status.Status)
}

func TestBudgetAlertDTO_AllFields(t *testing.T) {
	now := time.Now()
	budgetID := uuid.New()

	alert := BudgetAlertDTO{
		BudgetID:   budgetID,
		BudgetName: "Monthly Budget",
		AlertType:  "near_limit",
		Threshold:  80.0,
		Current:    85.0,
		Message:    "Budget is near limit",
		Severity:   "warning",
		CreatedAt:  now,
	}

	assert.Equal(t, budgetID, alert.BudgetID)
	assert.Equal(t, "near_limit", alert.AlertType)
	assert.Equal(t, 80.0, alert.Threshold)
	assert.Equal(t, 85.0, alert.Current)
	assert.Equal(t, "warning", alert.Severity)
}

func TestBudgetFilterDTO_ComplexFilter(t *testing.T) {
	categoryID := uuid.New()
	period := budget.PeriodMonthly
	isActive := true
	activeOn := time.Now()
	isOverBudget := false
	sortBy := "amount"
	sortOrder := "asc"

	filter := BudgetFilterDTO{
		CategoryID:   &categoryID,
		Period:       &period,
		IsActive:     &isActive,
		ActiveOn:     &activeOn,
		IsOverBudget: &isOverBudget,
		Limit:        50,
		Offset:       10,
		SortBy:       &sortBy,
		SortOrder:    &sortOrder,
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
