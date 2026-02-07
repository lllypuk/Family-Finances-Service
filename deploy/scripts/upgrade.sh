#!/bin/bash
# Family Budget Service - Upgrade Script
# Safely upgrades the application with automatic backup and rollback

set -euo pipefail

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"
DATA_DIR="${DATA_DIR:-${INSTALL_DIR}/data}"
COMPOSE_FILE="${COMPOSE_FILE:-${INSTALL_DIR}/docker-compose.yml}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
HEALTH_CHECK_TIMEOUT=60
HEALTH_CHECK_INTERVAL=2

# Source common functions if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/lib/common.sh" ]]; then
    source "${SCRIPT_DIR}/lib/common.sh"
else
    # Define basic logging functions if common.sh not available
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
    log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
    log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
    log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
    log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
fi

# Timestamp for backups
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
UPGRADE_BACKUP_DIR="${BACKUP_DIR}/upgrade_${TIMESTAMP}"
CURRENT_VERSION=""
TARGET_VERSION="latest"
ROLLBACK_ON_FAILURE=true

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                TARGET_VERSION="$2"
                shift 2
                ;;
            --no-rollback)
                ROLLBACK_ON_FAILURE=false
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            rollback)
                manual_rollback
                exit $?
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat <<EOF
Family Budget Service - Upgrade Script

Usage: sudo ./upgrade.sh [OPTIONS] [COMMAND]

OPTIONS:
    --version VERSION    Target version (default: latest)
    --no-rollback        Disable automatic rollback on failure
    --help, -h           Show this help message

COMMANDS:
    rollback             Manually rollback to previous version

EXAMPLES:
    # Upgrade to latest version
    sudo ./upgrade.sh

    # Upgrade to specific version
    sudo ./upgrade.sh --version v1.2.3

    # Upgrade without auto-rollback
    sudo ./upgrade.sh --no-rollback

    # Manual rollback
    sudo ./upgrade.sh rollback

EOF
}

# ============================================
# Pre-upgrade Checks
# ============================================

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
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
        log_info "Checked: ${COMPOSE_FILE}"
        exit 1
    fi

    log_success "Installation directory found"
}

check_disk_space() {
    log_info "Checking disk space..."

    local available=$(df -BM "${INSTALL_DIR}" | tail -1 | awk '{print $4}' | sed 's/M//')
    local required=500  # 500MB minimum

    if [[ ${available} -lt ${required} ]]; then
        log_error "Insufficient disk space. Available: ${available}MB, Required: ${required}MB"
        exit 1
    fi

    log_success "Disk space OK (${available}MB available)"
}

get_current_version() {
    log_info "Getting current version..."

    if docker ps --format '{{.Names}}' | grep -q 'family-budget'; then
        CURRENT_VERSION=$(docker inspect --format='{{.Config.Image}}' $(docker ps -q --filter "name=family-budget" | head -1) 2>/dev/null | cut -d: -f2 || echo "unknown")
    else
        CURRENT_VERSION="unknown"
    fi

    log_info "Current version: ${CURRENT_VERSION}"
}

check_database_integrity() {
    log_info "Checking database integrity..."

    local db_file="${DATA_DIR}/budget.db"

    if [[ ! -f "${db_file}" ]]; then
        log_warning "Database file not found, will be created on first run"
        return 0
    fi

    # Try to check integrity if container is running
    if docker ps --format '{{.Names}}' | grep -q 'family-budget'; then
        local integrity=$(docker exec $(docker ps -q --filter "name=family-budget" | head -1) sh -c "sqlite3 /data/budget.db 'PRAGMA integrity_check;'" 2>/dev/null || echo "cannot_check")

        if [[ "${integrity}" != "ok" && "${integrity}" != "cannot_check" ]]; then
            log_error "Database integrity check failed: ${integrity}"
            exit 1
        fi

        if [[ "${integrity}" == "ok" ]]; then
            log_success "Database integrity: OK"
        else
            log_warning "Could not check database integrity (container may not be running)"
        fi
    else
        log_warning "Container not running, skipping database integrity check"
    fi
}

# ============================================
# Backup Functions
# ============================================

create_upgrade_backup() {
    log_info "Creating pre-upgrade backup..."

    mkdir -p "${UPGRADE_BACKUP_DIR}"

    # Backup database if exists
    local db_file="${DATA_DIR}/budget.db"
    if [[ -f "${db_file}" ]]; then
        log_info "Backing up database..."
        if docker ps --format '{{.Names}}' | grep -q 'family-budget'; then
            docker exec $(docker ps -q --filter "name=family-budget" | head -1) sh -c "sqlite3 /data/budget.db \".backup '/data/pre_upgrade_${TIMESTAMP}.db'\""
            cp "${DATA_DIR}/pre_upgrade_${TIMESTAMP}.db" "${UPGRADE_BACKUP_DIR}/budget.db"
            rm -f "${DATA_DIR}/pre_upgrade_${TIMESTAMP}.db"
        else
            cp "${db_file}" "${UPGRADE_BACKUP_DIR}/budget.db"
        fi
        log_success "Database backed up"
    fi

    # Backup environment file
    if [[ -f "${INSTALL_DIR}/.env" ]]; then
        log_info "Backing up environment file..."
        cp "${INSTALL_DIR}/.env" "${UPGRADE_BACKUP_DIR}/.env"
    fi
    if [[ -f "${INSTALL_DIR}/config/.env" ]]; then
        mkdir -p "${UPGRADE_BACKUP_DIR}/config"
        cp "${INSTALL_DIR}/config/.env" "${UPGRADE_BACKUP_DIR}/config/.env"
    fi

    # Save current version info
    echo "${CURRENT_VERSION}" > "${UPGRADE_BACKUP_DIR}/version.txt"
    date > "${UPGRADE_BACKUP_DIR}/backup_time.txt"

    # Save container info if running
    if docker ps --format '{{.Names}}' | grep -q 'family-budget'; then
        docker inspect $(docker ps -q --filter "name=family-budget" | head -1) > "${UPGRADE_BACKUP_DIR}/container_info.json" 2>/dev/null || true
    fi

    log_success "Pre-upgrade backup created: ${UPGRADE_BACKUP_DIR}"
}

# ============================================
# Upgrade Functions
# ============================================

pull_new_version() {
    log_info "Pulling new version: ${TARGET_VERSION}..."

    cd "${INSTALL_DIR}"

    # Set version environment variable
    export APP_VERSION="${TARGET_VERSION}"

    # Pull new image
    if docker compose -f "${COMPOSE_FILE}" pull app 2>/dev/null; then
        log_success "Successfully pulled version: ${TARGET_VERSION}"
    else
        log_error "Failed to pull new version"
        return 1
    fi
}

stop_service() {
    log_info "Stopping service gracefully..."

    cd "${INSTALL_DIR}"

    if docker compose -f "${COMPOSE_FILE}" stop app 2>/dev/null; then
        log_success "Service stopped"
    else
        log_warning "Failed to stop service gracefully, forcing stop..."
        docker compose -f "${COMPOSE_FILE}" down 2>/dev/null || true
    fi

    # Wait for graceful shutdown
    sleep 5
}

start_service() {
    log_info "Starting service with new version..."

    cd "${INSTALL_DIR}"
    
    # Set version if needed
    export APP_VERSION="${TARGET_VERSION}"

    docker compose -f "${COMPOSE_FILE}" up -d app

    # Wait for startup
    log_info "Waiting for service to start..."
    sleep 10
}

verify_health() {
    log_info "Verifying service health..."

    local elapsed=0
    local max_wait=${HEALTH_CHECK_TIMEOUT}

    while [[ ${elapsed} -lt ${max_wait} ]]; do
        local http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "${HEALTH_URL}" 2>/dev/null || echo "000")

        if [[ "${http_code}" == "200" ]]; then
            log_success "Health check passed!"
            return 0
        fi

        log_info "Health check attempt (${elapsed}s/${max_wait}s): HTTP ${http_code}"
        sleep ${HEALTH_CHECK_INTERVAL}
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
    done

    log_error "Health check failed after ${max_wait} seconds"
    return 1
}

# ============================================
# Rollback Functions
# ============================================

rollback() {
    log_error "Upgrade failed, initiating automatic rollback..."

    if [[ ! -d "${UPGRADE_BACKUP_DIR}" ]]; then
        log_error "Backup directory not found, cannot rollback: ${UPGRADE_BACKUP_DIR}"
        log_error "Manual intervention required"
        return 1
    fi

    # Stop current service
    log_info "Stopping failed upgrade..."
    cd "${INSTALL_DIR}"
    docker compose -f "${COMPOSE_FILE}" stop app 2>/dev/null || true
    sleep 3

    # Restore database
    if [[ -f "${UPGRADE_BACKUP_DIR}/budget.db" ]]; then
        log_info "Restoring database from backup..."
        cp "${UPGRADE_BACKUP_DIR}/budget.db" "${DATA_DIR}/budget.db"
        log_success "Database restored"
    fi

    # Restore environment files
    if [[ -f "${UPGRADE_BACKUP_DIR}/.env" ]]; then
        cp "${UPGRADE_BACKUP_DIR}/.env" "${INSTALL_DIR}/.env"
    fi
    if [[ -f "${UPGRADE_BACKUP_DIR}/config/.env" ]]; then
        cp "${UPGRADE_BACKUP_DIR}/config/.env" "${INSTALL_DIR}/config/.env"
    fi

    # Restore previous version
    local prev_version=$(cat "${UPGRADE_BACKUP_DIR}/version.txt" 2>/dev/null || echo "latest")
    log_info "Restoring previous version: ${prev_version}"

    export APP_VERSION="${prev_version}"
    docker compose -f "${COMPOSE_FILE}" up -d app

    sleep 10

    if verify_health; then
        log_success "Rollback successful! Service is running on previous version: ${prev_version}"
        return 0
    else
        log_error "Rollback failed! Manual intervention required!"
        log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
        return 1
    fi
}

manual_rollback() {
    log_info "=== Manual Rollback ==="
    
    # Find most recent upgrade backup
    local latest_backup=$(ls -td "${BACKUP_DIR}"/upgrade_* 2>/dev/null | head -1)
    
    if [[ -z "${latest_backup}" ]]; then
        log_error "No upgrade backups found in ${BACKUP_DIR}"
        exit 1
    fi
    
    log_info "Found backup: ${latest_backup}"
    UPGRADE_BACKUP_DIR="${latest_backup}"
    
    rollback
}

# ============================================
# Main Upgrade Function
# ============================================

upgrade() {
    echo ""
    log_info "╔════════════════════════════════════════════════════════════════╗"
    log_info "║       Family Budget Service - Upgrade Process                 ║"
    log_info "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Target version: ${TARGET_VERSION}"
    log_info "Backup directory: ${UPGRADE_BACKUP_DIR}"
    log_info "Auto-rollback: ${ROLLBACK_ON_FAILURE}"
    echo ""

    # Pre-flight checks
    check_root
    check_installation
    check_disk_space
    get_current_version

    # Skip if same version
    if [[ "${CURRENT_VERSION}" == "${TARGET_VERSION}" && "${TARGET_VERSION}" != "latest" ]]; then
        log_info "Already running version ${TARGET_VERSION}"
        log_info "No upgrade needed"
        exit 0
    fi

    check_database_integrity

    # Create backup
    create_upgrade_backup

    # Pull new version
    if ! pull_new_version; then
        log_error "Failed to pull new version"
        exit 1
    fi

    # Stop service
    stop_service

    # Start with new version
    start_service

    # Verify health
    if verify_health; then
        log_success "╔════════════════════════════════════════════════════════════════╗"
        log_success "║       Upgrade Successful!                                      ║"
        log_success "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        log_info "Upgraded from: ${CURRENT_VERSION}"
        log_info "Upgraded to: ${TARGET_VERSION}"
        log_info "Backup location: ${UPGRADE_BACKUP_DIR}"
        log_info "Health check: PASSED"
        echo ""
        log_info "Service is running at: ${HEALTH_URL}"
        echo ""
        exit 0
    else
        log_error "Health check failed after upgrade"
        
        if [[ "${ROLLBACK_ON_FAILURE}" == "true" ]]; then
            if rollback; then
                log_warning "Upgrade was rolled back successfully"
                exit 1
            else
                log_error "Rollback failed!"
                log_error "Manual recovery required"
                log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
                exit 2
            fi
        else
            log_error "Auto-rollback disabled, manual intervention required"
            log_error "To rollback manually, run: $0 rollback"
            log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
            exit 1
        fi
    fi
}

# ============================================
# Entry Point
# ============================================

main() {
    parse_args "$@"
    upgrade
}

main "$@"
