# Database Migrations

This directory contains database schema migrations for the Family Budget Service.

## Migration Strategy

This project uses a **consolidated migration approach** with two files:

- `001_consolidated.up.sql` - Contains all schema creation statements
- `001_consolidated.down.sql` - Contains all schema rollback statements

### Why Consolidated Migrations?

1. **Simplicity**: Single source of truth for the complete database schema
2. **Clarity**: Easy to understand the entire database structure at a glance
3. **SQLite Compatibility**: Works seamlessly with SQLite's embedded nature
4. **Maintainability**: Easier to review and modify the complete schema

## Migration Files

### `001_consolidated.up.sql`

Contains all database objects in order of dependencies:

1. **Tables**: families, users, categories, transactions, budgets, budget_alerts, reports, user_sessions, invites
2. **Indexes**: Performance optimization indexes for all tables
3. **Triggers**: Automatic timestamp updates for all tables
4. **Analytics**: Statistics updates (ANALYZE)

### `001_consolidated.down.sql`

Contains rollback statements in **reverse order**:

1. Drop triggers
2. Drop indexes
3. Drop tables (in reverse dependency order)

## Adding New Migrations

To add a new migration (e.g., adding a new table or column):

### 1. Update `001_consolidated.up.sql`

Add your changes at the **end** of the appropriate section:

```sql
-- ==============================================================================
-- Migration XXX: Your migration description
-- ==============================================================================

-- Add your CREATE statements here
CREATE TABLE IF NOT EXISTS your_new_table (
    id TEXT PRIMARY KEY,
    -- ... columns ...
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_your_table_field ON your_new_table(field);

-- Add triggers if needed
CREATE TRIGGER IF NOT EXISTS update_your_table_updated_at
AFTER UPDATE ON your_new_table
BEGIN
    UPDATE your_new_table SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 2. Update `001_consolidated.down.sql`

Add corresponding DROP statements at the **beginning** of the file (reverse order):

```sql
-- ==============================================================================
-- Migration XXX Down: Rollback your migration
-- ==============================================================================

-- Drop in reverse order of creation
DROP TRIGGER IF EXISTS update_your_table_updated_at;
DROP INDEX IF EXISTS idx_your_table_field;
DROP TABLE IF EXISTS your_new_table;

-- ... rest of existing down migrations ...
```

### 3. Test Your Migration

```bash
# Clean database and test migration
make clean
make run-local

# Or test with Docker
make docker-down
make docker-up
```

## Migration Workflow

### Automatic Execution

Migrations run automatically when the application starts:

```go
// In internal/run.go
migrationManager := infrastructure.NewMigrationManager(dbURL, "./migrations")
if err := migrationManager.Up(); err != nil {
    return fmt.Errorf("migration failed: %w", err)
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

- Remove existing migration code (only add new)
- Modify existing tables without considering backwards compatibility
- Forget to update the DOWN migration
- Add breaking changes without a migration path
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
| `DECIMAL` | `REAL` | Floating point |
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
make clean
rm -f data/budget.db
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

## See Also

- [golang-migrate documentation](https://github.com/golang-migrate/migrate)
- [SQLite SQL syntax](https://www.sqlite.org/lang.html)
- [Project CLAUDE.md](../CLAUDE.md) - Development guide
- `make help` - Available Make commands