# Database Migrations

This directory contains database schema migrations for the Family Budget Service.

## Migration Strategy

`001_consolidated.{up,down}.sql` holds the **complete schema**: a fresh database is built from that one file, and
it stays the readable picture of the tables. Since `v0.1.0` is deployed, a schema change also needs a numbered
migration next to it — an already-applied `001` is never re-run (see "Changing the schema" below).

- `001_consolidated.up.sql` - the full schema
- `001_consolidated.down.sql` - the full rollback
- `002_budgets_name_period_partial_unique.{up,down}.sql` - the budget name/period `UNIQUE` → partial index
- `003_drop_reports.{up,down}.sql` - the `reports` table dropped (plan 10)
- `004_budgets_recurring.{up,down}.sql` - `budgets.recurring` / `budgets.series_id` (plan 12)
- `005_accounts.{up,down}.sql` - `accounts`, `account_reconciliations`, `transactions.account_id` (plan 16)
- `006_holdings.{up,down}.sql` - `holdings`, `holding_values` (plan 17)
- `007_holding_plans.{up,down}.sql` - `holding_plans` (plan 18)
- `008_account_balances.{up,down}.sql` - `account_balances` instead of `account_reconciliations` (plan 20)

### Why Consolidated Migrations?

1. **Simplicity**: Single source of truth for the complete database schema
2. **Clarity**: Easy to understand the entire database structure at a glance
3. **SQLite Compatibility**: Works seamlessly with SQLite's embedded nature
4. **Maintainability**: Easier to review and modify the complete schema

## Migration Files

### `001_consolidated.up.sql`

Contains all database objects in order of dependencies:

1. **Tables** (in FK order):

   | Table | Notes |
   |---|---|
   | `families` | `singleton` UNIQUE — ровно одна семья на инсталляцию; `timezone` (IANA) |
   | `users` | `role` CHECK (`admin`/`member`), `is_active` |
   | `categories` | `income`/`expense`, самоссылка `parent_id`, `color`/`icon` для клиента |
   | `accounts` | справочник счетов; `name_key` UNIQUE в семье, `is_archived` |
   | `transactions` | `amount_minor INTEGER > 0`, `date TEXT 'YYYY-MM-DD'` (CHECK GLOB), `account_id` nullable, FK `RESTRICT` |
   | `account_balances` | остаток счёта на конец месяца (`YYYY-MM`) со знаком, PK `(account_id, month)`, FK `RESTRICT` |
   | `budgets` | `amount_minor`, `spent_minor`, период `start_date`/`end_date` — `TEXT`-даты |
   | `holdings` | активы и пассивы: `side` (`asset`/`liability`), `kind`, `name_key` UNIQUE в семье, `is_archived` |
   | `holding_values` | снимки стоимости, PK `(holding_id, date)`, `value_minor >= 0`, FK `CASCADE` |
   | `sessions` | bearer-токены: только `token_hash` |

2. **Indexes**: только те, что закрывают реальные запросы (семья+дата, категория, автор)
3. **Triggers**: `updated_at` для families, users, categories, accounts, transactions, budgets, holdings; у остатков и снимков триггера нет — `updated_at` пишет upsert
4. **Analytics**: Statistics updates (ANALYZE)

`budget_alerts`, `invites` и `user_sessions` удалены; таблицу `schema_migrations` ведёт golang-migrate.

### `001_consolidated.down.sql`

Contains rollback statements in **reverse order**:

1. Drop triggers
2. Drop indexes
3. Drop tables (in reverse dependency order)

## Changing the schema

Every schema change is written **twice**:

1. **`001_consolidated.up.sql`** — the DDL in its final form (plus the matching `DROP` at the front of the
   `.down.sql`). This is what a new install gets.
2. **`NNN_<name>.{up,down}.sql`** — the same change as a step from the previous version, because golang-migrate
   records only the version number (`schema_migrations`): on a database already at version 1 `Up()` returns
   `ErrNoChange` and edits to `001` are skipped silently. The test path
   (`internal/testhelpers/sqlite.go`) always starts from an empty in-memory DB and will not reveal this.

On a fresh database `NNN` re-applies what `001` already did, so write it to be harmless there — `002` rebuilds
`budgets` into a table identical to the one `001` creates. Keep the two definitions in sync.

**Exception to "`002` = `001`": columns added after `v0.2.0`.** `002` is frozen on the `v0.2.0` schema, so a
column added later lives in `001` (for a new install) and in its own `NNN` — and on a fresh database `002`
drops what `001` created, because its `INSERT ... SELECT` lists the columns of that older table. `004` then puts
`recurring` / `series_id` back. Do not add new columns to the `budgets` definition inside `002`: it must keep
reproducing the released schema, or the upgrade path it tests stops being the one the server runs.

`005` has the same shape for `transactions`: `account_id` comes in by a rebuild (`ADD COLUMN` would fail with
`duplicate column` on a fresh database, where `001` already created it), so the `transactions` block inside
`005` is frozen on the `v0.6.0` schema. A later transaction column goes into `001` and its own `NNN`.

Cover the step in `internal/infrastructure/migrations_test.go`: `manager.Migrate(N-1)` puts the released schema
back on a temp file, so the upgrade a server will actually run is what the test exercises.

### SQLite: removing a table constraint

`DROP INDEX` refuses the implicit `sqlite_autoindex_*` that backs a table-level `UNIQUE`, and SQLite has no
`ALTER TABLE ... DROP CONSTRAINT`. The only way out is the rebuild in `002`: create the new table → `INSERT
... SELECT` → `DROP TABLE` the old one → `ALTER TABLE ... RENAME`. Dropping the table takes its indexes and its
`updated_at` trigger with it — recreate both.

## Migration Workflow

### Automatic Execution

Migrations run automatically when the application starts:

```go
// internal/bootstrap.go — internal.OpenDatabase, общий для сервера и CLI (setup, reset-password)
if err = infrastructure.NewMigrationManager(dbURL, migrationsDir).Up(); err != nil {
    return nil, fmt.Errorf("failed to run migrations: %w", err)
}
```

### Manual Migration Commands

The service ships the same thing as a subcommand — it reads `DATABASE_PATH` and refuses to run when that
file does not exist (golang-migrate would otherwise create an empty database and stamp it):

```bash
go run ./cmd/server migrate           # текущая версия схемы
go run ./cmd/server migrate --to 2    # подвинуть версию, в том числе вниз
```

In the container it is `docker compose run --rm --no-deps -T app migrate --to 2`. Stepping down is what
makes a release rollback possible: an old image does not start on a version it has no file for
(`deploy/README.md`). `--to 0` is rejected — the lowest reachable target is 1.

## Best Practices

### ✅ DO

- Use `IF NOT EXISTS` for CREATE statements
- Use `IF EXISTS` for DROP statements
- Add comments explaining the purpose of each migration section
- Test both UP and DOWN migrations
- Keep statements idempotent (safe to run multiple times)
- Add indexes for foreign keys and frequently queried columns
- Use CHECK constraints for data validation

### ❌ DON'T

- Забыть про `NNN` рядом с правкой `001` — на выкаченной базе `001` уже не переигрывается
- Forget to update the DOWN migration
- Forget `make db-reset` локально, если правка `001` не сопровождается `NNN`
- Use database-specific features (keep it SQLite compatible)

## SQLite-Specific Considerations

### Supported Features

- ✅ `CREATE TABLE IF NOT EXISTS`
- ✅ `CREATE INDEX IF NOT EXISTS`
- ✅ `CREATE TRIGGER IF NOT EXISTS`
- ✅ Foreign keys (must enable with `PRAGMA foreign_keys = ON`)
- ✅ Partial indexes with `WHERE` clause
- ✅ Triggers for automatic timestamp updates

### NOT Supported (PostgreSQL features)

- ❌ `DROP INDEX CONCURRENTLY`
- ❌ Schemas (no `CREATE SCHEMA`)
- ❌ Custom types (use TEXT with CHECK constraints)
- ❌ `ENUM` types (use TEXT with CHECK constraints)
- ❌ `TIMESTAMP WITH TIME ZONE` (use `DATETIME`)
- ❌ Function-based indexes (use simpler indexes)

### Type Mappings

| PostgreSQL | SQLite | Notes |
|------------|--------|-------|
| `UUID` | `TEXT` | Store as string |
| `ENUM` | `TEXT` + `CHECK` | Validate with constraints |
| `DECIMAL` (деньги) | `INTEGER` | Минимальные единицы, колонки `*_minor` |
| `DATE` | `TEXT` | `YYYY-MM-DD`, CHECK GLOB |
| `TIMESTAMP WITH TIME ZONE` | `DATETIME` | UTC recommended |
| `SERIAL` | Not needed | Use TEXT for UUIDs |
| `BOOLEAN` | `INTEGER` | 0 = false, 1 = true |

## Troubleshooting

### Database in Dirty State

If migrations fail and the database is marked as "dirty":

```bash
# Force migration version (use with caution!)
migrate -path ./migrations -database "sqlite://./data/budget.db" force 1
```

### Reset Database

```bash
# Complete reset
make db-reset
make run-local
```

### Verify Schema

```bash
# Check database schema
make sqlite-shell
.schema

# Check applied migrations
SELECT * FROM schema_migrations;
```

## Migration History

| Version | Description | Date |
|---------|-------------|------|
| 001 | Initial consolidated schema | 2025-01-12 |
| | - All base tables (families, users, categories, etc.) | |
| | - Performance indexes | |
| | - Automatic timestamp triggers | |
| | - User invitation system | |
| 001 | Bearer auth (plan 03): `sessions` replaces `user_sessions`; `families.singleton` UNIQUE | 2026-09-05 |
| 001 | Plan 04: файл переписан одним куском; `*_minor INTEGER` вместо `REAL`, даты — `TEXT`, `families.timezone`, роль только `admin`/`member`; `budget_alerts` и `invites` удалены | 2026-09-06 |
| 002 | Plan 09: табличный `UNIQUE` бюджетов → частичный индекс `idx_budgets_name_period_active` (`WHERE is_active = 1`) пересборкой таблицы | 2026-09-12 |
| 003 | Plan 10: таблица `reports` и её индексы удалены вместе с `/api/v1/reports` | 2026-09-13 |
| 004 | Plan 12: `budgets.recurring` и `budgets.series_id` — периодические бюджеты | 2026-09-14 |
| 005 | Plan 16: счета, сверки, `transactions.account_id` пересборкой таблицы; откат теряет счета, сверки и привязку операций | 2026-09-18 |
| 006 | Plan 17: `holdings` и `holding_values` — активы, пассивы и снимки; откат теряет все позиции и всю их историю | 2026-09-18 |
| 007 | Plan 18: `holding_plans` — плановые поступления и выплаты позиций; откат теряет все планы | 2026-09-19 |

## See Also

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [SQLite SQL syntax](https://www.sqlite.org/lang.html)
- [Project CLAUDE.md](../CLAUDE.md) - Development guide
- `make help` - Available Make commands