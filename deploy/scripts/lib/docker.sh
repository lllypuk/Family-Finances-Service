#!/bin/bash
# Docker installation functions

# shellcheck source=./common.sh
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Check if Docker is installed
check_docker() {
    if command -v docker &>/dev/null; then
        local docker_version=$(docker --version | awk '{print $3}' | sed 's/,//')
        log_info "Docker is already installed: $docker_version"
        return 0
    fi
    return 1
}

# Install Docker on Ubuntu/Debian
install_docker_debian() {
    log_info "Installing Docker on Debian/Ubuntu..."
    
    # Update package index
    apt-get update
    
    # Install prerequisites
    apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release
    
    # Add Docker's official GPG key
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/$OS/gpg | \
        gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    
    # Set up repository
    echo \
        "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$OS \
        $(lsb_release -cs) stable" | \
        tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Install Docker Engine
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    log_success "Docker installed successfully"
}

# Install Docker on Rocky/AlmaLinux
install_docker_rhel() {
    log_info "Installing Docker on Rocky/AlmaLinux..."
    
    # Install prerequisites
    yum install -y yum-utils
    
    # Add Docker repository
    yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    
    # Install Docker Engine
    yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # Start and enable Docker
    systemctl start docker
    systemctl enable docker
    
    log_success "Docker installed successfully"
}

# Install Docker based on OS
install_docker() {
    if check_docker; then
        return 0
    fi
    
    case "$OS" in
        ubuntu|debian)
            install_docker_debian
            ;;
        rocky|almalinux)
            install_docker_rhel
            ;;
        *)
            log_error "Unsupported OS for Docker installation: $OS"
            exit 1
            ;;
    esac
    
    # Start and enable Docker
    systemctl start docker
    systemctl enable docker
    
    # Verify installation
    if ! docker --version &>/dev/null; then
        log_error "Docker installation failed"
        exit 1
    fi
    
    # Verify Docker Compose
    if ! docker compose version &>/dev/null; then
        log_error "Docker Compose installation failed"
        exit 1
    fi
    
    local docker_version=$(docker --version)
    local compose_version=$(docker compose version)
    log_success "Docker installed: $docker_version"
    log_success "Docker Compose installed: $compose_version"
}

# Add user to docker group
add_user_to_docker_group() {
    local username=$1
    usermod -aG docker "$username" || true
    log_info "Added user $username to docker group"
}
