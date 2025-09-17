package transaction

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/infrastructure/validation"
)

// PostgreSQLRepository implements transaction repository using PostgreSQL
type PostgreSQLRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLRepository creates a new PostgreSQL transaction repository
func NewPostgreSQLRepository(db *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// ValidateTransactionType validates transaction type
func ValidateTransactionType(transactionType transaction.Type) error {
	if transactionType != transaction.TypeIncome && transactionType != transaction.TypeExpense {
		return errors.New("invalid transaction type")
	}
	return nil
}

// ValidateAmount validates transaction amount
func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if amount > 999999999.99 {
		return errors.New("amount too large")
	}
	return nil
}

// ValidateDescription validates transaction description
func ValidateDescription(description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("description cannot be empty")
	}
	if len(description) > 1000 {
		return errors.New("description too long")
	}
	return nil
}

// Create creates a new transaction in the database
func (r *PostgreSQLRepository) Create(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}
	if err := ValidateAmount(t.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := ValidateDescription(t.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	// Validate date
	if t.Date.After(time.Now().AddDate(1, 0, 0)) {
		return errors.New("transaction date cannot be more than 1 year in the future")
	}
	if t.Date.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return errors.New("transaction date too old")
	}

	// Set timestamps
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	// Convert tags to JSONB
	tagsJSON := "[]"
	if len(t.Tags) > 0 {
		var tagParts []string
		for _, tag := range t.Tags {
			// Escape quotes in tags
			escapedTag := strings.ReplaceAll(tag, `"`, `\"`)
			tagParts = append(tagParts, fmt.Sprintf(`"%s"`, escapedTag))
		}
		tagsJSON = "[" + strings.Join(tagParts, ",") + "]"
	}

	query := `
		INSERT INTO family_budget.transactions (
			id, amount, description, date, type, category_id, user_id, family_id,
			tags, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, query,
		t.ID, t.Amount, t.Description, t.Date, string(t.Type),
		t.CategoryID, t.UserID, t.FamilyID, tagsJSON, t.CreatedAt, t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a transaction by their ID
func (r *PostgreSQLRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, amount, description, date, type, category_id, user_id, family_id,
			   tags, created_at, updated_at
		FROM family_budget.transactions
		WHERE id = $1`

	var t transaction.Transaction
	var typeStr string
	var tagsJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Amount, &t.Description, &t.Date, &typeStr,
		&t.CategoryID, &t.UserID, &t.FamilyID, &tagsJSON, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get transaction by id: %w", err)
	}

	t.Type = transaction.Type(typeStr)

	// Parse tags from JSONB
	if err := r.parseTags(tagsJSON, &t.Tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}

	return &t, nil
}

// GetByFilter retrieves transactions based on filter criteria
func (r *PostgreSQLRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	// Validate family ID
	if err := validation.ValidateUUID(filter.FamilyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	// Build dynamic query
	query := `
		SELECT t.id, t.amount, t.description, t.date, t.type, t.category_id, t.user_id, t.family_id,
			   t.tags, t.created_at, t.updated_at
		FROM family_budget.transactions t`

	var conditions []string
	var args []any
	argIndex := 1

	// Family ID is always required
	conditions = append(conditions, fmt.Sprintf("t.family_id = $%d", argIndex))
	args = append(args, filter.FamilyID)
	argIndex++

	// Optional filters
	if filter.UserID != nil {
		if err := validation.ValidateUUID(*filter.UserID); err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("t.user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.CategoryID != nil {
		if err := validation.ValidateUUID(*filter.CategoryID); err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("t.category_id = $%d", argIndex))
		args = append(args, *filter.CategoryID)
		argIndex++
	}

	if filter.Type != nil {
		if err := ValidateTransactionType(*filter.Type); err != nil {
			return nil, fmt.Errorf("invalid transaction type: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("t.type = $%d", argIndex))
		args = append(args, string(*filter.Type))
		argIndex++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("t.date >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.date <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	if filter.AmountFrom != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount >= $%d", argIndex))
		args = append(args, *filter.AmountFrom)
		argIndex++
	}

	if filter.AmountTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.amount <= $%d", argIndex))
		args = append(args, *filter.AmountTo)
		argIndex++
	}

	if filter.Description != "" {
		conditions = append(conditions, fmt.Sprintf("t.description ILIKE $%d", argIndex))
		args = append(args, "%"+filter.Description+"%")
		argIndex++
	}

	// Tags filter using JSONB operations
	if len(filter.Tags) > 0 {
		var tagConditions []string
		for _, tag := range filter.Tags {
			tagConditions = append(tagConditions, fmt.Sprintf("t.tags @> $%d", argIndex))
			args = append(args, fmt.Sprintf(`["%s"]`, tag))
			argIndex++
		}
		conditions = append(conditions, "("+strings.Join(tagConditions, " OR ")+")")
	}

	// Build final query
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY t.date DESC, t.created_at DESC"

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by filter: %w", err)
	}
	defer rows.Close()

	var transactions []*transaction.Transaction
	for rows.Next() {
		var t transaction.Transaction
		var typeStr string
		var tagsJSON []byte

		err = rows.Scan(
			&t.ID, &t.Amount, &t.Description, &t.Date, &typeStr,
			&t.CategoryID, &t.UserID, &t.FamilyID, &tagsJSON, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		t.Type = transaction.Type(typeStr)

		// Parse tags from JSONB
		if err := r.parseTags(tagsJSON, &t.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}

		transactions = append(transactions, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}

// Update updates an existing transaction
func (r *PostgreSQLRepository) Update(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}
	if err := ValidateAmount(t.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := ValidateDescription(t.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	// Update timestamp
	t.UpdatedAt = time.Now()

	// Convert tags to JSONB
	tagsJSON := "[]"
	if len(t.Tags) > 0 {
		var tagParts []string
		for _, tag := range t.Tags {
			// Escape quotes in tags
			escapedTag := strings.ReplaceAll(tag, `"`, `\"`)
			tagParts = append(tagParts, fmt.Sprintf(`"%s"`, escapedTag))
		}
		tagsJSON = "[" + strings.Join(tagParts, ",") + "]"
	}

	query := `
		UPDATE family_budget.transactions
		SET amount = $2, description = $3, date = $4, type = $5, category_id = $6,
			user_id = $7, tags = $8, updated_at = $9
		WHERE id = $1 AND family_id = $10`

	result, err := r.db.Exec(ctx, query,
		t.ID, t.Amount, t.Description, t.Date, string(t.Type),
		t.CategoryID, t.UserID, tagsJSON, t.UpdatedAt, t.FamilyID,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("transaction with id %s not found", t.ID)
	}

	return nil
}

// Delete deletes a transaction
func (r *PostgreSQLRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid family ID parameter: %w", err)
	}

	query := `DELETE FROM family_budget.transactions WHERE id = $1 AND family_id = $2`

	result, err := r.db.Exec(ctx, query, id, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("transaction with id %s not found", id)
	}

	return nil
}

// GetTransactionSummary returns transaction summary for a family
func (r *PostgreSQLRepository) GetTransactionSummary(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*TransactionSummary, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	query := `
		SELECT
			COUNT(*) as total_count,
			COUNT(CASE WHEN type = 'income' THEN 1 END) as income_count,
			COUNT(CASE WHEN type = 'expense' THEN 1 END) as expense_count,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expenses,
			COALESCE(AVG(CASE WHEN type = 'income' THEN amount END), 0) as avg_income,
			COALESCE(AVG(CASE WHEN type = 'expense' THEN amount END), 0) as avg_expense
		FROM family_budget.transactions
		WHERE family_id = $1 AND date BETWEEN $2 AND $3`

	var summary TransactionSummary
	err := r.db.QueryRow(ctx, query, familyID, startDate, endDate).Scan(
		&summary.TotalCount, &summary.IncomeCount, &summary.ExpenseCount,
		&summary.TotalIncome, &summary.TotalExpenses, &summary.AvgIncome, &summary.AvgExpense,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction summary: %w", err)
	}

	summary.Balance = summary.TotalIncome - summary.TotalExpenses
	summary.FamilyID = familyID
	summary.StartDate = startDate
	summary.EndDate = endDate

	return &summary, nil
}

// GetMonthlySummary returns monthly transaction summary
func (r *PostgreSQLRepository) GetMonthlySummary(
	ctx context.Context,
	familyID uuid.UUID,
	year, month int,
) ([]*MonthlySummaryItem, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	query := `
		SELECT
			c.name as category_name,
			t.type,
			COUNT(*) as transaction_count,
			SUM(t.amount) as total_amount,
			AVG(t.amount) as avg_amount
		FROM family_budget.transactions t
		JOIN family_budget.categories c ON t.category_id = c.id
		WHERE t.family_id = $1
		AND EXTRACT(YEAR FROM t.date) = $2
		AND EXTRACT(MONTH FROM t.date) = $3
		GROUP BY c.id, c.name, t.type
		ORDER BY total_amount DESC`

	rows, err := r.db.Query(ctx, query, familyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly summary: %w", err)
	}
	defer rows.Close()

	var summaries []*MonthlySummaryItem
	for rows.Next() {
		var item MonthlySummaryItem
		var typeStr string

		err = rows.Scan(&item.CategoryName, &typeStr, &item.TransactionCount, &item.TotalAmount, &item.AvgAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly summary item: %w", err)
		}

		item.Type = transaction.Type(typeStr)
		summaries = append(summaries, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return summaries, nil
}

// GetByFamilyID retrieves transactions by family ID with pagination
func (r *PostgreSQLRepository) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	// Validate pagination parameters
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Maximum limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, amount, type, description, category_id, user_id, family_id,
			   date, tags, created_at, updated_at
		FROM family_budget.transactions
		WHERE family_id = $1 		ORDER BY date DESC, created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, familyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions by family: %w", err)
	}
	defer rows.Close()

	var transactions []*transaction.Transaction
	for rows.Next() {
		var t transaction.Transaction
		var tagsJSON []byte

		err := rows.Scan(
			&t.ID,
			&t.Amount,
			&t.Type,
			&t.Description,
			&t.CategoryID,
			&t.UserID,
			&t.FamilyID,
			&t.Date,
			&tagsJSON,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Parse tags from JSONB
		if err := r.parseTags(tagsJSON, &t.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}

		transactions = append(transactions, &t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}

// GetTotalByCategory calculates total amount for transactions by category and type
func (r *PostgreSQLRepository) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	transactionType transaction.Type,
) (float64, error) {
	// Validate parameters
	if err := validation.ValidateUUID(categoryID); err != nil {
		return 0, fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM family_budget.transactions
		WHERE category_id = $1 AND type = $2`

	var total float64
	err := r.db.QueryRow(ctx, query, categoryID, transactionType).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by category: %w", err)
	}

	return total, nil
}

// GetTotalByFamilyAndDateRange calculates total amount for transactions by family and date range
func (r *PostgreSQLRepository) GetTotalByFamilyAndDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return 0, fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM family_budget.transactions
		WHERE family_id = $1 AND type = $2 AND date >= $3 AND date <= $4 AND deleted_at IS NULL`

	var total float64
	err := r.db.QueryRow(ctx, query, familyID, transactionType, startDate, endDate).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by family and date range: %w", err)
	}

	return total, nil
}

// GetTotalByCategoryAndDateRange calculates total amount for transactions by category and date range
func (r *PostgreSQLRepository) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	// Validate parameters
	if err := validation.ValidateUUID(categoryID); err != nil {
		return 0, fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM family_budget.transactions
		WHERE category_id = $1 AND type = $2 AND date >= $3 AND date <= $4 AND deleted_at IS NULL`

	var total float64
	err := r.db.QueryRow(ctx, query, categoryID, transactionType, startDate, endDate).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by category and date range: %w", err)
	}

	return total, nil
}

// parseTags parses tags from JSONB format
func (r *PostgreSQLRepository) parseTags(tagsJSON []byte, tags *[]string) error {
	// Simple JSON array parsing for tags
	// In production, you might want to use a proper JSON library
	jsonStr := string(tagsJSON)
	if jsonStr == "[]" || jsonStr == "" {
		*tags = []string{}
		return nil
	}

	// Remove brackets and split by comma
	jsonStr = strings.Trim(jsonStr, "[]")
	if jsonStr == "" {
		*tags = []string{}
		return nil
	}

	parts := strings.Split(jsonStr, ",")
	*tags = make([]string, 0, len(parts))

	for _, part := range parts {
		// Remove quotes and whitespace
		tag := strings.Trim(strings.TrimSpace(part), `"`)
		if tag != "" {
			*tags = append(*tags, tag)
		}
	}

	return nil
}

// TransactionSummary holds transaction summary data
type TransactionSummary struct {
	FamilyID      uuid.UUID `json:"family_id"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	TotalCount    int       `json:"total_count"`
	IncomeCount   int       `json:"income_count"`
	ExpenseCount  int       `json:"expense_count"`
	TotalIncome   float64   `json:"total_income"`
	TotalExpenses float64   `json:"total_expenses"`
	Balance       float64   `json:"balance"`
	AvgIncome     float64   `json:"avg_income"`
	AvgExpense    float64   `json:"avg_expense"`
}

// MonthlySummaryItem holds monthly summary data by category
type MonthlySummaryItem struct {
	CategoryName     string           `json:"category_name"`
	Type             transaction.Type `json:"type"`
	TransactionCount int              `json:"transaction_count"`
	TotalAmount      float64          `json:"total_amount"`
	AvgAmount        float64          `json:"avg_amount"`
}
