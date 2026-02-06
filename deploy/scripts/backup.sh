#!/bin/bash
# Family Budget Service - Database Backup Script
# This script creates backups of the SQLite database with integrity checks

set -euo pipefail

# Configuration from environment or defaults
BACKUP_DIR="${BACKUP_DIR:-/opt/family-budget/backups}"
DATA_DIR="${DATA_DIR:-/opt/family-budget/data}"
DATABASE_FILE="${DATABASE_PATH:-$DATA_DIR/budget.db}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
MAX_BACKUPS="${MAX_BACKUPS:-50}"

# Timestamp for backup filename
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/budget_${TIMESTAMP}.db"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Error handler
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Create backup directory if not exists
mkdir -p "${BACKUP_DIR}" || error_exit "Cannot create backup directory: ${BACKUP_DIR}"

# Check if database exists
if [[ ! -f "${DATABASE_FILE}" ]]; then
    error_exit "Database file not found: ${DATABASE_FILE}"
fi

# Check if database is accessible
if [[ ! -r "${DATABASE_FILE}" ]]; then
    error_exit "Database file is not readable: ${DATABASE_FILE}"
fi

# Get database size
DB_SIZE=$(stat -c%s "${DATABASE_FILE}" 2>/dev/null || echo "unknown")
log "Starting backup of ${DATABASE_FILE} (${DB_SIZE} bytes)"

# Check if sqlite3 is available
if ! command -v sqlite3 &>/dev/null; then
    error_exit "sqlite3 command not found. Please install sqlite3."
fi

# Create backup using SQLite backup command (safe for live database)
# This method ensures consistency even if database is being written to
if sqlite3 "${DATABASE_FILE}" ".backup '${BACKUP_FILE}'" 2>/dev/null; then
    log "Backup file created: ${BACKUP_FILE}"
else
    error_exit "Failed to create backup"
fi

# Verify backup file was created
if [[ ! -f "${BACKUP_FILE}" ]]; then
    error_exit "Backup file was not created: ${BACKUP_FILE}"
fi

# Get backup file size
BACKUP_SIZE=$(stat -c%s "${BACKUP_FILE}" 2>/dev/null || echo "0")
log "Backup file size: ${BACKUP_SIZE} bytes"

# Verify backup file is not empty
if [[ "${BACKUP_SIZE}" -eq 0 ]]; then
    rm -f "${BACKUP_FILE}"
    error_exit "Backup file is empty"
fi

# Verify backup integrity using SQLite pragma
log "Verifying backup integrity..."
if sqlite3 "${BACKUP_FILE}" "PRAGMA integrity_check;" 2>/dev/null | grep -q "ok"; then
    log "Backup integrity check: PASSED"
else
    rm -f "${BACKUP_FILE}"
    error_exit "Backup integrity check FAILED"
fi

# Verify backup can be opened and has expected tables
TABLE_COUNT=$(sqlite3 "${BACKUP_FILE}" "SELECT COUNT(*) FROM sqlite_master WHERE type='table';" 2>/dev/null || echo "0")
if [[ "${TABLE_COUNT}" -gt 0 ]]; then
    log "Backup contains ${TABLE_COUNT} tables"
else
    rm -f "${BACKUP_FILE}"
    error_exit "Backup appears to be invalid (no tables found)"
fi

# Cleanup old backups by age
log "Cleaning up backups older than ${RETENTION_DAYS} days..."
REMOVED_BY_AGE=$(find "${BACKUP_DIR}" -name "budget_*.db" -type f -mtime +${RETENTION_DAYS} -delete -print | wc -l)
if [[ "${REMOVED_BY_AGE}" -gt 0 ]]; then
    log "Removed ${REMOVED_BY_AGE} old backup(s)"
fi

# Cleanup by count (keep only MAX_BACKUPS most recent)
log "Keeping only ${MAX_BACKUPS} most recent backups..."
BACKUP_COUNT=$(ls -1 "${BACKUP_DIR}"/budget_*.db 2>/dev/null | wc -l)
if [[ "${BACKUP_COUNT}" -gt "${MAX_BACKUPS}" ]]; then
    REMOVE_COUNT=$((BACKUP_COUNT - MAX_BACKUPS))
    ls -t "${BACKUP_DIR}"/budget_*.db | tail -n "${REMOVE_COUNT}" | xargs -r rm -f
    log "Removed ${REMOVE_COUNT} excess backup(s)"
fi

# Report final status
TOTAL_BACKUPS=$(ls -1 "${BACKUP_DIR}"/budget_*.db 2>/dev/null | wc -l)
TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" 2>/dev/null | cut -f1 || echo "unknown")
log "Backup complete successfully"
log "Total backups: ${TOTAL_BACKUPS}, Total size: ${TOTAL_SIZE}"

exit 0
