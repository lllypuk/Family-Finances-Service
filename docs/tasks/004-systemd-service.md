# Task 004: Systemd Service Files

## Overview

Create systemd service and timer files for running Family Budget Service natively on Linux without Docker, plus
automated backup scheduling.

## Priority: MEDIUM

## Status: TODO

## Requirements

### Use Cases

1. **Native deployment** - Running directly on VM without Docker
2. **Backup automation** - Scheduled database backups via systemd timer
3. **System integration** - Proper startup/shutdown with OS

---

### Main Service (`deploy/systemd/family-budget.service`)

```ini
[Unit]
Description=Family Budget Service
Documentation=https://github.com/lllypuk/Family-Finances-Service
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=familybudget
Group=familybudget

# Working directory
WorkingDirectory=/opt/family-budget

# Environment file
EnvironmentFile=/opt/family-budget/config/.env

# Main executable
ExecStart=/opt/family-budget/bin/family-budget-service

# Graceful shutdown
ExecStop=/bin/kill -TERM $MAINPID
TimeoutStopSec=30

# Restart policy
Restart=on-failure
RestartSec=5
StartLimitBurst=3
StartLimitIntervalSec=60

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/family-budget/data /opt/family-budget/backups /opt/family-budget/logs
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
MemoryDenyWriteExecute=yes
LockPersonality=yes

# Resource limits
MemoryMax=512M
CPUQuota=100%

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=family-budget

[Install]
WantedBy=multi-user.target
```

---

### Backup Service (`deploy/systemd/family-budget-backup.service`)

```ini
[Unit]
Description=Family Budget Database Backup
Documentation=https://github.com/lllypuk/Family-Finances-Service
After=family-budget.service
Requires=family-budget.service

[Service]
Type=oneshot
User=familybudget
Group=familybudget

# Working directory
WorkingDirectory=/opt/family-budget

# Environment
EnvironmentFile=/opt/family-budget/config/.env

# Backup script
ExecStart=/opt/family-budget/scripts/backup.sh

# Security (same as main service)
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/family-budget/data /opt/family-budget/backups
ProtectKernelTunables=yes
ProtectKernelModules=yes

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=family-budget-backup
```

---

### Backup Timer (`deploy/systemd/family-budget-backup.timer`)

```ini
[Unit]
Description=Daily Family Budget Database Backup
Documentation=https://github.com/lllypuk/Family-Finances-Service

[Timer]
# Run daily at 3:00 AM
OnCalendar=*-*-* 03:00:00
# Randomize start time within 15 minutes
RandomizedDelaySec=900
# Run immediately if missed (e.g., system was off)
Persistent=true
# Timezone
# AccuracySec=1s

[Install]
WantedBy=timers.target
```

---

### Backup Script (`deploy/scripts/backup.sh`)

```bash
#!/bin/bash
set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/opt/family-budget/backups}"
DATA_DIR="${DATA_DIR:-/opt/family-budget/data}"
DATABASE_FILE="${DATABASE_PATH:-$DATA_DIR/budget.db}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
MAX_BACKUPS="${MAX_BACKUPS:-50}"

# Timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/budget_${TIMESTAMP}.db"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Create backup directory if not exists
mkdir -p "${BACKUP_DIR}"

# Check if database exists
if [[ ! -f "${DATABASE_FILE}" ]]; then
    log "ERROR: Database file not found: ${DATABASE_FILE}"
    exit 1
fi

# Create backup using SQLite backup command (safe for live database)
log "Starting backup of ${DATABASE_FILE}"
sqlite3 "${DATABASE_FILE}" ".backup '${BACKUP_FILE}'"

# Verify backup
if [[ -f "${BACKUP_FILE}" ]]; then
    BACKUP_SIZE=$(stat -c%s "${BACKUP_FILE}")
    log "Backup created: ${BACKUP_FILE} (${BACKUP_SIZE} bytes)"

    # Verify backup integrity
    if sqlite3 "${BACKUP_FILE}" "PRAGMA integrity_check;" | grep -q "ok"; then
        log "Backup integrity check: OK"
    else
        log "ERROR: Backup integrity check failed!"
        rm -f "${BACKUP_FILE}"
        exit 1
    fi
else
    log "ERROR: Backup file was not created"
    exit 1
fi

# Cleanup old backups by age
log "Cleaning up backups older than ${RETENTION_DAYS} days"
find "${BACKUP_DIR}" -name "budget_*.db" -type f -mtime +${RETENTION_DAYS} -delete

# Cleanup by count (keep only MAX_BACKUPS most recent)
log "Keeping only ${MAX_BACKUPS} most recent backups"
ls -t "${BACKUP_DIR}"/budget_*.db 2>/dev/null | tail -n +$((MAX_BACKUPS + 1)) | xargs -r rm -f

# Report status
TOTAL_BACKUPS=$(ls -1 "${BACKUP_DIR}"/budget_*.db 2>/dev/null | wc -l)
TOTAL_SIZE=$(du -sh "${BACKUP_DIR}" | cut -f1)
log "Backup complete. Total backups: ${TOTAL_BACKUPS}, Total size: ${TOTAL_SIZE}"
```

---

### Health Check Script (`deploy/scripts/health-check.sh`)

```bash
#!/bin/bash
set -euo pipefail

# Configuration
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/health}"
TIMEOUT="${HEALTH_TIMEOUT:-5}"

# Check health endpoint
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT}" "${HEALTH_URL}" 2>/dev/null || echo "000")

if [[ "${HTTP_CODE}" == "200" ]]; then
    echo "OK"
    exit 0
else
    echo "FAIL (HTTP ${HTTP_CODE})"
    exit 1
fi
```

---

### Installation Script (`deploy/scripts/install-systemd.sh`)

```bash
#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/family-budget"
SERVICE_USER="familybudget"
SYSTEMD_DIR="/etc/systemd/system"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Check root
if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root"
    exit 1
fi

log "Installing Family Budget Service (systemd)..."

# Create user
if ! id "${SERVICE_USER}" &>/dev/null; then
    log "Creating service user: ${SERVICE_USER}"
    useradd --system --no-create-home --shell /usr/sbin/nologin "${SERVICE_USER}"
fi

# Create directories
log "Creating directories..."
mkdir -p "${INSTALL_DIR}"/{bin,config,data,backups,logs,scripts}

# Copy binary (assumes binary is in current directory)
if [[ -f "./family-budget-service" ]]; then
    cp ./family-budget-service "${INSTALL_DIR}/bin/"
    chmod 755 "${INSTALL_DIR}/bin/family-budget-service"
fi

# Copy scripts
cp ./scripts/backup.sh "${INSTALL_DIR}/scripts/"
cp ./scripts/health-check.sh "${INSTALL_DIR}/scripts/"
chmod 755 "${INSTALL_DIR}/scripts/"*.sh

# Create default config if not exists
if [[ ! -f "${INSTALL_DIR}/config/.env" ]]; then
    log "Creating default configuration..."
    cat > "${INSTALL_DIR}/config/.env" << 'EOF'
# Family Budget Service Configuration
SERVER_PORT=8080
SERVER_HOST=127.0.0.1
DATABASE_PATH=/opt/family-budget/data/budget.db
SESSION_SECRET=CHANGE_THIS_TO_RANDOM_VALUE
ENVIRONMENT=production
LOG_LEVEL=info
EOF
    chmod 600 "${INSTALL_DIR}/config/.env"
    log "WARNING: Please edit ${INSTALL_DIR}/config/.env and set SESSION_SECRET"
fi

# Set permissions
log "Setting permissions..."
chown -R "${SERVICE_USER}:${SERVICE_USER}" "${INSTALL_DIR}"
chmod 700 "${INSTALL_DIR}/data"
chmod 700 "${INSTALL_DIR}/backups"
chmod 700 "${INSTALL_DIR}/config"

# Install systemd units
log "Installing systemd units..."
cp ./systemd/family-budget.service "${SYSTEMD_DIR}/"
cp ./systemd/family-budget-backup.service "${SYSTEMD_DIR}/"
cp ./systemd/family-budget-backup.timer "${SYSTEMD_DIR}/"

# Reload systemd
systemctl daemon-reload

# Enable services
log "Enabling services..."
systemctl enable family-budget.service
systemctl enable family-budget-backup.timer

log "Installation complete!"
log ""
log "Next steps:"
log "  1. Edit configuration: ${INSTALL_DIR}/config/.env"
log "  2. Start service: systemctl start family-budget"
log "  3. Start backup timer: systemctl start family-budget-backup.timer"
log "  4. Check status: systemctl status family-budget"
```

---

## Directory Structure

```
deploy/
├── systemd/
│   ├── family-budget.service
│   ├── family-budget-backup.service
│   └── family-budget-backup.timer
└── scripts/
    ├── backup.sh
    ├── health-check.sh
    └── install-systemd.sh
```

---

## Operations Guide

### Service Management

```bash
# Start service
sudo systemctl start family-budget

# Stop service
sudo systemctl stop family-budget

# Restart service
sudo systemctl restart family-budget

# Check status
sudo systemctl status family-budget

# View logs
sudo journalctl -u family-budget -f

# View last 100 log lines
sudo journalctl -u family-budget -n 100
```

### Backup Timer Management

```bash
# Start backup timer
sudo systemctl start family-budget-backup.timer

# Check timer status
sudo systemctl list-timers family-budget-backup.timer

# Run backup manually
sudo systemctl start family-budget-backup.service

# View backup logs
sudo journalctl -u family-budget-backup
```

### Troubleshooting

```bash
# Check service logs for errors
sudo journalctl -u family-budget -p err

# Check configuration
sudo systemctl cat family-budget

# Verify permissions
sudo -u familybudget ls -la /opt/family-budget/

# Test health check
sudo -u familybudget /opt/family-budget/scripts/health-check.sh
```

---

## Security Features

### Systemd Hardening

| Feature                  | Setting       | Purpose                    |
|--------------------------|---------------|----------------------------|
| `NoNewPrivileges`        | yes           | Block privilege escalation |
| `PrivateTmp`             | yes           | Private /tmp directory     |
| `ProtectSystem`          | strict        | Read-only filesystem       |
| `ProtectHome`            | yes           | No access to /home         |
| `ReadWritePaths`         | explicit list | Whitelist writable paths   |
| `MemoryDenyWriteExecute` | yes           | No W+X memory              |

### File Permissions

| Path                             | Permission | Owner        |
|----------------------------------|------------|--------------|
| `/opt/family-budget/bin`         | 755        | familybudget |
| `/opt/family-budget/config`      | 700        | familybudget |
| `/opt/family-budget/data`        | 700        | familybudget |
| `/opt/family-budget/backups`     | 700        | familybudget |
| `/opt/family-budget/config/.env` | 600        | familybudget |

---

## Testing Checklist

- [ ] Service starts correctly
- [ ] Service restarts on failure
- [ ] Backup timer runs on schedule
- [ ] Backup script creates valid backups
- [ ] Backup cleanup works correctly
- [ ] Health check script works
- [ ] Security hardening doesn't break functionality
- [ ] Logs appear in journald
- [ ] Service stops gracefully

## Acceptance Criteria

1. Service starts and runs under dedicated user
2. All security hardening options work
3. Backup timer runs daily
4. Backups are valid SQLite databases
5. Old backups are cleaned up
6. Documentation covers all commands

## Dependencies

- Task 005 (upgrade script) - may need to restart service

## Estimated Complexity

Medium
