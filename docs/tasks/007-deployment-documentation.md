# Task 007: Deployment Documentation

## Overview

Create comprehensive deployment documentation including step-by-step guides, troubleshooting, and FAQ for self-hosted
users.

## Priority: MEDIUM

## Status: TODO

## Requirements

### Main Deployment Guide (`docs/DEPLOYMENT.md`)

```markdown
# Deployment Guide

Complete guide for deploying Family Budget Service on your own server.

## Table of Contents

1. [Requirements](#requirements)
2. [Quick Start](#quick-start)
3. [Docker Deployment](#docker-deployment)
4. [Native Deployment](#native-deployment)
5. [Reverse Proxy Setup](#reverse-proxy-setup)
6. [SSL/TLS Configuration](#ssltls-configuration)
7. [Post-Installation](#post-installation)
8. [Updating](#updating)
9. [Backup & Restore](#backup--restore)
10. [Troubleshooting](#troubleshooting)

---

## Requirements

### Minimum System Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 1 core | 2 cores |
| RAM | 512 MB | 1 GB |
| Disk | 1 GB | 10 GB |
| OS | Ubuntu 22.04+ | Ubuntu 24.04 LTS |

### Network Requirements

- Public IP address (or NAT with port forwarding)
- Domain name (recommended)
- Ports 80 and 443 accessible

### Software Requirements (Docker deployment)

- Docker Engine 24.0+
- Docker Compose v2.20+

### Software Requirements (Native deployment)

- Go 1.25+ (for building)
- SQLite 3.40+

---

## Quick Start

### Fastest deployment (Docker + Caddy)

```bash
# 1. Download deployment files
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/install.sh | bash

# 2. Edit configuration
nano /opt/family-budget/.env

# 3. Start services
cd /opt/family-budget
docker-compose up -d

# 4. Access the application
# Open https://your-domain.com in browser
```

---

## Docker Deployment

### Step 1: Install Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in

# Verify
docker --version
docker compose version
```

### Step 2: Create Directory Structure

```bash
sudo mkdir -p /opt/family-budget/{data,backups,config}
cd /opt/family-budget
```

### Step 3: Download Configuration Files

```bash
# Download docker-compose file
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/docker-compose.caddy.yml \
  -o docker-compose.yml

# Download Caddyfile
mkdir -p caddy
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/caddy/Caddyfile \
  -o caddy/Caddyfile

# Download environment template
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/.env.production.example \
  -o .env
```

### Step 4: Configure Environment

```bash
# Generate secrets
SESSION_SECRET=$(openssl rand -base64 32)
CSRF_SECRET=$(openssl rand -base64 32)

# Edit configuration
nano .env
```

Required settings:

```bash
DOMAIN=budget.yourdomain.com
ACME_EMAIL=your-email@example.com
SESSION_SECRET=<generated-secret>
```

### Step 5: Start Services

```bash
# Start in background
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f
```

### Step 6: Verify Installation

```bash
# Check health endpoint
curl -s https://your-domain.com/health

# Should return: OK
```

---

## Native Deployment

For environments where Docker is not available.

### Step 1: Install Dependencies

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y sqlite3 wget

# Download binary
wget https://github.com/lllypuk/Family-Finances-Service/releases/latest/download/family-budget-linux-amd64.tar.gz
tar -xzf family-budget-linux-amd64.tar.gz
```

### Step 2: Create User and Directories

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin familybudget
sudo mkdir -p /opt/family-budget/{bin,config,data,backups,logs}
sudo mv family-budget-service /opt/family-budget/bin/
sudo chown -R familybudget:familybudget /opt/family-budget
```

### Step 3: Configure Application

```bash
sudo nano /opt/family-budget/config/.env
```

```bash
SERVER_PORT=8080
SERVER_HOST=127.0.0.1
DATABASE_PATH=/opt/family-budget/data/budget.db
SESSION_SECRET=<your-secret-here>
ENVIRONMENT=production
LOG_LEVEL=info
```

### Step 4: Install Systemd Service

```bash
# Download service file
sudo curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/systemd/family-budget.service \
  -o /etc/systemd/system/family-budget.service

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable family-budget
sudo systemctl start family-budget

# Check status
sudo systemctl status family-budget
```

---

## Reverse Proxy Setup

### Option A: Caddy (Recommended - Auto SSL)

```bash
# Install Caddy
sudo apt install -y caddy

# Configure
sudo nano /etc/caddy/Caddyfile
```

```caddyfile
budget.yourdomain.com {
    reverse_proxy localhost:8080

    header {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
    }
}
```

```bash
sudo systemctl restart caddy
```

### Option B: Nginx

```bash
# Install Nginx and Certbot
sudo apt install -y nginx certbot python3-certbot-nginx

# Configure site
sudo nano /etc/nginx/sites-available/family-budget
```

```nginx
server {
    listen 80;
    server_name budget.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/family-budget /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx

# Get SSL certificate
sudo certbot --nginx -d budget.yourdomain.com
```

---

## SSL/TLS Configuration

### Let's Encrypt (Automatic)

With Caddy, SSL is automatic. With Nginx:

```bash
# Initial certificate
sudo certbot --nginx -d budget.yourdomain.com

# Verify auto-renewal
sudo certbot renew --dry-run
```

### Custom Certificate

```bash
# Place certificates
sudo mkdir -p /etc/ssl/family-budget
sudo cp your-cert.pem /etc/ssl/family-budget/fullchain.pem
sudo cp your-key.pem /etc/ssl/family-budget/privkey.pem
sudo chmod 600 /etc/ssl/family-budget/privkey.pem

# Update nginx/caddy config to use these paths
```

---

## Post-Installation

### 1. Create First Admin User

1. Open https://your-domain.com in browser
2. Click "Register"
3. Fill in admin details
4. First registered user becomes admin

### 2. Configure Firewall

```bash
# Install UFW
sudo apt install -y ufw

# Configure rules
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
sudo ufw enable
```

### 3. Set Up Backups

```bash
# Install backup timer
sudo curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/systemd/family-budget-backup.timer \
  -o /etc/systemd/system/family-budget-backup.timer

sudo systemctl enable family-budget-backup.timer
sudo systemctl start family-budget-backup.timer
```

---

## Updating

### Docker Update

```bash
cd /opt/family-budget

# Pull new image
docker compose pull

# Restart with new image
docker compose up -d
```

### Native Update

```bash
# Download upgrade script
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/scripts/upgrade.sh | sudo bash
```

---

## Backup & Restore

### Create Backup

```bash
# Docker
docker exec family-budget-app sqlite3 /data/budget.db ".backup '/data/backup.db'"
docker cp family-budget-app:/data/backup.db ./backup_$(date +%Y%m%d).db

# Native
sqlite3 /opt/family-budget/data/budget.db ".backup '/opt/family-budget/backups/backup_$(date +%Y%m%d).db'"
```

### Restore Backup

```bash
# Stop service first!
docker compose stop app
# OR
sudo systemctl stop family-budget

# Restore
cp backup_file.db /opt/family-budget/data/budget.db

# Start service
docker compose start app
# OR
sudo systemctl start family-budget
```

---

## Troubleshooting

### Service won't start

```bash
# Check logs
docker compose logs app
# OR
sudo journalctl -u family-budget -n 100

# Common issues:
# - SESSION_SECRET not set
# - Port 8080 already in use
# - Permissions on data directory
```

### Can't access web interface

```bash
# Check if app is running
curl http://localhost:8080/health

# Check firewall
sudo ufw status

# Check nginx/caddy
sudo systemctl status nginx
sudo systemctl status caddy
```

### SSL certificate issues

```bash
# Check certificate
sudo certbot certificates

# Force renewal
sudo certbot renew --force-renewal

# Check Caddy logs
sudo journalctl -u caddy
```

### Database issues

```bash
# Check integrity
sqlite3 /opt/family-budget/data/budget.db "PRAGMA integrity_check;"

# If corrupt, restore from backup
```

---

## FAQ

**Q: Can I run this on a Raspberry Pi?**
A: Yes, ARM64 builds are available. Use the `linux-arm64` binary or Docker image.

**Q: How do I change the port?**
A: Edit `SERVER_PORT` in `.env` and restart the service.

**Q: Can multiple families use one instance?**
A: Currently, one instance = one family. For multiple families, run separate instances.

**Q: How do I migrate from another budgeting app?**
A: Export your data to CSV and use the import feature in the admin panel.

**Q: Is my data encrypted?**
A: Data at rest is not encrypted by default. Use full-disk encryption for additional security.

```

---

## Files to Create

```

docs/
├── DEPLOYMENT.md # Main deployment guide
├── SECURITY.md # Security guide (Task 006)
├── BACKUP.md # Backup procedures
├── TROUBLESHOOTING.md # Common issues
└── FAQ.md # Frequently asked questions

```

## Testing Checklist

- [ ] Instructions work on fresh Ubuntu 22.04
- [ ] Instructions work on fresh Ubuntu 24.04
- [ ] Docker deployment works end-to-end
- [ ] Native deployment works end-to-end
- [ ] Caddy SSL automation works
- [ ] Nginx + Certbot works
- [ ] Backup/restore procedure works
- [ ] All troubleshooting steps are accurate

## Acceptance Criteria

1. Non-technical user can follow guide successfully
2. All commands are copy-paste ready
3. Troubleshooting covers common issues
4. Screenshots for key steps (optional)
5. Documentation is consistent with actual behavior

## Dependencies

- Task 001-006 (all deployment scripts)

## Estimated Complexity

Medium (mostly writing, some testing)
