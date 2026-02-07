#!/bin/bash
# Common utility functions for deployment scripts

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Check if running as root or with sudo
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
    log_success "Running with root privileges"
}

# Check system requirements
check_system_requirements() {
    log_info "Checking system requirements..."
    
    # Check RAM (minimum 2GB)
    local total_ram=$(free -m | awk '/^Mem:/{print $2}')
    if [[ $total_ram -lt 2000 ]]; then
        log_error "Insufficient RAM: ${total_ram}MB (minimum 2GB required)"
        exit 1
    fi
    log_success "RAM check passed: ${total_ram}MB"
    
    # Check disk space (minimum 10GB)
    local available_space=$(df -BG / | awk 'NR==2 {print $4}' | sed 's/G//')
    if [[ $available_space -lt 10 ]]; then
        log_error "Insufficient disk space: ${available_space}GB (minimum 10GB required)"
        exit 1
    fi
    log_success "Disk space check passed: ${available_space}GB available"
    
    # Check network connectivity
    if ! ping -c 1 -W 5 8.8.8.8 &>/dev/null; then
        log_error "No internet connectivity"
        exit 1
    fi
    log_success "Network connectivity check passed"
}

# Check if port is available
check_port() {
    local port=$1
    if ss -tuln | grep -q ":${port} "; then
        log_warning "Port ${port} is already in use"
        return 1
    fi
    return 0
}

# Check all required ports
check_ports() {
    log_info "Checking port availability..."
    
    local ports=(80 443 8080)
    local ports_in_use=()
    
    for port in "${ports[@]}"; do
        if ! check_port "$port"; then
            ports_in_use+=("$port")
        fi
    done
    
    if [[ ${#ports_in_use[@]} -gt 0 ]]; then
        log_warning "The following ports are in use: ${ports_in_use[*]}"
        log_warning "This may interfere with the installation"
        if [[ "${NON_INTERACTIVE:-false}" != "true" ]]; then
            read -p "Continue anyway? (y/N) " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        fi
    else
        log_success "All required ports are available"
    fi
}

# Detect OS
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log_info "Detected OS: $PRETTY_NAME"
    else
        log_error "Cannot detect OS"
        exit 1
    fi
    
    # Verify supported OS
    case "$OS" in
        ubuntu)
            if [[ ! "$OS_VERSION" =~ ^(22.04|24.04) ]]; then
                log_error "Unsupported Ubuntu version: $OS_VERSION (supported: 22.04, 24.04)"
                exit 1
            fi
            ;;
        debian)
            if [[ ! "$OS_VERSION" =~ ^(11|12) ]]; then
                log_error "Unsupported Debian version: $OS_VERSION (supported: 11, 12)"
                exit 1
            fi
            ;;
        rocky|almalinux)
            if [[ ! "$OS_VERSION" =~ ^9 ]]; then
                log_error "Unsupported $OS version: $OS_VERSION (supported: 9)"
                exit 1
            fi
            ;;
        *)
            log_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    log_success "OS is supported"
}

# Generate secure random secret
generate_secret() {
    openssl rand -base64 32
}

# Create user if doesn't exist
create_user() {
    local username=$1
    if id "$username" &>/dev/null; then
        log_info "User $username already exists"
    else
        useradd --system --no-create-home --shell /bin/false "$username"
        log_success "Created system user: $username"
    fi
}

# Set file ownership and permissions
set_permissions() {
    local path=$1
    local owner=$2
    local perms=$3
    
    chown -R "$owner:$owner" "$path"
    chmod "$perms" "$path"
}

# Backup existing directory
backup_directory() {
    local dir=$1
    if [[ -d "$dir" ]]; then
        local backup_name="${dir}.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Backing up existing directory to: $backup_name"
        mv "$dir" "$backup_name"
        log_success "Backup created"
    fi
}

# Prompt for input with default value
prompt_input() {
    local prompt=$1
    local default=$2
    local var_name=$3
    
    if [[ "${NON_INTERACTIVE:-false}" == "true" ]]; then
        eval "$var_name=\"$default\""
        return
    fi
    
    read -p "$prompt [$default]: " input
    if [[ -z "$input" ]]; then
        eval "$var_name=\"$default\""
    else
        eval "$var_name=\"$input\""
    fi
}

# Confirm action
confirm_action() {
    local message=$1
    
    if [[ "${NON_INTERACTIVE:-false}" == "true" ]]; then
        return 0
    fi
    
    read -p "$message (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}
