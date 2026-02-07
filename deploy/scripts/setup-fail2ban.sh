#!/bin/bash
# Family Budget Service - Fail2ban Setup Script

set -euo pipefail

# Source common functions if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/lib/common.sh" ]]; then
    source "${SCRIPT_DIR}/lib/common.sh"
else
    log_info() { echo "[INFO] $1"; }
    log_success() { echo "[SUCCESS] $1"; }
    log_warning() { echo "[WARNING] $1"; }
    log_error() { echo "[ERROR] $1"; }
fi

# Configuration
FAIL2BAN_DIR="/etc/fail2ban"
INSTALL_DIR="/opt/family-budget"

# Check root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

log_info "=== Family Budget Service - Fail2ban Setup ==="
echo ""

# Detect OS
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS=$ID
else
    log_error "Cannot detect OS"
    exit 1
fi

# Install fail2ban
if ! command -v fail2ban-client &>/dev/null; then
    log_info "Installing fail2ban..."
    
    case "$OS" in
        ubuntu|debian)
            apt-get update
            apt-get install -y fail2ban
            ;;
        rocky|almalinux|centos)
            yum install -y epel-release
            yum install -y fail2ban fail2ban-systemd
            ;;
        *)
            log_error "Unsupported OS for automatic fail2ban installation: $OS"
            exit 1
            ;;
    esac
    
    log_success "Fail2ban installed"
else
    log_info "Fail2ban is already installed"
fi

# Install filter
log_info "Installing Family Budget fail2ban filter..."
if [[ -f "${INSTALL_DIR}/fail2ban/family-budget.conf" ]] || [[ -f "${SCRIPT_DIR}/../fail2ban/family-budget.conf" ]]; then
    # Try to find the filter file
    FILTER_SOURCE=""
    if [[ -f "${INSTALL_DIR}/fail2ban/family-budget.conf" ]]; then
        FILTER_SOURCE="${INSTALL_DIR}/fail2ban/family-budget.conf"
    elif [[ -f "${SCRIPT_DIR}/../fail2ban/family-budget.conf" ]]; then
        FILTER_SOURCE="${SCRIPT_DIR}/../fail2ban/family-budget.conf"
    fi
    
    if [[ -n "${FILTER_SOURCE}" ]]; then
        cp "${FILTER_SOURCE}" "${FAIL2BAN_DIR}/filter.d/family-budget.conf"
        chmod 644 "${FAIL2BAN_DIR}/filter.d/family-budget.conf"
        log_success "Filter installed"
    fi
else
    log_warning "Filter file not found, skipping"
fi

# Install jail configuration
log_info "Installing jail configuration..."
if [[ -f "${INSTALL_DIR}/fail2ban/jail.local" ]] || [[ -f "${SCRIPT_DIR}/../fail2ban/jail.local" ]]; then
    JAIL_SOURCE=""
    if [[ -f "${INSTALL_DIR}/fail2ban/jail.local" ]]; then
        JAIL_SOURCE="${INSTALL_DIR}/fail2ban/jail.local"
    elif [[ -f "${SCRIPT_DIR}/../fail2ban/jail.local" ]]; then
        JAIL_SOURCE="${SCRIPT_DIR}/../fail2ban/jail.local"
    fi
    
    if [[ -n "${JAIL_SOURCE}" ]]; then
        cp "${JAIL_SOURCE}" "${FAIL2BAN_DIR}/jail.d/family-budget.local"
        chmod 644 "${FAIL2BAN_DIR}/jail.d/family-budget.local"
        log_success "Jail configuration installed"
    fi
else
    log_warning "Jail configuration file not found, skipping"
fi

# Enable and start fail2ban
log_info "Enabling fail2ban service..."
systemctl enable fail2ban

log_info "Restarting fail2ban..."
systemctl restart fail2ban

# Wait for fail2ban to start
sleep 3

# Show status
log_info "Fail2ban status:"
fail2ban-client status

# Try to show Family Budget jail status
log_info "Family Budget jail status:"
if fail2ban-client status family-budget 2>/dev/null; then
    log_success "Family Budget jail is active"
else
    log_warning "Family Budget jail not active yet (may need log files to exist)"
    log_info "Jail will activate automatically when application receives traffic"
fi

echo ""
log_success "╔════════════════════════════════════════════════════════════════╗"
log_success "║       Fail2ban Setup Complete!                                 ║"
log_success "╚════════════════════════════════════════════════════════════════╝"
echo ""
log_info "Configuration:"
log_info "  - Filter: ${FAIL2BAN_DIR}/filter.d/family-budget.conf"
log_info "  - Jail: ${FAIL2BAN_DIR}/jail.d/family-budget.local"
echo ""
log_info "Monitoring commands:"
log_info "  - Status: fail2ban-client status family-budget"
log_info "  - Banned IPs: fail2ban-client get family-budget banned"
log_info "  - Unban IP: fail2ban-client set family-budget unbanip <IP>"
echo ""
log_info "Jail settings:"
log_info "  - Max retries: 5 failed attempts"
log_info "  - Find time: 300 seconds (5 minutes)"
log_info "  - Ban time: 3600 seconds (1 hour)"
echo ""
