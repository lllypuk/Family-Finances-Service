# Family Finances Service

**Modern family budget management system** with full-featured web interface, REST API, and advanced security features.

## 🎯 Project Status: IN DEVELOPMENT 🚧

This project is in **active development** with the following achievements:

- ✅ Complete web interface (HTMX v2.0.4 + PicoCSS v2.1.1)
- ✅ REST API for core entities (users, categories, transactions, budgets, invites, backups)
- ✅ Advanced security (authentication, authorization, CSRF protection)
- ✅ **Invite system** — user onboarding via secure token links
- ✅ **Backup management** — create, download, restore, and auto-cleanup
- ✅ **Admin panel** — user and invite management
- ✅ **Lightweight SQLite database** for simple deployment
- ✅ CI/CD pipelines with GitHub Actions
- ✅ **Single Docker container**, built from source (`docker/Dockerfile`)
- 🚧 Multi-platform builds (linux/amd64, linux/arm64) — the workflow exists, but no
  release has been tagged yet, so nothing is published to GHCR. Every compose file
  builds locally instead of pulling; see
  [docs/specs/004-deployment-readiness.md](docs/specs/004-deployment-readiness.md#d-02)

## 🚀 Features

- 📊 **Complete Web Interface**: Modern UI based on HTMX with responsive design
- 👥 **Role-Based Access Control**: Family Admin, Member, Child with different permissions
- 💰 **Advanced Budget Management**: Category limits, period tracking, overspending alerts
- 📈 **Real-Time Analytics**: Interactive dashboards with live updates
- 🎯 **Financial Goals Tracking**: Savings goals with progress visualization
- 🔐 **Enterprise Security**: Session management, CSRF protection, input validation
- 📊 **Reporting (mixed readiness)**: Web UI supports preview/save/view/delete and CSV export for reports; public report-generation REST API is still in progress
- 📨 **Invite System**: Secure registration via links with role control and expiration
- 💾 **Backup Management**: Create, download, restore DB with auto-cleanup (up to 10 backups)
- 🛠️ **Admin Panel**: User, invite, and backup management
- 🌐 **Multi-Platform Ready**: REST API, web interface, mobile-ready design

## API Readiness (Ready / Experimental)

### Authentication is required — read this first

The whole `/api/v1` group sits behind `RequireAPIAuth`. There is no anonymous access
and there are no API tokens yet: **the only credential is the web session cookie**.

| Request | Response |
|---|---|
| Any `/api/v1/*` without a session | `401` + `{"error":{"code":"UNAUTHORIZED","message":"Authentication required"}}` |
| `POST`/`PUT`/`DELETE` without `X-Csrf-Token` | `403 CSRF token validation failed` (global CSRF middleware runs first) |
| `POST`/`PUT`/`DELETE` with a valid token but no session | `401` |
| Role not allowed for the route | `403` + `{"error":{"code":"FORBIDDEN","message":"Insufficient permissions"}}` |

Role model mirrors the web UI:

- `POST`/`PUT`/`DELETE /api/v1/users` and `DELETE /api/v1/categories/:id` — **admin only**
- `/api/v1/{categories,transactions,budgets,reports}` — **admin or member** (`child` gets `403`)
- `GET /api/v1/users/:id` — any authenticated user

The author of a record is taken from the session: `user_id` is no longer part of
`CreateTransactionRequest` / `CreateReportRequest`, so sending it in the body has no effect.

A programmatic client therefore has to keep a cookie jar and fetch a CSRF token from a
rendered page — see [`deploy/README.md`](deploy/README.md#programmatic-api-clients) for a
working `curl` walkthrough.

### Ready (current behavior)

- Core REST API for users, categories, transactions, budgets — session-authenticated, role-gated
- Invite, admin, and backup management APIs/web flows
- Stored reports API endpoints: list, get by ID, delete
- Web reports UI: generate preview/save/view/delete/export CSV for expense, income, budget, cash-flow, and category-breakdown reports
- Web UI for day-to-day finance workflows

### Experimental / In Progress

- **No API tokens.** Session cookies only — fine for a browser or a scripted cookie jar,
  awkward for a headless integration. Token auth is a separate, not-yet-written plan
- `POST /api/v1/reports` currently returns `501 Not Implemented` (report generation API is not exposed yet)
- Advanced analytics/report-generation features described in roadmap-style text are not fully available via public API
- Scheduled reports, forecasts, insights, and benchmark analytics remain hidden/placeholder service capabilities
- Treat "comprehensive reporting" as partial readiness: storage/retrieval is available, generation is pending

## 🏗️ Architecture and Technology Stack

### Backend (Production Ready)

- **Go 1.26.0** with Echo v4.15.0 framework
- **SQLite** (modernc.org/sqlite) - Pure Go, no CGO dependencies
- **Automatic migrations** on application startup
- **Clean Architecture** with domain-driven design
- **Repository pattern** with comprehensive error handling
- **Structured logging** with slog

### Frontend (Modern Web Interface)

- **HTMX v2.0.4** for dynamic updates without complex JavaScript
- **PicoCSS v2.1.1** minimalist CSS framework
- **Go Templates** with layout system and components
- **Progressive Web App** capabilities
- **Responsive design** for mobile and desktop

### DevOps and Quality

- **Single Docker container** (~50MB) for simple deployment
- **GitHub Actions** CI/CD with security scanning
- **Multi-platform builds** (linux/amd64, linux/arm64)
- **Fast testing** with in-memory SQLite (no Docker)
- **Security scanning** (CodeQL, Semgrep, TruffleHog)

### Monitoring and Reliability

- **Health check endpoint** for Docker
- **Structured logging** (JSON/text formats)
- **Graceful shutdown** with signal handling
- **Persistent storage** via Docker volumes

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
# 1. Create .env file in the repository root
cp .env.example .env
# Set SESSION_SECRET and CSRF_SECRET — both are required, compose won't start without them:
#   openssl rand -base64 32

# 2. Start container (make docker-up-d does the same)
docker compose --project-directory . -f docker/docker-compose.yml up -d

# 3. Open in browser
# http://localhost:8080
```

**Done!** All data is saved in `./data/budget.db`

### Option 2: Local Development

**Prerequisites:**

- Go 1.26.0+
- Make (optional)

```bash
# 1. Run application
make run-local  # or: go run ./cmd/server/main.go

# 2. Open in browser
# http://localhost:8080
```

**Database** is created automatically in `./data/budget.db`

### 📋 Development Commands

```bash
# Run and build
make run-local        # Run with local SQLite DB
make build            # Build binary
make clean            # Clean build artifacts

# Testing (⚡ fast with in-memory SQLite)
make test             # Run all tests
make test-coverage    # Tests with coverage report
make test-unit        # Unit tests only
make test-integration # Integration tests only

# Code quality
make lint             # Linter (golangci-lint)
make fmt              # Format code
make pre-commit       # Full pre-commit check

# Docker
make docker-up        # Run in Docker
make docker-down      # Stop container
make docker-logs      # View logs

# SQLite database
make sqlite-backup    # Create backup
make sqlite-restore   # Restore from backup
make sqlite-shell     # Open SQLite shell
make sqlite-stats     # DB statistics

# Development
make migrate-create   # Create new migration
make help             # Show all commands
```

## 🏛️ Architecture

The project follows **Clean Architecture** principles with production-ready implementations:

### Layer Structure

- **Domain layer** (`internal/domain/`): Business entities with comprehensive validation (User, Family, Invite,
  Transaction, Budget, etc.)
- **Services layer** (`internal/services/`): Business logic (invite, backup, budget, category, transaction, report,
  user)
- **Application layer** (`internal/application/`): HTTP server and handler orchestration
- **Web layer** (`internal/web/`): HTMX templates, middleware, authentication, admin panel
- **Infrastructure layer** (`internal/infrastructure/`): SQLite repositories and data persistence
- **Observability layer** (`internal/observability/`): Logging and health checks

### Project Structure

```
├── cmd/server/              # Application entry point with health checks
├── internal/
│   ├── domain/              # Business entities (User, Family, Invite, Transaction, Budget, Report)
│   ├── application/         # HTTP server, handlers, repository interfaces
│   ├── services/            # Business logic (invite, backup, budget, category, transaction, etc.)
│   ├── web/                 # Complete web interface
│   │   ├── handlers/        # Authentication, dashboard, admin, backups, HTMX endpoints
│   │   ├── middleware/      # Sessions, CSRF, authorization guards
│   │   ├── templates/       # HTML templates with layouts and admin pages
│   │   ├── static/          # CSS, JS, images
│   │   └── models/          # Form validation and web-specific structures
│   ├── infrastructure/      # SQLite repositories and connection management
│   ├── observability/       # Production monitoring and logging
│   └── testhelpers/         # Testing utilities and factories
├── tests/                   # Integration tests and benchmarks
│   ├── integration/        # Cross-component integration tests
│   └── benchmarks/         # Load testing and benchmarks
├── docs/                   # Project documentation, audits, and plans
├── deploy/                 # Self-hosted deployment configs and scripts
├── docker/                 # Docker Compose configurations
└── .github/workflows/      # CI/CD pipelines (ci, docker, security, release)
```

### Production Components

- **Authentication and Authorization**: Role-based access with session management
- **Invite System**: Secure tokens, expiration, role control
- **Backup Management**: Create, restore, download with auto-cleanup
- **Admin Panel**: User, invite, and backup management
- **Reports (UI-first)**: interactive preview generation and CSV export for saved reports
- **Data Validation**: Comprehensive input validation with go-playground/validator
- **Error Handling**: Structured error responses with proper HTTP status codes
- **Security**: CSRF protection, password hashing, input sanitization, path traversal protection
- **Testing**: 73+ test files across all layers
- **Observability**: Structured logging (slog), health check endpoint
- **Deployment**: Multi-platform Docker builds with GitHub Actions CI/CD

## Configuration

The application uses environment variables for configuration. Key variables:

| Variable               | Default                                              | Description                                           |
|------------------------|------------------------------------------------------|-------------------------------------------------------|
| `SERVER_HOST`          | `localhost`                                          | HTTP server host                                      |
| `SERVER_PORT`          | `8080`                                               | HTTP server port                                      |
| `SERVER_READ_TIMEOUT`  | `15s`                                                | HTTP server read timeout                              |
| `SERVER_WRITE_TIMEOUT` | `15s`                                                | HTTP server write timeout                             |
| `SERVER_IDLE_TIMEOUT`  | `60s`                                                | HTTP server idle timeout                              |
| `DATABASE_PATH`        | `./data/budget.db`                                   | SQLite database file path                             |
| `ENVIRONMENT`          | `development`                                        | App environment (`development`, `production`, `test`) |
| `LOG_LEVEL`            | `info`                                               | Logging level                                         |
| `LOG_FORMAT`           | `json`                                               | Log format                                            |
| `LOG_OUTPUT_PATH`      | `stdout`                                             | Log output destination                                |
| `SESSION_SECRET`       | insecure dev default (change in production)          | Session encryption key                                |
| `SESSION_TIMEOUT`      | `24h`                                                | Session lifetime                                      |
| `CSRF_SECRET`          | insecure dev default (change in production)          | CSRF signing secret                                   |
| `COOKIE_SECURE`        | `false` (forced `true` in production)                | Secure cookie flag                                    |
| `COOKIE_HTTP_ONLY`     | `true`                                               | HttpOnly cookie flag                                  |
| `COOKIE_SAME_SITE`     | `Lax`                                                | SameSite cookie mode                                  |

## Running with Docker

`SESSION_SECRET` and `CSRF_SECRET` are mandatory: the compose file declares them as
`${VAR:?...}`, so an unset or empty value aborts the command with an explicit message
instead of silently booting with a well-known default.

`.env` belongs in the **repository root**. Compose v2 resolves `.env` (and relative
paths such as `DATA_DIR`) against the *project directory*, which defaults to the
directory of the first `-f` file — `docker/`. `--project-directory .` moves it back to
the root, which is why every command below (and every `make docker-*` target) passes it.

```bash
# Build and start all services
docker compose --project-directory . -f docker/docker-compose.yml up --build

# Run in background
docker compose --project-directory . -f docker/docker-compose.yml up -d

# Stop services
docker compose --project-directory . -f docker/docker-compose.yml down
```

## Development

```bash
# Run application locally
make run-local
```

## 🧪 Testing and Quality

The project maintains **high quality standards** with comprehensive testing:

### Testing

- **73+ test files** across all application layers
- **Unit tests**: Domain models, services (invite, backup, etc.), repositories, middleware
- **Web tests**: Handlers (admin, auth, backup, budgets, categories, dashboard, reports, transactions, users)
- **Models**: Form validation (categories, budgets, forms, dashboard, reports, transactions)
- **Integration tests**: SQLite integration tests with in-memory database
- **Service tests**: Invite service, backup service, budget, category, transaction, report

### Quality Control

- **golangci-lint** with 50+ linters for code quality
- **Comprehensive CI/CD** with GitHub Actions
- **Security scanning** (CodeQL, Semgrep, TruffleHog, OSV Scanner)
- **Dependency management** with automated Dependabot updates
- **Multi-platform testing** (linux/amd64, linux/arm64)

```bash
# Run comprehensive test suite
make test              # All tests
make test-coverage    # With coverage report
make lint             # Code quality checks
```

## 📊 Production Readiness

### Deployment Readiness

- ✅ **Multi-platform Docker images** published to GitHub Container Registry
- ✅ **Docker-ready** with health checks and graceful shutdown
- ✅ **Environment configuration** with validation and defaults
- ✅ **DB connection management** and connection pooling
- ✅ **Logging and health check** for monitoring

### Monitoring and Observability

- ✅ **Health checks** - `/health` endpoint for container orchestration
- ✅ **Structured logging** - slog with configurable levels

### Security Features

- ✅ **Role-Based Access Control** (Admin, Member, Child)
- ✅ **Session security** with HTTP-only cookies and CSRF protection
- ✅ **Input validation and sanitization** for all endpoints
- ✅ **Password security** with bcrypt hashing
- ✅ **Secure invite tokens** (cryptographically strong, 32 bytes, 7-day expiration)
- ✅ **Backup protection** from path traversal with filename validation
- ✅ **Security headers** and modern security practices

## 🏠 Self-Hosted Deployment

The project includes **complete deployment infrastructure** for installation on your own server with enterprise-grade
automation, security, and monitoring.

### ⚡ Quick Deployment (One Command)

```bash
# Automatic installation on fresh Linux VM
curl -fsSL https://raw.githubusercontent.com/lllypuk/Family-Finances-Service/main/deploy/scripts/install.sh | sudo bash

# Or clone and run
git clone https://github.com/lllypuk/Family-Finances-Service.git
cd Family-Finances-Service
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

### 🖥️ Supported Operating Systems

- ✅ Ubuntu 22.04 LTS / 24.04 LTS
- ✅ Debian 11 / 12
- ✅ Rocky Linux 9 / AlmaLinux 9

### 🎯 Deployment Options

| Option             | Description       | Features                          | Complexity   |
|--------------------|-------------------|-----------------------------------|--------------|
| **Docker + Caddy** | Automatic SSL     | HTTP/3, zero-config SSL           | ⭐ Simple     |
| **Docker + Nginx** | Traditional stack | Flexible configuration, Certbot   | ⭐⭐ Medium    |
| **Native Systemd** | Without Docker    | Direct control, minimal resources | ⭐⭐⭐ Advanced |

### 🔒 Security Features

**Automatically configured during installation:**

- 🔐 **TLS/SSL** — automatic Let's Encrypt certificates with auto-renewal
- 🛡️ **Rate Limiting** — 5 attempts/min for login, brute-force protection
- 🔥 **Firewall** — UFW/firewalld with blocked direct app port access
- 🚫 **Fail2ban** — automatic IP blocking after failed login attempts (5 attempts → 1 hour ban)
- 🔑 **Security Headers** — CSP, XSS Protection, HSTS, Referrer Policy
- 📊 **Health Monitoring** — health checks for monitoring

### 🛠️ Deployment Scripts

**Main operations:**

```bash
# Installation (automatic)
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com

# Upgrade with automatic rollback
sudo ./deploy/scripts/upgrade.sh

# Upgrade to specific version
sudo ./deploy/scripts/upgrade.sh --version v1.2.3

# Rollback to previous version
sudo ./deploy/scripts/upgrade.sh rollback

# Uninstall with data preservation
sudo ./deploy/scripts/uninstall.sh --keep-data

# Create database backup
sudo ./deploy/scripts/backup.sh

# Setup fail2ban protection
sudo ./deploy/scripts/setup-fail2ban.sh
```

**Available scripts (deploy/scripts/):**

- ✅ `install.sh` — complete automatic installation
- ✅ `upgrade.sh` — safe upgrade with rollback
- ✅ `uninstall.sh` — clean removal
- ✅ `backup.sh` — DB backup with integrity verification
- ✅ `health-check.sh` — health monitoring
- ✅ `setup-ssl-nginx.sh` — SSL for Nginx
- ✅ `setup-ssl-caddy.sh` — SSL for Caddy (automatic)
- ✅ `setup-fail2ban.sh` — brute-force protection

### 📦 Deployment Configurations

**Docker Compose files:**

- `deploy/docker-compose.prod.yml` — standalone without reverse proxy
- `deploy/docker-compose.nginx.yml` — with Nginx + Certbot
- `deploy/docker-compose.caddy.yml` — with Caddy (automatic SSL)
- `deploy/docker-compose.minimal.yml` — for testing

**Reverse Proxy configurations:**

- `deploy/nginx/*` — 5 Nginx configuration files
- `deploy/caddy/*` — Caddy configuration with auto-SSL

**Systemd integration:**

- `deploy/systemd/family-budget.service` — main service
- `deploy/systemd/family-budget-backup.service` — backup service
- `deploy/systemd/family-budget-backup.timer` — daily backups at 3:00 AM

**Fail2ban protection:**

- `deploy/fail2ban/family-budget.conf` — filter for attack detection
- `deploy/fail2ban/jail.local` — jail configuration

### 🔧 Automation and Operations

**What's automated:**

✅ **Installation:**

- System requirements check (2GB RAM, 10GB disk)
- Docker and dependencies installation
- Firewall setup (SSH, HTTP, HTTPS allowed; port 8080 blocked)
- Cryptographically strong secret generation
- System user creation with proper permissions
- Application deployment
- Health check verification

✅ **Backup:**

- Daily automatic backups at 3:00 AM (systemd timer)
- SQLite integrity verification after creation
- Storage of up to 50 backups or 30 days
- Automatic old backup cleanup

✅ **Upgrade:**

- Current version check
- Pre-upgrade backup creation
- Service stop
- Upgrade to new version
- Health check verification
- **Automatic rollback** on failure

✅ **Security:**

- TLS 1.2+ only (no legacy protocols)
- Strong cipher suites (ECDHE, AES-GCM)
- Perfect Forward Secrecy
- Automatic SSL certificate renewal

### 📚 Deployment Documentation

**Complete documentation in `deploy/` directory:**

- 📖 **[deploy/README.md](deploy/README.md)** — comprehensive guide (10KB+)
    - Quick start
    - All deployment options
    - Security configuration
    - Common operations
    - Troubleshooting
    - Performance

**Task specifications in `docs/tasks/`:**

- ✅ [001: Install Script](docs/tasks/001-install-script.md) — **COMPLETE**
- ✅ [002: Reverse Proxy Config](docs/tasks/002-reverse-proxy-config.md) — **COMPLETE**
- ✅ [003: Production Docker Compose](docs/tasks/003-docker-compose-production.md) — **COMPLETE**
- ✅ [004: Systemd Services](docs/tasks/004-systemd-service.md) — **COMPLETE**
- ✅ [005: Upgrade Script](docs/tasks/005-upgrade-script.md) — **COMPLETE**
- ✅ [006: Security Hardening](docs/tasks/006-security-hardening.md) — **COMPLETE**
- ✅ [007: Deployment Documentation](docs/tasks/007-deployment-documentation.md) — **COMPLETE**
- ✅ [008: Uninstall Script](docs/tasks/008-uninstall-script.md) — **COMPLETE**

### 🎯 Deployment Statistics

- **30+ files** for deployment
- **10 executable bash scripts**
- **13 configuration files**
- **~20,000 lines** of automation code
- **100% coverage** of deployment tasks
- **6 supported OS** (Ubuntu, Debian, Rocky/Alma)
- **3 deployment options** (Docker+Nginx, Docker+Caddy, Native)

## 🚧 Known Issues and TODO

### Test Coverage Status

- ✅ **Web handlers**: Comprehensive coverage implemented (admin, auth, backup, budgets, categories, dashboard, reports,
  transactions, users)
- ✅ **Web models**: Extended validation tests added (categories, budgets, forms, dashboard, reports, transactions)
- ✅ **Admin/Backup handlers**: Full test coverage implemented
- ✅ **DTOs**: Comprehensive validation tests added

### Development Priorities

1. ~~Implement self-hosted deployment scripts~~ ✅ **COMPLETE** (8/8 tasks, 100%)
2. Deployment testing on real VMs (Ubuntu 22.04/24.04, Debian 11/12)
3. Add more integration test scenarios for invite flow
4. Performance optimization and benchmarking
5. End-to-end testing with agent-browser
6. Load testing and stress testing

## 📚 Documentation

### Developer Resources

- **[CLAUDE.md](CLAUDE.md)** - Comprehensive development and architecture guidance
- **[docs/](docs/README.md)** - Detailed project documentation including:
    - Product brief and business context
    - Technical architecture and design decisions
    - Testing strategy and implementation details
    - Audit findings (`docs/specs/`) and implementation plans (`docs/plans/`)
- **[deploy/README.md](deploy/README.md)** - Self-hosted deployment guide:
    - Installation and upgrade scripts
    - Nginx/Caddy configurations for TLS/SSL
    - Systemd services for native deployment
    - Security hardening (firewall, fail2ban)

### API Documentation

- **REST API** covering users, categories, transactions, budgets, and stored reports
- Request/response shapes live in `internal/application/handlers/types.go`; the error
  envelope is in `internal/application/handlers/errors.go`
- There is **no** OpenAPI spec, `/api/docs` endpoint, or Postman collection yet

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
