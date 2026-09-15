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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// budgetColumns — порядок колонок, в котором их ждёт scanBudget.
const budgetColumns = `id, name, amount_minor, spent_minor, period, start_date, end_date,
	category_id, family_id, is_active, recurring, series_id, created_at, updated_at`

// maxAdvancePeriods ограничивает один вызов Advance: серия, отставшая сильнее, — признак
// ошибки в датах, а не пропущенных месяцев.
const maxAdvancePeriods = 120

// SQLiteRepository implements budget repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// UsageStats holds budget usage statistics
type UsageStats struct {
	BudgetID             uuid.UUID     `json:"budget_id"`
	BudgetName           string        `json:"budget_name"`
	BudgetAmountMinor    money.Minor   `json:"budget_amount_minor"`
	SpentAmountMinor     money.Minor   `json:"spent_amount_minor"`
	RemainingAmountMinor money.Minor   `json:"remaining_amount_minor"`
	UsagePercentage      float64       `json:"usage_percentage"`
	Period               budget.Period `json:"period"`
	StartDate            date.Date     `json:"start_date"`
	EndDate              date.Date     `json:"end_date"`
	DaysRemaining        int           `json:"days_remaining"`
	Status               string        `json:"status"` // 'safe', 'on_track', 'warning', 'over_budget'
	CategoryName         string        `json:"category_name,omitempty"`
}

// NewSQLiteRepository creates a new SQLite budget repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// rowScanner — общий знаменатель *sql.Row и *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// rowQuerier — то, что умеют и *sql.DB, и *sql.Tx: проверки внутри транзакции обязаны
// идти через tx, иначе между проверкой и записью влезает чужой запрос.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
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

// withTx выполняет fn в транзакции. _txlock=immediate берёт write-lock уже на BeginTx,
// поэтому проверка и запись внутри fn не разъезжаются с чужим запросом.
func (r *SQLiteRepository) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// scanBudget scans a single row from SQL query into a Budget struct
func scanBudget(row rowScanner) (*budget.Budget, error) {
	var b budget.Budget
	var idStr, periodStr, familyIDStr string
	var categoryIDStr, seriesIDStr *string
	var isActiveInt, recurringInt int

	err := row.Scan(
		&idStr, &b.Name, &b.AmountMinor, &b.SpentMinor, &periodStr,
		&b.StartDate, &b.EndDate, &categoryIDStr, &familyIDStr,
		&isActiveInt, &recurringInt, &seriesIDStr, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan budget: %w", err)
	}

	// Parse UUID fields
	b.ID, _ = uuid.Parse(idStr)
	// familyIDStr unused - single family model
	if categoryIDStr != nil && *categoryIDStr != "" {
		categoryID, _ := uuid.Parse(*categoryIDStr)
		b.CategoryID = &categoryID
	}
	if seriesIDStr != nil && *seriesIDStr != "" {
		seriesID, _ := uuid.Parse(*seriesIDStr)
		b.SeriesID = &seriesID
	}

	b.Period = budget.Period(periodStr)
	b.IsActive = sqlitehelpers.IntToBool(isActiveInt)
	b.Recurring = sqlitehelpers.IntToBool(recurringInt)

	return &b, nil
}

func collectBudgets(rows *sql.Rows) ([]*budget.Budget, error) {
	defer rows.Close()

	var budgets []*budget.Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return budgets, nil
}

// takenBy — чем занят период: имя мешающего бюджета и совпала ли с ним область.
type takenBy struct {
	name      string
	sameScope bool
}

// conflict — отказ, который увидит клиент. Область и имя требуют разного действия:
// пересечение области лечится сдвигом дат, занятое имя — только переименованием.
func (t *takenBy) conflict() error {
	if t.sameScope {
		return &budget.OverlapError{Name: t.name}
	}

	return fmt.Errorf("%w: %s", budget.ErrNameExists, t.name)
}

// taken — занят ли период бюджета b: живой бюджет той же области с пересечением дат
// включительно или живой бюджет с тем же именем на пересекающихся датах. Вторая ветка
// нужна потому, что idx_budgets_name_period_active не знает о категориях.
func taken(
	ctx context.Context,
	q rowQuerier,
	familyID uuid.UUID,
	b *budget.Budget,
	excludeID uuid.UUID,
) (*takenBy, error) {
	const query = `
		SELECT name, category_id IS ? FROM budgets
		WHERE family_id = ? AND is_active = 1 AND id <> ?
		AND start_date <= ? AND end_date >= ?
		AND (category_id IS ? OR name = ?)
		LIMIT 1`

	scope := sqlitehelpers.UUIDPtrToString(b.CategoryID)
	var name string
	var sameScope int
	err := q.QueryRowContext(ctx, query,
		scope,
		sqlitehelpers.UUIDToString(familyID),
		sqlitehelpers.UUIDToString(excludeID),
		b.EndDate,
		b.StartDate,
		scope,
		b.Name,
	).Scan(&name, &sameScope)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // свободный период — не ошибка и не конфликт
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check budget period: %w", err)
	}

	return &takenBy{name: name, sameScope: sameScope == 1}, nil
}

// checkTailAvailable — включить recurring можно только последнему живому инстансу серии и
// только если хвоста у неё ещё нет.
func checkTailAvailable(ctx context.Context, q rowQuerier, familyID uuid.UUID, b *budget.Budget) error {
	seriesID := b.ID
	if b.SeriesID != nil {
		seriesID = *b.SeriesID
	}

	maxStart, tails, err := siblingTails(ctx, q, familyID, seriesID, b.ID)
	if err != nil {
		return err
	}

	if tails > 0 || maxStart > b.StartDate.String() {
		return budget.ErrNotTail
	}

	return nil
}

// checkSeriesStopped — снять флаг с не-хвоста нечего, но живой хвост у серии означает, что
// клиент правит устаревшую форму: серию возобновили после того, как он её прочитал.
func checkSeriesStopped(ctx context.Context, q rowQuerier, familyID uuid.UUID, current *budget.Budget) error {
	if current.SeriesID == nil {
		return nil
	}

	_, tails, err := siblingTails(ctx, q, familyID, *current.SeriesID, current.ID)
	if err != nil {
		return err
	}

	if tails > 0 {
		return budget.ErrNotTail
	}

	return nil
}

// siblingTails — максимальная start_date и число живых хвостов среди прочих инстансов серии.
func siblingTails(
	ctx context.Context,
	q rowQuerier,
	familyID, seriesID, excludeID uuid.UUID,
) (string, int, error) {
	const query = `
		SELECT COALESCE(MAX(start_date), ''), COALESCE(SUM(recurring), 0)
		FROM budgets
		WHERE family_id = ? AND series_id = ? AND is_active = 1 AND id <> ?`

	var maxStart string
	var tails int
	err := q.QueryRowContext(ctx, query,
		sqlitehelpers.UUIDToString(familyID),
		sqlitehelpers.UUIDToString(seriesID),
		sqlitehelpers.UUIDToString(excludeID),
	).Scan(&maxStart, &tails)
	if err != nil {
		return "", 0, fmt.Errorf("failed to check recurring series: %w", err)
	}

	return maxStart, tails, nil
}

func insertBudget(ctx context.Context, ex sqlExecutor, familyID uuid.UUID, b *budget.Budget) error {
	const query = `
		INSERT INTO budgets (
			id, name, amount_minor, spent_minor, period, start_date, end_date,
			category_id, family_id, is_active, recurring, series_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := ex.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(b.ID),
		b.Name,
		b.AmountMinor,
		b.SpentMinor,
		string(b.Period),
		b.StartDate,
		b.EndDate,
		sqlitehelpers.UUIDPtrToString(b.CategoryID),
		familyID.String(),
		sqlitehelpers.BoolToInt(b.IsActive),
		sqlitehelpers.BoolToInt(b.Recurring),
		sqlitehelpers.UUIDPtrToString(b.SeriesID),
		b.CreatedAt,
		b.UpdatedAt,
	)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			if strings.Contains(err.Error(), "budgets.id") {
				return fmt.Errorf("%w: %s", budget.ErrIDExists, b.ID)
			}
			return fmt.Errorf("%w: %s", budget.ErrNameExists, b.Name)
		}
		return fmt.Errorf("failed to create budget: %w", err)
	}

	return nil
}

// validateForWrite проверяет поля бюджета и возвращает id единственной семьи.
func (r *SQLiteRepository) validateForWrite(ctx context.Context, b *budget.Budget) (uuid.UUID, error) {
	if err := validation.ValidateUUID(b.ID); err != nil {
		return uuid.Nil, fmt.Errorf("invalid budget ID: %w", err)
	}

	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get family ID: %w", err)
	}
	if validationErr := validation.ValidateBudgetPeriod(b.Period); validationErr != nil {
		return uuid.Nil, fmt.Errorf("invalid period: %w", validationErr)
	}
	if validationErr := validation.ValidateBudgetAmount(b.AmountMinor); validationErr != nil {
		return uuid.Nil, fmt.Errorf("invalid amount: %w", validationErr)
	}
	if validationErr := validation.ValidateBudgetName(b.Name); validationErr != nil {
		return uuid.Nil, fmt.Errorf("invalid name: %w", validationErr)
	}

	if b.CategoryID != nil {
		if validationErr := validation.ValidateUUID(*b.CategoryID); validationErr != nil {
			return uuid.Nil, fmt.Errorf("invalid category ID: %w", validationErr)
		}
	}

	return familyID, nil
}

// Create creates a new budget in the database
func (r *SQLiteRepository) Create(ctx context.Context, b *budget.Budget) error {
	familyID, err := r.validateForWrite(ctx, b)
	if err != nil {
		return err
	}

	// Validate date range
	if !b.EndDate.After(b.StartDate) {
		return errors.New("end date must be after start date")
	}

	// Set timestamps
	now := time.Now()
	b.CreatedAt = now
	b.UpdatedAt = now

	return r.withTx(ctx, func(tx *sql.Tx) error {
		busy, takenErr := taken(ctx, tx, familyID, b, b.ID)
		if takenErr != nil {
			return takenErr
		}
		if busy != nil {
			return busy.conflict()
		}

		return insertBudget(ctx, tx, familyID, b)
	})
}

// GetByID retrieves a budget by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE id = ? AND is_active = 1`

	b, err := scanBudget(r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get budget by id: %w", err)
	}

	return b, nil
}

// GetAll retrieves all budgets belonging to the family
func (r *SQLiteRepository) GetAll(ctx context.Context) ([]*budget.Budget, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		ORDER BY start_date DESC, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by family id: %w", err)
	}

	return collectBudgets(rows)
}

// GetActiveBudgets retrieves budgets whose period covers the given calendar date.
func (r *SQLiteRepository) GetActiveBudgets(ctx context.Context, on date.Date) ([]*budget.Budget, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE family_id = ? AND is_active = 1
		AND start_date <= ? AND end_date >= ?
		ORDER BY start_date DESC, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), on, on)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	return collectBudgets(rows)
}

// ListRecurring возвращает хвосты всех серий семьи — по одному живому бюджету на серию.
func (r *SQLiteRepository) ListRecurring(ctx context.Context) ([]*budget.Budget, error) {
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE family_id = ? AND is_active = 1 AND recurring = 1
		ORDER BY start_date`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return nil, fmt.Errorf("failed to list recurring budgets: %w", err)
	}

	return collectBudgets(rows)
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

	today := date.Today(time.UTC)
	// Calculate spent dynamically using CTE to avoid code duplication
	query := `
		WITH budget_spent AS (
			SELECT
				b.id,
				COALESCE(SUM(t.amount_minor), 0) as spent
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
			b.amount_minor,
			bs.spent,
			b.period,
			b.start_date,
			b.end_date,
			(b.amount_minor - bs.spent) as remaining_amount,
			CASE WHEN b.amount_minor > 0 THEN ROUND((CAST(bs.spent AS REAL) / b.amount_minor * 100), 2) ELSE 0 END
				as usage_percentage,
			CAST((julianday(b.end_date) - julianday('now')) AS INTEGER) as days_remaining,
			CASE
				WHEN bs.spent > b.amount_minor THEN 'over_budget'
				WHEN bs.spent > (b.amount_minor * 0.8) THEN 'warning'
				WHEN bs.spent > (b.amount_minor * 0.5) THEN 'on_track'
				ELSE 'safe'
			END as status,
			c.name as category_name
		FROM budgets b
		JOIN budget_spent bs ON b.id = bs.id
		LEFT JOIN categories c ON b.category_id = c.id
		ORDER BY usage_percentage DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), today)
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
			&budgetIDStr, &stat.BudgetName, &stat.BudgetAmountMinor, &stat.SpentAmountMinor,
			&periodStr, &stat.StartDate, &stat.EndDate, &stat.RemainingAmountMinor,
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

// Update пишет колонки бюджета, а spent_minor пересчитывает из транзакций — присланное
// значение игнорируется, иначе сдвиг периода записал бы сумму от старых границ.
// expect — то, что сервис прочитал о серии; несовпадение означает, что серия продвинулась
// между чтением и записью, и даёт ErrNotTail.
func (r *SQLiteRepository) Update(ctx context.Context, b *budget.Budget, expect budget.UpdateExpect) error {
	familyID, err := r.validateForWrite(ctx, b)
	if err != nil {
		return err
	}

	b.UpdatedAt = time.Now()

	return r.withTx(ctx, func(tx *sql.Tx) error {
		current, curErr := budgetForUpdate(ctx, tx, familyID, b.ID)
		if curErr != nil {
			return curErr
		}
		if checkErr := checkUpdateAllowed(ctx, tx, familyID, b, current, expect); checkErr != nil {
			return checkErr
		}

		if updErr := updateBudgetRow(ctx, tx, familyID, b, expect.Recurring); updErr != nil {
			return updErr
		}

		// Расход пересчитывается в этой же транзакции: между записью дат и отдельным
		// UpdateSpent успел бы вклиниться расход, и его сумму затёрло бы старым итогом.
		return syncSpentInto(ctx, tx, b)
	})
}

// checkUpdateAllowed — правила серии и занятости периода, все на прочитанной в той же
// транзакции строке current.
func checkUpdateAllowed(
	ctx context.Context,
	tx *sql.Tx,
	familyID uuid.UUID,
	b, current *budget.Budget,
	expect budget.UpdateExpect,
) error {
	if current.Recurring != expect.Recurring {
		return budget.ErrNotTail
	}

	if b.Recurring && !current.Recurring {
		if err := checkTailAvailable(ctx, tx, familyID, b); err != nil {
			return err
		}
	}

	if expect.StopSeries && !current.Recurring {
		if err := checkSeriesStopped(ctx, tx, familyID, current); err != nil {
			return err
		}
	}

	if b.StartDate != current.StartDate || b.EndDate != current.EndDate ||
		b.Name != current.Name || !sameScope(b.CategoryID, current.CategoryID) {
		busy, err := taken(ctx, tx, familyID, b, b.ID)
		if err != nil {
			return err
		}
		if busy != nil {
			return busy.conflict()
		}
	}

	return nil
}

func budgetForUpdate(
	ctx context.Context,
	q rowQuerier,
	familyID, id uuid.UUID,
) (*budget.Budget, error) {
	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE id = ? AND family_id = ? AND is_active = 1`

	b, err := scanBudget(q.QueryRowContext(ctx, query,
		sqlitehelpers.UUIDToString(id), sqlitehelpers.UUIDToString(familyID)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to read budget for update: %w", err)
	}

	return b, nil
}

func sameScope(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

func updateBudgetRow(
	ctx context.Context,
	ex sqlExecutor,
	familyID uuid.UUID,
	b *budget.Budget,
	expectRecurring bool,
) error {
	const query = `
		UPDATE budgets
		SET name = ?, amount_minor = ?, period = ?, start_date = ?, end_date = ?,
			category_id = ?, is_active = ?, recurring = ?, series_id = ?, updated_at = ?
		WHERE id = ? AND family_id = ? AND is_active = 1 AND recurring = ?`

	result, err := ex.ExecContext(ctx, query,
		b.Name,
		b.AmountMinor,
		string(b.Period),
		b.StartDate,
		b.EndDate,
		sqlitehelpers.UUIDPtrToString(b.CategoryID),
		sqlitehelpers.BoolToInt(b.IsActive),
		sqlitehelpers.BoolToInt(b.Recurring),
		sqlitehelpers.UUIDPtrToString(b.SeriesID),
		b.UpdatedAt,
		sqlitehelpers.UUIDToString(b.ID),
		familyID.String(),
		sqlitehelpers.BoolToInt(expectRecurring),
	)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("%w: %s", budget.ErrNameExists, b.Name)
		}
		return fmt.Errorf("failed to update budget: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return budget.ErrNotTail
	}

	return nil
}

// Advance достраивает серию от хвоста tailID до today целиком в одной транзакции: занятый
// период пропускается, чтобы одна ручная правка не убивала серию. Возвращает число
// созданных инстансов; ErrNotTail — хвост уже продвинуло параллельное чтение.
func (r *SQLiteRepository) Advance(ctx context.Context, tailID uuid.UUID, today date.Date) (int, error) {
	if err := validation.ValidateUUID(tailID); err != nil {
		return 0, fmt.Errorf("invalid budget ID: %w", err)
	}

	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get family ID: %w", err)
	}

	created := 0
	err = r.withTx(ctx, func(tx *sql.Tx) error {
		var txErr error
		created, txErr = advanceSeries(ctx, tx, familyID, tailID, today)
		return txErr
	})
	if err != nil {
		return 0, err
	}

	return created, nil
}

func advanceSeries(
	ctx context.Context,
	tx *sql.Tx,
	familyID, tailID uuid.UUID,
	today date.Date,
) (int, error) {
	query := `SELECT ` + budgetColumns + `
		FROM budgets
		WHERE id = ? AND family_id = ? AND is_active = 1 AND recurring = 1`

	tail, err := scanBudget(tx.QueryRowContext(ctx, query,
		sqlitehelpers.UUIDToString(tailID), sqlitehelpers.UUIDToString(familyID)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, budget.ErrNotTail
		}
		return 0, fmt.Errorf("failed to read series tail: %w", err)
	}

	seriesID := tail.ID
	if tail.SeriesID != nil {
		seriesID = *tail.SeriesID
	}

	created, steps := 0, 0
	for cand := tail.Next(); cand != nil && !cand.StartDate.After(today); cand = cand.Next() {
		steps++
		if steps > maxAdvancePeriods {
			return 0, budget.ErrTooFarBehind
		}

		cand.SeriesID = &seriesID
		busy, takenErr := taken(ctx, tx, familyID, cand, cand.ID)
		if takenErr != nil {
			return 0, takenErr
		}
		if busy != nil {
			continue
		}

		if err = materializeNext(ctx, tx, familyID, tail, cand); err != nil {
			return 0, err
		}
		created++
		tail = cand
	}

	return created, nil
}

// spentFromTransactions — расход по периоду и области бюджета; коррелирует с внешним budgets.
const spentFromTransactions = `(
	SELECT COALESCE(SUM(t.amount_minor), 0)
	FROM transactions t
	WHERE t.type = 'expense'
	AND t.date BETWEEN budgets.start_date AND budgets.end_date
	AND (budgets.category_id IS NULL OR t.category_id = budgets.category_id)
	AND t.family_id = budgets.family_id
)`

// syncSpent приводит spent_minor строки к сумме транзакций её периода; updated_at не трогает.
func syncSpent(ctx context.Context, ex sqlExecutor, budgetID uuid.UUID) error {
	const query = `UPDATE budgets SET spent_minor = ` + spentFromTransactions + ` WHERE id = ?`

	if _, err := ex.ExecContext(ctx, query, sqlitehelpers.UUIDToString(budgetID)); err != nil {
		return fmt.Errorf("failed to sync spent amount: %w", err)
	}

	return nil
}

// syncSpentInto пересчитывает spent_minor строки и возвращает сумму в b: ответ обязан
// показать то же значение, которое прочитает проверка лимита следующей транзакции.
func syncSpentInto(ctx context.Context, tx *sql.Tx, b *budget.Budget) error {
	if err := syncSpent(ctx, tx, b.ID); err != nil {
		return err
	}

	err := tx.QueryRowContext(ctx,
		`SELECT spent_minor FROM budgets WHERE id = ?`,
		sqlitehelpers.UUIDToString(b.ID)).Scan(&b.SpentMinor)
	if err != nil {
		return fmt.Errorf("failed to read synced spent amount: %w", err)
	}

	return nil
}

// materializeNext вставляет следующий инстанс серии и снимает флаг с прежнего хвоста:
// recurring должен остаться ровно у одной строки серии.
func materializeNext(
	ctx context.Context,
	ex sqlExecutor,
	familyID uuid.UUID,
	tail, next *budget.Budget,
) error {
	now := time.Now()
	next.CreatedAt, next.UpdatedAt = now, now

	if err := insertBudget(ctx, ex, familyID, next); err != nil {
		return err
	}

	// Инстанс может материализоваться задним числом, когда расход его периода уже записан:
	// лимит транзакции читает spent_minor из БД без пересчёта.
	if err := syncSpent(ctx, ex, next.ID); err != nil {
		return err
	}

	_, err := ex.ExecContext(ctx,
		`UPDATE budgets SET recurring = 0, updated_at = ? WHERE id = ?`,
		now, sqlitehelpers.UUIDToString(tail.ID))
	if err != nil {
		return fmt.Errorf("failed to clear recurring flag: %w", err)
	}

	return nil
}

// UpdateSpent пишет только spent_minor: пересчёт расхода вне записи бюджета идёт через него.
func (r *SQLiteRepository) UpdateSpent(
	ctx context.Context,
	budgetID uuid.UUID,
	spentAmountMinor money.Minor,
) error {
	// Validate parameters
	if err := validation.ValidateUUID(budgetID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if spentAmountMinor < 0 {
		return errors.New("spent amount cannot be negative")
	}

	query := `
		UPDATE budgets
		SET spent_minor = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, spentAmountMinor, sqlitehelpers.UUIDToString(budgetID))
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
		SET spent_minor = ` + spentFromTransactions + `,
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
	transactionDate date.Date,
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

	query := `
		UPDATE budgets
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND family_id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), familyID.String())
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

// GetByCategory retrieves budgets optionally filtered by category ID
func (r *SQLiteRepository) GetByCategory(
	ctx context.Context,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	var query string
	var args []any

	if categoryID != nil {
		if validationErr := validation.ValidateUUID(*categoryID); validationErr != nil {
			return nil, fmt.Errorf("invalid category ID: %w", validationErr)
		}

		query = `SELECT ` + budgetColumns + `
			FROM budgets
			WHERE family_id = ? AND category_id = ? AND is_active = 1
			ORDER BY created_at DESC`
		args = []any{sqlitehelpers.UUIDToString(familyID), sqlitehelpers.UUIDToString(*categoryID)}
	} else {
		query = `SELECT ` + budgetColumns + `
			FROM budgets
			WHERE family_id = ? AND is_active = 1
			ORDER BY created_at DESC`
		args = []any{sqlitehelpers.UUIDToString(familyID)}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query budgets: %w", err)
	}

	return collectBudgets(rows)
}
