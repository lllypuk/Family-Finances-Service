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

This runs `go run ./cmd/server backup` (`VACUUM INTO`) with `BACKUP_DIR=./backups`:
- Format: `backup_YYYYMMDD_HHMMSSmmm.db`
- Location: `./backups/backup_20260130_104530123.db`
- Retention: the newest `BACKUP_KEEP` files (default 30); older ones are deleted

## Restore from Backup

```bash
make sqlite-restore BACKUP_FILE=./backups/backup_20260130_104530123.db
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
