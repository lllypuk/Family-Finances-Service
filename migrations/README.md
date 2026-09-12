# Database Migrations

This directory contains database schema migrations for the Family Budget Service.

## Migration Strategy

`001_consolidated.{up,down}.sql` holds the **complete schema**: a fresh database is built from that one file, and
it stays the readable picture of the tables. Since `v0.1.0` is deployed, a schema change also needs a numbered
migration next to it — an already-applied `001` is never re-run (see "Changing the schema" below).

- `001_consolidated.up.sql` - the full schema
- `001_consolidated.down.sql` - the full rollback
- `002_budgets_name_period_partial_unique.{up,down}.sql` - the budget name/period `UNIQUE` → partial index

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
   | `transactions` | `amount_minor INTEGER > 0`, `date TEXT 'YYYY-MM-DD'` (CHECK GLOB) |
   | `budgets` | `amount_minor`, `spent_minor`, период `start_date`/`end_date` — `TEXT`-даты |
   | `reports` | период `start_date`/`end_date` — `TEXT`-даты, `data` — JSON отчёта с суммами `*_minor` |
   | `sessions` | bearer-токены: только `token_hash` |

2. **Indexes**: только те, что закрывают реальные запросы (семья+дата, категория, автор)
3. **Triggers**: `updated_at` для families, users, categories, transactions, budgets
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

While the project uses `golang-migrate` library, manual execution is typically not needed since migrations run on startup.

If needed for testing:
```bash
# Run migrations
migrate -path ./migrations -database "sqlite://./data/budget.db" up

# Rollback migrations
migrate -path ./migrations -database "sqlite://./data/budget.db" down
```

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

## See Also

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [SQLite SQL syntax](https://www.sqlite.org/lang.html)
- [Project CLAUDE.md](../CLAUDE.md) - Development guide
- `make help` - Available Make commands