# Family Budget Service - Deployment Guide

This directory contains production-ready deployment configurations and scripts for self-hosted installations.

## Quick Start

### Prerequisites

- Linux server (Ubuntu 22.04+, Debian 11+, or Rocky Linux 9)
- Minimum 2GB RAM, 10GB disk space
- Root or sudo access
- Domain name pointed to your server (for SSL)

### One-Command Installation

```bash
# Download and run installation script
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/scripts/install.sh | sudo bash -s -- --domain budget.example.com --email admin@example.com
```

Or clone the repository and run locally:

```bash
git clone https://github.com/lllypuk/Family-Finances-Service.git
cd Family-Finances-Service
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

## Directory Structure

```
deploy/
├── scripts/                    # Installation and management scripts
│   ├── install.sh             # Main installation script
│   ├── setup-ssl-nginx.sh     # SSL setup for Nginx
│   ├── setup-ssl-caddy.sh     # SSL setup for Caddy
│   └── lib/                   # Shared library functions
│       ├── common.sh          # Common utilities
│       ├── docker.sh          # Docker installation
│       └── firewall.sh        # Firewall configuration
├── nginx/                      # Nginx reverse proxy configs
│   ├── nginx.conf             # Main Nginx configuration
│   ├── conf.d/                # Site configurations
│   └── snippets/              # Reusable config snippets
├── caddy/                      # Caddy reverse proxy configs
│   └── Caddyfile.template     # Caddy configuration
├── docker-compose.prod.yml     # Standalone production setup
├── docker-compose.nginx.yml    # Production with Nginx
├── docker-compose.caddy.yml    # Production with Caddy
└── .env.production.example     # Environment template
```

## Deployment Options

### Option 1: Standalone (No Reverse Proxy)

Best for testing or internal networks.

```bash
sudo ./deploy/scripts/install.sh --domain localhost
```

Uses `docker-compose.prod.yml` - application runs on port 8080 without SSL.

### Option 2: Nginx + Let's Encrypt

Best for traditional deployments with manual control.

1. Install with Nginx:
```bash
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

2. Copy Nginx docker-compose:
```bash
cd /opt/family-budget
sudo cp ~/Family-Finances-Service/deploy/docker-compose.nginx.yml docker-compose.yml
```

3. Setup SSL:
```bash
sudo ~/Family-Finances-Service/deploy/scripts/setup-ssl-nginx.sh \
  --domain budget.example.com \
  --email admin@example.com
```

**Features:**
- HTTP → HTTPS redirect
- Let's Encrypt SSL certificates (auto-renewal with Certbot)
- Rate limiting (5 req/min for login, 10 req/sec general)
- Security headers
- Static file caching

### Option 3: Caddy (Automatic SSL)

Best for easy setup and automatic certificate management.

1. Install with Caddy:
```bash
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

2. Copy Caddy docker-compose:
```bash
cd /opt/family-budget
sudo cp ~/Family-Finances-Service/deploy/docker-compose.caddy.yml docker-compose.yml
```

3. Setup SSL (automatic):
```bash
sudo ~/Family-Finances-Service/deploy/scripts/setup-ssl-caddy.sh \
  --domain budget.example.com \
  --email admin@example.com
```

**Features:**
- Automatic HTTPS (no manual certificate management)
- HTTP/3 support
- Automatic certificate renewal
- Rate limiting and security headers
- Simpler configuration

## Security Features

### Firewall Configuration

The installation script automatically configures the firewall:

- **Allow:** SSH (22), HTTP (80), HTTPS (443)
- **Block:** Direct application access (8080)
- **UFW** (Ubuntu/Debian) or **firewalld** (RHEL-based)

### SSL/TLS

Both Nginx and Caddy configurations include:

- **TLS 1.2 and 1.3** only (no outdated protocols)
- **Strong cipher suites** (ECDHE, AES-GCM)
- **Perfect Forward Secrecy**
- **OCSP stapling**
- **HSTS** (optional, enable after confirming SSL works)

### Security Headers

All configurations include:

- `X-Frame-Options: SAMEORIGIN` (clickjacking protection)
- `X-Content-Type-Options: nosniff` (MIME sniffing protection)
- `X-XSS-Protection: 1; mode=block` (XSS filter)
- `Content-Security-Policy` (script injection protection)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` (feature restrictions)

### Rate Limiting

| Endpoint    | Limit          | Purpose               |
|-------------|----------------|-----------------------|
| `/login`    | 5 req/min      | Brute force protection|
| `/api/*`    | 100 req/min    | API abuse prevention  |
| General     | 10 req/sec     | DDoS protection       |
| `/health`   | Unlimited      | Monitoring            |

## Environment Configuration

Create `.env` file in `/opt/family-budget/config/`:

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DOMAIN=budget.example.com

# Database
DATABASE_PATH=/data/budget.db

# Security (generate with: openssl rand -base64 32)
SESSION_SECRET=YOUR_GENERATED_SECRET_HERE
CSRF_SECRET=YOUR_GENERATED_SECRET_HERE

# Logging
LOG_LEVEL=info
ENVIRONMENT=production

# Admin Contact
ADMIN_EMAIL=admin@example.com
ACME_EMAIL=admin@example.com  # For Caddy SSL
```

## Common Operations

### View Logs

```bash
cd /opt/family-budget
docker compose logs -f
docker compose logs -f app      # Application only
docker compose logs -f nginx    # Nginx only
docker compose logs -f caddy    # Caddy only
```

### Restart Services

```bash
cd /opt/family-budget
docker compose restart
docker compose restart app      # Restart only application
```

### Update Application

```bash
cd /opt/family-budget
docker compose pull app
docker compose up -d app
```

### Backup Database

```bash
cd /opt/family-budget
docker compose exec app wget -q -O- http://localhost:8080/admin/backup/create
# Or copy database file directly
sudo cp /opt/family-budget/data/budget.db ~/budget-backup-$(date +%Y%m%d).db
```

### Check Status

```bash
cd /opt/family-budget
docker compose ps
curl -s http://localhost:8080/health
curl -s https://budget.example.com/health
```

## Upgrading

### Manual Upgrade

```bash
cd /opt/family-budget

# Backup database
sudo cp data/budget.db backups/budget-$(date +%Y%m%d-%H%M%S).db

# Pull latest image
docker compose pull app

# Restart with new image
docker compose up -d app

# Verify
docker compose logs -f app
curl -s https://budget.example.com/health
```

### Rollback

```bash
cd /opt/family-budget

# Stop services
docker compose down

# Restore database backup
sudo cp backups/budget-YYYYMMDD-HHMMSS.db data/budget.db

# Start services
docker compose up -d
```

## Troubleshooting

### Application won't start

```bash
# Check logs
docker compose logs app

# Check database permissions
ls -la /opt/family-budget/data/

# Verify environment variables
cat /opt/family-budget/config/.env
```

### SSL certificate issues (Nginx)

```bash
# Check Certbot logs
docker compose logs certbot

# Manually request certificate
docker compose run --rm certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  --email admin@example.com \
  --agree-tos \
  -d budget.example.com

# Test Nginx config
docker compose exec nginx nginx -t
```

### SSL certificate issues (Caddy)

```bash
# Check Caddy logs
docker compose logs caddy

# Caddy automatically retries failed certificate requests
# If domain is not resolving, check DNS settings
dig budget.example.com

# Verify port 80 and 443 are accessible from internet
curl -I http://budget.example.com
```

### Firewall blocking access

```bash
# Check firewall status (Ubuntu/Debian)
sudo ufw status

# Check firewall status (RHEL-based)
sudo firewall-cmd --list-all

# Check if Docker is running
docker ps

# Check if ports are listening
ss -tuln | grep -E ':(80|443|8080)'
```

## Performance Tuning

### Nginx

Edit `/opt/family-budget/nginx/nginx.conf`:

```nginx
worker_processes auto;  # One per CPU core
worker_connections 2048;  # Increase for high traffic
```

### Application

Edit `/opt/family-budget/config/.env`:

```bash
LOG_LEVEL=warn  # Reduce log verbosity
```

### Database

SQLite performs well for small to medium installations. For large deployments (>1000 users), consider:

- Regular VACUUM operations
- WAL mode (already enabled)
- Read-only replicas (future feature)

## Security Hardening

### 1. Enable HSTS (after confirming SSL works)

**Nginx:**
```bash
# Edit /opt/family-budget/nginx/snippets/security-headers.conf
# Uncomment the HSTS header
sudo docker compose exec nginx nginx -s reload
```

**Caddy:**
```bash
# Edit /opt/family-budget/caddy/Caddyfile
# Uncomment the HSTS header
sudo docker compose restart caddy
```

### 2. Setup fail2ban (recommended)

See `docs/tasks/006-security-hardening.md` for fail2ban configuration.

### 3. Regular Updates

```bash
# Update system packages
sudo apt update && sudo apt upgrade  # Ubuntu/Debian
sudo dnf update  # RHEL-based

# Update Docker images
cd /opt/family-budget
docker compose pull
docker compose up -d
```

### 4. Monitor Logs

```bash
# Watch for suspicious activity
docker compose logs -f app | grep -i "failed\|error\|unauthorized"
```

## Uninstalling

```bash
cd /opt/family-budget

# Stop and remove containers
docker compose down -v

# Remove installation directory (backup data first!)
sudo cp -r /opt/family-budget/data ~/family-budget-backup
sudo rm -rf /opt/family-budget

# Remove firewall rules
sudo ufw delete allow 80/tcp
sudo ufw delete allow 443/tcp
```

## Support

- **Documentation:** See `docs/tasks/` directory
- **Issues:** GitHub Issues
- **Security:** Report security issues privately

## License

See LICENSE file in repository root.
