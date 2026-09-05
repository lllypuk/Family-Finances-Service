package models

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/transaction"
)

// CSS класс константы
const (
	CSSClassTextSuccess = "text-success"
	CSSClassTextDanger  = "text-danger"
	CSSClassTextMuted   = "text-muted"
)

// DashboardViewModel представляет данные для главной страницы
type DashboardViewModel struct {
	MonthlySummary   *MonthlySummaryCard   `json:"monthly_summary"`
	BudgetOverview   *BudgetOverviewCard   `json:"budget_overview"`
	RecentActivity   *RecentActivityCard   `json:"recent_activity"`
	CategoryInsights *CategoryInsightsCard `json:"category_insights"`
}

// MonthlySummaryCard содержит финансовую сводку за месяц
type MonthlySummaryCard struct {
	TotalIncome      float64 `json:"total_income"`
	TotalExpenses    float64 `json:"total_expenses"`
	NetIncome        float64 `json:"net_income"`
	TransactionCount int     `json:"transaction_count"`
	PreviousIncome   float64 `json:"previous_income"`
	PreviousExpenses float64 `json:"previous_expenses"`
	IncomeChange     float64 `json:"income_change"`   // Percentage change
	ExpensesChange   float64 `json:"expenses_change"` // Percentage change
	CurrentMonth     string  `json:"current_month"`
	PreviousMonth    string  `json:"previous_month"`
	HasPreviousData  bool    `json:"has_previous_data"`
}

// GetIncomeChangeClass возвращает CSS класс для изменения доходов
func (m *MonthlySummaryCard) GetIncomeChangeClass() string {
	if m.IncomeChange > 0 {
		return CSSClassTextSuccess
	} else if m.IncomeChange < 0 {
		return CSSClassTextDanger
	}
	return CSSClassTextMuted
}

// GetExpensesChangeClass возвращает CSS класс для изменения расходов
func (m *MonthlySummaryCard) GetExpensesChangeClass() string {
	if m.ExpensesChange > 0 {
		return CSSClassTextDanger // Рост расходов - плохо
	} else if m.ExpensesChange < 0 {
		return CSSClassTextSuccess // Снижение расходов - хорошо
	}
	return CSSClassTextMuted
}

// GetNetIncomeClass возвращает CSS класс для чистого дохода
func (m *MonthlySummaryCard) GetNetIncomeClass() string {
	if m.NetIncome > 0 {
		return CSSClassTextSuccess
	} else if m.NetIncome < 0 {
		return CSSClassTextDanger
	}
	return CSSClassTextMuted
}

// BudgetOverviewCard содержит сводку по бюджетам
type BudgetOverviewCard struct {
	TotalBudgets  int                   `json:"total_budgets"`
	ActiveBudgets int                   `json:"active_budgets"`
	OverBudget    int                   `json:"over_budget"`
	NearLimit     int                   `json:"near_limit"`
	TopBudgets    []*BudgetProgressItem `json:"top_budgets"`
	AlertsSummary *BudgetAlertsSummary  `json:"alerts_summary"`
}

// BudgetProgressItem представляет элемент прогресса бюджета
type BudgetProgressItem struct {
	ID            uuid.UUID     `json:"id"`
	Name          string        `json:"name"`
	CategoryName  string        `json:"category_name,omitempty"`
	Amount        float64       `json:"amount"`
	Spent         float64       `json:"spent"`
	Remaining     float64       `json:"remaining"`
	Percentage    float64       `json:"percentage"`
	Period        budget.Period `json:"period"`
	StartDate     time.Time     `json:"start_date"`
	EndDate       time.Time     `json:"end_date"`
	DaysRemaining int           `json:"days_remaining"`
	IsOverBudget  bool          `json:"is_over_budget"`
	IsNearLimit   bool          `json:"is_near_limit"`
	AlertLevel    string        `json:"alert_level"` // "success", "warning", "danger"
}

// GetProgressBarClass возвращает CSS класс для progress bar
func (b *BudgetProgressItem) GetProgressBarClass() string {
	if b.IsOverBudget {
		return "progress-danger"
	} else if b.IsNearLimit {
		return "progress-warning"
	}
	return "progress-success"
}

// GetAlertBadgeClass возвращает CSS класс для badge
func (b *BudgetProgressItem) GetAlertBadgeClass() string {
	switch b.AlertLevel {
	case "danger":
		return "badge-danger"
	case "warning":
		return "badge-warning"
	default:
		return "badge-success"
	}
}

// BudgetAlertsSummary содержит сводку по алертам бюджетов
type BudgetAlertsSummary struct {
	CriticalAlerts int `json:"critical_alerts"`
	WarningAlerts  int `json:"warning_alerts"`
	TotalAlerts    int `json:"total_alerts"`
}

// RecentActivityCard содержит последние транзакции
type RecentActivityCard struct {
	Transactions []*RecentTransactionItem `json:"transactions"`
	TotalCount   int                      `json:"total_count"`
	ShowingCount int                      `json:"showing_count"`
	HasMoreData  bool                     `json:"has_more_data"`
	LastUpdated  time.Time                `json:"last_updated"`
}

// RecentTransactionItem представляет элемент последней транзакции
type RecentTransactionItem struct {
	ID           uuid.UUID        `json:"id"`
	Description  string           `json:"description"`
	Amount       float64          `json:"amount"`
	Type         transaction.Type `json:"type"`
	CategoryName string           `json:"category_name"`
	Date         time.Time        `json:"date"`
	CreatedAt    time.Time        `json:"created_at"`
	RelativeTime string           `json:"relative_time"` // "2 часа назад"
}

// GetTypeClass возвращает CSS класс для типа транзакции
func (t *RecentTransactionItem) GetTypeClass() string {
	switch t.Type {
	case transaction.TypeIncome:
		return "text-success"
	case transaction.TypeExpense:
		return "text-danger"
	default:
		return "text-muted"
	}
}

// GetTypeIcon возвращает иконку для типа транзакции
func (t *RecentTransactionItem) GetTypeIcon() string {
	switch t.Type {
	case transaction.TypeIncome:
		return "💰"
	case transaction.TypeExpense:
		return "💸"
	default:
		return "💳"
	}
}

// CategoryInsightsCard содержит аналитику по категориям
type CategoryInsightsCard struct {
	TopExpenseCategories []*CategoryInsightItem `json:"top_expense_categories"`
	TopIncomeCategories  []*CategoryInsightItem `json:"top_income_categories"`
	PeriodStart          time.Time              `json:"period_start"`
	PeriodEnd            time.Time              `json:"period_end"`
	TotalExpenses        float64                `json:"total_expenses"`
	TotalIncome          float64                `json:"total_income"`
}

// CategoryInsightItem представляет элемент аналитики по категории
type CategoryInsightItem struct {
	CategoryID       uuid.UUID `json:"category_id"`
	CategoryName     string    `json:"category_name"`
	CategoryColor    string    `json:"category_color,omitempty"`
	CategoryIcon     string    `json:"category_icon,omitempty"`
	Amount           float64   `json:"amount"`
	TransactionCount int       `json:"transaction_count"`
	Percentage       float64   `json:"percentage"`
}

// GetPercentageClass возвращает CSS класс для процента
func (c *CategoryInsightItem) GetPercentageClass() string {
	if c.Percentage >= CategoryHighPercentage {
		return "text-danger"
	} else if c.Percentage >= CategoryMediumPercentage {
		return "text-warning"
	}
	return "text-success"
}

// DashboardFilters содержит фильтры для dashboard
type DashboardFilters struct {
	Period      string     `query:"period"      validate:"omitempty,oneof=current_month last_month last_3_months last_6_months current_year"`
	StartDate   *time.Time `query:"start_date"`
	EndDate     *time.Time `query:"end_date"`
	CategoryID  *uuid.UUID `query:"category_id"`
	RefreshMode string     `query:"refresh"     validate:"omitempty,oneof=auto manual"`
}

// GetPeriodDates возвращает даты для выбранного периода
func (f *DashboardFilters) GetPeriodDates() (time.Time, time.Time) {
	now := time.Now()

	switch f.Period {
	case "last_month":
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return start, end
	case "last_3_months":
		start := time.Date(now.Year(), now.Month()-2, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
		return start, end
	case "last_6_months":
		start := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
		return start, end
	case "current_year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location()).Add(-time.Second)
		return start, end
	default: // "current_month"
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return start, end
	}
}

// ValidateCustomDateRange проверяет пользовательский диапазон дат
func (f *DashboardFilters) ValidateCustomDateRange() error {
	if f.StartDate != nil && f.EndDate != nil {
		if f.EndDate.Before(*f.StartDate) {
			return ErrInvalidDateRange
		}

		// Ограничиваем максимальный период 2 годами
		maxPeriod := MaxPeriodYears * DaysInYear * HoursInDay * time.Hour
		if f.EndDate.Sub(*f.StartDate) > maxPeriod {
			return ErrDateRangeTooLarge
		}
	}
	return nil
}

// Constants для dashboard
const (
	// Максимальное количество элементов в каждой секции
	MaxRecentTransactions = 10
	MaxTopBudgets         = 5
	MaxTopCategories      = 5

	// Пороги для алертов
	BudgetNearLimitThreshold = 80.0  // 80% от бюджета
	BudgetOverLimitThreshold = 100.0 // 100% от бюджета

	// Пороги для категорий
	CategoryHighPercentage   = 30.0 // 30% - высокий процент
	CategoryMediumPercentage = 15.0 // 15% - средний процент

	// Константы времени
	HoursInDay     = 24
	DaysInWeek     = 7
	DaysInMonth    = 30
	DaysInYear     = 365
	MaxPeriodYears = 2

	// Процентные расчеты
	PercentageMultiplier = 100.0
)

var (
	ErrInvalidDateRange  = NewValidationError("End date must be after start date")
	ErrDateRangeTooLarge = NewValidationError("Date range cannot exceed 2 years")
)

// NewValidationError creates a new validation error with the given message
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
