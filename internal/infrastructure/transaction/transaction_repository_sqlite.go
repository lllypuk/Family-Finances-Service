package transaction

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements transaction repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// Summary holds transaction summary statistics
type Summary struct {
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

// MonthlySummaryItem holds monthly summary by category
type MonthlySummaryItem struct {
	CategoryName     string           `json:"category_name"`
	Type             transaction.Type `json:"type"`
	TransactionCount int              `json:"transaction_count"`
	TotalAmount      float64          `json:"total_amount"`
	AvgAmount        float64          `json:"avg_amount"`
}

// NewSQLiteRepository creates a new SQLite transaction repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// getSingleFamilyID retrieves the ID of the single family from the database
func (r *SQLiteRepository) getSingleFamilyID(ctx context.Context) (uuid.UUID, error) {
	query := `SELECT id FROM families LIMIT 1`
	var idStr string
	err := r.db.QueryRowContext(ctx, query).Scan(&idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get family ID: %w", err)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse family ID: %w", err)
	}
	return id, nil
}

// scanTransactionRow scans a single row from SQL query into a Transaction struct
func scanTransactionRow(rows *sql.Rows) (*transaction.Transaction, error) {
	var t transaction.Transaction
	var idStr, typeStr, categoryIDStr, userIDStr, familyIDStr string // familyIDStr unused - single family model
	var tagsJSON string

	err := rows.Scan(
		&idStr,
		&t.Amount,
		&typeStr,
		&t.Description,
		&categoryIDStr,
		&userIDStr,
		&familyIDStr,
		&t.Date,
		&tagsJSON,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan transaction: %w", err)
	}

	// Parse UUID fields
	t.ID, _ = uuid.Parse(idStr)
	t.CategoryID, _ = uuid.Parse(categoryIDStr)
	t.UserID, _ = uuid.Parse(userIDStr)
	t.Type = transaction.Type(typeStr)

	// Parse tags from JSON
	if jsonErr := json.Unmarshal([]byte(tagsJSON), &t.Tags); jsonErr != nil {
		t.Tags = []string{}
	}

	return &t, nil
}

// Create creates a new transaction in the database
func (r *SQLiteRepository) Create(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}
	if err := validation.ValidateAmount(t.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := validation.ValidateDescription(t.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
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

	// Convert tags to JSON
	tagsJSON, err := json.Marshal(t.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO transactions (
			id, amount, description, date, type, category_id, user_id, family_id,
			tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(t.ID),
		t.Amount,
		t.Description,
		t.Date,
		string(t.Type),
		sqlitehelpers.UUIDToString(t.CategoryID),
		sqlitehelpers.UUIDToString(t.UserID),
		familyID.String(),
		string(tagsJSON),
		t.CreatedAt,
		t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a transaction by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, amount, description, date, type, category_id, user_id, family_id,
			   tags, created_at, updated_at
		FROM transactions
		WHERE id = ?`

	var t transaction.Transaction
	var idStr, typeStr, categoryIDStr, userIDStr, familyIDStr string // familyIDStr unused - single family model
	var tagsJSON string

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &t.Amount, &t.Description, &t.Date, &typeStr,
		&categoryIDStr, &userIDStr, &familyIDStr, &tagsJSON, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("transaction with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get transaction by id: %w", err)
	}

	// Parse UUID fields
	t.ID, _ = uuid.Parse(idStr)
	t.CategoryID, _ = uuid.Parse(categoryIDStr)
	t.UserID, _ = uuid.Parse(userIDStr)
	t.Type = transaction.Type(typeStr)

	// Parse tags from JSON
	if jsonErr := json.Unmarshal([]byte(tagsJSON), &t.Tags); jsonErr != nil {
		// If unmarshaling fails, set empty tags
		t.Tags = []string{}
	}

	return &t, nil
}

// GetByFilter retrieves transactions based on filter criteria
//
//nolint:gocognit,funlen // Complex function due to multiple optional filter conditions - necessary for comprehensive filtering
func (r *SQLiteRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	// Build dynamic query parts
	var conditions []string
	var args []any

	// Family ID is always required
	conditions = append(conditions, "family_id = ?")
	args = append(args, familyID.String())

	// Optional filters
	if filter.UserID != nil {
		if err := validation.ValidateUUID(*filter.UserID); err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		conditions = append(conditions, "user_id = ?")
		args = append(args, sqlitehelpers.UUIDToString(*filter.UserID))
	}

	if filter.CategoryID != nil {
		if err := validation.ValidateUUID(*filter.CategoryID); err != nil {
			return nil, fmt.Errorf("invalid category ID: %w", err)
		}
		conditions = append(conditions, "category_id = ?")
		args = append(args, sqlitehelpers.UUIDToString(*filter.CategoryID))
	}

	if filter.Type != nil {
		conditions = append(conditions, "type = ?")
		args = append(args, string(*filter.Type))
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, "date >= ?")
		args = append(args, *filter.DateFrom)
	}

	if filter.DateTo != nil {
		conditions = append(conditions, "date <= ?")
		args = append(args, *filter.DateTo)
	}

	if filter.AmountFrom != nil {
		conditions = append(conditions, "amount >= ?")
		args = append(args, *filter.AmountFrom)
	}

	if filter.AmountTo != nil {
		conditions = append(conditions, "amount <= ?")
		args = append(args, *filter.AmountTo)
	}

	if filter.Description != "" {
		conditions = append(conditions, "description LIKE ?")
		args = append(args, "%"+filter.Description+"%")
	}

	// Tag filtering - SQLite doesn't have JSONB operators like PostgreSQL
	// We need to check if any tag in the JSON array matches
	if len(filter.Tags) > 0 {
		tagConditions := make([]string, len(filter.Tags))
		for i, tag := range filter.Tags {
			// Check if the tag exists in the JSON array using SQLite json_each
			tagConditions[i] = "EXISTS (SELECT 1 FROM json_each(tags) WHERE value = ?)"
			args = append(args, tag)
		}
		conditions = append(conditions, "("+strings.Join(tagConditions, " OR ")+")")
	}

	// Build final query
	//nolint:gosec // SQL concatenation is safe here - conditions are built from validated inputs
	query := `
		SELECT id, amount, type, description, category_id, user_id, family_id,
			   date, tags, created_at, updated_at
		FROM transactions
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY date DESC, created_at DESC`

	// Add pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by filter: %w", err)
	}
	defer rows.Close()

	// Scan results
	var transactions []*transaction.Transaction
	for rows.Next() {
		var t *transaction.Transaction
		t, err = scanTransactionRow(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}

// Update updates an existing transaction
func (r *SQLiteRepository) Update(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}
	if err := validation.ValidateAmount(t.Amount); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := validation.ValidateDescription(t.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	// Update timestamp
	t.UpdatedAt = time.Now()

	// Convert tags to JSON
	tagsJSON, err := json.Marshal(t.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE transactions
		SET amount = ?, description = ?, date = ?, type = ?, category_id = ?,
			user_id = ?, tags = ?, updated_at = ?
		WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query,
		t.Amount,
		t.Description,
		t.Date,
		string(t.Type),
		sqlitehelpers.UUIDToString(t.CategoryID),
		sqlitehelpers.UUIDToString(t.UserID),
		string(tagsJSON),
		t.UpdatedAt,
		sqlitehelpers.UUIDToString(t.ID),
		familyID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction with id %s not found", t.ID)
	}

	return nil
}

// Delete deletes a transaction
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	query := `DELETE FROM transactions WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), familyID.String())
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction with id %s not found", id)
	}

	return nil
}

// GetSummary returns transaction summary for a family
func (r *SQLiteRepository) GetSummary(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*Summary, error) {
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
		FROM transactions
		WHERE family_id = ? AND date BETWEEN ? AND ?`

	var summary Summary
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(familyID), startDate, endDate).Scan(
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
func (r *SQLiteRepository) GetMonthlySummary(
	ctx context.Context,
	familyID uuid.UUID,
	year, month int,
) ([]*MonthlySummaryItem, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	// SQLite date extraction using substr for dates with timezones
	// substr extracts date components directly from ISO 8601 format (YYYY-MM-DD...)
	query := `
		SELECT
			c.name as category_name,
			t.type,
			COUNT(*) as transaction_count,
			SUM(t.amount) as total_amount,
			AVG(t.amount) as avg_amount
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.family_id = ?
		AND CAST(substr(t.date, 1, 4) AS INTEGER) = ?
		AND CAST(substr(t.date, 6, 2) AS INTEGER) = ?
		GROUP BY c.id, c.name, t.type
		ORDER BY total_amount DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), year, month)
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
func (r *SQLiteRepository) GetAll(
	ctx context.Context,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	// Validate pagination parameters
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > validation.MaxQueryLimit {
		limit = validation.MaxQueryLimit // Maximum limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, amount, type, description, category_id, user_id, family_id,
			   date, tags, created_at, updated_at
		FROM transactions
		WHERE family_id = ? 		ORDER BY date DESC, created_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions by family: %w", err)
	}
	defer rows.Close()

	var transactions []*transaction.Transaction
	for rows.Next() {
		var t *transaction.Transaction
		t, err = scanTransactionRow(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}

// GetTotalByCategory calculates total amount for transactions by category and type
func (r *SQLiteRepository) GetTotalByCategory(
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
		FROM transactions
		WHERE category_id = ? AND type = ?`

	var total float64
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(categoryID), transactionType).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by category: %w", err)
	}

	return total, nil
}

// GetTotalByFamilyAndDateRange calculates total amount for transactions by family and date range
func (r *SQLiteRepository) GetTotalByFamilyAndDateRange(
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

	// SQLite doesn't have deleted_at column in the current schema
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE family_id = ? AND type = ? AND date >= ? AND date <= ?`

	var total float64
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(familyID), transactionType, startDate, endDate).
		Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by family and date range: %w", err)
	}

	return total, nil
}

// GetTotalByDateRange calculates total amount for transactions by date range (single family model)
func (r *SQLiteRepository) GetTotalByDateRange(
	ctx context.Context,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return 0, err
	}
	return r.GetTotalByFamilyAndDateRange(ctx, familyID, startDate, endDate, transactionType)
}

// GetTotalByCategoryAndDateRange calculates total amount for transactions by category and date range
func (r *SQLiteRepository) GetTotalByCategoryAndDateRange(
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

	// SQLite doesn't have deleted_at column in the current schema
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE category_id = ? AND type = ? AND date >= ? AND date <= ?`

	var total float64
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(categoryID), transactionType, startDate, endDate).
		Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by category and date range: %w", err)
	}

	return total, nil
}
