# План 01 — Контракт API

Первый из пяти планов перехода на API-only ([spec 005](../specs/005-api-only-redesign.md)).
Здесь фиксируется целевой контракт до того, как его начнут реализовывать планы 02–04.

## Overview

- `docs/api/openapi.yaml` становится единственным описанием `/api/v1`: Android-приложение
  генерирует клиент из него, планы 02–04 приводят код в соответствие.
- Тест сверяет зарегистрированные роуты `/api/v1/*` со спецификацией: роут без описания
  роняет `make test`. Обратное (описано, не реализовано) допустимо до конца плана 04.
- `/health` отдаёт настоящую версию сборки вместо `1.0.0` из `internal/run.go:51`.
- `docs/patterns/api_standards.md` переписывается под решения A-01, A-05…A-09.

## Context (from discovery)

- Роуты: `internal/application/http_server.go:197-246` (`setupRoutes` с `:177`); envelope и коды —
  `internal/application/handlers/types.go`, `errors.go`, `helpers.go`.
- Версия: `observability.NewHealthService(version)` (`internal/observability/health.go:51`),
  значение захардкожено в `internal/run.go:51`. Сборка — `Makefile:16`, `docker/Dockerfile:21`.
  `docker.yml:74-76` и `release.yml` уже передают `build-args: VERSION=…`, но `Dockerfile` не
  объявляет `ARG VERSION` и не подставляет его в `-ldflags`.
- `go.yaml.in/yaml/v3` уже есть в `go.mod` как indirect — для теста покрытия хватит его.
- Текущие расхождения контракта перечислены в spec 005, раздел «Контракт».

## Development Approach

- **testing approach**: Regular (код, затем тесты).
- Каждая задача завершается зелёными `make fmt`, `make test`, `make lint` (0 issues).
- Тесты — обязательная часть каждой задачи, отдельными пунктами.
- Изменение объёма — правкой этого файла (➕ новые задачи, ⚠️ блокеры).

## Testing Strategy

- unit: `internal/version`, парсер спецификации в тесте покрытия.
- integration: `tests/integration/openapi_coverage_test.go` на `testhelpers.SetupHTTPServer`.
- e2e-тестов в проекте нет.

## Progress Tracking

- `[x]` сразу по завершении пункта; ➕ — новые задачи; ⚠️ — блокеры.

## Solution Overview

Спецификация пишется руками (OpenAPI 3.1, один файл, схемы в `components`). Генераторов
кода на сервере нет: контракт маленький, а генератор добавил бы зависимость и шаг сборки.
Синхронизацию держит тест покрытия, а не дисциплина.

## Technical Details

Целевой контракт (детали — в spec 005):

| Группа | Роуты |
|---|---|
| auth | `POST /auth/login` (публичный), `POST /auth/logout`, `GET /auth/sessions`, `DELETE /auth/sessions/{id}` |
| me | `GET /me`, `PUT /me`, `PUT /me/password` |
| family | `GET /family`, `PUT /family` (admin) |
| users (admin) | `GET /users`, `POST /users`, `GET|PUT /users/{id}`, `PATCH /users/{id}` (`is_active`, `role`), `PUT /users/{id}/password`. `DELETE /users/{id}` в контракт не входит: удаление пользователя сейчас — `is_active = 0` (`user_repository_sqlite.go:315`), а транзакции ссылаются на него `ON DELETE RESTRICT`; деактивация через `PATCH` — единственный путь (план 03) |
| health | `GET /health` — вне `/api/v1`, но в спецификации: клиент читает `setup_complete` и `version` |
| categories | `GET|POST /categories`, `GET|PUT|DELETE /categories/{id}` (DELETE — admin) |
| transactions | `GET|POST /transactions`, `GET|PUT|DELETE /transactions/{id}`, `POST /transactions/bulk-delete` |
| budgets | `GET|POST /budgets`, `GET|PUT|DELETE /budgets/{id}` |
| reports | `GET|POST /reports`, `GET|DELETE /reports/{id}`, `GET /reports/{id}/export` (CSV) |
| stats | `GET /stats/summary?from&to` |
| backups (admin) | `GET|POST /backups`, `GET /backups/{name}/download`, `DELETE /backups/{name}` |

Общие схемы: `Error` (одна форма, A-08), `Pagination` в `meta`, `Money` — `integer/int64`
с описанием «минимальные единицы валюты семьи», `CalendarDate` — `string/date`.
Заголовок `Authorization: Bearer <token>`; ответы `401 UNAUTHORIZED`, `403 FORBIDDEN`,
`409 SETUP_REQUIRED`, `429` с `Retry-After`.

Версия: пакет `internal/version` с переменной `Version` (значение по умолчанию `dev`),
подставляется `-ldflags "-X family-budget-service/internal/version.Version=$(VERSION)"`;
`VERSION ?= $(shell git describe --tags --always --dirty)` в Makefile, `ARG VERSION` в Dockerfile.

## Implementation Steps

### Task 1: Версия сборки в `/health`

**Files:**
- Create: `internal/version/version.go`, `internal/version/version_test.go`
- Modify: `internal/run.go`, `Makefile`, `docker/Dockerfile`, `.golangci.yml`

- [x] создать `internal/version` с `Version = "dev"` и функцией `String()` (единственное разрешённое место для package-level var — добавить исключение `gochecknoglobals` для этого файла в `.golangci.yml` с объяснением)
- [x] передать `version.String()` в `observability.NewHealthService` вместо литерала
- [x] `Makefile`: переменная `VERSION`, `-X` в `build`, `run`, `run-local`; `docker/Dockerfile`: `ARG VERSION=dev`, тот же `-X`; workflows не трогать — `docker.yml` и `release.yml` уже передают `VERSION`
- [x] тест: `String()` возвращает `dev` по умолчанию; интеграционный тест `/health` — поле `version` непустое
- [x] `make fmt && make test && make lint` — зелёные

### Task 2: `docs/api/openapi.yaml` — целевой контракт

**Files:**
- Create: `docs/api/openapi.yaml`, `docs/api/README.md`

- [x] описать все группы из «Technical Details», включая `GET /health` (`status`, `version`, `setup_complete`): параметры, тела, ответы, коды ошибок, роль в `description` каждой операции
- [x] `components/schemas`: `Error`, `Meta`, `Pagination`, `User`, `Family`, `Category`, `Transaction`, `Budget`, `Report`, `StatsSummary`, `Backup`, `Session`, `LoginRequest`, `LoginResponse`; деньги — `amount_minor` int64, даты операций — `format: date`
- [x] `components/securitySchemes.bearerAuth`; `security` по умолчанию на всех операциях, кроме `POST /auth/login`
- [x] `docs/api/README.md`: как читать спецификацию, как генерировать Kotlin-клиент (одна команда, без привязки к конкретному генератору), правило «роут без описания — красный тест»
- [x] проверить файл любым OpenAPI-валидатором (npx `@redocly/cli lint` или `swagger-cli`) — результат зафиксировать в PR, в CI не добавлять

⚠️ `DELETE /api/v1/users/{id}` всё же описан — с `deprecated: true` и пояснением: роут зарегистрирован
в текущем коде, а тест покрытия из задачи 3 требует описания каждого существующего роута. Уйдёт вместе
с роутом в плане 03.

### Task 3: Тест покрытия роутов спецификацией

**Files:**
- Create: `tests/integration/openapi_coverage_test.go`
- Modify: `go.mod` (`go.yaml.in/yaml/v3` → direct)

- [ ] загрузить `docs/api/openapi.yaml` через `testhelpers.RepoRoot(t)`, собрать множество `METHOD /path` из `paths`, нормализуя `{id}` ↔ `:id`
- [ ] пройти `e.Routes()` тестового сервера, взять роуты с префиксом `/api/v1` плюс `GET /health`, каждый должен быть в множестве; сообщение об ошибке — список недостающих. Обратная проверка (операция без роута) добавляется в плане 04, когда контракт реализован целиком
- [ ] отдельная проверка: у каждой операции есть `operationId` и хотя бы один ответ 4xx с `$ref: Error`
- [ ] `make test` — зелёный; в момент написания все существующие роуты должны попасть в spec (сверить с `http_server.go:197-246`, в том числе `POST /reports` со статусом 501 как временным)

### Task 4: Переписать `docs/patterns/api_standards.md`

**Files:**
- Modify: `docs/patterns/api_standards.md`

- [ ] раздел аутентификации: bearer-токены (A-01), таблица кодов `401/403/409/429`, ничего про cookie и CSRF
- [ ] пагинация `limit/offset/total` (A-08) вместо `page/page_size`; убрать `links`
- [ ] единая форма ошибки, `422` для валидации; убрать «третью» форму
- [ ] деньги и даты (A-05, A-06), идемпотентный `POST` с клиентским `id` (A-07)
- [ ] убрать раздел про версионирование с двумя одновременными версиями и лимиты «100 req/h анонимно» — их нет и не будет; оставить `X-Request-ID`

### Task 5: Verify acceptance criteria

- [ ] `openapi.yaml` валиден, тест покрытия зелёный, `/health` показывает `git describe`
- [ ] `make pre-commit` зелёный
- [ ] `docker build --build-arg VERSION=test -f docker/Dockerfile .` собирается, `/health` внутри контейнера отдаёт `test`

### Task 6: [Final] Update documentation

- [ ] `README.md`: раздел «API Readiness» → ссылка на `docs/api/openapi.yaml` и spec 005
- [ ] `CLAUDE.md`: абзац про `internal/version` и правило «новый роут → openapi.yaml, иначе тест красный»
- [ ] `docs/README.md`: строка про `docs/api/`
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

**Manual verification:** прогнать `openapi.yaml` через генератор Kotlin-клиента, который будет
использоваться в Android-проекте, и убедиться, что схемы `int64`/`date` мапятся в `Long`/`LocalDate`.
