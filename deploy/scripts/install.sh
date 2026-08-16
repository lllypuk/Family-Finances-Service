#!/bin/bash
# Family Budget Service - Installation Script
# This script automates the deployment of Family Budget Service on a fresh Linux VM

set -euo pipefail

# Constants
INSTALL_DIR="/opt/family-budget"
SRC_DIR="${INSTALL_DIR}/src"
APP_USER="familybudget"
REPO_URL="https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main"
REPO_GIT_URL="${REPO_GIT_URL:-https://github.com/lllypuk/Family-Finances-Service.git}"
REPO_REF="${REPO_REF:-main}"
LOG_FILE="/var/log/family-budget-install.log"

# UID/GID пользователя внутри образа (docker/Dockerfile: user `app`).
# Bind-mount ./data должен принадлежать именно ему, иначе SQLite не откроет БД (D-03).
CONTAINER_UID=1000
CONTAINER_GID=1000

# Source library functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/docker.sh"
source "${SCRIPT_DIR}/lib/firewall.sh"

# Global variables
DOMAIN=""
ADMIN_EMAIL=""
NON_INTERACTIVE=false
SESSION_SECRET=""
CSRF_SECRET=""

# Logging setup
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --non-interactive)
                NON_INTERACTIVE=true
                shift
                ;;
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --email)
                ADMIN_EMAIL="$2"
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

# Show help message
show_help() {
    cat <<EOF
Family Budget Service - Installation Script

Usage: sudo ./install.sh [OPTIONS]

OPTIONS:
    --non-interactive       Run without user prompts
    --domain DOMAIN        Set domain name (e.g., budget.example.com)
    --email EMAIL          Set admin email for Let's Encrypt
    --help, -h             Show this help message

EXAMPLES:
    # Interactive installation
    sudo ./install.sh

    # Non-interactive installation
    sudo ./install.sh --non-interactive --domain budget.example.com --email admin@example.com

ENVIRONMENT:
    REPO_GIT_URL           Repository to build from (default: upstream GitHub URL)
    REPO_REF               Branch or tag to check out (default: main)

REQUIREMENTS:
    - Ubuntu 22.04/24.04, Debian 11/12, or Rocky Linux 9
    - Minimum 512MB RAM (the service itself needs ~128-256MB;
      the rest is headroom for the Docker image build)
    - Minimum 10GB disk space
    - Root or sudo privileges
    - Outbound network access (the image is built from source, see D-02)

EOF
}

# Prompt for configuration
prompt_configuration() {
    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        if [[ -z "$DOMAIN" ]]; then
            DOMAIN="localhost"
        fi
        if [[ -z "$ADMIN_EMAIL" ]]; then
            ADMIN_EMAIL="admin@localhost"
        fi
        return
    fi
    
    echo ""
    log_info "=== Configuration ==="
    echo ""
    
    prompt_input "Enter domain name (or 'localhost' for local testing)" "localhost" DOMAIN
    
    if [[ "$DOMAIN" != "localhost" ]]; then
        prompt_input "Enter admin email for Let's Encrypt SSL" "admin@$DOMAIN" ADMIN_EMAIL
    else
        ADMIN_EMAIL="admin@localhost"
    fi
    
    echo ""
    log_info "Configuration summary:"
    log_info "  Domain: $DOMAIN"
    log_info "  Admin email: $ADMIN_EMAIL"
    log_info "  Install directory: $INSTALL_DIR"
    echo ""
    
    if ! confirm_action "Proceed with installation?"; then
        log_info "Installation cancelled by user"
        exit 0
    fi
}

# Generate secrets
generate_secrets() {
    log_info "Generating secure secrets..."
    
    SESSION_SECRET=$(generate_secret)
    CSRF_SECRET=$(generate_secret)
    
    log_success "Generated SESSION_SECRET and CSRF_SECRET"
}

# Create directory structure
create_directories() {
    log_info "Creating directory structure..."
    
    # Backup existing installation if present
    backup_directory "$INSTALL_DIR"
    
    # Create main directory
    mkdir -p "$INSTALL_DIR"/{data,backups,config,logs}
    
    log_success "Created directory structure at $INSTALL_DIR"
}

# Fetch application sources
#
# Публичного образа в GHCR нет (D-02), поэтому образ собирается на месте.
# Для этого рядом с compose-файлом нужны исходники: $INSTALL_DIR/src.
fetch_sources() {
    log_info "Fetching application sources into $SRC_DIR..."

    install_git

    if [[ -d "$SRC_DIR/.git" ]]; then
        log_info "Existing checkout found, updating..."
        git -C "$SRC_DIR" fetch --depth 1 origin "$REPO_REF"
        git -C "$SRC_DIR" checkout --detach FETCH_HEAD
    else
        rm -rf "$SRC_DIR"
        git clone --depth 1 --branch "$REPO_REF" "$REPO_GIT_URL" "$SRC_DIR"
    fi

    log_success "Sources ready at $SRC_DIR ($(git -C "$SRC_DIR" rev-parse --short HEAD))"
}

# Download deployment files
download_deployment_files() {
    log_info "Preparing deployment files..."

    cd "$INSTALL_DIR"

    # docker-compose.prod.yml — сначала из свежего checkout, затем из локального
    # репозитория (когда скрипт запущен из клона), иначе встроенный fallback.
    local compose_src=""
    if [[ -f "$SRC_DIR/deploy/docker-compose.prod.yml" ]]; then
        compose_src="$SRC_DIR/deploy/docker-compose.prod.yml"
    elif [[ -f "$SCRIPT_DIR/../docker-compose.prod.yml" ]]; then
        compose_src="$SCRIPT_DIR/../docker-compose.prod.yml"
    fi

    if [[ -n "$compose_src" ]]; then
        cp "$compose_src" "$INSTALL_DIR/docker-compose.yml"
        log_info "Copied docker-compose.prod.yml from $compose_src"
    else
        log_warning "docker-compose.prod.yml not found, using minimal configuration"
        cat > "$INSTALL_DIR/docker-compose.yml" <<'EOF'
version: '3.8'

services:
  app:
    build:
      context: ${BUILD_CONTEXT:-./src}
      dockerfile: docker/Dockerfile
    container_name: family-budget
    restart: unless-stopped
    ports:
      - "8080:8080"
    user: "1000:1000"
    environment:
      - SERVER_PORT=8080
      - SERVER_HOST=0.0.0.0
      - DATABASE_PATH=/data/budget.db
      - SESSION_SECRET=${SESSION_SECRET:?SESSION_SECRET is required}
      - CSRF_SECRET=${CSRF_SECRET:?CSRF_SECRET is required}
      - LOG_LEVEL=info
      - ENVIRONMENT=production
    volumes:
      - ./data:/data
      - ./backups:/backups
      - ./logs:/logs
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "-O", "/dev/null", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
EOF
    fi
    
    log_success "Deployment files ready"
}

# Create environment file
create_env_file() {
    log_info "Creating environment file..."
    
    cat > "$INSTALL_DIR/config/.env" <<EOF
# Family Budget Service - Production Configuration
# Generated on $(date)

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DOMAIN=$DOMAIN

# Database
DATABASE_PATH=/data/budget.db

# Build (образ собирается из исходников, D-02).
# Путь к клону репозитория относительно $INSTALL_DIR.
BUILD_CONTEXT=./src

# Security
SESSION_SECRET=$SESSION_SECRET
CSRF_SECRET=$CSRF_SECRET

# Logging
LOG_LEVEL=info
ENVIRONMENT=production

# Admin Contact
ADMIN_EMAIL=$ADMIN_EMAIL
EOF
    
    chmod 600 "$INSTALL_DIR/config/.env"
    log_success "Created environment file"
}

# Set permissions
set_file_permissions() {
    log_info "Setting file permissions..."
    
    # Create app user if doesn't exist
    create_user "$APP_USER"
    
    # Set ownership
    chown -R "$APP_USER:$APP_USER" "$INSTALL_DIR"

    # Каталоги, смонтированные в контейнер, должны принадлежать пользователю
    # внутри образа (docker/Dockerfile: app = 1000:1000), иначе SQLite не
    # откроет БД на bind-mount (D-03). $APP_USER — системный, его UID < 1000.
    chown -R "$CONTAINER_UID:$CONTAINER_GID" \
        "$INSTALL_DIR/data" "$INSTALL_DIR/backups" "$INSTALL_DIR/logs"

    # Set directory permissions
    chmod 755 "$INSTALL_DIR"
    chmod 700 "$INSTALL_DIR/data"
    chmod 700 "$INSTALL_DIR/backups"
    chmod 700 "$INSTALL_DIR/config"
    chmod 755 "$INSTALL_DIR/logs"

    log_success "File permissions set"
}

# Deploy application
deploy_application() {
    log_info "Deploying application..."
    
    cd "$INSTALL_DIR"
    
    # Create environment symlink for docker-compose
    ln -sf config/.env .env

    # Собираем образ из исходников в ./src — публичного образа нет (D-02),
    # поэтому `docker compose pull` здесь не сработает.
    log_info "Building Docker image from sources (this may take a few minutes)..."
    docker compose build app

    # Start services
    log_info "Starting services..."
    docker compose up -d

    log_success "Application deployed"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."
    
    # Wait for application to start
    sleep 5
    
    # Check if container is running
    if ! docker ps | grep -q family-budget; then
        log_error "Container is not running"
        docker compose logs
        exit 1
    fi
    log_success "Container is running"
    
    # Check health endpoint
    local max_attempts=30
    local attempt=0
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -sf http://localhost:8080/health &>/dev/null; then
            log_success "Health check passed"
            break
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    if [[ $attempt -eq $max_attempts ]]; then
        log_error "Health check failed after $max_attempts attempts"
        docker compose logs
        exit 1
    fi
}

# Display completion message
show_completion_message() {
    echo ""
    log_success "╔════════════════════════════════════════════════════════════════╗"
    log_success "║       Family Budget Service Installation Complete!            ║"
    log_success "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Installation directory: $INSTALL_DIR"
    log_info "Application URL: http://$DOMAIN:8080"
    echo ""
    log_info "Next steps:"
    log_info "  1. Set up reverse proxy with SSL (see deploy/README.md, 'Deployment Options')"
    log_info "  2. Create your first admin user by accessing the application"
    log_info "  3. Review logs: docker compose -f $INSTALL_DIR/docker-compose.yml logs -f"
    log_info "  4. Check status: docker compose -f $INSTALL_DIR/docker-compose.yml ps"
    echo ""
    log_info "Sources (image is built from them): $SRC_DIR"
    log_info "Database location: $INSTALL_DIR/data/budget.db"
    log_info "Backups location: $INSTALL_DIR/backups/"
    log_info "Installation log: $LOG_FILE"
    echo ""
    
    if [[ "$DOMAIN" == "localhost" ]]; then
        log_warning "You are using localhost. For production use, configure a proper domain and SSL."
    fi
    
    echo ""
}

# Cleanup on error
cleanup_on_error() {
    log_error "Installation failed. Cleaning up..."
    
    if [[ -d "$INSTALL_DIR" ]] && docker compose -f "$INSTALL_DIR/docker-compose.yml" ps &>/dev/null; then
        docker compose -f "$INSTALL_DIR/docker-compose.yml" down
    fi
    
    log_info "Check the log file for details: $LOG_FILE"
}

# Main installation flow
main() {
    trap cleanup_on_error ERR
    
    echo ""
    log_info "╔════════════════════════════════════════════════════════════════╗"
    log_info "║       Family Budget Service - Installation Script             ║"
    log_info "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    
    parse_args "$@"
    
    check_root
    detect_os
    check_system_requirements
    check_ports
    prompt_configuration
    
    install_docker
    setup_firewall
    generate_secrets
    create_directories
    fetch_sources
    download_deployment_files
    create_env_file
    set_file_permissions
    deploy_application
    verify_installation
    
    show_completion_message
}

# Run main function
main "$@"
