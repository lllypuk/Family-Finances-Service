#!/bin/bash
# Family Budget Service - Upgrade Script
# Safely upgrades the application with automatic backup and rollback

set -euo pipefail

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"
DATA_DIR="${DATA_DIR:-${INSTALL_DIR}/data}"
# Клон репозитория: публичного образа в GHCR нет (D-02), образ собирается на месте,
# поэтому «версия» — это git-ref в этом каталоге, а не тег образа.
SRC_DIR="${SRC_DIR:-${INSTALL_DIR}/src}"
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
# git-ref (тег, ветка или коммит), из которого пересобирается образ.
TARGET_VERSION="main"
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
    --version GIT_REF    Target git ref - tag, branch or commit (default: main).
                         The image is built from ${SRC_DIR}, there is no
                         published registry image to pull, so this is never an
                         image tag.
    --no-rollback        Disable automatic rollback on failure
    --help, -h           Show this help message

COMMANDS:
    rollback             Manually rollback to previous version

EXAMPLES:
    # Upgrade to the tip of the default branch
    sudo ./upgrade.sh

    # Upgrade to a specific git ref
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

# git >= 2.35.2 отказывается работать с репозиторием, принадлежащим другому
# пользователю ("dubious ownership"): install.sh отдаёт ${SRC_DIR} системному
# пользователю приложения, а этот скрипт работает из-под root.
ensure_git_safe_directory() {
    if ! git config --global --get-all safe.directory 2>/dev/null | grep -qxF "${SRC_DIR}"; then
        git config --global --add safe.directory "${SRC_DIR}"
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

    if [[ ! -d "${SRC_DIR}/.git" ]]; then
        log_error "Source checkout not found: ${SRC_DIR}"
        log_error "The image is built from source; re-run install.sh to create it"
        exit 1
    fi

    ensure_git_safe_directory

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

    # Версия — это коммит, из которого собран текущий образ.
    CURRENT_VERSION=$(git -C "${SRC_DIR}" rev-parse HEAD 2>/dev/null || echo "unknown")

    local human
    human=$(git -C "${SRC_DIR}" describe --tags --always 2>/dev/null || echo "${CURRENT_VERSION}")

    log_info "Current version: ${human} (${CURRENT_VERSION})"
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

# Обновить исходники и пересобрать образ.
# `docker compose pull` тут неприменим: сервис build-only (D-02).
build_new_version() {
    log_info "Fetching sources for: ${TARGET_VERSION}..."

    if ! git -C "${SRC_DIR}" fetch --tags --force origin "${TARGET_VERSION}"; then
        log_error "Failed to fetch ref: ${TARGET_VERSION}"
        return 1
    fi

    if ! git -C "${SRC_DIR}" checkout --force --detach FETCH_HEAD; then
        log_error "Failed to check out ref: ${TARGET_VERSION}"
        return 1
    fi

    log_info "Building image from $(git -C "${SRC_DIR}" rev-parse --short HEAD)..."

    cd "${INSTALL_DIR}"

    if docker compose -f "${COMPOSE_FILE}" build app; then
        log_success "Successfully built version: ${TARGET_VERSION}"
    else
        log_error "Failed to build new version"
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

# Откатить исходники на сохранённый коммит.
# install.sh клонирует с `--depth 1`, поэтому предыдущего коммита в истории
# может не быть вовсе — сначала углубляем историю, и только потом checkout.
restore_source_checkout() {
    local commit=$1

    if ! git -C "${SRC_DIR}" cat-file -e "${commit}^{commit}" 2>/dev/null; then
        log_info "Commit ${commit} is not in the local history, deepening the clone..."
        if [[ "$(git -C "${SRC_DIR}" rev-parse --is-shallow-repository 2>/dev/null)" == "true" ]]; then
            git -C "${SRC_DIR}" fetch --unshallow origin 2>/dev/null \
                || git -C "${SRC_DIR}" fetch --deepen=50 origin 2>/dev/null \
                || true
        fi
        # прицельная попытка: часть серверов отдаёт коммит по SHA
        git -C "${SRC_DIR}" fetch origin "${commit}" 2>/dev/null || true
    fi

    if ! git -C "${SRC_DIR}" cat-file -e "${commit}^{commit}" 2>/dev/null; then
        log_error "Commit ${commit} is still unreachable in ${SRC_DIR}"
        return 1
    fi

    git -C "${SRC_DIR}" checkout --force --detach "${commit}"
}

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

    # Restore previous version: откатываем исходники на сохранённый коммит и
    # пересобираем образ — готового образа для отката в реестре нет (D-02).
    local prev_version=$(cat "${UPGRADE_BACKUP_DIR}/version.txt" 2>/dev/null || echo "unknown")
    log_info "Restoring previous version: ${prev_version}"

    if [[ "${prev_version}" == "unknown" ]]; then
        log_error "Previous commit is unknown; restarting with the image that is already built"
        log_error "(that image is the FAILED upgrade — verify the service manually)"
    elif restore_source_checkout "${prev_version}"; then
        docker compose -f "${COMPOSE_FILE}" build app || log_error "Rebuild of previous version failed"
    else
        log_error "Could not restore sources at ${prev_version}; the image is NOT rebuilt"
        log_error "and still contains the failed upgrade. Restore manually from ${UPGRADE_BACKUP_DIR}"
    fi

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

    ensure_git_safe_directory

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

    # Skip if the requested ref is already checked out (полный SHA)
    if [[ "${CURRENT_VERSION}" == "${TARGET_VERSION}" ]]; then
        log_info "Already running ${TARGET_VERSION}"
        log_info "No upgrade needed"
        exit 0
    fi

    check_database_integrity

    # Create backup
    create_upgrade_backup

    # Fetch sources and rebuild the image
    if ! build_new_version; then
        log_error "Failed to build new version"
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
