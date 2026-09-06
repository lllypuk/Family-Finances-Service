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

# Configure UFW firewall.
# Правила не сбрасываются: `ufw --force reset` снёс бы разрешения, выданные
# оператором для всего остального на этом хосте, а нужные нам allow идемпотентны.
setup_ufw_firewall() {
    log_info "Configuring UFW firewall..."

    # Set default policies
    ufw default deny incoming
    ufw default allow outgoing

    # Порт sshd читаем из конфига: на нестандартном порту жёстко зашитый 22
    # запер бы оператора снаружи, а консоли у домашнего мини-сервера может не быть.
    local ssh_ports=()
    mapfile -t ssh_ports < <(sed -nE 's/^[[:space:]]*Port[[:space:]]+([0-9]+).*/\1/p' \
        /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null | sort -u)
    if [[ ${#ssh_ports[@]} -eq 0 ]]; then
        ssh_ports=(22)
    fi

    local port
    for port in "${ssh_ports[@]}"; do
        ufw allow "${port}/tcp" comment 'SSH'
        log_info "Allowed SSH (${port}/tcp)"
    done
    
    # Allow HTTP and HTTPS
    ufw allow 80/tcp comment 'HTTP'
    ufw allow 443/tcp comment 'HTTPS'
    log_info "Allowed HTTP (80/tcp) and HTTPS (443/tcp)"
    
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
