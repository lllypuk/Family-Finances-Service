# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Targets and their flags are in the `Makefile`; `make db-reset` (`rm ./data/budget.db*`) is required after any schema
change (see "Database & migrations").

Docker: `make docker-up` / `docker-up-d` / `docker-down` / `docker-logs` — all use `docker/docker-compose.yml`
(builds `docker/Dockerfile`; no secrets are required). Compose is invoked as `docker compose --project-directory .`
(the `DOCKER_COMPOSE` variable in the Makefile) so that `.env` is read from the repo root — hence `build.context: .`
inside `docker/docker-compose.yml`. `make compose-config` validates all three layouts (`docker/`, `deploy/docker-compose.yml` and that file
with `deploy/docker-compose.proxied.yml` on top) and runs in CI; `COMPOSE_VALIDATE_ENV` supplies the two
variables the deploy files demand via `${VAR:?}` (`FFS_IMAGE`, `FFS_EDGE_SUBNET`) — there are no secrets.
`make caddy-validate` checks `deploy/caddy/Caddyfile` with the Caddy image taken out of the deploy compose
(so the digest lives in one place).

SQLite: `make sqlite-shell`, `make sqlite-stats`, `make sqlite-backup` (runs `go run ./cmd/server backup`),
`make sqlite-restore BACKUP_FILE=./backups/backup_<ts>.db` (dev-only `cp`; in production restore is manual over ssh).

**Mandatory before handing off any code change: `make fmt`, `make test`, `make lint` — `make lint` must report
0 issues.** The linter config is strict (`.golangci.yml`); do not add `//nolint` without a specific
linter name and an explanation (`nolintlint` enforces both).

## Architecture

Layered/Clean architecture, single Go module `family-budget-service`. Wiring happens in one place —
`internal/run.go` (`NewApplication`) — in this order:

1. `LoadConfig()` + `Validate()` (`internal/config.go`) — all config is env vars, no config files:
   `SERVER_*`, `DATABASE_PATH`, `BACKUP_DIR`, `BACKUP_KEEP`, `LOG_*`, `ENVIRONMENT`, `TRUSTED_PROXIES`,
   `METRICS_ADDR`, `LLM_*` (see "Screenshot recognition").
   There are no secrets. `BACKUP_DIR` empty means `<dir(DATABASE_PATH)>/backups` (`Config.GetBackupDir()`,
   compose mounts `/backups`); `BACKUP_KEEP` (default 30) is the retention shared by `POST /api/v1/backups`
   and the `backup` subcommand.
2. `internal.OpenDatabase(cfg)` (`internal/bootstrap.go`) — `infrastructure.NewSQLiteConnection` + golang-migrate
   `Up()` from `./migrations`. `setup`/`reset-password` open the DB through the same function;
   `backup` uses `OpenDatabaseNoMigrate` — a copy must be possible from an outdated schema, and a cron
   `compose run` must not migrate the live DB under the running container. `migrate` opens no DB at all: it
   goes straight to `NewMigrationManager` and `os.Stat`s `DATABASE_PATH` first, because golang-migrate on a
   wrong path would create an empty database and stamp it with a version.
3. `infrastructure.NewRepositoriesSQLite(db)` → `*handlers.Repositories` (one struct holding every repo).
4. `auth.NewService(repos.Session, repos.User, repos.Family)` — built here and handed to
   `services.NewServices(...)` → `*services.Services` (`Services.Auth`). `StatsService.Summary(ctx, from, to)` owns
   the dashboard arithmetic behind `GET /api/v1/stats/summary`, `StatsService.Monthly(ctx, from, to)` the month
   series behind `GET /api/v1/stats/monthly` (capped at `monthlyMaxMonths` = 120 buckets, since the bounds
   come from the client), `StatsService.NetWorth(ctx, from, to)` the one behind `GET /api/v1/stats/net-worth`
   (same bounds and cap, `to` after today is `422`); the handler only formats.
5. `application.NewHTTPServerWithObservability(...)` — builds the Echo instance and registers `/health` and
   `/api/v1`. Nothing else is served: no HTML, no static files, no CORS. The metrics registry reaches it as
   `application.Config.Metrics`.
6. `metrics.NewServer(cfg.Server.MetricsAddr, registry.Handler())` — a second `net/http` listener with the one
   route `GET /metrics`, built only when `METRICS_ADDR` is set (see "Metrics on a second listener").

Both servers are now created in `NewApplication`, not inside their own `Start`, and `Run` hands them to
`serveAll(ctx, listeners...)`: it returns the first start error (a busy port used to be swallowed) together with
the shutdown errors, and on that error or on a signal stops every listener and waits for all of them. Listeners
are stopped there and only there — a second pass in `shutdown()` would spend the graceful window twice — so
`shutdown()` only closes the DB, which the scrape collectors read.

The version reported by `/health` comes from `internal/version` (`version.String()` → `observability.NewHealthService`).
`Version` is the one package-level var allowed by `.golangci.yml` (file-scoped `gochecknoglobals` exclusion); it
defaults to `dev` and is overwritten at link time by `-ldflags "-X family-budget-service/internal/version.Version=…"`
— `VERSION` in the `Makefile` (`git describe --tags --always --dirty`) and `ARG VERSION` in `docker/Dockerfile`.
Every build path that matters passes it: the Makefile `export`s `VERSION` so compose forwards it as a build-arg
(`args: VERSION: ${VERSION:-dev}` in both compose files) and `.gitlab-ci.yml` passes `--build-arg VERSION`. A `-X` flag
naming a symbol that does not exist is silently dropped by the linker, so keep the full package path in sync.
`go build ./...` without `-ldflags` reports `dev`, which is correct, not a bug.

Dependency direction: `application/handlers` → `services` → repository interfaces → `infrastructure`.
Repository interfaces are declared in `internal/services/interfaces.go`;
`internal/application/handlers/repositories.go` re-exports them as type aliases (plus one handler-only extra method
on `TransactionRepository`). Add a new repo method to the service-layer interface, not to the handler alias.

`internal/auth` sits beside that chain and imports neither `services` nor `application`: it declares the narrow
interfaces it needs (`SessionRepository`, `UserLookup`, `SetupChecker`), and the SQLite repositories satisfy them.
`services` imports `auth` only for the password helpers (`HashPassword`, `ValidatePassword`,
`RegisterPasswordValidation`); session revocation belongs to the user repository (see "User writes are
column-scoped" below), so `UserService` never touches sessions.

### Single-family model

The deployment serves exactly **one** family; `families.singleton UNIQUE` in the schema is the only guard.
There is no HTTP bootstrap: `cmd/server setup` (`cmd/server/setup.go` → `internal.Setup`) creates the family, the
default categories and the first admin in one `BEGIN IMMEDIATE` transaction (`FamilyRepository.Bootstrap`);
a second run fails with `ErrFamilyAlreadyExists`. Until it has run, `POST /api/v1/auth/login` answers
`409 SETUP_REQUIRED` and `/health` reports `setup_complete: false` while staying `200` (the docker healthcheck
must pass pre-setup). Passwords reach the CLI only through `--password-stdin` (first line of stdin), never argv.
`--timezone` is required, validated by the `timezone` validator tag (`time.LoadLocation`) and stored in
`families.timezone`; `Family.Location()` falls back to UTC on an unknown zone, and `StatsService` derives
"current month" from it. Invites are gone: a user is created by an admin over `POST /api/v1/users`.

### One HTTP surface: `/api/v1` + `GET /health`

Two public routes: `GET /health` and `POST /api/v1/auth/login`. Everything else lives in
`s.echo.Group("/api/v1", auth.RequireBearer(s.services.Auth))` (`internal/application/http_server.go`): the
login route is registered on the bare Echo instance so the group middleware never sees it, and because the group
has a catch-all, an unknown `/api/v1/...` path is `401` without a token and `404` JSON with one.
`s.echo.RouteNotFound("/*", echo.NotFoundHandler)` on the root exists for the metrics `route` label: without it
an unmatched path outside the group reports the nearest node's template (`/healthz` would count as `/health`).
The spec coverage test skips `echo.RouteNotFound` routes, and `/metrics` is not on Echo at all — on the API port
it is a plain `404`, with a token and without one.

- **Tokens** (`internal/auth`): 32 random bytes, base64url on the wire, `hex(sha256)` in the `sessions` table.
  `IdleTTL` 30 days sliding, `AbsoluteTTL` 180 days from creation; `last_used_at` is written at most once per
  `TouchInterval` (1h) so reads do not turn into SQLite writes. `Service.Authenticate` re-reads the owner on every
  request (one `JOIN users`), so a deactivated user or a changed role takes effect on the next request — there is
  nothing cached client-side to invalidate. Password change (`PUT /me/password`) keeps the current session and
  revokes the rest; `PUT /users/:id/password`, `reset-password` and deactivation revoke all.
- **Login** returns `{token, expires_at, user}`; unknown email and wrong password are the same
  `401 INVALID_CREDENTIALS` (an unknown email still runs bcrypt against `Service.dummyHash`). Passwords are
  10…72 bytes, enforced by the `password` validator tag (`auth.RegisterPasswordValidation`) in both
  `services.newValidator()` and `handlers.newAPIValidator()`; bcrypt cost 12. `LoginRequest` checks only
  `max=72` — policy lives on the password-setting paths, so tightening it never locks existing users out.
- **Rate limiter** (`internal/auth/ratelimit.go`, in-memory sliding window): 10 attempts per IP per 5 min, 20 per
  email per hour, `429 RATE_LIMITED` + `Retry-After`; a successful login resets the email counter, a blocked
  attempt is not counted. The IP comes from `e.IPExtractor = auth.IPExtractor(config.TrustedProxies)`.
  **With `TRUSTED_PROXIES` empty the per-IP bucket is off** (`auth.WithoutIPLimit()`, wired in
  `NewHTTPServerWithObservability` when `Config.LoginLimiter` is nil): behind a proxy every client would share
  the proxy's address and ten stray failures would lock the whole family out, so only the per-email limit
  applies until the proxy CIDR is listed. Integration tests that need the IP limit pass
  `testhelpers.WithTrustedProxies(t, "192.0.2.0/24")` (httptest's `RemoteAddr`).
- **Role gates** are built from `auth.RequireRole(roles...)`: `adminOnly` for `/api/v1/users`,
  `DELETE /api/v1/categories/:id`, `DELETE /api/v1/accounts/:id`, `DELETE /api/v1/holdings/:id`, `/api/v1/backups` and `PUT /api/v1/family`;
  `financeAccess` (admin or member) for categories/accounts/transactions/budgets/stats; `GET /api/v1/family` and the `/auth/*`, `/me*` routes are
  open to any authenticated role. Wrong role → `403 FORBIDDEN`.
- **User writes are column-scoped, and session revocation rides along.** `UserRepository.Update` writes only
  `email`/`first_name`/`last_name`; password, role and `is_active` go through
  `UpdatePassword(ctx, id, hash, keepSessionID)` and `Patch(ctx, id, role, active)`, so an overlapping `PUT /me`
  cannot write a stale hash or `is_active` back. Both drop the owner's sessions in the **same** `BeginTx` as the
  write (`keepSessionID` survives, `uuid.Nil` keeps none), next to the "last active admin" check that returns
  `user.ErrLastAdmin` (= `services.ErrLastAdmin`) — a failed revoke can no longer leave a new password committed
  with the old sessions alive. That is why the user repository owns `DELETE FROM sessions`: `auth.Service` has no
  `RevokeAllSessions` and `auth.SessionRepository` no `DeleteByUser`. `PATCH /users/:id` applies role and
  `is_active` in that one `UPDATE`, so its `409` leaves nothing half-applied. Still open: a concurrent `Login` may
  check the old password before the commit and create its session after it.
- **Errors outside handlers** (`internal/application/error_handler.go`, `newAPIErrorHandler`): `RequireBearer` and
  `RequireRole` return `*echo.HTTPError` (they cannot import `handlers` — cycle), and the error handler renders
  the JSON envelope for every error, router 404 and panics included. A non-`*echo.HTTPError` becomes a bare
  `500` — the text goes to the log only, so read the server log when debugging one.
- Handlers take the author from `auth.FromContext(c)` → `*auth.Principal{SessionID, UserID, Email, Role}`
  (see `TransactionHandler.CreateTransaction`); `UserID` is **not** a field of
  `CreateTransactionRequest`, so sending it in the body does nothing.
- **One error envelope, one pagination shape** (`internal/application/handlers/helpers.go`): answer with
  `respondAPI`/`respondError(c, status, code, message, details...)`, never a hand-built `ResponseMeta`; validation
  failures are `422 VALIDATION_ERROR` with `error.details[{field, message, code}]`, while broken JSON and an
  unparseable id stay `400`. The one carve-out: a body date that does not parse is a field error, so
  `respondBindError` answers `422` and names the offending field (`date`, `start_date`, …) — `date.JSONField`
  digs the name out of the `*json.UnmarshalTypeError` that `date.Date.UnmarshalJSON` returns for that purpose. Every list runs its query params through `parsePagination(c)` (`defaultLimit=50`,
  `maxLimit=200`, out-of-range → `422`) and reports `meta.pagination {limit, offset, total}` — including the short
  lists, because the Android client generates from a `ListMeta` where `pagination` is required.
  `parsePagination` **writes the 422 itself** and returns the `errResponseAlreadyWritten` sentinel: callers return
  `ignoreWritten(err)`, never the raw error, or Echo's error handler writes a second response over it. Lists the
  repository returns whole are windowed with `pageSlice(items, page)` and answered by `respondList(c, items, page,
  len(all))`. `category_id` on `GET /transactions` repeats (`explode: true`, read via `c.QueryParams()`, duplicates
  collapse) and matches any of the ids; an older server reads only the first and narrows the filter silently. The `field` in `error.details` is the json name (`start_date`), because every handler validator comes
  from `newAPIValidator()` — plain `validator.New()` would report Go field names.

### Screenshot recognition: `POST /api/v1/transactions/recognize`

Candidates only: the model's answer is returned for review, the client saves the rows with ordinary
`POST /transactions` carrying its own `id`; no image is stored. The call is **not** idempotent — a repeat is a
paid call, so neither side retries it on its own.

- **Layers.** `internal/recognize` is pure (input/output types, `CheckImage` by signature, prompt, `Parse`,
  `Normalize`) and knows nothing of `llm`. `internal/infrastructure/llmengine` is the only importer of
  `github.com/lllypuk/llm`: it builds the request and maps failure classes onto `recognize.UnavailableError`.
  `services.RecognizeService` adds the family's categories, today in `family.Timezone`, the currency and
  `similar` (up to three transactions of the same amount and type within ±1 day — a badge, never a dedupe).
  Moving to `llm.Router` touches only `llmengine` and `recognizer()` in `run.go`.
- **Config.** `LLM_OLLAMA_HOST` empty (the default) builds no engine, and `ErrDisabled` becomes `503`;
  `LLM_MODEL` and `LLM_TIMEOUT` are validated only when the host is set, `LLM_TIMEOUT` ≤ 60 s because the
  deadline chain below is computed from it. The ollama.com key lives in the daemon — still no secrets.
- **Codes.** `503 RECOGNITION_UNAVAILABLE` for off, unreachable, `needs_configuration` or timed out
  (`Retry-After` in whole seconds, only when > 0); `502 RECOGNITION_FAILED` when the model answered and
  `Parse` refused it; `413 PAYLOAD_TOO_LARGE` over `BodyLimit("11M")` (the limiter error is dug out of the
  multipart reader with `errors.As`); `422` with `field: images[i]` for a non-image, an oversized or sixth part
  or a foreign field name; `408 REQUEST_TIMEOUT` when the upload outlives its deadline; `400` for a body that is
  not multipart. A `RetryNever` failure from `llm` is none of these and stays a `500`.
- **No second retry loop.** Transport retries live inside `llm.Client` (`Attempts = 2`, `MaxRetryAfter = 10s`,
  `Budget()` = 130 s). An answer that does not parse is `ErrBadAnswer` at once — `llm` accepts any text as a
  success, and asking again would double the bill for the same garbage.
- **Deadlines in two phases** (`handlers/recognize.go`). The route is skipped by the `ContextTimeout`
  middleware (`recognizeRoute` in `http_server.go`). Before the body is read, `http.NewResponseController` sets
  the write deadline to `now + uploadTimeout + Budget() + 15s` and the read deadline to `now + uploadTimeout`
  (`http.ErrNotSupported` from a recorder is tolerated); the upload runs under
  `context.WithTimeout(uploadTimeout)`, the engine under a fresh `Budget() + 5s`. Worst case 205 s, which the
  client's read timeout (210 s) and any proxy in front must outlast. The read deadline is not reset after the
  upload: `net/http` clears it when it starts its background read. `uploadTimeout` is
  `application.Config.RecognizeUploadTimeout` (0 → `handlers.RecognizeUploadTimeout`);
  `tests/integration/recognize_deadlines_test.go` checks the phases on a real socket.
- **Subcommand.** `go run ./cmd/server recognize a.png b.jpg` runs one real call with the categories from
  `DATABASE_PATH` (`OpenDatabaseNoMigrate`), prints the `Result` JSON to stdout and the engine's report to
  stderr; without `LLM_OLLAMA_HOST` it exits with 2. This is the only way to try real screenshots — CI has no
  model.
- **Metrics:** `ffs_recognitions_total{outcome}` and `ffs_recognition_duration_seconds`, written by the service
  through `services.RecognizeObserver`; `llm.Observer` is not wired yet.

### Metrics on a second listener

`internal/metrics` holds a private `prometheus.Registry` (no package-level instruments — `gochecknoglobals`) and
travels as a parameter; with `METRICS_ADDR` empty `run.go` builds no registry and opens no extra socket
(`testhelpers.SetupHTTPServer` always builds one — see "Testing" — but never a socket). It is a leaf package: consumers declare the one-method observer interfaces
(`services.BackupObserver`, `handlers.LoginObserver`, each with a `Nop*` for the cron process, for
`NewHTTPServer` and for tests), and the scrape collectors take narrow readers (`metrics.BackupLister`,
`metrics.StateReader` — the latter satisfied by `infrastructure.NewMetricsReader(db)`, a separate type, so no
repository interface or its mocks grow). `Handler()` runs `promhttp` with `ContinueOnError`: one failing
collector costs its own metric, not the whole scrape, and every `Collect` gets its own `ScrapeTimeout` (5s)
context because the HTTP timeout does not reach SQL.

`ffs_backups_total` and `ffs_backup_duration_seconds` are written inside `CreateBackup`, i.e. only for API
calls — the cron `compose run` is a different process with no registry, and is visible only through
`ffs_backup_latest_file_timestamp_seconds` (the newest file's `ModTime`, not "last successful run", so deleting
it moves the gauge back). Metric names and labels are a contract with the observability repository: the accepted
list is in `docs/plans/completed/20260915-14-prometheus-metrics.md`, the deployment side in `deploy/README.md`.
`ffs_recognition*` are not in that list yet — agreeing them is open in plan 15's Post-Completion.

**Working directory matters:** `./migrations` is resolved relative to the process CWD (`migrationsDir` in
`internal/bootstrap.go`), so both the server and the CLI subcommands must be started from the repo root.

## Database & migrations

The whole schema lives in `migrations/001_consolidated.{up,down}.sql`
(tables: families, users, categories, accounts, transactions, account_balances, budgets, holdings,
holding_values, holding_plans, sessions) — that file is the readable
picture of the database, and a fresh DB is built from it.

**Editing `001` does not touch an existing database.** golang-migrate stores only the version number, so on a DB
that already has version 1 `Up()` returns `ErrNoChange` and the new DDL is skipped silently; the test path starts
from an empty in-memory DB and will not show this. Before `v0.1.0` that was fine — every database was recreated
with `make db-reset`. **Since `v0.1.0` is deployed, a schema change is a numbered migration as well**: write the
DDL into `001` (so a new install gets it) *and* a `NNN_*.{up,down}.sql` applying it to a live database, like
`002_budgets_name_period_partial_unique` (which rebuilds `budgets` — SQLite cannot `DROP INDEX` the implicit
`sqlite_autoindex_*` behind a table-level `UNIQUE`). `002` is therefore a no-op rebuild on a fresh DB, and its
table definition must stay frozen on the `v0.2.0` schema: a column added later lives in `001` *and* in its own
`NNN` (`004` — `budgets.recurring`, `budgets.series_id`), so on a fresh DB `002` drops what `001` created and
`004` puts it back. Never add a column to the `budgets` block inside `002` — it must keep reproducing the
released schema, or the upgrade path it tests is not the one the server runs. Cover it in
`internal/infrastructure/migrations_test.go`: `Migrate(1)` puts the released schema back, so the upgrade path is
testable. The same freeze holds for `transactions` inside `005_accounts`: it rebuilds the table (not
`ADD COLUMN`, which would fail on a fresh DB and put `account_id` at a different `cid`), so its definition stays
on the `v0.6.0` schema, and the next transaction column goes into `001` *and* its own `NNN`. See `migrations/README.md`, and `make migrate-create` for the reminder.

`go run ./cmd/server migrate` prints the schema version, `migrate --to N` moves it in either direction
(`--to 0` is rejected: golang-migrate answers `Migrate(0)` with "file does not exist"). This is the rollback
path — an image refuses to start on a version it has no file for, so the schema is stepped down with the
*new* image, with the container stopped, before the old one is deployed (`deploy/README.md`).

Two independent code paths apply migrations, and **both must keep working**:

- production/dev: golang-migrate (`internal/infrastructure/migrations.go`)
- tests: `internal/testhelpers/sqlite.go` reads and executes the `*.up.sql` files directly

`testhelpers.SQLiteTestDB.CleanTables` has a hardcoded, FK-ordered table list — add any new table to it
(`account_balances` before `transactions` before `accounts`: both FKs to `accounts` are `RESTRICT`;
`holding_values` and `holding_plans` before `holdings`).

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
  - `SetupHTTPServer` always builds a registry: `ts.Metrics` is the same instance the server middleware and the
    backup service write to, scraped in tests through `ts.Metrics.Handler()` without opening a socket.
  - Recognition is off on the stand unless `testhelpers.WithRecognizer(r)` is passed;
    `WithRecognizeUploadTimeout(d)` shortens the upload phase, `testhelpers/multipart.go` builds multipart bodies
    and PNG/JPEG images.
  - `testhelpers.RepoRoot(t)` walks up to `go.mod`; use it for anything cwd-relative (`openapi.yaml`, migrations
    in `bootstrap_test.go` via `t.Chdir`) — `go test` runs with cwd = the package directory.
- `testhelpers/factories.go` — `CreateTestFamily`, `CreateTestUser`, etc.
- Handler unit tests put a `*auth.Principal` into the context under `auth.ContextKey` — see `principalContext`
  in `internal/application/handlers/auth_test.go` and reuse it rather than hand-rolling contexts.
- Naming: `TestXxx_Method_Scenario` (e.g. `TestTransactionService_CreateTransaction_Success`).
- `testpackage` is enabled: use an external `package foo_test` unless the path is excluded in `.golangci.yml`
  (`internal/observability/`, `internal/services/dto/`, `tests/`).
- Use testify `require` for fatal preconditions, `assert` for the rest (`testifylint` enforces correct usage).

## Conventions

- **Money is `money.Minor`** (`internal/domain/money`) — `int64` in minor units, `amount_minor`/`spent_minor`
  in JSON and in the DB. No `float64` sums anywhere: percentages come from `Minor.Percent(total)`, averages
  from `Minor.DivRound(n)` (half-up), and SQL does `SUM`, never `AVG`. The currency is the family's;
  `PUT /api/v1/family` refuses to change it once a transaction exists (`services.ErrCurrencyLocked` →
  `409 CURRENCY_LOCKED`).
- **A transaction date is `date.Date`** (`internal/domain/date`) — a calendar `YYYY-MM-DD` with no time or zone,
  `TEXT` in SQLite (CHECK GLOB). Only `created_at`/`updated_at`/`expires_at` stay RFC3339 UTC. Period bounds
  ("current month") are computed in `family.Timezone`, not in the server's zone.
- **`POST` of a transaction, budget, category or account is idempotent** when the body carries a client-generated
  `id` (any valid UUID): an existing record answers `200` with itself and the repeated body is ignored — the id is
  the only thing compared. The check is a plain read-then-insert; two simultaneous retries can still collide.
- **Budget business refusals are `409` with their own codes** — `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`,
  `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`; only shape errors (`amount_minor` out of range,
  reversed dates) stay `422`. Periods of one
  scope overlap **inclusively** — a shared boundary day is a conflict, because spending is summed over
  `date >= start AND date <= end`. The occupancy predicate matches on scope *or* on name, and the two answer with
  different codes (`takenBy.conflict`): a busy scope is `BUDGET_OVERLAP` and moves with the dates, a name taken in
  another scope is `BUDGET_NAME_EXISTS` and only renaming clears it. `is_active` is a soft-delete marker and nothing else: it is not in
  `UpdateBudgetRequest`, and `GetByID` filters on it, so a deleted budget is `404` for GET/PUT/DELETE alike.
  A deleted row keeps its primary key, so a `POST` reusing that `id` cannot be idempotent: the repository tells the
  two unique violations apart (`budgets.id` in the message → `budget.ErrIDExists`) and the client gets
  `409 BUDGET_ID_EXISTS`, not a name conflict it could never fix by renaming. The name+period uniqueness, by
  contrast, is the partial index `idx_budgets_name_period_active` (`WHERE is_active = 1`), so after a delete the
  same name and period can be created again under a new `id`.
- **A recurring budget is the tail of a series.** `recurring: true` marks the tail; `series_id` (the first
  instance's id) ties the instances together. There is no background job: the next calendar period is
  materialised lazily on reads (`GET /budgets`, `GET /stats/summary` → `BudgetServiceImpl.advanceRecurring` →
  `Advance`, at most `maxAdvancePeriods` = 120 steps per call), the flag moves to the new instance and the past
  ones stay as history with the same `series_id`. Dates of a recurring budget must match the calendar period
  (`budget.ValidateRecurring`; `custom` cannot recur) and a series member's dates are immutable — both are `422`.
  A `PUT` on an instance that is no longer the tail is `409 BUDGET_NOT_TAIL`: the client re-reads and edits the
  new tail. Stopping a series that never advanced clears `series_id` as well — otherwise a one-instance "series"
  would keep its dates frozen forever. A period already taken by a manual budget of the same scope or name is **stepped over**, not a stop
  of the series — one manual edit must not kill it silently. Occupancy is one predicate inside the repository
  transaction (`taken`, `budget_repository_sqlite.go`), never a read in the service, so a manual `POST` and a
  materialisation cannot both land on one month; `Update` never writes the `spent_minor` it is handed —
  it recomputes it from the transactions inside its own transaction (`syncSpent`), so a shifted period and the
  expense sum commit together and a concurrent expense cannot be overwritten by a stale total.
- **Accounts are optional on a transaction.** In `PUT /transactions/:id` a missing (or `null`) `account_id` keeps
  the stored one — the client sends with `explicitNulls = false`, so `null` cannot mean "unbind"; unbinding is
  `clear_account: true`, both at once is `422`. An unknown or archived account is `422 field: account_id`, but only
  when the id differs from the stored one: editing a transaction whose account was archived later still passes.
  Deleting an account used by a transaction **or** a balance row is `409 ACCOUNT_IN_USE` (both FKs `RESTRICT`,
  a cascade would take the balance history); a reissued card is archived. The name is unique per family by
  `names.Key` (Go-side `ToLower(TrimSpace)`, since `NOCASE` folds ASCII only) — never change it after a release,
  it defines the stored `name_key`.
- **A reconciliation stores only month-end balances.** `PUT /accounts/:id/balances/:month` keeps one signed
  `balance_minor` per account and month (a credit card is negative); `GET /stats/reconciliation` computes
  `gap = (closing − opening) − (income − expense)` on read, so a late transaction closes the gap without a new
  `PUT`, and the correction is an ordinary transaction the client prefills. Which accounts count on each edge
  (created inside the month → `opening = 0`, archived without a row → `closing = 0`, created after the month →
  absent) is `reconciliationService.Summary`; an empty list is `null`, not a zero gap. A future month is `422`.
  A transfer between own accounts is not a transaction — booked as one, it shows up as `gap`. Having a balance
  means having the row — a `0` counts, and blocks `CURRENCY_LOCKED` like a transaction does
  (`FamilyRepository.HasMonetaryData`: transactions, balances, holding values and holding plans, archived and
  zero ones included).
- **Holdings carry their sign in `side`, not in the number.** `value_minor >= 0`, and `side` is fixed at creation
  (`UpdateHoldingRequest` has no such field) — changing it would flip the whole history. A holding's value on a
  day is its latest snapshot with `date <=` that day, carried forward with no expiry (a flat is revalued once a
  year); before its first snapshot it counts for nothing. A future snapshot date is `422`, and `current` is cut
  at today in `family.Timezone` all the same, so it equals the holding's share of the last bucket of the default
  `GET /stats/net-worth`. `is_archived` only hides a holding from the list: the series has no archive filter,
  so archiving never rewrites the past — a sold or repaid holding gets a `0` snapshot. `DELETE /holdings/:id`
  (admin) cascades its snapshots and changes past buckets. `assets_minor`/`liabilities_minor`/`net_minor` are
  sums, so the spec types them as `int64` without `Money.maximum`; in Go they stay `money.Minor`.
- **A holding's monthly plan is a row in `holding_plans` that exists only while non-zero.** In `PUT /holdings/:id`
  a missing number is left alone and `0` clears it; both at `0` delete the row, so "has a plan" is "has a row"
  (that is all `HasMonetaryData` checks) and `plan_updated_at` does not outlive the plan. The holding and its plan
  are written in one `BeginTx`, merged inside it — never through `r.db` or `GetByID` there, with one connection
  that deadlocks. The "План в месяц" total is the client's, over non-archived holdings; there is no server aggregate.
- **Fractions come in two units.** Shares (`share`, `*_delta`, `stats.budgets[].utilization`) are 0…1; fields named
  `percentage` and `budgets[].utilization` on `/budgets` are percent 0…100. Both are documented per field in
  `docs/api/openapi.yaml`.
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
Android app (one instance = one family, two users, `ffs.shatrov.tech` behind Caddy). Plans 01–10 and 12–22 are
done (`docs/plans/completed/`, 06 = the Android client, 07 = its budgets tab, 08 = its settings screen,
09 = the server findings of 07–08, 10 = `/reports` removed and `GET /stats/monthly` added, 12 = recurring
budgets, both sides, 13 = the client's UI audit: segments instead of chips, a FAB on "Операции", empty states
with an action, password visibility — client only, the contract did not move; 14 = the `/metrics` listener).
Tagged releases: server `v0.3.0` (plan 10), `v0.4.0` (plan 12), `v0.8.0` (plans 14–18), `v0.9.0` (plan 20), client up to
`app-v0.5.0`, then `app-v0.10.0` (client side of plans 13–18), `app-v0.11.0` (plan 19), `app-v0.12.0` (plan 20). `v0.5.0`–`v0.7.0` and `app-v0.6.0`–`app-v0.9.0` were never cut
and will not be: a server tag deploys its image, so it cannot go on an old commit. Plans 15 (screenshot
recognition), 16 (accounts and the monthly reconciliation), 17 (holdings and net worth) and 18 (a holding's
monthly plan) shipped together in `v0.8.0`/`app-v0.10.0`; older clients keep working against a newer server.
Plan 19 (client `0.11.0`, contract untouched) closes half of plan 11: a journal that lets an unfinished
screenshot import survive process death, and the "Обзор" screen over `summary` + `monthly`; multi-select over
transactions and `bulkDeleteTransactions` remain.
Plan 20 (server `v0.9.0`, client `0.12.0`, one MR) replaces plan 16's bank-figure reconciliation with month-end
balances (`008_account_balances`) and breaks the contract: a `0.11.0` client loses its reconciliation screen
and, silently, the reconciliation card on the home screen (the old model requires `recorded_minor`).
`008.down` recreates `account_reconciliations` empty; `migrate --to 8 → 7 → 8` was run on a copy of the production
DB before `v0.9.0` was tagged (19.09.2026).
Plan 21 (client `0.13.0`, contract untouched) adds a light theme with a System/Light/Dark switch in settings.
Plan 22 (server `v0.10.0`, client `0.14.0`, one MR, additive) takes categories off the nav bar, draws icon and color
as an avatar and makes `category_id` on `GET /transactions` an array; deploy the server first.

`docs/api/openapi.yaml` is the contract for `/api/v1` (plus `GET /health`) — the Android client generates
from it, and code and spec now match. **A registered route with no operation in the spec fails `make test`**
(`tests/integration/openapi_coverage_test.go`: `TestOpenAPISpec_CoversRegisteredRoutes`, plus
`TestOpenAPISpec_OperationsHaveIDAndErrorResponse` requiring an `operationId` and a 4xx `$ref: Error` on every
operation). The reverse also fails it (`TestOpenAPISpec_DescribesOnlyRegisteredRoutes`): spec and routes match
exactly, with no exceptions. See `docs/api/README.md`.
**Deployment** is `deploy/` — see `deploy/CLAUDE.md` and `deploy/README.md`.

When runtime/dev commands disagree between documents, `Makefile` + this file win.

## Android client (`android/`)

The Kotlin/Compose client to `/api/v1` lives in this repository (decision A-13). See `android/CLAUDE.md` and
`android/core/api/CLAUDE.md`; commands are `make -C android <target>`; CI details are in `android/CLAUDE.md`.

## CI

GitLab (`.gitlab-ci.yml`), the repository lives on `gitlab.shatrov.tech`; GitHub is a read-only mirror.
Pipeline layout, runner, mirror and version pins: the `gitlab-ci` skill.
