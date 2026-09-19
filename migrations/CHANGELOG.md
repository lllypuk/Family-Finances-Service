# Migrations Changelog

All notable changes to database migrations will be documented in this file.

## [2026-09-19] - Plan 18: holding monthly plans

### Added
- `holding_plans` (PK `holding_id`, FK `CASCADE`, оба числа `>= 0`, CHECK «хоть одно больше нуля»):
  строка есть, пока план ненулевой.
- `007_holding_plans.{up,down}.sql` — таблица новая, `CREATE ... IF NOT EXISTS`, на свежей базе — no-op.

### Rollback
- `007.down` удаляет таблицу: теряются все планы, позиции и снимки остаются. Перед `migrate --to 6` — бэкап.

## [2026-09-18] - Plan 17: holdings and net worth

### Added
- `holdings` (`side` CHECK `asset`/`liability`, `kind` проверяется в домене, `name_key` UNIQUE в семье,
  `is_archived`, триггер `update_holdings_updated_at`) и `holding_values` (PK `(holding_id, date)`,
  `value_minor >= 0`, FK `CASCADE`).
- `006_holdings.{up,down}.sql` — обе таблицы новые, `CREATE ... IF NOT EXISTS` без пересборок, на свежей
  базе — no-op.

### Rollback
- `006.down` удаляет обе таблицы: теряются все позиции и вся история снимков. Перед `migrate --to 5` — бэкап.

## [2026-09-18] - Plan 16: accounts and reconciliations

### Added
- `accounts` (`name_key` UNIQUE в семье, `is_archived`, триггер `update_accounts_updated_at`),
  `account_reconciliations` (PK `(account_id, month)`, `bank_expense_minor >= 0`),
  `transactions.account_id` (FK `RESTRICT`) и `idx_transactions_account_date`.
- `005_accounts.{up,down}.sql` — то же для базы версии 4. `account_id` добавляется пересборкой
  `transactions`, чтобы колонка встала на тот же `cid`, что в `001`; блок `transactions` в `005`
  заморожен на схеме `v0.6.0`, как `budgets` в `002`.

### Rollback
- `005.down` удаляет счета, сверки и `account_id`: операции остаются, их привязка к счетам и все сверки
  теряются. Перед `migrate --to 4` — бэкап.

## [2026-09-14] - Plan 12: recurring budgets

### Added
- `budgets.recurring INTEGER NOT NULL DEFAULT 0 CHECK (recurring IN (0, 1))` — хвост серии, который
  сервер продлевает при чтении, и `budgets.series_id TEXT` (без FK: первый инстанс мягко удаляется).
- `004_budgets_recurring.{up,down}.sql` — те же две колонки для базы версии 3. На свежей базе `004`
  не дублирует `001`: `002` пересобирает `budgets` по схеме `v0.2.0`, то есть без этих колонок, и `004`
  возвращает их после пересборки. `002` на новые колонки не трогаем — она обязана воспроизводить
  выкаченную схему.

## [2026-09-13] - Plan 10: reports dropped

### Removed
- Таблица `reports` с индексами `idx_reports_family_type` / `idx_reports_generated_by` — из `001`
  и, для выкаченной базы версии 2, миграцией `003_drop_reports.{up,down}.sql`. На свежей базе `003`
  — no-op. `003.down` восстанавливает только схему: данных в таблице на проде не было.
- `001.down` продолжает удалять `reports`: полный откат проходит через `003.down`, которая её
  создаёт заново.

## [2026-09-12] - Plan 09: budget name uniqueness

### Changed
- `budgets`: табличный `UNIQUE (family_id, name, start_date, end_date)` → частичный
  `idx_budgets_name_period_active ... WHERE is_active = 1`, чтобы мягко удалённая строка не занимала
  имя и период

### Added
- `002_budgets_name_period_partial_unique.{up,down}.sql` — та же правка для уже выкаченной базы
  (`v0.1.0`): `001` на ней не переигрывается, а табличный `UNIQUE` в SQLite снимается только
  пересборкой таблицы. На свежей базе это пересборка в ту же схему.

## [2026-09-06] - Plan 04: money, dates, roles

### Changed
- `001_consolidated.up.sql` rewritten as one file; `.down.sql` drops in reverse order
- `transactions.amount REAL` → `amount_minor INTEGER NOT NULL CHECK (amount_minor > 0)`;
  `budgets` gains `amount_minor` / `spent_minor`
- `transactions.date`, `budgets.start_date/end_date`, `reports.start_date/end_date` → `TEXT`
  `'YYYY-MM-DD'` with a `CHECK … GLOB` guard
- `families` gains `timezone TEXT NOT NULL` (IANA); `users.role` CHECK narrowed to `admin`/`member`

### Removed
- Tables `budget_alerts` and `invites` with their indexes and triggers

## [2026-09-05] - Plan 03: bearer auth

### Changed
- `user_sessions` replaced by `sessions` (token hash only); `families.singleton` UNIQUE added

## [2025-01-12] - Consolidated Migrations

### Changed
- **Migration Strategy**: Migrated from multiple versioned migration files to consolidated approach
- **Structure**: All migrations now consolidated in two files:
  - `001_consolidated.up.sql` - All forward migrations
  - `001_consolidated.down.sql` - All rollback migrations

### Migration History Consolidated

#### Migration 001: Initial Schema
- Created base tables: `families`, `users`, `categories`, `transactions`, `budgets`, `budget_alerts`, `reports`, `user_sessions` (replaced by `sessions` in 2026-09, see below)
- Created performance indexes for all tables
- Created triggers for automatic `updated_at` timestamp updates
- Enabled foreign key constraints with `PRAGMA foreign_keys = ON`

#### Migration 002: Budget Trigger Fix
- **Status**: Deprecated (logic moved to Go code)
- Trigger logic now handled in `BudgetRepository` for better testability

#### Migration 003: Performance Optimization Indexes
- Added `idx_transactions_monthly_summary` for monthly summary queries
- Added `idx_transactions_complex_filter` for complex filtering
- Added `idx_transactions_summary_calc` for summary calculations
- Added `idx_categories_hierarchy` for category hierarchy operations
- Added `idx_transactions_pagination` for efficient pagination
- Added `idx_transactions_budget_calc` for budget calculations
- Added `idx_budgets_active_lookup` for active budget lookups
- Added `ANALYZE` to update SQLite query optimizer statistics

#### Migration 004: Budget Alerts Schema Fix
- **Status**: Deprecated (logic moved to Go code)
- Alert checking now handled in application layer

#### Migration 005: User Invitations
- Created `invites` table for user invitation system
- Added indexes: `idx_invites_token`, `idx_invites_family_id`, `idx_invites_email`, `idx_invites_status`, `idx_invites_expires_at`
- Added trigger `update_invites_updated_at` for automatic timestamp updates
- Supports invitation statuses: `pending`, `accepted`, `expired`, `revoked`

### Benefits of Consolidated Approach

1. **Simplicity**: Single source of truth for complete database schema
2. **Clarity**: Easy to understand entire database structure at a glance
3. **Maintainability**: Simpler to review and modify complete schema
4. **SQLite Compatibility**: Better alignment with SQLite's embedded nature
5. **Reduced Complexity**: No need to track multiple migration versions

### Migration Files Removed

The following individual migration files were consolidated:
- `001_initial_schema.up/down.sql`
- `002_fix_budget_trigger.up/down.sql`
- `003_performance_indexes.up/down.sql`
- `004_fix_budget_alerts_schema.up/down.sql`
- `1769938780_create_invites_table.up/down.sql`

### Developer Impact

- **Adding Migrations**: Edit consolidated files directly instead of creating new versioned files
- **Make Command**: `make migrate-create` now shows guide for editing consolidated files
- **Testing**: Same process - migrations run automatically on application startup
- **Rollback**: Complete rollback via `001_consolidated.down.sql`

### Technical Details

- **Version**: All migrations consolidated as version `001`
- **Tool**: Uses `golang-migrate/migrate` v4.19.1
- **Auto-execution**: Migrations run automatically via `MigrationManager.Up()` on startup
- **Idempotency**: All statements use `IF NOT EXISTS` / `IF EXISTS` for safe re-runs

### Documentation

- See [README.md](./README.md) for detailed migration guide
- See [../CLAUDE.md](../CLAUDE.md) for development workflow
- See [../docs/tech_stack.md](../docs/tech_stack.md) for architecture details

---

## [2026-09-05] - Bearer auth (plan 03)

- `user_sessions` dropped; `sessions` (token hash, device, idle/absolute expiry) added with `idx_sessions_user_id`
- `families.singleton` UNIQUE — one family per instance, enforced by the schema
- Existing databases must be recreated: `make db-reset && make run-local` (an already-applied `001` is not re-run)

## Future Changes

When adding new database changes:
1. Add SQL statements to end of `001_consolidated.up.sql`
2. Add corresponding DROP statements to beginning of `001_consolidated.down.sql`
3. Add `NNN_<name>.{up,down}.sql` applying the same change to a deployed database, and cover it in
   `internal/infrastructure/migrations_test.go`
4. Test with `make db-reset && make run-local`
5. Document changes in this CHANGELOG