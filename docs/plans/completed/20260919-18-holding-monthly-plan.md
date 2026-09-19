# План 18. Плановый месячный доход и расход позиций капитала

## Overview

У позиции капитала появляется картина обязательств: сколько она приносит и сколько стоит в месяц. Квартира
под сдачу — аренда и коммуналка, вклад — проценты, ипотека — платёж, машина — обслуживание. Числа плановые,
вводятся руками; с операциями не связаны и с ними не сверяются.

- Два необязательных числа у позиции, на «Капитале» — вторая строка у позиции и итог «План в месяц» в шапке.
- Сервер `v0.8.0`, клиент `0.10.0`. Клиент `0.9.0` с новым сервером работает: новые поля он не читает и не шлёт;
  клиент `0.10.0` со старым сервером открывает «Капитал» без планов.

Не входит (обсуждено с codex, тред `holding-cashflow-ideas`): факт из операций через привязку категорий,
сравнение плана с фактом, доходность актива и цена долга — последние две требуют движений капитала
(перевод, тело кредита, пополнение вклада), которых в модели нет; история планов; несколько статей на позицию;
периодичность — годовые платежи вводятся поделёнными на 12.

## Context (from discovery)

- Позиции — план 17: `internal/domain/holding/holding.go` (`Holding` :72), репозиторий
  `internal/infrastructure/holding/holding_repository_sqlite.go` (`selectWithCurrent` :21, `Create` :41,
  `Update` :104 — один `ExecContext` с `COALESCE`, `scanHolding` :269), сервис `internal/services/holding_service.go`
  (`Create` :37, `Update` :82), запросы и ответ `handlers/types.go:188-212`, `toHoldingResponse`
  (`handlers/holdings.go:225`).
- Блокировка валюты: `FamilyRepository.HasMonetaryData` (`infrastructure/user/family_repository_sqlite.go:193`) —
  три `EXISTS`.
- Миграции: `holdings` не пересобиралась, поэтому `ADD COLUMN` упал бы на свежей базе (`001` уже создаст
  колонку); `006` — образец «только `CREATE … IF NOT EXISTS`». `CleanTables` — `testhelpers/sqlite.go:74`
  (список с :78), список таблиц — `migrations_test.go:32`.
- Клиент: `ui/networth/HoldingEditViewModel.kt` (состояние формы :69, `canSubmit` :92), `HoldingEditScreen.kt`,
  `NetWorthScreen.kt`, `NetWorthViewModel.kt`; список — одна страница `HOLDING_LIMIT = 200` вместе с архивными,
  `total` клиент сейчас не смотрит (`NetWorthViewModel.kt:83`). `null` на провод не уходит (`explicitNulls = false`).
- Соединение: `_txlock=immediate`, `MaxOpenConns=1` (`internal/infrastructure/sqlite.go:33`, `:46`) — внутри
  открытой транзакции любой вызов через `r.db` ждёт её же соединение. Тестовый DSN (`testhelpers/sqlite.go:22`)
  этих настроек не несёт.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Сервер (1–3), затем клиент (4–5). Правка спеки — в одном коммите с кодом и `make -C android api-gen`.
- Серверная задача — `make fmt`, `make test`, `make lint` (0 issues); клиентская — `make -C android check`.
- **CRITICAL: update this plan file when scope changes during implementation**

## Testing Strategy

- Unit: доменная проверка плана; сервис на моках; handler через `principalContext`.
- Репозиторий и миграция — in-memory SQLite. Интеграция — `testhelpers.SetupHTTPServer`.
- Клиент — Robolectric-тесты ViewModel'ей и Compose-тест формы (экран с вводом).

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

**Отдельная таблица, а не колонки.** `holding_plans` 1:1 с позицией: миграция `007` — один
`CREATE TABLE IF NOT EXISTS`, без пересборки `holdings`, на которую ссылаются снимки.

**Строка есть, пока план ненулевой.** Оба числа `0` — строка удаляется. Тогда «есть план» = «есть строка»:
четвёртый `EXISTS` в `HasMonetaryData` не смотрит на суммы, а `plan_updated_at` не остаётся от стёртого плана.

**`0` — очистка, `null` не нужен.** В `PUT` отсутствие поля = «не трогать», `0` = убрать. Клиент `null` послать
не может, и здесь это не мешает.

**Запись позиции и плана — одна транзакция, и слияние внутри неё.** `Create` и `Update` репозитория переходят
на `BeginTx`: иначе отказ на плане оставил бы переименованную позицию со старым планом. Сервис передаёт
указатели на присланные числа, не подставляя старых; репозиторий читает сохранённый план через `tx`,
сливает, проверяет итог и пишет. Через `r.db` и публичный `GetByID` внутри транзакции ходить нельзя — при
одном соединении это взаимная блокировка. Два `PUT` разных чисел сохраняются оба, одного числа — побеждает
последний.

**`plan_updated_at` — дата правки, не подтверждение актуальности.** Плановое число никто не обновляет сам; дата
последней правки плана отдаётся клиенту, он подписывает её под числами. Порогов «устарело» нет. Переименование,
архивация и `PUT` без чисел её не двигают. Время — один `time.Now().UTC()` из Go, как у позиции:
`CURRENT_TIMESTAMP` секундной точности сделал бы тест «дата выросла» нестабильным. От `updated_at` позиции она
отличается ровно этим: архивация и переименование плановое число свежее не делают.

**Оба числа у обеих сторон.** Запрета дохода у пассива нет: кэшбэк рублями по кредитке — реальный случай, а
`side` задаёт только знак стоимости в капитале. Поля называются «Поступления» и «Выплаты» у актива и у
пассива одинаково — без особой валидации и без обнуления при смене стороны.

**Итог считает клиент и говорит, по чему он посчитан.** «План в месяц» = сумма по **неархивным** позициям:
проданная квартира аренду не приносит. Серверного агрегата нет — он не сделал бы шапку и список атомарными.
Два условия честности: при `total` больше пришедшей страницы итог не показывается вовсе; рядом с итогом —
«по N из M позиций», потому что незаполненная ипотека выглядит в сумме как бесплатная. С итогом капитала
иначе — он из ряда `net-worth`, потому что архивные в нём участвуют.

**«В месяц» — среднее, а не календарь платежей.** Годовая страховка 120 000 — это 10 000 в плане, но в месяц
оплаты нужны все 120 000. Подпись итога это говорит; периодичности и дат платежей в модели нет.

**Поля ответа необязательные, без `default`.** Новый сервер отдаёт оба числа всегда, включая нули, но в спеке
они не `required`: тогда модель — `Long? = null`, и APK `0.10.0` разбирает ответ сервера после отката (план он
при этом не покажет). `required` уронил бы вкладку «Капитал» целиком. `default: 0` не ставить: рядом с `$ref`
генератор 7.24 его не переносит, проверено codex. Распознавать «сервер без поддержки» клиент не пытается:
старый сервер поля плана в запросе молча игнорирует.

## Technical Details

```sql
CREATE TABLE holding_plans (
    holding_id TEXT PRIMARY KEY REFERENCES holdings(id) ON DELETE CASCADE,
    monthly_income_minor INTEGER NOT NULL DEFAULT 0 CHECK (monthly_income_minor >= 0),
    monthly_expense_minor INTEGER NOT NULL DEFAULT 0 CHECK (monthly_expense_minor >= 0),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (monthly_income_minor > 0 OR monthly_expense_minor > 0)
);
```

Триггера нет: `updated_at` пишет upsert, временем из Go. `updated_at` позиции правка плана тоже двигает.

| Где | Что |
|---|---|
| `Holding` (ответ) | `monthly_income_minor`, `monthly_expense_minor`, `plan_updated_at` — необязательные; сервер шлёт числа всегда (`0` без плана), дату — только при плане |
| `CreateHoldingRequest` | оба поля необязательны, `0 … MaxAmount`; в спеке — `allOf: [{$ref: Money}]` + `minimum: 0`, как в `openapi.yaml:1860`: голый `integer` потерял бы `maximum` |
| `UpdateHoldingRequest` | оба поля необязательны; нет поля — не трогать, `0` — убрать; учитываются в «пустое тело — `422`» |
| `PUT /family` | `CURRENCY_LOCKED` и при строке в `holding_plans` |

Идемпотентный `POST` по `id` отвечает сохранённой позицией, план из повторного тела не применяется — как и
остальные поля. `POST` возвращает `plan_updated_at` сразу: сервис сейчас отдаёт созданный объект без перечитывания.

Экран «Капитал»:

```
┌──────────────────────────────────────────┐   Форма позиции (актив)
│ 6 100 000,00 ₽                           │   Название   [ Квартира на Лесной ]
│ активы 12 500 000 · пассивы 6 400 000    │   Вид        ( недвижимость ˅ )
│ ▁▂▂▃▃▄▅▅▆▆▇█   12 мес                    │   План в месяц
│ План в месяц  +48 200 · −74 300 = −26 100│   Поступления [ 45 000,00 ] ₽
│ среднее · по 3 из 5 позиций              │   Выплаты     [  8 300,00 ] ₽
└──────────────────────────────────────────┘   среднее за месяц; годовые — поделив на 12
Активы
┌──────────────────────────────────────────┐
│ Квартира на Лесной     11 000 000,00     │   У пассива форма та же.
│ недвижимость · 12.01.2026                │
│ +45 000 · −8 300 в мес · план от 03.2026 │
├──────────────────────────────────────────┤
│ Вклад Сбер                850 000,00     │
│ вклад · 01.09.2026                       │
│ +3 200 в мес · план от 09.2026           │
└──────────────────────────────────────────┘
```

- Третья строка позиции — только ненулевые числа; плана нет — строки нет. Строка «План в месяц» в шапке
  скрыта, пока ни у одной неархивной позиции плана нет.
- Итог: доход — цветом дохода, расход — цветом расхода, разница — по знаку; стиль сумм `money/*`.
- «по N из M»: M — неархивные позиции, N — из них с планом; при `N = M` подпись — просто «среднее».
- Пустое поле суммы — `0`; нечитаемый ввод — ошибка поля, а не очистка; сохранённый `0` открывается пустым полем, не «0».
- Шапка сейчас рисуется только при непустом ряде (`NetWorthScreen.kt:124`); строка «План в месяц» живёт в ней же,
  поэтому условие показа шапки — «есть последняя корзина **или** есть итог плана».
- В группе архивных третья строка приглушена, в итог не входит.

## Implementation Steps

### Task 1: Миграция `007`

**Files:**
- Modify: `migrations/001_consolidated.{up,down}.sql`, `internal/testhelpers/sqlite.go`, `internal/infrastructure/migrations_test.go`
- Create: `migrations/007_holding_plans.{up,down}.sql`

- [x] `001`: таблица после `holding_values`; в `001.down` — `DROP TABLE IF EXISTS holding_plans` перед `holding_values`
- [x] `007.up`: `CREATE TABLE IF NOT EXISTS`; `007.down`: `DROP TABLE holding_plans`
- [x] `CleanTables`: `holding_plans` перед `holdings`; список `tables` в `migrations_test.go:33`
- [x] тест: `Up → Migrate(6) → Up` — таблица есть, позиции и снимки целы; на двух базах схемы совпадают по `table_info` / `foreign_key_list`; `CHECK` отвергает строку `0/0` и отрицательные; удаление позиции уносит план; `Migrate(6)` с заполненным планом: позиции и снимки целы, планы потеряны — это записать в `migrations/CHANGELOG.md`
- [x] `make fmt && make test && make lint`

### Task 2: План в домене, репозитории, сервисе и контракте

**Files:**
- Modify: `internal/domain/holding/holding.go`, `holding_test.go`
- Modify: `internal/infrastructure/holding/holding_repository_sqlite.go`, `holding_repository_test.go`
- Modify: `internal/services/holding_service.go` (`HoldingRepository` объявлен здесь, :16), `holding_service_test.go` (`mockHoldingRepo`), `services/interfaces.go:74` (`HoldingService`)
- Modify: `internal/application/handlers/types.go`, `holdings.go`, `errors.go` (константы полей, :145), `holdings_test.go` (`mockHoldingService`), `tests/integration/holdings_test.go`, `internal/testhelpers/factories.go`
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen` → `android/core/api/generated/**`

- [x] домен: `Plan{MonthlyIncomeMinor, MonthlyExpenseMinor, UpdatedAt *time.Time}` в `Holding`; `CheckPlan(income, expense)` — диапазон `0 … MaxAmount`; ошибка — типизированная `PlanError{Field}`, `respondHoldingError` (`holdings.go:198`) достаёт поле через `errors.As`, а не новым сентинелом на каждое число
- [x] репозиторий: `selectWithCurrent` — второй `LEFT JOIN holding_plans`, `COALESCE(…, 0)`; `scanHolding`; `Create` и `Update` — в `BeginTx`; в `Create` при `0/0` строка плана не пишется вовсе (безусловный `INSERT` упёрся бы в `CHECK` и откатил позицию в `500`); запись плана: оба `0` → `DELETE`, иначе `INSERT … ON CONFLICT(holding_id) DO UPDATE`, время — параметром из Go; `PUT` без чисел плана таблицу не трогает и дату плана не двигает
- [x] `Update(ctx, id, name, kind, archived, income, expense *money.Minor)`: `BeginTx` → сохранённый план через `tx.QueryRowContext` → слияние присланного → `CheckPlan` итога → `UPDATE holdings` → upsert/delete → `Commit`; ни одного обращения к `r.db` внутри
- [x] сервис: `CheckPlan` присланных чисел в `Create`; в `Update` — только передать указатели; `Create` возвращает позицию с `Plan.UpdatedAt`
- [x] handler: поля в `CreateHoldingRequest`, `UpdateHoldingRequest` (указатели; в проверке пустого тела), `HoldingResponse` + `toHoldingResponse`; ошибка плана → `422` с `field`
- [x] спека: три необязательных поля в `Holding` без `default`, по два в обоих запросах; после генерации убедиться, что в модели `Long? = null` и что `NetWorthScreenTest.kt:49` (прямой конструктор `Holding`) собирается без правок; описание `CURRENCY_LOCKED` у `currency`
- [x] тесты репозитория: создание с планом, без плана → `0/0` и `plan_updated_at: null`; частичная правка сохраняет второе число; `0/0` удаляет строку; отказ на плане откатывает и переименование (отказ — отрицательное число мимо сервиса; ➕ `CheckPlan` в репозитории идёт после `UPDATE holdings`, так что откат проверяется и без `CHECK`); правка плана двигает `plan_updated_at`, переименование — нет
- [x] один тест на настройках production (`infrastructure.NewSQLiteConnection` на временном файле): `Update` с планом завершается при `MaxOpenConns=1`; два `PUT` разных чисел сохраняют оба
- [x] интеграция: `POST` с планом, `PUT` только с `monthly_expense_minor`, очистка нулями, доход у пассива принимается, отрицательное и выше `MaxAmount` → `422`, `PUT` только с `name` (как шлёт `0.9.0`) план не трогает, повтор `POST` с другим планом → `200` и прежний план, `member` — `200`
- [x] `make fmt && make test && make lint`; `make -C android check`

### Task 3: `CURRENCY_LOCKED` и документация сервера

**Files:**
- Modify: `internal/infrastructure/user/family_repository_sqlite.go` и его тест, `handlers/errors.go` (`ErrMessageCurrencyLocked`, :101), `tests/integration/families_test.go`
- Modify: `CLAUDE.md`, `migrations/README.md`, `migrations/CHANGELOG.md`, `docs/api/README.md`, `docs/backlog.md`, `android/app/src/main/res/values/strings.xml`

- [x] Android-текст `settings_error_currency_locked` (`strings.xml:259`) — дописать планы позиций
- [x] четвёртый `EXISTS` в `HasMonetaryData` — `holding_plans` через `holdings.family_id`
- [x] тест: позиция с планом без снимков и операций → смена валюты `409`; после очистки плана нулями — проходит
- [x] `migrations/README.md` и `CHANGELOG.md`: `007`, при откате планы теряются; `docs/api/README.md`: три поля `Holding`; `docs/backlog.md`: отложенное из «Не входит»
- [x] `CLAUDE.md`: таблица в списке схемы, `CleanTables`, «Conventions» — план позиции: строка живёт, пока ненулевая; `0` очищает; итог считает клиент
- [x] `make fmt && make test && make lint`

### Task 4: Клиент — поля плана в форме позиции

**Files:**
- Modify: `ui/networth/HoldingEditViewModel.kt`, `HoldingEditScreen.kt`, `MainActivity.kt` (:550 — вызов экрана, подпись растёт), `strings.xml`, `HoldingEditViewModelTest.kt`
- Create: `app/src/test/kotlin/tech/shatrov/familyfinances/ui/networth/HoldingEditScreenTest.kt`

- [x] состояние формы: `income`, `expense` — строки ввода, разбор через `parseAmountMinor` (`ui/format/Money.kt:67`), но пустая строка здесь `0`, а не `null` как в `ValueEditor.kt:34`; сохранённый `0` — пустое поле, не `formatAmountInput(0)`; `canSubmit` учитывает изменение плана; diff для `PUT` — только изменённое число, очищенное поле уходит как `0`
- [x] «Поступления» и «Выплаты» у обеих сторон; подсказка «среднее за месяц; годовые — поделив на 12»; пустое поле — `0`, нечитаемый ввод — ошибка поля (`parseAmountMinor == null` при непустой строке)
- [x] форма ищет позицию в той же первой странице списка — при `total > HOLDING_LIMIT` и ненайденной позиции показать ошибку загрузки, а не пустую форму
- [x] ошибка поля с сервера (`monthly_income_minor`, `monthly_expense_minor`) ложится под своё поле
- [x] тесты ViewModel: создание с планом, правка одного числа шлёт одно поле, очистка шлёт `0`, мусор в поле не уходит на сервер, без изменений «Сохранить» выключено; ответ без полей плана (старый сервер) форма открывает с нулями
- [x] Compose-тест: ввод обоих чисел и сохранение
- [x] `make -C android check`
- ➕ `NetWorthScreenTest`: кнопки архива и удаления ушли ниже экрана — `performScrollTo` перед нажатием

### Task 5: Клиент — строка позиции, итог в шапке, релиз

**Files:**
- Modify: `ui/networth/NetWorthScreen.kt`, `NetWorthViewModel.kt`, `ui/format/Dates.kt`, `MainActivity.kt` (:456 — экран получает `zone`), `strings.xml`, их тесты, `app/src/test/kotlin/tech/shatrov/familyfinances/TestFixtures.kt` (JSON `Holding` с планом; старые без полей оставить — это ответ сервера после отката), `android/gradle/libs.versions.toml`, `android/CLAUDE.md`

- [x] третья строка позиции по макету; «план от MM.YYYY» — новый `formatPlanMonth(at, zone)` в `ui/format/Dates.kt` (готового формата там нет), `zone` — параметром экрана; шапка показывается и без ряда, если есть итог плана
- [x] итог «План в месяц» в модели экрана: суммы по неархивным, разница со знаком, «по N из M»; скрыт без планов и при `total` больше страницы
- [x] тесты ViewModel: итог не включает архивные; нет планов — итога нет; только выплаты → отрицательная разница; `N из M`; `total = 201` → итога нет; ответ без полей плана разбирается
- [x] Compose-тест: строка плана есть у позиции с планом и отсутствует без него
- [x] `appVersionName = "0.10.0"`, `appVersionCode = "10"`; `android/CLAUDE.md`: планы позиций есть с сервера `v0.8.0`; со старым сервером `0.10.0` работает без них
- [x] `make -C android check`

### Task 6: Verify acceptance criteria

- [x] все пункты Overview реализованы; клиент `0.9.0` против нового сервера открывает «Капитал» и правит позицию, план при этом не теряется (интеграционный тест `PUT` только с `name`; на устройстве не проверялось)
- [x] `make fmt && make test && make lint` — 0 issues; `make -C android check`; `make -C android api-check` — на чистом дереве после коммита
- [x] копия прод-базы: `migrate --to 7` → `--to 6` → `--to 7`, `PRAGMA foreign_key_check` (на копии локальной `data/budget.db` с версии 1; копии прода здесь нет — повторить перед тегом `v0.8.0`)

### Task 7: [Final] Update documentation

- [x] `CLAUDE.md` «Current direction»: план 18, версии
- [x] перенести план в `docs/plans/completed/`

## Post-Completion

**Выкатка:** `007` только создаёт таблицу. APK `0.10.0` ставится после выката сервера. Откат `migrate --to 6`
удаляет все планы безвозвратно; позиции и снимки остаются.

**Ручная проверка:** завести планы у реальных позиций, сверить итог шапки с ручной суммой, убедиться, что
архивация убирает позицию из итога.

**Дальше (выбрано владельцем 19.09.2026):** сохранение незавершённого импорта скриншотов после смерти
процесса (`docs/backlog.md:165`) и экран «Обзор» с годовым периодом (план 11). Отложено: «закрытие месяца»,
повторяющиеся ожидаемые операции, подушка в месяцах, движения капитала.
