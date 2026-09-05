package services_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func exportSavedReport(t *testing.T, reportType report.Type, data report.Data) string {
	t.Helper()

	service, mockReportRepo, _, _, _, _ := setupReportService()
	saved := report.NewReport("csv", reportType, report.PeriodMonthly, uuid.New(), time.Now(), time.Now())
	saved.Data = data
	mockReportRepo.On("GetByID", t.Context(), saved.ID).Return(saved, nil)

	csvBytes, err := service.ExportReport(t.Context(), saved.ID, "csv", dto.ExportOptionsDTO{})
	require.NoError(t, err)

	return string(csvBytes)
}

func TestReportService_ExportReport_IncomeCSVTotalFromIncome(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeIncome, report.Data{
		TotalIncome: 1500,
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Зарплата", Amount: 1200, Percentage: 80, Count: 1},
			{CategoryName: "Подработка", Amount: 300, Percentage: 20, Count: 2},
		},
	})

	assert.True(t, strings.HasPrefix(csvText, "\ufeff"), "CSV должен начинаться с BOM")
	assert.Contains(t, csvText, "Зарплата,1200.00,80.0%,1")
	assert.Contains(t, csvText, "TOTAL,1500.00,100.0%,")
}

func TestReportService_ExportReport_CategoryBreakdownTotalFromRows(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeCategoryBreak, report.Data{
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", Amount: 400, Percentage: 40, Count: 3},
			{CategoryName: "Транспорт", Amount: 600, Percentage: 60, Count: 1},
		},
	})

	assert.Contains(t, csvText, "TOTAL,1000.00,100.0%,")
}

func TestReportService_ExportReport_ExpensesCSVTotalFromExpenses(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeExpenses, report.Data{
		TotalExpenses: 900,
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", Amount: 900, Percentage: 100, Count: 4},
		},
	})

	assert.Contains(t, csvText, "TOTAL,900.00,100.0%,")
}

func TestReportService_ExportReport_CashFlowCSVRows(t *testing.T) {
	day := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	csvText := exportSavedReport(t, report.TypeCashFlow, report.Data{
		DailyBreakdown: []report.DailyReportItem{
			{Date: day, Income: 100, Expenses: 40, Balance: 60},
		},
	})

	assert.Contains(t, csvText, "Date,Income,Expenses,Balance")
	assert.Contains(t, csvText, "2026-03-02,100.00,40.00,60.00")
}

func TestReportService_ExportReport_BudgetCSVRows(t *testing.T) {
	csvText := exportSavedReport(t, report.TypeBudget, report.Data{
		BudgetComparison: []report.BudgetComparisonItem{
			{BudgetName: "Еда", Planned: 1000, Actual: 750, Difference: 250, Percentage: 75},
		},
	})

	assert.Contains(t, csvText, "Budget,Planned,Actual,Difference,Percentage")
	assert.Contains(t, csvText, "Еда,1000.00,750.00,250.00,75.0%")
}

// ExportReportData работает с голыми данными: тип отчёта ей неизвестен,
// поэтому csv она собирает по общей ветке, а чужие данные отвергает.
func TestReportService_ExportReportData_Formats(t *testing.T) {
	service, _, _, _, _, _ := setupReportService()
	data := report.Data{
		CategoryBreakdown: []report.CategoryReportItem{
			{CategoryName: "Еда", Amount: 250, Percentage: 100, Count: 1},
		},
	}

	csvBytes, err := service.ExportReportData(t.Context(), data, "csv", dto.ExportOptionsDTO{})
	require.NoError(t, err)
	assert.Contains(t, string(csvBytes), "TOTAL,250.00,100.0%,")

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
			{CategoryName: `=HYPERLINK("http://evil","x")`, Amount: 1, Percentage: 100, Count: 1},
		},
	}

	csvBytes, err := service.ExportReportData(t.Context(), data, "csv", dto.ExportOptionsDTO{})
	require.NoError(t, err)
	assert.Contains(t, string(csvBytes), `"'=HYPERLINK(""http://evil"",""x"")"`)
}
