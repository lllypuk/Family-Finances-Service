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
# --reinstall: явное согласие снести существующую установку (каталог целиком
# уезжает в $INSTALL_DIR.backup.<ts>). Без флага повторный запуск сохраняет
# данные и секреты — см. create_directories/generate_secrets.
REINSTALL=false
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
            --reinstall)
                REINSTALL=true
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
    --reinstall            DESTRUCTIVE. Move the existing $INSTALL_DIR aside
                           (database, backups and config included) and install
                           from scratch. Without this flag a repeated run keeps
                           data/, backups/ and the secrets in config/.env.
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
    - Minimum 512MB RAM (the service itself needs ~128-256MB; the rest is headroom
      for the Docker image build). 1GB is recommended: below that the local
      `go build` inside the image build may be OOM-killed — add swap if you stay
      at 512MB
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

# Read a KEY=value entry out of the existing config/.env, if it is there.
read_env_value() {
    local key=$1
    local env_file="$INSTALL_DIR/config/.env"

    if [[ ! -f "$env_file" ]]; then
        return 0
    fi

    sed -n "s/^${key}=//p" "$env_file" | tail -1
}

# Generate secrets.
#
# Повторный запуск НЕ перевыпускает секреты, если они уже есть в config/.env:
# новый SESSION_SECRET разлогинил бы всех и превратил бы уже выданные cookie в
# «битые» (см. ensureCSRFToken). Свежие значения генерируются только когда
# файла нет — то есть при первой установке или после --reinstall.
#
# Вызывается ПОСЛЕ create_directories: только там становится понятно, остался
# ли config/.env на месте.
generate_secrets() {
    SESSION_SECRET=$(read_env_value SESSION_SECRET)
    CSRF_SECRET=$(read_env_value CSRF_SECRET)

    if [[ -n "$SESSION_SECRET" && -n "$CSRF_SECRET" ]]; then
        log_info "Reusing SESSION_SECRET and CSRF_SECRET from $INSTALL_DIR/config/.env"
        return 0
    fi

    log_info "Generating secure secrets..."

    if [[ -z "$SESSION_SECRET" ]]; then
        SESSION_SECRET=$(generate_secret)
    fi
    if [[ -z "$CSRF_SECRET" ]]; then
        CSRF_SECRET=$(generate_secret)
    fi

    log_success "Generated SESSION_SECRET and CSRF_SECRET"
}

# Create directory structure.
#
# Раньше здесь безусловно вызывался `backup_directory "$INSTALL_DIR"` (обычный
# `mv`): второй запуск install.sh на живой установке уносил в сторону боевую
# базу, бэкапы и config/.env, после чего сервис поднимался на пустом ./data с
# новыми секретами — то есть выглядел как полная потеря данных. Теперь снос
# требует явного --reinstall, а обычный повторный запуск идемпотентен.
create_directories() {
    log_info "Creating directory structure..."

    if [[ -d "$INSTALL_DIR" ]]; then
        if [[ "$REINSTALL" == "true" ]]; then
            log_warning "--reinstall: moving the existing installation aside"
            # Контейнер держит открытым файл БД на bind-mount: сначала гасим.
            if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
                (cd "$INSTALL_DIR" && docker compose down 2>/dev/null) || true
            fi
            backup_directory "$INSTALL_DIR"
        else
            log_info "Existing installation found at $INSTALL_DIR - keeping data/, backups/ and config/.env"
            log_info "Pass --reinstall to wipe it instead (the old tree is moved to ${INSTALL_DIR}.backup.<timestamp>)"
        fi
    fi

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

    # git >= 2.35.2 отказывается работать с репозиторием, принадлежащим другому
    # пользователю ("dubious ownership"). set_file_permissions отдаёт $SRC_DIR
    # пользователю $APP_USER, а install.sh/upgrade.sh работают из-под root.
    if ! git config --global --get-all safe.directory 2>/dev/null | grep -qxF "$SRC_DIR"; then
        git config --global --add safe.directory "$SRC_DIR"
    fi

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

    # Единственный источник compose-файла — свежий checkout в $SRC_DIR
    # (fetch_sources отработал выше). Встроенной копии здесь нет намеренно:
    # она неизбежно расходится с deploy/docker-compose.prod.yml.
    local compose_src="$SRC_DIR/deploy/docker-compose.prod.yml"

    if [[ ! -f "$compose_src" ]]; then
        log_error "Compose file not found: $compose_src"
        log_error "The checkout at $SRC_DIR is incomplete (wrong REPO_GIT_URL/REPO_REF?)"
        exit 1
    fi

    cp "$compose_src" "$INSTALL_DIR/docker-compose.yml"
    log_info "Copied docker-compose.prod.yml from $compose_src"

    log_success "Deployment files ready"
}

# Дописать KEY=value в config/.env, если такого ключа там ещё нет.
# Существующие значения не трогаем: их правит оператор.
ensure_env_key() {
    local key=$1
    local value=$2
    local env_file="$INSTALL_DIR/config/.env"

    if grep -q "^${key}=" "$env_file"; then
        return 0
    fi

    printf '%s=%s\n' "$key" "$value" >> "$env_file"
    log_info "Added missing $key to config/.env"
}

# Create environment file.
#
# Повторный запуск НЕ перезаписывает существующий config/.env. Раньше здесь был
# безусловный `cat >`, и второй прогон сбрасывал правки оператора: COOKIE_SECURE
# возвращался в true (на плайн-HTTP браузер молча выбрасывает Secure-cookie, и
# вход зацикливается на "CSRF token not found in session"), а DOMAIN, LOG_LEVEL и
# ENVIRONMENT — к значениям промпта, которые в --non-interactive равны localhost.
# Файл переписывается только при первой установке и при --reinstall; в остальных
# случаях лишь дописываются ключи, появившиеся в новых версиях.
create_env_file() {
    local env_file="$INSTALL_DIR/config/.env"

    if [[ -f "$env_file" && "$REINSTALL" != "true" ]]; then
        log_info "Keeping existing environment file $env_file"
        log_info "Operator settings (COOKIE_SECURE, DOMAIN, LOG_LEVEL, ENVIRONMENT, ...) are preserved"

        ensure_env_key SERVER_PORT 8080
        ensure_env_key SERVER_HOST 0.0.0.0
        ensure_env_key DOMAIN "$DOMAIN"
        ensure_env_key DATABASE_PATH /data/budget.db
        ensure_env_key BUILD_CONTEXT ./src
        ensure_env_key SESSION_SECRET "$SESSION_SECRET"
        ensure_env_key CSRF_SECRET "$CSRF_SECRET"
        ensure_env_key LOG_LEVEL info
        ensure_env_key ENVIRONMENT production
        ensure_env_key COOKIE_SECURE true
        ensure_env_key ADMIN_EMAIL "$ADMIN_EMAIL"

        chmod 600 "$env_file"
        log_success "Environment file is up to date"
        return 0
    fi

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

# Флаг Secure на cookie сессии. За TLS-прокси оставить true.
# Если сервис пока отдаётся по http:// (кроме localhost/SSH-туннеля), поставьте
# false: браузер молча выбрасывает Secure-cookie на небезопасном origin, и вход
# зацикливается на "CSRF token not found in session".
COOKIE_SECURE=true

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
    # Порядок важен: create_directories решает, остаётся ли config/.env на
    # месте, а generate_secrets переиспользует секреты именно оттуда.
    create_directories
    generate_secrets
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
