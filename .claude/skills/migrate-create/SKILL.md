---
name: migrate-create
description: Add a schema change to the consolidated migration files
disable-model-invocation: true
argument-hint: [migration_name]
allowed-tools: Bash(make migrate-create *)
---

# Add a Database Migration

Guide for changing the schema in the consolidated migration files.

## Usage

```bash
make migrate-create NAME=add_user_preferences
```

Replace `add_user_preferences` with a descriptive name for your migration.

## What Happens

`make migrate-create` creates nothing: the project keeps the whole schema in two consolidated files. The
target prints the three steps to follow by hand:

1. Append the new DDL to the end of `migrations/001_consolidated.up.sql`
2. Add the matching `DROP` statements to the front of `migrations/001_consolidated.down.sql` (reverse order)
3. Recreate the local database — an already-migrated DB ignores edits to `001`:
   ```bash
   make db-reset && make run-local
   ```

Also add any new table to `CleanTables` in `internal/testhelpers/sqlite.go`.

## Migration Naming Conventions

Use descriptive names that indicate the change:

✅ **Good names:**
- `add_email_to_users`
- `create_categories_table`
- `add_index_on_transactions_date`
- `remove_deprecated_columns`

❌ **Bad names:**
- `migration1`
- `update`
- `fix`
- `changes`

## Example Migration Content

### Up (appended to `001_consolidated.up.sql`)
```sql
-- Add user preferences column
ALTER TABLE users ADD COLUMN preferences TEXT DEFAULT '{}';

-- Create index for faster lookups
CREATE INDEX idx_users_preferences ON users(preferences);
```

### Down (prepended to `001_consolidated.down.sql`)
```sql
-- Remove index
DROP INDEX IF EXISTS idx_users_preferences;

-- Remove column
ALTER TABLE users DROP COLUMN preferences;
```

## Migration Best Practices

1. **Atomic changes**: One logical change per migration
2. **Always reversible**: Write down migration that undoes up migration
3. **Test both directions**: Verify up and down migrations work
4. **Safe operations**: Consider impact on existing data
5. **Backup first**: Use `/db-backup` before applying migrations

## Automatic Execution

Migrations run automatically:
- **On startup**: `internal.OpenDatabase` — the server, `make docker-up`, and the CLI `setup`/`reset-password`
- **Once only**: golang-migrate records the applied version, so edits to `001` need `make db-reset`

## Migration Workflow

1. **Read the reminder**:
   ```bash
   make migrate-create NAME=add_budget_categories
   ```

2. **Edit up migration**:
   Append SQL to `migrations/001_consolidated.up.sql`

3. **Edit down migration**:
   Prepend the reverting SQL to `migrations/001_consolidated.down.sql`

4. **Test locally**:
   ```bash
   make db-reset && make run-local
   ```

5. **Verify in database**:
   ```bash
   make sqlite-shell
   sqlite> .schema  # Check schema changes
   ```

6. **Commit to version control**:
   ```bash
   git add migrations/
   git commit -m "Add budget categories migration"
   ```

## SQLite-Specific Considerations

SQLite has some limitations compared to PostgreSQL:

- **No DROP COLUMN**: Use table recreation pattern
- **Limited ALTER TABLE**: Some operations require table rebuild
- **No concurrent DDL**: Migrations lock the database

Example table recreation pattern:
```sql
-- Create new table with desired schema
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    -- new column
    preferences TEXT DEFAULT '{}'
);

-- Copy data
INSERT INTO users_new (id, email)
SELECT id, email FROM users;

-- Swap tables
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
```

## Troubleshooting

### Migration fails
1. Check SQL syntax in migration files
2. Verify SQLite compatibility
3. Check logs: `make docker-logs` or console output
4. Restore from backup: `/db-backup`

### Migration already applied
Migrations are tracked in `schema_migrations` table. Don't modify applied migrations.

### Need to rollback
Currently, down migrations are not automatically executed. To rollback:
1. Backup: `/db-backup`
2. Manually run down SQL in `/db-shell`
3. Remove migration record from `schema_migrations`

## See Also

- `/db-shell` - Execute SQL directly
- `/db-backup` - Backup before migrations
- `make sqlite-stats` - View database statistics
