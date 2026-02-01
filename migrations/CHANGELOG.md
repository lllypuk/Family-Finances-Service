# Migrations Changelog

All notable changes to database migrations will be documented in this file.

## [2025-01-12] - Consolidated Migrations

### Changed
- **Migration Strategy**: Migrated from multiple versioned migration files to consolidated approach
- **Structure**: All migrations now consolidated in two files:
  - `001_consolidated.up.sql` - All forward migrations
  - `001_consolidated.down.sql` - All rollback migrations

### Migration History Consolidated

#### Migration 001: Initial Schema
- Created base tables: `families`, `users`, `categories`, `transactions`, `budgets`, `budget_alerts`, `reports`, `user_sessions`
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
- See [../.memory_bank/tech_stack.md](../.memory_bank/tech_stack.md) for architecture details

---

## Future Changes

When adding new database changes:
1. Add SQL statements to end of `001_consolidated.up.sql`
2. Add corresponding DROP statements to beginning of `001_consolidated.down.sql`
3. Test with `make clean && make run-local`
4. Document changes in this CHANGELOG