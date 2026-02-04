package models_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/web/models"
)

func TestMonthlySummaryCard_GetIncomeChangeClass(t *testing.T) {
	tests := []struct {
		name         string
		incomeChange float64
		expected     string
	}{
		{"positive change", 10.0, models.CSSClassTextSuccess},
		{"negative change", -5.0, models.CSSClassTextDanger},
		{"no change", 0.0, models.CSSClassTextMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &models.MonthlySummaryCard{IncomeChange: tt.incomeChange}
			result := card.GetIncomeChangeClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMonthlySummaryCard_GetExpensesChangeClass(t *testing.T) {
	tests := []struct {
		name           string
		expensesChange float64
		expected       string
	}{
		{"positive change (bad)", 10.0, models.CSSClassTextDanger},
		{"negative change (good)", -5.0, models.CSSClassTextSuccess},
		{"no change", 0.0, models.CSSClassTextMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &models.MonthlySummaryCard{ExpensesChange: tt.expensesChange}
			result := card.GetExpensesChangeClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMonthlySummaryCard_GetNetIncomeClass(t *testing.T) {
	tests := []struct {
		name      string
		netIncome float64
		expected  string
	}{
		{"positive net income", 1000.0, models.CSSClassTextSuccess},
		{"negative net income", -500.0, models.CSSClassTextDanger},
		{"zero net income", 0.0, models.CSSClassTextMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &models.MonthlySummaryCard{NetIncome: tt.netIncome}
			result := card.GetNetIncomeClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBudgetProgressItem_GetProgressBarClass(t *testing.T) {
	tests := []struct {
		name         string
		isOverBudget bool
		isNearLimit  bool
		expected     string
	}{
		{"over budget", true, false, "progress-danger"},
		{"near limit", false, true, "progress-warning"},
		{"normal", false, false, "progress-success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.BudgetProgressItem{
				IsOverBudget: tt.isOverBudget,
				IsNearLimit:  tt.isNearLimit,
			}
			result := item.GetProgressBarClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBudgetProgressItem_GetAlertBadgeClass(t *testing.T) {
	tests := []struct {
		name       string
		alertLevel string
		expected   string
	}{
		{"danger", "danger", "badge-danger"},
		{"warning", "warning", "badge-warning"},
		{"success", "success", "badge-success"},
		{"default", "", "badge-success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.BudgetProgressItem{AlertLevel: tt.alertLevel}
			result := item.GetAlertBadgeClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecentTransactionItem_GetTypeClass(t *testing.T) {
	tests := []struct {
		name     string
		txType   transaction.Type
		expected string
	}{
		{"income", transaction.TypeIncome, "text-success"},
		{"expense", transaction.TypeExpense, "text-danger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.RecentTransactionItem{Type: tt.txType}
			result := item.GetTypeClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRecentTransactionItem_GetTypeIcon(t *testing.T) {
	tests := []struct {
		name     string
		txType   transaction.Type
		expected string
	}{
		{"income", transaction.TypeIncome, "💰"},
		{"expense", transaction.TypeExpense, "💸"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.RecentTransactionItem{Type: tt.txType}
			result := item.GetTypeIcon()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategoryInsightItem_GetPercentageClass(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{"high percentage", 35.0, "text-danger"},
		{"medium percentage", 20.0, "text-warning"},
		{"low percentage", 10.0, "text-success"},
		{"edge case high", 30.0, "text-danger"},
		{"edge case medium", 15.0, "text-warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &models.CategoryInsightItem{Percentage: tt.percentage}
			result := item.GetPercentageClass()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDashboardFilters_GetPeriodDates(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"current_month", "current_month"},
		{"last_month", "last_month"},
		{"last_3_months", "last_3_months"},
		{"last_6_months", "last_6_months"},
		{"current_year", "current_year"},
		{"default (empty)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &models.DashboardFilters{Period: tt.period}
			start, end := filters.GetPeriodDates()

			assert.False(t, start.IsZero())
			assert.False(t, end.IsZero())
			assert.True(t, start.Before(end) || start.Equal(end))
			assert.True(t, end.After(start) || end.Equal(start))
		})
	}
}

func TestDashboardFilters_GetPeriodDates_CurrentMonth(t *testing.T) {
	now := time.Now()
	filters := &models.DashboardFilters{Period: "current_month"}
	start, end := filters.GetPeriodDates()

	assert.Equal(t, now.Year(), start.Year())
	assert.Equal(t, now.Month(), start.Month())
	assert.Equal(t, 1, start.Day())
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, 0, start.Minute())

	// End should be end of current month
	assert.Equal(t, now.Year(), end.Year())
	assert.Equal(t, now.Month(), end.Month())
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
}

func TestDashboardFilters_ValidateCustomDateRange(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)
	twoYearsAgo := now.AddDate(-2, 0, 0)
	twoYearsFromNow := now.AddDate(2, 0, 0)

	tests := []struct {
		name      string
		startDate *time.Time
		endDate   *time.Time
		expectErr bool
		errCheck  func(error) bool
	}{
		{
			name:      "valid range",
			startDate: &yesterday,
			endDate:   &tomorrow,
			expectErr: false,
		},
		{
			name:      "end before start",
			startDate: &tomorrow,
			endDate:   &yesterday,
			expectErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, models.ErrInvalidDateRange)
			},
		},
		{
			name:      "range too large",
			startDate: &twoYearsAgo,
			endDate:   &twoYearsFromNow,
			expectErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, models.ErrDateRangeTooLarge)
			},
		},
		{
			name:      "nil dates",
			startDate: nil,
			endDate:   nil,
			expectErr: false,
		},
		{
			name:      "only start date",
			startDate: &now,
			endDate:   nil,
			expectErr: false,
		},
		{
			name:      "only end date",
			startDate: nil,
			endDate:   &now,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &models.DashboardFilters{
				StartDate: tt.startDate,
				EndDate:   tt.endDate,
			}

			err := filters.ValidateCustomDateRange()

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errCheck != nil {
					assert.True(t, tt.errCheck(err))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := models.NewValidationError("test error")
	assert.Error(t, err)
	assert.Equal(t, "test error", err.Error())

	// Test as ValidationError type
	var validationErr *models.ValidationError
	assert.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "test error", validationErr.Message)
}

func TestDashboardViewModel_Structure(t *testing.T) {
	vm := &models.DashboardViewModel{
		MonthlySummary: &models.MonthlySummaryCard{
			TotalIncome:   10000.0,
			TotalExpenses: 7000.0,
			NetIncome:     3000.0,
		},
		BudgetOverview: &models.BudgetOverviewCard{
			TotalBudgets:  5,
			ActiveBudgets: 3,
			OverBudget:    1,
		},
		RecentActivity: &models.RecentActivityCard{
			Transactions: []*models.RecentTransactionItem{},
			TotalCount:   0,
		},
		CategoryInsights: &models.CategoryInsightsCard{
			TopExpenseCategories: []*models.CategoryInsightItem{},
			TopIncomeCategories:  []*models.CategoryInsightItem{},
		},
	}

	assert.NotNil(t, vm.MonthlySummary)
	assert.NotNil(t, vm.BudgetOverview)
	assert.NotNil(t, vm.RecentActivity)
	assert.NotNil(t, vm.CategoryInsights)
	assert.Equal(t, 3000.0, vm.MonthlySummary.NetIncome)
}

func TestBudgetProgressItem_CompleteFields(t *testing.T) {
	now := time.Now()
	item := &models.BudgetProgressItem{
		ID:            uuid.New(),
		Name:          "Monthly Budget",
		CategoryName:  "Food",
		Amount:        1000.0,
		Spent:         750.0,
		Remaining:     250.0,
		Percentage:    75.0,
		Period:        budget.PeriodMonthly,
		StartDate:     now.AddDate(0, 0, -15),
		EndDate:       now.AddDate(0, 0, 15),
		DaysRemaining: 15,
		IsOverBudget:  false,
		IsNearLimit:   true,
		AlertLevel:    "warning",
	}

	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Equal(t, "Monthly Budget", item.Name)
	assert.Equal(t, 1000.0, item.Amount)
	assert.Equal(t, 750.0, item.Spent)
	assert.Equal(t, 250.0, item.Remaining)
	assert.Equal(t, 75.0, item.Percentage)
	assert.False(t, item.IsOverBudget)
	assert.True(t, item.IsNearLimit)
	assert.Equal(t, "warning", item.AlertLevel)
}

func TestRecentActivityCard_Pagination(t *testing.T) {
	card := &models.RecentActivityCard{
		Transactions: make([]*models.RecentTransactionItem, 5),
		TotalCount:   25,
		ShowingCount: 5,
		HasMoreData:  true,
		LastUpdated:  time.Now(),
	}

	assert.Len(t, card.Transactions, 5)
	assert.Equal(t, 25, card.TotalCount)
	assert.Equal(t, 5, card.ShowingCount)
	assert.True(t, card.HasMoreData)
	assert.False(t, card.LastUpdated.IsZero())
}

func TestEnhancedStatsCard(t *testing.T) {
	card := &models.EnhancedStatsCard{
		AvgIncomePerDay:          100.0,
		IncomeTransactionsCount:  10,
		AvgExpensePerDay:         75.0,
		ExpenseTransactionsCount: 15,
		AvgTransactionAmount:     50.0,
		SavingsRate:              25.0,
		Forecast: &models.ForecastData{
			ExpectedIncome:   3000.0,
			ExpectedExpenses: 2250.0,
			MonthEndBalance:  750.0,
			DaysRemaining:    10,
		},
	}

	assert.Equal(t, 100.0, card.AvgIncomePerDay)
	assert.Equal(t, 75.0, card.AvgExpensePerDay)
	assert.Equal(t, 25.0, card.SavingsRate)
	assert.NotNil(t, card.Forecast)
	assert.Equal(t, 750.0, card.Forecast.MonthEndBalance)
}

func TestCategoryInsightsCard(t *testing.T) {
	now := time.Now()
	card := &models.CategoryInsightsCard{
		TopExpenseCategories: []*models.CategoryInsightItem{
			{
				CategoryID:       uuid.New(),
				CategoryName:     "Food",
				Amount:           500.0,
				TransactionCount: 20,
				Percentage:       50.0,
			},
		},
		TopIncomeCategories: []*models.CategoryInsightItem{
			{
				CategoryID:       uuid.New(),
				CategoryName:     "Salary",
				Amount:           3000.0,
				TransactionCount: 1,
				Percentage:       100.0,
			},
		},
		PeriodStart:   now.AddDate(0, -1, 0),
		PeriodEnd:     now,
		TotalExpenses: 1000.0,
		TotalIncome:   3000.0,
	}

	assert.Len(t, card.TopExpenseCategories, 1)
	assert.Len(t, card.TopIncomeCategories, 1)
	assert.Equal(t, 1000.0, card.TotalExpenses)
	assert.Equal(t, 3000.0, card.TotalIncome)
	assert.True(t, card.PeriodStart.Before(card.PeriodEnd))
}

func TestDashboardConstants(t *testing.T) {
	assert.Equal(t, 10, models.MaxRecentTransactions)
	assert.Equal(t, 5, models.MaxTopBudgets)
	assert.Equal(t, 5, models.MaxTopCategories)
	assert.Equal(t, 80.0, models.BudgetNearLimitThreshold)
	assert.Equal(t, 100.0, models.BudgetOverLimitThreshold)
	assert.Equal(t, 30.0, models.CategoryHighPercentage)
	assert.Equal(t, 15.0, models.CategoryMediumPercentage)
	assert.Equal(t, 24, models.HoursInDay)
	assert.Equal(t, 7, models.DaysInWeek)
	assert.Equal(t, 30, models.DaysInMonth)
	assert.Equal(t, 365, models.DaysInYear)
	assert.Equal(t, 2, models.MaxPeriodYears)
	assert.Equal(t, 100.0, models.PercentageMultiplier)
}
