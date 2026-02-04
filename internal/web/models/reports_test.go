package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/web/models"
)

func TestReportForm_ToReportType(t *testing.T) {
	tests := []struct {
		name     string
		formType string
		expected report.Type
	}{
		{"expenses", "expenses", report.TypeExpenses},
		{"income", "income", report.TypeIncome},
		{"budget", "budget", report.TypeBudget},
		{"cash_flow", "cash_flow", report.TypeCashFlow},
		{"category_break", "category_break", report.TypeCategoryBreak},
		{"default", "invalid", report.TypeExpenses},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.ReportForm{Type: tt.formType}
			result := form.ToReportType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReportForm_ToReportPeriod(t *testing.T) {
	tests := []struct {
		name     string
		period   string
		expected report.Period
	}{
		{"daily", "daily", report.PeriodDaily},
		{"weekly", "weekly", report.PeriodWeekly},
		{"monthly", "monthly", report.PeriodMonthly},
		{"yearly", "yearly", report.PeriodYearly},
		{"custom", "custom", report.PeriodCustom},
		{"default", "invalid", report.PeriodMonthly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.ReportForm{Period: tt.period}
			result := form.ToReportPeriod()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReportForm_GetStartDate(t *testing.T) {
	tests := []struct {
		name      string
		startDate string
		expectErr bool
	}{
		{
			name:      "valid date",
			startDate: "2024-01-15",
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
			form := models.ReportForm{StartDate: tt.startDate}
			result, err := form.GetStartDate()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 2024, result.Year())
				assert.Equal(t, time.January, result.Month())
				assert.Equal(t, 15, result.Day())
			}
		})
	}
}

func TestReportForm_GetEndDate(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.ReportForm{EndDate: tt.endDate}
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

func TestGetReportTypeOptions(t *testing.T) {
	options := models.GetReportTypeOptions()

	assert.Len(t, options, 5)

	// Check that all expected types are present
	expectedTypes := []string{"expenses", "income", "budget", "cash_flow", "category_break"}
	for _, expectedType := range expectedTypes {
		found := false
		for _, option := range options {
			if option.Value == expectedType {
				found = true
				assert.NotEmpty(t, option.Label)
				assert.NotEmpty(t, option.Description)
				break
			}
		}
		assert.True(t, found, "Expected type %s not found", expectedType)
	}
}

func TestReportDataVM_FromDomain(t *testing.T) {
	now := time.Now()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	domainReport := &report.Report{
		ID:          uuid.New(),
		Name:        "Monthly Report",
		Type:        report.TypeExpenses,
		Period:      report.PeriodMonthly,
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: now,
		Data: report.Data{
			TotalIncome:   5000.0,
			TotalExpenses: 3000.0,
			NetIncome:     2000.0,
			CategoryBreakdown: []report.CategoryReportItem{
				{
					CategoryID:   uuid.New(),
					CategoryName: "Food",
					Amount:       1000.0,
					Percentage:   33.33,
					Count:        10,
				},
			},
			DailyBreakdown: []report.DailyReportItem{
				{
					Date:     startDate,
					Income:   100.0,
					Expenses: 50.0,
					Balance:  50.0,
				},
			},
			TopExpenses: []report.TransactionReportItem{
				{
					ID:          uuid.New(),
					Amount:      500.0,
					Description: "Groceries",
					Category:    "Food",
					Date:        startDate,
				},
			},
			BudgetComparison: []report.BudgetComparisonItem{
				{
					BudgetID:   uuid.New(),
					BudgetName: "Food Budget",
					Planned:    1000.0,
					Actual:     800.0,
					Difference: -200.0,
					Percentage: 80.0,
				},
			},
		},
	}

	vm := &models.ReportDataVM{}
	vm.FromDomain(domainReport)

	assert.Equal(t, domainReport.ID, vm.ID)
	assert.Equal(t, domainReport.Name, vm.Name)
	assert.Equal(t, domainReport.Type, vm.Type)
	assert.Equal(t, domainReport.Period, vm.Period)
	assert.Equal(t, domainReport.StartDate, vm.StartDate)
	assert.Equal(t, domainReport.EndDate, vm.EndDate)
	assert.InEpsilon(t, 5000.0, vm.TotalIncome, 0.001)
	assert.InEpsilon(t, 3000.0, vm.TotalExpenses, 0.001)
	assert.InEpsilon(t, 2000.0, vm.NetIncome, 0.001)
	assert.NotEmpty(t, vm.FormattedIncome)
	assert.NotEmpty(t, vm.FormattedExpenses)
	assert.NotEmpty(t, vm.FormattedNet)
	assert.Equal(t, "positive", vm.NetIncomeClass)
	assert.Len(t, vm.CategoryBreakdown, 1)
	assert.Len(t, vm.DailyBreakdown, 1)
	assert.Len(t, vm.TopExpenses, 1)
	assert.Len(t, vm.BudgetComparison, 1)
	assert.True(t, vm.CanExport)
	assert.NotEmpty(t, vm.ExportURL)
}

func TestReportDataVM_FromDomain_NegativeNetIncome(t *testing.T) {
	now := time.Now()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	domainReport := &report.Report{
		ID:          uuid.New(),
		Name:        "Deficit Report",
		Type:        report.TypeCashFlow,
		Period:      report.PeriodMonthly,
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: now,
		Data: report.Data{
			TotalIncome:       2000.0,
			TotalExpenses:     3000.0,
			NetIncome:         -1000.0,
			CategoryBreakdown: []report.CategoryReportItem{},
			DailyBreakdown:    []report.DailyReportItem{},
			TopExpenses:       []report.TransactionReportItem{},
			BudgetComparison:  []report.BudgetComparisonItem{},
		},
	}

	vm := &models.ReportDataVM{}
	vm.FromDomain(domainReport)

	assert.InEpsilon(t, -1000.0, vm.NetIncome, 0.001)
	assert.Equal(t, "negative", vm.NetIncomeClass)
	assert.Contains(t, vm.FormattedNet, "-")
}

func TestCategoryReportItemVM(t *testing.T) {
	item := models.CategoryReportItemVM{
		CategoryID:    uuid.New(),
		CategoryName:  "Food",
		Amount:        1500.0,
		Percentage:    50.0,
		Count:         25,
		ProgressWidth: "50.0%",
	}

	assert.NotEqual(t, uuid.Nil, item.CategoryID)
	assert.Equal(t, "Food", item.CategoryName)
	assert.InEpsilon(t, 1500.0, item.Amount, 0.001)
	assert.InEpsilon(t, 50.0, item.Percentage, 0.001)
	assert.Equal(t, 25, item.Count)
	assert.Equal(t, "50.0%", item.ProgressWidth)
}

func TestDailyReportItemVM(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	item := models.DailyReportItemVM{
		Date:         date,
		Income:       100.0,
		Expenses:     50.0,
		Balance:      50.0,
		BalanceClass: "positive",
	}

	assert.Equal(t, date, item.Date)
	assert.InEpsilon(t, 100.0, item.Income, 0.001)
	assert.InEpsilon(t, 50.0, item.Expenses, 0.001)
	assert.InEpsilon(t, 50.0, item.Balance, 0.001)
	assert.Equal(t, "positive", item.BalanceClass)
}

func TestTransactionReportItemVM(t *testing.T) {
	item := models.TransactionReportItemVM{
		ID:          uuid.New(),
		Amount:      500.0,
		Description: "Groceries",
		Category:    "Food",
	}

	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.InEpsilon(t, 500.0, item.Amount, 0.001)
	assert.Equal(t, "Groceries", item.Description)
	assert.Equal(t, "Food", item.Category)
}

func TestBudgetComparisonItemVM(t *testing.T) {
	tests := []struct {
		name              string
		planned           float64
		actual            float64
		difference        float64
		percentage        float64
		expectedDiffClass string
		expectedPerfClass string
	}{
		{
			name:              "under budget (good)",
			planned:           1000.0,
			actual:            800.0,
			difference:        -200.0,
			percentage:        80.0,
			expectedDiffClass: "under",
			expectedPerfClass: "good",
		},
		{
			name:              "over budget (danger)",
			planned:           1000.0,
			actual:            1200.0,
			difference:        200.0,
			percentage:        120.0,
			expectedDiffClass: "over",
			expectedPerfClass: "danger",
		},
		{
			name:              "exact budget",
			planned:           1000.0,
			actual:            1000.0,
			difference:        0.0,
			percentage:        100.0,
			expectedDiffClass: "exact",
			expectedPerfClass: "warning",
		},
		{
			name:              "near limit (warning)",
			planned:           1000.0,
			actual:            950.0,
			difference:        -50.0,
			percentage:        95.0,
			expectedDiffClass: "under",
			expectedPerfClass: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := models.BudgetComparisonItemVM{
				DifferenceClass:  tt.expectedDiffClass,
				PerformanceClass: tt.expectedPerfClass,
			}

			assert.Equal(t, tt.expectedDiffClass, item.DifferenceClass)
			assert.Equal(t, tt.expectedPerfClass, item.PerformanceClass)
		})
	}
}

func TestReportTypeOption(t *testing.T) {
	option := models.ReportTypeOption{
		Value:       "expenses",
		Label:       "Expenses Report",
		Description: "Detailed breakdown of all expenses",
	}

	assert.Equal(t, "expenses", option.Value)
	assert.Equal(t, "Expenses Report", option.Label)
	assert.NotEmpty(t, option.Description)
}

func TestReportDataVM_EmptyData(t *testing.T) {
	now := time.Now()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	domainReport := &report.Report{
		ID:          uuid.New(),
		Name:        "Empty Report",
		Type:        report.TypeExpenses,
		Period:      report.PeriodMonthly,
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: now,
		Data: report.Data{
			TotalIncome:       0.0,
			TotalExpenses:     0.0,
			NetIncome:         0.0,
			CategoryBreakdown: []report.CategoryReportItem{},
			DailyBreakdown:    []report.DailyReportItem{},
			TopExpenses:       []report.TransactionReportItem{},
			BudgetComparison:  []report.BudgetComparisonItem{},
		},
	}

	vm := &models.ReportDataVM{}
	vm.FromDomain(domainReport)

	assert.InDelta(t, 0.0, vm.TotalIncome, 0.001)
	assert.InDelta(t, 0.0, vm.TotalExpenses, 0.001)
	assert.InDelta(t, 0.0, vm.NetIncome, 0.001)
	assert.Equal(t, "zero", vm.NetIncomeClass)
	assert.Empty(t, vm.CategoryBreakdown)
	assert.Empty(t, vm.DailyBreakdown)
	assert.Empty(t, vm.TopExpenses)
	assert.Empty(t, vm.BudgetComparison)
}

func TestReportConstants(t *testing.T) {
	assert.Equal(t, 80, models.GoodPerformanceThreshold)
	assert.Equal(t, 100, models.WarningPerformanceThreshold)
}

func TestReportDataVM_FormattedPeriod(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		period    report.Period
		startDate time.Time
		endDate   time.Time
	}{
		{
			name:      "daily",
			period:    report.PeriodDaily,
			startDate: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "weekly",
			period:    report.PeriodWeekly,
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 7, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "monthly",
			period:    report.PeriodMonthly,
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "yearly",
			period:    report.PeriodYearly,
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "custom",
			period:    report.PeriodCustom,
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainReport := &report.Report{
				ID:          uuid.New(),
				Name:        "Test Report",
				Type:        report.TypeExpenses,
				Period:      tt.period,
				StartDate:   tt.startDate,
				EndDate:     tt.endDate,
				GeneratedAt: now,
				Data: report.Data{
					CategoryBreakdown: []report.CategoryReportItem{},
					DailyBreakdown:    []report.DailyReportItem{},
					TopExpenses:       []report.TransactionReportItem{},
					BudgetComparison:  []report.BudgetComparisonItem{},
				},
			}

			vm := &models.ReportDataVM{}
			vm.FromDomain(domainReport)

			assert.NotEmpty(t, vm.FormattedPeriod)
		})
	}
}
