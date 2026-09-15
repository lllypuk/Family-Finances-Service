package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MetricsReader считает состояние базы для коллектора скрейпа: только COUNT,
// поэтому он живёт отдельно от репозиториев и не расширяет их интерфейсы.
type MetricsReader struct {
	db *sql.DB
}

// NewMetricsReader создаёт читателя счётчиков состояния.
func NewMetricsReader(db *sql.DB) *MetricsReader {
	return &MetricsReader{db: db}
}

// ActiveSessions считает непросроченные на момент now сессии.
func (r *MetricsReader) ActiveSessions(ctx context.Context, now time.Time) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM sessions WHERE expires_at > ?`
	if err := r.db.QueryRowContext(ctx, query, now.UTC().Format(time.RFC3339)).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}
	return count, nil
}

// UsersByActive возвращает число активных и отключённых пользователей.
func (r *MetricsReader) UsersByActive(ctx context.Context) (int, int, error) {
	query := `SELECT is_active, COUNT(*) FROM users GROUP BY is_active`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var active, inactive int
	for rows.Next() {
		var flag, count int
		if scanErr := rows.Scan(&flag, &count); scanErr != nil {
			return 0, 0, fmt.Errorf("failed to scan users count: %w", scanErr)
		}
		if flag == 1 {
			active = count
		} else {
			inactive = count
		}
	}
	if err = rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("failed to read users count: %w", err)
	}
	return active, inactive, nil
}

// Transactions считает все операции.
func (r *MetricsReader) Transactions(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

// SetupComplete сообщает, создана ли семья (cmd/server setup).
func (r *MetricsReader) SetupComplete(ctx context.Context) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM families`).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to count families: %w", err)
	}
	return count > 0, nil
}
