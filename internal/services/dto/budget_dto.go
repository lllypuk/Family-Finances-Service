package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
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
	Name        string        `validate:"required,min=2,max=100"`
	AmountMinor money.Minor   `validate:"required,gt=0"`
	Period      budget.Period `validate:"required,oneof=weekly monthly yearly custom"`
	CategoryID  *uuid.UUID    `validate:"omitempty"`
	StartDate   date.Date     `validate:"required"`
	EndDate     date.Date     `validate:"required"`
}

// UpdateBudgetDTO represents the data that can be updated for an existing budget
type UpdateBudgetDTO struct {
	Name        *string      `validate:"omitempty,min=2,max=100"`
	AmountMinor *money.Minor `validate:"omitempty,gt=0"`
	StartDate   *date.Date   `validate:"omitempty"`
	EndDate     *date.Date   `validate:"omitempty"`
	IsActive    *bool        `validate:"omitempty"`
}

// BudgetFilterDTO represents filtering and pagination options for budgets
type BudgetFilterDTO struct {
	// Core filters
	CategoryID *uuid.UUID     `validate:"omitempty"`
	Period     *budget.Period `validate:"omitempty,oneof=weekly monthly yearly custom"`
	IsActive   *bool          `validate:"omitempty"`

	// Date filters
	ActiveOn *date.Date `validate:"omitempty"` // Budgets active on specific date
	DateFrom *date.Date `validate:"omitempty"`
	DateTo   *date.Date `validate:"omitempty"`

	// Amount filters
	AmountFromMinor *money.Minor `validate:"omitempty,gte=0"`
	AmountToMinor   *money.Minor `validate:"omitempty,gte=0"`

	// Status filters
	IsOverBudget    *bool `validate:"omitempty"` // Spent > Amount
	IsNearLimit     *bool `validate:"omitempty"` // Spent > 80% of Amount
	HasUnspentFunds *bool `validate:"omitempty"` // Remaining > 0

	// Pagination
	Limit  int `validate:"min=1,max=200"`
	Offset int `validate:"min=0"`

	// Sorting
	SortBy    *string `validate:"omitempty,oneof=name amount spent remaining created_at updated_at start_date end_date"`
	SortOrder *string `validate:"omitempty,oneof=asc desc"`
}

// BudgetStatusDTO represents detailed budget status information
type BudgetStatusDTO struct {
	BudgetID             uuid.UUID   `json:"budget_id"`
	Name                 string      `json:"name"`
	TotalAmountMinor     money.Minor `json:"total_amount_minor"`
	SpentAmountMinor     money.Minor `json:"spent_amount_minor"`
	RemainingAmountMinor money.Minor `json:"remaining_amount_minor"`
	UtilizationPercent   float64     `json:"utilization_percent"`
	DaysTotal            int         `json:"days_total"`
	DaysElapsed          int         `json:"days_elapsed"`
	DaysRemaining        int         `json:"days_remaining"`
	IsOverBudget         bool        `json:"is_over_budget"`
	IsNearLimit          bool        `json:"is_near_limit"`      // > 80%
	IsCriticalLimit      bool        `json:"is_critical_limit"`  // > 90%
	DailyBudgetMinor     money.Minor `json:"daily_budget_minor"` // Amount / DaysTotal
	DailySpentMinor      money.Minor `json:"daily_spent_minor"`  // SpentAmount / DaysElapsed
	// ProjectedOverrunMinor — превышение, если темп трат сохранится
	ProjectedOverrunMinor money.Minor `json:"projected_overrun_minor"`
	// Status — healthy | warning | critical | exceeded
	Status string `json:"status"`
}

// BudgetUtilizationDTO represents budget utilization analytics
type BudgetUtilizationDTO struct {
	BudgetID              uuid.UUID                 `json:"budget_id"`
	Period                string                    `json:"period"`
	UtilizationPercent    float64                   `json:"utilization_percent"`
	SpendingVelocityMinor money.Minor               `json:"spending_velocity_minor"` // Amount spent per day
	ProjectedCompletion   *time.Time                `json:"projected_completion"`    // When budget will be exhausted
	Recommendations       []string                  `json:"recommendations"`
	WeeklyBreakdown       []WeeklyBudgetBreakdown   `json:"weekly_breakdown,omitempty"`
	CategoryBreakdown     []CategoryBudgetBreakdown `json:"category_breakdown,omitempty"`
}

// WeeklyBudgetBreakdown represents weekly spending within a budget period
type WeeklyBudgetBreakdown struct {
	WeekStart           date.Date   `json:"week_start"`
	WeekEnd             date.Date   `json:"week_end"`
	WeeklyBudgetMinor   money.Minor `json:"weekly_budget_minor"`
	WeeklySpentMinor    money.Minor `json:"weekly_spent_minor"`
	WeeklyVarianceMinor money.Minor `json:"weekly_variance_minor"` // spent - budget
}

// CategoryBudgetBreakdown represents spending breakdown by category within a budget
type CategoryBudgetBreakdown struct {
	CategoryID   uuid.UUID   `json:"category_id"`
	CategoryName string      `json:"category_name"`
	AmountMinor  money.Minor `json:"amount_minor"`
	Percentage   float64     `json:"percentage"`
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
)

// NewBudgetFilterDTO creates a new BudgetFilterDTO with default values
func NewBudgetFilterDTO() BudgetFilterDTO {
	return BudgetFilterDTO{
		Limit:     DefaultBudgetLimit,
		Offset:    0,
		SortBy:    new("created_at"),
		SortOrder: new("desc"),
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
	if f.AmountFromMinor != nil && f.AmountToMinor != nil {
		if *f.AmountToMinor < *f.AmountFromMinor {
			return ErrInvalidAmountRange
		}
	}
	return nil
}

// ValidatePeriod validates that budget end date is after start date
func (c *CreateBudgetDTO) ValidatePeriod() error {
	if !c.EndDate.After(c.StartDate) {
		return ErrInvalidBudgetPeriod
	}
	return nil
}

// ValidatePeriod validates that budget end date is after start date for updates
func (u *UpdateBudgetDTO) ValidatePeriod() error {
	if u.StartDate != nil && u.EndDate != nil {
		if !u.EndDate.After(*u.StartDate) {
			return ErrInvalidBudgetPeriod
		}
	}
	return nil
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
