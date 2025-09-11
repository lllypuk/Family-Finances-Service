package models

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
)

const (
	// AlertLevelDanger represents danger alert level for budgets
	AlertLevelDanger  = "danger"
	AlertLevelWarning = "warning"
	AlertLevelInfo    = "info"

	// ThresholdHigh represents high threshold for budget alerts
	ThresholdHigh   = 80
	ThresholdMedium = 50
	ThresholdMax    = 100
)

// BudgetForm представляет форму создания/редактирования бюджета
type BudgetForm struct {
	Name       string `form:"name"        validate:"required,min=1,max=100"                      json:"name"`
	Amount     string `form:"amount"      validate:"required,numeric,gt=0"                       json:"amount"`
	Period     string `form:"period"      validate:"required,oneof=weekly monthly yearly custom" json:"period"`
	CategoryID string `form:"category_id" validate:"omitempty,uuid"                              json:"category_id,omitempty"`
	StartDate  string `form:"start_date"  validate:"required"                                    json:"start_date"`
	EndDate    string `form:"end_date"    validate:"required"                                    json:"end_date"`
	IsActive   bool   `form:"is_active"                                                          json:"is_active"`
}

// BudgetProgressVM представляет прогресс бюджета для отображения
type BudgetProgressVM struct {
	ID                 uuid.UUID     `json:"id"`
	Name               string        `json:"name"`
	Amount             float64       `json:"amount"`
	Spent              float64       `json:"spent"`
	Remaining          float64       `json:"remaining"`
	Percentage         float64       `json:"percentage"`
	Period             budget.Period `json:"period"`
	CategoryID         *uuid.UUID    `json:"category_id,omitempty"`
	CategoryName       string        `json:"category_name,omitempty"`
	CategoryColor      string        `json:"category_color,omitempty"`
	StartDate          time.Time     `json:"start_date"`
	EndDate            time.Time     `json:"end_date"`
	IsActive           bool          `json:"is_active"`
	IsOverBudget       bool          `json:"is_over_budget"`
	DaysLeft           int           `json:"days_left"`
	DaysElapsed        int           `json:"days_elapsed"`
	DaysTotal          int           `json:"days_total"`
	TimePercentage     float64       `json:"time_percentage"`
	FormattedAmount    string        `json:"formatted_amount"`
	FormattedSpent     string        `json:"formatted_spent"`
	FormattedRemaining string        `json:"formatted_remaining"`
	FormattedOverage   string        `json:"formatted_overage"`
	ProgressBarClass   string        `json:"progress_bar_class"`
	AlertLevel         string        `json:"alert_level"` // "", "warning", "danger"
	DailyBudgetPace    float64       `json:"daily_budget_pace"`
	DailySpendingPace  float64       `json:"daily_spending_pace"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// BudgetAlertVM представляет алерт для бюджета
type BudgetAlertVM struct {
	ID          uuid.UUID  `json:"id"`
	BudgetID    uuid.UUID  `json:"budget_id"`
	BudgetName  string     `json:"budget_name"`
	Threshold   float64    `json:"threshold"`
	IsTriggered bool       `json:"is_triggered"`
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
	Message     string     `json:"message"`
	AlertClass  string     `json:"alert_class"` // "warning", "danger"
}

// BudgetFilter представляет фильтры для поиска бюджетов
type BudgetFilter struct {
	Name       string `form:"name"        json:"name,omitempty"`
	Period     string `form:"period"      json:"period,omitempty"      validate:"omitempty,oneof=weekly monthly yearly custom"`
	CategoryID string `form:"category_id" json:"category_id,omitempty" validate:"omitempty,uuid"`
	IsActive   *bool  `form:"is_active"   json:"is_active,omitempty"`
	IsExpired  *bool  `form:"is_expired"  json:"is_expired,omitempty"`
}

// BudgetAlertForm представляет форму настройки алертов
type BudgetAlertForm struct {
	BudgetID  string `form:"budget_id" validate:"required,uuid"                  json:"budget_id"`
	Threshold string `form:"threshold" validate:"required,numeric,min=1,max=100" json:"threshold"`
}

// SpendingAnalysis представляет анализ трат по бюджету
type SpendingAnalysis struct {
	DailyAverage   float64 `json:"daily_average"`
	BudgetPace     float64 `json:"budget_pace"`
	ProjectedTotal float64 `json:"projected_total"`
	DaysElapsed    int     `json:"days_elapsed"`
	Variance       float64 `json:"variance"` // Разница между DailyAverage и BudgetPace
}

// TransactionSummary представляет краткую информацию о транзакции
type TransactionSummary struct {
	Description     string    `json:"description"`
	Amount          float64   `json:"amount"`
	FormattedAmount string    `json:"formatted_amount"`
	Type            string    `json:"type"` // "expense" или "income"
	CategoryName    string    `json:"category_name"`
	Date            time.Time `json:"date"`
}

// FromDomain создает BudgetProgressVM из domain модели
func (vm *BudgetProgressVM) FromDomain(b *budget.Budget) {
	vm.ID = b.ID
	vm.Name = b.Name
	vm.Amount = b.Amount
	vm.Spent = b.Spent
	vm.Remaining = b.GetRemainingAmount()
	vm.Percentage = b.GetSpentPercentage()
	vm.Period = b.Period
	vm.CategoryID = b.CategoryID
	vm.StartDate = b.StartDate
	vm.EndDate = b.EndDate
	vm.IsActive = b.IsActive
	vm.IsOverBudget = b.IsOverBudget()
	vm.CreatedAt = b.CreatedAt
	vm.UpdatedAt = b.UpdatedAt

	// Вычисляем временные метрики
	now := time.Now()
	totalDuration := b.EndDate.Sub(b.StartDate)
	vm.DaysTotal = int(totalDuration.Hours() / HoursInDay)

	if b.EndDate.After(now) {
		vm.DaysLeft = int(b.EndDate.Sub(now).Hours() / HoursInDay)
	} else {
		vm.DaysLeft = 0
	}

	if now.After(b.StartDate) {
		elapsed := now.Sub(b.StartDate)
		vm.DaysElapsed = max(int(elapsed.Hours()/HoursInDay),
			// Минимум 1 день для расчетов
			1)
	} else {
		vm.DaysElapsed = 0
	}

	// Прогресс по времени
	if vm.DaysTotal > 0 {
		vm.TimePercentage = float64(vm.DaysElapsed) / float64(vm.DaysTotal) * ThresholdMax
		if vm.TimePercentage > ThresholdMax {
			vm.TimePercentage = ThresholdMax
		}
	}

	// Дневные темпы
	if vm.DaysTotal > 0 {
		vm.DailyBudgetPace = b.Amount / float64(vm.DaysTotal)
	}
	if vm.DaysElapsed > 0 {
		vm.DailySpendingPace = b.Spent / float64(vm.DaysElapsed)
	}

	// Форматирование для отображения
	vm.FormattedAmount = formatMoney(b.Amount)
	vm.FormattedSpent = formatMoney(b.Spent)
	vm.FormattedRemaining = formatMoney(vm.Remaining)

	// Форматированная сумма превышения
	if vm.IsOverBudget {
		overage := b.Spent - b.Amount
		vm.FormattedOverage = formatMoney(overage)
	}

	// CSS классы для прогресс-бара
	vm.ProgressBarClass = getProgressBarClass(vm.Percentage, vm.IsOverBudget)
	vm.AlertLevel = getAlertLevel(vm.Percentage, vm.IsOverBudget)
}

// GetAmount возвращает сумму как float64
func (f *BudgetForm) GetAmount() (float64, error) {
	return strconv.ParseFloat(f.Amount, 64)
}

// ToBudgetPeriod конвертирует строку в тип периода бюджета
func (f *BudgetForm) ToBudgetPeriod() budget.Period {
	switch f.Period {
	case "weekly":
		return budget.PeriodWeekly
	case "monthly":
		return budget.PeriodMonthly
	case "yearly":
		return budget.PeriodYearly
	case "custom":
		return budget.PeriodCustom
	default:
		return budget.PeriodMonthly
	}
}

// GetCategoryID возвращает UUID категории или nil
func (f *BudgetForm) GetCategoryID() *uuid.UUID {
	if f.CategoryID == "" {
		return nil
	}

	id, err := uuid.Parse(f.CategoryID)
	if err != nil {
		return nil
	}

	return &id
}

// GetStartDate возвращает дату начала как time.Time
func (f *BudgetForm) GetStartDate() (time.Time, error) {
	return time.Parse("2006-01-02", f.StartDate)
}

// GetEndDate возвращает дату окончания как time.Time
func (f *BudgetForm) GetEndDate() (time.Time, error) {
	date, err := time.Parse("2006-01-02", f.EndDate)
	if err != nil {
		return time.Time{}, err
	}

	// Устанавливаем время на конец дня
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
	return endOfDay, nil
}

// GetBudgetID возвращает UUID бюджета
func (f *BudgetAlertForm) GetBudgetID() (uuid.UUID, error) {
	return uuid.Parse(f.BudgetID)
}

// GetThreshold возвращает порог как float64
func (f *BudgetAlertForm) GetThreshold() (float64, error) {
	return strconv.ParseFloat(f.Threshold, 64)
}

// FromDomainAlert создает BudgetAlertVM из domain модели
func (vm *BudgetAlertVM) FromDomainAlert(alert *budget.Alert, budgetName string) {
	vm.ID = alert.ID
	vm.BudgetID = alert.BudgetID
	vm.BudgetName = budgetName
	vm.Threshold = alert.Threshold
	vm.IsTriggered = alert.IsTriggered
	vm.TriggeredAt = alert.TriggeredAt

	// Создаем сообщение
	if alert.IsTriggered {
		vm.Message = formatAlertMessage(alert.Threshold, true)
	} else {
		vm.Message = formatAlertMessage(alert.Threshold, false)
	}

	// CSS класс для алерта
	vm.AlertClass = getAlertClassForThreshold(alert.Threshold)
}

// formatMoney форматирует денежную сумму
func formatMoney(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// getProgressBarClass возвращает CSS класс для прогресс-бара
func getProgressBarClass(percentage float64, isOverBudget bool) string {
	if isOverBudget {
		return "progress-danger"
	}

	if percentage >= ThresholdHigh {
		return "progress-warning"
	}

	if percentage >= ThresholdMedium {
		return "progress-info"
	}

	return "progress-success"
}

// getAlertLevel возвращает уровень алерта
func getAlertLevel(percentage float64, isOverBudget bool) string {
	if isOverBudget {
		return AlertLevelDanger
	}

	if percentage >= ThresholdHigh {
		return AlertLevelWarning
	}

	return ""
}

// formatAlertMessage создает сообщение для алерта
func formatAlertMessage(threshold float64, isTriggered bool) string {
	thresholdStr := strconv.FormatFloat(threshold, 'f', 0, 64)

	if isTriggered {
		if threshold >= ThresholdMax {
			return "Budget exceeded! You've spent more than allocated."
		}
		return "Alert: You've reached " + thresholdStr + "% of your budget."
	}

	return "Alert will trigger at " + thresholdStr + "% of budget."
}

// getAlertClassForThreshold возвращает CSS класс для алерта по порогу
func getAlertClassForThreshold(threshold float64) string {
	if threshold >= ThresholdMax {
		return AlertLevelDanger
	}

	if threshold >= ThresholdHigh {
		return AlertLevelWarning
	}

	return AlertLevelInfo
}
