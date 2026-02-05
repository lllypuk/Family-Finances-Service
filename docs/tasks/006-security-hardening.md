# Task 006: Security Hardening

## Overview

Implement comprehensive security measures for self-hosted deployment including firewall configuration, fail2ban
integration, and security documentation.

## Priority: HIGH

## Status: TODO

## Requirements

### 1. Firewall Configuration (UFW)

#### Setup Script (`deploy/scripts/setup-firewall.sh`)

```bash
#!/bin/bash
set -euo pipefail

# ============================================
# Family Budget Service - Firewall Setup
# ============================================

log_info() { echo "[INFO] $1"; }
log_warn() { echo "[WARN] $1"; }

# Check root
if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root"
    exit 1
fi

log_info "Configuring UFW firewall..."

# Install UFW if not present
if ! command -v ufw &> /dev/null; then
    log_info "Installing UFW..."
    apt-get update && apt-get install -y ufw
fi

# Reset UFW to default
log_info "Resetting UFW to defaults..."
ufw --force reset

# Default policies
log_info "Setting default policies..."
ufw default deny incoming
ufw default allow outgoing

# SSH (adjust port if non-standard)
SSH_PORT="${SSH_PORT:-22}"
log_info "Allowing SSH on port ${SSH_PORT}..."
ufw allow ${SSH_PORT}/tcp comment 'SSH'

# HTTP/HTTPS
log_info "Allowing HTTP/HTTPS..."
ufw allow 80/tcp comment 'HTTP'
ufw allow 443/tcp comment 'HTTPS'

# Block direct access to application port (internal only)
log_info "Blocking direct access to port 8080..."
ufw deny 8080/tcp comment 'Block direct app access'

# Rate limiting for SSH
log_info "Enabling SSH rate limiting..."
ufw limit ${SSH_PORT}/tcp comment 'SSH rate limit'

# Enable UFW
log_info "Enabling UFW..."
ufw --force enable

# Show status
log_info "Firewall configuration complete:"
ufw status verbose

# Additional iptables rules for protection
log_info "Adding additional protections..."

# Block invalid packets
iptables -A INPUT -m conntrack --ctstate INVALID -j DROP

# Block port scanning
iptables -A INPUT -p tcp --tcp-flags ALL NONE -j DROP
iptables -A INPUT -p tcp --tcp-flags ALL ALL -j DROP

# Save iptables rules
if command -v netfilter-persistent &> /dev/null; then
    netfilter-persistent save
fi

log_info "Firewall setup complete!"
```

---

### 2. Fail2ban Configuration

#### Fail2ban Filter (`deploy/fail2ban/family-budget.conf`)

```ini
# /etc/fail2ban/filter.d/family-budget.conf
[Definition]
# Failed login attempts
failregex = ^.*"POST /login.*" 401.*client=<HOST>.*$
            ^.*"POST /api/v1/auth/login.*" 401.*client=<HOST>.*$
            ^.*authentication failed.*client=<HOST>.*$
            ^.*invalid password.*client=<HOST>.*$

# Ignore successful logins
ignoreregex = ^.*"POST /login.*" 200.*$
              ^.*"POST /login.*" 302.*$
```

#### Fail2ban Jail (`deploy/fail2ban/jail.local`)

```ini
# /etc/fail2ban/jail.d/family-budget.local

[family-budget]
enabled = true
port = http,https
filter = family-budget
logpath = /var/log/nginx/family-budget.access.log
maxretry = 5
findtime = 300
bantime = 3600
action = %(action_mwl)s

[family-budget-aggressive]
enabled = true
port = http,https
filter = family-budget
logpath = /var/log/nginx/family-budget.access.log
maxretry = 10
findtime = 60
bantime = 86400
action = %(action_mwl)s

# Rate limiting on nginx errors
[nginx-limit-req]
enabled = true
port = http,https
filter = nginx-limit-req
logpath = /var/log/nginx/family-budget.error.log
maxretry = 10
findtime = 60
bantime = 600
```

#### Setup Script (`deploy/scripts/setup-fail2ban.sh`)

```bash
#!/bin/bash
set -euo pipefail

log_info() { echo "[INFO] $1"; }

# Check root
if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root"
    exit 1
fi

log_info "Setting up fail2ban..."

# Install fail2ban
if ! command -v fail2ban-client &> /dev/null; then
    log_info "Installing fail2ban..."
    apt-get update && apt-get install -y fail2ban
fi

# Copy filter
log_info "Installing fail2ban filter..."
cp /opt/family-budget/fail2ban/family-budget.conf /etc/fail2ban/filter.d/

# Copy jail config
log_info "Installing jail configuration..."
cp /opt/family-budget/fail2ban/jail.local /etc/fail2ban/jail.d/family-budget.local

# Restart fail2ban
log_info "Restarting fail2ban..."
systemctl restart fail2ban

# Enable on boot
systemctl enable fail2ban

# Show status
log_info "Fail2ban status:"
fail2ban-client status
fail2ban-client status family-budget 2>/dev/null || log_info "Jail not active yet (needs log file)"

log_info "Fail2ban setup complete!"
```

---

### 3. Security Headers Verification

#### Check Script (`deploy/scripts/check-security.sh`)

```bash
#!/bin/bash
set -euo pipefail

# ============================================
# Security Configuration Check
# ============================================

DOMAIN="${1:-localhost}"
URL="https://${DOMAIN}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

echo "========================================"
echo "Security Check for ${DOMAIN}"
echo "========================================"
echo ""

# Get headers
HEADERS=$(curl -sI "${URL}" 2>/dev/null || echo "")

if [[ -z "${HEADERS}" ]]; then
    fail "Could not connect to ${URL}"
    exit 1
fi

# Check security headers
echo "Security Headers:"
echo "-----------------"

# X-Frame-Options
if echo "${HEADERS}" | grep -qi "x-frame-options"; then
    pass "X-Frame-Options present"
else
    fail "X-Frame-Options missing"
fi

# X-Content-Type-Options
if echo "${HEADERS}" | grep -qi "x-content-type-options"; then
    pass "X-Content-Type-Options present"
else
    fail "X-Content-Type-Options missing"
fi

# X-XSS-Protection
if echo "${HEADERS}" | grep -qi "x-xss-protection"; then
    pass "X-XSS-Protection present"
else
    warn "X-XSS-Protection missing (deprecated but still useful)"
fi

# Content-Security-Policy
if echo "${HEADERS}" | grep -qi "content-security-policy"; then
    pass "Content-Security-Policy present"
else
    fail "Content-Security-Policy missing"
fi

# Strict-Transport-Security
if echo "${HEADERS}" | grep -qi "strict-transport-security"; then
    pass "Strict-Transport-Security (HSTS) present"
else
    warn "Strict-Transport-Security missing (enable after SSL verification)"
fi

# Referrer-Policy
if echo "${HEADERS}" | grep -qi "referrer-policy"; then
    pass "Referrer-Policy present"
else
    fail "Referrer-Policy missing"
fi

# Permissions-Policy
if echo "${HEADERS}" | grep -qi "permissions-policy"; then
    pass "Permissions-Policy present"
else
    warn "Permissions-Policy missing"
fi

# Server header (should be hidden)
if echo "${HEADERS}" | grep -qi "^server:.*nginx"; then
    warn "Server header exposed (consider hiding)"
else
    pass "Server header hidden or generic"
fi

echo ""
echo "SSL/TLS Configuration:"
echo "----------------------"

# Check SSL (requires openssl)
if command -v openssl &> /dev/null; then
    # Check TLS version
    if openssl s_client -connect "${DOMAIN}:443" -tls1_2 </dev/null 2>/dev/null | grep -q "Cipher"; then
        pass "TLS 1.2 supported"
    fi

    if openssl s_client -connect "${DOMAIN}:443" -tls1_3 </dev/null 2>/dev/null | grep -q "Cipher"; then
        pass "TLS 1.3 supported"
    fi

    # Check certificate expiry
    EXPIRY=$(echo | openssl s_client -connect "${DOMAIN}:443" 2>/dev/null | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
    if [[ -n "${EXPIRY}" ]]; then
        EXPIRY_EPOCH=$(date -d "${EXPIRY}" +%s 2>/dev/null || echo "0")
        NOW_EPOCH=$(date +%s)
        DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

        if [[ ${DAYS_LEFT} -gt 30 ]]; then
            pass "Certificate valid for ${DAYS_LEFT} days"
        elif [[ ${DAYS_LEFT} -gt 0 ]]; then
            warn "Certificate expires in ${DAYS_LEFT} days"
        else
            fail "Certificate expired or invalid"
        fi
    fi
fi

echo ""
echo "Application Security:"
echo "--------------------"

# Check if app port is exposed
if curl -s --connect-timeout 2 "http://${DOMAIN}:8080/health" &>/dev/null; then
    fail "Port 8080 is publicly accessible (should be blocked)"
else
    pass "Port 8080 is not publicly accessible"
fi

# Check HTTP to HTTPS redirect
HTTP_REDIRECT=$(curl -sI "http://${DOMAIN}" 2>/dev/null | head -1)
if echo "${HTTP_REDIRECT}" | grep -q "301\|302\|308"; then
    pass "HTTP redirects to HTTPS"
else
    warn "HTTP does not redirect to HTTPS"
fi

echo ""
echo "========================================"
echo "Check complete!"
echo ""
echo "For detailed SSL analysis, visit:"
echo "https://www.ssllabs.com/ssltest/analyze.html?d=${DOMAIN}"
echo ""
echo "For security headers analysis, visit:"
echo "https://securityheaders.com/?q=${URL}"
echo "========================================"
```

---

### 4. Security Documentation

#### Security Guide (`docs/SECURITY.md`)

```markdown
# Security Guide

## Overview

This document describes security measures implemented in Family Budget Service
and recommendations for secure self-hosted deployment.

## Security Features

### Authentication

- **Password hashing**: bcrypt with cost factor 12
- **Session management**: HTTP-only, Secure, SameSite=Lax cookies
- **CSRF protection**: Token-based CSRF protection on all forms
- **Role-based access**: Admin, Member, Child roles with different permissions

### Data Protection

- **SQLite encryption**: Database file permissions (600)
- **Backup encryption**: Recommended for off-site backups
- **No external dependencies**: All data stays on your server

### Network Security

- **TLS 1.2/1.3**: Strong encryption for data in transit
- **Security headers**: Full set of modern security headers
- **Rate limiting**: Protection against brute force attacks
- **Fail2ban integration**: Automatic IP blocking

## Deployment Security Checklist

### Before Deployment

- [ ] Generate strong SESSION_SECRET (32+ bytes)
- [ ] Generate strong CSRF_SECRET
- [ ] Change default admin credentials immediately after first login
- [ ] Review firewall rules

### After Deployment

- [ ] Enable HTTPS with valid certificate
- [ ] Enable HSTS (after verifying SSL works)
- [ ] Configure fail2ban
- [ ] Set up automated backups
- [ ] Test backup restoration
- [ ] Run security check script

### Ongoing Maintenance

- [ ] Keep application updated
- [ ] Monitor fail2ban logs
- [ ] Review access logs periodically
- [ ] Renew SSL certificates (automatic with Let's Encrypt)
- [ ] Regular backup verification

## Secret Generation

Generate secure secrets using:

```bash
# Session secret
openssl rand -base64 32

# CSRF secret
openssl rand -base64 32

# Admin password (initial)
openssl rand -base64 16
```

## Network Architecture

Recommended network setup:

```
Internet
    │
    ▼ (HTTPS only)
┌─────────────┐
│   Firewall  │ ← UFW: Allow 80, 443 only
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Nginx/     │ ← TLS termination
│  Caddy      │ ← Rate limiting
└──────┬──────┘ ← Security headers
       │
       │ (HTTP, localhost only)
       ▼
┌─────────────┐
│    App      │ ← Port 8080 (internal)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   SQLite    │ ← File permissions 600
└─────────────┘
```

## Incident Response

### Suspected Breach

1. Immediately change all secrets
2. Review access logs
3. Check fail2ban logs for suspicious IPs
4. Restore from known-good backup if necessary
5. Re-generate invite tokens

### Password Reset

If admin password is lost:

1. Stop the service
2. Use SQLite shell to reset password hash
3. Restart service
4. Change password immediately via web UI

## Security Contacts

Report security vulnerabilities via GitHub Security Advisories.

## Compliance Notes

This application is designed for personal/family use. For business use,
additional measures may be required depending on your jurisdiction:

- GDPR (EU): Data export/deletion features available via admin panel
- Data retention: Configurable backup retention
- Access logging: All authentication events logged

```

---

## Files to Create

```

deploy/
├── scripts/
│ ├── setup-firewall.sh
│ ├── setup-fail2ban.sh
│ └── check-security.sh
├── fail2ban/
│ ├── family-budget.conf # Filter definition
│ └── jail.local # Jail configuration
docs/
└── SECURITY.md

```

## Testing Checklist

- [ ] UFW blocks direct port 8080 access
- [ ] UFW allows SSH, HTTP, HTTPS
- [ ] Fail2ban blocks after failed logins
- [ ] Fail2ban unblocks after ban time
- [ ] Security check script runs correctly
- [ ] All security headers present
- [ ] HTTP redirects to HTTPS
- [ ] SSL Labs score A or A+

## Acceptance Criteria

1. Firewall script configures UFW correctly
2. Fail2ban blocks brute force attacks
3. Security check script validates configuration
4. Documentation covers all security aspects
5. No false positives in fail2ban

## Dependencies

- Task 002 (nginx config) - for rate limiting and headers
- Task 003 (docker-compose) - for network isolation

## Estimated Complexity

Medium
