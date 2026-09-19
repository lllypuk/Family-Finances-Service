package infrastructure_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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

	tables := []string{"families", "users", "categories", "transactions", "budgets", "sessions",
		"accounts", "account_balances", "holdings", "holding_values", "holding_plans"}
	for _, table := range tables {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count), table)
	}

	for _, dropped := range []string{"budget_alerts", "invites", "user_sessions", "reports", "account_reconciliations"} {
		assert.Equal(t, 0, objectCount(ctx, t, db, "name = '"+dropped+"'"), dropped+" удалена")
	}

	// Деньги — целые в минимальных единицах, даты операций и периодов — TEXT (план 04).
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "transactions", "amount_minor"))
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "amount_minor"))
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "spent_minor"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "transactions", "date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "start_date"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "end_date"))
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

// 003 снимает reports с базы, стоящей на версии 2: правку 001 golang-migrate ей не отдаст.
func TestMigrations_DropReports(t *testing.T) {
	ctx := t.Context()
	root := testhelpers.RepoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "reports.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath, filepath.Join(root, "migrations"))

	require.NoError(t, manager.Up())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	// Версия 2 — схема, выкаченная до этого плана: 003.down возвращает таблицу с индексами.
	require.NoError(t, manager.Migrate(2))
	assert.Equal(t, 1, objectCount(ctx, t, db, "name = 'reports'"))
	assert.Equal(t, 2, objectCount(ctx, t, db, "name LIKE 'idx_reports%'"))

	require.NoError(t, manager.Up())
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'reports'"))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name LIKE 'idx_reports%'"))

	// Полный откат идёт через 003.down, которая таблицу воссоздаёт, — убирает её 001.down.
	require.NoError(t, manager.Down())
	left := objectCount(ctx, t, db, "type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'")
	assert.Equal(t, 0, left, "down не оставляет таблиц")
}

// 004 добавляет колонки серии базе на версии 3; на свежей 002 пересобирает budgets по схеме v0.2.0,
// то есть снимает то, что создала 001, — колонки возвращает та же 004.
func TestMigrations_BudgetsRecurring(t *testing.T) {
	ctx := t.Context()
	root := testhelpers.RepoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "recurring.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath, filepath.Join(root, "migrations"))

	require.NoError(t, manager.Up())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "recurring"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "series_id"))

	// Версия 3 — схема, выкаченная до этого плана.
	require.NoError(t, manager.Migrate(3))
	assert.Empty(t, columnType(ctx, t, db, "budgets", "recurring"))
	assert.Empty(t, columnType(ctx, t, db, "budgets", "series_id"))

	require.NoError(t, manager.Up())
	assert.Equal(t, "INTEGER", columnType(ctx, t, db, "budgets", "recurring"))
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "budgets", "series_id"))

	familyID, _, categoryID := seedForChecks(ctx, t, db)
	_, err = db.ExecContext(ctx, insertBudget, "budget-recurring", 1, categoryID, familyID)
	require.NoError(t, err)

	var recurring int
	var seriesID sql.NullString
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT recurring, series_id FROM budgets WHERE id = 'budget-recurring'").Scan(&recurring, &seriesID))
	assert.Equal(t, 0, recurring, "старая строка получает значение по умолчанию")
	assert.False(t, seriesID.Valid)

	_, err = db.ExecContext(ctx, "UPDATE budgets SET recurring = 2 WHERE id = 'budget-recurring'")
	require.Error(t, err, "recurring вне 0/1 обязан отбиваться CHECK-ом")
}

// 005 пересобирает transactions базе на версии 4: строки, timestamps и триггер обязаны пережить пересборку.
func TestMigrations_AccountsUpgradeKeepsTransactions(t *testing.T) {
	ctx := t.Context()
	manager, db := migratedDB(t)

	// Up → Migrate(4), а не Migrate(4) на пустой базе: так воспроизводится выпущенная v4.
	require.NoError(t, manager.Migrate(4))
	assert.Empty(t, columnType(ctx, t, db, "transactions", "account_id"))

	familyID, userID, categoryID := seedForChecks(ctx, t, db)
	_, err := db.ExecContext(ctx, `
		INSERT INTO transactions (id, amount_minor, type, description, date, category_id, user_id, family_id,
		                          tags, created_at, updated_at)
		VALUES ('tx-old', 1500, 'expense', 'check', '2026-09-04', ?, ?, ?, '["a"]',
		        '2026-09-04 10:00:00', '2026-09-05 11:00:00')`, categoryID, userID, familyID)
	require.NoError(t, err)

	require.NoError(t, manager.Up())

	var amount int64
	var tags, createdAt, updatedAt string
	var accountID sql.NullString
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT amount_minor, tags, created_at, updated_at, account_id FROM transactions WHERE id = 'tx-old'`).
		Scan(&amount, &tags, &createdAt, &updatedAt, &accountID))
	assert.Equal(t, int64(1500), amount)
	assert.JSONEq(t, `["a"]`, tags)
	assert.Contains(t, createdAt, "2026-09-04")
	assert.Contains(t, updatedAt, "2026-09-05", "пересборка не переписывает updated_at")
	assert.False(t, accountID.Valid)

	for _, index := range []string{
		"idx_transactions_family_date", "idx_transactions_family_type_date", "idx_transactions_category_id",
		"idx_transactions_user_id", "idx_transactions_account_date",
	} {
		assert.Equal(t, 1, objectCount(ctx, t, db, "type = 'index' AND name = '"+index+"'"), index)
	}

	_, err = db.ExecContext(ctx, "UPDATE transactions SET description = 'edited' WHERE id = 'tx-old'")
	require.NoError(t, err)
	require.NoError(t, db.QueryRowContext(ctx, "SELECT updated_at FROM transactions WHERE id = 'tx-old'").
		Scan(&updatedAt))
	assert.NotContains(t, updatedAt, "2026-09-05", "триггер updated_at пересоздан")
}

// 005.down снимает счета и сверки, а операции оставляет.
func TestMigrations_AccountsRollback(t *testing.T) {
	ctx := t.Context()
	manager, db := migratedDB(t)
	// account_reconciliations есть до 007 включительно: 008 её роняет.
	require.NoError(t, manager.Migrate(7))

	familyID, userID, categoryID := seedForChecks(ctx, t, db)
	_, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, family_id, name, name_key) VALUES ('acc-1', ?, 'Карта', 'карта')`, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, insertTransaction, "tx-1", 100, "2026-09-04", categoryID, userID, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE transactions SET account_id = 'acc-1' WHERE id = 'tx-1'")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO account_reconciliations (account_id, month, bank_expense_minor) VALUES ('acc-1', '2026-09', 100)`)
	require.NoError(t, err)

	require.NoError(t, manager.Migrate(4))

	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM transactions").Scan(&count))
	assert.Equal(t, 1, count, "операции переживают откат")
	assert.Empty(t, columnType(ctx, t, db, "transactions", "account_id"))
	for _, dropped := range []string{"accounts", "account_reconciliations", "update_accounts_updated_at",
		"idx_transactions_account_date"} {
		assert.Equal(t, 0, objectCount(ctx, t, db, "name = '"+dropped+"'"), dropped)
	}

	require.NoError(t, manager.Up())
	assert.Equal(t, "TEXT", columnType(ctx, t, db, "transactions", "account_id"))
}

// Живая база (v4 → 005) и свежая (001 уже со счетами) обязаны прийти к одной схеме. Сравнение идёт
// по pragma, а не по тексту sqlite_master: DDL в 001 и 005 оформлен по-разному.
func TestMigrations_AccountsSchemaMatchesFreshInstall(t *testing.T) {
	ctx := t.Context()
	_, fresh := migratedDB(t)
	upgradedManager, upgraded := migratedDB(t)
	require.NoError(t, upgradedManager.Migrate(4))
	require.NoError(t, upgradedManager.Up())

	for _, table := range []string{"transactions", "accounts", "account_balances"} {
		assert.Equal(t, schemaOf(ctx, t, fresh, table), schemaOf(ctx, t, upgraded, table), table)
	}
}

// 006 на живой базе версии 5: позиции появляются, данные плана 16 не трогаются.
func TestMigrations_HoldingsUpgradeKeepsAccounts(t *testing.T) {
	ctx := t.Context()
	manager, db := migratedDB(t)
	require.NoError(t, manager.Migrate(5))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name IN ('holdings', 'holding_values')"))

	familyID, userID, categoryID := seedForChecks(ctx, t, db)
	_, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, family_id, name, name_key) VALUES ('acc-1', ?, 'Карта', 'карта')`, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, insertTransaction, "tx-1", 100, "2026-09-04", categoryID, userID, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE transactions SET account_id = 'acc-1' WHERE id = 'tx-1'")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO account_reconciliations (account_id, month, bank_expense_minor) VALUES ('acc-1', '2026-09', 100)`)
	require.NoError(t, err)

	// Не Up: 008 роняет account_reconciliations.
	require.NoError(t, manager.Migrate(7))

	var accountID string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT account_id FROM transactions WHERE id = 'tx-1'").
		Scan(&accountID))
	assert.Equal(t, "acc-1", accountID)
	var bank int64
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT bank_expense_minor FROM account_reconciliations WHERE account_id = 'acc-1'").Scan(&bank))
	assert.Equal(t, int64(100), bank)

	_, err = db.ExecContext(ctx, `
		INSERT INTO holdings (id, family_id, name, name_key, side, kind, updated_at)
		VALUES ('h-1', ?, 'Вклад', 'вклад', 'asset', 'deposit', '2026-09-01 10:00:00')`, familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO holding_values (holding_id, date, value_minor) VALUES ('h-1', '2026-09-01', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE holdings SET is_archived = 1 WHERE id = 'h-1'")
	require.NoError(t, err)
	var updatedAt string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT updated_at FROM holdings WHERE id = 'h-1'").Scan(&updatedAt))
	assert.NotContains(t, updatedAt, "2026-09-01", "триггер updated_at создан")
}

// Живая база (v5 → 006) и свежая (001 уже с позициями) обязаны прийти к одной схеме.
func TestMigrations_HoldingsSchemaMatchesFreshInstall(t *testing.T) {
	ctx := t.Context()
	_, fresh := migratedDB(t)
	upgradedManager, upgraded := migratedDB(t)
	require.NoError(t, upgradedManager.Migrate(5))
	require.NoError(t, upgradedManager.Up())

	for _, table := range []string{"holdings", "holding_values"} {
		assert.Equal(t, schemaOf(ctx, t, fresh, table), schemaOf(ctx, t, upgraded, table), table)
	}
}

// 007 на живой базе версии 6: план появляется, позиции и снимки не трогаются; откат теряет только планы.
func TestMigrations_HoldingPlansUpgradeAndRollback(t *testing.T) {
	ctx := t.Context()
	manager, db := migratedDB(t)
	require.NoError(t, manager.Migrate(6))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'holding_plans'"))

	familyID, _, _ := seedForChecks(ctx, t, db)
	for _, id := range []string{"h-1", "h-2"} {
		_, err := db.ExecContext(ctx, `
			INSERT INTO holdings (id, family_id, name, name_key, side, kind) VALUES (?, ?, ?, ?, 'asset', 'deposit')`,
			id, familyID, id, id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx,
			`INSERT INTO holding_values (holding_id, date, value_minor) VALUES (?, '2026-09-01', 500)`, id)
		require.NoError(t, err)
	}

	require.NoError(t, manager.Up())
	assert.Equal(t, 2, rowCount(ctx, t, db, "holdings"))
	assert.Equal(t, 2, rowCount(ctx, t, db, "holding_values"))

	// Каскад работает только с foreign_keys=ON, а pragma живёт на соединении.
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer func() { assert.NoError(t, conn.Close()) }()
	_, err = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	const insertPlan = `
		INSERT INTO holding_plans (holding_id, monthly_income_minor, monthly_expense_minor) VALUES (?, ?, ?)`
	_, err = conn.ExecContext(ctx, insertPlan, "h-1", 0, 0)
	require.Error(t, err, "план 0/0 обязан отбиваться CHECK-ом")
	_, err = conn.ExecContext(ctx, insertPlan, "h-1", -1, 100)
	require.Error(t, err, "отрицательный доход обязан отбиваться CHECK-ом")
	_, err = conn.ExecContext(ctx, insertPlan, "h-1", 100, -1)
	require.Error(t, err, "отрицательный расход обязан отбиваться CHECK-ом")
	_, err = conn.ExecContext(ctx, insertPlan, "h-1", 4500000, 0)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, insertPlan, "h-2", 0, 830000)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, "DELETE FROM holdings WHERE id = 'h-2'")
	require.NoError(t, err)
	var plans int
	require.NoError(t, conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM holding_plans WHERE holding_id = 'h-2'").Scan(&plans))
	assert.Equal(t, 0, plans, "удаление позиции уносит план")

	require.NoError(t, manager.Migrate(6))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'holding_plans'"), "откат теряет планы")
	assert.Equal(t, 1, rowCount(ctx, t, db, "holdings"))
	assert.Equal(t, 1, rowCount(ctx, t, db, "holding_values"))

	require.NoError(t, manager.Up())
	assert.Equal(t, 0, rowCount(ctx, t, db, "holding_plans"))
	assert.Equal(t, 1, rowCount(ctx, t, db, "holdings"))
}

// Живая база (v6 → 007) и свежая (001 уже с планами) обязаны прийти к одной схеме.
func TestMigrations_HoldingPlansSchemaMatchesFreshInstall(t *testing.T) {
	ctx := t.Context()
	_, fresh := migratedDB(t)
	upgradedManager, upgraded := migratedDB(t)
	require.NoError(t, upgradedManager.Migrate(6))
	require.NoError(t, upgradedManager.Up())

	assert.Equal(t, schemaOf(ctx, t, fresh, "holding_plans"), schemaOf(ctx, t, upgraded, "holding_plans"))
}

// 008 на живой базе версии 7: остатки появляются вместо сверок, счета не трогаются; откат теряет остатки и
// возвращает сверки пустыми.
func TestMigrations_AccountBalancesUpgradeAndRollback(t *testing.T) {
	ctx := t.Context()
	manager, db := migratedDB(t)
	require.NoError(t, manager.Migrate(7))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'account_balances'"))

	familyID, _, _ := seedForChecks(ctx, t, db)
	_, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, family_id, name, name_key) VALUES ('a-1', ?, 'Карта', 'карта')`, familyID)
	require.NoError(t, err)
	const insertReconciliation = `
		INSERT INTO account_reconciliations (account_id, month, bank_expense_minor) VALUES ('a-1', '2026-09', 100)`
	_, err = db.ExecContext(ctx, insertReconciliation)
	require.NoError(t, err)

	require.NoError(t, manager.Up())
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'account_reconciliations'"), "сверки удалены")

	const insertBalance = `INSERT INTO account_balances (account_id, month, balance_minor) VALUES ('a-1', ?, ?)`
	_, err = db.ExecContext(ctx, insertBalance, "2026-9", 100)
	require.Error(t, err, "месяц не в формате YYYY-MM обязан отбиваться CHECK-ом")
	_, err = db.ExecContext(ctx, insertBalance, "2026-08", -1500)
	require.NoError(t, err, "остаток кредитки отрицательный")
	_, err = db.ExecContext(ctx, insertBalance, "2026-09", 0)
	require.NoError(t, err)

	require.NoError(t, manager.Migrate(7))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'account_balances'"), "откат теряет остатки")
	assert.Equal(t, 1, rowCount(ctx, t, db, "accounts"))
	assert.Equal(t, 0, rowCount(ctx, t, db, "account_reconciliations"), "сверки возвращаются пустыми")
	_, err = db.ExecContext(ctx, insertReconciliation)
	require.NoError(t, err, "старый образ пишет сверки")

	require.NoError(t, manager.Up())
	assert.Equal(t, 0, rowCount(ctx, t, db, "account_balances"))
	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'account_reconciliations'"))
}

// Живая база (v7 → 008) и свежая (001 уже с остатками) обязаны прийти к одной схеме.
func TestMigrations_AccountBalancesSchemaMatchesFreshInstall(t *testing.T) {
	ctx := t.Context()
	_, fresh := migratedDB(t)
	upgradedManager, upgraded := migratedDB(t)
	require.NoError(t, upgradedManager.Migrate(7))
	require.NoError(t, upgradedManager.Up())

	assert.Equal(t, schemaOf(ctx, t, fresh, "account_balances"), schemaOf(ctx, t, upgraded, "account_balances"))
}

func rowCount(ctx context.Context, t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count))
	return count
}

func migratedDB(t *testing.T) (*infrastructure.MigrationManager, *sql.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "accounts.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath,
		filepath.Join(testhelpers.RepoRoot(t), "migrations"))
	require.NoError(t, manager.Up())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	return manager, db
}

// schemaOf снимает колонки, внешние ключи и индексы таблицы в сравнимом виде.
func schemaOf(ctx context.Context, t *testing.T, db *sql.DB, table string) []string {
	t.Helper()

	schema := pragmaRows(ctx, t, db, "SELECT * FROM pragma_table_info(?)", table)
	schema = append(schema, pragmaRows(ctx, t, db, "SELECT * FROM pragma_foreign_key_list(?)", table)...)
	indexes := pragmaRows(ctx, t, db,
		`SELECT name || '|' || "unique" || '|' || origin || '|' || partial FROM pragma_index_list(?) ORDER BY name`,
		table)
	schema = append(schema, indexes...)
	for _, index := range indexes {
		name, _, _ := strings.Cut(index, "|")
		schema = append(schema, pragmaRows(ctx, t, db, "SELECT * FROM pragma_index_xinfo(?)", name)...)
	}

	return schema
}

func pragmaRows(ctx context.Context, t *testing.T, db *sql.DB, query, arg string) []string {
	t.Helper()

	rows, err := db.QueryContext(ctx, query, arg)
	require.NoError(t, err)
	defer func() { assert.NoError(t, rows.Close()) }()

	columns, err := rows.Columns()
	require.NoError(t, err)

	var out []string
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		out = append(out, fmt.Sprint(values...))
	}
	require.NoError(t, rows.Err())

	return out
}
