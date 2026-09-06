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

# Порты sshd читаем из конфига: на нестандартном порту жёстко зашитый 22
# запер бы оператора снаружи, а консоли у домашнего мини-сервера может не быть.
detect_ssh_ports() {
    local ports=()
    mapfile -t ports < <(sed -nE 's/^[[:space:]]*Port[[:space:]]+([0-9]+).*/\1/p' \
        /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null | sort -u)
    if [[ ${#ports[@]} -eq 0 ]]; then
        ports=(22)
    fi
    printf '%s\n' "${ports[@]}"
}

# Configure UFW firewall.
# Правила не сбрасываются: `ufw --force reset` снёс бы разрешения, выданные
# оператором для всего остального на этом хосте, а нужные нам allow идемпотентны.
setup_ufw_firewall() {
    log_info "Configuring UFW firewall..."

    # Set default policies
    ufw default deny incoming
    ufw default allow outgoing

    local ssh_ports=()
    mapfile -t ssh_ports < <(detect_ssh_ports)

    local port
    for port in "${ssh_ports[@]}"; do
        ufw allow "${port}/tcp" comment 'SSH'
        log_info "Allowed SSH (${port}/tcp)"
    done
    
    # Allow HTTP and HTTPS
    ufw allow 80/tcp comment 'HTTP'
    ufw allow 443/tcp comment 'HTTPS'
    # 443/udp — HTTP/3: этот порт compose публикует наравне с TCP.
    ufw allow 443/udp comment 'HTTPS (HTTP/3)'
    log_info "Allowed HTTP (80/tcp) and HTTPS (443/tcp, 443/udp)"
    
    # Enable UFW
    ufw --force enable
    
    log_success "UFW firewall configured and enabled"
    
    # Show status
    ufw status numbered
}

# Configure firewalld (for RHEL-based systems)
setup_firewalld() {
    log_info "Configuring firewalld..."

    local ssh_ports=()
    mapfile -t ssh_ports < <(detect_ssh_ports)

    local port
    # Зона public по умолчанию пускает только ssh/22, поэтому на нестандартном порту старт
    # демона оборвал бы текущую сессию: правила для sshd пишем ДО `systemctl start`.
    if ! systemctl is-active --quiet firewalld && command -v firewall-offline-cmd &>/dev/null; then
        for port in "${ssh_ports[@]}"; do
            firewall-offline-cmd --add-port="${port}/tcp" >/dev/null
        done
    fi

    # Start and enable firewalld
    systemctl start firewalld
    systemctl enable firewalld

    for port in "${ssh_ports[@]}"; do
        firewall-cmd --permanent --add-port="${port}/tcp"
        log_info "Allowed SSH (${port}/tcp)"
    done

    # Allow HTTP and HTTPS
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    # HTTP/3: службы https в firewalld только TCP, порт публикуется и по UDP.
    firewall-cmd --permanent --add-port=443/udp
    log_info "Allowed HTTP and HTTPS (incl. 443/udp for HTTP/3)"

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
