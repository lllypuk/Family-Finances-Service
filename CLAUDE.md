# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make run-local        # run on localhost:8080, SQLite at ./data/budget.db, LOG_LEVEL=debug
make build            # -> ./build/family-budget-service (CGO_ENABLED=0)
make test             # go test -v ./...
make test-unit        # ./internal/...
make test-integration # ./tests/...
make test-coverage    # coverage.out + coverage.html
make fmt              # go fmt ./...
make lint             # golangci-lint run --fix
make pre-commit       # fmt + test + lint
make security-check   # gosec + govulncheck
make deps             # go mod download && go mod tidy
```

Run a single test / package:

```bash
go test ./internal/services -run TestInviteService_CreateInvite -v
go test ./tests/integration -run TestInviteFlow_FullCycle -v
go test ./internal/web/handlers -run 'TestAdminHandler_.*' -v
```

Docker: `make docker-up` / `docker-up-d` / `docker-down` / `docker-logs` — all use `docker/docker-compose.yml`
(builds `docker/Dockerfile`, requires `SESSION_SECRET` in env or `.env`).

SQLite: `make sqlite-shell`, `make sqlite-stats`, `make sqlite-backup`,
`make sqlite-restore BACKUP_FILE=./backups/<file>.db`.

**Mandatory before handing off any code change: `make fmt`, `make test`, `make lint` — `make lint` must report
0 issues.** The linter config is strict (see "Linter constraints" below); do not add `//nolint` without a specific
linter name and an explanation (`nolintlint` enforces both).

### Local testing notes

- Curl the local server with `--noproxy '*'`: `curl -s --noproxy '*' 127.0.0.1:8080/health`
- Test credentials for the local setup flow: `admin@test.com` / `Admin123!`, family `Test Family`
- Project skills in `.claude/skills/`: `/test-frontend` (drives `agent-browser` against a running `make run-local`
  instance), `/pre-commit`, `/db-backup`, `/db-shell`, `/docker-up`, `/migrate-create`, `/memory-update`

## Architecture

Layered/Clean architecture, single Go module `family-budget-service`. Wiring happens in one place —
`internal/run.go` (`NewApplication`) — in this order:

1. `LoadConfig()` + `Validate()` (`internal/config.go`) — all config is env vars, no config files.
2. `infrastructure.NewSQLiteConnection(path)` then `infrastructure.NewMigrationManager(dbURL, "./migrations").Up()`
   — migrations run automatically at startup via golang-migrate.
3. `infrastructure.NewRepositoriesSQLite(db)` → `*handlers.Repositories` (one struct holding every repo).
4. `services.NewServices(...)` → `*services.Services` (one struct holding every service).
5. `application.NewHTTPServerWithObservability(...)` — builds the Echo instance, registers `/api/v1` handlers, and
   calls `web.NewWebServer(...)` which mounts the HTML/HTMX interface onto the *same* Echo instance.

Dependency direction: `web`/`application/handlers` → `services` → repository interfaces → `infrastructure`.
Repository interfaces are declared in `internal/services/interfaces.go`;
`internal/application/handlers/repositories.go` re-exports them as type aliases (plus one handler-only extra method
on `TransactionRepository`). Add a new repo method to the service-layer interface, not to the handler alias.

### Single-family model

The deployment serves exactly **one** family. `middleware.RequireSetup` is registered globally: if no family exists,
every path redirects to `/setup`; once it exists, `/setup` redirects to `/login`. `FamilyService.SetupFamily` is the
bootstrap path that creates the family plus the first admin user. New members join through the invite system
(`/invite/:token`), not through open registration.

### Two HTTP surfaces on one Echo instance

- **Web UI** (`internal/web/`): session cookie auth. `middleware.SessionStore` (gorilla/sessions cookie store,
  session name `family-budget-session`) + `middleware.CSRFProtection`. Routes are grouped in `web.go`:
  `RequireAuth()` for the protected group, `RequireAdmin()` for `/users` and `/admin`, `RequireAdminOrMember()` for
  finance pages. Session data lands in the Echo context under key `"user"` (`middleware.GetUserFromContext`).
- **REST API** (`internal/application/handlers/`): `/api/v1/{users,categories,transactions,budgets,reports}`.
  Note this group is currently registered **without any auth middleware** — do not assume API routes are protected.
  `POST /api/v1/reports` intentionally returns `501 Not Implemented`; report *generation* is only exposed through the
  web UI. Stored-report list/get/delete work.

### Templates

`internal/web/renderer.go` parses `templates/layouts/*.html`, `templates/components/*.html`,
`templates/admin/*.html` (ParseGlob) and walks `templates/pages/**` (ParseFiles) into one `template.Template`;
`Render` executes by **template name**, so `{{define "..."}}` names must be unique across the whole tree.
Custom funcs (`formatCurrency`, `dict`/`map`, `formatBytes`, `safe`, …) are registered in `createTemplateFuncMap`.
Template data-map keys are centralized as constants in `internal/web/handlers/template_keys.go` — reuse them instead
of new string literals (`goconst`).

**Working directory matters:** templates (`internal/web/templates`), static files (`internal/web/static`) and
`./migrations` are all resolved as paths relative to the process CWD. The server must be started from the repo root.

### Frontend rules (hard requirements)

- HTMX **v2.0.4+** and PicoCSS **v2.1.1+** only — never downgrade to HTMX 1.x.
- No custom JavaScript for interactivity; use `hx-*` attributes and server-rendered partials.
- Handlers branch on HTMX via `BaseHandler.IsHTMXRequest` / the `Hx-Request` header, and redirect with the
  `Hx-Redirect` response header (see `internal/web/handlers/base.go`).

## Database & migrations

All schema lives in **two consolidated files**: `migrations/001_consolidated.up.sql` and `001_consolidated.down.sql`
(tables: families, users, categories, transactions, budgets, budget_alerts, reports, user_sessions, invites).
There is no per-change migration file; append new DDL to the end of the `.up.sql` and the matching `DROP` to the
front of the `.down.sql`. See `migrations/README.md`, and `make migrate-create` for the reminder.

Two independent code paths apply migrations, and **both must keep working**:

- production/dev: golang-migrate (`internal/infrastructure/migrations.go`)
- tests: `internal/testhelpers/sqlite.go` reads and executes the `*.up.sql` files directly

`testhelpers.SQLiteTestDB.CleanTables` has a hardcoded, FK-ordered table list — add any new table to it.

## Testing

- In-memory SQLite (`:memory:?_foreign_keys=ON&_journal_mode=WAL`), no Docker, no sockets. Prefer keeping it that way.
- `testhelpers.SetupSQLiteTestDB(t)` — fresh migrated DB with automatic `t.Cleanup`.
- `testhelpers.SetupHTTPServer(t)` — full repo + service + `application.HTTPServer` stack over an in-memory DB;
  this is the entry point for `tests/integration/*`.
- `testhelpers/factories.go` — `CreateTestFamily`, `CreateTestUser`, etc.
- Naming: `TestXxx_Method_Scenario` (e.g. `TestTransactionService_CreateTransaction_Success`).
- `testpackage` is enabled: use an external `package foo_test` unless the path is excluded in `.golangci.yml`
  (`internal/web/handlers/`, `internal/observability/`, `internal/services/dto/`, `tests/`).
- Web handler tests bypass real sessions: `RequireAuth`/`GetSessionData` honor the context keys
  `mock_session_data` and `mock_session_error`. See `internal/web/handlers/testhelpers_test.go`
  (`newTestContext`, `withSession`, `withHTMX`) — reuse those helpers rather than hand-rolling contexts.
- Use testify `require` for fatal preconditions, `assert` for the rest (`testifylint` enforces correct usage).

## Linter constraints worth knowing up front

`.golangci.yml` is the "maratori golden config" with project tweaks. The rules that most often force a rewrite:

- `golines` max line length **120**; `goimports` local prefix `family-budget-service` (local imports last group).
- `funlen` 100 lines / 50 statements, `gocognit` 20, `cyclop` 30 per function and **10.0 package average**.
- `gochecknoglobals` / `gochecknoinits` — no package-level vars or `init()`; `mnd` — no magic numbers (declare consts).
- `sloglint`: no global loggers, and use the `...Context` slog methods when a context is in scope. Pass `*slog.Logger`
  down explicitly (as `NewServices` does).
- `depguard`: `log` is banned outside `main.go` (use `log/slog`), `math/rand` banned (use `math/rand/v2`).
- `nonamedreturns`, `nakedret` (max 0 lines), `errorlint` (wrap with `%w`), `errcheck` incl. type assertions,
  `govet` shadow-strict, `exhaustive` on switches **and** maps.
- `funcorder`: constructors go immediately after the type they construct (the struct-method ordering check is off).

## Conventions

- Comments and log messages are a mix of Russian and English; match the surrounding file rather than converting it.
- File names are snake_case-ish and descriptive: `transaction_service.go`, `user_repository_sqlite.go`.
- Keep handlers thin — business logic belongs in `internal/services/`.
- Commit prefixes in use: `feat:`, `fix:`, `docs:`, `refactor:`, `security:`, `deps(deps):`.
- PRs: summary + rationale, link to `docs/backlog.md` item when applicable, test evidence, screenshots for UI changes.

## Reference docs

Project documentation lives in `docs/` (this replaced the older `.memory_bank/` directory that some docs still
reference): `docs/README.md` (navigation), `docs/product_brief.md`, `docs/tech_stack.md`, `docs/backlog.md`,
`docs/guides/{coding_standards,testing_strategy}.md`, `docs/patterns/{api_standards,error_handling}.md`.
Self-hosted deployment (install/upgrade/backup scripts, nginx & Caddy configs, systemd units, fail2ban) is in
`deploy/` — see `deploy/README.md`.

When runtime/dev commands disagree between documents, `Makefile` + this file win.

## Stack versions (keep in sync with go.mod)

Go **1.26.5** (also pinned as `GO_VERSION` in `.github/workflows/ci.yml`), Echo **v4.15.4**,
`modernc.org/sqlite` (pure Go, no CGO), golang-migrate v4, gorilla/sessions, go-playground/validator v10,
testify. Frontend: HTMX 2.0.4, PicoCSS 2.1.1.

CI (`.github/workflows/ci.yml`) runs golangci-lint, `govulncheck`, `make test-coverage`, `make build`, and a Docker
build/run smoke test. Additional workflows: `docker.yml`, `security.yml` (CodeQL, Semgrep, TruffleHog, OSV,
Scorecard), `release.yml`.
