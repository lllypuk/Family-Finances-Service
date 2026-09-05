# План 02 — Полнота API

Второй план перехода на API-only ([spec 005](../specs/005-api-only-redesign.md)).
Цель — чтобы всё, что умеет веб-интерфейс, было доступно через `/api/v1` **до** его удаления
в плане 03. Аутентификация пока прежняя (cookie-сессия), деньги пока `float64`: контракт
`openapi.yaml` в этих двух местах догоняют планы 03 и 04.

## Overview

- Логика, живущая в веб-обработчиках (дашборд, генерация отчётов, CSV), переезжает в сервисы
  и получает JSON-роуты.
- Появляются недостающие роуты: список пользователей, семья, бэкапы, bulk-delete.
- Единый envelope ошибок и пагинация `limit/offset/total` на всех списках (A-08).
- Заглушки (плановые отчёты, прогноз, бенчмарки, алерты бюджетов) удаляются из интерфейсов.

## Context (from discovery)

- Дашборд: `internal/web/handlers/dashboard.go:242-915` (`buildMonthlySummary`,
  `calculatePreviousData`, `calculateChanges`, `buildBudgetOverview`, `buildRecentActivity`,
  `buildCategoryInsights`, `buildEnhancedStats`), тесты — `dashboard_test.go:413-1223`.
- Отчёты: диспетчер `generateReport` (`internal/web/handlers/reports.go:240`),
  `convertReportDataToStandard` (`:654`), CSV (`:533-639`); в сервисе `ReportService.ExportReport`
  и `ExportReportData` (`internal/services/report_service.go`), заглушки `:1074-1124`.
- `FamilyHandler` (`internal/application/handlers/families.go`) не зарегистрирован.
- `BackupService` (`internal/services/backup_service.go`) — каталог `<dir(db)>/backups`,
  compose монтирует `/backups`; единственный HTTP-вход — веб-админка.
- Envelope: `respondAPI`/`respondError` (`internal/application/handlers/helpers.go:24,31`),
  валидация — своя форма (`:74`), `users.go` и `categories.go` собирают `ResponseMeta` вручную.
- Пагинация: только транзакции, `limit/offset` без `total` (`transactions.go:345-371`);
  `CountTransactions` в сервисе не используется.
- `dto/api_mappers.go` почти не используется (только `ToCategoryAPIResponse`).

## Development Approach

- **testing approach**: Regular.
- Порядок задач важен: сервисы (1–2) раньше роутов, envelope (7) последним, чтобы менять
  ответы один раз.
- Каждая задача: `make fmt && make test && make lint` зелёные; тесты — отдельными пунктами.
- Веб-обработчики после переноса логики становятся тонкими обёртками над сервисами и
  продолжают работать до плана 03 — их тесты не удалять, только адаптировать.

## Testing Strategy

- unit: сервисы с моками репозиториев (по образцу `internal/services/*_test.go`), обработчики
  через `internal/application/handlers/*_test.go`.
- integration: `tests/integration/*` через `ts.Auth(t)` / `ts.AuthAs(t, role)` — cookie и CSRF
  пока обязательны (`sess.Apply(req)`).
- План 01 уже описал целевые роуты в `docs/api/openapi.yaml`; тест покрытия ловит расхождение
  формы пути. Если реализация отклоняется от спецификации — править спецификацию в той же задаче.

## Progress Tracking

- `[x]` по завершении; ➕ новые задачи; ⚠️ блокеры.

## Solution Overview

Новый `StatsService` берёт на себя вычисления дашборда: вход — период `[from, to]`, выход —
`dto.StatsSummary` (доходы, расходы, баланс, дельты к прошлому периоду, топ категорий с долями,
прогресс активных бюджетов, последние операции). Веб-дашборд рендерит этот DTO, API отдаёт
его как JSON. Форматирование («3 дня назад», строки валют) остаётся в веб-слое и умрёт с ним.

`ReportService.GenerateReport(ctx, req)` инкапсулирует диспетчер по типу отчёта и конвертацию в
`report.Data`; `POST /api/v1/reports` вызывает его и сохраняет результат; веб делает то же.

## Technical Details

- `GET /stats/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` — по умолчанию текущий месяц.
- `POST /reports` — тело как `CreateReportRequest`, ответ `201` с сохранённым отчётом.
- `GET /reports/{id}/export` — `text/csv`, `Content-Disposition: attachment`.
- `GET /users`, `GET /family`, `PUT /family` (admin).
- `POST /transactions/bulk-delete {ids: [uuid]}` → `200 {deleted: n}`; отсутствующие id
  игнорируются, чужих быть не может (одна семья).
- Бэкапы: `POST /backups` → `201 {name, size, created_at}`; `GET /backups`;
  `GET /backups/{name}/download` — файл; `DELETE /backups/{name}`. Каталог — `BACKUP_DIR`
  (по умолчанию `<dir(db)>/backups`, в compose `/backups`). Restore через API не делаем (A-11).
- Envelope: `respondError(c, status, code, message, details)`; валидация → `422 VALIDATION_ERROR`
  с `details[{field, message, code}]`; `respondValidationErrors` и inline-`ResponseMeta` удаляются.
- Списки: `meta.pagination {limit, offset, total}`; лимит по умолчанию 50, максимум 200 —
  на всех списках без исключений (A-08), включая короткие: клиент генерируется из
  `ListMeta`, где `pagination` обязательна.

## Implementation Steps

### Task 1: `StatsService` — перенос вычислений дашборда

**Files:**
- Create: `internal/services/stats_service.go`, `internal/services/stats_service_test.go`,
  `internal/services/dto/stats_dto.go`
- Modify: `internal/services/interfaces.go`, `internal/services/container.go`,
  `internal/web/handlers/dashboard.go`, `internal/web/handlers/dashboard_test.go`

- [x] `dto.StatsSummary` и вложенные типы (`PeriodTotals`, `CategoryShare`, `BudgetProgress`, `RecentTransaction`)
- [x] `StatsService.Summary(ctx, from, to)`: перенести `buildMonthlySummary`, `calculatePreviousData`, `calculateChanges`, `buildBudgetOverview`, `buildRecentActivity`, `buildCategoryInsights` из `dashboard.go`; `buildEnhancedStats`/`buildForecast` не переносить — удалить вместе с их партиалами
- [x] `DashboardHandler` строит view-модель из `StatsSummary`; функции вычислений из обработчика удалить
- [x] перенести unit-тесты вычислений из `dashboard_test.go:413-1223` в `stats_service_test.go` (моки репозиториев — по образцу `helpers_test.go`)
- [x] тесты ошибок: пустой период, репозиторий вернул ошибку, `from > to`
- [x] `make test` — зелёный, тесты веб-дашборда адаптированы

### Task 2: `ReportService.GenerateReport` и чистка заглушек

**Files:**
- Modify: `internal/services/report_service.go`, `internal/services/interfaces.go`,
  `internal/services/report_service_test.go`, `internal/web/handlers/reports.go`,
  `internal/web/handlers/reports_test.go`

- [x] `GenerateReport(ctx, dto.ReportRequestDTO) (*report.Report, error)` (используется существующий DTO; `SaveReport` теперь принимает готовый `*report.Report`) — диспетчер по `report.Type` и конвертация в `report.Data` из `reports.go:240,654-800`
- [x] удалить из интерфейса и реализации `ScheduleReport`, `GetScheduledReports`, `UpdateScheduledReport`, `DeleteScheduledReport`, `ExecuteScheduledReport`, `GenerateSpendingForecast`, `CalculateBenchmarks` (заглушки `report_service.go:1074-1124`) и их тесты
- [x] веб-обработчик и HTMX-партиал `reports/generate` вызывают `GenerateReport`; локальный диспетчер удалить
- [x] тесты `GenerateReport`: по одному на каждый из пяти типов, ошибка неизвестного типа, ошибка репозитория
- [x] `make test` — зелёный

### Task 3: Роуты отчётов и статистики

**Files:**
- Create: `internal/application/handlers/stats.go`, `internal/application/handlers/stats_test.go`
- Modify: `internal/application/handlers/reports.go`, `reports_test.go`,
  `internal/application/http_server.go`, `tests/integration/reports_test.go`,
  `docs/api/openapi.yaml`

- [x] `POST /api/v1/reports` → `GenerateReport` + `SaveReport`, `201`; убрать `501`
- [x] `GET /api/v1/reports/:id/export` — CSV через `ReportService.ExportReport`; перенести CSV-писатели из `internal/web/handlers/reports.go:533-639` в сервис, веб вызывает сервис
- [x] `GET /api/v1/stats/summary` — парсинг `from`/`to` (`YYYY-MM-DD`), по умолчанию текущий месяц
- [x] тесты обработчиков: успех, невалидные даты, ошибка сервиса; интеграционные: сгенерировать отчёт → получить → экспорт CSV с ожидаемым заголовком
- [x] `openapi.yaml` обновлён; `make fmt && make test && make lint` — зелёные

### Task 4: Пользователи и семья

**Files:**
- Modify: `internal/application/handlers/users.go`, `users_test.go`, `families.go`,
  `families_test.go`, `internal/application/http_server.go`, `internal/services/user_service.go`,
  `user_service_test.go`, `tests/integration/users_test.go`, `tests/integration/families_test.go`,
  `docs/api/openapi.yaml`

- [x] `GET /api/v1/users` (admin) через `UserService.GetUsers`
- [x] зарегистрировать `GET /api/v1/family` и `PUT /api/v1/family` (admin); `GetFamilyMembers` удалить — дублирует `GET /users`
- [x] `PATCH /api/v1/users/:id {role}` через `ChangeUserRole`; в сервисе — запрет понижать последнего админа (`ensureNotLastAdmin` вызывать и из `ChangeUserRole`)
- [x] тесты: список для admin — 200, для member — 403; понижение единственного админа — 409 `LAST_ADMIN` (заодно `DELETE` с `ErrLastAdmin` переведён с 400 на 409 — контракт уже описывал 409)
- [x] `openapi.yaml` обновлён (правок не потребовалось: план 01 уже описал все три роута); `make fmt && make test && make lint` — зелёные

### Task 5: Бэкапы через API

**Files:**
- Create: `internal/application/handlers/backups.go`, `backups_test.go`,
  `tests/integration/backups_test.go`
- Modify: `internal/services/backup_service.go`, `backup_service_test.go`, `internal/config.go`,
  `internal/run.go`, `internal/application/http_server.go`, `docs/api/openapi.yaml`,
  `docker/docker-compose.yml` (файлы `deploy/*.yml` не трогать — их заменяет план 05)

- [x] `BACKUP_DIR` в конфиге (по умолчанию `<dir(DATABASE_PATH)>/backups`), передаётся в `NewBackupService`; dev-compose задаёт `/backups`
- [x] `POST|GET /api/v1/backups`, `GET /api/v1/backups/:name/download` (`c.Attachment`), `DELETE /api/v1/backups/:name` — группа `adminOnly`; имя проверяет сервис (`ErrInvalidBackupFilename` из `GetBackup`/`DeleteBackup` → `400 INVALID_BACKUP_NAME`)
- [x] тесты обработчиков: создание → список содержит файл → скачивание отдаёт `application/octet-stream` → удаление; имя не по шаблону `backup_*.db` → 400; несуществующее → 404; member → 403
- [x] интеграционный тест на временном каталоге (`t.TempDir()`), веб-тесты бэкапов адаптированы под `BACKUP_DIR`
- [x] `openapi.yaml` обновлён; `make fmt && make test && make lint` — зелёные

### Task 6: Массовое удаление и пагинация с `total`

**Files:**
- Modify: `internal/application/handlers/transactions.go`, `transactions_test.go`, `budgets.go`,
  `budgets_test.go`, `reports.go`, `types.go`, `helpers.go`, `internal/services/transaction_service.go`,
  `internal/services/interfaces.go`, `tests/integration/transactions_test.go`, `docs/api/openapi.yaml`

- [ ] `TransactionService.BulkDelete(ctx, ids)` — одна транзакция БД, возвращает число удалённых; веб-`bulk-delete` использует его
- [ ] `POST /api/v1/transactions/bulk-delete`
- [ ] `PaginationMeta{limit, offset, total}` в `ResponseMeta`; хелпер `parsePagination(c)` с константами `defaultLimit=50`, `maxLimit=200`
- [ ] транзакции: `total` из `CountTransactions`; бюджеты и отчёты: `limit/offset` в репозиторные фильтры (сейчас бюджеты выбираются страницами по 100 внутри обработчика — убрать); users, categories, sessions, backups — `total` по длине выборки
- [ ] тесты: `total` совпадает с числом созданных, `limit=500` → 400 `INVALID_QUERY_PARAM` (становится 422 в задаче 7), `offset` за пределом → пустой `data` и верный `total`; bulk-delete: часть id не существует → удалены существующие
- [ ] `openapi.yaml` обновлён

### Task 7: Единый envelope ошибок

**Files:**
- Modify: `internal/application/handlers/helpers.go`, `errors.go`, `types.go`, `users.go`,
  `categories.go`, `transactions.go`, `budgets.go`, `reports.go`, все `*_test.go` рядом,
  `internal/services/dto/api_mappers.go`, `api_mappers_test.go`, `tests/integration/*_test.go`

- [ ] `respondError` принимает `details []ErrorDetail`; `respondValidationErrors` → `422 VALIDATION_ERROR` с details; `APIResponse.Errors` удалить
- [ ] `users.go`, `categories.go`: заменить inline-`ResponseMeta` на `respondAPI`/`respondError`; убрать двойную валидацию в `categories.go:48-96`
- [ ] удалить неиспользуемое из `dto/api_mappers.go` (оставить только то, на что есть вызовы) и `types.go` (`UpdateFamilyRequest` — если не понадобился в Task 4)
- [ ] обновить ожидания во всех тестах обработчиков и интеграционных тестах (статус 400 → 422 для валидации, форма `error.details`)
- [ ] `make test && make lint` — зелёные

### Task 8: Verify acceptance criteria

- [ ] каждая строка таблицы «Полнота API» из spec 005 закрыта роутом или помечена «удалено»
- [ ] тест покрытия `openapi.yaml` зелёный; веб-интерфейс по-прежнему работает (`make run-local`, вручную пройти дашборд, отчёты, бэкапы)
- [ ] `make pre-commit` зелёный, покрытие не ниже текущего (`make test-coverage`)

### Task 9: [Final] Update documentation

- [ ] `README.md`: раздел «API Readiness» — убрать «Experimental»/501, перечислить новые группы
- [ ] `CLAUDE.md`: `StatsService`, `GenerateReport`, `BACKUP_DIR`, единый envelope, пагинация
- [ ] `docs/backlog.md`: закрыть пункт про `POST /api/v1/reports` → 501
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

**Manual verification:** сравнить JSON `GET /stats/summary` с цифрами веб-дашборда за тот же
месяц — суммы и доли категорий должны совпадать до копейки.
