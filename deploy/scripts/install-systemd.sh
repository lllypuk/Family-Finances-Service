#!/bin/bash
# Family Budget Service - Systemd Installation Script
# This script installs the service for native deployment without Docker

set -euo pipefail

# Configuration
INSTALL_DIR="/opt/family-budget"
SERVICE_USER="familybudget"
SYSTEMD_DIR="/etc/systemd/system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "${SCRIPT_DIR}/lib/common.sh"

# Check if running as root
check_root

log_info "=== Family Budget Service - Systemd Installation ==="
echo ""

# Create service user if doesn't exist
create_user "${SERVICE_USER}"

# Create directory structure
log_info "Creating directory structure..."
mkdir -p "${INSTALL_DIR}"/{bin,config,data,backups,logs,scripts}
log_success "Directories created"

# Copy or download binary
if [[ -f "${SCRIPT_DIR}/../../build/family-budget-service" ]]; then
    log_info "Copying binary from build directory..."
    cp "${SCRIPT_DIR}/../../build/family-budget-service" "${INSTALL_DIR}/bin/"
    chmod 755 "${INSTALL_DIR}/bin/family-budget-service"
    log_success "Binary installed"
elif [[ -f "${SCRIPT_DIR}/../../server" ]]; then
    log_info "Copying binary from repository root..."
    cp "${SCRIPT_DIR}/../../server" "${INSTALL_DIR}/bin/family-budget-service"
    chmod 755 "${INSTALL_DIR}/bin/family-budget-service"
    log_success "Binary installed"
else
    log_warning "Binary not found in build directory or repository root"
    log_info "You'll need to manually copy the binary to ${INSTALL_DIR}/bin/family-budget-service"
fi

# Copy scripts
log_info "Installing scripts..."
cp "${SCRIPT_DIR}/backup.sh" "${INSTALL_DIR}/scripts/"
cp "${SCRIPT_DIR}/health-check.sh" "${INSTALL_DIR}/scripts/"
chmod 755 "${INSTALL_DIR}/scripts/"*.sh
log_success "Scripts installed"

# Create default config if not exists
if [[ ! -f "${INSTALL_DIR}/config/.env" ]]; then
    log_info "Creating default configuration..."
    
    SESSION_SECRET=$(generate_secret)
    CSRF_SECRET=$(generate_secret)
    
    cat > "${INSTALL_DIR}/config/.env" <<EOF
# Family Budget Service Configuration
# Generated on $(date)

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=127.0.0.1

# Database
DATABASE_PATH=/opt/family-budget/data/budget.db

# Security
SESSION_SECRET=${SESSION_SECRET}
CSRF_SECRET=${CSRF_SECRET}

# Logging
LOG_LEVEL=info
ENVIRONMENT=production

# Backup Configuration
BACKUP_DIR=/opt/family-budget/backups
DATA_DIR=/opt/family-budget/data
BACKUP_RETENTION_DAYS=30
MAX_BACKUPS=50

# Health Check
HEALTH_URL=http://127.0.0.1:8080/health
HEALTH_TIMEOUT=5
EOF
    chmod 600 "${INSTALL_DIR}/config/.env"
    log_success "Configuration created with secure secrets"
else
    log_info "Configuration file already exists, skipping"
fi

# Set permissions
log_info "Setting permissions..."
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"
chmod 700 "${INSTALL_DIR}/data"
chmod 700 "${INSTALL_DIR}/backups"
chmod 700 "${INSTALL_DIR}/config"
chmod 755 "${INSTALL_DIR}/bin"
chmod 755 "${INSTALL_DIR}/scripts"
chmod 755 "${INSTALL_DIR}/logs"
log_success "Permissions set"

# Install systemd units
log_info "Installing systemd service units..."
cp "${SCRIPT_DIR}/../systemd/family-budget.service" "${SYSTEMD_DIR}/"
cp "${SCRIPT_DIR}/../systemd/family-budget-backup.service" "${SYSTEMD_DIR}/"
cp "${SCRIPT_DIR}/../systemd/family-budget-backup.timer" "${SYSTEMD_DIR}/"
log_success "Systemd units installed"

# Reload systemd
log_info "Reloading systemd daemon..."
systemctl daemon-reload
log_success "Systemd reloaded"

# Enable services
log_info "Enabling services..."
systemctl enable family-budget.service
systemctl enable family-budget-backup.timer
log_success "Services enabled"

# Completion message
echo ""
log_success "╔════════════════════════════════════════════════════════════════╗"
log_success "║       Family Budget Service Installation Complete!            ║"
log_success "╚════════════════════════════════════════════════════════════════╝"
echo ""
log_info "Installation directory: ${INSTALL_DIR}"
log_info "Configuration file: ${INSTALL_DIR}/config/.env"
log_info "Service user: ${SERVICE_USER}"
echo ""
log_info "Next steps:"
log_info "  1. Review configuration: ${INSTALL_DIR}/config/.env"
log_info "  2. Start service: systemctl start family-budget"
log_info "  3. Start backup timer: systemctl start family-budget-backup.timer"
log_info "  4. Check status: systemctl status family-budget"
log_info "  5. View logs: journalctl -u family-budget -f"
echo ""
log_info "Service will be available at: http://127.0.0.1:8080"
log_info "Health check: curl http://127.0.0.1:8080/health"
echo ""
log_warning "Don't forget to set up a reverse proxy (Nginx/Caddy) for HTTPS!"
echo ""
