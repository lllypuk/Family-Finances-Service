---
name: db-backup
description: Create a backup of the SQLite database
disable-model-invocation: true
allowed-tools: Bash(make sqlite-backup), Bash(make sqlite-restore *)
---

# SQLite Database Backup

Create and manage backups of the SQLite database.

## Create Backup

```bash
make sqlite-backup
```

This creates a timestamped backup in `./backups/` directory:
- Format: `budget_YYYYMMDD_HHMMSS.db`
- Location: `./backups/budget_20260130_104530.db`

## Restore from Backup

```bash
make sqlite-restore BACKUP_FILE=./backups/budget_20260130_104530.db
```

**⚠️ WARNING**: This will overwrite the current database at `./data/budget.db`

## Best Practices

1. **Before major changes**: Always backup before migrations or bulk updates
2. **Regular backups**: Schedule periodic backups for production
3. **Test restores**: Verify backup integrity by testing restore process
4. **Keep multiple versions**: Maintain at least 3-5 recent backups

## Backup Location

All backups are stored in `./backups/` directory (created automatically).

## Database Location

- **Development**: `./data/budget.db`
- **Docker**: Persisted in Docker volume at `./data/`

## See Also

- `/db-shell` - Open interactive SQLite shell
- `make sqlite-stats` - View database statistics
