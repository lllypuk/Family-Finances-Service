# План 04 — Схема БД, деньги и даты

Четвёртый план перехода на API-only ([spec 005](../specs/005-api-only-redesign.md), решения
A-03…A-07). Продовых данных нет, поэтому `001_consolidated` переписывается целиком, без
инкрементальных миграций. После этого плана код совпадает с `docs/api/openapi.yaml`, и можно
начинать Android-приложение.

## Overview

- Деньги — `int64` в минимальных единицах (`amount_minor`) в домене, БД и JSON (A-05).
- Дата операции — календарная `YYYY-MM-DD`; у семьи появляется `timezone` (A-06).
- Клиентские `id` на создании — идемпотентный `POST` (A-07).
- Удаляются инвайты, роль `child`, `budget_alerts` и всё, что их использует (A-03, A-04, A-12).
- Смена валюты семьи блокируется при наличии операций.

## Context (from discovery)

- `float64`: `transaction.Transaction.Amount`, `budget.Budget.Amount/Spent`, `report.Data.*`,
  DTO в `internal/services/dto/*`, JSON-типы в `internal/application/handlers/types.go`;
  SQL `SUM`/`AVG`/`ORDER BY amount` в `internal/infrastructure/transaction/transaction_repository_sqlite.go:628-670`,
  `internal/infrastructure/report/report_repository_sqlite.go:357-431`; деление в
  `internal/services/report_service.go:267-272,545-595` и `dto/budget_dto.go:229`.
- `Transaction.Date` — `time.Time` (`internal/domain/transaction/transaction.go:10`), колонка `DATE`.
- `Family` без часового пояса (`internal/domain/user/user.go:38`); `UpdateFamily` меняет валюту
  без проверок (`internal/services/family_service.go:107-134`).
- Инвайты: `internal/domain/user/invite.go`, `internal/services/invite_service.go` (+2 теста),
  `internal/infrastructure/user/invite_repository_sqlite.go`, таблица `invites`.
- Роли: `internal/domain/user/user.go:34` (`RoleChild`), CHECK в миграции, `oneof=admin member child`
  в `dto/user_dto.go:76` и `handlers/types.go:45`; `exhaustive` включён для switch и map.
- `budget_alerts` **не** заглушка на уровне репозитория: `GetAlerts` и `CreateAlert`
  (`internal/infrastructure/budget/budget_repository_sqlite.go:617,673`) и тип `Alert` (`:39`)
  читают и пишут таблицу; заглушки — только веб-обработчики.
- Триггеры `updated_at` — `migrations/001_consolidated.up.sql:198-227`.
- `CleanTables` (`internal/testhelpers/sqlite.go:76-85`) уже сейчас не содержит `invites`.
- Соединение к SQLite одно (`internal/infrastructure/sqlite.go:46`) — check-then-insert
  внутри одного запроса не гоняется с другим.

## Development Approach

- **testing approach**: Regular.
- Порядок: миграция → удаления → типы → репозитории → сервисы → обработчики. Сборка красная
  только внутри задачи; в конце каждой — `make fmt && make test && make lint` зелёные.
- Правило округления живёт в одном пакете `internal/domain/money`; никаких `math.Round` по месту.
- **Миграция переписывается на месте.** golang-migrate хранит только номер версии: на существующей
  БД версия 1 уже применена, и `Up()` вернёт `ErrNoChange`, а `CREATE TABLE IF NOT EXISTS` ничего
  не изменит. Тестовый путь (`testhelpers`) исполняет файл на пустой БД и этого не покажет.
  `make db-reset` и правило в `migrations/README.md` появились в плане 03; здесь — тот же путь:
  после задачи 1 локальные и серверные БД пересоздаются.

## Testing Strategy

- unit: `internal/domain/money`, `internal/domain/date`, репозитории на in-memory SQLite,
  сервисы с моками.
- integration: `tests/integration/transactions_test.go` и `budgets_test.go` — JSON с
  `amount_minor`, даты `YYYY-MM-DD`, повтор `POST` с тем же `id`.
- Тест покрытия `openapi.yaml` в обе стороны (задача 7): после плана спецификация и код
  совпадают без исключений.

## Progress Tracking

- `[x]` по завершении; ➕ новые задачи; ⚠️ блокеры.

## Solution Overview

`money.Minor` — `type Minor int64` с `Percent(total) float64`, `DivRound(n int64) Minor`
(half-up), `Abs`. Сложение и вычитание — обычные операторы. Проценты и утилизация остаются
`float64` и считаются через `Percent`, чтобы не получить целочисленное деление. `AVG` в SQL
заменяется на `SUM`/`COUNT` в Go с `DivRound` — иначе SQLite вернёт дробь, которую каждый вызов
округлял бы по-своему.

Дата хранится как `TEXT 'YYYY-MM-DD'`; в домене — `date.Date` из своего пакета
`internal/domain/date` (`struct{ Year, Month, Day }`, `Parse`/`String`/`In(loc) time.Time`),
чтобы `budget` и `report` не импортировали `transaction`. Границы периодов («текущий месяц»)
считает сервис по `family.Timezone`.

## Technical Details

Целевая схема (`001_consolidated.up.sql`):

| Таблица | Изменение |
|---|---|
| `families` | `+ timezone TEXT NOT NULL`, `singleton` (из плана 03) |
| `users` | `role CHECK (role IN ('admin','member'))`, `is_active` без изменений |
| `categories` | без изменений |
| `transactions` | `amount REAL` → `amount_minor INTEGER NOT NULL CHECK (amount_minor > 0)`, `date TEXT NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')` |
| `budgets` | `amount_minor`, `spent_minor INTEGER NOT NULL DEFAULT 0`; `start_date`/`end_date` → `TEXT` дата |
| `reports` | `start_date`/`end_date` → `TEXT` дата; `data` JSON с `*_minor` |
| `sessions` | из плана 03 |
| удалить | `user_sessions`, `budget_alerts`, `invites` и их индексы |

JSON: `amount_minor` (int64), `date` (`"2026-09-04"`), `currency` и `timezone` в `Family`.
Фильтры транзакций: `amount_from_minor`, `amount_to_minor`, `date_from`, `date_to` (даты).
`POST` с `id`: если запись с таким `id` есть — `200` и она; иначе `201`. Безопасно без блокировки:
одно соединение к SQLite сериализует запросы.

## Implementation Steps

### Task 1: Переписать `001_consolidated`, `db-reset`, репозиторий бюджетов без алертов

**Files:**
- Modify: `migrations/001_consolidated.up.sql`, `001_consolidated.down.sql`, `migrations/README.md`,
  `Makefile`, `internal/testhelpers/sqlite.go`,
  `internal/infrastructure/budget/budget_repository_sqlite.go`, `budget_repository_test.go`,
  `internal/domain/budget/budget.go`, `internal/services/interfaces.go`

- [ ] переписать `.up.sql` по таблице выше одним файлом; `.down.sql` — `DROP` в обратном порядке; триггеры `updated_at` сохранить
- [ ] `migrations/README.md`: описание новой схемы; напоминание про `make db-reset`
- [ ] `CleanTables` — актуальный список (в том числе убрать `budget_alerts`, `user_sessions`; `invites` там и не было)
- [ ] удалить `budget.Alert`, `GetAlerts`/`CreateAlert`/`Alert` из репозитория бюджетов и из `BudgetRepository` в `interfaces.go`; их тесты — тоже
- [ ] тесты: обе ветки миграций поднимаются на пустой БД; вставка второй семьи и роли `child` падают на UNIQUE/CHECK (тест репозитория)
- [ ] `make fmt && make test && make lint` — зелёные (сборка ломается на `child` и инвайтах — их чинят задачи 2–3; до этого держать старые CHECK на роли и таблицу `invites` в миграции и убрать их в задаче 3)

### Task 2: Удалить инвайты

**Files:**
- Delete: `internal/domain/user/invite.go`, `internal/services/invite_service.go`,
  `invite_service_test.go`, `invite_security_test.go`,
  `internal/infrastructure/user/invite_repository_sqlite.go` (+ тест)
- Modify: `internal/infrastructure/repositories_sqlite.go`, `internal/services/container.go`,
  `internal/services/interfaces.go`, `internal/run.go`, `internal/application/handlers/repositories.go`,
  `migrations/001_consolidated.up.sql`, `001_consolidated.down.sql`

- [ ] удалить файлы, поля `Invite` из `Repositories`/`Services`, таблицу `invites` из миграции
- [ ] `grep -rn -i invite internal cmd tests` — пусто
- [ ] `make fmt && make test && make lint` — зелёные

### Task 3: Убрать роль `child`

**Files:**
- Modify: `internal/domain/user/user.go`, `user_test.go`, `internal/services/dto/user_dto.go`,
  `internal/application/handlers/types.go`, все `switch`/`map` по `user.Role` (найти
  `grep -rn 'RoleChild\|"child"' internal tests`), `migrations/001_consolidated.up.sql`

- [ ] удалить `RoleChild`; `oneof=admin member` в DTO и `types.go`; `exhaustive` подскажет остальные места
- [ ] CHECK `role IN ('admin','member')` в миграции
- [ ] тесты: `POST /users` с `role: child` → 422; тест домена на `Role.IsValid`
- [ ] `make fmt && make test && make lint` — зелёные

### Task 4: Пакеты `money` и `date`

**Files:**
- Create: `internal/domain/money/money.go`, `money_test.go`, `internal/domain/date/date.go`, `date_test.go`

- [ ] `money.Minor` с `Abs`, `Percent(total Minor) float64` (0 при `total == 0`), `DivRound(n int64) Minor` half-up, `MarshalJSON` как число
- [ ] `date.Date`: `Parse("2006-01-02")`, `String()`, `In(*time.Location) time.Time`, `Before/After`, `MonthBounds(loc)`; `Scan`/`Value` для `database/sql`; JSON — строка
- [ ] тесты: `DivRound` на отрицательных и на половине, `Percent` с нулём, невалидные даты, `MonthBounds` на границах года
- [ ] `make fmt && make test && make lint` — зелёные

### Task 5: Домен и репозитории

**Files:**
- Modify: `internal/domain/transaction/transaction.go`, `internal/domain/budget/budget.go`,
  `internal/domain/report/report.go`, `internal/domain/user/user.go` (`Family.Timezone`),
  `internal/infrastructure/transaction/transaction_repository_sqlite.go`,
  `internal/infrastructure/budget/budget_repository_sqlite.go`,
  `internal/infrastructure/report/report_repository_sqlite.go`,
  `internal/infrastructure/user/family_repository_sqlite.go`, все `*_test.go` рядом,
  `internal/testhelpers/factories.go`

- [ ] `Transaction.AmountMinor money.Minor`, `Date date.Date`; `Budget.AmountMinor/SpentMinor`, `StartDate/EndDate date.Date`; методы бюджета (`GetRemainingAmount`, `GetSpentPercentage`, `IsOverBudget`) через `money`
- [ ] `report.Data`: суммы `money.Minor`, проценты `float64`; `Report.StartDate/EndDate` — даты
- [ ] репозитории: колонки `*_minor`, `SUM` как `int64`, `AVG` убрать (считать в Go), `ORDER BY amount_minor`, фильтры по датам как строки `TEXT`
- [ ] `Family.Timezone` в домене и репозитории; `factories.go` — `Europe/Moscow`, `RUB`
- [ ] тесты репозиториев: суммы без потери копеек (три операции по 33 копейки → 99), сортировка, фильтр по диапазону дат включает границы
- [ ] `make fmt && make test && make lint` — зелёные

### Task 6: Сервисы и отчёты

**Files:**
- Modify: `internal/services/transaction_service.go`, `budget_service.go`, `report_service.go`,
  `stats_service.go`, `family_service.go`, `internal/services/dto/*.go`, все `*_test.go` рядом

- [ ] `TransactionService`/`BudgetService`: `money.Minor`; `ValidateTransactionLimits`, `UpdateBudgetSpent`, `CalculateBudgetUtilization` через `Percent`/`DivRound`
- [ ] `ReportService`: средние (`total / days`, `total / count`) через `DivRound`; проценты через `Percent`; CSV пишет `amount_minor` целым и колонку `currency`
- [ ] `StatsService`: границы периода по `family.Timezone`, доли категорий через `Percent`
- [ ] `FamilyService.UpdateFamily`: смена `currency` при `CountTransactions > 0` → `ErrCurrencyLocked`; `timezone` валидируется `time.LoadLocation`
- [ ] тесты: проценты не нулевые при малых суммах (1 из 3 → 33.33), средние округлены half-up, `ErrCurrencyLocked`, невалидный timezone → ошибка валидации
- [ ] `make fmt && make test && make lint` — зелёные

### Task 7: Обработчики, идемпотентный `POST`, спецификация в обе стороны

**Files:**
- Modify: `internal/application/handlers/types.go`, `transactions.go`, `budgets.go`,
  `categories.go`, `reports.go`, `stats.go`, `families.go`, все `*_test.go` рядом,
  `tests/integration/transactions_test.go`, `budgets_test.go`, `families_test.go`,
  `tests/integration/openapi_coverage_test.go`, `docs/api/openapi.yaml`

- [ ] запросы/ответы: `amount_minor` (`gt=0`), `date`/`start_date`/`end_date` как `YYYY-MM-DD`, фильтры `amount_from_minor`/`amount_to_minor`; `Family{currency, timezone}`
- [ ] `id` (uuid4, optional) в `Create*Request` транзакций, бюджетов, категорий: сервис `GetByID` → есть → `200`, нет → `201`
- [ ] `openapi.yaml` привести к коду; в тест покрытия добавить обратную проверку — каждая операция спецификации зарегистрирована как роут
- [ ] тесты обработчиков: `amount_minor: 0` → 422, `date: "2026-13-01"` → 422, повтор `POST` с тем же `id` → 200 и одна запись в БД
- [ ] интеграционные тесты: полный цикл создания/фильтрации/отчёта на копейках
- [ ] `make fmt && make test && make lint` — зелёные

### Task 8: Verify acceptance criteria

- [ ] `grep -rn 'float64' internal --include='*.go' | grep -v _test | grep -iv 'percent\|utiliz\|rate\|ratio'` — только объяснимые места
- [ ] тест покрытия `openapi.yaml` (обе стороны) зелёный, валидатор спецификации без ошибок
- [ ] `make db-reset && make run-local`: `setup` через CLI → login → создать транзакцию на 12 345 копеек → `GET /stats/summary` показывает 12345
- [ ] `make pre-commit` зелёный; `docker build` собирается

### Task 9: [Final] Update documentation

- [ ] `CLAUDE.md`: «Database & migrations» (список таблиц, `make db-reset`), деньги и даты в «Conventions», нет инвайтов и `child`
- [ ] `README.md`, `docs/product_brief.md`, `docs/tech_stack.md`: инвайты → «админ создаёт пользователя», роли `admin|member`
- [ ] `docs/patterns/api_standards.md`: сверить примеры с реальными ответами
- [ ] `migrations/README.md`: описание новой схемы
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

**External:** Android-приложение стартует после этого плана — контракт денег, дат и `id`
зафиксирован. Если понадобится офлайн-синхронизация (A-10), это отдельный spec с `revision` и
tombstones **до** первых реальных данных.
