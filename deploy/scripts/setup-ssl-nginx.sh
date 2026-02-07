#!/bin/bash
# Setup SSL certificate with Let's Encrypt using Nginx

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
Setup SSL Certificate for Family Budget Service (Nginx)

Usage: sudo ./setup-ssl-nginx.sh --domain DOMAIN --email EMAIL

OPTIONS:
    --domain DOMAIN    Domain name for SSL certificate
    --email EMAIL      Email for Let's Encrypt notifications
    --help, -h         Show this help message

EXAMPLE:
    sudo ./setup-ssl-nginx.sh --domain budget.example.com --email admin@example.com

EOF
}

# Generate DH parameters if not exists
generate_dhparam() {
    local dhparam_file="$INSTALL_DIR/nginx/dhparam.pem"
    
    if [[ -f "$dhparam_file" ]]; then
        log_info "DH parameters already exist"
        return
    fi
    
    log_info "Generating DH parameters (this may take several minutes)..."
    mkdir -p "$INSTALL_DIR/nginx"
    openssl dhparam -out "$dhparam_file" 4096
    chmod 644 "$dhparam_file"
    log_success "DH parameters generated"
}

# Request certificate
request_certificate() {
    log_info "Requesting SSL certificate for $DOMAIN..."
    
    cd "$INSTALL_DIR"
    
    # Create certbot directories
    mkdir -p certbot/{www,conf}
    
    # Request certificate
    docker compose run --rm certbot certonly \
        --webroot \
        --webroot-path=/var/www/certbot \
        --email "$EMAIL" \
        --agree-tos \
        --no-eff-email \
        -d "$DOMAIN"
    
    if [[ ! -d "$INSTALL_DIR/certbot/conf/live/$DOMAIN" ]]; then
        log_error "Certificate request failed"
        exit 1
    fi
    
    log_success "SSL certificate obtained"
}

# Update nginx configuration
update_nginx_config() {
    log_info "Updating Nginx configuration..."
    
    # Replace domain placeholder in nginx config
    local template="$INSTALL_DIR/nginx/conf.d/family-budget.conf.template"
    local config="$INSTALL_DIR/nginx/conf.d/family-budget.conf"
    
    sed "s/\${DOMAIN}/$DOMAIN/g" "$template" > "$config"
    
    log_success "Nginx configuration updated"
}

# Reload nginx
reload_nginx() {
    log_info "Reloading Nginx..."
    
    cd "$INSTALL_DIR"
    docker compose exec nginx nginx -s reload
    
    log_success "Nginx reloaded"
}

# Enable HSTS
enable_hsts() {
    log_info "To enable HSTS (recommended after confirming SSL works):"
    log_info "Edit $INSTALL_DIR/nginx/snippets/security-headers.conf"
    log_info "Uncomment the HSTS header line and reload nginx"
}

main() {
    parse_args "$@"
    
    if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
        log_error "Domain and email are required"
        show_help
        exit 1
    fi
    
    check_root
    
    log_info "Setting up SSL certificate for $DOMAIN"
    
    generate_dhparam
    request_certificate
    update_nginx_config
    reload_nginx
    
    log_success "SSL certificate setup complete!"
    log_info "Test your site at: https://$DOMAIN"
    enable_hsts
}

main "$@"
