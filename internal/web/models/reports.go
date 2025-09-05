package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/report"
)

const (
	// GoodPerformanceThreshold represents threshold for good budget performance
	GoodPerformanceThreshold = 80
	// WarningPerformanceThreshold represents threshold for warning performance
	WarningPerformanceThreshold = 100
)

// ReportForm представляет форму создания отчета
type ReportForm struct {
	Name      string `form:"name"       validate:"required,min=1,max=100"                                         json:"name"`
	Type      string `form:"type"       validate:"required,oneof=expenses income budget cash_flow category_break" json:"type"`
	Period    string `form:"period"     validate:"required,oneof=daily weekly monthly yearly custom"              json:"period"`
	StartDate string `form:"start_date" validate:"required"                                                       json:"start_date"`
	EndDate   string `form:"end_date"   validate:"required"                                                       json:"end_date"`
}

// ReportDataVM представляет данные отчета для отображения
type ReportDataVM struct {
	ID              uuid.UUID     `json:"id"`
	Name            string        `json:"name"`
	Type            report.Type   `json:"type"`
	Period          report.Period `json:"period"`
	StartDate       time.Time     `json:"start_date"`
	EndDate         time.Time     `json:"end_date"`
	GeneratedAt     time.Time     `json:"generated_at"`
	FormattedPeriod string        `json:"formatted_period"`

	// Основные показатели
	TotalIncome       float64 `json:"total_income"`
	TotalExpenses     float64 `json:"total_expenses"`
	NetIncome         float64 `json:"net_income"`
	FormattedIncome   string  `json:"formatted_income"`
	FormattedExpenses string  `json:"formatted_expenses"`
	FormattedNet      string  `json:"formatted_net"`
	NetIncomeClass    string  `json:"net_income_class"` // "positive", "negative", "zero"

	// Разбивка по категориям
	CategoryBreakdown []CategoryReportItemVM `json:"category_breakdown"`

	// Дневная разбивка (для графиков/таблиц)
	DailyBreakdown []DailyReportItemVM `json:"daily_breakdown"`

	// Топ расходов
	TopExpenses []TransactionReportItemVM `json:"top_expenses"`

	// Сравнение с бюджетом
	BudgetComparison []BudgetComparisonItemVM `json:"budget_comparison"`

	// Метаданные для отображения
	CanExport bool   `json:"can_export"`
	ExportURL string `json:"export_url"`
}

// CategoryReportItemVM представляет элемент разбивки по категориям
type CategoryReportItemVM struct {
	CategoryID      uuid.UUID `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Amount          float64   `json:"amount"`
	Percentage      float64   `json:"percentage"`
	Count           int       `json:"count"`
	FormattedAmount string    `json:"formatted_amount"`
	ProgressWidth   string    `json:"progress_width"` // Для CSS width в %
}

// DailyReportItemVM представляет дневной элемент отчета
type DailyReportItemVM struct {
	Date              time.Time `json:"date"`
	Income            float64   `json:"income"`
	Expenses          float64   `json:"expenses"`
	Balance           float64   `json:"balance"`
	FormattedDate     string    `json:"formatted_date"`
	FormattedIncome   string    `json:"formatted_income"`
	FormattedExpenses string    `json:"formatted_expenses"`
	FormattedBalance  string    `json:"formatted_balance"`
	BalanceClass      string    `json:"balance_class"` // "positive", "negative", "zero"
}

// TransactionReportItemVM представляет транзакцию в отчете
type TransactionReportItemVM struct {
	ID              uuid.UUID `json:"id"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	Date            time.Time `json:"date"`
	FormattedAmount string    `json:"formatted_amount"`
	FormattedDate   string    `json:"formatted_date"`
}

// BudgetComparisonItemVM представляет сравнение с бюджетом
type BudgetComparisonItemVM struct {
	BudgetID         uuid.UUID `json:"budget_id"`
	BudgetName       string    `json:"budget_name"`
	Planned          float64   `json:"planned"`
	Actual           float64   `json:"actual"`
	Difference       float64   `json:"difference"`
	Percentage       float64   `json:"percentage"`
	FormattedPlanned string    `json:"formatted_planned"`
	FormattedActual  string    `json:"formatted_actual"`
	FormattedDiff    string    `json:"formatted_difference"`
	DifferenceClass  string    `json:"difference_class"`  // "over", "under", "exact"
	PerformanceClass string    `json:"performance_class"` // "good", "warning", "danger"
}

// ReportTypeOption представляет опцию типа отчета
type ReportTypeOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// GetReportTypeOptions возвращает доступные типы отчетов
func GetReportTypeOptions() []ReportTypeOption {
	return []ReportTypeOption{
		{
			Value:       "expenses",
			Label:       "Expenses Report",
			Description: "Detailed breakdown of all expenses",
		},
		{
			Value:       "income",
			Label:       "Income Report",
			Description: "Analysis of income sources",
		},
		{
			Value:       "budget",
			Label:       "Budget Performance",
			Description: "Compare actual vs planned spending",
		},
		{
			Value:       "cash_flow",
			Label:       "Cash Flow Summary",
			Description: "Income vs expenses over time",
		},
		{
			Value:       "category_break",
			Label:       "Category Breakdown",
			Description: "Spending analysis by category",
		},
	}
}

// FromDomain создает ReportDataVM из domain модели
func (vm *ReportDataVM) FromDomain(r *report.Report) {
	vm.ID = r.ID
	vm.Name = r.Name
	vm.Type = r.Type
	vm.Period = r.Period
	vm.StartDate = r.StartDate
	vm.EndDate = r.EndDate
	vm.GeneratedAt = r.GeneratedAt
	vm.FormattedPeriod = formatPeriod(r.StartDate, r.EndDate, r.Period)

	// Основные показатели
	vm.TotalIncome = r.Data.TotalIncome
	vm.TotalExpenses = r.Data.TotalExpenses
	vm.NetIncome = r.Data.NetIncome
	vm.FormattedIncome = formatMoney(r.Data.TotalIncome)
	vm.FormattedExpenses = formatMoney(r.Data.TotalExpenses)
	vm.FormattedNet = formatMoneyWithSign(r.Data.NetIncome)
	vm.NetIncomeClass = getMoneyClass(r.Data.NetIncome)

	// Конвертируем разбивку по категориям
	vm.CategoryBreakdown = make([]CategoryReportItemVM, len(r.Data.CategoryBreakdown))
	for i, item := range r.Data.CategoryBreakdown {
		vm.CategoryBreakdown[i] = CategoryReportItemVM{
			CategoryID:      item.CategoryID,
			CategoryName:    item.CategoryName,
			Amount:          item.Amount,
			Percentage:      item.Percentage,
			Count:           item.Count,
			FormattedAmount: formatMoney(item.Amount),
			ProgressWidth:   strconv.FormatFloat(item.Percentage, 'f', 1, 64) + "%",
		}
	}

	// Конвертируем дневную разбивку
	vm.DailyBreakdown = make([]DailyReportItemVM, len(r.Data.DailyBreakdown))
	for i, item := range r.Data.DailyBreakdown {
		vm.DailyBreakdown[i] = DailyReportItemVM{
			Date:              item.Date,
			Income:            item.Income,
			Expenses:          item.Expenses,
			Balance:           item.Balance,
			FormattedDate:     item.Date.Format("02.01.2006"),
			FormattedIncome:   formatMoney(item.Income),
			FormattedExpenses: formatMoney(item.Expenses),
			FormattedBalance:  formatMoneyWithSign(item.Balance),
			BalanceClass:      getMoneyClass(item.Balance),
		}
	}

	// Конвертируем топ расходы
	vm.TopExpenses = make([]TransactionReportItemVM, len(r.Data.TopExpenses))
	for i, item := range r.Data.TopExpenses {
		vm.TopExpenses[i] = TransactionReportItemVM{
			ID:              item.ID,
			Amount:          item.Amount,
			Description:     item.Description,
			Category:        item.Category,
			Date:            item.Date,
			FormattedAmount: formatMoney(item.Amount),
			FormattedDate:   item.Date.Format("02.01.2006"),
		}
	}

	// Конвертируем сравнение с бюджетом
	vm.BudgetComparison = make([]BudgetComparisonItemVM, len(r.Data.BudgetComparison))
	for i, item := range r.Data.BudgetComparison {
		vm.BudgetComparison[i] = BudgetComparisonItemVM{
			BudgetID:         item.BudgetID,
			BudgetName:       item.BudgetName,
			Planned:          item.Planned,
			Actual:           item.Actual,
			Difference:       item.Difference,
			Percentage:       item.Percentage,
			FormattedPlanned: formatMoney(item.Planned),
			FormattedActual:  formatMoney(item.Actual),
			FormattedDiff:    formatMoneyWithSign(item.Difference),
			DifferenceClass:  getDifferenceClass(item.Difference),
			PerformanceClass: getPerformanceClass(item.Percentage),
		}
	}

	// Метаданные
	vm.CanExport = true
	vm.ExportURL = "/reports/" + r.ID.String() + "/export?format=csv"
}

// ToReportType конвертирует строку в тип отчета
func (f *ReportForm) ToReportType() report.Type {
	switch f.Type {
	case "expenses":
		return report.TypeExpenses
	case "income":
		return report.TypeIncome
	case "budget":
		return report.TypeBudget
	case "cash_flow":
		return report.TypeCashFlow
	case "category_break":
		return report.TypeCategoryBreak
	default:
		return report.TypeExpenses
	}
}

// ToReportPeriod конвертирует строку в период отчета
func (f *ReportForm) ToReportPeriod() report.Period {
	switch f.Period {
	case "daily":
		return report.PeriodDaily
	case "weekly":
		return report.PeriodWeekly
	case "monthly":
		return report.PeriodMonthly
	case "yearly":
		return report.PeriodYearly
	case "custom":
		return report.PeriodCustom
	default:
		return report.PeriodMonthly
	}
}

// GetStartDate возвращает дату начала как time.Time
func (f *ReportForm) GetStartDate() (time.Time, error) {
	return time.Parse("2006-01-02", f.StartDate)
}

// GetEndDate возвращает дату окончания как time.Time
func (f *ReportForm) GetEndDate() (time.Time, error) {
	date, err := time.Parse("2006-01-02", f.EndDate)
	if err != nil {
		return time.Time{}, err
	}

	// Устанавливаем время на конец дня
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
	return endOfDay, nil
}

// formatPeriod форматирует период отчета
func formatPeriod(startDate, endDate time.Time, period report.Period) string {
	start := startDate.Format("02.01.2006")
	end := endDate.Format("02.01.2006")

	switch period {
	case report.PeriodDaily:
		return start
	case report.PeriodWeekly:
		return "Week " + start + " - " + end
	case report.PeriodMonthly:
		return startDate.Format("January 2006")
	case report.PeriodYearly:
		return startDate.Format("2006")
	case report.PeriodCustom:
		return start + " - " + end
	default:
		return start + " - " + end
	}
}

// formatMoneyWithSign форматирует сумму со знаком
func formatMoneyWithSign(amount float64) string {
	if amount >= 0 {
		return "+" + formatMoney(amount)
	}
	return formatMoney(amount)
}

// getMoneyClass возвращает CSS класс для денежной суммы
func getMoneyClass(amount float64) string {
	if amount > 0 {
		return "positive"
	}
	if amount < 0 {
		return "negative"
	}
	return "zero"
}

// getDifferenceClass возвращает CSS класс для разности с бюджетом
func getDifferenceClass(difference float64) string {
	if difference > 0 {
		return "over"
	}
	if difference < 0 {
		return "under"
	}
	return "exact"
}

// getPerformanceClass возвращает CSS класс для производительности бюджета
func getPerformanceClass(percentage float64) string {
	if percentage <= GoodPerformanceThreshold {
		return "good"
	}
	if percentage <= WarningPerformanceThreshold {
		return "warning"
	}
	return "danger"
}
