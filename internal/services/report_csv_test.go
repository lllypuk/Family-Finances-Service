package services_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func exportSavedReport(t *testing.T, reportType report.Type, data report.Data) string {
	t.Helper()

	service, mockReportRepo, _, _, _, _ := setupReportService()
	saved := report.NewReport(
		"csv", reportType, report.PeriodMonthly, uuid.New(),
		date.Today(time.UTC), date.Today(time.UTC),
	)
	saved.Data = data
	mockReportRepo.On("GetByID", t.Context(), saved.ID).Return(saved, nil)

	csvBytes, err := service.ExportReport(t.Context(), saved.ID, "csv", dto.ExportOptionsDTO{})
	require.NoError(t, err)

	return string(csvBytes)
}

func TestReportService_ExportReport_IncomeCSVTotalFromIncome(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeIncome, report.Data{
		TotalIncomeMinor: 150_000,
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Зарплата", AmountMinor: 120_000, Percentage: 80, Count: 1},
			{CategoryName: "Подработка", AmountMinor: 30_000, Percentage: 20, Count: 2},
		},
	})

	assert.True(t, strings.HasPrefix(csvText, "\ufeff"), "CSV должен начинаться с BOM")
	assert.Contains(t, csvText, "Зарплата,120000,RUB,80.0%,1")
	assert.Contains(t, csvText, "TOTAL,150000,RUB,100.0%,")
}

func TestReportService_ExportReport_CategoryBreakdownTotalFromRows(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeCategoryBreak, report.Data{
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", AmountMinor: 40_000, Percentage: 40, Count: 3},
			{CategoryName: "Транспорт", AmountMinor: 60_000, Percentage: 60, Count: 1},
		},
	})

	assert.Contains(t, csvText, "TOTAL,100000,RUB,100.0%,")
}

func TestReportService_ExportReport_ExpensesCSVTotalFromExpenses(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeExpenses, report.Data{
		TotalExpensesMinor: 90_000,
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", AmountMinor: 90_000, Percentage: 100, Count: 4},
		},
	})

	assert.Contains(t, csvText, "TOTAL,90000,RUB,100.0%,")
}

func TestReportService_ExportReport_CashFlowCSVRows(t *testing.T) {
	day := date.New(2026, time.March, 2)
	csvText := exportSavedReport(t, report.TypeCashFlow, report.Data{
		DailyBreakdown: []report.DailyReportItem{
			{Date: day, IncomeMinor: 10_000, ExpensesMinor: 4_000, BalanceMinor: 6_000},
		},
	})

	assert.Contains(t, csvText, "Date,Currency,Income Minor,Expenses Minor,Balance Minor")
	assert.Contains(t, csvText, "2026-03-02,RUB,10000,4000,6000")
}

func TestReportService_ExportReport_BudgetCSVRows(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeBudget, report.Data{
		BudgetComparison: []report.BudgetComparisonItem{
			{
				BudgetName:      "Еда",
				PlannedMinor:    100_000,
				ActualMinor:     75_000,
				DifferenceMinor: 25_000,
				Percentage:      75,
			},
		},
	})

	assert.Contains(t, csvText, "Budget,Currency,Planned Minor,Actual Minor,Difference Minor,Percentage")
	assert.Contains(t, csvText, "Еда,RUB,100000,75000,25000,75.0%")
}

// ExportReportData работает с голыми данными: тип отчёта ей неизвестен,
// поэтому csv она собирает по общей ветке, а чужие данные отвергает.
func TestReportService_ExportReportData_Formats(t *testing.T) {
	service, _, _, _, _, _ := setupReportService()
	data := report.Data{
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", AmountMinor: 25_000, Percentage: 100, Count: 1},
		},
	}

	csvBytes, err := service.ExportReportData(t.Context(), data, "csv", dto.ExportOptionsDTO{Currency: "RUB"})
	require.NoError(t, err)
	assert.Contains(t, string(csvBytes), "TOTAL,25000,RUB,100.0%,")

	jsonBytes, err := service.ExportReportData(t.Context(), data, "json", dto.ExportOptionsDTO{})
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"category_breakdown"`)

	_, err = service.ExportReportData(t.Context(), "not a report", "csv", dto.ExportOptionsDTO{})
	require.ErrorIs(t, err, services.ErrUnsupportedCSVData)

	_, err = service.ExportReportData(t.Context(), data, "xml", dto.ExportOptionsDTO{})
	require.Error(t, err)
}

// Имена категорий и бюджетов приходят из БД: ведущий =/+/-/@ Excel считает формулой.
func TestReportService_ExportReportData_EscapesFormulaPrefix(t *testing.T) {
	service, _, _, _, _, _ := setupReportService()
	data := report.Data{
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: `=HYPERLINK("http://evil","x")`, AmountMinor: 100, Percentage: 100, Count: 1},
		},
	}

	csvBytes, err := service.ExportReportData(t.Context(), data, "csv", dto.ExportOptionsDTO{})
	require.NoError(t, err)
	assert.Contains(t, string(csvBytes), `"'=HYPERLINK(""http://evil"",""x"")"`)
}
