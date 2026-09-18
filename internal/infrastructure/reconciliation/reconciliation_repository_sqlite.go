// Package reconciliation — SQLite-репозиторий сверок счетов.
package reconciliation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Upsert заменяет сверку целиком, заметку тоже.
func (r *SQLiteRepository) Upsert(ctx context.Context, rec *reconciliation.Reconciliation) error {
	rec.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_reconciliations (account_id, month, bank_expense_minor, note, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(account_id, month) DO UPDATE SET
			bank_expense_minor = excluded.bank_expense_minor,
			note = excluded.note,
			updated_at = excluded.updated_at`,
		sqlitehelpers.UUIDToString(rec.AccountID), rec.Month, rec.BankExpenseMinor, rec.Note, rec.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("%w: %s", account.ErrNotFound, rec.AccountID)
		}
		return fmt.Errorf("failed to upsert reconciliation: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, accountID uuid.UUID, month string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM account_reconciliations WHERE account_id = ? AND month = ?`,
		sqlitehelpers.UUIDToString(accountID), month,
	)
	if err != nil {
		return fmt.Errorf("failed to delete reconciliation: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s %s", reconciliation.ErrNotFound, accountID, month)
	}

	return nil
}

// ListByMonth — сверки счетов семьи за месяц `YYYY-MM`.
func (r *SQLiteRepository) ListByMonth(ctx context.Context, month string) ([]*reconciliation.Reconciliation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.account_id, r.month, r.bank_expense_minor, r.note, r.updated_at
		FROM account_reconciliations r
		JOIN accounts a ON a.id = r.account_id
		WHERE a.family_id = (SELECT id FROM families LIMIT 1) AND r.month = ?`,
		month,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list reconciliations: %w", err)
	}
	defer rows.Close()

	recs := make([]*reconciliation.Reconciliation, 0)
	for rows.Next() {
		var (
			rec   reconciliation.Reconciliation
			idStr string
		)
		if err = rows.Scan(&idStr, &rec.Month, &rec.BankExpenseMinor, &rec.Note, &rec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reconciliation: %w", err)
		}
		if rec.AccountID, err = uuid.Parse(idStr); err != nil {
			return nil, fmt.Errorf("invalid reconciliation account id %q: %w", idStr, err)
		}
		recs = append(recs, &rec)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate reconciliations: %w", err)
	}

	return recs, nil
}
