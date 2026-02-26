# Repository Guidelines

## Project Structure & Module Organization

This is a Go monorepo for a family finance service with layered architecture.

## Agent Context Bootstrap

- Before starting work, load `CLAUDE.md` into the agent context. It contains repo-specific development workflows, commands, and operational notes that complement this file.
- Use `README.md` + `CLAUDE.md` as the primary source for current runtime/dev commands when documentation appears inconsistent.

- `cmd/server/`: application entry point
- `internal/domain/`: domain models and validation
- `internal/services/`: business logic (use-cases)
- `internal/application/`: API server and HTTP handlers
- `internal/web/`: web UI handlers, middleware, templates
- `internal/infrastructure/`: SQLite repositories, migrations integration
- `internal/observability/`: logging and health checks
- `tests/integration/`, `tests/benchmarks/`: cross-layer tests and benchmarks
- `migrations/`: consolidated SQL migrations (`001_consolidated.up/down.sql`)
- `deploy/`: self-hosted deployment scripts, compose files, and systemd units
- `docs/`, `docs/backlog.md`: design notes and implementation backlog
- `CLAUDE.md`: repo-specific agent guidance (must be added to context)

## Build, Test, and Development Commands

- `make run-local`: run locally with SQLite (`./data/budget.db`) and dev env vars
- `make build`: build binary to `./build/family-budget-service`
- `make test`: run all tests (`go test ./...`)
- `make test-unit`: run internal package tests
- `make test-integration`: run tests from `./tests/...`
- `make test-coverage`: generate `coverage.out` and `coverage.html`
- `make fmt`: format Go code (`go fmt ./...`)
- `make lint`: run `golangci-lint`
- `make pre-commit`: format + test + lint

After any code change, run `make fmt`, `make lint`, and `make test` before handing off work or opening a PR.

## Coding Style & Naming Conventions

- Follow standard Go style; always run `gofmt` before committing.
- Use idiomatic Go naming: exported `PascalCase`, internal `camelCase`.
- Keep handlers thin; prefer business logic in `internal/services/`.
- File names should be descriptive and snake_case-like for Go (`transaction_service.go`, `user_repository_sqlite.go`).
- Add tests alongside code using `_test.go`.

## Testing Guidelines

- Use Go `testing` plus `testify` (`assert`, `require`, `mock`).
- Prefer unit tests for services/repositories and integration tests in `tests/integration/`.
- Test names follow `TestXxx_*` patterns (e.g., `TestTransactionService_CreateTransaction_Success`).
- For sandboxed environments, prefer tests that do not require opening local sockets unless necessary.

## Commit & Pull Request Guidelines

- Current history follows conventional prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `security:`.
- Keep commits focused and scoped to one change.
- PRs should include:
  - clear summary and rationale
  - linked issue/task (`docs/backlog.md` and/or related task doc if applicable)
  - test evidence (`make test`, targeted `go test ...`)
  - screenshots for UI/template changes

## Security & Configuration Tips

- Configure secrets via env vars (`SESSION_SECRET`, `CSRF_SECRET`); do not commit real secrets.
- Use `make security-check` for `gosec` and `govulncheck`.
- When changing schema, update consolidated migrations and test locally (`make run-local` + relevant tests).
