package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

// Currency validation constants
const (
	currencyCodeLength = 3
)

// PostgreSQLFamilyRepository implements family repository using PostgreSQL
type PostgreSQLFamilyRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLFamilyRepository creates a new PostgreSQL family repository
func NewPostgreSQLFamilyRepository(db *pgxpool.Pool) *PostgreSQLFamilyRepository {
	return &PostgreSQLFamilyRepository{
		db: db,
	}
}

// ValidateCurrency validates currency code format
func ValidateCurrency(currency string) error {
	if currency == "" {
		return errors.New("currency cannot be empty")
	}

	// Convert to uppercase for consistency
	currency = strings.ToUpper(strings.TrimSpace(currency))

	// Must be exactly 3 characters (ISO 4217)
	if len(currency) != currencyCodeLength {
		return errors.New("currency must be exactly 3 characters")
	}

	// Check for valid characters (A-Z only)
	for _, char := range currency {
		if char < 'A' || char > 'Z' {
			return errors.New("currency must contain only uppercase letters")
		}
	}

	// Common currency codes validation (extend as needed)
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "RUB": true,
		"CNY": true, "CAD": true, "AUD": true, "CHF": true, "SEK": true,
		"NOK": true, "DKK": true, "PLN": true, "CZK": true, "HUF": true,
	}

	if !validCurrencies[currency] {
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	return nil
}

// SanitizeFamilyName sanitizes family name for safe storage
func SanitizeFamilyName(name string) string {
	return strings.TrimSpace(name)
}

// Create creates a new family in the database
func (r *PostgreSQLFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before creating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Set timestamps
	now := time.Now()
	family.CreatedAt = now
	family.UpdatedAt = now

	query := `
		INSERT INTO family_budget.families (id, name, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(ctx, query, family.ID, family.Name, family.Currency, family.CreatedAt, family.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}

	return nil
}

// GetByID retrieves a family by their ID
func (r *PostgreSQLFamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, currency, created_at, updated_at
		FROM family_budget.families
		WHERE id = $1`

	var family user.Family
	err := r.db.QueryRow(ctx, query, id).Scan(
		&family.ID, &family.Name, &family.Currency, &family.CreatedAt, &family.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("family with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get family by id: %w", err)
	}

	return &family, nil
}

// Update updates an existing family
func (r *PostgreSQLFamilyRepository) Update(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before updating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Update timestamp
	family.UpdatedAt = time.Now()

	query := `
		UPDATE family_budget.families
		SET name = $2, currency = $3, updated_at = $4
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, family.ID, family.Name, family.Currency, family.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update family: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("family with id %s not found", family.ID)
	}

	return nil
}

// Delete deletes a family (this will cascade to all related data due to FK constraints)
func (r *PostgreSQLFamilyRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// For family deletion, id and familyID should be the same
	if id != familyID {
		return errors.New("family id and familyID must be the same when deleting a family")
	}

	query := `DELETE FROM family_budget.families WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete family: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("family with id %s not found", id)
	}

	return nil
}

// GetFamilyStatistics returns statistics about the family
func (r *PostgreSQLFamilyRepository) GetFamilyStatistics(
	ctx context.Context,
	familyID uuid.UUID,
) (*FamilyStatistics, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

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
		FROM family_budget.families f
		LEFT JOIN family_budget.users u ON f.id = u.family_id AND u.is_active = true
		LEFT JOIN family_budget.categories c ON f.id = c.family_id AND c.is_active = true
		LEFT JOIN family_budget.transactions t ON f.id = t.family_id
		LEFT JOIN family_budget.budgets b ON f.id = b.family_id AND b.is_active = true
		WHERE f.id = $1
		GROUP BY f.id, f.name, f.currency, f.created_at`

	var stats FamilyStatistics
	err := r.db.QueryRow(ctx, query, familyID).Scan(
		&stats.ID, &stats.Name, &stats.Currency, &stats.CreatedAt,
		&stats.UserCount, &stats.CategoryCount, &stats.TransactionCount,
		&stats.BudgetCount, &stats.TotalIncome, &stats.TotalExpenses,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("family with id %s not found", familyID)
		}
		return nil, fmt.Errorf("failed to get family statistics: %w", err)
	}

	stats.Balance = stats.TotalIncome - stats.TotalExpenses

	return &stats, nil
}

// CreateWithTransaction creates a family within a database transaction
func (r *PostgreSQLFamilyRepository) CreateWithTransaction(ctx context.Context, tx pgx.Tx, family *user.Family) error {
	// Validate family ID parameter before creating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate and sanitize family name
	family.Name = SanitizeFamilyName(family.Name)
	if family.Name == "" {
		return errors.New("family name cannot be empty")
	}

	// Validate currency
	if err := ValidateCurrency(family.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}

	// Ensure currency is uppercase
	family.Currency = strings.ToUpper(family.Currency)

	// Set timestamps
	now := time.Now()
	family.CreatedAt = now
	family.UpdatedAt = now

	query := `
		INSERT INTO family_budget.families (id, name, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := tx.Exec(ctx, query, family.ID, family.Name, family.Currency, family.CreatedAt, family.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}

	return nil
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

// GetAllFamilies retrieves all families (admin function)
func (r *PostgreSQLFamilyRepository) GetAllFamilies(ctx context.Context, limit, offset int) ([]*user.Family, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, currency, created_at, updated_at
		FROM family_budget.families
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get families: %w", err)
	}
	defer rows.Close()

	var families []*user.Family
	for rows.Next() {
		var family user.Family
		err = rows.Scan(&family.ID, &family.Name, &family.Currency, &family.CreatedAt, &family.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, &family)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return families, nil
}
