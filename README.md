# Family Finances Service

**Self-hosted family budget backend**: a single Go binary with an embedded SQLite database that serves a JSON
API for the Android client. One instance = one family.

## 🎯 Project Status: IN DEVELOPMENT 🚧

> **Direction (September 2026):** API-only backend for an Android app. Decisions and the five implementation
> plans: [docs/specs/005-api-only-redesign.md](docs/specs/005-api-only-redesign.md). Plans 01–03 are done: the
> web interface, cookie sessions and CSRF are gone; the sections below describe the code as it is today.

- ✅ REST API for family, users, categories, transactions, budgets, reports, stats, backups
- ✅ Bearer-token authentication with server-side sessions and a login rate limiter
- ✅ Family bootstrap and password reset from the CLI (`setup`, `reset-password`)
- ✅ Lightweight SQLite database, migrations applied at startup
- ✅ CI/CD pipelines with GitHub Actions
- ✅ Single Docker container, built from source (`docker/Dockerfile`)
- 🚧 Multi-platform builds (linux/amd64, linux/arm64) — the workflow exists, but no release has been tagged yet,
  so nothing is published to GHCR. Every compose file builds locally instead of pulling; see
  [docs/specs/004-deployment-readiness.md](docs/specs/004-deployment-readiness.md#d-02)
- 🚧 Money is still `float64` and `deploy/` still targets the old web build — plans 04 and 05

## API

The **target** contract lives in [`docs/api/openapi.yaml`](docs/api/openapi.yaml) (OpenAPI 3.1, hand-written,
see [`docs/api/README.md`](docs/api/README.md)); the Android client generates from it. Authentication and the
error/pagination envelope already match it; money in minor units and calendar dates land in plan 04.

### Authentication

Two public routes: `GET /health` and `POST /api/v1/auth/login`. Everything else needs
`Authorization: Bearer <token>`.

| Request | Response |
|---|---|
| Login before `setup` has been run | `409 SETUP_REQUIRED` |
| Wrong email or password (same answer for both) | `401 INVALID_CREDENTIALS` |
| 11th attempt from one IP in 5 min, or 21st for one email in an hour | `429 RATE_LIMITED` + `Retry-After` |
| Any `/api/v1/*` without a token, or with an expired/revoked one | `401 UNAUTHORIZED` |
| Role not allowed for the route | `403 FORBIDDEN` |

Tokens are opaque (32 random bytes; the server stores only a SHA-256), slide 30 days without activity and expire
180 days after login regardless. `POST /auth/logout` revokes the current one, `GET`/`DELETE /auth/sessions`
manage the rest; changing your own password keeps the current session and revokes the others, an admin
password reset or deactivation revokes all.

Roles:

- `/api/v1/users` and `/api/v1/backups`, `PUT /api/v1/family`, `DELETE /api/v1/categories/:id` — **admin only**
- `/api/v1/{categories,transactions,budgets,reports,stats}` — **admin or member** (`child` gets `403`)
- `GET /api/v1/family`, `/api/v1/me*`, `/api/v1/auth/*` — any authenticated role

The author of a record is taken from the token, so `user_id` in a request body is ignored.

### Ready (current behavior)

- `POST /api/v1/reports` generates and stores a report (expense, income, budget, cash-flow,
  category-breakdown); `GET /api/v1/reports/:id/export` returns CSV
- `GET /api/v1/stats/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` — dashboard summary (totals, deltas to the
  previous period, top categories, budget progress, recent transactions); defaults to the current month
- `GET`/`PUT /api/v1/me`, `PUT /api/v1/me/password`; `PATCH /api/v1/users/:id` (`role`, `is_active`),
  `PUT /api/v1/users/:id/password`
- Backups over API: `POST`/`GET /api/v1/backups`, `GET /api/v1/backups/:name/download`,
  `DELETE /api/v1/backups/:name`
- `POST /api/v1/transactions/bulk-delete`
- Every list answers with `meta.pagination {limit, offset, total}` — `limit` defaults to 50, max 200
- One error envelope everywhere: `{"error":{"code","message","details"},"meta":{...}}`;
  validation fails with `422 VALIDATION_ERROR` and per-field `details`

### Not available yet

- Money is still `float64` (minor units land in plan 04, together with the rest of the `openapi.yaml` gap)
- Users are never deleted, only deactivated (`PATCH /users/:id {"is_active": false}`)
- Backup **restore** is deliberately not exposed over the API — use `make sqlite-restore`
- Invites have no route any more and are removed for good in plan 04

## 🏗️ Architecture and Technology Stack

- **Go 1.26** with Echo v4.15
- **SQLite** (modernc.org/sqlite) — pure Go, no CGO; migrations applied automatically at startup
- **Clean Architecture**: `domain` → `services` → repository interfaces → `infrastructure`; `internal/auth`
  (tokens, sessions, middleware, rate limiter) beside them
- **Structured logging** with slog, `/health` for orchestration, graceful shutdown
- **Single Docker container** (~50MB), in-memory SQLite for tests (no Docker needed)

## 🚀 Quick Start

### Option 1: Docker

```bash
cp .env.example .env          # no secrets to fill in; adjust TRUSTED_PROXIES if behind a reverse proxy
make docker-up-d              # = docker compose --project-directory . -f docker/docker-compose.yml up -d
```

### Option 2: Local development

```bash
make run-local                # localhost:8080, SQLite at ./data/budget.db
```

### Create the family and log in

The family and its first admin are created from the CLI, not over HTTP. Run it from the repo root with the same
`DATABASE_PATH` the server uses (default `./data/budget.db`). In Docker the service is `family-budget` and
`exec` needs `-T` so the password pipe reaches stdin:
`printf 'Admin1234!\n' | docker compose --project-directory . -f docker/docker-compose.yml exec -T family-budget
/app/family-budget-service setup … --password-stdin`.

```bash
printf 'Admin1234!\n' | go run ./cmd/server setup \
    --family 'Test Family' --currency RUB --timezone Europe/Moscow \
    --email admin@test.com --first-name Admin --last-name Test --password-stdin

TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@test.com","password":"Admin1234!","device_name":"curl"}' | jq -r .data.token)

curl -s localhost:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
```

Passwords are 10–72 bytes and are read from stdin only. `reset-password --email … --password-stdin` sets a new
password and revokes every session of that user.

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
make compose-config   # Validate all docker-compose files (docker/ + deploy/)

# SQLite database
make sqlite-backup    # Create backup
make sqlite-restore   # Restore from backup
make sqlite-shell     # Open SQLite shell
make sqlite-stats     # DB statistics
make db-reset         # Delete ./data/budget.db* (required after editing the migration, see migrations/README.md)

# Development
make migrate-create   # Reminder on how schema changes are made
make help             # Show all commands
```

## 🏛️ Project Structure

```
├── cmd/server/              # Entry point: server, `-health-check`, `setup`, `reset-password`
├── internal/
│   ├── domain/              # Business entities (User, Family, Transaction, Budget, Report, …)
│   ├── auth/                # Bearer tokens, sessions, RequireBearer/RequireRole, login rate limiter
│   ├── application/         # Echo server, JSON error handler, /api/v1 handlers
│   ├── services/            # Business logic
│   ├── infrastructure/      # SQLite repositories, migrations, connection
│   ├── observability/       # Logging and /health
│   ├── testhelpers/         # In-memory DB, full test server, bearer helpers, factories
│   ├── bootstrap.go         # OpenDatabase (DB + migrations), Setup, ResetPassword — shared by server and CLI
│   ├── config.go            # Env-var configuration
│   └── run.go               # Wiring
├── migrations/              # 001_consolidated.{up,down}.sql — the whole schema
├── tests/integration/       # HTTP tests over the full stack, OpenAPI coverage test
├── docs/                    # Product brief, tech stack, audits (specs/), plans, API contract
├── deploy/                  # Self-hosted deployment (stale until plan 05)
├── docker/                  # Dockerfile + docker-compose.yml
└── .github/workflows/       # CI/CD pipelines (ci, docker, security, scorecard, release)
```

## Configuration

All configuration is environment variables; there are no secrets.

| Variable               | Default                                | Description                                                                 |
|------------------------|----------------------------------------|-----------------------------------------------------------------------------|
| `SERVER_HOST`          | `localhost`                            | HTTP server host                                                            |
| `SERVER_PORT`          | `8080`                                 | HTTP server port                                                            |
| `SERVER_READ_TIMEOUT`  | `15s`                                  | HTTP server read timeout                                                    |
| `SERVER_WRITE_TIMEOUT` | `15s`                                  | HTTP server write timeout                                                   |
| `SERVER_IDLE_TIMEOUT`  | `60s`                                  | HTTP server idle timeout                                                    |
| `TRUSTED_PROXIES`      | empty                                  | Comma-separated CIDRs whose `X-Forwarded-For` is trusted for the client IP (login rate limiter). Empty — the client IP is unknown and only the per-email limit applies; behind a reverse proxy set it to the proxy network (e.g. `172.20.0.0/16`) to enable the per-IP limit |
| `DATABASE_PATH`        | `./data/budget.db`                     | SQLite database file path                                                   |
| `BACKUP_DIR`           | empty → `<dir(DATABASE_PATH)>/backups` | Where `POST /api/v1/backups` writes. Docker compose sets `/backups` so `VACUUM INTO` copies do not land inside the database volume |
| `ENVIRONMENT`          | `development`                          | App environment (`development`, `production`, `test`)                       |
| `LOG_LEVEL`            | `info`                                 | Logging level                                                               |
| `LOG_FORMAT`           | `json`                                 | Log format                                                                  |
| `LOG_OUTPUT_PATH`      | `stdout`                               | Log output destination                                                      |

## Running with Docker

`docker/docker-compose.yml` bind-mounts `${DATA_DIR:-.}/backups`; a bind volume does not create
its source directory, so create it before the first `up` (`make docker-up` does it for you).

`.env` belongs in the **repository root**. Compose v2 resolves `.env` (and relative
paths such as `DATA_DIR`) against the *project directory*, which defaults to the
directory of the first `-f` file — `docker/`. `--project-directory .` moves it back to
the root, which is why every command below (and every `make docker-*` target) passes it.

```bash
docker compose --project-directory . -f docker/docker-compose.yml up --build   # build and start
docker compose --project-directory . -f docker/docker-compose.yml up -d        # background
docker compose --project-directory . -f docker/docker-compose.yml down         # stop
```

## 🧪 Testing and Quality

- Unit tests for domain, services, repositories, `internal/auth` (tokens, limiter, middleware) and handlers
- Integration tests in `tests/integration/` run the real HTTP stack — bearer middleware, role gates, rate
  limiter, JSON error handler — over an in-memory SQLite database
- A registered route with no operation in `docs/api/openapi.yaml` fails `make test`
- golangci-lint with 50+ linters, 0 issues required; CodeQL, Semgrep, TruffleHog, OSV Scanner in CI; Dependabot

```bash
make test              # All tests
make test-coverage     # With coverage report
make lint              # Code quality checks
```

## 🔒 Security

- Bearer tokens with server-side sessions: only the SHA-256 of a token is stored, sliding 30-day / absolute
  180-day lifetime, per-session revocation
- Login rate limiter in the application (per IP and per email), `TRUSTED_PROXIES` for the real client IP
- Passwords: bcrypt cost 12, 10–72 bytes, never on the command line
- Role-based access (Admin, Member, Child), input validation on every endpoint
- Backups protected from path traversal with filename validation

## 🏠 Self-Hosted Deployment

> `deploy/` still targets the previous web build: its compose files require `SESSION_SECRET`/`CSRF_SECRET`
> that the application no longer reads, and the nginx/Caddy/fail2ban rules watch a `/login` page that no
> longer exists. Plan 05 replaces the directory with one compose + Caddy setup for `ffs.shatrov.tech`
> ([docs/plans/20260904-05-deploy-ffs.md](docs/plans/20260904-05-deploy-ffs.md)). Until then the notes below
> describe what the scripts do, not a recommended path.

```bash
git clone https://github.com/lllypuk/Family-Finances-Service.git
cd Family-Finances-Service
sudo ./deploy/scripts/install.sh --domain budget.example.com --email admin@example.com
```

The installer cannot be piped into `bash` (`curl … | sudo bash`): it sources `lib/common.sh`,
`lib/docker.sh` and `lib/firewall.sh` from its own directory, which does not exist when the script
is read from stdin. Always clone first.

**The deployment builds from source, it does not pull an image.** `install.sh` clones the repository
into `/opt/family-budget/src` (`REPO_GIT_URL` / `REPO_REF` env vars, default: upstream URL and `main`)
and builds the Docker image on the server, so the machine needs `git` and outbound network access; the
512MB RAM floor is sized for that build, not for the running service (128–256MB).

Supported: Ubuntu 22.04/24.04, Debian 11/12, Rocky/AlmaLinux 9. Options: Docker + Caddy (automatic SSL),
Docker + Nginx (Certbot), native systemd. Scripts in `deploy/scripts/`: `install.sh`, `upgrade.sh`
(`--version <ref>`, `rollback`), `uninstall.sh --keep-data`, `backup.sh`, `health-check.sh`,
`setup-ssl-{nginx,caddy}.sh`, `setup-fail2ban.sh`. Details: [deploy/README.md](deploy/README.md) and
[docs/specs/004-deployment-readiness.md](docs/specs/004-deployment-readiness.md).

## 📚 Documentation

- **[CLAUDE.md](CLAUDE.md)** — development and architecture guidance
- **[docs/](docs/README.md)** — product brief, tech stack, testing strategy, audits (`docs/specs/`), plans
  (`docs/plans/`)
- **[docs/api/openapi.yaml](docs/api/openapi.yaml)** — the API contract; request/response structs live in
  `internal/application/handlers/types.go`, the error envelope in `handlers/errors.go`
- **[deploy/README.md](deploy/README.md)** — self-hosted deployment guide (see the note above)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
