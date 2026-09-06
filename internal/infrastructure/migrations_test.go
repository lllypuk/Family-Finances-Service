package infrastructure_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Pure Go SQLite driver

	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/testhelpers"
)

// Тестовый путь (testhelpers) исполняет только *.up.sql, поэтому ветка down проверяется здесь.
func TestMigrations_UpAndDownOnEmptyDatabase(t *testing.T) {
	ctx := t.Context()
	root := testhelpers.RepoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "migrations.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath, filepath.Join(root, "migrations"))

	require.NoError(t, manager.Up())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	tables := []string{"families", "users", "categories", "transactions", "budgets", "reports", "sessions"}
	for _, table := range tables {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count), table)
	}

	for _, dropped := range []string{"budget_alerts", "invites", "user_sessions"} {
		assert.Equal(t, 0, objectCount(ctx, t, db, "name = '"+dropped+"'"), dropped+" удалена")
	}

	// Деньги — целые в минимальных единицах, даты операций и периодов — TEXT (план 04).
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "transactions", "amount_minor"))
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "amount_minor"))
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "spent_minor"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "transactions", "date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "start_date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "end_date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "reports", "start_date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "families", "timezone"))
	assert.Empty(t, columnType(ctx, t, db, "transactions", "amount"), "старая колонка amount не должна остаться")

	familyID, userID, categoryID := seedForChecks(ctx, t, db)

	// CHECK-и, на которые опирается контракт: сумма строго больше нуля и дата — YYYY-MM-DD.
	_, err = db.ExecContext(ctx, insertTransaction, "tx-zero", 0, "2026-09-04", categoryID, userID, familyID)
	require.Error(t, err, "amount_minor = 0 обязан отбиваться CHECK-ом")
	_, err = db.ExecContext(ctx, insertTransaction, "tx-date", 100, "04.09.2026", categoryID, userID, familyID)
	require.Error(t, err, "дата не в формате YYYY-MM-DD обязана отбиваться CHECK-ом")
	_, err = db.ExecContext(ctx, insertTransaction, "tx-ok", 100, "2026-09-04", categoryID, userID, familyID)
	require.NoError(t, err)

	require.NoError(t, manager.Down())
	left := objectCount(ctx, t, db, "type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'")
	assert.Equal(t, 0, left, "down не оставляет таблиц")
}

func objectCount(ctx context.Context, t *testing.T, db *sql.DB, where string) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE "+where).Scan(&count))
	return count
}

const insertTransaction = `
	INSERT INTO transactions (id, amount_minor, type, description, date, category_id, user_id, family_id)
	VALUES (?, ?, 'expense', 'check', ?, ?, ?, ?)`

// seedForChecks создаёт минимальный набор строк, без которого FK не дадут проверить CHECK-и.
func seedForChecks(ctx context.Context, t *testing.T, db *sql.DB) (string, string, string) {
	t.Helper()

	const familyID, userID, categoryID = "fam-1", "user-1", "cat-1"
	_, err := db.ExecContext(ctx,
		`INSERT INTO families (id, name, currency, timezone) VALUES (?, 'F', 'RUB', 'Europe/Moscow')`, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, first_name, last_name, role, family_id)
		 VALUES (?, 'admin@example.com', 'hash', 'A', 'B', 'admin', ?)`, userID, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO categories (id, name, type, family_id) VALUES (?, 'C', 'expense', ?)`,
		categoryID, familyID)
	require.NoError(t, err)

	return familyID, userID, categoryID
}

// columnType возвращает объявленный тип колонки или "" если её нет.
func columnType(ctx context.Context, t *testing.T, db *sql.DB, table, column string) string {
	t.Helper()

	rows, err := db.QueryContext(ctx, "SELECT name, type FROM pragma_table_info(?)", table)
	require.NoError(t, err)
	defer func() { assert.NoError(t, rows.Close()) }()

	for rows.Next() {
		var name, columnType string
		require.NoError(t, rows.Scan(&name, &columnType))
		if name == column {
			return columnType
		}
	}
	require.NoError(t, rows.Err())

	return ""
}
