#!/bin/bash
# Family Budget Service - uninstaller. Removes the compose deployment, optionally keeping data.

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
COMPOSE_FILE="${INSTALL_DIR}/docker-compose.yml"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/lib/common.sh" ]]; then
    # shellcheck source=./lib/common.sh
    source "${SCRIPT_DIR}/lib/common.sh"
else
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
    log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
    log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
    log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
    log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
fi

SKIP_CONFIRMATION=false
KEEP_DATA=false

# Имена заданы в compose жёстко, поэтому контейнеры находятся и без compose-файла.
CONTAINER_NAMES=(family-budget-app family-budget-caddy)

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --yes|-y)
                SKIP_CONFIRMATION=true
                shift
                ;;
            --keep-data)
                KEEP_DATA=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
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
Family Budget Service - Uninstall Script

Usage: sudo ./uninstall.sh [OPTIONS]

OPTIONS:
    --yes, -y      Skip confirmation prompts (removes everything, no backup copy)
    --keep-data    Keep ${INSTALL_DIR} (database, backups, .env) and the Caddy volumes
    --help, -h     Show this help message

ENVIRONMENT:
    INSTALL_DIR    Installation directory (default: /opt/family-budget)
EOF
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

confirm_uninstall() {
    if [[ "${SKIP_CONFIRMATION}" == "true" ]]; then
        return 0
    fi

    echo ""
    log_warning "This removes Family Budget Service from ${INSTALL_DIR}."
    read -r -p "Do you want to proceed with uninstall? (type 'yes' to confirm): " confirm
    if [[ "${confirm}" != "yes" ]]; then
        log_info "Uninstall cancelled by user"
        exit 0
    fi
}

backup_data() {
    if [[ "${KEEP_DATA}" == "true" || "${SKIP_CONFIRMATION}" == "true" ]]; then
        return 0
    fi

    read -r -p "Copy database, backups and .env somewhere safe first? (yes/no): " backup_confirm
    if [[ "${backup_confirm}" != "yes" ]]; then
        return 0
    fi

    local backup_dir
    backup_dir="${HOME}/family-budget-backup-$(date +%Y%m%d_%H%M%S)"
    log_info "Creating copy at ${backup_dir}..."
    mkdir -p "${backup_dir}"

    if [[ -f "${INSTALL_DIR}/data/budget.db" ]]; then
        cp "${INSTALL_DIR}/data/budget.db" "${backup_dir}/"
        # -wal/-shm остаются, если контейнер завершился нештатно: без них копия
        # теряет последние транзакции.
        local suffix
        for suffix in -wal -shm; do
            if [[ -f "${INSTALL_DIR}/data/budget.db${suffix}" ]]; then
                cp "${INSTALL_DIR}/data/budget.db${suffix}" "${backup_dir}/"
            fi
        done
    fi
    if [[ -f "${INSTALL_DIR}/.env" ]]; then
        cp "${INSTALL_DIR}/.env" "${backup_dir}/"
    fi
    if compgen -G "${INSTALL_DIR}/backups/*.db" > /dev/null; then
        mkdir -p "${backup_dir}/backups"
        cp "${INSTALL_DIR}"/backups/*.db "${backup_dir}/backups/"
    fi

    log_success "Copy created at: ${backup_dir}"
}

# Пропавший compose-файл не значит «ничего не запущено»: у повреждённой установки
# контейнеры продолжают писать в bind-mount базу, которую дальше удаляет rm -rf.
stop_containers_by_name() {
    local name existing
    for name in "${CONTAINER_NAMES[@]}"; do
        if ! existing="$(docker ps -aq --filter "name=^/${name}$")"; then
            log_error "docker ps failed; cannot tell whether ${name} is running"
            log_error "Nothing was removed. Check: docker ps -a --filter name=family-budget"
            exit 1
        fi
        if [[ -n "${existing}" ]]; then
            log_info "Removing container ${name}..."
            docker rm -f "${name}" >/dev/null || log_warning "Could not remove container ${name}"
        fi
    done

    ensure_nothing_running "docker ps --filter name=family-budget"

    log_warning "Named volumes (Caddy certificates) were not removed: no compose file to name them"
}

# Неудачный запрос к docker считается за «контейнеры живы»: цена ошибки в другую
# сторону — rm -rf по базе под работающим приложением.
ensure_nothing_running() {
    local hint="$1" still_running
    if ! still_running="$(running_containers)"; then
        log_error "Cannot query docker; refusing to remove ${INSTALL_DIR}"
        log_error "Check: ${hint}"
        exit 1
    fi
    if [[ -n "${still_running}" ]]; then
        log_error "Containers are still running; refusing to remove ${INSTALL_DIR}"
        log_error "Check: ${hint}"
        exit 1
    fi
}

running_containers() {
    local name filters=()
    for name in "${CONTAINER_NAMES[@]}"; do
        filters+=(--filter "name=^/${name}$")
    done
    docker ps -q "${filters[@]}"
}

stop_services() {
    if [[ ! -f "${COMPOSE_FILE}" ]]; then
        log_warning "No compose file at ${COMPOSE_FILE}; falling back to the fixed container names"
        stop_containers_by_name
        return 0
    fi

    log_info "Stopping services..."
    cd "${INSTALL_DIR}"

    # Caddy volumes hold the issued certificates: dropping them makes the next
    # install request new ones and burn Let's Encrypt rate limits for the domain.
    local down_args=(--remove-orphans)
    if [[ "${KEEP_DATA}" != "true" ]]; then
        down_args+=(--volumes)
    fi

    # Ошибку глушить нельзя: дальше идёт rm -rf ${INSTALL_DIR}, а удаление
    # bind-mount базы из-под живого контейнера уничтожает её без копии.
    if ! docker compose -f "${COMPOSE_FILE}" down "${down_args[@]}"; then
        log_error "docker compose down failed; containers may still be running"
        log_error "Nothing was removed. Check: docker compose -f ${COMPOSE_FILE} ps"
        exit 1
    fi

    local compose_ps
    if ! compose_ps="$(docker compose -f "${COMPOSE_FILE}" ps -q)"; then
        log_error "docker compose ps failed after 'down'; refusing to remove ${INSTALL_DIR}"
        log_error "Check: docker compose -f ${COMPOSE_FILE} ps"
        exit 1
    fi
    if [[ -n "${compose_ps}" ]]; then
        log_error "Containers are still running after 'down'; refusing to remove ${INSTALL_DIR}"
        log_error "Check: docker compose -f ${COMPOSE_FILE} ps"
        exit 1
    fi
    ensure_nothing_running "docker compose -f ${COMPOSE_FILE} ps"

    log_success "Services stopped"
}

remove_images() {
    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        read -r -p "Remove the locally built Docker image? (yes/no): " remove_img
        if [[ "${remove_img}" != "yes" ]]; then
            return 0
        fi
    fi

    # The image is built locally, so compose names it <project>-app, where the
    # project is the name of the installation directory.
    local project
    project="$(basename "${INSTALL_DIR}")"
    docker rmi "${project}-app" 2>/dev/null || true
    log_success "Docker image removed"
}

remove_installation_directory() {
    if [[ "${KEEP_DATA}" == "true" ]]; then
        log_info "Keeping ${INSTALL_DIR} (remove it later with: rm -rf ${INSTALL_DIR})"
        return 0
    fi

    if [[ -d "${INSTALL_DIR}" ]]; then
        rm -rf "${INSTALL_DIR}"
        log_success "Installation directory removed: ${INSTALL_DIR}"
    fi
}

remove_firewall_rules() {
    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        read -r -p "Remove firewall rules for 80/443? (yes/no): " remove_fw
        if [[ "${remove_fw}" != "yes" ]]; then
            return 0
        fi
    else
        return 0
    fi

    log_info "Removing firewall rules..."

    if command -v ufw &>/dev/null; then
        ufw delete allow 80/tcp 2>/dev/null || true
        ufw delete allow 443/tcp 2>/dev/null || true
        ufw delete allow 443/udp 2>/dev/null || true
    fi
    if command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --remove-service=http 2>/dev/null || true
        firewall-cmd --permanent --remove-service=https 2>/dev/null || true
        firewall-cmd --permanent --remove-port=443/udp 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
    fi

    log_success "Firewall rules removed"
}

main() {
    parse_args "$@"

    check_root
    confirm_uninstall
    # Сначала стоп, потом копия: приложение сливает WAL только при штатном
    # завершении, иначе копия -wal рядом с budget.db — единственный шанс на
    # последние транзакции.
    stop_services
    backup_data
    remove_images
    remove_installation_directory
    remove_firewall_rules

    echo ""
    log_success "Family Budget Service uninstalled"
    if [[ "${KEEP_DATA}" == "true" ]]; then
        log_info "Data preserved in: ${INSTALL_DIR}"
    fi
    log_warning "The host crontab entry for the daily backup is not touched: remove it with 'crontab -e'"
    echo ""
}

main "$@"
