package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
)

// DTO validation errors
var (
	ErrInvalidBudgetPeriod   = errors.New("budget end date must be after start date")
	ErrInvalidBudgetAmount   = errors.New("budget amount must be greater than 0")
	ErrBudgetPeriodOverlap   = errors.New("budget period overlaps with existing budget for this category")
	ErrBudgetAlreadyExceeded = errors.New("budget amount is less than already spent amount")
)

// CreateBudgetDTO represents the data required to create a new budget
type CreateBudgetDTO struct {
	Name       string        `validate:"required,min=2,max=100"`
	Amount     float64       `validate:"required,gt=0"`
	Period     budget.Period `validate:"required,oneof=weekly monthly yearly custom"`
	CategoryID *uuid.UUID    `validate:"omitempty"`
	StartDate  time.Time     `validate:"required"`
	EndDate    time.Time     `validate:"required"`
}

// UpdateBudgetDTO represents the data that can be updated for an existing budget
type UpdateBudgetDTO struct {
	Name      *string    `validate:"omitempty,min=2,max=100"`
	Amount    *float64   `validate:"omitempty,gt=0"`
	StartDate *time.Time `validate:"omitempty"`
	EndDate   *time.Time `validate:"omitempty"`
	IsActive  *bool      `validate:"omitempty"`
}

// BudgetFilterDTO represents filtering and pagination options for budgets
type BudgetFilterDTO struct {
	// Core filters
	CategoryID *uuid.UUID     `validate:"omitempty"`
	Period     *budget.Period `validate:"omitempty,oneof=weekly monthly yearly custom"`
	IsActive   *bool          `validate:"omitempty"`

	// Date filters
	ActiveOn *time.Time `validate:"omitempty"` // Budgets active on specific date
	DateFrom *time.Time `validate:"omitempty"`
	DateTo   *time.Time `validate:"omitempty"`

	// Amount filters
	AmountFrom *float64 `validate:"omitempty,gte=0"`
	AmountTo   *float64 `validate:"omitempty,gte=0"`

	// Status filters
	IsOverBudget    *bool `validate:"omitempty"` // Spent > Amount
	IsNearLimit     *bool `validate:"omitempty"` // Spent > 80% of Amount
	HasUnspentFunds *bool `validate:"omitempty"` // Remaining > 0

	// Pagination
	Limit  int `validate:"min=1,max=100"`
	Offset int `validate:"min=0"`

	// Sorting
	SortBy    *string `validate:"omitempty,oneof=name amount spent remaining created_at updated_at start_date end_date"`
	SortOrder *string `validate:"omitempty,oneof=asc desc"`
}

// BudgetResponseDTO represents budget data for API responses
type BudgetResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Amount     float64    `json:"amount"`
	Spent      float64    `json:"spent"`
	Remaining  float64    `json:"remaining"`
	Period     string     `json:"period"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Calculated fields
	UtilizationPercent float64 `json:"utilization_percent"`
	DaysRemaining      int     `json:"days_remaining"`
	IsOverBudget       bool    `json:"is_over_budget"`
	IsNearLimit        bool    `json:"is_near_limit"`
}

// BudgetStatusDTO represents detailed budget status information
type BudgetStatusDTO struct {
	BudgetID           uuid.UUID `json:"budget_id"`
	Name               string    `json:"name"`
	TotalAmount        float64   `json:"total_amount"`
	SpentAmount        float64   `json:"spent_amount"`
	RemainingAmount    float64   `json:"remaining_amount"`
	UtilizationPercent float64   `json:"utilization_percent"`
	DaysTotal          int       `json:"days_total"`
	DaysElapsed        int       `json:"days_elapsed"`
	DaysRemaining      int       `json:"days_remaining"`
	IsOverBudget       bool      `json:"is_over_budget"`
	IsNearLimit        bool      `json:"is_near_limit"`     // > 80%
	IsCriticalLimit    bool      `json:"is_critical_limit"` // > 90%
	DailyBudget        float64   `json:"daily_budget"`      // Amount / DaysTotal
	DailySpent         float64   `json:"daily_spent"`       // SpentAmount / DaysElapsed
	ProjectedOverrun   float64   `json:"projected_overrun"` // If current spending rate continues
	Status             string    `json:"status"`            // "healthy", "warning", "critical", "exceeded"
}

// BudgetUtilizationDTO represents budget utilization analytics
type BudgetUtilizationDTO struct {
	BudgetID            uuid.UUID                 `json:"budget_id"`
	Period              string                    `json:"period"`
	UtilizationPercent  float64                   `json:"utilization_percent"`
	SpendingVelocity    float64                   `json:"spending_velocity"`    // Amount spent per day
	ProjectedCompletion *time.Time                `json:"projected_completion"` // When budget will be exhausted
	Recommendations     []string                  `json:"recommendations"`
	WeeklyBreakdown     []WeeklyBudgetBreakdown   `json:"weekly_breakdown,omitempty"`
	CategoryBreakdown   []CategoryBudgetBreakdown `json:"category_breakdown,omitempty"`
}

// WeeklyBudgetBreakdown represents weekly spending within a budget period
type WeeklyBudgetBreakdown struct {
	WeekStart      time.Time `json:"week_start"`
	WeekEnd        time.Time `json:"week_end"`
	WeeklyBudget   float64   `json:"weekly_budget"`
	WeeklySpent    float64   `json:"weekly_spent"`
	WeeklyVariance float64   `json:"weekly_variance"` // spent - budget
}

// CategoryBudgetBreakdown represents spending breakdown by category within a budget
type CategoryBudgetBreakdown struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Amount       float64   `json:"amount"`
	Percentage   float64   `json:"percentage"`
}

// BudgetAlertDTO represents budget alert/notification data
type BudgetAlertDTO struct {
	BudgetID   uuid.UUID `json:"budget_id"`
	BudgetName string    `json:"budget_name"`
	AlertType  string    `json:"alert_type"` // "near_limit", "over_budget", "exceeded"
	Threshold  float64   `json:"threshold"`  // The threshold that triggered the alert (80%, 100%, etc.)
	Current    float64   `json:"current"`    // Current utilization percentage
	Message    string    `json:"message"`
	Severity   string    `json:"severity"` // "info", "warning", "critical"
	CreatedAt  time.Time `json:"created_at"`
}

const (
	// BudgetStatusHealthy indicates budget utilization under 70%
	BudgetStatusHealthy = "healthy"
	// BudgetStatusWarning indicates budget utilization between 70-89%
	BudgetStatusWarning = "warning"
	// BudgetStatusCritical indicates budget utilization between 90-99%
	BudgetStatusCritical = "critical"
	// BudgetStatusExceeded indicates budget utilization at or above 100%
	BudgetStatusExceeded = "exceeded"

	// BudgetAlertNearLimit threshold for near-limit alerts
	BudgetAlertNearLimit = 80.0
	// BudgetAlertCritical threshold for critical alerts
	BudgetAlertCritical = 90.0
	// BudgetAlertOverBudget threshold for over-budget alerts
	BudgetAlertOverBudget = 100.0

	// DefaultBudgetLimit default pagination limit for budget queries
	DefaultBudgetLimit = 20

	// HoursPerDay number of hours in a day
	HoursPerDay = 24
	// PercentMultiplier multiplier for percentage calculations
	PercentMultiplier = 100
)

// NewBudgetFilterDTO creates a new BudgetFilterDTO with default values
func NewBudgetFilterDTO() BudgetFilterDTO {
	return BudgetFilterDTO{
		Limit:     DefaultBudgetLimit,
		Offset:    0,
		SortBy:    stringPtr("created_at"),
		SortOrder: stringPtr("desc"),
	}
}

// ValidateDateRange validates that EndDate is after StartDate if both are provided
func (f *BudgetFilterDTO) ValidateDateRange() error {
	if f.DateFrom != nil && f.DateTo != nil {
		if f.DateTo.Before(*f.DateFrom) {
			return ErrInvalidDateRange
		}
	}
	return nil
}

// ValidateAmountRange validates that AmountTo is greater than AmountFrom if both are provided
func (f *BudgetFilterDTO) ValidateAmountRange() error {
	if f.AmountFrom != nil && f.AmountTo != nil {
		if *f.AmountTo < *f.AmountFrom {
			return ErrInvalidAmountRange
		}
	}
	return nil
}

// ValidatePeriod validates that budget end date is after start date
func (c *CreateBudgetDTO) ValidatePeriod() error {
	if c.EndDate.Before(c.StartDate) || c.EndDate.Equal(c.StartDate) {
		return ErrInvalidBudgetPeriod
	}
	return nil
}

// ValidatePeriod validates that budget end date is after start date for updates
func (u *UpdateBudgetDTO) ValidatePeriod() error {
	if u.StartDate != nil && u.EndDate != nil {
		if u.EndDate.Before(*u.StartDate) || u.EndDate.Equal(*u.StartDate) {
			return ErrInvalidBudgetPeriod
		}
	}
	return nil
}

// CalculateUtilizationPercent calculates budget utilization percentage
func CalculateUtilizationPercent(spent, amount float64) float64 {
	if amount == 0 {
		return 0
	}
	return (spent / amount) * PercentMultiplier
}

// DetermineBudgetStatus determines budget status based on utilization
func DetermineBudgetStatus(utilizationPercent float64) string {
	switch {
	case utilizationPercent >= BudgetAlertOverBudget:
		return BudgetStatusExceeded
	case utilizationPercent >= BudgetAlertCritical:
		return BudgetStatusCritical
	case utilizationPercent >= BudgetAlertNearLimit:
		return BudgetStatusWarning
	default:
		return BudgetStatusHealthy
	}
}

// CalculateDaysRemaining calculates days remaining in budget period
func CalculateDaysRemaining(endDate time.Time) int {
	now := time.Now()
	if endDate.Before(now) {
		return 0
	}
	return int(endDate.Sub(now).Hours() / HoursPerDay)
}
