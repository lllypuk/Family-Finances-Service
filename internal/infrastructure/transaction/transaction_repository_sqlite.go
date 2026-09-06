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

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements transaction repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// minTransactionYear — нижняя граница даты операции; всё раньше почти наверняка опечатка.
const minTransactionYear = 1900

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

func (r *SQLiteRepository) getSingleFamilyIDWithTx(ctx context.Context, tx *sql.Tx) (uuid.UUID, error) {
	query := `SELECT id FROM families LIMIT 1`
	var idStr string
	err := tx.QueryRowContext(ctx, query).Scan(&idStr)
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
		&t.AmountMinor,
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
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	return r.createWithFamilyID(ctx, r.db, familyID, t)
}

// CreateWithBudgetUpdate creates a transaction and updates the matching active budget in one DB transaction.
// If no active budget exists for the category, only the transaction is created.
func (r *SQLiteRepository) CreateWithBudgetUpdate(ctx context.Context, t *transaction.Transaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	rollbackNeeded := true
	defer func() {
		if rollbackNeeded {
			_ = tx.Rollback()
		}
	}()

	familyID, err := r.getSingleFamilyIDWithTx(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	if err = r.createWithFamilyID(ctx, tx, familyID, t); err != nil {
		return err
	}

	if t.Type == transaction.TypeExpense {
		err = r.updateMatchingActiveBudgetSpentTx(ctx, tx, familyID, t.CategoryID, t.AmountMinor, t.Date)
		if err != nil {
			return fmt.Errorf("failed to update budget after transaction create: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	rollbackNeeded = false

	return nil
}

func (r *SQLiteRepository) createWithFamilyID(
	ctx context.Context,
	execer sqlExecer,
	familyID uuid.UUID,
	t *transaction.Transaction,
) error {
	if err := validateTransactionForCreate(t); err != nil {
		return err
	}

	if err := prepareTransactionForCreate(t); err != nil {
		return err
	}

	t.Tags = normalizeTags(t.Tags)

	tagsJSON, err := json.Marshal(t.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO transactions (
			id, amount_minor, description, date, type, category_id, user_id, family_id,
			tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = execer.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(t.ID),
		int64(t.AmountMinor),
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

func validateTransactionForCreate(t *transaction.Transaction) error {
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
	if err := validation.ValidateAmount(t.AmountMinor); err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if err := validation.ValidateDescription(t.Description); err != nil {
		return fmt.Errorf("invalid description: %w", err)
	}

	return nil
}

// normalizeTags превращает nil в пустой список: колонка и JSON-контракт держат массив, не null.
func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}

	return tags
}

func prepareTransactionForCreate(t *transaction.Transaction) error {
	if t.Date.After(date.FromTime(time.Now().AddDate(1, 0, 0))) {
		return errors.New("transaction date cannot be more than 1 year in the future")
	}
	if t.Date.Before(date.New(minTransactionYear, time.January, 1)) {
		return errors.New("transaction date too old")
	}

	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	return nil
}

func (r *SQLiteRepository) updateMatchingActiveBudgetSpentTx(
	ctx context.Context,
	tx *sql.Tx,
	familyID uuid.UUID,
	categoryID uuid.UUID,
	amountMinor money.Minor,
	on date.Date,
) error {
	now := time.Now()

	// Бюджет выбирается по дате операции, а не по «сегодня»: зона процесса не обязана
	// совпадать с зоной семьи, а операция может быть задним числом.
	var budgetID string
	selectQuery := `
		SELECT id
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		  AND start_date <= ? AND end_date >= ?
		  AND category_id = ?
		ORDER BY start_date DESC, name
		LIMIT 1`

	err := tx.QueryRowContext(ctx, selectQuery, familyID.String(), on, on, sqlitehelpers.UUIDToString(categoryID)).
		Scan(&budgetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to find active budget for category: %w", err)
	}

	updateQuery := `
		UPDATE budgets
		SET spent_minor = spent_minor + ?, updated_at = ?
		WHERE id = ? AND family_id = ?`

	result, err := tx.ExecContext(ctx, updateQuery, int64(amountMinor), now, budgetID, familyID.String())
	if err != nil {
		return fmt.Errorf("failed to update budget spent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for budget update: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New("budget update affected no rows")
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
		SELECT id, amount_minor, description, date, type, category_id, user_id, family_id,
			   tags, created_at, updated_at
		FROM transactions
		WHERE id = ?`

	var t transaction.Transaction
	var idStr, typeStr, categoryIDStr, userIDStr, familyIDStr string // familyIDStr unused - single family model
	var tagsJSON string

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &t.AmountMinor, &t.Description, &t.Date, &typeStr,
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

// buildFilterConditions собирает WHERE-условия и аргументы для фильтра транзакций.
// Пагинация (LIMIT/OFFSET) намеренно не добавляется: она нужна только выборке строк,
// но не подсчёту общего количества (CountByFilter).
func (r *SQLiteRepository) buildFilterConditions(
	ctx context.Context,
	filter transaction.Filter,
) ([]string, []any, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	// Build dynamic query parts
	var conditions []string
	var args []any

	// Family ID is always required
	conditions = append(conditions, "family_id = ?")
	args = append(args, familyID.String())

	// Optional filters
	if filter.UserID != nil {
		if validationErr := validation.ValidateUUID(*filter.UserID); validationErr != nil {
			return nil, nil, fmt.Errorf("invalid user ID: %w", validationErr)
		}
		conditions = append(conditions, "user_id = ?")
		args = append(args, sqlitehelpers.UUIDToString(*filter.UserID))
	}

	if filter.CategoryID != nil {
		if validationErr := validation.ValidateUUID(*filter.CategoryID); validationErr != nil {
			return nil, nil, fmt.Errorf("invalid category ID: %w", validationErr)
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

	if filter.AmountFromMinor != nil {
		conditions = append(conditions, "amount_minor >= ?")
		args = append(args, int64(*filter.AmountFromMinor))
	}

	if filter.AmountToMinor != nil {
		conditions = append(conditions, "amount_minor <= ?")
		args = append(args, int64(*filter.AmountToMinor))
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

	return conditions, args, nil
}

// CountByFilter возвращает общее количество транзакций, подходящих под фильтр,
// без учёта LIMIT/OFFSET. Нужен пагинации: без реального итога UI не может
// узнать, есть ли следующая страница.
func (r *SQLiteRepository) CountByFilter(ctx context.Context, filter transaction.Filter) (int, error) {
	conditions, args, err := r.buildFilterConditions(ctx, filter)
	if err != nil {
		return 0, err
	}

	query := `SELECT COUNT(*) FROM transactions WHERE ` + strings.Join(conditions, " AND ")

	var total int
	if scanErr := r.db.QueryRowContext(ctx, query, args...).Scan(&total); scanErr != nil {
		return 0, fmt.Errorf("failed to count transactions by filter: %w", scanErr)
	}

	return total, nil
}

// GetByFilter retrieves transactions based on filter criteria
func (r *SQLiteRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	conditions, args, err := r.buildFilterConditions(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Build final query
	//nolint:gosec // SQL concatenation is safe here - conditions are built from validated inputs
	query := `
		SELECT id, amount_minor, type, description, category_id, user_id, family_id,
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
	if err := validation.ValidateAmount(t.AmountMinor); err != nil {
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

	t.Tags = normalizeTags(t.Tags)

	tagsJSON, err := json.Marshal(t.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE transactions
		SET amount_minor = ?, description = ?, date = ?, type = ?, category_id = ?,
			user_id = ?, tags = ?, updated_at = ?
		WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query,
		int64(t.AmountMinor),
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

// DeleteBulk удаляет транзакции семьи одним запросом; отсутствующие id просто не попадают
// в rows affected, поэтому повтор того же вызова вернёт 0.
func (r *SQLiteRepository) DeleteBulk(ctx context.Context, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get family ID: %w", err)
	}

	args := make([]any, 0, len(ids)+1)
	args = append(args, familyID.String())
	for _, id := range ids {
		if validateErr := validation.ValidateUUID(id); validateErr != nil {
			return 0, fmt.Errorf("invalid id parameter: %w", validateErr)
		}
		args = append(args, sqlitehelpers.UUIDToString(id))
	}

	//nolint:gosec // G202: в запрос подставляются только плейсхолдеры "?", значения идут через args
	query := `DELETE FROM transactions WHERE family_id = ? AND id IN (` +
		strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",") + `)`

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete transactions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetAll retrieves transactions for the family with pagination
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
		SELECT id, amount_minor, type, description, category_id, user_id, family_id,
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
) (money.Minor, error) {
	// Validate parameters
	if err := validation.ValidateUUID(categoryID); err != nil {
		return 0, fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	query := `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM transactions
		WHERE category_id = ? AND type = ?`

	var total money.Minor
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
	startDate, endDate date.Date,
	transactionType transaction.Type,
) (money.Minor, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return 0, fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	// SQLite doesn't have deleted_at column in the current schema
	query := `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM transactions
		WHERE family_id = ? AND type = ? AND date >= ? AND date <= ?`

	var total money.Minor
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
	startDate, endDate date.Date,
	transactionType transaction.Type,
) (money.Minor, error) {
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
	startDate, endDate date.Date,
	transactionType transaction.Type,
) (money.Minor, error) {
	// Validate parameters
	if err := validation.ValidateUUID(categoryID); err != nil {
		return 0, fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateTransactionType(transactionType); err != nil {
		return 0, fmt.Errorf("invalid transaction type: %w", err)
	}

	// SQLite doesn't have deleted_at column in the current schema
	query := `
		SELECT COALESCE(SUM(amount_minor), 0)
		FROM transactions
		WHERE category_id = ? AND type = ? AND date >= ? AND date <= ?`

	var total money.Minor
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(categoryID), transactionType, startDate, endDate).
		Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total by category and date range: %w", err)
	}

	return total, nil
}
