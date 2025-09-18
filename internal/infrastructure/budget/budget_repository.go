package budget

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/infrastructure/validation"
)

// Budget validation constants
const (
	maxBudgetAmount     = 999999999.99
	maxBudgetNameLength = 255
)

// PostgreSQLRepository implements budget repository using PostgreSQL
type PostgreSQLRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLRepository creates a new PostgreSQL budget repository
func NewPostgreSQLRepository(db *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// ValidateBudgetPeriod validates budget period
func ValidateBudgetPeriod(period budget.Period) error {
	switch period {
	case budget.PeriodWeekly, budget.PeriodMonthly, budget.PeriodYearly, budget.PeriodCustom:
		return nil
	default:
		return errors.New("invalid budget period")
	}
}

// ValidateBudgetAmount validates budget amount
func ValidateBudgetAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("budget amount must be positive")
	}
	if amount > maxBudgetAmount {
		return errors.New("budget amount too large")
	}
	return nil
}

// ValidateBudgetName validates budget name
func ValidateBudgetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("budget name cannot be empty")
	}
	if len(name) > maxBudgetNameLength {
		return errors.New("budget name too long")
	}
	return nil
}

// Create creates a new budget in the database
func (r *PostgreSQLRepository) Create(ctx context.Context, b *budget.Budget) error {
	// Validate budget parameters
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := ValidateBudgetPeriod(b.Period); err != nil {
		return fmt.Errorf("invalid period: %w", err)
	}
	if err := ValidateBudgetAmount(b.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := ValidateBudgetName(b.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Validate category ID if provided
	if b.CategoryID != nil {
		if err := validation.ValidateUUID(*b.CategoryID); err != nil {
			return fmt.Errorf("invalid category ID: %w", err)
		}
	}

	// Validate date range
	if !b.EndDate.After(b.StartDate) {
		return errors.New("end date must be after start date")
	}

	// Set timestamps
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now

	query := `
		INSERT INTO family_budget.budgets (
			id, name, amount, spent, period, start_date, end_date,
			category_id, family_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.Exec(ctx, query,
		b.ID, b.Name, b.Amount, b.Spent, string(b.Period),
		b.StartDate, b.EndDate, b.CategoryID, b.FamilyID,
		b.IsActive, b.CreatedAt, b.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("budget with name '%s' already exists for this period", b.Name)
		}
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

// GetByID retrieves a budget by their ID
func (r *PostgreSQLRepository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM family_budget.budgets
		WHERE id = $1`

	var b budget.Budget
	var periodStr string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.Name, &b.Amount, &b.Spent, &periodStr,
		&b.StartDate, &b.EndDate, &b.CategoryID, &b.FamilyID,
		&b.IsActive, &b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get budget by id: %w", err)
	}

	b.Period = budget.Period(periodStr)
	return &b, nil
}

// GetByFamilyID retrieves all budgets belonging to a specific family
func (r *PostgreSQLRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM family_budget.budgets
		WHERE family_id = $1 AND is_active = true
		ORDER BY start_date DESC, name`

	rows, err := r.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by family id: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b budget.Budget
		var periodStr string

		err = rows.Scan(
			&b.ID, &b.Name, &b.Amount, &b.Spent, &periodStr,
			&b.StartDate, &b.EndDate, &b.CategoryID, &b.FamilyID,
			&b.IsActive, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		b.Period = budget.Period(periodStr)
		budgets = append(budgets, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetActiveBudgets retrieves all active budgets for a family
func (r *PostgreSQLRepository) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	now := time.Now()
	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM family_budget.budgets
		WHERE family_id = $1 AND is_active = true
		AND start_date <= $2 AND end_date >= $2
		ORDER BY start_date DESC, name`

	rows, err := r.db.Query(ctx, query, familyID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b budget.Budget
		var periodStr string

		err = rows.Scan(
			&b.ID, &b.Name, &b.Amount, &b.Spent, &periodStr,
			&b.StartDate, &b.EndDate, &b.CategoryID, &b.FamilyID,
			&b.IsActive, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		b.Period = budget.Period(periodStr)
		budgets = append(budgets, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetUsageStats returns comprehensive budget usage statistics
func (r *PostgreSQLRepository) GetUsageStats(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*UsageStats, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT
			b.id,
			b.name,
			b.amount,
			b.spent,
			b.period,
			b.start_date,
			b.end_date,
			(b.amount - b.spent) as remaining_amount,
			CASE
				WHEN b.amount > 0 THEN ROUND((b.spent / b.amount * 100)::NUMERIC, 2)
				ELSE 0
			END as usage_percentage,
			(b.end_date - CURRENT_DATE) as days_remaining,
			CASE
				WHEN b.spent > b.amount THEN 'over_budget'
				WHEN b.spent > (b.amount * 0.8) THEN 'warning'
				WHEN b.spent > (b.amount * 0.5) THEN 'on_track'
				ELSE 'safe'
			END as status,
			c.name as category_name
		FROM family_budget.budgets b
		LEFT JOIN family_budget.categories c ON b.category_id = c.id
		WHERE b.family_id = $1 AND b.is_active = true
		AND CURRENT_DATE BETWEEN b.start_date AND b.end_date
		ORDER BY usage_percentage DESC`

	rows, err := r.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget usage stats: %w", err)
	}
	defer rows.Close()

	var stats []*UsageStats
	for rows.Next() {
		var stat UsageStats
		var periodStr string
		var categoryName *string

		err = rows.Scan(
			&stat.BudgetID, &stat.BudgetName, &stat.BudgetAmount, &stat.SpentAmount,
			&periodStr, &stat.StartDate, &stat.EndDate, &stat.RemainingAmount,
			&stat.UsagePercentage, &stat.DaysRemaining, &stat.Status, &categoryName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget usage stat: %w", err)
		}

		stat.Period = budget.Period(periodStr)
		if categoryName != nil {
			stat.CategoryName = *categoryName
		}

		stats = append(stats, &stat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return stats, nil
}

// Update updates an existing budget
func (r *PostgreSQLRepository) Update(ctx context.Context, b *budget.Budget) error {
	// Validate budget parameters
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := ValidateBudgetPeriod(b.Period); err != nil {
		return fmt.Errorf("invalid period: %w", err)
	}
	if err := ValidateBudgetAmount(b.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := ValidateBudgetName(b.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Validate category ID if provided
	if b.CategoryID != nil {
		if err := validation.ValidateUUID(*b.CategoryID); err != nil {
			return fmt.Errorf("invalid category ID: %w", err)
		}
	}

	// Update timestamp
	b.UpdatedAt = time.Now()

	query := `
		UPDATE family_budget.budgets
		SET name = $2, amount = $3, spent = $4, period = $5,
			start_date = $6, end_date = $7, category_id = $8, is_active = $9, updated_at = $10
		WHERE id = $1 AND family_id = $11`

	result, err := r.db.Exec(ctx, query,
		b.ID, b.Name, b.Amount, b.Spent, string(b.Period),
		b.StartDate, b.EndDate, b.CategoryID, b.IsActive, b.UpdatedAt, b.FamilyID,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("budget with name '%s' already exists for this period", b.Name)
		}
		return fmt.Errorf("failed to update budget: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget with id %s not found", b.ID)
	}

	return nil
}

// UpdateSpentAmount updates the spent amount for a budget
func (r *PostgreSQLRepository) UpdateSpentAmount(ctx context.Context, budgetID uuid.UUID, spentAmount float64) error {
	// Validate parameters
	if err := validation.ValidateUUID(budgetID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if spentAmount < 0 {
		return errors.New("spent amount cannot be negative")
	}

	query := `
		UPDATE family_budget.budgets
		SET spent = $2, updated_at = NOW()
		WHERE id = $1 AND is_active = true`

	result, err := r.db.Exec(ctx, query, budgetID, spentAmount)
	if err != nil {
		return fmt.Errorf("failed to update spent amount: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget with id %s not found", budgetID)
	}

	return nil
}

// Delete soft deletes a budget (sets is_active to false)
func (r *PostgreSQLRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid family ID parameter: %w", err)
	}

	query := `
		UPDATE family_budget.budgets
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND family_id = $2 AND is_active = true`

	result, err := r.db.Exec(ctx, query, id, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("budget with id %s not found", id)
	}

	return nil
}

// GetAlerts retrieves all alerts for a budget
func (r *PostgreSQLRepository) GetAlerts(ctx context.Context, budgetID uuid.UUID) ([]*Alert, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(budgetID); err != nil {
		return nil, fmt.Errorf("invalid budget ID parameter: %w", err)
	}

	query := `
		SELECT id, budget_id, threshold_percentage, is_triggered, triggered_at, created_at
		FROM family_budget.budget_alerts
		WHERE budget_id = $1
		ORDER BY threshold_percentage ASC`

	rows, err := r.db.Query(ctx, query, budgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var alert Alert

		err = rows.Scan(
			&alert.ID, &alert.BudgetID, &alert.ThresholdPercentage,
			&alert.IsTriggered, &alert.TriggeredAt, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget alert: %w", err)
		}

		alerts = append(alerts, &alert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return alerts, nil
}

// CreateAlert creates a new budget alert
func (r *PostgreSQLRepository) CreateAlert(ctx context.Context, alert *Alert) error {
	// Validate parameters
	if err := validation.ValidateUUID(alert.ID); err != nil {
		return fmt.Errorf("invalid alert ID: %w", err)
	}
	if err := validation.ValidateUUID(alert.BudgetID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if alert.ThresholdPercentage <= 0 || alert.ThresholdPercentage > 100 {
		return errors.New("threshold percentage must be between 1 and 100")
	}

	alert.CreatedAt = time.Now()

	query := `
		INSERT INTO family_budget.budget_alerts (
			id, budget_id, threshold_percentage, is_triggered, created_at
		) VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(ctx, query,
		alert.ID, alert.BudgetID, alert.ThresholdPercentage,
		alert.IsTriggered, alert.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("alert with threshold %d%% already exists for this budget", alert.ThresholdPercentage)
		}
		return fmt.Errorf("failed to create budget alert: %w", err)
	}

	return nil
}

// GetByFamilyAndCategory retrieves budgets by family ID and optionally by category ID
func (r *PostgreSQLRepository) GetByFamilyAndCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	// Validate UUID parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	var query string
	var args []interface{}

	if categoryID != nil {
		if err := validation.ValidateUUID(*categoryID); err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}

		query = `
			SELECT id, name, amount, spent, period, start_date, end_date,
				   category_id, family_id, is_active, created_at, updated_at
			FROM family_budget.budgets
			WHERE family_id = $1 AND category_id = $2 AND is_active = true
			ORDER BY created_at DESC`
		args = []interface{}{familyID, *categoryID}
	} else {
		query = `
			SELECT id, name, amount, spent, period, start_date, end_date,
				   category_id, family_id, is_active, created_at, updated_at
			FROM family_budget.budgets
			WHERE family_id = $1 AND is_active = true
			ORDER BY created_at DESC`
		args = []interface{}{familyID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query budgets: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b budget.Budget
		var categoryIDPtr *uuid.UUID

		err = rows.Scan(
			&b.ID,
			&b.Name,
			&b.Amount,
			&b.Spent,
			&b.Period,
			&b.StartDate,
			&b.EndDate,
			&categoryIDPtr,
			&b.FamilyID,
			&b.IsActive,
			&b.CreatedAt,
			&b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		// Set category ID if not null
		if categoryIDPtr != nil {
			b.CategoryID = categoryIDPtr
		}

		budgets = append(budgets, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetByPeriod retrieves budgets by family ID and date range
func (r *PostgreSQLRepository) GetByPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM family_budget.budgets
		WHERE family_id = $1 AND is_active = true
		AND (
			(start_date <= $3 AND end_date >= $2) OR
			(start_date >= $2 AND start_date <= $3) OR
			(end_date >= $2 AND end_date <= $3)
		)
		ORDER BY start_date ASC`

	rows, err := r.db.Query(ctx, query, familyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query budgets by period: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b budget.Budget
		var categoryIDPtr *uuid.UUID

		err = rows.Scan(
			&b.ID,
			&b.Name,
			&b.Amount,
			&b.Spent,
			&b.Period,
			&b.StartDate,
			&b.EndDate,
			&categoryIDPtr,
			&b.FamilyID,
			&b.IsActive,
			&b.CreatedAt,
			&b.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget: %w", err)
		}

		// Set category ID if not null
		if categoryIDPtr != nil {
			b.CategoryID = categoryIDPtr
		}

		budgets = append(budgets, &b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// UsageStats holds comprehensive budget usage statistics
type UsageStats struct {
	BudgetID        uuid.UUID     `json:"budget_id"`
	BudgetName      string        `json:"budget_name"`
	BudgetAmount    float64       `json:"budget_amount"`
	SpentAmount     float64       `json:"spent_amount"`
	RemainingAmount float64       `json:"remaining_amount"`
	UsagePercentage float64       `json:"usage_percentage"`
	Period          budget.Period `json:"period"`
	StartDate       time.Time     `json:"start_date"`
	EndDate         time.Time     `json:"end_date"`
	DaysRemaining   int           `json:"days_remaining"`
	Status          string        `json:"status"` // 'safe', 'on_track', 'warning', 'over_budget'
	CategoryName    string        `json:"category_name,omitempty"`
}

// Alert represents a budget alert
type Alert struct {
	ID                  uuid.UUID  `json:"id"`
	BudgetID            uuid.UUID  `json:"budget_id"`
	ThresholdPercentage int        `json:"threshold_percentage"`
	IsTriggered         bool       `json:"is_triggered"`
	TriggeredAt         *time.Time `json:"triggered_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}
