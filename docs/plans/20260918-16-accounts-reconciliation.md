# План 16. Счета и ежемесячная сверка

## Overview

В конце месяца семья сверяет по каждому платёжному счёту сумму записанных расходов с цифрой «траты за
месяц» из банка: расхождение значит, что что-то не записали. Сейчас у операции нет счёта, и сверка
делается вне приложения.

- Справочник счетов, необязательный `account_id` у операции, фильтр по счёту.
- Запись сверки: на счёт и месяц хранится цифра банка; записанное считается на чтении, поэтому
  дописанная операция сама закрывает разницу. Месяц не блокируется.
- Сервер `v0.6.0`, клиент `0.8.0`. Клиенты `0.6/0.7` продолжают работать.

Не входит: остатки, переводы между счетами, построчная сверка с выпиской, активы и пассивы (план 17).

## Context (from discovery)

- Образец слоя «справочник»: `internal/domain/category`, `internal/infrastructure/category`,
  `internal/services/category_service.go`, `handlers/categories.go`.
- Операции: `internal/domain/transaction/transaction.go` (`Transaction` :15, `Filter` :35),
  `handlers/types.go` (`UpdateTransactionRequest` :183, `TransactionFilterParams` :193, `isEmpty` :222),
  `handlers/transactions.go` (ответ :122, фильтр :305), `services/transaction_service.go` (интерфейс :48,
  update :312, фильтр :784), `infrastructure/transaction/transaction_repository_sqlite.go` (сканер :67,
  INSERT :173, `GetByID` :311, `buildFilterConditions` :341, SELECT :453 и :661, UPDATE :534).
- Сборка: `services/container.go:23`, `infrastructure/repositories_sqlite.go:15`,
  `handlers/repositories.go:8`, `services/interfaces.go`.
- Месяц: `internal/domain/date/date.go` — `MonthKey`, `MonthBounds`, `AddMonths` есть, разбора `YYYY-MM` нет.
- Валюта: `services/family_service.go:103` — `hasTransactions` смотрит только на операции.
- Миграции: `001` — картина схемы, живая база получает `NNN`; тестовый путь
  (`internal/testhelpers/sqlite.go`) выполняет все `*.up.sql` подряд. golang-migrate открывает своё
  соединение без `_foreign_keys` (`internal/bootstrap.go`, `databaseURL`).
- Клиент: `ApiClient.kt:43` — `explicitNulls = false`, `null` на провод не уходит; `ApiGraph.kt:36`,
  `TransactionEditViewModel.kt:192`, `RecognizeViewModel.kt:263`, `ui/settings`, `ui/transactions/Filters.kt`.
- Решения обсуждены с codex: тред `accounts-reconciliation-holdings`.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Сервер (1–6), затем клиент (7–10). Каждая задача оставляет репозиторий зелёным: правка спеки идёт
  в одном коммите с маршрутом **и** с `make -C android api-gen`.
- Серверная задача — `make fmt`, `make test`, `make lint` (0 issues); клиентская — `make -C android check`.
- **CRITICAL: update this plan file when scope changes during implementation**
- Обратная совместимость: все новые поля запроса необязательны, `account_id` в ответе nullable.

## Testing Strategy

- Unit: домен (`ParseMonth`, нормализация имени), сервисы на моках, handler'ы через `principalContext`.
- Репозитории и миграции — in-memory SQLite (`testhelpers.SetupSQLiteTestDB`).
- Интеграция — `testhelpers.SetupHTTPServer`: роли, коды, идемпотентность, покрытие спеки.
- Клиент — Robolectric-тесты ViewModel'ей, как в `ui/recognize`. E2E в проекте нет.

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

**`recorded` не хранится.** Хранится только цифра банка; сумма записанного — `SUM` на чтении. Сохранённая
копия врала бы после первой же правки операции.

**Только расходы.** `recorded` = `SUM(amount_minor) WHERE type = 'expense'`. Возврат, записанный доходом,
сумму не уменьшает: сверяется цифра «траты за месяц», а не оборот.

**Отвязка — отдельным флагом.** В `PUT /transactions/:id` отсутствие `account_id` (и `null` — для
`*uuid.UUID` это одно и то же) = «не трогать», UUID = назначить, `clear_account: true` = отвязать; оба
сразу — `422`. Различать «нет поля» и `null` пришлось бы и на сервере (обёртка с `Present`), и на клиенте
(`explicitNulls = false`). Без отвязки нельзя: клиент подставляет последний счёт, и покупка за наличные,
сохранённая не глядя, навсегда искажала бы сверку карты. Старый клиент, который полей не знает, счёт не
стирает.

**Расшифровка разницы — тот же список операций.** Переход со строки сверки открывает операции с
`type=expense`, точными границами месяца и `account_id=` либо `unassigned=true`, иначе список не сойдётся
с `recorded_minor`. `unassigned` — отдельный булев параметр, а не `account_id=none`: UUID-параметр остаётся
типизированным в спеке и в сгенерированном клиенте.

**`005.up` пересобирает `transactions`.** На свежей базе `001` уже создаёт `account_id`, и `ADD COLUMN`
упал бы с `duplicate column`. У `004` так можно только потому, что `002` перед ней сносит колонки
пересборкой `budgets`. Следствие: блок `transactions` внутри `005` заморожен на схеме `v0.6.0`, как
`budgets` в `002`, — следующая колонка операций живёт в `001` и в своей `NNN`, и на свежей базе `005`
сносит то, что создала `001`, а `NNN` возвращает. `005.down` пересборки не требует: `DROP INDEX` +
`DROP COLUMN` (SQLite 3.53 снимает inline `REFERENCES` вместе с колонкой).

**Сверка — агрегат в `/stats`, без пагинации.** Ответ — объект со строкой на счёт и `unassigned_minor`
на весь месяц; прецедент — `GET /stats/monthly`. Постраничная выдача счетов разошлась бы с
`unassigned_minor`.

**Удаление счёта.** `DELETE` при операциях **или** сверках — `409 ACCOUNT_IN_USE`, оба FK `RESTRICT`:
каскад молча унёс бы историю сверок счёта без операций. Перевыпущенная карта — архив.

**Имя счёта** уникально в семье по `name_key` = `strings.ToLower(strings.TrimSpace(name))`, считается в
Go: `COLLATE NOCASE` в SQLite сворачивает только ASCII. Архивный счёт имя занимает. Функция живёт в
`internal/domain/names` (`names.Key`) — второй потребитель, позиции плана 17, уже известен; алгоритм после
релиза не менять: это смысл записанных ключей.

**`CURRENCY_LOCKED` — один вопрос к семейному репозиторию.** `FamilyRepository.HasMonetaryData(ctx, familyID)`:
один `SELECT` с `EXISTS` по операциям и по сверкам (через `accounts.family_id`). `FamilyService` не
обрастает репозиторием на каждый денежный источник; план 17 допишет третий `EXISTS`. Проверка и смена
валюты по-прежнему не атомарны — как и сейчас.

## Technical Details

Схема (`001` + `005`):

```sql
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (LENGTH(TRIM(name)) > 0),
    UNIQUE (family_id, name_key)
);
-- transactions: account_id TEXT REFERENCES accounts(id) ON DELETE RESTRICT
CREATE INDEX idx_transactions_account_date ON transactions(account_id, date);
CREATE TABLE account_reconciliations (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    month TEXT NOT NULL,
    bank_expense_minor INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, month),
    CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
    CHECK (bank_expense_minor >= 0)
);
```

API (`financeAccess`, кроме помеченного):

| Маршрут | Заметки |
|---|---|
| `GET /accounts` | `?archived=true` добавляет архивные; `meta.pagination` |
| `POST /accounts` | `{id?, name}`; идемпотентен по `id`; `409 ACCOUNT_NAME_EXISTS` |
| `PUT /accounts/:id` | `{name?, is_archived?}` |
| `DELETE /accounts/:id` | `adminOnly`; `409 ACCOUNT_IN_USE` |
| `POST /transactions` | `account_id?`; неизвестный или архивный — `422`; повтор по `id` отвечает сохранённой операцией раньше проверки счёта |
| `PUT /transactions/:id` | `account_id?`, `clear_account?`; проверка счёта только когда UUID **отличается** от сохранённого — правка описания у операции с архивным счётом проходит |
| `GET /transactions` | `?account_id=` или `?unassigned=true` (вместе — `422`) через общий `buildFilterConditions`, чтобы `total` совпадал |
| `PUT /accounts/:id/reconciliations/:month` | `{bank_expense_minor, note?}`, полная замена: без `note` заметка очищается; `0 … money.MaxAmount`, поле — указатель, иначе `required` отвергнет законный `0`; `updated_by` из principal |
| `DELETE /accounts/:id/reconciliations/:month` | `404`, если записи нет |
| `GET /stats/reconciliation?month=` | по умолчанию текущий месяц в `family.Location()` |

```json
{"month": "2026-09", "unassigned_minor": 125000, "accounts": [
  {"account": {"id": "…", "name": "Тинькофф", "is_archived": false},
   "recorded_minor": 4310000, "bank_expense_minor": 4500000, "diff_minor": 190000,
   "note": "", "updated_at": "…"}]}
```

`diff_minor = bank − recorded` (плюс — не записали); пока сверки нет, `bank_expense_minor`, `diff_minor`,
`note` и `updated_at` — `null`. Наличие сверки — это наличие строки, а не `bank > 0`.
В `accounts[]` — все неархивные и те архивные, у которых в этом месяце есть расход или сверка.
`month` разбирает `date.ParseMonth`: строгий `YYYY-MM`, месяц 01…12, иначе `422`.

`CURRENCY_LOCKED`: `hasTransactions` заменяется на `HasMonetaryData`; считается наличие строки — нулевая
сверка тоже блокирует.

## Implementation Steps

### Task 1: `date.ParseMonth`

**Files:**
- Modify: `internal/domain/date/date.go`, `date_test.go`

- [ ] `ParseMonth(s string) (first, last Date, err error)` поверх `MonthBounds`; `date.New` нормализует месяц 13 — отсекать до него
- [ ] тесты: `2026-02` → 01…28, `2024-02` → 29, отказ на `2026-13`, `2026-1`, `2026-09-01`, пустой строке
- [ ] `make fmt && make test && make lint`

### Task 2: Миграция `005`

**Files:**
- Modify: `migrations/001_consolidated.{up,down}.sql`, `internal/testhelpers/sqlite.go` (`CleanTables`)
- Create: `migrations/005_accounts.{up,down}.sql`
- Modify: `internal/infrastructure/migrations_test.go`

- [ ] `001`: `accounts` перед `transactions`, колонка и индекс, `account_reconciliations`; в `001.down` новые `DROP … IF EXISTS` — `005.down` уже снёс эти объекты
- [ ] `005.up`: `CREATE TABLE IF NOT EXISTS` для `accounts` **и** `account_reconciliations` (на свежей базе обе уже есть из `001`); пересборка `transactions` (`transactions_new` со всеми `CHECK` из `001:69` → `INSERT … SELECT` с явным списком колонок, включая `tags`, `created_at`, `updated_at`, и `account_id = NULL` → `DROP` → `RENAME`), четыре индекса `001:141-144`, новый индекс; триггер `update_transactions_updated_at` — последним, после копирования. Своего `BEGIN` в файле нет: golang-migrate уже открыл транзакцию
- [ ] `005.down`: `DROP TABLE account_reconciliations`, `DROP INDEX idx_transactions_account_date`, `ALTER TABLE transactions DROP COLUMN account_id`, `DROP TABLE accounts`
- [ ] `CleanTables`: `account_reconciliations` → `transactions` → `accounts`
- [ ] тест: `Up()` → `Migrate(4)` (как `Up → Migrate(1)` в `migrations_test.go:124`; `Migrate(4)` на пустой базе выпущенную v4 не воспроизводит) → семья, категория, операции → `Up()` → строки и timestamps целы, триггер работает, `account_id IS NULL`, индексы на месте
- [ ] тест: заполненные `account_id` и сверка → `Migrate(4)` → операции целы, колонки и таблиц нет → снова `Up()`
- [ ] тест: схема свежей базы совпадает со схемой пути обновления — по `table_info`, `foreign_key_list`, `index_list`/`index_xinfo` трёх таблиц, не по тексту `sqlite_master` (DDL в `001` и `005` оформлен по-разному)
- [ ] `make fmt && make test && make lint`

### Task 3: Счета — домен, репозиторий, сервис, маршруты

**Files:**
- Create: `internal/domain/names/names.go`, `names_test.go`, `internal/domain/account/account.go`, `account_test.go`
- Create: `internal/infrastructure/account/account_repository_sqlite.go`, `account_repository_test.go`
- Create: `internal/services/account_service.go`, `account_service_test.go`
- Create: `internal/application/handlers/accounts.go`, `accounts_test.go`, `tests/integration/accounts_test.go`
- Modify: `services/interfaces.go`, `services/container.go`, `infrastructure/repositories_sqlite.go`, `handlers/repositories.go`, `handlers/types.go`, `handlers/errors.go`, `application/http_server.go`, `internal/run.go`, `internal/testhelpers` (стенд, фабрика `CreateTestAccount`)
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] `internal/domain/names`: `Key(name)` + тест; домен: `Account`, `ErrNameExists`, `ErrInUse`, `ErrNotFound`
- [ ] репозиторий: `Create`, `GetByID`, `List(includeArchived)`, `Update`, `Delete`; в `ErrNameExists` превращается только нарушение `name_key` (по тексту ошибки, как `budgets.id` у бюджетов), FK-нарушение при `DELETE` → `ErrInUse`
- [ ] сервис: `Create`, `Update` пересчитывает `name_key`
- [ ] handler: повтор по `id` — `respondClientID` до вызова сервиса (`categories.go:39`); `respondList` + `pageSlice`, `ignoreWritten(parsePagination)`; коды `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`
- [ ] маршруты в группе `financeAccess`, `DELETE` — `adminOnly`
- [ ] спека: `listAccounts`, `createAccount`, `updateAccount`, `deleteAccount`, схемы `Account`, `CreateAccountRequest`, `UpdateAccountRequest`; `npx @redocly/cli lint`
- [ ] unit-тесты: `names.Key` («Карта» = « карта »), репозиторий (уникальность, архив в списке), сервис, handler
- [ ] интеграция: CRUD, повтор `POST` с тем же `id` → `200`, дубль имени → `409`, `member` не удаляет → `403`, без токена → `401`
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 4: `account_id` в операциях

**Files:**
- Modify: `internal/domain/transaction/transaction.go`, `infrastructure/transaction/transaction_repository_sqlite.go`, `services/transaction_service.go`, `services/dto/transaction_dto.go` (:20, :33, :43), `services/container.go` (конструктору операций нужен `AccountRepository`), `handlers/types.go`, `handlers/transactions.go`
- Modify: соответствующие `*_test.go`, `services/helpers_test.go:729` (фабрика сервиса, мок счетов), `tests/integration/transactions_test.go`, `internal/testhelpers/factories.go`, `testhelpers/integration_server.go` (:88, :122 — стенд собирается вручную)
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] `Transaction.AccountID *uuid.UUID`, `Filter.AccountID *uuid.UUID`, `Filter.Unassigned bool`
- [ ] репозиторий: INSERT, UPDATE, `GetByID.Scan`, общий сканер, оба SELECT; `account_id = ?` и `account_id IS NULL` в `buildFilterConditions`
- [ ] `CreateTransactionRequest.AccountID`, `UpdateTransactionRequest.AccountID` + `ClearAccount *bool` (оба в `isEmpty`), `TransactionFilterParams.AccountID` + `Unassigned`; оба преобразования фильтра; взаимоисключения → `422`
- [ ] сервис, между `transaction_service.go:301` и `:312`: `nil` — ничего; UUID равен сохранённому (сравнение значений, не указателей) — ничего; отличается — счёт существует и не архивный, иначе `422` с `field: account_id`; `clear_account` — `NULL`
- [ ] ответ операции: `account_id` nullable; `Recent` в `stats` не меняется
- [ ] спека: поле в `Transaction`, обоих запросах, параметр `account_id` у `listTransactions`
- [ ] тесты: создание со счётом и без, оба фильтра (`total` = длине выборки), `PUT` без поля сохраняет счёт, `clear_account` отвязывает, оба поля → `422`, архивный → `422`, неизвестный → `422`, правка описания при архивном счёте → `200`, `DELETE` счёта с операцией → `409`
- [ ] тесты идемпотентности: повтор `POST` с тем же `id` после архивации счёта → `200`; повтор с другим `account_id` операцию не переназначает
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 5: Сверки и `GET /stats/reconciliation`

**Files:**
- Create: `internal/domain/reconciliation/reconciliation.go`
- Create: `internal/infrastructure/reconciliation/reconciliation_repository_sqlite.go`, `_test.go`
- Create: `internal/services/reconciliation_service.go`, `_test.go`
- Create: `internal/application/handlers/reconciliations.go`, `_test.go`, `tests/integration/reconciliation_test.go`
- Modify: `services/interfaces.go`, `container.go`, `repositories_sqlite.go`, `handlers/repositories.go`, `handlers/stats.go`, `services/dto/stats_dto.go`, `services/family_service.go`, `family_service_test.go:33`, `services/user_service.go:55` (интерфейс `FamilyRepository`), `infrastructure/user/family_repository_sqlite.go`, `services/helpers_test.go:49` (`MockFamilyRepository`), `handlers/errors.go:87` (текст `CURRENCY_LOCKED`), `http_server.go`, `testhelpers/integration_server.go`
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] репозиторий: `Upsert` (`ON CONFLICT(account_id, month) DO UPDATE`), `Delete`, `ListByMonth`, `Exists`; `RecordedByAccount(familyID, from, to)` — один `SELECT account_id, SUM(amount_minor) … WHERE family_id = ? AND type = 'expense' AND date >= ? AND date <= ? GROUP BY account_id` без `JOIN accounts` (внутренний JOIN потерял бы операции без счёта); `NULL`-группа сканируется nullable-типом и даёт `unassigned`, её отсутствие — 0; сверки семьи выбираются через `accounts.family_id`
- [ ] сервис: `Put` (счёт существует, сумма `0 … MaxAmount`), `Delete`, `Summary(month)` — от `List(true)` всех счетов, затем суммы и сверки; архивный входит только с расходом или строкой сверки в месяце (нулевая сверка считается)
- [ ] handler'ы: `PUT/DELETE /accounts/:id/reconciliations/:month`, `GET /stats/reconciliation`; месяц по умолчанию — из `family.Location()`
- [ ] `FamilyRepository.HasMonetaryData` вместо `hasTransactions` в `family_service`; конструктор `NewFamilyService` при этом теряет `TransactionRepository`, если тот больше ни для чего не нужен (`bootstrap.go:110`, `container.go`)
- [ ] спека: `putReconciliation`, `deleteReconciliation`, `getReconciliationStats`, схемы `ReconciliationRequest`, `ReconciliationStats`, `ReconciliationRow`
- [ ] тесты сервиса: только расходы, границы месяца (1-е и последнее число входят, соседние нет), `unassigned`, `diff` отрицательный, `null` без сверки, архивный счёт
- [ ] интеграция: upsert дважды → одна запись, дописанная операция меняет `diff` без нового `PUT`, `DELETE` счёта со сверкой → `409`, `month=2026-13` → `422`, смена валюты при сверке без операций → `409 CURRENCY_LOCKED`
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 6: Документация сервера

**Files:**
- Modify: `CLAUDE.md`, `migrations/README.md`, `migrations/CHANGELOG.md`, `docs/api/README.md`, `docs/backlog.md`

- [ ] `CLAUDE.md`: таблицы схемы; заморозка блока `transactions` в `005` рядом с правилом про `002`; «Conventions» — счёт без отвязки, `recorded` только расходы и на чтении, `ACCOUNT_IN_USE`; `CleanTables`
- [ ] `migrations/README.md` и `CHANGELOG.md`: `005`, потери при откате (связи со счетами, сверки)
- [ ] `docs/api/README.md`: новые операции

### Task 7: Клиент — `ApiGraph` и «Счета» в настройках

**Files:**
- Modify: `android/core/api/…/ApiGraph.kt`, `ui/settings/SettingsRootScreen.kt`, `SettingsHost.kt`, `SettingsPage.kt`, `SettingsConflicts.kt`, `strings.xml`
- Create: `ui/settings/AccountsScreen.kt`, `AccountsViewModel.kt`, `AccountEditScreen.kt` + тесты

- [ ] методы счетов и сверок в `ApiGraph`
- [ ] список: активные, сворачиваемая группа архивных, пустое состояние с действием
- [ ] создание и переименование (клиентский `id`), архив/возврат, удаление для admin; `409` → текст через `SettingsConflicts.kt`
- [ ] тесты ViewModel: загрузка, создание, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`
- [ ] `make -C android check`

### Task 8: Клиент — счёт в операции, фильтре и распознавании

**Files:**
- Modify: `ui/transactions/TransactionEditScreen.kt`, `TransactionEditViewModel.kt`, `Filters.kt`, `TransactionsViewModel.kt`, `ui/recognize/RecognizeViewModel.kt` и экран
- Create: `ui/transactions/AccountSheet.kt` (по образцу `CategorySheet.kt`)

- [ ] необязательное поле «Счёт»; новая операция получает последний выбранный счёт (DataStore), если он не архивный
- [ ] правка: уже привязанный архивный счёт показывается, но в листе выбора его нет; пункт «Без счёта» в листе шлёт `clear_account: true`
- [ ] фильтр по счёту в списке операций
- [ ] распознавание: один выбор счёта на пачку, уходит в каждый `POST /transactions`
- [ ] тесты ViewModel: подстановка последнего счёта, сохранение без счёта, отвязка, фильтр, пачка со счётом
- [ ] `make -C android check`

### Task 9: Клиент — экран «Сверка»

**Files:**
- Create: `ui/reconciliation/ReconciliationScreen.kt`, `ReconciliationViewModel.kt` + тест
- Modify: `AppScreen.kt`, `MainActivity.kt`, `ui/transactions/Filters.kt`, `TransactionsViewModel.kt`, `strings.xml` (точка входа — по макету)

- [ ] ⚠️ до кода — макет в Figma и решение, где вход: «Обзор» или отдельный пункт
- [ ] переключатель месяца; строка на счёт: записано / в банке / разница; строка «Без счёта»
- [ ] ввод цифры банка и заметки по тапу на «в банке», удаление сверки
- [ ] тап по строке → операции с `type=expense`, границами выбранного месяца и `account_id` либо `unassigned=true`: `Filters.kt:8` знает только ALL/THIS_MONTH/PREV_MONTH по часам телефона — нужен период с явными датами, а `AppScreen.Transactions` (`AppScreen.kt:21`, сейчас `data object`) и его `Saver` — начальный фильтр
- [ ] состояния: нет счетов → действие «Завести счёт»; сошлось (`diff = 0`) отмечено
- [ ] тесты ViewModel: загрузка, сохранение, смена месяца, ошибка сети
- [ ] `make -C android check`

### Task 10: Документация клиента

**Files:**
- Modify: `android/CLAUDE.md`, `android/core/api/CLAUDE.md`

- [ ] требование сервера `v0.6.0`; почему счёт не отвязывается (`explicitNulls = false`)

### Task 11: Verify acceptance criteria

- [ ] все пункты Overview реализованы; клиент `0.7.0` против нового сервера создаёт и правит операции, счёт при правке не теряется
- [ ] `make fmt && make test && make lint` — 0 issues; `make compose-config`
- [ ] `make -C android check`; `make -C android api-check` — на чистом дереве после коммита: цель сравнивает `generated` с Git
- [ ] обновление копии прод-базы: `DATABASE_PATH=<копия бэкапа>`, `migrate --to 5` → `--to 4` → `--to 5` (`migrate` без `--to` только печатает версию), после каждого шага — число операций и `PRAGMA foreign_key_check`
- метрики и R8 не трогаем: `StateReader` считает существующие сущности, keep-правило в `consumer-rules.pro:4` — wildcard по всем моделям API

### Task 12: [Final] Update documentation

- [ ] `CLAUDE.md` «Current direction»: план 16, версии
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Ручная проверка:** сверка реального месяца на телефоне по двум картам; распознавание пачки скриншотов со
счётом.

**Выкатка:** перед деплоем `v0.6.0` — `make sqlite-backup` на mini: `005` пересобирает главную таблицу.
Откат — новым образом при остановленном контейнере `migrate --to 4`, затем старый образ; связи операций со
счетами и сверки при этом теряются. Теги `v0.6.0` и `app-v0.8.0` — после проверки на телефоне.

**Дальше:** план 17 — активы и пассивы (`holdings`, `holding_values`, `GET /stats/net-worth`), миграция `006`;
`CURRENCY_LOCKED` там расширяется ещё раз — на снимки.
