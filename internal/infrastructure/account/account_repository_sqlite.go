// Package account — SQLite-репозиторий счетов.
package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/names"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
)

const selectColumns = `SELECT id, name, is_archived, created_at, updated_at FROM accounts`

// SQLiteRepository хранит счета единственной семьи; name_key пересчитывается при каждой записи.
type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, a *account.Account) error {
	now := time.Now().UTC()
	a.CreatedAt, a.UpdatedAt = now, now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO accounts (id, family_id, name, name_key, is_archived, created_at, updated_at)
		VALUES (?, (SELECT id FROM families LIMIT 1), ?, ?, ?, ?, ?)`,
		sqlitehelpers.UUIDToString(a.ID), a.Name, names.Key(a.Name), sqlitehelpers.BoolToInt(a.IsArchived),
		a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return writeError("create", err)
	}

	return nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	a, err := scanAccount(r.db.QueryRowContext(ctx, selectColumns+` WHERE id = ?`, sqlitehelpers.UUIDToString(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", account.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return a, nil
}

// List отдаёт счета по имени; архивные — только с includeArchived.
func (r *SQLiteRepository) List(ctx context.Context, includeArchived bool) ([]*account.Account, error) {
	query := selectColumns
	if !includeArchived {
		query += ` WHERE is_archived = 0`
	}

	rows, err := r.db.QueryContext(ctx, query+` ORDER BY name_key, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*account.Account, 0)
	for rows.Next() {
		a, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan account: %w", scanErr)
		}
		accounts = append(accounts, a)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate accounts: %w", err)
	}

	return accounts, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, a *account.Account) error {
	a.UpdatedAt = time.Now().UTC()

	res, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET name = ?, name_key = ?, is_archived = ?, updated_at = ? WHERE id = ?`,
		a.Name, names.Key(a.Name), sqlitehelpers.BoolToInt(a.IsArchived), a.UpdatedAt,
		sqlitehelpers.UUIDToString(a.ID),
	)
	if err != nil {
		return writeError("update", err)
	}

	return requireAffected(res, a.ID)
}

func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, sqlitehelpers.UUIDToString(id))
	if err != nil {
		return writeError("delete", err)
	}

	return requireAffected(res, id)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row rowScanner) (*account.Account, error) {
	var (
		a        account.Account
		idStr    string
		archived int
	)
	if err := row.Scan(&idStr, &a.Name, &archived, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid account id %q: %w", idStr, err)
	}
	a.ID = id
	a.IsArchived = sqlitehelpers.IntToBool(archived)

	return &a, nil
}

// writeError различает нарушения по тексту драйвера: имя — только UNIQUE по name_key,
// коллизия первичного ключа остаётся обычной ошибкой.
func writeError(op string, err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed") && strings.Contains(msg, "accounts.name_key"):
		return fmt.Errorf("%w: %w", account.ErrNameExists, err)
	case strings.Contains(msg, "FOREIGN KEY constraint failed"):
		return fmt.Errorf("%w: %w", account.ErrInUse, err)
	default:
		return fmt.Errorf("failed to %s account: %w", op, err)
	}
}

func requireAffected(res sql.Result, id uuid.UUID) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", account.ErrNotFound, id)
	}

	return nil
}
