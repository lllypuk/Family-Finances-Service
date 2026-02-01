package report_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
)

func TestNewReport_Success(t *testing.T) {
	// Arrange
	name := "Monthly Expenses Report"
	reportType := report.TypeExpenses
	period := report.PeriodMonthly
	userID := uuid.New()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	// Act
	reportItem := report.NewReport(name, reportType, period, userID, startDate, endDate)

	// Assert
	require.NotNil(t, reportItem)
	assert.NotEqual(t, uuid.Nil, reportItem.ID)
	assert.Equal(t, name, reportItem.Name)
	assert.Equal(t, reportType, reportItem.Type)
	assert.Equal(t, period, reportItem.Period)
	assert.Equal(t, userID, reportItem.UserID)
	assert.Equal(t, startDate, reportItem.StartDate)
	assert.Equal(t, endDate, reportItem.EndDate)
	assert.False(t, reportItem.GeneratedAt.IsZero())

	// Проверяем что Data инициализирована пустой структурой
	assert.NotNil(t, reportItem.Data)
	assert.InDelta(t, 0.0, reportItem.Data.TotalIncome, 0.01)
	assert.InDelta(t, 0.0, reportItem.Data.TotalExpenses, 0.01)
	assert.InDelta(t, 0.0, reportItem.Data.NetIncome, 0.01)
	assert.Empty(t, reportItem.Data.CategoryBreakdown)
	assert.Empty(t, reportItem.Data.DailyBreakdown)
	assert.Empty(t, reportItem.Data.TopExpenses)
	assert.Empty(t, reportItem.Data.BudgetComparison)
}

func TestReportType_Constants(t *testing.T) {
	// Проверяем что все константы типов отчетов определены корректно
	assert.Equal(t, report.TypeExpenses, report.Type("expenses"))
	assert.Equal(t, report.TypeIncome, report.Type("income"))
	assert.Equal(t, report.TypeBudget, report.Type("budget"))
	assert.Equal(t, report.TypeCashFlow, report.Type("cash_flow"))
	assert.Equal(t, report.TypeCategoryBreak, report.Type("category_breakdown"))
}

func TestReportPeriod_Constants(t *testing.T) {
	// Проверяем что все константы периодов определены корректно
	assert.Equal(t, report.PeriodDaily, report.Period("daily"))
	assert.Equal(t, report.PeriodWeekly, report.Period("weekly"))
	assert.Equal(t, report.PeriodMonthly, report.Period("monthly"))
	assert.Equal(t, report.PeriodYearly, report.Period("yearly"))
	assert.Equal(t, report.PeriodCustom, report.Period("custom"))
}

func TestCategoryReportItem_Structure(t *testing.T) {
	categoryID := uuid.New()

	item := report.CategoryReportItem{
		CategoryID:   categoryID,
		CategoryName: "Groceries",
		Amount:       1500.75,
		Percentage:   35.5,
		Count:        25,
	}

	assert.Equal(t, categoryID, item.CategoryID)
	assert.Equal(t, "Groceries", item.CategoryName)
	assert.InDelta(t, 1500.75, item.Amount, 0.01)
	assert.InDelta(t, 35.5, item.Percentage, 0.01)
	assert.Equal(t, 25, item.Count)
}

func TestDailyReportItem_Structure(t *testing.T) {
	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	item := report.DailyReportItem{
		Date:     date,
		Income:   2500.00,
		Expenses: 800.50,
		Balance:  1699.50,
	}

	assert.Equal(t, date, item.Date)
	assert.InDelta(t, 2500.00, item.Income, 0.01)
	assert.InDelta(t, 800.50, item.Expenses, 0.01)
	assert.InDelta(t, 1699.50, item.Balance, 0.01)
}

func TestTransactionReportItem_Structure(t *testing.T) {
	transactionID := uuid.New()
	date := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)

	item := report.TransactionReportItem{
		ID:          transactionID,
		Amount:      125.50,
		Description: "Grocery shopping",
		Category:    "Groceries",
		Date:        date,
	}

	assert.Equal(t, transactionID, item.ID)
	assert.InDelta(t, 125.50, item.Amount, 0.01)
	assert.Equal(t, "Grocery shopping", item.Description)
	assert.Equal(t, "Groceries", item.Category)
	assert.Equal(t, date, item.Date)
}

func TestBudgetComparisonItem_Structure(t *testing.T) {
	budgetID := uuid.New()

	item := report.BudgetComparisonItem{
		BudgetID:   budgetID,
		BudgetName: "Monthly Groceries",
		Planned:    800.0,
		Actual:     650.75,
		Difference: 149.25,
		Percentage: 81.34,
	}

	assert.Equal(t, budgetID, item.BudgetID)
	assert.Equal(t, "Monthly Groceries", item.BudgetName)
	assert.InDelta(t, 800.0, item.Planned, 0.01)
	assert.InDelta(t, 650.75, item.Actual, 0.01)
	assert.InDelta(t, 149.25, item.Difference, 0.01)
	assert.InDelta(t, 81.34, item.Percentage, 0.01)
}
