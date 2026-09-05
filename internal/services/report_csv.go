package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"family-budget-service/internal/domain/report"
)

// csvUTF8BOM — Excel распознаёт UTF-8 в CSV только по BOM (docs/api/openapi.yaml).
const csvUTF8BOM = "\ufeff"

const csvDateLayout = "2006-01-02"

// csvFormulaPrefixes — с этих символов Excel/LibreOffice/Sheets начинают разбирать ячейку
// как формулу, поэтому имена из БД экранируются апострофом (CWE-1236).
const csvFormulaPrefixes = "=+-@\t\r"

func csvSafeText(value string) string {
	if value == "" || !strings.ContainsRune(csvFormulaPrefixes, rune(value[0])) {
		return value
	}

	return "'" + value
}

// reportToCSV собирает CSV отчёта; набор колонок зависит от типа отчёта.
func reportToCSV(data report.Data, reportType report.Type) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(csvUTF8BOM)

	writer := csv.NewWriter(&buf)

	var err error
	switch reportType {
	case report.TypeCashFlow:
		err = writeDailyBreakdownCSV(writer, data)
	case report.TypeBudget:
		err = writeBudgetComparisonCSV(writer, data)
	case report.TypeExpenses, report.TypeIncome, report.TypeCategoryBreak:
		err = writeCategoryBreakdownCSV(writer, data, reportType)
	default:
		err = writeCategoryBreakdownCSV(writer, data, reportType)
	}
	if err != nil {
		return nil, err
	}

	writer.Flush()
	if flushErr := writer.Error(); flushErr != nil {
		return nil, fmt.Errorf("failed to write csv: %w", flushErr)
	}

	return buf.Bytes(), nil
}

func writeCategoryBreakdownCSV(writer *csv.Writer, data report.Data, reportType report.Type) error {
	if err := writer.Write([]string{"Category", "Amount", "Percentage", "Transaction Count"}); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.CategoryBreakdown {
		row := []string{
			csvSafeText(item.CategoryName),
			fmt.Sprintf("%.2f", item.Amount),
			fmt.Sprintf("%.1f%%", item.Percentage),
			strconv.Itoa(item.Count),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	total := []string{"TOTAL", fmt.Sprintf("%.2f", categoryBreakdownTotal(data, reportType)), "100.0%", ""}
	if err := writer.Write(total); err != nil {
		return fmt.Errorf("failed to write csv row: %w", err)
	}

	return nil
}

// categoryBreakdownTotal выбирает итог по типу отчёта: income-отчёт заполняет только
// TotalIncome, category_breakdown — ни одного из них, там итог считается по строкам.
func categoryBreakdownTotal(data report.Data, reportType report.Type) float64 {
	switch reportType {
	case report.TypeIncome:
		return data.TotalIncome
	case report.TypeExpenses, report.TypeBudget, report.TypeCashFlow:
		return data.TotalExpenses
	case report.TypeCategoryBreak:
		return sumCategoryAmounts(data)
	default:
		return sumCategoryAmounts(data)
	}
}

func sumCategoryAmounts(data report.Data) float64 {
	total := 0.0
	for _, item := range data.CategoryBreakdown {
		total += item.Amount
	}

	return total
}

func writeDailyBreakdownCSV(writer *csv.Writer, data report.Data) error {
	if err := writer.Write([]string{"Date", "Income", "Expenses", "Balance"}); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.DailyBreakdown {
		row := []string{
			item.Date.Format(csvDateLayout),
			fmt.Sprintf("%.2f", item.Income),
			fmt.Sprintf("%.2f", item.Expenses),
			fmt.Sprintf("%.2f", item.Balance),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	return nil
}

func writeBudgetComparisonCSV(writer *csv.Writer, data report.Data) error {
	if err := writer.Write([]string{"Budget", "Planned", "Actual", "Difference", "Percentage"}); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.BudgetComparison {
		row := []string{
			csvSafeText(item.BudgetName),
			fmt.Sprintf("%.2f", item.Planned),
			fmt.Sprintf("%.2f", item.Actual),
			fmt.Sprintf("%.2f", item.Difference),
			fmt.Sprintf("%.1f%%", item.Percentage),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	return nil
}
