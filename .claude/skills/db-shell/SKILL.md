---
name: db-shell
description: Open interactive SQLite shell to query and inspect the database
disable-model-invocation: true
allowed-tools: Bash(make sqlite-shell), Bash(make sqlite-stats)
---

# SQLite Interactive Shell

Open an interactive SQLite shell to query and inspect the database directly.

## Open Shell

```bash
make sqlite-shell
```

This opens the SQLite CLI connected to `./data/budget.db`

## Common Queries

### View all tables
```sql
.tables
```

### View table schema
```sql
.schema users
.schema transactions
.schema budgets
```

### Query data
```sql
SELECT * FROM users LIMIT 10;
SELECT COUNT(*) FROM transactions;
SELECT * FROM budgets WHERE status = 'active';
```

### View database statistics
Exit the shell and run:
```bash
make sqlite-stats
```

Shows:
- Database file size
- Number of tables
- Row counts per table
- Index information

## Useful SQLite Commands

- `.help` - Show all available commands
- `.exit` or `.quit` - Exit the shell
- `.mode` - Change output format (column, csv, json, etc.)
- `.headers on` - Show column headers
- `.schema` - Show all table schemas
- `.dump [table]` - Export table as SQL
- `.read file.sql` - Execute SQL from file

## Safety Tips

1. **Read-only queries**: Use SELECT for inspection
2. **Backup first**: Run `/db-backup` before any modifications
3. **Test in dev**: Never experiment on production database
4. **Use transactions**: Wrap modifications in BEGIN/COMMIT

## Example Session

```sql
-- Connect to database
sqlite> .tables

-- View users
sqlite> SELECT id, email, role FROM users;

-- Count transactions by type
sqlite> SELECT type, COUNT(*) FROM transactions GROUP BY type;

-- Check budget status
sqlite> SELECT name, amount, spent FROM budgets ORDER BY created_at DESC LIMIT 5;

-- Exit
sqlite> .exit
```

## See Also

- `/db-backup` - Create database backup
- `make sqlite-stats` - View database statistics
- `make migrate-create` - Create new migration
