# Task 005: Upgrade Script

## Overview

Create a safe upgrade script that updates the Family Budget Service to a new version with automatic backup, rollback
capability, and zero-downtime option.

## Priority: MEDIUM

## Status: COMPLETE

## Completed Items

- [x] Created `deploy/scripts/upgrade.sh` - Comprehensive upgrade script with:
  - **Pre-upgrade checks:**
    - Root privileges verification
    - Installation directory validation
    - Disk space check (500MB minimum)
    - Current version detection
    - Database integrity verification
  
  - **Automatic backup:**
    - Database backup with timestamp
    - Environment file backup
    - Version information storage
    - Container configuration backup
    - Backup directory: `/opt/family-budget/backups/upgrade_YYYYMMDD_HHMMSS/`
  
  - **Safe upgrade process:**
    - Pull new Docker image
    - Graceful service stop (with 5s timeout)
    - Start service with new version
    - Health check verification (60s timeout)
    - Support for specific version tags or latest
  
  - **Automatic rollback:**
    - Triggers on health check failure
    - Restores database from backup
    - Restores previous version
    - Verifies rollback success
    - Can be disabled with `--no-rollback` flag
  
  - **Manual rollback:**
    - Command: `./upgrade.sh rollback`
    - Finds most recent upgrade backup
    - Performs complete restoration
    - Verifies service health
  
  - **User interface:**
    - Color-coded output (info/warning/error/success)
    - Progress indicators
    - Clear error messages
    - Help documentation (`--help`)
    - Command-line options for automation

## Implementation Features

✅ **Safety Features:**
- Pre-flight checks prevent upgrades in unsafe conditions
- Automatic database backup before any changes
- Integrity verification before and after upgrade
- Automatic rollback on failure (optional)
- Preserves configuration files

✅ **Flexibility:**
- Upgrade to specific version: `--version v1.2.3`
- Upgrade to latest: default behavior
- Disable auto-rollback: `--no-rollback`
- Manual rollback command
- Works with any Docker Compose file location

✅ **Error Handling:**
- Comprehensive error messages
- Exit codes (0=success, 1=failure, 2=rollback failed)
- Backup preservation on failure
- Manual recovery instructions

✅ **Zero Downtime Preparation:**
- Health check verification
- Graceful shutdown support
- Can be extended for blue-green deployments

## Usage Examples

### Standard Upgrade:
```bash
sudo ./deploy/scripts/upgrade.sh
```

### Upgrade to Specific Version:
```bash
sudo ./deploy/scripts/upgrade.sh --version v1.2.3
```

### Upgrade Without Auto-Rollback:
```bash
sudo ./deploy/scripts/upgrade.sh --no-rollback
```

### Manual Rollback:
```bash
sudo ./deploy/scripts/upgrade.sh rollback
```

## Remaining Items

## Requirements

### Script Capabilities

1. **Pre-upgrade checks**
    - Verify current installation
    - Check disk space
    - Verify database integrity
    - Check for running processes

2. **Backup before upgrade**
    - Create database backup
    - Save current configuration
    - Store current version info

3. **Upgrade process**
    - Download new version (Docker pull or binary)
    - Stop service gracefully
    - Apply upgrade
    - Run migrations (automatic)
    - Start service
    - Verify health

4. **Rollback capability**
    - Automatic rollback on failure
    - Manual rollback command
    - Restore database from backup

---

### Main Upgrade Script (`deploy/scripts/upgrade.sh`)

```bash
#!/bin/bash
set -euo pipefail

# ============================================
# Family Budget Service - Upgrade Script
# ============================================

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"
DATA_DIR="${DATA_DIR:-${INSTALL_DIR}/data}"
COMPOSE_FILE="${COMPOSE_FILE:-${INSTALL_DIR}/docker-compose.prod.yml}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
ROLLBACK_TIMEOUT=60

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Timestamp for backups
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
UPGRADE_BACKUP_DIR="${BACKUP_DIR}/upgrade_${TIMESTAMP}"

# ============================================
# Pre-upgrade Checks
# ============================================

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_installation() {
    log_info "Checking installation..."

    if [[ ! -d "${INSTALL_DIR}" ]]; then
        log_error "Installation directory not found: ${INSTALL_DIR}"
        exit 1
    fi

    if [[ ! -f "${COMPOSE_FILE}" ]]; then
        log_error "Docker Compose file not found: ${COMPOSE_FILE}"
        exit 1
    fi
}

check_disk_space() {
    log_info "Checking disk space..."

    local available=$(df -BM "${INSTALL_DIR}" | tail -1 | awk '{print $4}' | sed 's/M//')
    local required=500  # 500MB minimum

    if [[ ${available} -lt ${required} ]]; then
        log_error "Insufficient disk space. Available: ${available}MB, Required: ${required}MB"
        exit 1
    fi

    log_info "Disk space OK (${available}MB available)"
}

get_current_version() {
    log_info "Getting current version..."

    CURRENT_VERSION=$(docker inspect --format='{{.Config.Image}}' family-budget-app 2>/dev/null | cut -d: -f2 || echo "unknown")
    log_info "Current version: ${CURRENT_VERSION}"
}

check_database_integrity() {
    log_info "Checking database integrity..."

    local db_file="${DATA_DIR}/budget.db"

    if [[ ! -f "${db_file}" ]]; then
        log_warn "Database file not found, will be created on first run"
        return 0
    fi

    local integrity=$(docker exec family-budget-app sqlite3 /data/budget.db "PRAGMA integrity_check;" 2>/dev/null || echo "error")

    if [[ "${integrity}" != "ok" ]]; then
        log_error "Database integrity check failed: ${integrity}"
        exit 1
    fi

    log_info "Database integrity: OK"
}

# ============================================
# Backup Functions
# ============================================

create_upgrade_backup() {
    log_info "Creating pre-upgrade backup..."

    mkdir -p "${UPGRADE_BACKUP_DIR}"

    # Backup database
    local db_file="${DATA_DIR}/budget.db"
    if [[ -f "${db_file}" ]]; then
        log_info "Backing up database..."
        docker exec family-budget-app sqlite3 /data/budget.db ".backup '/data/pre_upgrade_${TIMESTAMP}.db'"
        cp "${DATA_DIR}/pre_upgrade_${TIMESTAMP}.db" "${UPGRADE_BACKUP_DIR}/budget.db"
        rm -f "${DATA_DIR}/pre_upgrade_${TIMESTAMP}.db"
    fi

    # Backup configuration
    log_info "Backing up configuration..."
    if [[ -f "${INSTALL_DIR}/.env" ]]; then
        cp "${INSTALL_DIR}/.env" "${UPGRADE_BACKUP_DIR}/.env"
    fi

    # Save current image info
    docker inspect family-budget-app > "${UPGRADE_BACKUP_DIR}/container_info.json" 2>/dev/null || true

    # Save version info
    echo "${CURRENT_VERSION}" > "${UPGRADE_BACKUP_DIR}/version.txt"

    log_info "Backup created: ${UPGRADE_BACKUP_DIR}"
}

# ============================================
# Upgrade Functions
# ============================================

pull_new_version() {
    local target_version="${1:-latest}"

    log_info "Pulling new version: ${target_version}..."

    cd "${INSTALL_DIR}"

    # Update image tag in compose file or use environment variable
    export APP_VERSION="${target_version}"

    docker-compose -f "${COMPOSE_FILE}" pull app

    NEW_VERSION=$(docker inspect --format='{{.RepoDigests}}' "ghcr.io/lllypuk/family-finances-service:${target_version}" 2>/dev/null | head -c 20 || echo "${target_version}")
    log_info "Pulled version: ${NEW_VERSION}"
}

stop_service() {
    log_info "Stopping service..."

    cd "${INSTALL_DIR}"
    docker-compose -f "${COMPOSE_FILE}" stop app

    # Wait for graceful shutdown
    sleep 5
}

start_service() {
    log_info "Starting service..."

    cd "${INSTALL_DIR}"
    docker-compose -f "${COMPOSE_FILE}" up -d app

    # Wait for startup
    sleep 10
}

verify_health() {
    log_info "Verifying service health..."

    local attempts=0
    local max_attempts=30

    while [[ ${attempts} -lt ${max_attempts} ]]; do
        local http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "${HEALTH_URL}" 2>/dev/null || echo "000")

        if [[ "${http_code}" == "200" ]]; then
            log_info "Health check passed!"
            return 0
        fi

        attempts=$((attempts + 1))
        log_info "Health check attempt ${attempts}/${max_attempts} (HTTP ${http_code})"
        sleep 2
    done

    log_error "Health check failed after ${max_attempts} attempts"
    return 1
}

# ============================================
# Rollback Functions
# ============================================

rollback() {
    log_error "Upgrade failed, initiating rollback..."

    if [[ ! -d "${UPGRADE_BACKUP_DIR}" ]]; then
        log_error "Backup directory not found, cannot rollback"
        exit 1
    fi

    # Stop current service
    cd "${INSTALL_DIR}"
    docker-compose -f "${COMPOSE_FILE}" stop app 2>/dev/null || true

    # Restore database
    if [[ -f "${UPGRADE_BACKUP_DIR}/budget.db" ]]; then
        log_info "Restoring database..."
        cp "${UPGRADE_BACKUP_DIR}/budget.db" "${DATA_DIR}/budget.db"
    fi

    # Restore previous version
    local prev_version=$(cat "${UPGRADE_BACKUP_DIR}/version.txt" 2>/dev/null || echo "latest")
    log_info "Restoring previous version: ${prev_version}"

    export APP_VERSION="${prev_version}"
    docker-compose -f "${COMPOSE_FILE}" up -d app

    sleep 10

    if verify_health; then
        log_info "Rollback successful"
    else
        log_error "Rollback failed, manual intervention required"
        exit 1
    fi
}

# ============================================
# Main Upgrade Function
# ============================================

upgrade() {
    local target_version="${1:-latest}"

    log_info "=========================================="
    log_info "Family Budget Service Upgrade"
    log_info "=========================================="
    log_info "Target version: ${target_version}"
    log_info ""

    # Pre-flight checks
    check_root
    check_installation
    check_disk_space
    get_current_version

    # Skip if same version
    if [[ "${CURRENT_VERSION}" == "${target_version}" && "${target_version}" != "latest" ]]; then
        log_info "Already running version ${target_version}"
        exit 0
    fi

    check_database_integrity

    # Create backup
    create_upgrade_backup

    # Pull new version
    pull_new_version "${target_version}"

    # Stop service
    stop_service

    # Start with new version
    start_service

    # Verify
    if verify_health; then
        log_info "=========================================="
        log_info "Upgrade completed successfully!"
        log_info "Previous version: ${CURRENT_VERSION}"
        log_info "New version: ${target_version}"
        log_info "Backup location: ${UPGRADE_BACKUP_DIR}"
        log_info "=========================================="
    else
        rollback
        exit 1
    fi
}

# ============================================
# CLI Interface
# ============================================

show_help() {
    cat << EOF
Family Budget Service Upgrade Script

Usage: $0 [command] [options]

Commands:
    upgrade [version]    Upgrade to specified version (default: latest)
    rollback [backup]    Rollback to previous version
    check                Run pre-upgrade checks only
    help                 Show this help message

Options:
    --no-backup          Skip backup creation (not recommended)
    --force              Skip confirmation prompts

Examples:
    $0 upgrade                     # Upgrade to latest
    $0 upgrade v1.2.3              # Upgrade to specific version
    $0 rollback                    # Rollback using latest backup
    $0 check                       # Check readiness for upgrade

EOF
}

# ============================================
# Entry Point
# ============================================

main() {
    local command="${1:-help}"
    shift || true

    case "${command}" in
        upgrade)
            upgrade "${1:-latest}"
            ;;
        rollback)
            rollback
            ;;
        check)
            check_root
            check_installation
            check_disk_space
            get_current_version
            check_database_integrity
            log_info "All checks passed, ready for upgrade"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: ${command}"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
```

---

### Rollback Script (`deploy/scripts/rollback.sh`)

```bash
#!/bin/bash
set -euo pipefail

# Simplified rollback script for manual use

INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"

# List available backups
echo "Available upgrade backups:"
ls -la "${BACKUP_DIR}"/upgrade_* 2>/dev/null || echo "No backups found"

echo ""
read -p "Enter backup directory name (e.g., upgrade_20240115_120000): " BACKUP_NAME

RESTORE_DIR="${BACKUP_DIR}/${BACKUP_NAME}"

if [[ ! -d "${RESTORE_DIR}" ]]; then
    echo "ERROR: Backup not found: ${RESTORE_DIR}"
    exit 1
fi

echo "Restoring from: ${RESTORE_DIR}"
read -p "Continue? (y/N): " confirm

if [[ "${confirm}" != "y" && "${confirm}" != "Y" ]]; then
    echo "Aborted"
    exit 0
fi

# Execute rollback via main script
"${INSTALL_DIR}/scripts/upgrade.sh" rollback "${RESTORE_DIR}"
```

---

## Upgrade Workflow

```
┌─────────────────────┐
│   Pre-flight Check  │
│  - Installation OK  │
│  - Disk space OK    │
│  - DB integrity OK  │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Create Backup     │
│  - Database         │
│  - Configuration    │
│  - Version info     │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Pull New Version  │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Stop Service      │
│   (graceful)        │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Start New Version │
│   (auto-migration)  │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│   Health Check      │
└─────────┬───────────┘
          │
    ┌─────┴─────┐
    │           │
    ▼           ▼
┌───────┐  ┌─────────┐
│  OK   │  │  FAIL   │
│       │  │         │
│ Done! │  │Rollback │
└───────┘  └─────────┘
```

---

## Usage Examples

### Standard Upgrade

```bash
# Upgrade to latest version
sudo /opt/family-budget/scripts/upgrade.sh upgrade

# Upgrade to specific version
sudo /opt/family-budget/scripts/upgrade.sh upgrade v1.2.3
```

### Check Before Upgrade

```bash
# Run pre-flight checks only
sudo /opt/family-budget/scripts/upgrade.sh check
```

### Manual Rollback

```bash
# Rollback to previous version
sudo /opt/family-budget/scripts/upgrade.sh rollback
```

---

## Files to Create

```
deploy/scripts/
├── upgrade.sh         # Main upgrade script
└── rollback.sh        # Simplified rollback helper
```

## Testing Checklist

- [ ] Upgrade from version X to Y works
- [ ] Upgrade to same version is skipped
- [ ] Backup is created before upgrade
- [ ] Health check detects failures
- [ ] Automatic rollback works on failure
- [ ] Manual rollback works
- [ ] Database is preserved after upgrade
- [ ] Configuration is preserved
- [ ] Works with Docker deployment
- [ ] Works with systemd deployment

## Acceptance Criteria

1. Upgrade completes without data loss
2. Automatic rollback on failure
3. Clear progress output
4. Backup created before every upgrade
5. Version history preserved

## Dependencies

- Task 003 (docker-compose.prod.yml) - for Docker upgrades
- Task 004 (systemd) - for native upgrades

## Estimated Complexity

Medium-High
