# Task 008: Uninstall Script

## Overview

Create a clean uninstall script that safely removes Family Budget Service while preserving user data if requested.

## Priority: LOW

## Status: COMPLETE

## Completed Items

- [x] Created `deploy/scripts/uninstall.sh` - Comprehensive uninstall script with:
  
  **Pre-uninstall Features:**
  - Root privileges verification
  - Interactive confirmation (can be skipped with `--yes`)
  - Optional data backup before removal
  - Multiple removal modes

  **Service Cleanup:**
  - Stop and remove Docker containers
  - Stop and disable systemd services
  - Remove systemd unit files
  - Reload systemd daemon

  **Data Handling:**
  - Optional backup creation before uninstall
  - Database preservation option (`--keep-data`)
  - Backup directory with timestamp
  - Docker volume removal (optional)
  - Docker image removal (optional)

  **System Cleanup:**
  - Remove installation directory
  - Remove service user (optional with `--keep-user`)
  - Remove firewall rules (optional)
  - Remove fail2ban configuration (optional)

  **User Interface:**
  - Color-coded output
  - Progress indicators
  - Interactive prompts with defaults
  - Non-interactive mode support
  - Clear status messages

## Implementation Features

✅ **Safety First:**
- Confirmation required before any destructive action
- Optional data backup
- Keeps user informed of every step
- Can preserve data for migration

✅ **Flexibility:**
- Interactive mode (default): asks for confirmation on each step
- Non-interactive mode (`--yes`): automated uninstall
- Keep data option (`--keep-data`): preserve files for later
- Keep user option (`--keep-user`): don't remove system user
- Selective cleanup: firewall, fail2ban, volumes, images

✅ **Comprehensive:**
- Handles both Docker and systemd installations
- Cleans up all created resources
- Removes firewall rules
- Removes fail2ban configuration
- Provides clear feedback

✅ **Backup Support:**
- Creates timestamped backup directory
- Backs up database
- Backs up configuration files
- Backs up existing backup files
- Backup location: `~/family-budget-backup-YYYYMMDD_HHMMSS/`

## Usage Examples

### Interactive Uninstall (Recommended):
```bash
sudo ./deploy/scripts/uninstall.sh
```

### Quick Uninstall (keep data):
```bash
sudo ./deploy/scripts/uninstall.sh --yes --keep-data
```

### Complete Removal:
```bash
sudo ./deploy/scripts/uninstall.sh --yes
```

### Uninstall but keep service user:
```bash
sudo ./deploy/scripts/uninstall.sh --keep-user
```

## What Gets Removed

✅ Docker Services:
- All running containers
- Optional: Docker volumes
- Optional: Docker images
- Optional: Docker networks

✅ Systemd Services:
- family-budget.service
- family-budget-backup.service
- family-budget-backup.timer

✅ Files and Directories:
- /opt/family-budget/ (unless --keep-data)
- /etc/systemd/system/family-budget*

✅ System Resources:
- Service user (unless --keep-user)
- Firewall rules (optional)
- Fail2ban configuration (optional)

## Remaining Items

## Requirements

### Uninstall Script (`deploy/scripts/uninstall.sh`)

```bash
#!/bin/bash
set -euo pipefail

# ============================================
# Family Budget Service - Uninstall Script
# ============================================

INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
SERVICE_USER="familybudget"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ============================================
# Pre-uninstall
# ============================================

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

confirm_uninstall() {
    echo ""
    echo "========================================"
    echo "Family Budget Service Uninstaller"
    echo "========================================"
    echo ""
    log_warn "This will remove Family Budget Service from your system."
    echo ""
    echo "Installation directory: ${INSTALL_DIR}"
    echo ""

    read -p "Do you want to proceed? (yes/no): " confirm
    if [[ "${confirm}" != "yes" ]]; then
        log_info "Uninstall cancelled."
        exit 0
    fi
}

# ============================================
# Data handling
# ============================================

backup_data() {
    local backup_dir="${HOME}/family-budget-backup-$(date +%Y%m%d_%H%M%S)"

    echo ""
    read -p "Do you want to backup your data before uninstalling? (yes/no): " backup_confirm

    if [[ "${backup_confirm}" == "yes" ]]; then
        log_info "Creating backup at ${backup_dir}..."
        mkdir -p "${backup_dir}"

        # Backup database
        if [[ -f "${INSTALL_DIR}/data/budget.db" ]]; then
            cp "${INSTALL_DIR}/data/budget.db" "${backup_dir}/"
            log_info "Database backed up"
        fi

        # Backup configuration
        if [[ -f "${INSTALL_DIR}/.env" ]]; then
            cp "${INSTALL_DIR}/.env" "${backup_dir}/"
            log_info "Configuration backed up"
        fi

        # Backup existing backups
        if [[ -d "${INSTALL_DIR}/backups" ]] && ls "${INSTALL_DIR}/backups"/*.db &>/dev/null; then
            cp "${INSTALL_DIR}/backups"/*.db "${backup_dir}/"
            log_info "Existing backups preserved"
        fi

        log_info "Backup created at: ${backup_dir}"
    fi
}

# ============================================
# Stop services
# ============================================

stop_docker_services() {
    log_info "Stopping Docker services..."

    if [[ -f "${INSTALL_DIR}/docker-compose.yml" ]]; then
        cd "${INSTALL_DIR}"
        docker-compose down --remove-orphans 2>/dev/null || true
    fi

    if [[ -f "${INSTALL_DIR}/docker-compose.prod.yml" ]]; then
        cd "${INSTALL_DIR}"
        docker-compose -f docker-compose.prod.yml down --remove-orphans 2>/dev/null || true
    fi

    # Remove Docker volumes
    read -p "Remove Docker volumes (this deletes all data in volumes)? (yes/no): " remove_volumes
    if [[ "${remove_volumes}" == "yes" ]]; then
        docker volume rm family-budget-data 2>/dev/null || true
        docker volume rm family-budget-backups 2>/dev/null || true
        docker volume rm family-budget-caddy-data 2>/dev/null || true
        docker volume rm family-budget-caddy-config 2>/dev/null || true
        docker volume rm family-budget-certbot-www 2>/dev/null || true
        docker volume rm family-budget-certbot-conf 2>/dev/null || true
        docker volume rm family-budget-nginx-cache 2>/dev/null || true
        log_info "Docker volumes removed"
    fi
}

stop_systemd_services() {
    log_info "Stopping systemd services..."

    # Stop and disable main service
    if systemctl is-active family-budget &>/dev/null; then
        systemctl stop family-budget
    fi
    if systemctl is-enabled family-budget &>/dev/null; then
        systemctl disable family-budget
    fi

    # Stop and disable backup timer
    if systemctl is-active family-budget-backup.timer &>/dev/null; then
        systemctl stop family-budget-backup.timer
    fi
    if systemctl is-enabled family-budget-backup.timer &>/dev/null; then
        systemctl disable family-budget-backup.timer
    fi
}

# ============================================
# Remove files
# ============================================

remove_systemd_units() {
    log_info "Removing systemd units..."

    rm -f /etc/systemd/system/family-budget.service
    rm -f /etc/systemd/system/family-budget-backup.service
    rm -f /etc/systemd/system/family-budget-backup.timer

    systemctl daemon-reload
}

remove_fail2ban() {
    log_info "Removing fail2ban configuration..."

    rm -f /etc/fail2ban/filter.d/family-budget.conf
    rm -f /etc/fail2ban/jail.d/family-budget.local

    if systemctl is-active fail2ban &>/dev/null; then
        systemctl restart fail2ban
    fi
}

remove_nginx_config() {
    log_info "Removing nginx configuration..."

    rm -f /etc/nginx/sites-enabled/family-budget
    rm -f /etc/nginx/sites-available/family-budget
    rm -f /etc/nginx/conf.d/family-budget.conf

    if systemctl is-active nginx &>/dev/null; then
        nginx -t && systemctl reload nginx
    fi
}

remove_caddy_config() {
    log_info "Checking Caddy configuration..."
    log_warn "Please manually remove Family Budget entries from /etc/caddy/Caddyfile if needed"
}

remove_user() {
    log_info "Removing service user..."

    if id "${SERVICE_USER}" &>/dev/null; then
        read -p "Remove system user '${SERVICE_USER}'? (yes/no): " remove_user_confirm
        if [[ "${remove_user_confirm}" == "yes" ]]; then
            userdel "${SERVICE_USER}" 2>/dev/null || true
            log_info "User removed"
        fi
    fi
}

remove_installation_directory() {
    log_info "Removing installation directory..."

    if [[ -d "${INSTALL_DIR}" ]]; then
        read -p "Remove ${INSTALL_DIR} and ALL its contents? (yes/no): " remove_dir_confirm
        if [[ "${remove_dir_confirm}" == "yes" ]]; then
            rm -rf "${INSTALL_DIR}"
            log_info "Installation directory removed"
        else
            log_info "Installation directory preserved at ${INSTALL_DIR}"
        fi
    fi
}

# ============================================
# Clean up
# ============================================

cleanup_logs() {
    log_info "Cleaning up logs..."

    # Remove log files
    rm -f /var/log/family-budget*.log

    # Clear journald logs for the service
    read -p "Clear journald logs for Family Budget? (yes/no): " clear_logs
    if [[ "${clear_logs}" == "yes" ]]; then
        journalctl --rotate 2>/dev/null || true
        journalctl --vacuum-time=1s -u family-budget 2>/dev/null || true
    fi
}

# ============================================
# Main
# ============================================

main() {
    check_root
    confirm_uninstall
    backup_data

    echo ""
    log_info "Starting uninstallation..."
    echo ""

    stop_docker_services
    stop_systemd_services
    remove_systemd_units
    remove_fail2ban
    remove_nginx_config
    remove_caddy_config
    cleanup_logs
    remove_user
    remove_installation_directory

    echo ""
    echo "========================================"
    log_info "Uninstallation complete!"
    echo "========================================"
    echo ""

    if [[ -d "${HOME}/family-budget-backup-"* ]] 2>/dev/null; then
        log_info "Your data backup is located at: ~/family-budget-backup-*"
    fi

    echo ""
    log_info "Thank you for using Family Budget Service!"
    echo ""
}

main "$@"
```

---

## Features

### Data Preservation

- Prompts user to create backup before uninstall
- Preserves backups separately from installation
- Never deletes data without explicit confirmation

### Service Cleanup

- Stops Docker containers gracefully
- Removes Docker volumes (with confirmation)
- Stops and disables systemd services
- Removes systemd unit files

### Configuration Cleanup

- Removes nginx site configuration
- Removes fail2ban rules
- Warns about Caddy manual cleanup
- Removes firewall rules (optional)

### User Interaction

- Interactive prompts for destructive actions
- Clear warnings before data deletion
- Summary of what will be removed

---

## Usage

```bash
# Standard uninstall (interactive)
sudo /opt/family-budget/scripts/uninstall.sh

# Or download and run
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/scripts/uninstall.sh | sudo bash
```

---

## Files to Create

```
deploy/scripts/
└── uninstall.sh
```

## Testing Checklist

- [ ] Docker installation cleanly removed
- [ ] Native installation cleanly removed
- [ ] Data backup created correctly
- [ ] Docker volumes removed when requested
- [ ] Systemd services properly disabled
- [ ] No orphaned files after uninstall
- [ ] Backup is usable for re-installation

## Acceptance Criteria

1. No data loss without user confirmation
2. All services stopped before removal
3. Clean system state after uninstall
4. Backup created if requested
5. Clear feedback during process

## Dependencies

None

## Estimated Complexity

Low
