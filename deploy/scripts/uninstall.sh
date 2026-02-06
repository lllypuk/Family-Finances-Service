#!/bin/bash
# Family Budget Service - Uninstall Script
# Safely removes Family Budget Service with optional data preservation

set -euo pipefail

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
SERVICE_USER="familybudget"
SYSTEMD_DIR="/etc/systemd/system"

# Source common functions if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/lib/common.sh" ]]; then
    source "${SCRIPT_DIR}/lib/common.sh"
else
    # Define basic logging functions
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
    log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
    log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
    log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
    log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
fi

# Command line options
SKIP_CONFIRMATION=false
KEEP_DATA=false
REMOVE_USER=true

# ============================================
# Parse arguments
# ============================================

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
            --keep-user)
                REMOVE_USER=false
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
    --yes, -y         Skip confirmation prompts
    --keep-data       Keep database and backup files
    --keep-user       Don't remove the service user
    --help, -h        Show this help message

EXAMPLES:
    # Interactive uninstall (recommended)
    sudo ./uninstall.sh

    # Non-interactive with data backup
    sudo ./uninstall.sh --yes --keep-data

    # Complete removal
    sudo ./uninstall.sh --yes

EOF
}

# ============================================
# Pre-uninstall checks
# ============================================

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
    log_info "╔════════════════════════════════════════════════════════════════╗"
    log_info "║       Family Budget Service - Uninstaller                     ║"
    log_info "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_warning "This will remove Family Budget Service from your system."
    echo ""
    log_info "Installation directory: ${INSTALL_DIR}"
    log_info "Service user: ${SERVICE_USER}"
    echo ""

    read -p "Do you want to proceed with uninstall? (type 'yes' to confirm): " confirm
    if [[ "${confirm}" != "yes" ]]; then
        log_info "Uninstall cancelled by user"
        exit 0
    fi
}

# ============================================
# Data handling
# ============================================

backup_data() {
    if [[ "${KEEP_DATA}" == "true" ]]; then
        log_info "Data will be preserved in ${INSTALL_DIR}"
        return 0
    fi

    local backup_dir="${HOME}/family-budget-backup-$(date +%Y%m%d_%H%M%S)"

    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        echo ""
        read -p "Create backup before uninstalling? (yes/no): " backup_confirm
    else
        backup_confirm="no"
    fi

    if [[ "${backup_confirm}" == "yes" ]]; then
        log_info "Creating backup at ${backup_dir}..."
        mkdir -p "${backup_dir}"

        # Backup database
        if [[ -f "${INSTALL_DIR}/data/budget.db" ]]; then
            mkdir -p "${backup_dir}/data"
            cp "${INSTALL_DIR}/data/budget.db" "${backup_dir}/data/"
            log_success "Database backed up"
        fi

        # Backup configuration
        if [[ -f "${INSTALL_DIR}/.env" ]]; then
            cp "${INSTALL_DIR}/.env" "${backup_dir}/"
        fi
        if [[ -f "${INSTALL_DIR}/config/.env" ]]; then
            mkdir -p "${backup_dir}/config"
            cp "${INSTALL_DIR}/config/.env" "${backup_dir}/config/"
        fi
        log_success "Configuration backed up"

        # Backup existing backups
        if [[ -d "${INSTALL_DIR}/backups" ]] && ls "${INSTALL_DIR}/backups"/*.db &>/dev/null; then
            mkdir -p "${backup_dir}/backups"
            cp "${INSTALL_DIR}/backups"/*.db "${backup_dir}/backups/" 2>/dev/null || true
            log_success "Existing backups preserved"
        fi

        log_success "Backup created at: ${backup_dir}"
    fi
}

# ============================================
# Stop services
# ============================================

stop_docker_services() {
    log_info "Stopping Docker services..."

    # Try various compose file names
    local compose_files=(
        "docker-compose.yml"
        "docker-compose.prod.yml"
        "docker-compose.nginx.yml"
        "docker-compose.caddy.yml"
    )

    cd "${INSTALL_DIR}" 2>/dev/null || return 0

    for compose_file in "${compose_files[@]}"; do
        if [[ -f "${compose_file}" ]]; then
            log_info "Stopping services from ${compose_file}..."
            docker compose -f "${compose_file}" down --remove-orphans 2>/dev/null || true
        fi
    done

    # Remove Docker volumes if requested
    if [[ "${SKIP_CONFIRMATION}" == "false" && "${KEEP_DATA}" == "false" ]]; then
        echo ""
        read -p "Remove Docker volumes? This will delete all data in volumes. (yes/no): " remove_volumes
    else
        remove_volumes="no"
    fi

    if [[ "${remove_volumes}" == "yes" || "${KEEP_DATA}" == "false" ]]; then
        log_info "Removing Docker volumes..."
        docker volume rm family-budget-data 2>/dev/null || true
        docker volume rm family-budget-backups 2>/dev/null || true
        docker volume rm family-budget-caddy-data 2>/dev/null || true
        docker volume rm family-budget-caddy-config 2>/dev/null || true
        docker volume rm family-budget-certbot-www 2>/dev/null || true
        docker volume rm family-budget-certbot-conf 2>/dev/null || true
        docker volume rm family-budget-nginx-cache 2>/dev/null || true
        log_success "Docker volumes removed"
    fi

    # Remove Docker images if requested
    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        read -p "Remove Docker images? (yes/no): " remove_images
        if [[ "${remove_images}" == "yes" ]]; then
            docker rmi ghcr.io/lllypuk/family-finances-service:latest 2>/dev/null || true
            docker rmi $(docker images -q 'ghcr.io/lllypuk/family-finances-service:*') 2>/dev/null || true
            log_success "Docker images removed"
        fi
    fi
}

stop_systemd_services() {
    log_info "Stopping systemd services..."

    # Stop and disable main service
    if systemctl is-active family-budget.service &>/dev/null; then
        systemctl stop family-budget.service
        log_info "Stopped family-budget.service"
    fi
    if systemctl is-enabled family-budget.service &>/dev/null; then
        systemctl disable family-budget.service
        log_info "Disabled family-budget.service"
    fi

    # Stop and disable backup service
    if systemctl is-active family-budget-backup.service &>/dev/null; then
        systemctl stop family-budget-backup.service
        log_info "Stopped family-budget-backup.service"
    fi

    # Stop and disable backup timer
    if systemctl is-active family-budget-backup.timer &>/dev/null; then
        systemctl stop family-budget-backup.timer
        log_info "Stopped family-budget-backup.timer"
    fi
    if systemctl is-enabled family-budget-backup.timer &>/dev/null; then
        systemctl disable family-budget-backup.timer
        log_info "Disabled family-budget-backup.timer"
    fi
}

# ============================================
# Remove files
# ============================================

remove_systemd_units() {
    log_info "Removing systemd units..."

    rm -f "${SYSTEMD_DIR}/family-budget.service"
    rm -f "${SYSTEMD_DIR}/family-budget-backup.service"
    rm -f "${SYSTEMD_DIR}/family-budget-backup.timer"

    systemctl daemon-reload

    log_success "Systemd units removed"
}

remove_installation_directory() {
    if [[ "${KEEP_DATA}" == "true" ]]; then
        log_info "Keeping installation directory: ${INSTALL_DIR}"
        log_info "You can manually remove it later with: rm -rf ${INSTALL_DIR}"
        return 0
    fi

    log_info "Removing installation directory..."

    if [[ -d "${INSTALL_DIR}" ]]; then
        rm -rf "${INSTALL_DIR}"
        log_success "Installation directory removed: ${INSTALL_DIR}"
    else
        log_info "Installation directory not found, skipping"
    fi
}

remove_service_user() {
    if [[ "${REMOVE_USER}" == "false" ]]; then
        log_info "Keeping service user: ${SERVICE_USER}"
        return 0
    fi

    log_info "Removing service user..."

    if id "${SERVICE_USER}" &>/dev/null; then
        userdel "${SERVICE_USER}" 2>/dev/null || true
        log_success "Service user removed: ${SERVICE_USER}"
    else
        log_info "Service user not found, skipping"
    fi
}

remove_firewall_rules() {
    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        echo ""
        read -p "Remove firewall rules? (yes/no): " remove_fw
    else
        remove_fw="no"
    fi

    if [[ "${remove_fw}" != "yes" ]]; then
        return 0
    fi

    log_info "Removing firewall rules..."

    # UFW (Ubuntu/Debian)
    if command -v ufw &>/dev/null; then
        ufw delete allow 80/tcp 2>/dev/null || true
        ufw delete allow 443/tcp 2>/dev/null || true
        ufw delete deny 8080/tcp 2>/dev/null || true
        log_success "UFW rules removed"
    fi

    # Firewalld (RHEL-based)
    if command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --remove-service=http 2>/dev/null || true
        firewall-cmd --permanent --remove-service=https 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
        log_success "Firewalld rules removed"
    fi
}

remove_fail2ban_config() {
    if [[ "${SKIP_CONFIRMATION}" == "false" ]]; then
        echo ""
        read -p "Remove fail2ban configuration? (yes/no): " remove_f2b
    else
        remove_f2b="no"
    fi

    if [[ "${remove_f2b}" != "yes" ]]; then
        return 0
    fi

    log_info "Removing fail2ban configuration..."

    rm -f /etc/fail2ban/filter.d/family-budget.conf
    rm -f /etc/fail2ban/jail.d/family-budget.local

    if systemctl is-active fail2ban &>/dev/null; then
        systemctl restart fail2ban
    fi

    log_success "Fail2ban configuration removed"
}

# ============================================
# Main uninstall function
# ============================================

uninstall() {
    echo ""
    log_info "Starting uninstall process..."
    echo ""

    check_root
    confirm_uninstall
    backup_data

    stop_docker_services
    stop_systemd_services
    remove_systemd_units
    remove_installation_directory
    remove_service_user
    remove_firewall_rules
    remove_fail2ban_config

    echo ""
    log_success "╔════════════════════════════════════════════════════════════════╗"
    log_success "║       Family Budget Service Uninstalled Successfully!         ║"
    log_success "╚════════════════════════════════════════════════════════════════╝"
    echo ""

    if [[ "${KEEP_DATA}" == "true" ]]; then
        log_info "Data preserved in: ${INSTALL_DIR}"
    fi

    log_info "Thank you for using Family Budget Service!"
    echo ""
}

# ============================================
# Entry point
# ============================================

main() {
    parse_args "$@"
    uninstall
}

main "$@"
