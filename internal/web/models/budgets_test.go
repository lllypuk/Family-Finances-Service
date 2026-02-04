package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/web/models"
)

func TestBudgetForm_GetAmount(t *testing.T) {
	tests := []struct {
		name      string
		amount    string
		expected  float64
		expectErr bool
	}{
		{
			name:      "valid integer",
			amount:    "1000",
			expected:  1000.0,
			expectErr: false,
		},
		{
			name:      "valid decimal",
			amount:    "1234.56",
			expected:  1234.56,
			expectErr: false,
		},
		{
			name:      "valid small amount",
			amount:    "0.99",
			expected:  0.99,
			expectErr: false,
		},
		{
			name:      "valid large amount",
			amount:    "999999.99",
			expected:  999999.99,
			expectErr: false,
		},
		{
			name:      "invalid - text",
			amount:    "invalid",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "invalid - empty",
			amount:    "",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetForm{Amount: tt.amount}
			result, err := form.GetAmount()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBudgetForm_ToBudgetPeriod(t *testing.T) {
	tests := []struct {
		name     string
		period   string
		expected budget.Period
	}{
		{"weekly", "weekly", budget.PeriodWeekly},
		{"monthly", "monthly", budget.PeriodMonthly},
		{"yearly", "yearly", budget.PeriodYearly},
		{"custom", "custom", budget.PeriodCustom},
		{"default", "invalid", budget.PeriodMonthly},
		{"empty", "", budget.PeriodMonthly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetForm{Period: tt.period}
			result := form.ToBudgetPeriod()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBudgetForm_GetCategoryID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name       string
		categoryID string
		expectNil  bool
	}{
		{
			name:       "valid UUID",
			categoryID: validUUID.String(),
			expectNil:  false,
		},
		{
			name:       "empty string",
			categoryID: "",
			expectNil:  true,
		},
		{
			name:       "invalid UUID",
			categoryID: "invalid-uuid",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetForm{CategoryID: tt.categoryID}
			result := form.GetCategoryID()

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, validUUID, *result)
			}
		})
	}
}

func TestBudgetForm_GetStartDate(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		expected  time.Time
		expectErr bool
	}{
		{
			name:      "valid date",
			startDate: "2024-01-15",
			expected:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "invalid format",
			startDate: "15-01-2024",
			expectErr: true,
		},
		{
			name:      "empty",
			startDate: "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetForm{StartDate: tt.startDate}
			result, err := form.GetStartDate()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Year(), result.Year())
				assert.Equal(t, tt.expected.Month(), result.Month())
				assert.Equal(t, tt.expected.Day(), result.Day())
			}
		})
	}
}

func TestBudgetForm_GetEndDate(t *testing.T) {
	tests := []struct {
		name      string
		endDate   string
		expectErr bool
	}{
		{
			name:      "valid date",
			endDate:   "2024-01-31",
			expectErr: false,
		},
		{
			name:      "invalid format",
			endDate:   "31-01-2024",
			expectErr: true,
		},
		{
			name:      "empty",
			endDate:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetForm{EndDate: tt.endDate}
			result, err := form.GetEndDate()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Should be end of day
				assert.Equal(t, 23, result.Hour())
				assert.Equal(t, 59, result.Minute())
				assert.Equal(t, 59, result.Second())
			}
		})
	}
}

func TestBudgetAlertForm_GetBudgetID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name      string
		budgetID  string
		expectErr bool
	}{
		{
			name:      "valid UUID",
			budgetID:  validUUID.String(),
			expectErr: false,
		},
		{
			name:      "invalid UUID",
			budgetID:  "invalid-uuid",
			expectErr: true,
		},
		{
			name:      "empty",
			budgetID:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetAlertForm{BudgetID: tt.budgetID}
			result, err := form.GetBudgetID()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, validUUID, result)
			}
		})
	}
}

func TestBudgetAlertForm_GetThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold string
		expected  float64
		expectErr bool
	}{
		{
			name:      "valid threshold",
			threshold: "80",
			expected:  80.0,
			expectErr: false,
		},
		{
			name:      "valid decimal threshold",
			threshold: "75.5",
			expected:  75.5,
			expectErr: false,
		},
		{
			name:      "invalid - text",
			threshold: "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BudgetAlertForm{Threshold: tt.threshold}
			result, err := form.GetThreshold()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBudgetProgressVM_FromDomain(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -15) // 15 days ago
	endDate := now.AddDate(0, 0, 15)    // 15 days from now
	categoryID := uuid.New()

	testBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       "Test Budget",
		Amount:     1000.0,
		Spent:      750.0,
		Period:     budget.PeriodMonthly,
		CategoryID: &categoryID,
		StartDate:  startDate,
		EndDate:    endDate,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	vm := &models.BudgetProgressVM{}
	vm.FromDomain(testBudget)

	assert.Equal(t, testBudget.ID, vm.ID)
	assert.Equal(t, testBudget.Name, vm.Name)
	assert.Equal(t, testBudget.Amount, vm.Amount)
	assert.Equal(t, testBudget.Spent, vm.Spent)
	assert.InDelta(t, 250.0, vm.Remaining, 0.01)
	assert.InDelta(t, 75.0, vm.Percentage, 0.01)
	assert.Equal(t, testBudget.Period, vm.Period)
	assert.Equal(t, testBudget.CategoryID, vm.CategoryID)
	assert.Equal(t, testBudget.StartDate, vm.StartDate)
	assert.Equal(t, testBudget.EndDate, vm.EndDate)
	assert.Equal(t, testBudget.IsActive, vm.IsActive)
	assert.False(t, vm.IsOverBudget)
	assert.Positive(t, vm.DaysLeft)
	assert.Positive(t, vm.DaysElapsed)
	assert.NotEmpty(t, vm.FormattedAmount)
	assert.NotEmpty(t, vm.FormattedSpent)
	assert.NotEmpty(t, vm.FormattedRemaining)
}

func TestBudgetProgressVM_FromDomain_OverBudget(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -15)
	endDate := now.AddDate(0, 0, 15)

	testBudget := &budget.Budget{
		ID:        uuid.New(),
		Name:      "Over Budget",
		Amount:    1000.0,
		Spent:     1500.0, // Over budget
		Period:    budget.PeriodMonthly,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	vm := &models.BudgetProgressVM{}
	vm.FromDomain(testBudget)

	assert.True(t, vm.IsOverBudget)
	assert.Equal(t, -500.0, vm.Remaining)
	assert.InDelta(t, 150.0, vm.Percentage, 0.01)
	assert.NotEmpty(t, vm.FormattedOverage)
	assert.Contains(t, vm.ProgressBarClass, "danger")
	assert.Equal(t, "danger", vm.AlertLevel)
}

func TestBudgetProgressVM_FromDomain_ExpiredBudget(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -60) // 60 days ago
	endDate := now.AddDate(0, 0, -30)   // 30 days ago (expired)

	testBudget := &budget.Budget{
		ID:        uuid.New(),
		Name:      "Expired Budget",
		Amount:    1000.0,
		Spent:     500.0,
		Period:    budget.PeriodMonthly,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  false,
		CreatedAt: startDate,
		UpdatedAt: now,
	}

	vm := &models.BudgetProgressVM{}
	vm.FromDomain(testBudget)

	assert.Equal(t, 0, vm.DaysLeft) // Expired, no days left
	assert.Positive(t, vm.DaysElapsed)
	assert.False(t, vm.IsActive)
}

func TestBudgetAlertVM_FromDomainAlert(t *testing.T) {
	now := time.Now()
	budgetID := uuid.New()

	alert := &budget.Alert{
		ID:          uuid.New(),
		BudgetID:    budgetID,
		Threshold:   80.0,
		IsTriggered: true,
		TriggeredAt: &now,
	}

	vm := &models.BudgetAlertVM{}
	vm.FromDomainAlert(alert, "Test Budget")

	assert.Equal(t, alert.ID, vm.ID)
	assert.Equal(t, alert.BudgetID, vm.BudgetID)
	assert.Equal(t, "Test Budget", vm.BudgetName)
	assert.Equal(t, alert.Threshold, vm.Threshold)
	assert.Equal(t, alert.IsTriggered, vm.IsTriggered)
	assert.Equal(t, alert.TriggeredAt, vm.TriggeredAt)
	assert.NotEmpty(t, vm.Message)
	assert.NotEmpty(t, vm.AlertClass)
}

func TestBudgetAlertVM_FromDomainAlert_NotTriggered(t *testing.T) {
	budgetID := uuid.New()

	alert := &budget.Alert{
		ID:          uuid.New(),
		BudgetID:    budgetID,
		Threshold:   80.0,
		IsTriggered: false,
		TriggeredAt: nil,
	}

	vm := &models.BudgetAlertVM{}
	vm.FromDomainAlert(alert, "Test Budget")

	assert.False(t, vm.IsTriggered)
	assert.Nil(t, vm.TriggeredAt)
	assert.Contains(t, vm.Message, "will trigger")
}
