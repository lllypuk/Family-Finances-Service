#!/bin/bash
# Firewall configuration functions

# shellcheck source=./common.sh
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Check if UFW is installed
check_ufw() {
    if command -v ufw &>/dev/null; then
        log_info "UFW is already installed"
        return 0
    fi
    return 1
}

# Install UFW
install_ufw() {
    if check_ufw; then
        return 0
    fi
    
    log_info "Installing UFW..."
    
    case "$OS" in
        ubuntu|debian)
            apt-get install -y ufw
            ;;
        rocky|almalinux)
            # RHEL-based systems use firewalld by default
            log_info "RHEL-based system detected, using firewalld instead of UFW"
            return 0
            ;;
        *)
            log_warning "Cannot install UFW on $OS"
            return 1
            ;;
    esac
    
    log_success "UFW installed"
}

# Configure UFW firewall
setup_ufw_firewall() {
    log_info "Configuring UFW firewall..."
    
    # Reset UFW to default
    ufw --force reset
    
    # Set default policies
    ufw default deny incoming
    ufw default allow outgoing
    
    # Allow SSH (prevent lockout)
    ufw allow 22/tcp comment 'SSH'
    log_info "Allowed SSH (22/tcp)"
    
    # Allow HTTP and HTTPS
    ufw allow 80/tcp comment 'HTTP'
    ufw allow 443/tcp comment 'HTTPS'
    log_info "Allowed HTTP (80/tcp) and HTTPS (443/tcp)"
    
    # Block direct access to application port from outside
    # (it will be accessed through reverse proxy only)
    ufw deny 8080/tcp comment 'Block direct access to app'
    log_info "Blocked direct access to port 8080"
    
    # Enable UFW
    ufw --force enable
    
    log_success "UFW firewall configured and enabled"
    
    # Show status
    ufw status numbered
}

# Configure firewalld (for RHEL-based systems)
setup_firewalld() {
    log_info "Configuring firewalld..."
    
    # Start and enable firewalld
    systemctl start firewalld
    systemctl enable firewalld
    
    # Allow HTTP and HTTPS
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    log_info "Allowed HTTP and HTTPS"
    
    # Block direct access to application port
    firewall-cmd --permanent --remove-port=8080/tcp 2>/dev/null || true
    log_info "Blocked direct access to port 8080"
    
    # Reload firewall
    firewall-cmd --reload
    
    log_success "Firewalld configured and enabled"
    
    # Show status
    firewall-cmd --list-all
}

# Setup firewall based on OS
setup_firewall() {
    case "$OS" in
        ubuntu|debian)
            install_ufw
            setup_ufw_firewall
            ;;
        rocky|almalinux)
            setup_firewalld
            ;;
        *)
            log_warning "Firewall setup not implemented for $OS"
            ;;
    esac
}
