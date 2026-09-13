# План 10 — Снос `/reports`, статистика на SQL, `GET /stats/monthly`

Отчёты в мобильном приложении — это экран поверх статистики, а не CSV-файл (решение владельца
13.09.2026; строка «CSV-экспорт: оставить» в `docs/specs/005-api-only-redesign.md:190` отменена).
Серверный релиз `v0.3.0`; клиент (период на главной, экран «Обзор», мультивыбор операций) — план 11.
Ревью: codex (тред `reports-replacement`, два раунда) и plan-review, 13.09.2026.

## Overview

- `/api/v1/reports*` (5 роутов), `ReportService`, репозиторий, домен `report`, таблица `reports` — удаляются.
  Миграция `003_drop_reports`. На проде 0 строк в `reports` (проверено 13.09.2026), терять нечего.
- `StatsService.Summary` считает итоги и доли категорий SQL-агрегатами, а не загрузкой всех операций в память:
  сейчас выборка молча обрывается на 20 000 операций (`internal/services/stats_service.go:26,140`), и
  сводка за год отдаст заниженные суммы без ошибки.
- Контракт `StatsSummary` не меняется: дельта при нулевой базе своей метрики остаётся `0` (клиент 0.4.0
  распарсит только `Double`), а скрывать её по `previous.income_minor == 0` будет клиент в плане 11 — сейчас он
  смотрит только на общий `has_previous_data` (`android/.../ui/home/HomeScreen.kt:144`).
- Новый `GET /api/v1/stats/monthly?from&to` — ряд по месяцам для экрана «Обзор».
- Сервис в `BudgetHandler`/`TransactionHandler` становится обязательным параметром, мёртвые ветки
  `if h.xxxService != nil` (13 штук) уходят вместе с юнит-тестами на репозиторный путь.

## Context (from discovery)

- Роуты: `internal/application/http_server.go:221–225` (`reports`), `:228` (`stats`); хендлеры собираются в
  `:142–145` — сервис уже передаётся всегда.
- `/reports` держат: `handlers/reports.go`, `handlers/repositories.go:17,43` (`Report` в `Repositories`, алиас),
  `handlers/helpers.go:338` (`ErrReportNotFound` в `isNotFoundError`), `services/report_service.go` (1269 строк),
  `services/report_csv.go`, `services/dto/report_dto.go`, `services/interfaces.go:136` (`ReportService`),
  `services/container.go:14,30,44,61,79` (`ReportRepository`, `Services.Report`, параметр `NewServices`),
  `internal/run.go:97`, `internal/infrastructure/report/`, `internal/infrastructure/validation/validation.go`
  (`ValidateReportType/Period/Name`, `maxReportNameLength`), `internal/domain/report/`,
  `testhelpers/integration_server.go:75,93`, `testhelpers/factories.go:23–24,106–118`,
  `testhelpers/sqlite.go:80` (`CleanTables`). Тесты: `handlers/{reports,report_service_mock}_test.go`,
  `application/http_server_test.go:392` (`MockReportService`), `domain/report/report_test.go`,
  `services/{report_service,report_csv,helpers}_test.go`, `services/dto/report_dto_test.go`,
  `infrastructure/report/report_repository_test.go`, `validation/validation_test.go`,
  `tests/integration/{reports,transactions,api_pagination,api_roles,api_auth}_test.go`.
- Схема: таблица `migrations/001_consolidated.up.sql:114–132`, индексы `idx_reports_family_type`,
  `idx_reports_generated_by` — `:172–173`; `001_consolidated.down.sql:11–12,33`.
  `migrations_test.go:30` (список таблиц), `:47` (`reports.start_date`), `:125` (`Migrate(1)` после `Up()` —
  released-схема восстанавливается down-миграциями, а не повторным `001.up`). Прод на версии 2.
- Спека: `docs/api/openapi.yaml:716–813` (три пути `reports`), схемы `ReportOk` (`:1049`, `:1168` — один из шести
  envelope-ов, см. `CLAUDE.md`, раздел Android), `ReportType/ReportPeriod` (`:1219–1230`),
  `ReportSummary/Report/ReportData` (`:1505–1596`); `getStatsSummary` `:814`, `StatsSummary` `:1648`,
  `PeriodTotals` `:1597`. Клиент: `android/core/api/generated/kotlin/.../core/api/ReportsApi.kt`, модели `Report*`,
  `CreateReportRequest.kt`, `ListReports200Response.kt` исчезнут при регенерации; приложение их не вызывает.
  Единственные HTTP-тесты `/stats/summary` живут в `tests/integration/reports_test.go:561,593`.
- Статистика: `statsService` (`stats_service.go:34`) зависит от сервисов, не репозиториев; `Summary` (`:58`)
  → `transactionsBetween` (`:121`, обрыв на `:141`) → `periodTotals` (`:303`), `categoryShares` (`:232`)
  в памяти; `previousTotals` (`:148`) — ошибка выборки читается как «данных нет»; `previousPeriod` (`:324`) —
  соседний интервал той же длины (соответствует `openapi.yaml:1658`); `periodDeltas` (`:330`) при нулевой базе
  возвращает 0. Бюджеты (`:82`) и `recent` (`:87`) считаются на `today`, а не на период — так и остаётся,
  контракт это описывает. Хендлер: `handlers/stats.go:24`, `parseStatsPeriod` (`:45`) переиспользуется.
- Интерфейс `TransactionRepository` — `internal/services/transaction_service.go:48` (не `interfaces.go`), соседи
  `GetTotalByCategory/ByDateRange/ByCategoryAndDateRange` (`:58–68`). Репозиторий уже умеет `SUM` по периоду
  (`transaction_repository_sqlite.go:719`), `GROUP BY` — только в бюджетах (`budget_repository_sqlite.go:305`).
  В `date.go` есть `AddDays` (`:77`) и `MonthBounds` (`:90`), арифметики по месяцам нет.
  `date` — TEXT `YYYY-MM-DD` без зоны (`001_consolidated.up.sql:73,85`), поэтому `substr(date, 1, 7)` даёт месяц
  без пересчёта пояса; пояс семьи нужен только для границ по умолчанию.
- Мёртвые ветки: `handlers/transactions.go:74,100,262,491,528,619,643`, `handlers/budgets.go:56,84,101,129,178,234`;
  конструкторы с вариативным сервисом — `budgets.go:26`, `transactions.go:32`. Юнит-тесты на репозиторный путь:
  `budgets_test.go:88` (`setupBudgetHandler`), `transactions_test.go:130` (`setupTransactionHandler`);
  сервисный путь уже тестируется через `stubBudgetService` (`budgets_test.go:239,268,292`). `MockTransactionRepository`
  (`transactions_test.go`) встраивает `handlers.TransactionRepository` и нужен `setupBudgetHandler` (`budgets_test.go:88,90`);
  его `GetTotalsByCategory` (`:117`) — мёртвый метод, единственное вхождение в дереве.

## Development Approach

- Regular: код, затем тесты; `make fmt && make test && make lint` (0 issues) после каждой задачи.
- Задачи 1–3 — чистка, 4–6 — статистика; они независимы, каждая — свой коммит. Спека и роуты меняются в одной
  задаче (двусторонний тест `tests/integration/openapi_coverage_test.go`), `make -C android api-gen` — в той же
  задаче, что правит спеку, и результат коммитится (CI `android:api-check`).
- Схема: правка `001` + нумерованная `003` (`migrations/README.md`), обе с покрытием в `migrations_test.go`.

## Solution Overview

- **Хендлеры.** `NewBudgetHandler(repos, budgetService)`, `NewTransactionHandler(repos, transactionService)`;
  методы `*ViaService` становятся телом публичных методов; `UpdateEntityHelper`/`DeleteEntityHelper` и
  поля-функции удаляются, если после сноса на них никто не ссылается. Юнит-тесты репозиторного пути
  переписываются на стаб сервиса; сценарии, уже покрытые интеграционно, не дублируются.
- **`reports` в схеме.** Из `001.up` таблица и индексы уходят; в `001.down` их `DROP … IF EXISTS` остаются
  с комментарием: при полном откате `003.down` таблицу восстановит, и убирать её должен `001.down`.
  `003.up` — `DROP INDEX IF EXISTS` ×2, `DROP TABLE IF EXISTS reports` (на свежей базе — no-op);
  `003.down` — таблица и индексы в точности из `001` версии 2.
- **Агрегаты.** Репозиторий транзакций получает два метода поверх одного `WHERE family_id = ? AND date >= ? AND
  date <= ?`: `TotalsByCategory` (`GROUP BY category_id, type` → `category_id, type, SUM(amount_minor), COUNT(*)`)
  и `TotalsByMonth` (`GROUP BY substr(date,1,7), type`). Типы результата — в `internal/domain/transaction`
  (`CategoryTotal`, `MonthTotal`), деньги `money.Minor`. `TransactionService` пробрасывает их без логики;
  `statsService` строит `PeriodTotals`, `CategoryShare` и месячный ряд из этих строк. `transactionsBetween`,
  `periodTotals`, `statsTransactionLimit`, `statsMaxTransactions` удаляются. Ошибка `previousTotals` больше не
  маскируется под «данных нет» — сводка отвечает 500, как любой другой сбой БД.
- **`GET /stats/monthly`.** Параметры `from`/`to` как у `summary`; по умолчанию — 12 календарных месяцев по
  сегодняшний в поясе семьи (`from` = первое число месяца одиннадцать месяцев назад, `to` = сегодня).
  Ответ `StatsMonthly {from, to, months: [{month: "YYYY-MM", income_minor, expenses_minor, net_minor,
  transaction_count}]}` — по одной корзине на каждый месяц, попадающий в `[from, to]`, нули для пустых, крайние
  месяцы не расширяются до полных (корзина покрывает только дни внутри интервала — это описано в спеке).
  `from > to` → `422`, как у `summary`. `financeAccess`.

## Implementation Steps

### Task 1: Сервис обязателен в `BudgetHandler` и `TransactionHandler`

**Files:**
- Modify: `internal/application/handlers/budgets.go`, `internal/application/handlers/transactions.go`
- Modify: `internal/application/handlers/helpers.go` (если `UpdateEntityHelper`/`DeleteEntityHelper` осиротеют)
- Modify: `internal/application/http_server.go:142–143`
- Modify: `internal/application/handlers/budgets_test.go`, `internal/application/handlers/transactions_test.go`,
  `internal/application/http_server_test.go`

- [x] `NewBudgetHandler(repos, budgetService services.BudgetService)`, `NewTransactionHandler(repos, transactionService
      services.TransactionService)`; `nil` сервис — паника в конструкторе с понятным текстом
- [x] снести 12 ветвлений и репозиторные реализации; `*ViaService` влить в публичные методы; удалить осиротевшие
      `UpdateEntityParams`/`UpdateEntityHelper`, `updateTransactionFields` и т.п. `DeleteEntityHelper` и
      `ParseIDParamWithError` остаются — их зовут сервисные пути (`transactions.go:550,620`, `budgets.go:235`)
- [x] `h.repositories.Family` остаётся: `getBudgetsViaService` берёт `familyToday` для `active_only=true`
      (`budgets.go:272`, `helpers.go:88` — `nil` молча даёт UTC); остальные поля `Repositories` в этих хендлерах
      больше не читаются. Extra-метод `GetAll` в `handlers.TransactionRepository` (`repositories.go:28–36`) после
      сноса никто не зовёт — алиас становится обычным `= services.TransactionRepository`
- [x] `setupBudgetHandler`/`setupTransactionHandler` — на стаб/мок сервиса (+ мок `FamilyRepository` для
      `active_only`); `MockTransactionRepository` остаётся (его ждёт `setupBudgetHandler`), мёртвый
      `GetTotalsByCategory` (`transactions_test.go:117`) — удалить, чтобы не путался с новым `TotalsByCategory`
- [x] тесты сервисного пути: успех, 404 (`isNotFoundError`), 409/422 через `handle*ServiceError` — по одному
      случаю на метод, без дублирования интеграционных
- [x] `make fmt && make test && make lint` — 0 issues; коммит `refactor: сервис обязателен в хендлерах бюджетов и транзакций`

### Task 2: Удалить `/reports` из кода и контракта

**Files:**
- Delete: `internal/application/handlers/reports.go`, `internal/application/handlers/{reports,report_service_mock}_test.go`,
  `internal/services/report_service.go`, `internal/services/report_csv.go`, `internal/services/{report_service,report_csv}_test.go`,
  `internal/services/dto/report_dto.go`, `internal/services/dto/report_dto_test.go`, `internal/infrastructure/report/`,
  `internal/domain/report/`, `tests/integration/reports_test.go`
- Modify: `internal/application/http_server.go`, `internal/application/http_server_test.go`,
  `internal/application/handlers/repositories.go`, `internal/application/handlers/helpers.go`,
  `internal/services/interfaces.go`, `internal/services/container.go`, `internal/services/helpers_test.go`,
  `internal/run.go`, `internal/infrastructure/repositories_sqlite.go:10,25`,
  `internal/infrastructure/validation/validation.go` (+ `_test`),
  `internal/testhelpers/integration_server.go`, `internal/testhelpers/factories.go`,
  `tests/integration/{transactions,api_pagination,api_roles,api_auth}_test.go`
- Create: `tests/integration/stats_test.go` (перенос `TestStatsAPI_Summary*` из `reports_test.go:561,593`)
- Modify: `internal/application/handlers/errors.go:52–56`, `docs/api/openapi.yaml`, `android/core/api/generated/**`

- [x] снести роуты, хендлер, сервис, репозиторий, домен, DTO, `ReportRepository`/`Services.Report`, параметр
      `NewServices`, `ErrReportNotFound` из `isNotFoundError`, валидаторы отчётов, фабрику `CreateTestReport`,
      коды `ErrCodeGenerationFailed`/`ErrCodeSaveFailed`/`ErrCodeExportFailed` (`errors.go:52–56`, живут только в
      `reports.go:84,88,112`)
- [x] `TestStatsAPI_Summary`, `TestStatsAPI_Summary_InvalidDate` → `tests/integration/stats_test.go` до удаления
      `reports_test.go`
- [x] интеграционные тесты: `api_pagination_test.go:115,276`, `api_auth_test.go:193,337`, `api_roles_test.go:154`,
      `transactions_test.go:690` — механически на другой ресурс; `TestAPIPagination_Reports_TotalBeyondRepositoryLimit`
      (`api_pagination_test.go:333`, потолок репозитория в 100 строк) — перенести на `/api/v1/transactions`
- [x] `openapi.yaml`: убрать три пути, `ReportOk` из `responses` и `schemas`, `ReportType/ReportPeriod/ReportSummary/
      Report/ReportData`, тег `reports`; коды `GENERATION_FAILED`/`SAVE_FAILED`/`EXPORT_FAILED` из шапки, если есть
- [x] `make -C android api-gen` — `ReportsApi.kt` и модели `Report*` исчезают; закоммитить; `make -C android api-check`,
      `make -C android check`
- [x] `make fmt && make test && make lint` — 0 issues (в т.ч. `TestOpenAPISpec_*`); коммит `refactor: снос /reports`

### Task 3: Миграция `003_drop_reports`

**Files:**
- Modify: `migrations/001_consolidated.up.sql`, `migrations/001_consolidated.down.sql`
- Create: `migrations/003_drop_reports.up.sql`, `migrations/003_drop_reports.down.sql`
- Modify: `internal/infrastructure/migrations_test.go`, `internal/testhelpers/sqlite.go`
- Create: `cmd/server/migrate.go`; Modify: `cmd/server/main.go`, `internal/bootstrap.go` (+ `_test`), `deploy/README.md`

- [x] `001.up`: убрать таблицу и два индекса; `001.down`: оставить их `DROP … IF EXISTS` с комментарием про `003.down`
- [x] `003.up`: `DROP INDEX IF EXISTS idx_reports_generated_by; DROP INDEX IF EXISTS idx_reports_family_type;
      DROP TABLE IF EXISTS reports;` `003.down`: `CREATE TABLE reports` + индексы дословно из `001` версии 2
- [x] `CleanTables`: убрать `reports`; `migrations_test.go`: убрать из списка таблиц и проверку `reports.start_date`,
      добавить `assert` отсутствия таблицы после `Up()`
- [x] тест `TestMigrations_DropReports`: `Up()`, `Migrate(2)` → таблица есть, `Up()` → нет, `Down()` → таблиц нет
- [x] подкоманда `migrate --to N` (`NewMigrationManager` берёт URL, `migrations.go:21` — `OpenDatabaseNoMigrate` не
      нужен; `os.Stat(DATABASE_PATH)` до запуска, как `backup.go:33`, иначе опечатка в пути создаст пустую базу
      версии N; без аргумента — печать текущей версии): старый образ не стартует на версии схемы, которой нет в его `./migrations`
      (golang-migrate: `no migration found for version 3`), так что откат образа начинается с `migrate --to 2`
      образом `v0.3.0`. Это касалось и `v0.1.0` ↔ версии 2, просто никто не откатывал. Тест в `bootstrap_test.go`;
      раздел «Откат» в `deploy/README.md`
- [x] `make test` (оба пути: golang-migrate и `testhelpers`); коммит `feat: миграция 003 — таблица reports удалена`

### Task 4: Агрегаты в репозитории транзакций

**Files:**
- Modify: `internal/domain/transaction/transaction.go` (типы `CategoryTotal`, `MonthTotal`)
- Modify: `internal/infrastructure/transaction/transaction_repository_sqlite.go` (+ `_test`)
- Modify: `internal/services/transaction_service.go:48` (`TransactionRepository`), `internal/services/container.go`,
  `internal/services/stats_service.go:34–57` (конструктор), моки: `internal/services/helpers_test.go`,
  `internal/application/handlers/transactions_test.go` (`MockTransactionRepository` встраивает
  `handlers.TransactionRepository`, `repositories.go:31`), `internal/testhelpers/integration_server.go`

- [x] имена в стиле соседей: `GetTotalsByCategoryAndDateRange`, `GetTotalsByMonth` (ниже — коротко).
      `TotalsByCategory(ctx, from, to) ([]transaction.CategoryTotal, error)`: `category_id, type, SUM(amount_minor),
      COUNT(*)`, `GROUP BY category_id, type`, `family_id` из `getSingleFamilyID`
- [x] `TotalsByMonth(ctx, from, to) ([]transaction.MonthTotal, error)`: `substr(date, 1, 7) AS month, type, SUM, COUNT`,
      `GROUP BY month, type ORDER BY month`
- [x] методы — в `services.TransactionRepository` (там живёт SQL), а `statsService` получает их через узкий
      интерфейс `statsAggregates { TotalsByCategory; TotalsByMonth }` отдельным параметром `NewStatsService`;
      контейнер передаёт `transactionRepo`. `TransactionService` не растёт; комментарий «только сервисы» у
      `statsService` (`stats_service.go:33`) переписать. Моки репозитория — по два пустых метода
- [x] тесты репозитория: суммы по типу и категории, границы включительно, пустой период → пустой срез, месяц
      на стыке (`2026-08-31`/`2026-09-01` — разные корзины)
- [x] `make fmt && make test && make lint`; коммит `feat: агрегаты транзакций по категориям и месяцам`

### Task 5: `Summary` на агрегатах

**Files:**
- Modify: `internal/services/stats_service.go`, `internal/services/stats_service_test.go`,
  `tests/integration/stats_test.go` (создан в Task 2)

- [x] `Summary`: `PeriodTotals` текущего и предыдущего периода и `CategoryShare` — из `TotalsByCategory`
      (`categoryName` как сейчас); `transactionsBetween`, `periodTotals`, оба лимита — удалить; ошибка предыдущего
      периода → ошибка `Summary`. `CountTransactions` (`:93`) — фильтр `dto.NewTransactionFilterDTO()`, не нулевой
      (нулевой `Limit` не проходит валидацию, и каждая сводка станет ошибкой)
- [x] тесты сервиса: суммы сходятся с прежними случаями (`stats_service_test.go`); доля категорий; ошибка
      репозитория → ошибка `Summary`. Интеграционно (`tests/integration/stats_test.go`, in-memory SQLite): 20 001
      операция одной вставкой в транзакции → `current.transaction_count == 20001` и сумма сходится — мок полноту
      выборки не проверяет
- [x] `make fmt && make test && make lint`; коммит `fix: сводка считается в SQL, без потолка в 20 000 операций`

### Task 6: `GET /api/v1/stats/monthly`

**Files:**
- Modify: `internal/domain/date/date.go` (+ `_test`): `MonthKey() string` (`YYYY-MM`), `AddMonths(n)` или
  `MonthStart` со сдвигом — арифметики по месяцам сейчас нет
- Modify: `internal/services/dto/stats_dto.go` (`StatsMonthly`, `MonthTotals`), `internal/services/interfaces.go`
  (`StatsService.Monthly`), `internal/services/stats_service.go` (+ `_test`)
- Modify: `internal/application/handlers/stats.go` (+ `_test`), `internal/application/http_server.go`
- Modify: `docs/api/openapi.yaml`, `android/core/api/generated/**`, `tests/integration/stats_test.go`

- [x] `StatsService.Monthly(ctx, from, to *date.Date) (*dto.StatsMonthly, error)`: границы по умолчанию — 12 месяцев
      по сегодняшний в поясе семьи; корзины на каждый месяц `[from, to]`, нули для пустых; суммы из `TotalsByMonth`
- [x] `StatsHandler.GetMonthly` через `parseStatsPeriod`; `from > to` → `422` тем же `ErrorDetail`, что у `GetSummary`;
      роут `stats.GET("/monthly", …)` под `financeAccess`
- [x] сборка корзин — по ключу месяца из заранее построенной последовательности `[from, to]`; `TotalsByMonth`
      отдаёт 0–2 строки на месяц (по `type`), порядок типов внутри месяца не гарантирован — раскладывать по `type`,
      `transaction_count` суммировать, пары строк не предполагать
- [x] спека: операция `getStatsMonthly` (tag `stats`, `operationId`, `4xx $ref: Error`), схемы `StatsMonthly`,
      `MonthTotals`; описать дефолт и правило крайних месяцев. Тут же — уточнение описаний `income_delta`/
      `expenses_delta` в `StatsSummary` («`0` и при нулевой базе метрики; клиент сверяет `previous.*_minor`»):
      описание попадает в Kotlin-комментарии, и `api-check` требует регенерации. `make -C android api-gen`,
      закоммитить, `make -C android api-check`, `make -C android check`
- [x] тесты сервиса: без границ и без операций → 12 нулевых корзин; явный интервал внутри одного месяца → одна
      корзина; месяц только с доходами, только с расходами, с обоими; операции в двух месяцах. Интеграционные:
      `200` с токеном `member`, `401` без, `422` при `from > to`
- [x] `make fmt && make test && make lint`; коммит `feat: GET /stats/monthly — ряд по месяцам`

### Task 7: Verify acceptance criteria
- [x] `grep -rn -i 'report' internal tests cmd migrations docs/api/openapi.yaml` — только `003_*`, `001.down`
      (намеренные `DROP`) и `migrations_test.go`; сверх списка — `migrations/{README,CHANGELOG}.md` (описание самой
      `003`), слово «reporting» в `transaction_service.go:211` и дословный лог красной фазы S-01 в
      `api_auth_test.go:39,47` (исторический вывод, не ссылка на код)
- [x] `make fmt && make test && make lint` — 0 issues; `make -C android api-check && make -C android check`
- [x] `make compose-config`; сервер на свежей базе и на копии базы версии 2 (`migrate --to 2` перед стартом) —
      `/health` 200, `schema_migrations` = 3, таблицы `reports` нет в обеих

### Task 8: [Final] Update documentation
- [ ] `CLAUDE.md`: роуты (`reports` из перечня `financeAccess`, абзац про `POST /reports`/`export`), список таблиц
      в «Database & migrations», «шесть envelope-ов» → пять, `TestOpenAPISpec_*` не трогать; `StatsService.Monthly`
- [ ] `docs/specs/005-api-only-redesign.md`: строка `:190` — решение «удалить, отчёты — экран поверх статистики»,
      таблица планов (10), абзац `:241` — следующий релиз `v0.3.0`, дальше план 11 (клиент)
- [ ] `docs/backlog.md`: снять пункт про мёртвые ветки хендлеров; вписать план 11 (клиент): экран «Обзор» с выбором
      периода на `summary`+`monthly` (главная остаётся «этот месяц» — бюджеты и `recent` в `summary` считаются на
      сегодня, смешивать их с произвольным периодом нельзя), дельта скрывается при `previous.*_minor == 0`,
      мультивыбор операций + `bulk-delete`
- [ ] `docs/patterns/api_standards.md`: коды `REPORT_*`, если были; `docs/api/README.md` — при упоминании отчётов;
      `deploy/README.md` — при упоминании
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Релиз:** тег `v0.3.0` на merge-коммите; пайплайн выкатит на mini, миграция `003` пройдёт при старте
(`schema_migrations` 2 → 3). Откат на `v0.2.0` — только через `migrate --to 2` образом `v0.3.0` до подмены
образа (см. Task 3): старый образ на версии 3 не стартует. Полный `Down()` на рабочей базе не гарантирован
(`002.down` возвращает табличный `UNIQUE`, а после мягких удалений дубликаты допустимы) — откатывать адресно.

**Клиент:** план 11 (`app-v0.5.0`) — после выката `v0.3.0`; `app-v0.4.0` к `v0.3.0` совместим: не зовёт
`/reports`, а `StatsSummary` не изменился.

**Ручная проверка:** с телефона `GET /stats/monthly` без параметров за год (в `sqlite-shell` сравнить с
`SELECT substr(date,1,7), type, SUM(amount_minor) … GROUP BY 1,2`).
