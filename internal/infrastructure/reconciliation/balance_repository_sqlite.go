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

type BalanceSQLiteRepository struct {
	db *sql.DB
}

func NewBalanceSQLiteRepository(db *sql.DB) *BalanceSQLiteRepository {
	return &BalanceSQLiteRepository{db: db}
}

// Upsert пишет updated_at из Go: CURRENT_TIMESTAMP с секундной точностью не различает два upsert подряд.
func (r *BalanceSQLiteRepository) Upsert(ctx context.Context, b *reconciliation.Balance) error {
	b.UpdatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_balances (account_id, month, balance_minor, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(account_id, month) DO UPDATE SET
			balance_minor = excluded.balance_minor,
			updated_at = excluded.updated_at`,
		sqlitehelpers.UUIDToString(b.AccountID), b.Month, b.BalanceMinor, b.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return fmt.Errorf("%w: %s", account.ErrNotFound, b.AccountID)
		}
		return fmt.Errorf("failed to upsert account balance: %w", err)
	}

	return nil
}

func (r *BalanceSQLiteRepository) Delete(ctx context.Context, accountID uuid.UUID, month string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM account_balances WHERE account_id = ? AND month = ?`,
		sqlitehelpers.UUIDToString(accountID), month,
	)
	if err != nil {
		return fmt.Errorf("failed to delete account balance: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s %s", reconciliation.ErrBalanceNotFound, accountID, month)
	}

	return nil
}

// ByMonths — остатки счетов семьи за два месяца `YYYY-MM` одним запросом.
func (r *BalanceSQLiteRepository) ByMonths(ctx context.Context, prev, month string) ([]*reconciliation.Balance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.account_id, b.month, b.balance_minor, b.updated_at
		FROM account_balances b
		JOIN accounts a ON a.id = b.account_id
		WHERE a.family_id = (SELECT id FROM families LIMIT 1) AND b.month IN (?, ?)`,
		prev, month,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list account balances: %w", err)
	}
	defer rows.Close()

	balances := make([]*reconciliation.Balance, 0)
	for rows.Next() {
		var (
			b     reconciliation.Balance
			idStr string
		)
		if err = rows.Scan(&idStr, &b.Month, &b.BalanceMinor, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan account balance: %w", err)
		}
		if b.AccountID, err = uuid.Parse(idStr); err != nil {
			return nil, fmt.Errorf("invalid account balance account id %q: %w", idStr, err)
		}
		balances = append(balances, &b)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate account balances: %w", err)
	}

	return balances, nil
}
