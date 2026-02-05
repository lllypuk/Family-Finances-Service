# Task 001: Create Installation Script

## Overview

Create a comprehensive installation script (`deploy/scripts/install.sh`) that automates the deployment of Family Budget
Service on a fresh Linux VM.

## Priority: HIGH

## Status: TODO

## Requirements

### Supported Operating Systems

- Ubuntu 22.04 LTS / 24.04 LTS
- Debian 11 / 12
- Rocky Linux 9 / AlmaLinux 9

### Script Capabilities

The script must handle:

1. **Pre-flight Checks**
    - Root/sudo privileges verification
    - Minimum system requirements (2GB RAM, 10GB disk)
    - Network connectivity check
    - Port availability (80, 443, 8080)

2. **Dependency Installation**
    - Docker Engine (latest stable)
    - Docker Compose v2
    - curl, wget, openssl
    - UFW firewall (if not present)

3. **Security Setup**
    - Generate cryptographically secure SESSION_SECRET (32 bytes)
    - Generate CSRF_SECRET
    - Create dedicated system user `familybudget`
    - Set proper file permissions (700 for data, 600 for configs)

4. **Application Deployment**
    - Create directory structure:
      ```
      /opt/family-budget/
      ├── data/           # SQLite database
      ├── backups/        # Database backups
      ├── config/         # Environment files
      └── logs/           # Application logs
      ```
    - Download docker-compose.prod.yml
    - Create .env file from template
    - Pull Docker images
    - Start services

5. **Firewall Configuration**
    - Allow SSH (22)
    - Allow HTTP (80)
    - Allow HTTPS (443)
    - Block direct access to 8080 (internal only)

6. **Post-Installation**
    - Verify health endpoint responds
    - Display access URL and credentials info
    - Show next steps (SSL setup, first user creation)

## Script Structure

```bash
#!/bin/bash
set -euo pipefail

# Constants
INSTALL_DIR="/opt/family-budget"
APP_USER="familybudget"
REPO_URL="https://github.com/lllypuk/Family-Finances-Service"

# Functions
log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1" >&2; }
log_success() { echo "[SUCCESS] $1"; }

check_root() { ... }
check_system_requirements() { ... }
install_docker() { ... }
setup_firewall() { ... }
generate_secrets() { ... }
create_directories() { ... }
deploy_application() { ... }
verify_installation() { ... }

# Main
main() {
    log_info "Starting Family Budget Service installation..."
    check_root
    check_system_requirements
    install_docker
    setup_firewall
    generate_secrets
    create_directories
    deploy_application
    verify_installation
    log_success "Installation complete!"
}

main "$@"
```

## User Interaction

The script should support two modes:

### Interactive Mode (default)

```bash
sudo ./install.sh
```

- Prompts for domain name
- Prompts for admin email (for Let's Encrypt)
- Confirms before making changes

### Non-Interactive Mode

```bash
sudo ./install.sh --non-interactive \
  --domain budget.example.com \
  --email admin@example.com
```

## Error Handling

- All commands must check exit codes
- Rollback on failure (remove partial installation)
- Clear error messages with remediation steps
- Log all actions to `/var/log/family-budget-install.log`

## Testing Checklist

- [ ] Fresh Ubuntu 22.04 VM
- [ ] Fresh Ubuntu 24.04 VM
- [ ] Fresh Debian 12 VM
- [ ] VM with existing Docker installation
- [ ] VM behind NAT (no public IP)
- [ ] Re-run on already installed system (idempotent)

## Acceptance Criteria

1. Script runs without errors on supported OS
2. Application accessible on port 8080 after installation
3. Firewall rules correctly configured
4. Secrets are unique and secure
5. Data directory has correct permissions
6. Health check passes
7. Logs capture all installation steps

## Dependencies

- Task 002 (nginx config) - for full HTTPS setup
- Task 003 (docker-compose.prod.yml) - production compose file

## Estimated Complexity

Medium-High (requires testing on multiple OS)

## Files to Create

- `deploy/scripts/install.sh`
- `deploy/scripts/lib/common.sh` (shared functions)
- `deploy/scripts/lib/docker.sh` (Docker installation)
- `deploy/scripts/lib/firewall.sh` (UFW configuration)
