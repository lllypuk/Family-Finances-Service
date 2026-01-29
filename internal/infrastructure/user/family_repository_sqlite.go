package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteFamilyRepository implements family repository using SQLite
type SQLiteFamilyRepository struct {
	db *sql.DB
}

// FamilyStatistics holds family statistics
type FamilyStatistics struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Currency         string    `json:"currency"`
	CreatedAt        time.Time `json:"created_at"`
	UserCount        int       `json:"user_count"`
	CategoryCount    int       `json:"category_count"`
	TransactionCount int       `json:"transaction_count"`
	BudgetCount      int       `json:"budget_count"`
	TotalIncome      float64   `json:"total_income"`
	TotalExpenses    float64   `json:"total_expenses"`
	Balance          float64   `json:"balance"`
}

// NewSQLiteFamilyRepository creates a new SQLite family repository
func NewSQLiteFamilyRepository(db *sql.DB) *SQLiteFamilyRepository {
	return &SQLiteFamilyRepository{
		db: db,
	}
}

// Create creates a new family in the database
func (r *SQLiteFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before creating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = validation.SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := validation.ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Set timestamps
	now := time.Now()
	family.CreatedAt = now
	family.UpdatedAt = now

	query := `
		INSERT INTO families (id, name, currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		family.ID.String(), family.Name, family.Currency, family.CreatedAt, family.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}

	return nil
}

// Get retrieves the single family from the database
func (r *SQLiteFamilyRepository) Get(ctx context.Context) (*user.Family, error) {
	query := `
		SELECT id, name, currency, created_at, updated_at
		FROM families
		LIMIT 1`

	var family user.Family
	var idStr string

	err := r.db.QueryRowContext(ctx, query).Scan(
		&idStr, &family.Name, &family.Currency, &family.CreatedAt, &family.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("family not found")
		}
		return nil, fmt.Errorf("failed to get family: %w", err)
	}

	// Parse UUID
	family.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	return &family, nil
}

// Update updates an existing family
func (r *SQLiteFamilyRepository) Update(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before updating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = validation.SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := validation.ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Update timestamp
	family.UpdatedAt = time.Now()

	query := `
		UPDATE families
		SET name = ?, currency = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		family.Name, family.Currency, family.UpdatedAt, family.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to update family: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("family with id %s not found", family.ID)
	}

	return nil
}

// Exists checks if a family exists in the database
func (r *SQLiteFamilyRepository) Exists(ctx context.Context) (bool, error) {
	query := `SELECT COUNT(*) FROM families`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check family existence: %w", err)
	}

	return count > 0, nil
}

// GetFamilyStatistics returns statistics about the single family
func (r *SQLiteFamilyRepository) GetFamilyStatistics(ctx context.Context) (*FamilyStatistics, error) {
	query := `
		SELECT
			f.id,
			f.name,
			f.currency,
			f.created_at,
			COUNT(DISTINCT u.id) as user_count,
			COUNT(DISTINCT c.id) as category_count,
			COUNT(DISTINCT t.id) as transaction_count,
			COUNT(DISTINCT b.id) as budget_count,
			COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) as total_expenses
		FROM families f
		LEFT JOIN users u ON f.id = u.family_id AND u.is_active = 1
		LEFT JOIN categories c ON f.id = c.family_id AND c.is_active = 1
		LEFT JOIN transactions t ON f.id = t.family_id
		LEFT JOIN budgets b ON f.id = b.family_id AND b.is_active = 1
		GROUP BY f.id, f.name, f.currency, f.created_at
		LIMIT 1`

	var stats FamilyStatistics
	var idStr string

	err := r.db.QueryRowContext(ctx, query).Scan(
		&idStr, &stats.Name, &stats.Currency, &stats.CreatedAt,
		&stats.UserCount, &stats.CategoryCount, &stats.TransactionCount,
		&stats.BudgetCount, &stats.TotalIncome, &stats.TotalExpenses,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("family not found")
		}
		return nil, fmt.Errorf("failed to get family statistics: %w", err)
	}

	// Parse UUID
	stats.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	stats.Balance = stats.TotalIncome - stats.TotalExpenses

	return &stats, nil
}

// CreateWithTransaction creates a family within a database transaction
func (r *SQLiteFamilyRepository) CreateWithTransaction(ctx context.Context, tx *sql.Tx, family *user.Family) error {
	// Validate family ID parameter before creating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = validation.SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := validation.ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Set timestamps
	now := time.Now()
	family.CreatedAt = now
	family.UpdatedAt = now

	query := `
		INSERT INTO families (id, name, currency, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`

	_, err := tx.ExecContext(ctx, query,
		family.ID.String(), family.Name, family.Currency, family.CreatedAt, family.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}

	return nil
}
