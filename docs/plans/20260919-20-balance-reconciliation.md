# План 20. Сверка остатков вместо сверки расходов по выписке

## Overview

Семья сверяется так: в конце месяца фиксирует остатки по картам и наличке, сравнивает изменение их суммы с
приходом−расходом месяца, ищет незаписанное, остаток закрывает одной корректирующей операцией. Нынешняя сверка
(план 16: цифра расходов из банка на счёт, сравнение с расходами, привязанными к счёту) этого не умеет и
заменяется **целиком**.

- Сервер `v0.9.0`: таблица `account_balances` вместо `account_reconciliations`, `PUT/DELETE
  /accounts/{id}/balances/{month}`, новая форма `GET /stats/reconciliation`.
- Клиент `0.12.0`: экран «Сверка» — итог «остатки против операций», ввод остатков, «Операции месяца»,
  «Закрыть разницу».
- Один MR на обе стороны. Контракт ломается: у клиента 0.11.0 перестаёт работать только экран сверки.

Переводы во вклады и погашения кредитов семья пишет обычным расходом/приходом, поэтому формула без поправок:
`gap = (closing − opening) − (income − expense)`. Перевод **между своими счетами** сумму остатков не меняет и
операцией не пишется — записанный, он даёт `gap` на свою сумму без всякой диагностики; это говорит и подсказка
на экране.

**Не делаем:** статус «месяц закрыт», посчётную сверку по операциям, переводы между счетами, снимки на
произвольную дату, автозаполнение остатков, серверную категорию корректировки.

## Context (from discovery)

Сервер, всё про нынешнюю сверку:
- `migrations/001_consolidated.{up,down}.sql`, `005_accounts.{up,down}.sql` (заморожена — не трогать)
- `internal/domain/reconciliation/reconciliation.go`, `internal/services/reconciliation_service.go` (+ test),
  `internal/services/dto/reconciliation_dto.go`, `internal/services/interfaces.go` (`ReconciliationService`),
  `internal/services/container.go`, `internal/services/helpers_test.go`
- `internal/infrastructure/reconciliation/reconciliation_repository_sqlite.go` (+ test),
  `internal/infrastructure/repositories_sqlite.go`
- `RecordedByAccount`: `internal/services/transaction_service.go:85`,
  `internal/infrastructure/transaction/transaction_repository_sqlite.go`, `recorded_by_account_test.go`
- `HasMonetaryData`: `internal/infrastructure/user/family_repository_sqlite.go` (+ test)
- `internal/application/handlers/{reconciliations.go,reconciliations_test.go,types.go,errors.go,repositories.go}`,
  маршруты `internal/application/http_server.go:280-281,305`
- `internal/testhelpers/{sqlite.go,integration_server.go}`, `tests/integration/reconciliation_test.go`,
  `internal/infrastructure/migrations_test.go`, `internal/infrastructure/account/account_repository_test.go`
- `docs/api/openapi.yaml` (`putReconciliation`, `deleteReconciliation`, `getReconciliationStats`, схемы
  `Reconciliation*`), `docs/api/README.md`, `migrations/{README,CHANGELOG}.md`

Клиент:
- `ui/reconciliation/{ReconciliationScreen,ReconciliationViewModel}.kt` (+ тесты)
- `ui/home/{HomeViewModel,HomeScreen}.kt`: `ReconciliationCard.Ready(month, matched, total)`,
  `reconciliationMonth(today)` — до 10-го числа сверяется прошлый месяц (остаётся)
- `ui/transactions/Filters.kt:60` `reconciliation(...)` — фильтр по счёту; `AppScreen.kt`
  (`Transactions.reconciliation`, `AppScreen.Reconciliation`), `MainActivity.kt`
- тесты: `AppScreenSaverTest`, `TransactionsViewModelTest`, `HomeViewModelTest`, `HomeScreenTest`, `TestFixtures`

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Задачи 1–4 — сервер, каждая кончается `make fmt && make test && make lint` (0 issues); 5–7 — клиент,
  `make -C android check`.
- Чтобы задачи 1–2 кончались зелёными, `008` сначала только создаёт `account_balances`; `DROP TABLE
  account_reconciliations`, чистка `001` и удаление старого кода — в задаче 3, одним коммитом с переключением
  сервиса. `008` не выпущена, править её до мержа можно.
- Пакеты и сервис не переименовываются (`reconciliation`, `ReconciliationService`, экран «Сверка») — меняется
  содержимое; так меньше шума в wiring и в навигации клиента.
- **CRITICAL: update this plan file when scope changes during implementation**

## Testing Strategy

- Репозиторий и миграции — in-memory SQLite; upgrade-путь `Migrate(7)→8→7→8`.
- Арифметика сверки — юнит-тесты сервиса на моках; HTTP-коды, роли и `CURRENCY_LOCKED` — `tests/integration`.
- Клиент — Robolectric: ViewModel, Compose-экран, `AppScreenSaverTest`.
- Покраснеют и правятся в задаче 3: `recorded_by_account_test.go` (удаляется), `family_repository_test.go`,
  `account_repository_test.go`, четыре места `migrations_test.go` (см. задачу 3), тесты OpenAPI-покрытия —
  поэтому спека правится там же, где маршруты. `tests/integration/reconciliation_test.go` переписывается в
  задаче 3 до зелёного минимума, сценарии — в задаче 4.

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

Хранится только остаток счёта на конец месяца; всё остальное считается на чтении, как и раньше, — дописанная
операция закрывает разницу без нового `PUT`. Корректировка — обычная транзакция, которую клиент предзаполняет;
сервер о ней не знает. Первый месяц — не особый случай: остатки вводятся за предыдущий месяц и становятся
стартом.

Привязка операций к счетам для сверки больше не нужна и остаётся справочной.

## Technical Details

### Схема

```sql
CREATE TABLE IF NOT EXISTS account_balances (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    month TEXT NOT NULL,
    balance_minor INTEGER NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, month),
    CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]')
);
```

`balance_minor` со знаком (кредитка), `note` нет. `updated_at` пишет upsert из Go (`time.Now().UTC()`,
`excluded.updated_at`), не `CURRENT_TIMESTAMP`: секундная точность не различает два upsert подряд — комментарий
об этом переезжает из блока `account_reconciliations` в `001`. `008.up` в итоге: создать + `DROP TABLE
account_reconciliations`; `008.down`: `DROP TABLE account_balances` + пересоздать `account_reconciliations`
пустой в виде из `005` (цифры банка теряются — осознанно). В `001` блок `account_reconciliations` заменяется на
`account_balances`; на свежей БД `005` создаёт старую таблицу, `008` её роняет, `CREATE … IF NOT EXISTS` в `008`
— no-op. Семьи в таблице нет: чтение идёт через `JOIN accounts`, как в старом репозитории.

### Контракт

- `PUT /api/v1/accounts/{id}/balances/{month}` `{balance_minor}` → `200 {data: AccountBalance}`;
  `DELETE` → `204`. `financeAccess`. `month` не `YYYY-MM` или позже текущего в `family.Timezone` — `422`;
  `|balance_minor|` больше `Money.maximum` — `422`; `ACCOUNT_NOT_FOUND`, `BALANCE_NOT_FOUND` — `404`.
  `PUT` на архивный счёт разрешён (дозаполнить прошлое).
- `balance_minor` в спеке — `$ref: Money` с описанием про знак, как `diff_minor` и `net_minor` (у `Money` нет
  `minimum`); сервер проверяет `|x| ≤ money.MaxAmount`.
- Каждая новая операция — свой `operationId` и блок `400/401/403/404` с `$ref: Error`, как у нынешнего
  `deleteReconciliation` (одного `204` тесту покрытия мало).
- `GET /api/v1/stats/reconciliation?month=`:

```
month
opening_minor | null     сумма остатков прошлого месяца
closing_minor | null     сумма остатков этого
income_minor, expense_minor
gap_minor | null         (closing − opening) − (income − expense)
complete                 оба края заполнены
accounts[]: { account, opening_minor | null, closing_minor | null, updated_at | null }
```

  Суммы — `int64` без `Money.maximum` (как net-worth), в Go `money.Minor`. `gap > 0` — не записан приход или
  записан лишний расход; `< 0` — наоборот. Будущий `month` — `422`; без `month` — текущий месяц в
  `family.Timezone`, как сейчас (записать в спеку).

### Правила расчёта (`reconciliationService.Summary`)

Все даты — в `family.Timezone`; `prev` — месяц перед сверяемым.

- **Список счетов:** неархивные с `created_at` не позже конца сверяемого месяца + архивные, у которых есть
  строка за этот месяц или **ненулевая** за `prev`. Счёт, заведённый после сверяемого месяца, в него не
  попадает — иначе карта, открытая 1-го числа, навсегда ломает `complete` прошлого месяца, который «Главная»
  сверяет до 10-го.
- **`opening` счёта:** строка за `prev`; нет строки и `created_at` внутри сверяемого месяца — `0`; иначе пропуск.
  Строка всегда побеждает исключение: счёт, на котором деньги были до заведения в приложении («Наличные»),
  дозаполняется переключением месяца назад — иначе `gap` молча врёт на этот остаток.
- **`closing` счёта:** строка за месяц; у архивного без строки — `0`, не пропуск (карта закрыта, деньги ушли в
  `closing` другого счёта). Так архивный счёт сам выпадает из списка через месяц. Даты архивации нет, поэтому
  счёт, архивированный позже сверяемого месяца, тоже получит `0` — принято.
- Край заполнен, когда у каждого счёта списка есть значение. **Пустой список** — `opening`/`closing`/`gap` =
  `null`, `complete = false` (иначе «у каждого» истинно на пустом и `gap` = −(приход − расход)).
- `opening`/`closing` — `null`, пока свой край не заполнен; `gap` — пока не заполнен любой.
- `income`/`expense` — `TransactionRepository.GetTotalsByMonth(first, last)` (уже есть, сервис его уже держит):
  строки `MonthTotal` по `type`. Нового SQL нет.
- Оба края читаются одним запросом `ByMonths(ctx, prev, month)`.
- Будущий месяц — сентинел `services.ErrReconciliationMonthInFuture`; хендлер `GET` сейчас сводит любую ошибку
  к `500` (`reconciliations.go:80-83`) и получает ветку `422`.

### Клиент

- Карточка итога: «Остатки: было → стало, Δ», «Операции: приход − расход, Δ», «Разница». Незаполненный край:
  «Заполните остатки: N счетов» (этот месяц) / «Нет остатков за <месяц>» + кнопка, переключающая месяц назад.
  Подпись по знаку `gap`; `0` — «Сошлось». Месяц позже текущего не выбирается — `MonthSwitcher`
  (`ReconciliationScreen.kt:118-136`) сейчас без границы, ему нужен `today` параметром. Подсказка: «перевод
  между своими счетами операцией не пишется».
- Строка счёта: имя, `opening` серым, `closing`; тап — диалог с одним числом, минус разрешён; «Сохранить» →
  `PUT`, «Очистить» → `DELETE`.
- «Операции месяца» → `AppScreen.Transactions(filters = даты месяца, reconciliation = month)`; фильтр по счёту
  из `Filters.reconciliation` убирается.
- «Закрыть разницу» → обычная форма новой операции: сумма `|gap|`, тип `income` при `gap > 0`, иначе
  `expense`; дата — последний день месяца, но не позже сегодня; описание «Корректировка сверки»; категория за
  пользователем. После сохранения — назад в сверку; перечитывание даёт уже существующий
  `LifecycleResumeEffect(current.month)` (`MainActivity.kt:649-653`), кэша у экрана нет. Форма уже ставит
  `listStale/homeStale/overviewStale/budgetsStale` (`MainActivity.kt:834-842`) — возврат в сверку, а не в
  `Transactions`, не должен их пропустить.
- Навигация формы: `TransactionEdit.back` (`AppScreen.kt:54`) типа `Transactions` — расширить до `AppScreen`;
  восстановление `:169` (`as? Transactions ?: Transactions()`) после этого не должно молча уводить на
  «Операции». Ключ `Saver` (`:125`, разбор `split(':', limit = 3)` на `:165`): ключ `back` содержит двоеточия и
  обязан остаться последним — поля предзаполнения вставляются **перед** ним, `limit` растёт.
  `TransactionEditViewModel.kt:97-107` получает параметры предзаполнения через `MainActivity.kt:822-832`;
  `onTypeChange` сбрасывает `categoryId` — тип ставится до категории.
- Главная: `ReconciliationCard` — `Ready(month, gapMinor)` при `complete`, `Incomplete(month)` с приглашением
  заполнить остатки иначе; `NoAccounts`, `Hidden`, `reconciliationMonth` — как есть.

## Implementation Steps

### Task 1: Миграция `008_account_balances` — только создание

**Files:**
- Create: `migrations/008_account_balances.up.sql`, `migrations/008_account_balances.down.sql`
- Modify: `migrations/001_consolidated.{up,down}.sql`, `internal/testhelpers/sqlite.go`,
  `internal/infrastructure/migrations_test.go`

- [x] `008.up` = `CREATE TABLE account_balances`, `008.down` = `DROP`; `005` не трогать
- [x] `001`: добавить `account_balances` рядом с `account_reconciliations` (старая пока остаётся)
- [x] `CleanTables`: добавить `account_balances` перед `transactions` (старая запись пока остаётся)
- [x] тесты: `expectedTables` (`migrations_test.go:33`) + `account_balances`; свежая БД и upgrade с `7` совпадают
  по `schemaOf`/pragma (не по тексту `sqlite_master` — пробелы в `001` и `008` разные)
- [x] `make fmt && make test && make lint`

### Task 2: Домен и репозиторий остатков (старый код не трогается)

**Files:**
- Modify: `internal/domain/reconciliation/reconciliation.go`
- Create: `internal/infrastructure/reconciliation/balance_repository_sqlite.go` (+ `_test.go`)

- [x] `reconciliation.Balance{AccountID, Month, BalanceMinor, UpdatedAt}`, `ValidBalance` — `|x| ≤ money.MaxAmount`,
  `ErrBalanceNotFound`
- [x] репозиторий: `Upsert` (`updated_at` из Go), `Delete` (`ErrBalanceNotFound` по `RowsAffected`),
  `ByMonths(ctx, prev, month)` через `JOIN accounts` по семье
- [x] тесты: upsert поверх меняет `updated_at`, отрицательный и нулевой остаток, `ByMonths` отдаёт оба месяца и
  только их, delete несуществующего, FK `RESTRICT` при удалении счёта
- [x] `make fmt && make test && make lint`

### Task 3: Переключение: сервис, хендлеры, маршруты, спека, снос старой сверки

Один коммит: между его половинами не собирается пакет `services` и расходятся маршруты со спекой.

**Files:**
- Modify: `migrations/008_account_balances.{up,down}.sql`, `migrations/001_consolidated.{up,down}.sql`,
  `migrations/{README,CHANGELOG}.md`, `internal/testhelpers/{sqlite.go,integration_server.go}`,
  `internal/infrastructure/migrations_test.go`,
  `internal/services/{reconciliation_service.go,reconciliation_service_test.go,interfaces.go,container.go,helpers_test.go,transaction_service.go,family_service.go}`,
  `internal/services/dto/reconciliation_dto.go`, `internal/domain/reconciliation/reconciliation.go`,
  `internal/domain/transaction/transaction.go` (`AccountTotal`, :100), `internal/domain/account/account.go`
  (текст `ErrInUse`, :21), `internal/infrastructure/repositories_sqlite.go`,
  `internal/infrastructure/transaction/transaction_repository_sqlite.go`,
  `internal/infrastructure/user/family_repository_sqlite.go` (+ `_test.go`),
  `internal/infrastructure/account/account_repository_test.go`,
  `internal/application/handlers/{reconciliations.go,reconciliations_test.go,types.go,errors.go,repositories.go}`,
  `internal/application/http_server.go`, `internal/run.go`, `tests/integration/reconciliation_test.go`,
  `docs/api/openapi.yaml`, `docs/api/README.md`
- Delete: `internal/infrastructure/reconciliation/reconciliation_repository_sqlite.go` (+ `_test.go`),
  `internal/infrastructure/transaction/recorded_by_account_test.go`

- [x] `008.up` + `DROP TABLE account_reconciliations`, `008.down` + пересоздание из `005`; из `001` старый блок
  убрать, комментарий про `updated_at` перенести; `CleanTables` — убрать старую запись
- [x] `migrations_test.go`, не удаляя гарантий замороженных схем: `expectedTables` (:33);
  `TestMigrations_AccountsRollback` (:272-300) вставляет в старую таблицу на head — опустить до версии 7 или
  перевести на `account_balances`; `TestMigrations_AccountsSchemaMatchesFreshInstall` (:303-315) — сравнивать
  `account_balances`; `TestMigrations_HoldingsUpgradeKeepsAccounts` (:318-346) читает `bank_expense_minor`
  после `Up()` — читать до `008` либо проверять остальное; новый тест `7 → 8 → 7 → 8` с заполненной старой
  таблицей (после `down` она есть и пуста)
- [x] `ReconciliationService`: `PutBalance`, `DeleteBalance`, `Summary` по «Правилам расчёта»;
  `ErrReconciliationMonthInFuture`
- [x] `RecordedByAccount`, `transaction.AccountTotal`, старый репозиторий, алиас в `repositories.go:35-36` —
  удалить; `HasMonetaryData` смотрит в `account_balances`; тексты `ErrInUse`, `CURRENCY_LOCKED`,
  `ACCOUNT_IN_USE` (`family_service.go:22`, `errors.go:53,101,110`) — «balances» вместо «reconciliations»
- [x] хендлеры и маршруты `…/balances/:month`, старые убрать; `BALANCE_NOT_FOUND` вместо
  `RECONCILIATION_NOT_FOUND`; ветка `422` в `GetReconciliationStats`
- [x] спека: `putAccountBalance`, `deleteAccountBalance`, новые `ReconciliationStats`/`ReconciliationRow`,
  `AccountBalance`; старые операции и схемы удалить
- [x] тесты сервиса: оба края заполнены; пропуск на каждом краю → `gap = null`; счёт создан в месяце →
  `opening = 0`; создан в месяце, но строка за `prev` есть → `opening` из строки; создан раньше без строки за
  `prev` → пропуск; создан позже месяца → в списке нет; архивный с ненулевым `opening` без `closing` →
  `closing = 0`, `complete`, в следующем месяце его нет; архивный с нулевым `prev` и без строки → нет; счетов
  нет → всё `null`; отрицательный остаток; граница месяца и `created_at` в таймзоне; будущий месяц; мок
  `GetTotalsByMonth`
- [x] тесты хендлеров: `422` (формат месяца, будущий — `PUT` и `GET`, диапазон), `404`, `400` на битый id;
  `HasMonetaryData` при нулевом остатке
- [x] `make fmt && make test && make lint`
- ➕ `Repositories.Reconciliation` → `Repositories.Balance` (алиас `BalanceRepository`); будущий месяц
  отбивается и в `DELETE`, не только в `PUT`/`GET`

### Task 4: Интеграционные тесты

**Files:**
- Modify: `tests/integration/reconciliation_test.go`

- [x] сценарий ритуала: остатки за два месяца → `gap` → `POST /transactions` на `|gap|` → `gap = 0` без нового
  `PUT`
- [x] роли: admin и member пишут, без токена `401`; `PUT` на архивный счёт — `200`
- [x] `PUT /family` со сменой валюты при одном нулевом остатке — `409 CURRENCY_LOCKED`; `DELETE /accounts/:id` со
  строкой остатка — `409 ACCOUNT_IN_USE`
- [x] `make fmt && make test && make lint`

### Task 5: Клиент — контракт, модель и экран «Сверки»

**Files:**
- Modify: `android/core/api/…` (генерация), `ui/reconciliation/{ReconciliationViewModel,ReconciliationScreen}.kt`,
  `app/src/test/…/ui/reconciliation/{ReconciliationViewModelTest,ReconciliationScreenTest}.kt`, `TestFixtures.kt`, `app/src/main/res/values/strings.xml`

- [x] `make -C android api-gen`; починить компиляцию
- [x] ViewModel: состояние итога (заполнен / не заполнен этот край / не заполнен прошлый), подпись по знаку,
  `saveBalance`/`clearBalance`, перечитывание; месяц не позже текущего
- [x] экран: карточка итога, строки счетов, диалог остатка со знаком, кнопка «Ввести остатки за <месяц>»,
  `today` в `MonthSwitcher`, подсказка про переводы; отжившие строки `reconciliation_*` (`strings.xml:29-31,60-78`)
  удалить
  ➕ `home_reconciliation*` (`strings.xml:29-31`) пока живы: карточка «Главной» до задачи 7 считает «N из M»
  по `closing_minor != null`; знак вводится кнопкой «±» — на цифровой клавиатуре минуса часто нет
- [x] тесты ViewModel: три состояния края, знак подписи, `0` → «Сошлось», ошибка сети; Compose: диалог
  сохраняет отрицательное число, «Очистить» зовёт `DELETE`
- [x] `make -C android check`

### Task 6: Клиент — «Операции месяца» и «Закрыть разницу»

**Files:**
- Modify: `ui/transactions/Filters.kt`, `AppScreen.kt`, `MainActivity.kt`, форма новой операции
  (`ui/transactions/…Edit…`), `ReconciliationScreen.kt`,
  тесты: `TransactionsViewModelTest`, `AppScreenSaverTest`, `ReconciliationViewModelTest`, `app/src/main/res/values/strings.xml`

- [x] `Filters.reconciliation(month)` — только даты месяца, без счёта; «назад» возвращает в сверку (как сейчас)
- [x] `TransactionEdit.back: AppScreen`, поля предзаполнения в ключе `Saver` перед `back`, параметры
  `TransactionEditViewModel` (см. «Клиент» в Technical Details)
- [x] после сохранения — возврат в `AppScreen.Reconciliation(month)`; флаги `*Stale` выставлены
- [x] тесты: предзаполнение по знаку `gap`; дата = конец месяца для прошлого и «сегодня» для текущего;
  `AppScreenSaverTest`: `TransactionEdit` с предзаполнением и `back = Reconciliation` переживает `Saver`, прежние
  ключи читаются; фильтр без счёта
- [x] `make -C android check`

### Task 7: Клиент — карточка на «Главной», версия

**Files:**
- Modify: `ui/home/{HomeViewModel,HomeScreen}.kt`, `HomeViewModelTest`, `HomeScreenTest`,
  `android/gradle/libs.versions.toml`, `app/src/main/res/values/strings.xml`

- [ ] `ReconciliationCard.Ready(month, gapMinor)` / `Incomplete(month)`; тексты карточки
- [ ] `appVersionCode = 12`, `appVersionName = 0.12.0`
- [ ] тесты: `complete` с нулевым и ненулевым `gap`, незаполненный край, нет счетов, ошибка → `Hidden`
- [ ] `make -C android check`

### Task 8: Verify acceptance criteria

- [ ] все пункты Overview реализованы; `grep -ri "bank_expense\|RecordedByAccount\|AccountTotal\|reconciliations/\|RECONCILIATION_NOT_FOUND\|unassigned_minor\|recorded_minor\|diff_minor"`
  по `internal`, `tests`, `docs/api`, `android` (кроме `build/`) пуст (`diff_minor` — кроме чужих схем)
- [ ] `make fmt && make test && make lint` (0 issues), `make -C android check`, `make -C android api-check`
- [ ] `go run ./cmd/server migrate --to 7` и обратно `--to 8` на локальной БД с данными

### Task 9: [Final] Update documentation

- [ ] `CLAUDE.md`: список таблиц, порядок `CleanTables`, абзац «A reconciliation stores only the bank's figure»
  → остатки, `HasMonetaryData`, удаление счёта, «Current direction» (план 20, `v0.9.0`/`0.12.0`, прогон
  `--to 8 → 7 → 8` на копии прода перед тегом)
- [ ] `android/CLAUDE.md`: экран сверки, переход в операции, предзаполнение формы
- [ ] `deploy/README.md:173-179`: абзац отката `008` — `migrate --to 7` пересоздаёт `account_reconciliations`
  пустой и удаляет все остатки
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Выкат.** Сервер уходит в прод сам с мержа в `main`; с этого момента у клиента 0.11.0 экран сверки не работает
(`PUT` — `404`, ответ `GET` не разбирается), остальное цело. Сразу за мержем — APK `0.12.0` на оба телефона.
Теги `v0.9.0` и `app-v0.12.0` — после прогона `migrate --to 8` → `--to 7` → `--to 8` на копии прод-базы.

**Риск.** Есть ли в проде строки в `account_reconciliations`, не проверено (чтение прода из сессии
заблокировано); `008` их удаляет. Перед мержем: `sqlite3 … 'SELECT COUNT(*) FROM account_reconciliations'` на
mini, при ненуле — `make sqlite-backup`-копия остаётся единственным местом, где они лежат.

**На телефоне (релизная сборка):**
1. Ввести остатки за прошлый месяц → за этот → разница совпадает с ручным расчётом.
2. «Операции месяца» → дописать забытую операцию → «назад» → разница уменьшилась.
3. «Закрыть разницу» → выбрать категорию → сохранить → «Сошлось»; операция датирована концом месяца.
4. Кредитка с отрицательным остатком; новый счёт, заведённый в этом месяце, не ломает сверку.
5. Карточка на «Главной» до 10-го числа зовёт сверять прошлый месяц.
