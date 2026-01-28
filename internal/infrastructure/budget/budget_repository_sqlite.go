package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements budget repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// UsageStats holds budget usage statistics
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

// NewSQLiteRepository creates a new SQLite budget repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// scanBudgetRow scans a single row from SQL query into a Budget struct
func scanBudgetRow(rows *sql.Rows) (*budget.Budget, error) {
	var b budget.Budget
	var idStr, periodStr, familyIDStr string
	var categoryIDStr *string
	var isActiveInt int

	err := rows.Scan(
		&idStr, &b.Name, &b.Amount, &b.Spent, &periodStr,
		&b.StartDate, &b.EndDate, &categoryIDStr, &familyIDStr,
		&isActiveInt, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan budget: %w", err)
	}

	// Parse UUID fields
	b.ID, _ = uuid.Parse(idStr)
	b.FamilyID, _ = uuid.Parse(familyIDStr)
	if categoryIDStr != nil && *categoryIDStr != "" {
		categoryID, _ := uuid.Parse(*categoryIDStr)
		b.CategoryID = &categoryID
	}

	b.Period = budget.Period(periodStr)
	b.IsActive = sqlitehelpers.IntToBool(isActiveInt)

	return &b, nil
}

// Create creates a new budget in the database
func (r *SQLiteRepository) Create(ctx context.Context, b *budget.Budget) error {
	// Validate budget parameters
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateBudgetPeriod(b.Period); err != nil {
		return fmt.Errorf("invalid period: %w", err)
	}
	if err := validation.ValidateBudgetAmount(b.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := validation.ValidateBudgetName(b.Name); err != nil {
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
		INSERT INTO budgets (
			id, name, amount, spent, period, start_date, end_date,
			category_id, family_id, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(b.ID),
		b.Name,
		b.Amount,
		b.Spent,
		string(b.Period),
		b.StartDate,
		b.EndDate,
		sqlitehelpers.UUIDPtrToString(b.CategoryID),
		sqlitehelpers.UUIDToString(b.FamilyID),
		sqlitehelpers.BoolToInt(b.IsActive),
		b.CreatedAt,
		b.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("budget with name '%s' already exists for this period", b.Name)
		}
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

// GetByID retrieves a budget by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM budgets
		WHERE id = ?`

	var b budget.Budget
	var idStr, periodStr, familyIDStr string
	var categoryIDStr *string
	var isActiveInt int

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &b.Name, &b.Amount, &b.Spent, &periodStr,
		&b.StartDate, &b.EndDate, &categoryIDStr, &familyIDStr,
		&isActiveInt, &b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get budget by id: %w", err)
	}

	// Parse UUID fields
	b.ID, _ = uuid.Parse(idStr)
	b.FamilyID, _ = uuid.Parse(familyIDStr)
	if categoryIDStr != nil && *categoryIDStr != "" {
		categoryID, _ := uuid.Parse(*categoryIDStr)
		b.CategoryID = &categoryID
	}

	b.Period = budget.Period(periodStr)
	b.IsActive = sqlitehelpers.IntToBool(isActiveInt)

	return &b, nil
}

// GetByFamilyID retrieves all budgets belonging to a specific family
func (r *SQLiteRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		ORDER BY start_date DESC, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by family id: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b *budget.Budget
		b, err = scanBudgetRow(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetActiveBudgets retrieves all active budgets for a family
func (r *SQLiteRepository) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	now := time.Now()
	query := `
		SELECT id, name, amount, spent, period, start_date, end_date,
			   category_id, family_id, is_active, created_at, updated_at
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		AND start_date <= ? AND end_date >= ?
		ORDER BY start_date DESC, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b *budget.Budget
		b, err = scanBudgetRow(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetUsageStats returns comprehensive budget usage statistics
func (r *SQLiteRepository) GetUsageStats(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*UsageStats, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	now := time.Now()
	// SQLite uses julianday for date calculations
	// Format dates to ISO 8601 for proper SQLite comparison
	nowStr := now.Format(time.RFC3339)
	// Calculate spent dynamically using CTE to avoid code duplication
	query := `
		WITH budget_spent AS (
			SELECT
				b.id,
				COALESCE(SUM(t.amount), 0) as spent
			FROM budgets b
			LEFT JOIN transactions t ON
				t.type = 'expense'
				AND t.date BETWEEN b.start_date AND b.end_date
				AND (b.category_id IS NULL OR t.category_id = b.category_id)
				AND t.family_id = b.family_id
			WHERE b.family_id = ? AND b.is_active = 1
			AND ? BETWEEN b.start_date AND b.end_date
			GROUP BY b.id
		)
		SELECT
			b.id,
			b.name,
			b.amount,
			bs.spent,
			b.period,
			b.start_date,
			b.end_date,
			(b.amount - bs.spent) as remaining_amount,
			CASE WHEN b.amount > 0 THEN ROUND((bs.spent / b.amount * 100), 2) ELSE 0 END as usage_percentage,
			CAST((julianday(substr(b.end_date, 1, 10)) - julianday('now')) AS INTEGER) as days_remaining,
			CASE
				WHEN bs.spent > b.amount THEN 'over_budget'
				WHEN bs.spent > (b.amount * 0.8) THEN 'warning'
				WHEN bs.spent > (b.amount * 0.5) THEN 'on_track'
				ELSE 'safe'
			END as status,
			c.name as category_name
		FROM budgets b
		JOIN budget_spent bs ON b.id = bs.id
		LEFT JOIN categories c ON b.category_id = c.id
		ORDER BY usage_percentage DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), nowStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget usage stats: %w", err)
	}
	defer rows.Close()

	var stats []*UsageStats
	for rows.Next() {
		var stat UsageStats
		var budgetIDStr, periodStr string
		var categoryName *string

		err = rows.Scan(
			&budgetIDStr, &stat.BudgetName, &stat.BudgetAmount, &stat.SpentAmount,
			&periodStr, &stat.StartDate, &stat.EndDate, &stat.RemainingAmount,
			&stat.UsagePercentage, &stat.DaysRemaining, &stat.Status, &categoryName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget usage stat: %w", err)
		}

		stat.BudgetID, _ = uuid.Parse(budgetIDStr)
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
func (r *SQLiteRepository) Update(ctx context.Context, b *budget.Budget) error {
	// Validate budget parameters
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateBudgetPeriod(b.Period); err != nil {
		return fmt.Errorf("invalid period: %w", err)
	}
	if err := validation.ValidateBudgetAmount(b.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := validation.ValidateBudgetName(b.Name); err != nil {
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
		UPDATE budgets
		SET name = ?, amount = ?, spent = ?, period = ?,
			start_date = ?, end_date = ?, category_id = ?, is_active = ?, updated_at = ?
		WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query,
		b.Name,
		b.Amount,
		b.Spent,
		string(b.Period),
		b.StartDate,
		b.EndDate,
		sqlitehelpers.UUIDPtrToString(b.CategoryID),
		sqlitehelpers.BoolToInt(b.IsActive),
		b.UpdatedAt,
		sqlitehelpers.UUIDToString(b.ID),
		sqlitehelpers.UUIDToString(b.FamilyID),
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("budget with name '%s' already exists for this period", b.Name)
		}
		return fmt.Errorf("failed to update budget: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("budget with id %s not found", b.ID)
	}

	return nil
}

// UpdateSpentAmount updates the spent amount for a budget
func (r *SQLiteRepository) UpdateSpentAmount(ctx context.Context, budgetID uuid.UUID, spentAmount float64) error {
	// Validate parameters
	if err := validation.ValidateUUID(budgetID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if spentAmount < 0 {
		return errors.New("spent amount cannot be negative")
	}

	query := `
		UPDATE budgets
		SET spent = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, spentAmount, sqlitehelpers.UUIDToString(budgetID))
	if err != nil {
		return fmt.Errorf("failed to update spent amount: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("budget with id %s not found", budgetID)
	}

	return nil
}

// RecalculateSpent recalculates the spent amount for a budget based on transactions
// This replaces the PostgreSQL trigger update_budget_spent
func (r *SQLiteRepository) RecalculateSpent(ctx context.Context, budgetID uuid.UUID) error {
	// Validate budget ID
	if err := validation.ValidateUUID(budgetID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}

	// Recalculate spent amount from transactions
	query := `
		UPDATE budgets
		SET spent = (
			SELECT COALESCE(SUM(t.amount), 0)
			FROM transactions t
			WHERE t.type = 'expense'
			AND t.date BETWEEN budgets.start_date AND budgets.end_date
			AND (budgets.category_id IS NULL OR t.category_id = budgets.category_id)
			AND t.family_id = budgets.family_id
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(budgetID))
	if err != nil {
		return fmt.Errorf("failed to recalculate spent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("budget with id %s not found", budgetID)
	}

	return nil
}

// FindBudgetsAffectedByTransaction finds budgets that should be recalculated after a transaction change
// This helper method is used by TransactionRepository to trigger budget updates
func (r *SQLiteRepository) FindBudgetsAffectedByTransaction(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID uuid.UUID,
	transactionDate time.Time,
) ([]uuid.UUID, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateUUID(categoryID); err != nil {
		return nil, fmt.Errorf("invalid category ID: %w", err)
	}

	query := `
		SELECT id
		FROM budgets
		WHERE family_id = ?
		AND is_active = 1
		AND ? BETWEEN start_date AND end_date
		AND (category_id IS NULL OR category_id = ?)`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		sqlitehelpers.UUIDToString(familyID),
		transactionDate,
		sqlitehelpers.UUIDToString(categoryID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find affected budgets: %w", err)
	}
	defer rows.Close()

	var budgetIDs []uuid.UUID
	for rows.Next() {
		var idStr string
		if scanErr := rows.Scan(&idStr); scanErr != nil {
			return nil, fmt.Errorf("failed to scan budget ID: %w", scanErr)
		}
		id, _ := uuid.Parse(idStr)
		budgetIDs = append(budgetIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgetIDs, nil
}

// Delete soft deletes a budget (sets is_active to false)
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid family ID parameter: %w", err)
	}

	query := `
		UPDATE budgets
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND family_id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("budget with id %s not found", id)
	}

	return nil
}

// GetAlerts retrieves all alerts for a budget
func (r *SQLiteRepository) GetAlerts(ctx context.Context, budgetID uuid.UUID) ([]*Alert, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(budgetID); err != nil {
		return nil, fmt.Errorf("invalid budget ID parameter: %w", err)
	}

	query := `
		SELECT id, budget_id, threshold_percentage, is_triggered, triggered_at, created_at
		FROM budget_alerts
		WHERE budget_id = ?
		ORDER BY threshold_percentage ASC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(budgetID))
	if err != nil {
		return nil, fmt.Errorf("failed to get budget alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var idStr, budgetIDStr string
		var isTriggeredInt int
		var triggeredAtStr *string

		err = rows.Scan(
			&idStr, &budgetIDStr, &alert.ThresholdPercentage,
			&isTriggeredInt, &triggeredAtStr, &alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget alert: %w", err)
		}

		alert.ID, _ = uuid.Parse(idStr)
		alert.BudgetID, _ = uuid.Parse(budgetIDStr)
		alert.IsTriggered = sqlitehelpers.IntToBool(isTriggeredInt)

		// Parse triggered_at timestamp if present
		if triggeredAtStr != nil && *triggeredAtStr != "" {
			triggeredAt, parseErr := time.Parse(time.RFC3339, *triggeredAtStr)
			if parseErr == nil {
				alert.TriggeredAt = &triggeredAt
			}
		}

		alerts = append(alerts, &alert)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return alerts, nil
}

// CreateAlert creates a new budget alert
func (r *SQLiteRepository) CreateAlert(ctx context.Context, alert *Alert) error {
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
		INSERT INTO budget_alerts (
			id, budget_id, threshold_percentage, is_triggered, created_at
		) VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(alert.ID),
		sqlitehelpers.UUIDToString(alert.BudgetID),
		alert.ThresholdPercentage,
		sqlitehelpers.BoolToInt(alert.IsTriggered),
		alert.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("alert with threshold %d%% already exists for this budget", alert.ThresholdPercentage)
		}
		return fmt.Errorf("failed to create budget alert: %w", err)
	}

	return nil
}

// GetByFamilyAndCategory retrieves budgets by family ID and optionally by category ID
func (r *SQLiteRepository) GetByFamilyAndCategory(
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
			FROM budgets
			WHERE family_id = ? AND category_id = ? AND is_active = 1
			ORDER BY created_at DESC`
		args = []interface{}{sqlitehelpers.UUIDToString(familyID), sqlitehelpers.UUIDToString(*categoryID)}
	} else {
		query = `
			SELECT id, name, amount, spent, period, start_date, end_date,
				   category_id, family_id, is_active, created_at, updated_at
			FROM budgets
			WHERE family_id = ? AND is_active = 1
			ORDER BY created_at DESC`
		args = []interface{}{sqlitehelpers.UUIDToString(familyID)}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query budgets: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b *budget.Budget
		b, err = scanBudgetRow(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// GetByPeriod retrieves budgets by family ID and date range
func (r *SQLiteRepository) GetByPeriod(
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
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		AND (
			(start_date <= ? AND end_date >= ?) OR
			(start_date >= ? AND start_date <= ?) OR
			(end_date >= ? AND end_date <= ?)
		)
		ORDER BY start_date ASC`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		sqlitehelpers.UUIDToString(familyID),
		endDate,
		startDate,
		startDate,
		endDate,
		startDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query budgets by period: %w", err)
	}
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		var b *budget.Budget
		b, err = scanBudgetRow(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}
