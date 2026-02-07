#!/bin/bash
# Setup SSL certificate with Let's Encrypt using Caddy

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

DOMAIN=""
EMAIL=""
INSTALL_DIR="/opt/family-budget"

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --email)
                EMAIL="$2"
                shift 2
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
Setup SSL Certificate for Family Budget Service (Caddy)

Usage: sudo ./setup-ssl-caddy.sh --domain DOMAIN --email EMAIL

OPTIONS:
    --domain DOMAIN    Domain name for SSL certificate
    --email EMAIL      Email for Let's Encrypt notifications
    --help, -h         Show this help message

EXAMPLE:
    sudo ./setup-ssl-caddy.sh --domain budget.example.com --email admin@example.com

NOTE:
    Caddy automatically obtains and renews SSL certificates.
    This script just prepares the configuration.

EOF
}

# Update Caddy configuration
update_caddy_config() {
    log_info "Updating Caddy configuration..."
    
    local template="$INSTALL_DIR/caddy/Caddyfile.template"
    local config="$INSTALL_DIR/caddy/Caddyfile"
    
    # Replace placeholders
    sed -e "s/\${DOMAIN}/$DOMAIN/g" \
        -e "s/\${ACME_EMAIL}/$EMAIL/g" \
        "$template" > "$config"
    
    chmod 644 "$config"
    log_success "Caddy configuration updated"
}

# Update environment file
update_env_file() {
    log_info "Updating environment file..."
    
    local env_file="$INSTALL_DIR/config/.env"
    
    # Update or add DOMAIN and ACME_EMAIL
    if grep -q "^DOMAIN=" "$env_file"; then
        sed -i "s|^DOMAIN=.*|DOMAIN=$DOMAIN|" "$env_file"
    else
        echo "DOMAIN=$DOMAIN" >> "$env_file"
    fi
    
    if grep -q "^ACME_EMAIL=" "$env_file"; then
        sed -i "s|^ACME_EMAIL=.*|ACME_EMAIL=$EMAIL|" "$env_file"
    else
        echo "ACME_EMAIL=$EMAIL" >> "$env_file"
    fi
    
    log_success "Environment file updated"
}

# Restart Caddy
restart_caddy() {
    log_info "Restarting Caddy..."
    
    cd "$INSTALL_DIR"
    docker compose restart caddy
    
    # Wait for Caddy to start
    sleep 5
    
    log_success "Caddy restarted"
}

# Check certificate
check_certificate() {
    log_info "Waiting for Caddy to obtain certificate..."
    
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -sf "https://$DOMAIN/health" &>/dev/null; then
            log_success "SSL certificate obtained and working!"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_warning "Could not verify SSL certificate automatically"
    log_info "Check Caddy logs: docker compose -f $INSTALL_DIR/docker-compose.yml logs caddy"
}

main() {
    parse_args "$@"
    
    if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
        log_error "Domain and email are required"
        show_help
        exit 1
    fi
    
    check_root
    
    log_info "Setting up SSL certificate for $DOMAIN with Caddy"
    log_info "Caddy will automatically obtain certificates from Let's Encrypt"
    
    update_caddy_config
    update_env_file
    restart_caddy
    check_certificate
    
    log_success "SSL setup complete!"
    log_info "Test your site at: https://$DOMAIN"
    log_info "Caddy will automatically renew certificates"
}

main "$@"
