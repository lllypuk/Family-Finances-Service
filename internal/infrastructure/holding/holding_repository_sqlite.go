// Package holding — SQLite-репозиторий позиций капитала.
package holding

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/names"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
)

// selectWithCurrent присоединяет последний по дате снимок не позже today (первый параметр):
// updated_at здесь ни при чём, поздняя правка старой даты текущей не становится.
const selectWithCurrent = `
	SELECT h.id, h.name, h.side, h.kind, h.is_archived, h.created_at, h.updated_at, v.date, v.value_minor,
		COALESCE(p.monthly_income_minor, 0), COALESCE(p.monthly_expense_minor, 0), p.updated_at
	FROM holdings h
	LEFT JOIN holding_plans p ON p.holding_id = h.id
	LEFT JOIN holding_values v ON v.holding_id = h.id AND v.date = (
		SELECT x.date FROM holding_values x
		WHERE x.holding_id = h.id AND x.date <= ?
		ORDER BY x.date DESC LIMIT 1
	)`

// SQLiteRepository хранит позиции единственной семьи; name_key пересчитывается при каждой записи.
type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Create пишет позицию и её план одной транзакцией; строка плана 0/0 не пишется — её отверг бы CHECK.
func (r *SQLiteRepository) Create(ctx context.Context, h *holding.Holding) error {
	now := time.Now().UTC()
	h.CreatedAt, h.UpdatedAt = now, now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO holdings (id, family_id, name, name_key, side, kind, is_archived, created_at, updated_at)
		VALUES (?, (SELECT id FROM families LIMIT 1), ?, ?, ?, ?, ?, ?, ?)`,
		sqlitehelpers.UUIDToString(h.ID), h.Name, names.Key(h.Name), string(h.Side), string(h.Kind),
		sqlitehelpers.BoolToInt(h.IsArchived), h.CreatedAt, h.UpdatedAt,
	)
	if err != nil {
		return writeError("create", err)
	}

	h.Plan.UpdatedAt = nil
	if !h.Plan.IsZero() {
		if err = writePlan(ctx, tx, h.ID, h.Plan, now); err != nil {
			return err
		}
		h.Plan.UpdatedAt = &now
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit holding: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID, today date.Date) (*holding.Holding, error) {
	h, err := scanHolding(r.db.QueryRowContext(ctx, selectWithCurrent+` WHERE h.id = ?`,
		today, sqlitehelpers.UUIDToString(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", holding.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get holding: %w", err)
	}

	return h, nil
}

// List отдаёт позиции по имени, каждую со снимком на today; архивные — только с includeArchived.
func (r *SQLiteRepository) List(
	ctx context.Context,
	includeArchived bool,
	today date.Date,
) ([]*holding.Holding, error) {
	query := selectWithCurrent
	if !includeArchived {
		query += ` WHERE h.is_archived = 0`
	}

	rows, err := r.db.QueryContext(ctx, query+` ORDER BY h.name_key, h.id`, today)
	if err != nil {
		return nil, fmt.Errorf("failed to list holdings: %w", err)
	}
	defer rows.Close()

	holdings := make([]*holding.Holding, 0)
	for rows.Next() {
		h, scanErr := scanHolding(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", scanErr)
		}
		holdings = append(holdings, h)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate holdings: %w", err)
	}

	return holdings, nil
}

// Update пишет только переданные поля; side не пишется никогда. Присланные числа плана сливаются
// с сохранёнными внутри транзакции: при MaxOpenConns=1 обращение к r.db здесь ждало бы само себя.
func (r *SQLiteRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	kind *holding.Kind,
	archived *bool,
	income, expense *money.Minor,
) error {
	var nameKey, archivedInt any
	if name != nil {
		nameKey = names.Key(*name)
	}
	if archived != nil {
		archivedInt = sqlitehelpers.BoolToInt(*archived)
	}
	now := time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		UPDATE holdings
		SET name = COALESCE(?, name), name_key = COALESCE(?, name_key), kind = COALESCE(?, kind),
		    is_archived = COALESCE(?, is_archived), updated_at = ?
		WHERE id = ?`,
		name, nameKey, kind, archivedInt, now, sqlitehelpers.UUIDToString(id),
	)
	if err != nil {
		return writeError("update", err)
	}
	if err = requireAffected(res, id); err != nil {
		return err
	}

	if income != nil || expense != nil {
		if err = mergePlan(ctx, tx, id, income, expense, now); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit holding: %w", err)
	}

	return nil
}

// mergePlan накладывает присланные числа на сохранённый план и проверяет итог.
func mergePlan(ctx context.Context, tx *sql.Tx, id uuid.UUID, income, expense *money.Minor, now time.Time) error {
	var plan holding.Plan
	err := tx.QueryRowContext(ctx,
		`SELECT monthly_income_minor, monthly_expense_minor FROM holding_plans WHERE holding_id = ?`,
		sqlitehelpers.UUIDToString(id),
	).Scan(&plan.MonthlyIncomeMinor, &plan.MonthlyExpenseMinor)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to read holding plan: %w", err)
	}

	if income != nil {
		plan.MonthlyIncomeMinor = *income
	}
	if expense != nil {
		plan.MonthlyExpenseMinor = *expense
	}
	if err = holding.CheckPlan(plan.MonthlyIncomeMinor, plan.MonthlyExpenseMinor); err != nil {
		return err
	}

	return writePlan(ctx, tx, id, plan, now)
}

// writePlan держит строку плана только ненулевой: 0/0 удаляет её.
func writePlan(ctx context.Context, tx *sql.Tx, id uuid.UUID, plan holding.Plan, now time.Time) error {
	var err error
	if plan.IsZero() {
		_, err = tx.ExecContext(ctx, `DELETE FROM holding_plans WHERE holding_id = ?`,
			sqlitehelpers.UUIDToString(id))
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO holding_plans (holding_id, monthly_income_minor, monthly_expense_minor, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(holding_id) DO UPDATE SET
				monthly_income_minor = excluded.monthly_income_minor,
				monthly_expense_minor = excluded.monthly_expense_minor,
				updated_at = excluded.updated_at`,
			sqlitehelpers.UUIDToString(id), plan.MonthlyIncomeMinor, plan.MonthlyExpenseMinor, now)
	}
	if err != nil {
		return fmt.Errorf("failed to write holding plan: %w", err)
	}

	return nil
}

// Delete уносит снимки каскадом.
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM holdings WHERE id = ?`, sqlitehelpers.UUIDToString(id))
	if err != nil {
		return writeError("delete", err)
	}

	return requireAffected(res, id)
}

// UpsertValue заменяет снимок на дату; позиции нет — ErrNotFound по внешнему ключу.
func (r *SQLiteRepository) UpsertValue(ctx context.Context, holdingID uuid.UUID, v *holding.Value) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO holding_values (holding_id, date, value_minor) VALUES (?, ?, ?)
		ON CONFLICT(holding_id, date) DO UPDATE SET
			value_minor = excluded.value_minor,
			updated_at = CURRENT_TIMESTAMP
		RETURNING updated_at`,
		sqlitehelpers.UUIDToString(holdingID), v.Date, v.ValueMinor,
	).Scan(&v.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("%w: %s", holding.ErrNotFound, holdingID)
		}
		return fmt.Errorf("failed to upsert holding value: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) DeleteValue(ctx context.Context, holdingID uuid.UUID, day date.Date) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM holding_values WHERE holding_id = ? AND date = ?`,
		sqlitehelpers.UUIDToString(holdingID), day)
	if err != nil {
		return fmt.Errorf("failed to delete holding value: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s %s", holding.ErrValueNotFound, holdingID, day)
	}

	return nil
}

// ListValues — страница истории позиции, новые сверху, и число всех её снимков.
func (r *SQLiteRepository) ListValues(
	ctx context.Context,
	holdingID uuid.UUID,
	limit, offset int,
) ([]*holding.Value, int, error) {
	id := sqlitehelpers.UUIDToString(holdingID)

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM holding_values WHERE holding_id = ?`, id).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count holding values: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT date, value_minor, updated_at FROM holding_values
		WHERE holding_id = ?
		ORDER BY date DESC
		LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list holding values: %w", err)
	}
	defer rows.Close()

	values := make([]*holding.Value, 0)
	for rows.Next() {
		var v holding.Value
		if err = rows.Scan(&v.Date, &v.ValueMinor, &v.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan holding value: %w", err)
		}
		values = append(values, &v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate holding values: %w", err)
	}

	return values, total, nil
}

// SeriesValues — последний снимок каждой позиции строго до from и все снимки в [from, to],
// по возрастанию даты; архив не фильтруется. Одним запросом, чтобы обе части видели одно состояние.
// CROSS JOIN в SQLite фиксирует порядок: снаружи позиции, снимки ищутся по PK, без него план сканирует снимки.
func (r *SQLiteRepository) SeriesValues(ctx context.Context, from, to date.Date) ([]holding.SeriesRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.id, h.side, v.date, v.value_minor
		FROM holdings h
		CROSS JOIN holding_values v ON v.holding_id = h.id AND v.date = (
			SELECT x.date FROM holding_values x
			WHERE x.holding_id = h.id AND x.date < ?
			ORDER BY x.date DESC LIMIT 1
		)
		UNION ALL
		SELECT h.id, h.side, v.date, v.value_minor
		FROM holdings h
		CROSS JOIN holding_values v ON v.holding_id = h.id AND v.date >= ? AND v.date <= ?
		ORDER BY 3, 1`, from, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to read holding series: %w", err)
	}
	defer rows.Close()

	series := make([]holding.SeriesRow, 0)
	for rows.Next() {
		var (
			row   holding.SeriesRow
			idStr string
			side  string
		)
		if err = rows.Scan(&idStr, &side, &row.Date, &row.ValueMinor); err != nil {
			return nil, fmt.Errorf("failed to scan holding series row: %w", err)
		}
		if row.HoldingID, err = uuid.Parse(idStr); err != nil {
			return nil, fmt.Errorf("invalid holding id %q: %w", idStr, err)
		}
		row.Side = holding.Side(side)
		series = append(series, row)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate holding series: %w", err)
	}

	return series, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanHolding(row rowScanner) (*holding.Holding, error) {
	var (
		h           holding.Holding
		idStr       string
		side, kind  string
		archived    int
		currentDate date.Date
		currentVal  sql.NullInt64
		planAt      sql.NullTime
	)
	if err := row.Scan(&idStr, &h.Name, &side, &kind, &archived, &h.CreatedAt, &h.UpdatedAt,
		&currentDate, &currentVal, &h.Plan.MonthlyIncomeMinor, &h.Plan.MonthlyExpenseMinor, &planAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid holding id %q: %w", idStr, err)
	}
	h.ID = id
	h.Side = holding.Side(side)
	h.Kind = holding.Kind(kind)
	h.IsArchived = sqlitehelpers.IntToBool(archived)
	if currentVal.Valid {
		h.Current = &holding.Value{Date: currentDate, ValueMinor: money.Minor(currentVal.Int64)}
	}

	if planAt.Valid {
		h.Plan.UpdatedAt = &planAt.Time
	}

	return &h, nil
}

// writeError различает нарушения по тексту драйвера: имя — только UNIQUE по name_key,
// коллизия первичного ключа остаётся обычной ошибкой.
func writeError(op string, err error) error {
	if msg := err.Error(); strings.Contains(msg, "UNIQUE constraint failed") &&
		strings.Contains(msg, "holdings.name_key") {
		return fmt.Errorf("%w: %w", holding.ErrNameExists, err)
	}

	return fmt.Errorf("failed to %s holding: %w", op, err)
}

func requireAffected(res sql.Result, id uuid.UUID) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", holding.ErrNotFound, id)
	}

	return nil
}
