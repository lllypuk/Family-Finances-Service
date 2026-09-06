package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
)

// csvUTF8BOM — Excel распознаёт UTF-8 в CSV только по BOM (docs/api/openapi.yaml).
const csvUTF8BOM = "\ufeff"

// csvFormulaPrefixes — с этих символов Excel/LibreOffice/Sheets начинают разбирать ячейку
// как формулу, поэтому имена из БД экранируются апострофом (CWE-1236).
const csvFormulaPrefixes = "=+-@\t\r"

func csvSafeText(value string) string {
	if value == "" || !strings.ContainsRune(csvFormulaPrefixes, rune(value[0])) {
		return value
	}

	return "'" + value
}

// csvCurrencyColumn — валюта вынесена в колонку, суммы в строках целые (A-05).
const csvCurrencyColumn = "Currency"

func csvMinor(amount money.Minor) string {
	const decimalBase = 10

	return strconv.FormatInt(int64(amount), decimalBase)
}

// reportToCSV собирает CSV отчёта; набор колонок зависит от типа отчёта.
// Суммы — целые минимальные единицы, валюта вынесена в отдельную колонку.
func reportToCSV(data report.Data, reportType report.Type, currency string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(csvUTF8BOM)

	writer := csv.NewWriter(&buf)

	var err error
	switch reportType {
	case report.TypeCashFlow:
		err = writeDailyBreakdownCSV(writer, data, currency)
	case report.TypeBudget:
		err = writeBudgetComparisonCSV(writer, data, currency)
	case report.TypeExpenses, report.TypeIncome, report.TypeCategoryBreak:
		err = writeCategoryBreakdownCSV(writer, data, reportType, currency)
	default:
		err = writeCategoryBreakdownCSV(writer, data, reportType, currency)
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

func writeCategoryBreakdownCSV(
	writer *csv.Writer,
	data report.Data,
	reportType report.Type,
	currency string,
) error {
	header := []string{"Category", "Amount Minor", csvCurrencyColumn, "Percentage", "Transaction Count"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.CategoryBreakdown {
		row := []string{
			csvSafeText(item.CategoryName),
			csvMinor(item.AmountMinor),
			currency,
			fmt.Sprintf("%.1f%%", item.Percentage),
			strconv.Itoa(item.Count),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	total := []string{
		"TOTAL",
		csvMinor(categoryBreakdownTotal(data, reportType)),
		currency,
		"100.0%",
		"",
	}
	if err := writer.Write(total); err != nil {
		return fmt.Errorf("failed to write csv row: %w", err)
	}

	return nil
}

// categoryBreakdownTotal выбирает итог по типу отчёта: income-отчёт заполняет только
// TotalIncome, category_breakdown — ни одного из них, там итог считается по строкам.
func categoryBreakdownTotal(data report.Data, reportType report.Type) money.Minor {
	switch reportType {
	case report.TypeIncome:
		return data.TotalIncomeMinor
	case report.TypeExpenses, report.TypeBudget, report.TypeCashFlow:
		return data.TotalExpensesMinor
	case report.TypeCategoryBreak:
		return sumCategoryAmounts(data)
	default:
		return sumCategoryAmounts(data)
	}
}

func sumCategoryAmounts(data report.Data) money.Minor {
	var total money.Minor
	for _, item := range data.CategoryBreakdown {
		total += item.AmountMinor
	}

	return total
}

func writeDailyBreakdownCSV(writer *csv.Writer, data report.Data, currency string) error {
	header := []string{"Date", csvCurrencyColumn, "Income Minor", "Expenses Minor", "Balance Minor"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.DailyBreakdown {
		row := []string{
			item.Date.String(),
			currency,
			csvMinor(item.IncomeMinor),
			csvMinor(item.ExpensesMinor),
			csvMinor(item.BalanceMinor),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	return nil
}

func writeBudgetComparisonCSV(writer *csv.Writer, data report.Data, currency string) error {
	header := []string{"Budget", csvCurrencyColumn, "Planned Minor", "Actual Minor", "Difference Minor", "Percentage"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
	}

	for _, item := range data.BudgetComparison {
		row := []string{
			csvSafeText(item.BudgetName),
			currency,
			csvMinor(item.PlannedMinor),
			csvMinor(item.ActualMinor),
			csvMinor(item.DifferenceMinor),
			fmt.Sprintf("%.1f%%", item.Percentage),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	return nil
}
