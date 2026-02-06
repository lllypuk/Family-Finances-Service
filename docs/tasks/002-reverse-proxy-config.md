# Task 002: Reverse Proxy Configuration

## Overview

Create production-ready reverse proxy configurations for both Nginx and Caddy to provide TLS/SSL termination, security
headers, and rate limiting.

## Priority: HIGH

## Status: COMPLETE

## Completed Items

- [x] Created Nginx configuration files:
  - `deploy/nginx/nginx.conf` - Main Nginx configuration
  - `deploy/nginx/conf.d/family-budget.conf.template` - Site configuration with HTTP->HTTPS redirect
  - `deploy/nginx/snippets/ssl-params.conf` - Modern SSL/TLS settings (TLSv1.2+)
  - `deploy/nginx/snippets/security-headers.conf` - Security headers (CSP, XSS, etc.)
  - `deploy/nginx/snippets/proxy-params.conf` - Proxy headers and settings

- [x] Created Caddy configuration:
  - `deploy/caddy/Caddyfile.template` - Complete Caddy configuration with automatic SSL
  - Automatic certificate management from Let's Encrypt
  - Built-in rate limiting and security headers
  - HTTP/3 support

- [x] Created Docker Compose files:
  - `deploy/docker-compose.nginx.yml` - Production setup with Nginx + Certbot
  - `deploy/docker-compose.caddy.yml` - Production setup with Caddy
  - Network isolation (internal/external networks)
  - Security hardening (no-new-privileges, minimal capabilities)

- [x] Created SSL setup scripts:
  - `deploy/scripts/setup-ssl-nginx.sh` - Automated SSL setup for Nginx
  - `deploy/scripts/setup-ssl-caddy.sh` - Automated SSL setup for Caddy
  - DH parameters generation (4096-bit)
  - Certificate verification

## Implementation Details

### Security Features Implemented

✅ **SSL/TLS Configuration:**
- TLS 1.2 and 1.3 only (no outdated protocols)
- Strong cipher suites (ECDHE, AES-GCM)
- OCSP stapling enabled
- DH parameters support (4096-bit)
- Perfect Forward Secrecy

✅ **Security Headers:**
- X-Frame-Options (clickjacking protection)
- X-Content-Type-Options (MIME sniffing protection)
- X-XSS-Protection (XSS filter)
- Content-Security-Policy (script injection protection)
- Referrer-Policy (privacy)
- Permissions-Policy (feature restrictions)
- HSTS ready (commented out for initial deployment)

✅ **Rate Limiting:**
- Login endpoints: 5 requests/minute (brute force protection)
- API endpoints: 100 requests/minute
- General endpoints: 10 requests/second with burst
- Health check: unlimited (for monitoring)

✅ **Network Security:**
- Application not exposed directly (reverse proxy only)
- Internal network isolation
- External network for internet access only

### Remaining Items

## Requirements

### Nginx Configuration

#### Main Config (`deploy/nginx/nginx.conf`)

```nginx
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging format
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;

    # Performance
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Security
    server_tokens off;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json
               application/javascript application/xml;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=general:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

    # Include site configs
    include /etc/nginx/conf.d/*.conf;
}
```

#### Site Config (`deploy/nginx/family-budget.conf`)

```nginx
# Redirect HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

# HTTPS Server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${DOMAIN};

    # SSL certificates (Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;

    # SSL configuration
    include /etc/nginx/snippets/ssl-params.conf;

    # Security headers
    include /etc/nginx/snippets/security-headers.conf;

    # Logging
    access_log /var/log/nginx/family-budget.access.log main;
    error_log /var/log/nginx/family-budget.error.log;

    # Max upload size (for backup restore)
    client_max_body_size 100M;

    # Rate limiting for login
    location /login {
        limit_req zone=login burst=5 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include /etc/nginx/snippets/proxy-params.conf;
    }

    # Rate limiting for API
    location /api/ {
        limit_req zone=general burst=20 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include /etc/nginx/snippets/proxy-params.conf;
    }

    # Health check (no rate limit)
    location /health {
        proxy_pass http://127.0.0.1:8080;
        include /etc/nginx/snippets/proxy-params.conf;
    }

    # Static files with caching
    location /static/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_cache_valid 200 1d;
        add_header Cache-Control "public, max-age=86400";
        include /etc/nginx/snippets/proxy-params.conf;
    }

    # Default location
    location / {
        limit_req zone=general burst=20 nodelay;
        proxy_pass http://127.0.0.1:8080;
        include /etc/nginx/snippets/proxy-params.conf;
    }
}
```

#### SSL Parameters (`deploy/nginx/snippets/ssl-params.conf`)

```nginx
# Modern SSL configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
ssl_prefer_server_ciphers off;

# SSL session
ssl_session_timeout 1d;
ssl_session_cache shared:SSL:50m;
ssl_session_tickets off;

# OCSP Stapling
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;

# DH parameters
ssl_dhparam /etc/nginx/dhparam.pem;
```

#### Security Headers (`deploy/nginx/snippets/security-headers.conf`)

```nginx
# Security headers
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'self';" always;
add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;

# HSTS (uncomment after confirming SSL works)
# add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
```

#### Proxy Parameters (`deploy/nginx/snippets/proxy-params.conf`)

```nginx
proxy_http_version 1.1;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Host $host;
proxy_set_header X-Forwarded-Port $server_port;
proxy_connect_timeout 60s;
proxy_send_timeout 60s;
proxy_read_timeout 60s;
proxy_buffering on;
proxy_buffer_size 4k;
proxy_buffers 8 4k;
```

---

### Caddy Configuration

#### Caddyfile (`deploy/caddy/Caddyfile`)

```caddyfile
# Global options
{
    email {$ACME_EMAIL}

    # Staging for testing (uncomment to avoid rate limits)
    # acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
}

# Main site
{$DOMAIN} {
    # Enable compression
    encode gzip

    # Security headers
    header {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
        Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'self';"
        Permissions-Policy "geolocation=(), microphone=(), camera=()"
        -Server
    }

    # Rate limiting for login (5 requests per minute)
    @login {
        path /login
        method POST
    }
    rate_limit @login {
        zone login {
            key {remote_host}
            events 5
            window 1m
        }
    }

    # Rate limiting for general requests
    rate_limit {
        zone general {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # Static files caching
    @static {
        path /static/*
    }
    header @static Cache-Control "public, max-age=86400"

    # Health check (no logging)
    @health {
        path /health
    }
    log @health {
        output discard
    }

    # Proxy to application
    reverse_proxy localhost:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}

        # Health checks
        health_uri /health
        health_interval 30s
        health_timeout 5s
    }

    # Logging
    log {
        output file /var/log/caddy/access.log {
            roll_size 100mb
            roll_keep 5
        }
    }
}
```

---

### Docker Integration

#### docker-compose.prod.yml with Nginx

```yaml
version: '3.8'

services:
  app:
    image: ghcr.io/lllypuk/family-finances-service:latest
    container_name: family-budget-app
    restart: unless-stopped
    environment:
      - DATABASE_PATH=/data/budget.db
      - SESSION_SECRET=${SESSION_SECRET}
      - ENVIRONMENT=production
      - LOG_LEVEL=info
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
    volumes:
      - budget_data:/data
    networks:
      - internal
    healthcheck:
      test: [ "CMD", "wget", "-q", "--spider", "http://localhost:8080/health" ]
      interval: 30s
      timeout: 3s
      retries: 3

  nginx:
    image: nginx:alpine
    container_name: family-budget-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/snippets:/etc/nginx/snippets:ro
      - ./nginx/dhparam.pem:/etc/nginx/dhparam.pem:ro
      - certbot_www:/var/www/certbot:ro
      - certbot_conf:/etc/letsencrypt:ro
    depends_on:
      - app
    networks:
      - internal
      - external

  certbot:
    image: certbot/certbot
    container_name: family-budget-certbot
    volumes:
      - certbot_www:/var/www/certbot
      - certbot_conf:/etc/letsencrypt
    entrypoint: "/bin/sh -c 'trap exit TERM; while :; do certbot renew; sleep 12h & wait $${!}; done;'"

volumes:
  budget_data:
  certbot_www:
  certbot_conf:

networks:
  internal:
    driver: bridge
  external:
    driver: bridge
```

---

## Security Considerations

### Rate Limiting Strategy

| Endpoint  | Limit     | Burst | Reason                 |
|-----------|-----------|-------|------------------------|
| `/login`  | 5/min     | 5     | Brute force protection |
| `/api/*`  | 10/sec    | 20    | API abuse prevention   |
| `/`       | 10/sec    | 20    | General protection     |
| `/health` | unlimited | -     | Monitoring             |

### SSL/TLS Best Practices

1. TLS 1.2+ only (no SSLv3, TLS 1.0, TLS 1.1)
2. Strong cipher suites (ECDHE, AES-GCM)
3. OCSP stapling enabled
4. DH parameters 4096-bit
5. Session resumption with secure settings

### Headers Checklist

- [x] X-Frame-Options (clickjacking protection)
- [x] X-Content-Type-Options (MIME sniffing protection)
- [x] X-XSS-Protection (XSS filter)
- [x] Content-Security-Policy (script injection protection)
- [x] Referrer-Policy (privacy)
- [x] Permissions-Policy (feature restrictions)
- [ ] HSTS (enable after SSL verification)

---

## Files to Create

```
deploy/
├── nginx/
│   ├── nginx.conf
│   ├── conf.d/
│   │   └── family-budget.conf.template
│   └── snippets/
│       ├── ssl-params.conf
│       ├── security-headers.conf
│       └── proxy-params.conf
├── caddy/
│   └── Caddyfile.template
└── docker-compose.prod.yml
```

### Remaining Items

- [ ] Testing on actual deployment (requires live server with domain)

## Testing Checklist

- [ ] HTTP redirects to HTTPS
- [ ] SSL Labs score A or A+
- [ ] Security headers present (check with securityheaders.com)
- [ ] Rate limiting works (test with `ab` or `wrk`)
- [ ] Let's Encrypt certificate auto-renewal
- [ ] Static files cached correctly
- [ ] WebSocket connections work (for HTMX)
- [ ] Large file upload works (backup restore - 100MB limit)

## Usage Guide

### Option 1: Nginx (Traditional)

1. Use `docker-compose.nginx.yml` instead of `docker-compose.prod.yml`
2. Run `./deploy/scripts/setup-ssl-nginx.sh --domain your-domain.com --email your@email.com`
3. Nginx will handle SSL termination with Certbot for certificate management

**Pros:** Well-known, mature, extensive documentation
**Cons:** Requires certbot for SSL, more configuration files

### Option 2: Caddy (Modern)

1. Use `docker-compose.caddy.yml` instead of `docker-compose.prod.yml`
2. Run `./deploy/scripts/setup-ssl-caddy.sh --domain your-domain.com --email your@email.com`
3. Caddy automatically obtains and renews SSL certificates

**Pros:** Automatic SSL, simpler configuration, HTTP/3 support
**Cons:** Less familiar to some administrators

Both configurations provide the same security level and features.

## Acceptance Criteria

1. Both Nginx and Caddy configs work out of the box
2. SSL/TLS configuration scores A+ on SSL Labs
3. All security headers present and correct
4. Rate limiting prevents brute force attacks
5. Automatic certificate renewal works
6. Documentation explains both options

## Dependencies

- Task 001 (install script) - will use these configs
- Task 003 (docker-compose.prod.yml) - integrates with proxy

## Estimated Complexity

Medium (well-documented patterns exist)
