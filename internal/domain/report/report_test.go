package report_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
)

func TestNewReport_Success(t *testing.T) {
	// Arrange
	name := "Monthly Expenses Report"
	reportType := report.TypeExpenses
	period := report.PeriodMonthly
	userID := uuid.New()
	startDate := date.New(2025, time.January, 1)
	endDate := date.New(2025, time.January, 31)

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
	assert.Equal(t, money.Minor(0), reportItem.Data.TotalIncomeMinor)
	assert.Equal(t, money.Minor(0), reportItem.Data.TotalExpensesMinor)
	assert.Equal(t, money.Minor(0), reportItem.Data.NetIncomeMinor)
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
		AmountMinor:  150_075,
		Percentage:   35.5,
		Count:        25,
	}

	assert.Equal(t, categoryID, item.CategoryID)
	assert.Equal(t, "Groceries", item.CategoryName)
	assert.Equal(t, money.Minor(150_075), item.AmountMinor)
	assert.InDelta(t, 35.5, item.Percentage, 0.01)
	assert.Equal(t, 25, item.Count)
}

func TestDailyReportItem_Structure(t *testing.T) {
	on := date.New(2025, time.January, 15)

	item := report.DailyReportItem{
		Date:          on,
		IncomeMinor:   250_000,
		ExpensesMinor: 80_050,
		BalanceMinor:  169_950,
	}

	assert.Equal(t, on, item.Date)
	assert.Equal(t, money.Minor(250_000), item.IncomeMinor)
	assert.Equal(t, money.Minor(80_050), item.ExpensesMinor)
	assert.Equal(t, money.Minor(169_950), item.BalanceMinor)
}

func TestTransactionReportItem_Structure(t *testing.T) {
	transactionID := uuid.New()
	on := date.New(2025, time.January, 15)

	item := report.TransactionReportItem{
		ID:          transactionID,
		AmountMinor: 12_550,
		Description: "Grocery shopping",
		Category:    "Groceries",
		Date:        on,
	}

	assert.Equal(t, transactionID, item.ID)
	assert.Equal(t, money.Minor(12_550), item.AmountMinor)
	assert.Equal(t, "Grocery shopping", item.Description)
	assert.Equal(t, "Groceries", item.Category)
	assert.Equal(t, on, item.Date)
}

func TestBudgetComparisonItem_Structure(t *testing.T) {
	budgetID := uuid.New()

	item := report.BudgetComparisonItem{
		BudgetID:        budgetID,
		BudgetName:      "Monthly Groceries",
		PlannedMinor:    80_000,
		ActualMinor:     65_075,
		DifferenceMinor: 14_925,
		Percentage:      81.34,
	}

	assert.Equal(t, budgetID, item.BudgetID)
	assert.Equal(t, "Monthly Groceries", item.BudgetName)
	assert.Equal(t, money.Minor(80_000), item.PlannedMinor)
	assert.Equal(t, money.Minor(65_075), item.ActualMinor)
	assert.Equal(t, money.Minor(14_925), item.DifferenceMinor)
	assert.InDelta(t, 81.34, item.Percentage, 0.01)
}
