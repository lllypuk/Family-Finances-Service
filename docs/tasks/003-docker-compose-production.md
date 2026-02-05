# Task 003: Production Docker Compose Configuration

## Overview

Create a production-ready Docker Compose configuration that includes the application, reverse proxy, SSL certificate management, and optional monitoring stack.

## Priority: HIGH

## Status: TODO

## Requirements

### Main Production Compose (`deploy/docker-compose.prod.yml`)

```yaml
version: '3.8'

services:
  # Main application
  app:
    image: ghcr.io/lllypuk/family-finances-service:${APP_VERSION:-latest}
    container_name: family-budget-app
    restart: unless-stopped
    user: "65534:65534"  # nobody user
    read_only: true
    security_opt:
      - no-new-privileges:true
    environment:
      - DATABASE_PATH=/data/budget.db
      - SESSION_SECRET=${SESSION_SECRET:?SESSION_SECRET is required}
      - CSRF_SECRET=${CSRF_SECRET:-}
      - ENVIRONMENT=production
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
    volumes:
      - app_data:/data
      - app_backups:/backups
    tmpfs:
      - /tmp:mode=1777,size=64M
    networks:
      - backend
    healthcheck:
      test: [ "CMD", "wget", "-q", "--spider", "http://localhost:8080/health" ]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 128M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  # Nginx reverse proxy
  nginx:
    image: nginx:1.25-alpine
    container_name: family-budget-nginx
    restart: unless-stopped
    read_only: true
    security_opt:
      - no-new-privileges:true
    ports:
      - "${HTTP_PORT:-80}:80"
      - "${HTTPS_PORT:-443}:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/snippets:/etc/nginx/snippets:ro
      - ./nginx/dhparam.pem:/etc/nginx/dhparam.pem:ro
      - certbot_www:/var/www/certbot:ro
      - certbot_conf:/etc/letsencrypt:ro
      - nginx_cache:/var/cache/nginx
    tmpfs:
      - /var/run:mode=1777
      - /tmp:mode=1777
    depends_on:
      app:
        condition: service_healthy
    networks:
      - frontend
      - backend
    healthcheck:
      test: [ "CMD", "wget", "-q", "--spider", "http://localhost/health" ]
      interval: 30s
      timeout: 5s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"

  # Certbot for Let's Encrypt
  certbot:
    image: certbot/certbot:latest
    container_name: family-budget-certbot
    volumes:
      - certbot_www:/var/www/certbot
      - certbot_conf:/etc/letsencrypt
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew --quiet; sleep 12h & wait $${!}; done;'"
    depends_on:
      - nginx

volumes:
  app_data:
    name: family-budget-data
  app_backups:
    name: family-budget-backups
  certbot_www:
    name: family-budget-certbot-www
  certbot_conf:
    name: family-budget-certbot-conf
  nginx_cache:
    name: family-budget-nginx-cache

networks:
  frontend:
    name: family-budget-frontend
    driver: bridge
  backend:
    name: family-budget-backend
    driver: bridge
    internal: true  # No external access
```

---

### Alternative: Caddy Compose (`deploy/docker-compose.caddy.yml`)

```yaml
version: '3.8'

services:
  app:
    image: ghcr.io/lllypuk/family-finances-service:${APP_VERSION:-latest}
    container_name: family-budget-app
    restart: unless-stopped
    user: "65534:65534"
    read_only: true
    security_opt:
      - no-new-privileges:true
    environment:
      - DATABASE_PATH=/data/budget.db
      - SESSION_SECRET=${SESSION_SECRET:?SESSION_SECRET is required}
      - ENVIRONMENT=production
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
    volumes:
      - app_data:/data
      - app_backups:/backups
    tmpfs:
      - /tmp:mode=1777,size=64M
    networks:
      - backend
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

  caddy:
    image: caddy:2-alpine
    container_name: family-budget-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    environment:
      - DOMAIN=${DOMAIN:?DOMAIN is required}
      - ACME_EMAIL=${ACME_EMAIL:?ACME_EMAIL is required}
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    depends_on:
      app:
        condition: service_healthy
    networks:
      - frontend
      - backend

volumes:
  app_data:
    name: family-budget-data
  app_backups:
    name: family-budget-backups
  caddy_data:
    name: family-budget-caddy-data
  caddy_config:
    name: family-budget-caddy-config

networks:
  frontend:
    name: family-budget-frontend
  backend:
    name: family-budget-backend
    internal: true
```

---

### Environment File Template (`deploy/.env.production.example`)

```bash
# ===========================================
# Family Budget Service - Production Config
# ===========================================

# Application Version (use specific tag in production)
APP_VERSION=latest

# Domain Configuration
DOMAIN=budget.example.com
ACME_EMAIL=admin@example.com

# Security Secrets (CHANGE THESE!)
# Generate with: openssl rand -base64 32
SESSION_SECRET=CHANGE_ME_TO_RANDOM_32_BYTE_STRING
CSRF_SECRET=CHANGE_ME_TO_ANOTHER_RANDOM_STRING

# Server Configuration
HTTP_PORT=80
HTTPS_PORT=443
LOG_LEVEL=info

# Database
# Default location in Docker volume, change only if using bind mount
# DATABASE_PATH=/data/budget.db

# Backup Retention (optional)
BACKUP_RETENTION_DAYS=30

# Timezone
TZ=UTC
```

---

### Minimal Compose for Testing (`deploy/docker-compose.minimal.yml`)

For quick testing without SSL:

```yaml
version: '3.8'

services:
  app:
    image: ghcr.io/lllypuk/family-finances-service:latest
    container_name: family-budget-app
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - DATABASE_PATH=/data/budget.db
      - SESSION_SECRET=${SESSION_SECRET:-dev-secret-change-in-production}
      - ENVIRONMENT=development
      - LOG_LEVEL=debug
    volumes:
      - ./data:/data

# Usage:
# docker-compose -f docker-compose.minimal.yml up -d
# Access at http://localhost:8080
```

---

## Security Features

### Container Hardening

| Feature | Setting | Purpose |
|---------|---------|---------|
| `user: "65534:65534"` | nobody user | Non-root execution |
| `read_only: true` | Immutable filesystem | Prevent tampering |
| `no-new-privileges` | Block privilege escalation | Security |
| `tmpfs` | Temporary filesystem | No persistent temp files |
| `internal: true` | Backend network | Isolate app from internet |

### Resource Limits

| Resource | Limit | Reservation |
|----------|-------|-------------|
| CPU | 1 core | 0.25 core |
| Memory | 512MB | 128MB |

### Network Isolation

```
Internet
    │
    ▼
┌─────────┐    frontend network
│  Nginx  │◄────────────────────►
└────┬────┘
     │
     │         backend network (internal)
     ▼
┌─────────┐
│   App   │
└─────────┘
```

---

## Deployment Scenarios

### Scenario 1: Single Server with Nginx

```bash
cd /opt/family-budget
cp .env.production.example .env
# Edit .env with your values

# Generate DH parameters (one-time, takes a few minutes)
openssl dhparam -out nginx/dhparam.pem 4096

# Get initial SSL certificate
docker-compose -f docker-compose.prod.yml run --rm certbot certonly \
  --webroot -w /var/www/certbot \
  --email ${ACME_EMAIL} \
  -d ${DOMAIN} \
  --agree-tos --no-eff-email

# Start services
docker-compose -f docker-compose.prod.yml up -d
```

### Scenario 2: Single Server with Caddy (simpler)

```bash
cd /opt/family-budget
cp .env.production.example .env
# Edit .env with your values

# Start services (Caddy handles SSL automatically)
docker-compose -f docker-compose.caddy.yml up -d
```

### Scenario 3: Behind Existing Reverse Proxy

If you already have Traefik/nginx/Caddy:

```bash
# Use minimal compose, expose only to internal network
docker-compose -f docker-compose.minimal.yml up -d

# Configure your existing proxy to forward to localhost:8080
```

---

## Operations

### View Logs

```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Application only
docker-compose -f docker-compose.prod.yml logs -f app

# Last 100 lines
docker-compose -f docker-compose.prod.yml logs --tail=100 app
```

### Restart Services

```bash
# Graceful restart
docker-compose -f docker-compose.prod.yml restart app

# Full restart
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d
```

### Update Application

```bash
# Pull latest image
docker-compose -f docker-compose.prod.yml pull app

# Recreate container
docker-compose -f docker-compose.prod.yml up -d --no-deps app
```

### Backup Database

```bash
# Create backup
docker-compose -f docker-compose.prod.yml exec app \
  cp /data/budget.db /backups/budget_$(date +%Y%m%d_%H%M%S).db

# Or use Makefile
make sqlite-backup
```

---

## Files to Create

```
deploy/
├── docker-compose.prod.yml      # Full production with Nginx
├── docker-compose.caddy.yml     # Alternative with Caddy
├── docker-compose.minimal.yml   # Testing without SSL
├── .env.production.example      # Environment template
└── README.md                    # Deployment instructions
```

## Testing Checklist

- [ ] Containers start without errors
- [ ] Health checks pass
- [ ] Application accessible via HTTPS
- [ ] SSL certificate auto-renewal works
- [ ] Container restarts after failure
- [ ] Resource limits enforced
- [ ] Logs captured correctly
- [ ] Backup/restore works
- [ ] Update procedure works

## Acceptance Criteria

1. All compose files pass `docker-compose config` validation
2. Containers start with `--wait` successfully
3. Security scan (Trivy) passes with no HIGH/CRITICAL
4. Resource limits are respected
5. Network isolation prevents direct app access
6. Documentation covers all scenarios

## Dependencies

- Task 002 (nginx/caddy configs) - required for proxy
- Task 001 (install script) - uses these compose files

## Estimated Complexity

Medium
