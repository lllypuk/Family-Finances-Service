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

// 002 снимает табличный UNIQUE с уже существующей базы: правку 001 golang-migrate не переигрывает,
// а sqlite_autoindex_* от табличного UNIQUE не удаляется через DROP INDEX.
func TestMigrations_BudgetNameFreedAfterSoftDelete(t *testing.T) {
	ctx := t.Context()
	root := testhelpers.RepoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "budgets.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath, filepath.Join(root, "migrations"))

	require.NoError(t, manager.Up())
	// Откат на версию 1 воспроизводит схему, выкаченную с v0.1.0: табличный UNIQUE поверх всех строк.
	require.NoError(t, manager.Migrate(1))

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	familyID, _, categoryID := seedForChecks(ctx, t, db)
	_, err = db.ExecContext(ctx, insertBudget, "budget-deleted", 0, categoryID, familyID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, insertBudget, "budget-before", 1, categoryID, familyID)
	require.Error(t, err, "на версии 1 табличный UNIQUE считает мягко удалённую строку")

	require.NoError(t, manager.Up())

	assert.Equal(t, 1, objectCount(ctx, t, db, "name = 'idx_budgets_name_period_active'"))
	var kept int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM budgets").Scan(&kept))
	assert.Equal(t, 1, kept, "пересборка таблицы сохраняет строки")

	_, err = db.ExecContext(ctx, insertBudget, "budget-after", 1, categoryID, familyID)
	assert.NoError(t, err, "после 002 имя и период удалённого бюджета свободны")
}

const insertBudget = `
	INSERT INTO budgets (id, name, amount_minor, period, start_date, end_date, is_active, category_id, family_id)
	VALUES (?, 'Еда', 100000, 'monthly', '2026-09-01', '2026-09-30', ?, ?, ?)`
