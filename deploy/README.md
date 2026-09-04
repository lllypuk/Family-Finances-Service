# Family Budget Service - Deployment Guide

This directory contains production-ready deployment configurations and scripts for self-hosted installations.

## Quick Start

### Prerequisites

- Linux server (Ubuntu 22.04+, Debian 11+, or Rocky Linux 9)
- Minimum 512MB RAM, 10GB disk space. **1GB is recommended.**
  The running service itself fits in **128-256MB** (a single Go process plus SQLite);
  the extra headroom is for building the Docker image locally. Below 1GB the `go build`
  inside that image build may be OOM-killed — `install.sh` warns and suggests adding
  swap, which is enough to make a 512MB box work.
- Root or sudo access
- `git` and outbound network access — the image is built from source (see below)
- Domain name pointed to your server (for SSL)

### The image is built from source

There is no published `ghcr.io/lllypuk/family-finances-service` image yet, so every
compose file in this directory builds the app from `docker/Dockerfile` instead of
pulling. The build context is `${BUILD_CONTEXT:-..}`:

| Where the compose file runs from | `BUILD_CONTEXT` | Sources |
|---|---|---|
| in-place, from `deploy/` | `..` (default) | the repository checkout you are in |
| `/opt/family-budget` (after `install.sh`) | `./src` | clone made by `install.sh` |

`install.sh` clones the repository into `/opt/family-budget/src` and writes
`BUILD_CONTEXT=./src` into `config/.env`. Consequently `docker compose pull` no
longer works for the `app` service — use `docker compose build app`.

Set `REPO_GIT_URL` / `REPO_REF` before running `install.sh` to build from a fork or
a specific tag.

### Installation

Clone the repository and run the installer from the checkout — it sources
`deploy/scripts/lib/*.sh` next to itself, so piping it into `bash` (`curl … | sudo bash`)
cannot work: `BASH_SOURCE[0]` is not a path there and the `source` aborts under `set -e`.

```bash
git clone https://github.com/lllypuk/Family-Finances-Service.git
cd Family-Finances-Service
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

Re-running `install.sh` on an existing installation is safe: `data/`, `backups/` and the
secrets in `config/.env` are kept, the sources are re-fetched and the image is rebuilt.
Pass `--reinstall` only when you deliberately want the old tree moved to
`/opt/family-budget.backup.<timestamp>` and fresh secrets generated (that invalidates
every session).

### First run over an SSH tunnel

Do the first run without a public domain. `docker-compose.prod.yml` binds
`127.0.0.1:8080:8080`, so the service is not reachable from the network at all —
reach it from your laptop through SSH port forwarding:

```bash
# on your laptop; leave this running
ssh -N -L 8080:127.0.0.1:8080 mini
```

Then open <http://localhost:8080> in a browser and walk the whole flow:

1. `/` redirects to `/setup` while no family exists. Create the family and the first
   admin user — this is the only way an admin account is ever created.
2. Log in, add a category, add a transaction, open `/transactions` — the pages must
   render with the navigation bar and a Russian title.
3. `/health` answers `200` even before setup, so you can point monitoring at it from
   the start.
4. **Take a backup and restore it.** A backup you have never restored is not a backup:

   ```bash
   sudo /opt/family-budget/src/deploy/scripts/backup.sh
   # then verify the copy actually opens and holds your rows
   sqlite3 /opt/family-budget/backups/budget_<timestamp>.db \
     'pragma integrity_check; select count(*) from transactions;'
   ```

Only after that walk-through is clean should you point a domain at the server and
move to Option 2 or 3. Note that the application itself still has **no rate limiting
on login** — brute-force protection comes entirely from the nginx/Caddy configs and
fail2ban, so a public deployment must go through a reverse proxy.

Useful checks while the tunnel is up:

```bash
curl -s localhost:8080/health                 # {"status":"healthy",...,"version":"v0.4.1-3-gabc1234"}
# version — `git describe` сборки; `dev`, если образ собран без --build-arg VERSION
curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/api/v1/categories   # 401
```

That `401` is the expected answer: the REST API requires a session (see below).

## Directory Structure

```
deploy/
├── scripts/                    # Installation and management scripts
│   ├── install.sh             # Main installation script
│   ├── upgrade.sh             # Upgrade with backup and rollback
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
├── fail2ban/                   # fail2ban jail and filter
├── systemd/                    # systemd unit files
├── docker-compose.prod.yml     # Standalone production setup
├── docker-compose.nginx.yml    # Production with Nginx
├── docker-compose.caddy.yml    # Production with Caddy
├── docker-compose.minimal.yml  # No SSL, behind an existing proxy
└── .env.production.example     # Environment template
```

All four compose files build the app image from source; validate them all at once
from the repository root with `make compose-config`.

## Deployment Options

### Prepare the bind-mount directories first

`docker-compose.minimal.yml`, `docker-compose.nginx.yml` and `docker-compose.caddy.yml`
bind-mount `./data`, `./backups` and `./logs` from the host, and the container runs as
`1000:1000` (`docker/Dockerfile`: `USER 1000:1000`). Docker creates a *missing*
bind-mount source as **root-owned**, and then SQLite cannot create the database.

`install.sh` does this for you. When you run one of those compose files by hand, create
and chown the directories first, from the same directory the compose file runs in:

```bash
mkdir -p data backups logs
sudo chown -R 1000:1000 data backups logs
```

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
sudo cp src/deploy/docker-compose.nginx.yml docker-compose.yml
sudo docker compose build app
```

3. Setup SSL:

```bash
sudo /opt/family-budget/src/deploy/scripts/setup-ssl-nginx.sh \
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
sudo cp src/deploy/docker-compose.caddy.yml docker-compose.yml
sudo docker compose build app
```

3. Setup SSL (automatic):

```bash
sudo /opt/family-budget/src/deploy/scripts/setup-ssl-caddy.sh \
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

| Endpoint  | Limit       | Purpose                |
|-----------|-------------|------------------------|
| `/login`  | 5 req/min   | Brute force protection |
| `/api/*`  | 100 req/min | API abuse prevention   |
| General   | 10 req/sec  | DDoS protection        |
| `/health` | Unlimited   | Monitoring             |

These limits live in the nginx/Caddy configs only. The application has no built-in
rate limiting, so `docker-compose.minimal.yml` and native systemd deployments are not
covered by the `/login` limit — put a proxy in front of anything reachable from the
internet.

### API authentication

`/api/v1/*` is not public. The group is wrapped in `RequireAPIAuth`, and the only
credential is the same session cookie the web UI uses — there are no API tokens.

| Request | Response |
|---|---|
| Any `/api/v1/*` without a session | `401` + `{"error":{"code":"UNAUTHORIZED","message":"Authentication required"}}` |
| `POST`/`PUT`/`DELETE` without `X-Csrf-Token` | `403 CSRF token validation failed` |
| `POST`/`PUT`/`DELETE` with a valid token but no session | `401` |
| Role not allowed for the route | `403` + `{"error":{"code":"FORBIDDEN","message":"Insufficient permissions"}}` |

The `403`-before-`401` ordering is deliberate: CSRF protection is global middleware
and runs before the API group's auth middleware.

Role gates mirror the web UI: user management (`POST`/`PUT`/`DELETE /api/v1/users`)
and `DELETE /api/v1/categories/:id` are admin-only; the categories, transactions,
budgets and reports groups are admin-or-member, so the `child` role gets `403`.

### Programmatic API clients

CSRF protection stays on for `/api/v1` — with cookie-based auth it has to, otherwise
any third-party page could act as a logged-in user. A script therefore has to behave
like a browser: keep a cookie jar, and read a CSRF token out of a rendered page.

The token is rotated on login, so the token scraped from `/login` is worthless
afterwards — fetch a fresh one from an authenticated page.

```bash
BASE=http://localhost:8080
JAR=$(mktemp)

# 1. anonymous GET /login -> session cookie + a token for the login form
TOKEN=$(curl -s -c "$JAR" "$BASE/login" \
  | grep -oE '_token" value="[^"]*"' | head -1 | sed 's/.*value="//;s/"//')

# 2. log in (302 on success); the session cookie is replaced and the token rotated
curl -s -b "$JAR" -c "$JAR" -X POST "$BASE/login" -o /dev/null \
  --data-urlencode "_token=$TOKEN" \
  --data-urlencode "email=admin@example.com" \
  --data-urlencode "password=…"

# 3. read the post-login token from any authenticated page (the logout form carries it)
TOKEN=$(curl -s -b "$JAR" -c "$JAR" "$BASE/transactions" \
  | grep -oE '_token" value="[^"]*"' | head -1 | sed 's/.*value="//;s/"//')

# 4. reads need only the cookie
curl -s -b "$JAR" "$BASE/api/v1/categories"

# 5. writes need the cookie *and* the header
curl -s -b "$JAR" -X POST "$BASE/api/v1/categories" \
  -H 'Content-Type: application/json' -H "X-Csrf-Token: $TOKEN" \
  -d '{"name":"Кофе","type":"expense","color":"#ff0000","icon":"cup"}'
# -> 201 Created
```

Gotchas found while verifying this against a running server:

- Don't re-fetch `/login` after logging in — an authenticated visitor is redirected
  away from it, so there is no form to scrape. Use a page you have access to.
- A single request can write the session cookie more than once; let curl's cookie jar
  handle it (`-b`/`-c` on every call) rather than parsing `Set-Cookie` yourself.
- Never send `user_id` in a request body. It is not a field of the create requests any
  more — the author always comes from the session.

## Environment Configuration

Create `.env` file in `/opt/family-budget/config/`:

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DOMAIN=budget.example.com

# Database
DATABASE_PATH=/data/budget.db

# Build context: path to the source checkout, relative to /opt/family-budget
BUILD_CONTEXT=./src

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

# Preferred: the upgrade script backs up the DB and rolls back on failure
sudo ./src/deploy/scripts/upgrade.sh --version main

# Manual equivalent
git -C src fetch origin main && git -C src checkout --force --detach FETCH_HEAD
docker compose build app
docker compose up -d app
```

`--version` takes a **git ref** (tag, branch or commit), not an image tag.

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

# Update sources and rebuild the image (there is nothing to pull)
git -C src fetch origin main
git -C src checkout --force --detach FETCH_HEAD
docker compose build app

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

See `deploy/fail2ban/` for the ready-made jail and filter definitions.

### 3. Regular Updates

```bash
# Update system packages
sudo apt update && sudo apt upgrade  # Ubuntu/Debian
sudo dnf update  # RHEL-based

# Rebuild the application image from the latest sources
cd /opt/family-budget
git -C src fetch origin main && git -C src checkout --force --detach FETCH_HEAD
docker compose build app

# Pull the sidecar images (nginx/caddy/certbot are still pulled from upstream)
docker compose pull --ignore-buildable
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

- **Documentation:** this file, plus `docs/` in the repository root
- **Issues:** GitHub Issues
- **Security:** Report security issues privately

## License

See LICENSE file in repository root.
