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
	SELECT h.id, h.name, h.side, h.kind, h.is_archived, h.created_at, h.updated_at, v.date, v.value_minor
	FROM holdings h
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

func (r *SQLiteRepository) Create(ctx context.Context, h *holding.Holding) error {
	now := time.Now().UTC()
	h.CreatedAt, h.UpdatedAt = now, now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO holdings (id, family_id, name, name_key, side, kind, is_archived, created_at, updated_at)
		VALUES (?, (SELECT id FROM families LIMIT 1), ?, ?, ?, ?, ?, ?, ?)`,
		sqlitehelpers.UUIDToString(h.ID), h.Name, names.Key(h.Name), string(h.Side), string(h.Kind),
		sqlitehelpers.BoolToInt(h.IsArchived), h.CreatedAt, h.UpdatedAt,
	)
	if err != nil {
		return writeError("create", err)
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

// Update пишет только переданные поля одним UPDATE; side не пишется никогда.
func (r *SQLiteRepository) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	kind *holding.Kind,
	archived *bool,
) error {
	var nameKey, archivedInt any
	if name != nil {
		nameKey = names.Key(*name)
	}
	if archived != nil {
		archivedInt = sqlitehelpers.BoolToInt(*archived)
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE holdings
		SET name = COALESCE(?, name), name_key = COALESCE(?, name_key), kind = COALESCE(?, kind),
		    is_archived = COALESCE(?, is_archived), updated_at = ?
		WHERE id = ?`,
		name, nameKey, kind, archivedInt, time.Now().UTC(), sqlitehelpers.UUIDToString(id),
	)
	if err != nil {
		return writeError("update", err)
	}

	return requireAffected(res, id)
}

// Delete уносит снимки каскадом.
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM holdings WHERE id = ?`, sqlitehelpers.UUIDToString(id))
	if err != nil {
		return writeError("delete", err)
	}

	return requireAffected(res, id)
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
	)
	if err := row.Scan(&idStr, &h.Name, &side, &kind, &archived, &h.CreatedAt, &h.UpdatedAt,
		&currentDate, &currentVal); err != nil {
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
