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
(builds `docker/Dockerfile`; **both** `SESSION_SECRET` and `CSRF_SECRET` are required via `${VAR:?}` and compose
refuses to start without them). Compose is invoked as `docker compose --project-directory .` (the `DOCKER_COMPOSE`
variable in the Makefile) so that `.env` is read from the repo root — hence `build.context: .` inside
`docker/docker-compose.yml`. `make compose-config` validates all five compose files (`docker/` + the four
`deploy/*.yml`) and runs in CI.

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
(`/invite/:token`), not through open registration — there is no `/register` route or template.

`RequireSetup` details worth knowing (`internal/web/middleware/setup.go`): it matches on `c.Request().URL.Path`, not
`c.Path()` (which is the *route pattern*, e.g. `/static*`); `isSetupExempt` lets `/health`, `/favicon.ico` and
`/static/…` through **before** any DB call, so styles load on `/setup` and health checks work pre-setup; a completed
setup is cached in an `atomic.Bool` inside the closure. Only `true` is cached — a DB error or `false` is not, so the
setup→ready transition needs no restart.

### Two HTTP surfaces on one Echo instance

- **Web UI** (`internal/web/`): session cookie auth. `middleware.SessionStore` (gorilla/sessions cookie store,
  session name `family-budget-session`) + `middleware.CSRFProtection`. Routes are grouped in `web.go`:
  `RequireAuth()` for the protected group, `RequireAdmin()` for `/users` and `/admin`, `RequireAdminOrMember()` for
  finance pages. Session data lands in the Echo context under key `"user"` (`middleware.GetUserFromContext`).
- **REST API** (`internal/application/handlers/`): `/api/v1/{users,categories,transactions,budgets,reports}`.
  The group is registered as `s.echo.Group("/api/v1", webmw.RequireAPIAuth())` — the *same* session cookie as the web
  UI is the only credential; there are no API tokens. No session → `401` + JSON
  `{"error":{"code":"UNAUTHORIZED",…},"meta":{…}}`. Per-route role gates mirror the web:
  `RequireAPIAdmin` on `POST/PUT/DELETE /api/v1/users` and `DELETE /api/v1/categories/:id`,
  `RequireAPIAdminOrMember` on the categories/transactions/budgets/reports groups
  (wrong role → `403 FORBIDDEN`). All of it lives in `internal/web/middleware/auth.go`.
  **Middleware order matters:** the global `CSRFProtection` (`e.Use` in `web.go`) runs *before* the group
  middleware, so an anonymous write without `X-Csrf-Token` is `403`, and `401` only once a valid token is present.
  Handlers take the author from the session (`sessionUserID` in `handlers/helpers.go`) — `UserID` is **not** a field
  of `CreateTransactionRequest`/`CreateReportRequest`, so sending it in the body does nothing.
  `POST /api/v1/reports` intentionally returns `501 Not Implemented`; report *generation* is only exposed through the
  web UI. Stored-report list/get/delete work.

### Templates

`internal/web/renderer.go` parses `templates/layouts/*.html`, `templates/components/*.html`,
`templates/admin/*.html` (ParseGlob) and walks `templates/pages/**` (ParseFiles) into one `template.Template`;
`Render` executes by **template name**, so `{{define "..."}}` names must be unique across the whole tree.
Custom funcs (`formatCurrency`, `dict`/`map`, `formatBytes`, `safe`, …) are registered in `createTemplateFuncMap`.

**Page data is a struct, not a map.** The transactions/categories/budgets/reports handlers pass a named struct that
**embeds `*PageData`** (`internal/web/handlers/base.go`), built by `BaseHandler.buildPageData(c, title)` — or
`formPageData(c, title, errors)` when re-rendering a form. `buildPageData` fills `Title`, flash `Messages`,
`CSRFToken` and `CurrentUser` (name/surname are read via `services.User.GetUserByID`; `middleware.SessionData` has
no such fields). Because the embedded field is still called `PageData`, existing `{{.PageData.X}}` keeps working
while `{{if .CurrentUser}}` and `{{.CSRFToken}}` now resolve at the template root — that omission was the U-02 bug.
Two consequences when adding a field to a page:

- a template reading a field the struct does not have is a **runtime error (500)**, where a map silently rendered
  `<no value>`. Add the field to the struct instead of hoping.
- page titles are Russian constants in `base.go` (`titleTransactions`, `titleNewBudget`, …) — do not inline the
  literals. `renderXxxFormWithErrors` picks the template from the entity it was given (`existing != nil`,
  `budgetID != ""`), never from the title string.

Older/simpler pages still pass `map[string]any`; keys for those are constants in
`internal/web/handlers/template_keys.go` — reuse them instead of new string literals (`goconst`).

**Rendering fails loudly.** `TemplateRenderer.Render` executes into a buffer and only then writes the response, so a
template reading a missing field returns an error instead of a truncated `200`; `customHTTPErrorHandler` renders the
error page with the real status code (`c.HTMLBlob(code, …)`). Both were silent before: broken pages looked like
successful responses, and every 404/500 page was served as `200`.

**Working directory matters:** `./migrations` is resolved relative to the process CWD, so the server must be started
from the repo root. Templates and static files default to `internal/web/templates` / `internal/web/static` but both
paths are overridable through `application.Config.TemplatesDir` / `Config.StaticDir` (passed on to `web.Paths`) —
that is how the test helper stays cwd-independent.

### Frontend rules (hard requirements)

- HTMX **v2.0.4+** and PicoCSS **v2.1.1+** only — never downgrade to HTMX 1.x.
- No custom JavaScript for interactivity; use `hx-*` attributes and server-rendered partials.
- Handlers branch on HTMX via `BaseHandler.IsHTMXRequest` / the `Hx-Request` header, and redirect with the
  `Hx-Redirect` response header (see `internal/web/handlers/base.go`).

### Known rough edges (verified, not fixed)

Do not treat these as regressions you introduced, and do not paper over them with a `nolint` or a template guard:

- **No rate limiting on login** ([S-03](docs/specs/002-security-audit.md#s-03)) — protection exists only in the
  nginx/Caddy configs and fail2ban, i.e. not at all for `docker-compose.minimal.yml` or a bare systemd deployment.
- **`.github/workflows/docker.yml` cannot publish anything.** Its `docker/build-push-action` step passes
  `context: .` with **no `file:`**, so it looks for `./Dockerfile` — which does not exist (the Dockerfile lives in
  `docker/Dockerfile`). Pushing a tag would fail the build with "failed to read dockerfile"; add
  `file: docker/Dockerfile` before the first release ([D-02](docs/specs/004-deployment-readiness.md#d-02)).
- **`internal/config.go` still defaults `SESSION_SECRET`/`CSRF_SECRET` to known placeholders** and `Validate()`
  compares against those exact strings, so a secret of `123` passes. The compose files now demand both via `${VAR:?}`,
  which covers the documented paths but not `go run ./cmd/server` with `ENVIRONMENT=development`.

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
- `testhelpers.SetupHTTPServer(t)` — full repo + service + `application.HTTPServer` stack over an in-memory DB,
  **including the whole web layer**: `SessionStore`, `CSRFProtection`, `RequireSetup` and the HTML routes, rendered
  from the real templates. This is the entry point for `tests/integration/*`.
  - Templates are resolved cwd-independently: the helper walks up to `go.mod` (`testhelpers.RepoRoot(t)`) and passes
    the absolute path as `application.Config.TemplatesDir`. Do not reintroduce a cwd-relative default here — `go test`
    runs with cwd = the package directory.
  - A web-layer init failure is no longer swallowed: `HTTPServer.WebServerInitError()` surfaces it and the helper
    calls `t.Fatalf`. If a test suddenly dies on "web server initialization failed", a template failed to parse.
  - Because the real middleware is in play, integration requests need a session **and** a CSRF token on writes:
    `ts.Auth(t)` (admin of the test family, memoized), `ts.AuthAs(t, role)` (extra user in the *same* family),
    or `testhelpers.LoginAs(t, ts, u)`. All return an `*AuthSession{Cookie, CSRFToken}`; call `sess.Apply(req)`.
    `ts.AuthUser` / `ts.AuthFamily` hold what `Auth` created.
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
`docs/specs/` holds the audit findings (project assessment, security, UI/UX, deployment readiness) with per-finding
status; `docs/plans/` holds implementation plans, `docs/plans/completed/` the finished ones.
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
