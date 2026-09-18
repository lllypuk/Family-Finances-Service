# Family Finances Service

**Self-hosted family budget backend**: a single Go binary with an embedded SQLite database that serves a JSON
API for the Android client. One instance = one family.

## 🎯 Project Status: IN DEVELOPMENT 🚧

> **Direction (September 2026):** API-only backend for an Android app. Decisions and the implementation
> plans: [docs/specs/005-api-only-redesign.md](docs/specs/005-api-only-redesign.md). Plans 01–10 and 12–17 are done: the
> web interface, cookie sessions and CSRF are gone, money is integer minor units, dates are calendar dates,
> the deployment is one compose with Caddy, the Android client lives in `android/` with its settings screen,
> stored reports are gone in favour of `GET /api/v1/stats/monthly`, budgets can repeat as a series, and the
> client's UI audit is closed (plan 13, client only — the contract did not move), and Prometheus metrics
> moved onto a second listener (plan 14 — the contract did not move either), and bank screenshots can be
recognized into candidate transactions (plan 15, `POST /api/v1/transactions/recognize`; client `0.7.0`, not
tagged yet, needs a server that has it), and transactions can be bound to accounts with a monthly
reconciliation against the bank (plan 16, server `v0.6.0`, client `0.8.0`, not tagged yet), and holdings
with a monthly net-worth series (plan 17, server `v0.7.0`, client `0.9.0`, not tagged yet);
> the sections below describe the code as it is today. Releases: server `v0.3.0` (plan 10), `v0.4.0`
> (plan 12) and `v0.5.0` (plan 14), client `app-v0.6.0` — it needs a server of `v0.4.0` or newer (`recurring` is required in the
> generated model). Left: plan 11, the client's "Обзор" screen over `summary` + `monthly` and multi-select
> over transactions ([docs/backlog.md](docs/backlog.md)).

- ✅ REST API for family, users, categories, accounts, transactions, budgets, holdings, stats, backups
- ✅ Bearer-token authentication with server-side sessions and a login rate limiter
- ✅ Family bootstrap, password reset and schema moves from the CLI (`setup`, `reset-password`, `migrate`)
- ✅ Screenshot recognition over an Ollama daemon (`LLM_OLLAMA_HOST`, off by default): candidates for review,
  nothing is stored
- ✅ Lightweight SQLite database, migrations applied at startup
- ✅ CI/CD in GitLab: checks on every merge request, image and deploy from `main`
- ✅ Single Docker container, built from source (`docker/Dockerfile`)
- ✅ Money as integer minor units (`amount_minor`), calendar `YYYY-MM-DD` dates, idempotent `POST`
- ✅ Prometheus metrics (`ffs_*`) on a separate listener (`METRICS_ADDR`, off by default); `/metrics` is not
  part of `/api/v1`
- ✅ Self-hosted deployment: one compose (`deploy/`) with Caddy, Let's Encrypt and daily CLI backups
- ✅ Image published to `registry.gitlab.shatrov.tech` on every push to `main` and on every `v*` tag;
  the server pulls it, `docker/docker-compose.yml` still builds locally for development

## API

The contract lives in [`docs/api/openapi.yaml`](docs/api/openapi.yaml) (OpenAPI 3.1, hand-written,
see [`docs/api/README.md`](docs/api/README.md)); the Android client generates from it. Code and spec match
exactly — `make test` fails on a route missing from the spec **and** on an operation with no route.

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

There are two roles, `admin` and `member`; both see all of the family's data. There are no invites — an admin
creates a user with `POST /api/v1/users`.

- `/api/v1/users` and `/api/v1/backups`, `PUT /api/v1/family`, `DELETE /api/v1/categories/:id`,
  `DELETE /api/v1/accounts/:id`, `DELETE /api/v1/holdings/:id` — **admin only**
- `/api/v1/{categories,accounts,transactions,budgets,holdings,stats}` — **admin or member**
- `GET /api/v1/family`, `/api/v1/me*`, `/api/v1/auth/*` — any authenticated role

The author of a record is taken from the token, so `user_id` in a request body is ignored.

### Ready (current behavior)

- `GET /api/v1/stats/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` — dashboard summary (totals, deltas to the
  previous period, top categories, budget progress, recent transactions); defaults to the current month
- `GET /api/v1/stats/monthly?from=YYYY-MM-DD&to=YYYY-MM-DD` — one bucket per calendar month in `[from, to]`
  (income, expenses, net, count); months without transactions are zeros and the edge months cover only the
  days inside the interval. Defaults to twelve months ending today in the family's timezone; a period wider
  than 120 months is `422`
- `GET`/`PUT /api/v1/me`, `PUT /api/v1/me/password`; `PATCH /api/v1/users/:id` (`role`, `is_active`),
  `PUT /api/v1/users/:id/password`
- Backups over API: `POST`/`GET /api/v1/backups`, `GET /api/v1/backups/:name/download`,
  `DELETE /api/v1/backups/:name`
- `POST /api/v1/transactions/bulk-delete`
- Accounts: `GET /api/v1/accounts` (`?archived=true` adds archived ones), `POST` (`{id?, name}`), `PUT`
  (`name`, `is_archived`), `DELETE` — `409 ACCOUNT_IN_USE` while a transaction or reconciliation refers to it
  (archive it instead); the name is unique case-insensitively, `409 ACCOUNT_NAME_EXISTS`
- `account_id` on a transaction is optional: in `PUT` a missing or `null` field keeps the stored one,
  `clear_account: true` unbinds it; `GET /api/v1/transactions?account_id=…` or `?unassigned=true`
- Reconciliation: `PUT`/`DELETE /api/v1/accounts/:id/reconciliations/YYYY-MM` store the bank's expense figure
  for the month; `GET /api/v1/stats/reconciliation?month=YYYY-MM` answers recorded expenses (summed on read),
  the bank figure and the difference per account, plus the expenses with no account
- Holdings: `GET /api/v1/holdings` (`?archived=true` adds archived ones), `POST` (`{id?, name, side, kind}`), `PUT`
  (`side` cannot change), `DELETE` (admin, takes the snapshot history with it); `409 HOLDING_NAME_EXISTS`.
  Snapshots: `PUT`/`DELETE /api/v1/holdings/:id/values/YYYY-MM-DD` (a future date is `422`), history in
  `GET /api/v1/holdings/:id/values`; `GET /api/v1/stats/net-worth?from&to` answers monthly assets, liabilities
  and net, each value carried forward until the next snapshot
- `POST /api/v1/transactions/recognize` — multipart `images` (1–5 PNG/JPEG, ≤ 2 MiB each), answers candidate
  transactions for review and saves nothing; the client saves the chosen ones with ordinary `POST /transactions`.
  A paid, non-idempotent call — neither side retries it on its own; it may run up to ~205 s. `503
  RECOGNITION_UNAVAILABLE` when switched off or the model is unreachable (`Retry-After` when known), `502
  RECOGNITION_FAILED` for an answer that does not parse, `413`, `408` for a too slow upload, `422` with `field: images[i]`
- Money is `amount_minor` — an integer in the family's minor units (kopeks/cents); percentages and utilization
  stay fractional. `PUT /api/v1/family` returns `409 CURRENCY_LOCKED` if a transaction, a
  reconciliation or a holding snapshot already exists
- Transaction and budget dates are calendar `YYYY-MM-DD`; period bounds use the family's `timezone`
- `POST` of a transaction, budget, category or account accepts a client-generated `id` (any valid UUID): a retry with the same
  `id` answers `200` with the existing record instead of creating a duplicate
- Budgets: business refusals are `409` with their own codes — `BUDGET_OVERLAP` (periods of one scope may not share
  even a single day), `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS` and `BUDGET_NOT_TAIL`. `DELETE`
  is final: a deleted budget is `404` for GET/PUT/DELETE, its `id` stays taken (its name and period do not), and
  `is_active` can no longer be sent in `PUT /api/v1/budgets/:id`
- A budget with `recurring: true` is the tail of a series, tied together by `series_id`. There is no background job:
  the next calendar period is materialised on reads (`GET /api/v1/budgets`, `GET /api/v1/stats/summary`) and the flag
  moves to the new instance, past ones stay as history. Dates must match the calendar period and `custom` cannot
  recur (`422`); editing an instance that is no longer the tail is `409 BUDGET_NOT_TAIL` — re-read the list. A period
  already taken by a manual budget is stepped over, not a stop of the series
- Every list answers with `meta.pagination {limit, offset, total}` — `limit` defaults to 50, max 200
- One error envelope everywhere: `{"error":{"code","message","details"},"meta":{...}}`;
  validation fails with `422 VALIDATION_ERROR` and per-field `details`

### Not available yet

- Users are never deleted, only deactivated (`PATCH /users/:id {"is_active": false}`)
- Backup **restore** is deliberately not exposed over the API and has no subcommand — in production it is
  manual over ssh ([deploy/README.md](deploy/README.md)); `make sqlite-restore` is a dev-only `cp`
- More than one currency: a family has exactly one, and it can no longer be changed once a transaction, a reconciliation or a holding snapshot exists

## 🏗️ Architecture and Technology Stack

- **Go 1.26** with Echo v4.15
- **SQLite** (modernc.org/sqlite) — pure Go, no CGO; migrations applied automatically at startup
- **Clean Architecture**: `domain` → `services` → repository interfaces → `infrastructure`; `internal/auth`
  (tokens, sessions, middleware, rate limiter) beside them
- **Structured logging** with slog, `/health` for orchestration, optional `/metrics` on a second port,
  graceful shutdown
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
make sqlite-backup    # Create backup in ./backups (go run ./cmd/server backup)
make sqlite-restore   # Restore from backup (dev only)
make sqlite-shell     # Open SQLite shell
make sqlite-stats     # DB statistics
make db-reset         # Delete ./data/budget.db* (required after editing the migration, see migrations/README.md)

# Development
make migrate-create   # Reminder on how schema changes are made
make help             # Show all commands
```

## 🏛️ Project Structure

```
├── cmd/server/              # Entry point: server, `-health-check`, `setup`, `reset-password`, `backup`, `migrate`, `recognize`
├── internal/
│   ├── domain/              # Business entities (User, Family, Transaction, Budget, Category, …)
│   ├── auth/                # Bearer tokens, sessions, RequireBearer/RequireRole, login rate limiter
│   ├── application/         # Echo server, JSON error handler, /api/v1 handlers
│   ├── services/            # Business logic
│   ├── infrastructure/      # SQLite repositories, migrations, connection; llmengine/ — the Ollama call
│   ├── recognize/           # Screenshot recognition: prompt, parsing and normalizing the model's answer
│   ├── observability/       # Logging and /health
│   ├── metrics/             # Prometheus registry, Echo middleware, scrape collectors, /metrics listener
│   ├── version/             # Version reported by /health, set at link time by -ldflags
│   ├── testhelpers/         # In-memory DB, full test server, bearer helpers, factories
│   ├── bootstrap.go         # OpenDatabase (DB + migrations), Setup, ResetPassword — shared by server and CLI
│   ├── config.go            # Env-var configuration
│   └── run.go               # Wiring
├── migrations/              # 001_consolidated.{up,down}.sql + numbered NNN_* steps
├── tests/integration/       # HTTP tests over the full stack, OpenAPI coverage test
├── docs/                    # Product brief, tech stack, audits (specs/), plans, API contract
├── deploy/                  # Self-hosted deployment: compose + Caddy + install/release scripts
├── docker/                  # Dockerfile + docker-compose.yml
├── android/                 # Android client: :app (Compose) + :core:api (generated client, transport)
└── .gitlab-ci.yml           # CI/CD: checks, image, deploy to the mini-server
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
| `BACKUP_DIR`           | empty → `<dir(DATABASE_PATH)>/backups` | Where `POST /api/v1/backups` and the `backup` subcommand write. Docker compose sets `/backups` so `VACUUM INTO` copies do not land inside the database volume |
| `BACKUP_KEEP`          | `30`                                   | How many newest backup files to keep; shared by `POST /api/v1/backups` and the `backup` subcommand (`--keep N` overrides it). A non-numeric or non-positive value is ignored |
| `METRICS_ADDR`         | empty                                  | `host:port` of the second listener serving `GET /metrics` (Prometheus). Empty — neither the listener nor the registry is built; production compose sets `0.0.0.0:9091` and does not publish the port. `/metrics` is never served on the API port |
| `LLM_OLLAMA_HOST`      | empty                                  | Ollama daemon URL for screenshot recognition. Empty — recognition is off and `POST /api/v1/transactions/recognize` answers `503`. Inside Docker `localhost` is the container itself |
| `LLM_MODEL`            | `gemma4:31b-cloud`                     | Model name; checked only when `LLM_OLLAMA_HOST` is set                      |
| `LLM_TIMEOUT`          | `60s`                                  | Timeout of one model attempt, at most `60s`; checked only when `LLM_OLLAMA_HOST` is set |
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
- `docs/api/openapi.yaml` is checked both ways by `make test`: a route with no operation, and an operation
  with no route, both fail
- golangci-lint with 50+ linters (gosec among them), 0 issues required; `govulncheck` in CI

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
- Role-based access (`admin`, `member`), input validation on every endpoint
- Backups protected from path traversal with filename validation

## 🏠 Self-Hosted Deployment

The server pulls a prebuilt image; nothing is compiled there. Two layouts share one file set and are
chosen by `COMPOSE_FILE` in `.env`: compose starts its own Caddy with Let's Encrypt, or the app joins
the external docker network `edge` behind a Caddy that already owns 80/443 (this is how the
mini-server runs it).

Point an A record at the server, log in to the registry, then:

```bash
git clone ssh://git@gitlab.shatrov.tech:2222/shatrov.tech/family-finances-service.git
cd family-finances-service
sudo ./deploy/scripts/install.sh --domain ffs.shatrov.tech --email admin@example.com
```

The installer cannot be piped into `bash` (`curl … | sudo bash`): it sources `lib/common.sh`,
`lib/docker.sh` and `lib/firewall.sh` from its own directory, which does not exist when the script
is read from stdin. Always clone first.

After the first install, updates are the pipeline's job: every green build on `main` copies `deploy/`
to the host and runs `release.sh`, which snapshots the database, swaps the image and restarts.
Rollback is a previous image tag in `.env`, no rebuild. If the release being rolled back migrated the
schema, step it down first with the image that is still deployed (`docker compose run --rm --no-deps -T app
migrate --to N`) — the older image refuses to start on a version it has no migration file for; see
[deploy/README.md](deploy/README.md).

The family and the first admin are created over ssh (`docker compose exec app
/app/family-budget-service setup …`), the second user through `POST /api/v1/users`; backups are a host
cron job running the `backup` subcommand.

Supported: Ubuntu 22.04/24.04, Debian 11/12, Rocky/AlmaLinux 9. Scripts in `deploy/scripts/`:
`install.sh` (`--domain`, `--email`, `--image`, `--non-interactive`, `--dry-run`, `--reinstall`),
`uninstall.sh --keep-data`, `health-check.sh` (`HEALTH_URL`, default `https://$DOMAIN/health`).
Details: [deploy/README.md](deploy/README.md).

## 📱 Android client

`android/` — the online Kotlin/Compose client to `/api/v1` (login, home, transactions, categories, budgets;
the four roots are switched by a bottom tab bar). The profile icon on home opens settings: own profile and
password, sessions with revoke, and — for an admin — users, family and backups; sign-out lives there too.
Models and typed interfaces are generated from `docs/api/openapi.yaml` into `android/core/api/generated`
and committed; generation needs the network, so it stays out of `check` and its freshness is a separate
target and CI step.

Locally this needs JDK 21+ and an Android SDK with `android-37.0` and build-tools 37; the CI job
installs the same set.

```bash
make -C android check        # format, Robolectric unit tests, Android Lint, R8 model check — offline
make -C android api-check    # regenerate the client and fail if the contract moved without it
make -C android apk          # signed release APK -> android/app/build/outputs/apk/release/
```

The APK is installed from a laptop; there is no store. Signing needs one permanent keystore
(`FFS_KEYSTORE_PATH` / `FFS_KEYSTORE_PASSWORD`, defaults under `~/.android`) — the same one CI uses,
because another certificate means uninstalling the app together with its data. A tag `app-vX.Y.Z` builds
the APK in CI and keeps it as a job artifact for a week; server tags `vX.Y.Z` do not run the Android jobs.
Bump `appVersionCode`/`appVersionName` in `android/gradle/libs.versions.toml` before tagging. The two release
independently: a contract change must be additive, or the APK goes out first; a new required response field is the
reverse — the APK goes out after the server (`app-v0.5.0` after `v0.4.0`). Details: [android/CLAUDE.md](android/CLAUDE.md).

## 📚 Documentation

- **[CLAUDE.md](CLAUDE.md)** — development and architecture guidance
- **[docs/](docs/README.md)** — product brief, tech stack, testing strategy, audits (`docs/specs/`), plans
  (`docs/plans/`), task workflows (`docs/workflows/`)
- **[docs/api/openapi.yaml](docs/api/openapi.yaml)** — the API contract; request/response structs live in
  `internal/application/handlers/types.go`, the error envelope in `handlers/errors.go`
- **[deploy/README.md](deploy/README.md)** — self-hosted deployment guide (see the note above)
- **[android/CLAUDE.md](android/CLAUDE.md)** — Android client: modules, version rules, code generation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
