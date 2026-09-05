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
make db-reset         # rm ./data/budget.db* — required after any schema change (see "Database & migrations")
```

Run a single test / package:

```bash
go test ./internal/auth -run TestService_Login_Success -v
go test ./tests/integration -run TestAuth_BearerFullCycle -v
go test ./internal/application/handlers -run 'TestAuthHandler_.*' -v
```

Docker: `make docker-up` / `docker-up-d` / `docker-down` / `docker-logs` — all use `docker/docker-compose.yml`
(builds `docker/Dockerfile`; no secrets are required). Compose is invoked as `docker compose --project-directory .`
(the `DOCKER_COMPOSE` variable in the Makefile) so that `.env` is read from the repo root — hence `build.context: .`
inside `docker/docker-compose.yml`. `make compose-config` validates all five compose files (`docker/` + the four
`deploy/*.yml`) and runs in CI; `COMPOSE_VALIDATE_ENV` still feeds dummy `SESSION_SECRET`/`CSRF_SECRET` to the
`deploy/*.yml` pass because those files demand them via `${VAR:?}` until plan 05 replaces them — the application
does not read either variable.

SQLite: `make sqlite-shell`, `make sqlite-stats`, `make sqlite-backup`,
`make sqlite-restore BACKUP_FILE=./backups/<file>.db`.

**Mandatory before handing off any code change: `make fmt`, `make test`, `make lint` — `make lint` must report
0 issues.** The linter config is strict (see "Linter constraints" below); do not add `//nolint` without a specific
linter name and an explanation (`nolintlint` enforces both).

### Local testing notes

- Curl the local server with `--noproxy '*'`: `curl -s --noproxy '*' 127.0.0.1:8080/health`
- The family is created from the CLI, not over HTTP. From the repo root, with the same `DATABASE_PATH` the server
  uses (`./data/budget.db` is the default for both):

  ```bash
  printf 'Admin1234!\n' | go run ./cmd/server setup --family 'Test Family' --currency RUB \
      --timezone Europe/Moscow --email admin@test.com --first-name Admin --last-name Test --password-stdin
  curl -s --noproxy '*' -X POST 127.0.0.1:8080/api/v1/auth/login \
      -H 'Content-Type: application/json' -d '{"email":"admin@test.com","password":"Admin1234!"}'
  curl -s --noproxy '*' 127.0.0.1:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
  ```

  `go run ./cmd/server reset-password --email … --password-stdin` sets a new password and revokes every session.
- Project skills in `.claude/skills/`: `/pre-commit`, `/db-backup`, `/db-shell`, `/docker-up`, `/migrate-create`,
  `/memory-update`

## Architecture

Layered/Clean architecture, single Go module `family-budget-service`. Wiring happens in one place —
`internal/run.go` (`NewApplication`) — in this order:

1. `LoadConfig()` + `Validate()` (`internal/config.go`) — all config is env vars, no config files:
   `SERVER_*`, `DATABASE_PATH`, `BACKUP_DIR`, `LOG_*`, `ENVIRONMENT`, `TRUSTED_PROXIES`. There are no secrets.
   `BACKUP_DIR` empty means `<dir(DATABASE_PATH)>/backups` (`Config.GetBackupDir()`, compose mounts `/backups`).
2. `internal.OpenDatabase(cfg)` (`internal/bootstrap.go`) — `infrastructure.NewSQLiteConnection` + golang-migrate
   `Up()` from `./migrations`. The CLI subcommands open the DB through the same function.
3. `infrastructure.NewRepositoriesSQLite(db)` → `*handlers.Repositories` (one struct holding every repo).
4. `auth.NewService(repos.Session, repos.User, repos.Family)` — built here and handed to
   `services.NewServices(...)` → `*services.Services` (`Services.Auth`). `StatsService.Summary(ctx, from, to)` owns
   the dashboard arithmetic behind `GET /api/v1/stats/summary`; the handler only formats.
5. `application.NewHTTPServerWithObservability(...)` — builds the Echo instance and registers `/health` and
   `/api/v1`. Nothing else is served: no HTML, no static files, no CORS.

The version reported by `/health` comes from `internal/version` (`version.String()` → `observability.NewHealthService`).
`Version` is the one package-level var allowed by `.golangci.yml` (file-scoped `gochecknoglobals` exclusion); it
defaults to `dev` and is overwritten at link time by `-ldflags "-X family-budget-service/internal/version.Version=…"`
— `VERSION` in the `Makefile` (`git describe --tags --always --dirty`) and `ARG VERSION` in `docker/Dockerfile`.
Every build path that matters passes it: the Makefile `export`s `VERSION` so compose forwards it as a build-arg
(`args: VERSION: ${VERSION:-dev}` in all five compose files), `deploy/scripts/{install,upgrade}.sh` set it from
`git describe` in `./src`, and `docker.yml`/`release.yml` pass `--build-arg`/`-ldflags`. A `-X` flag naming a
symbol that does not exist is silently dropped by the linker, so keep the full package path in sync.
`go build ./...` without `-ldflags` reports `dev`, which is correct, not a bug.

Dependency direction: `application/handlers` → `services` → repository interfaces → `infrastructure`.
Repository interfaces are declared in `internal/services/interfaces.go`;
`internal/application/handlers/repositories.go` re-exports them as type aliases (plus one handler-only extra method
on `TransactionRepository`). Add a new repo method to the service-layer interface, not to the handler alias.

`internal/auth` sits beside that chain and imports neither `services` nor `application`: it declares the narrow
interfaces it needs (`SessionRepository`, `UserLookup`, `SetupChecker`), and the SQLite repositories satisfy them.
`services` imports `auth` only for the password helpers (`HashPassword`, `ValidatePassword`,
`RegisterPasswordValidation`) and for `SessionRevoker`, which `*auth.Service` implements so that `UserService`
can revoke sessions on deactivation without knowing about tokens.

### Single-family model

The deployment serves exactly **one** family; `families.singleton UNIQUE` in the schema is the only guard.
There is no HTTP bootstrap: `cmd/server setup` (`cmd/server/setup.go` → `internal.Setup`) creates the family, the
default categories and the first admin in one `BEGIN IMMEDIATE` transaction (`FamilyRepository.Bootstrap`);
a second run fails with `ErrFamilyAlreadyExists`. Until it has run, `POST /api/v1/auth/login` answers
`409 SETUP_REQUIRED` and `/health` reports `setup_complete: false` while staying `200` (the docker healthcheck
must pass pre-setup). Passwords reach the CLI only through `--password-stdin` (first line of stdin), never argv.
`--timezone` is required and validated but **not persisted** — `user.Family` gets the column in plan 04.
`InviteService` still exists in `internal/services` but has no route; plan 04 deletes it.

### One HTTP surface: `/api/v1` + `GET /health`

Two public routes: `GET /health` and `POST /api/v1/auth/login`. Everything else lives in
`s.echo.Group("/api/v1", auth.RequireBearer(s.services.Auth))` (`internal/application/http_server.go`): the
login route is registered on the bare Echo instance so the group middleware never sees it, and because the group
has a catch-all, an unknown `/api/v1/...` path is `401` without a token and `404` JSON with one.

- **Tokens** (`internal/auth`): 32 random bytes, base64url on the wire, `hex(sha256)` in the `sessions` table.
  `IdleTTL` 30 days sliding, `AbsoluteTTL` 180 days from creation; `last_used_at` is written at most once per
  `TouchInterval` (1h) so reads do not turn into SQLite writes. `Service.Authenticate` re-reads the owner on every
  request (one `JOIN users`), so a deactivated user or a changed role takes effect on the next request — there is
  nothing cached client-side to invalidate. Password change (`PUT /me/password`) keeps the current session and
  revokes the rest; `PUT /users/:id/password`, `reset-password` and deactivation revoke all.
- **Login** returns `{token, expires_at, user}`; unknown email and wrong password are the same
  `401 INVALID_CREDENTIALS` (an unknown email still runs bcrypt against `Service.dummyHash`). Passwords are
  10…72 bytes, enforced by the `password` validator tag (`auth.RegisterPasswordValidation`) in both
  `services.newValidator()` and `handlers.newAPIValidator()`; bcrypt cost 12.
- **Rate limiter** (`internal/auth/ratelimit.go`, in-memory sliding window): 10 attempts per IP per 5 min, 20 per
  email per hour, `429 RATE_LIMITED` + `Retry-After`; a successful login resets the email counter, a blocked
  attempt is not counted. The IP comes from `e.IPExtractor = auth.IPExtractor(config.TrustedProxies)`:
  with `TRUSTED_PROXIES` empty only `RemoteAddr` counts, so behind a proxy every client shares one bucket until
  the proxy CIDR is listed.
- **Role gates** are built from `auth.RequireRole(roles...)`: `adminOnly` for `/api/v1/users`,
  `DELETE /api/v1/categories/:id`, `/api/v1/backups` and `PUT /api/v1/family`; `financeAccess` (admin or member)
  for categories/transactions/budgets/reports/stats; `GET /api/v1/family` and the `/auth/*`, `/me*` routes are
  open to any authenticated role. Wrong role → `403 FORBIDDEN`.
- **Errors outside handlers** (`internal/application/error_handler.go`, `apiErrorHandler`): `RequireBearer` and
  `RequireRole` return `*echo.HTTPError` (they cannot import `handlers` — cycle), and the error handler renders
  the JSON envelope for every error, router 404 and panics included. A non-`*echo.HTTPError` becomes a bare
  `500` — the text goes to the log only, so read the server log when debugging one.
- Handlers take the author from `auth.FromContext(c)` → `*auth.Principal{SessionID, UserID, Email, Role}`
  (see `TransactionHandler.CreateTransaction`); `UserID` is **not** a field of
  `CreateTransactionRequest`/`CreateReportRequest`, so sending it in the body does nothing.
  `POST /api/v1/reports` generates the report through `ReportService.GenerateReport` and stores it (`201`);
  `GET /api/v1/reports/:id/export` returns CSV from `ReportService.ExportReport`.
- **One error envelope, one pagination shape** (`internal/application/handlers/helpers.go`): answer with
  `respondAPI`/`respondError(c, status, code, message, details...)`, never a hand-built `ResponseMeta`; validation
  failures are `422 VALIDATION_ERROR` with `error.details[{field, message, code}]`, while broken JSON and an
  unparseable id stay `400`. Every list runs its query params through `parsePagination(c)` (`defaultLimit=50`,
  `maxLimit=200`, out-of-range → `422`) and reports `meta.pagination {limit, offset, total}` — including the short
  lists, because the Android client generates from a `ListMeta` where `pagination` is required.
  `parsePagination` **writes the 422 itself** and returns the `errResponseAlreadyWritten` sentinel: callers return
  `ignoreWritten(err)`, never the raw error, or Echo's error handler writes a second response over it. Lists the
  repository returns whole are windowed with `pageSlice(items, page)` and answered by `respondList(c, items, page,
  len(all))`. The `field` in `error.details` is the json name (`start_date`), because every handler validator comes
  from `newAPIValidator()` — plain `validator.New()` would report Go field names.

**Working directory matters:** `./migrations` is resolved relative to the process CWD (`migrationsDir` in
`internal/bootstrap.go`), so both the server and the CLI subcommands must be started from the repo root.

### Known rough edges (verified, not fixed)

Do not treat these as regressions you introduced:

- **`deploy/**` is stale until plan 05**: the four `deploy/*.yml`, `deploy/.env.production.example` and the
  nginx/Caddy/fail2ban configs still require `SESSION_SECRET`/`CSRF_SECRET` and rate-limit a `/login` page that
  no longer exists. Do not fix them piecemeal; plan 05 replaces the directory.
- **`--timezone` is accepted and dropped** (see "Single-family model") until plan 04 adds the column.

## Database & migrations

All schema lives in **two consolidated files**: `migrations/001_consolidated.up.sql` and `001_consolidated.down.sql`
(tables: families, users, categories, transactions, budgets, budget_alerts, reports, invites, sessions).
There is no per-change migration file; append new DDL to the end of the `.up.sql` and the matching `DROP` to the
front of the `.down.sql`. See `migrations/README.md`, and `make migrate-create` for the reminder.

**Editing `001` does not touch an existing database.** golang-migrate stores only the version number, so on a DB
that already has version 1 `Up()` returns `ErrNoChange` and the new DDL is skipped silently; the test path starts
from an empty in-memory DB and will not show this. Until the first release the schema changes by rewriting `001`,
and local and server databases are recreated: `make db-reset` (deletes `./data/budget.db*`) then `make run-local`
and `setup` again.

Two independent code paths apply migrations, and **both must keep working**:

- production/dev: golang-migrate (`internal/infrastructure/migrations.go`)
- tests: `internal/testhelpers/sqlite.go` reads and executes the `*.up.sql` files directly

`testhelpers.SQLiteTestDB.CleanTables` has a hardcoded, FK-ordered table list — add any new table to it.

SQLite is opened with `_txlock=immediate` (`infrastructure.NewSQLiteConnection`), so every `BeginTx` takes the
write lock up front; with `MaxOpenConns=1` this is invisible, but do not "optimise" it away — `Bootstrap` relies
on it.

## Testing

- In-memory SQLite (`:memory:?_foreign_keys=ON&_journal_mode=WAL`), no Docker, no sockets. Prefer keeping it that way.
- `testhelpers.SetupSQLiteTestDB(t)` — fresh migrated DB with automatic `t.Cleanup`.
- `testhelpers.SetupHTTPServer(t)` — full repo + `auth.Service` + services + `application.HTTPServer` stack over
  an in-memory DB, with the real `RequireBearer`, role gates, rate limiter and error handler. This is the entry
  point for `tests/integration/*`.
  - Every `/api/v1` request needs a token: `ts.Auth(t)` (admin of the test family, memoized),
    `ts.AuthAs(t, role)` (extra user in the *same* family — a second family cannot exist), or
    `testhelpers.LoginAs(t, ts, u)`, which writes a `sessions` row for `u` directly because factory users carry a
    placeholder password hash. All return an `*AuthSession{Token}`; call `sess.Apply(req)` to set
    `Authorization: Bearer`. `ts.AuthUser` / `ts.AuthFamily` hold what `Auth` created.
  - `testhelpers.RepoRoot(t)` walks up to `go.mod`; use it for anything cwd-relative (`openapi.yaml`, migrations
    in `bootstrap_test.go` via `t.Chdir`) — `go test` runs with cwd = the package directory.
- `testhelpers/factories.go` — `CreateTestFamily`, `CreateTestUser`, etc.
- Handler unit tests put a `*auth.Principal` into the context under `auth.ContextKey` — see `principalContext`
  in `internal/application/handlers/auth_test.go` and reuse it rather than hand-rolling contexts.
- Naming: `TestXxx_Method_Scenario` (e.g. `TestTransactionService_CreateTransaction_Success`).
- `testpackage` is enabled: use an external `package foo_test` unless the path is excluded in `.golangci.yml`
  (`internal/observability/`, `internal/services/dto/`, `tests/`).
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
- PRs: summary + rationale, link to `docs/backlog.md` item when applicable, test evidence.

## Reference docs

Project documentation lives in `docs/` (this replaced the older `.memory_bank/` directory that some docs still
reference): `docs/README.md` (navigation), `docs/product_brief.md`, `docs/tech_stack.md`, `docs/backlog.md`,
`docs/guides/{coding_standards,testing_strategy}.md`, `docs/patterns/{api_standards,error_handling}.md`.
`docs/specs/` holds the audit findings (project assessment, security, deployment readiness) with per-finding
status; `docs/plans/` holds implementation plans, `docs/plans/completed/` the finished ones.

**Current direction:** `docs/specs/005-api-only-redesign.md` — the service is an API-only backend for an
Android app (one instance = one family, two users, `ffs.shatrov.tech` behind Caddy). Plans 01–03 are done
(`docs/plans/completed/`); `docs/plans/20260904-0[4-5]-*.md` run next, in order.

`docs/api/openapi.yaml` is the target contract for `/api/v1` (plus `GET /health`) — the Android client generates
from it. **A registered route with no operation in the spec fails `make test`**
(`tests/integration/openapi_coverage_test.go`: `TestOpenAPISpec_CoversRegisteredRoutes`, plus
`TestOpenAPISpec_OperationsHaveIDAndErrorResponse` requiring an `operationId` and a 4xx `$ref: Error` on every
operation). The reverse — described but not implemented — is allowed until plan 04. See `docs/api/README.md`.
Self-hosted deployment (install/upgrade/backup scripts, nginx & Caddy configs, systemd units, fail2ban) is in
`deploy/` — see `deploy/README.md` and the "Known rough edges" note above.

When runtime/dev commands disagree between documents, `Makefile` + this file win.

## Stack versions (keep in sync with go.mod)

Go **1.26.7** (also pinned as `GO_VERSION` in `.github/workflows/ci.yml`), Echo **v4.15.4**,
`modernc.org/sqlite` (pure Go, no CGO), golang-migrate v4, `golang.org/x/crypto` (bcrypt),
go-playground/validator v10, testify, `go.yaml.in/yaml/v3` (test-only: parses `docs/api/openapi.yaml` in the
coverage test).

CI (`.github/workflows/ci.yml`) runs golangci-lint, `govulncheck`, `make test-coverage`, `make build`, and a Docker
build/run smoke test. Additional workflows: `docker.yml`, `security.yml` (CodeQL, Semgrep, TruffleHog, OSV),
`scorecard.yml` (OSSF Scorecard), `release.yml`.

`scorecard.yml` is deliberately a **separate file**: with `publish_results: true` the OSSF API rejects results from
any workflow that declares a top-level `env` or `defaults` (`security.yml` declares `env.GO_VERSION`), and only the
job running `ossf/scorecard-action` may hold `id-token: write`. Keep that file free of global `env`/`defaults`, and
do not add steps beyond the action's approved list (`actions/checkout`, `actions/upload-artifact`,
`github/codeql-action/upload-sarif`, `ossf/scorecard-action`, `step-security/harden-runner`).

**Every `uses:` is pinned to a commit SHA** with the version as a trailing comment
(`uses: actions/checkout@d23441a… # v6.1.0`), and both `FROM` lines in `docker/Dockerfile` are pinned by digest.
There are no exceptions. Do not reintroduce a tag or branch ref (`@v4`, `@main`, `@master`) — Dependabot updates
the SHA and its comment together. `go install` in CI likewise pins exact tool versions, never `@latest`.

**Token permissions:** every workflow declares a top-level `permissions: contents: read`, and write scopes are
granted per job (`packages: write` to push images, `security-events: write` to upload SARIF, `contents: write` only
for the release job). Job-level `permissions` *replaces* the top-level block rather than merging with it, so a job
that needs `security-events: write` must also restate `contents: read` or its `actions/checkout` loses the token.
