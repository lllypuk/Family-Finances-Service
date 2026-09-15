# План 12 — Периодические бюджеты

Семейная практика владельца: месячный лимит на каждую важную категорию, который «обнуляется» первого
числа и живёт дальше. Сейчас это ручное пересоздание каждого бюджета в начале месяца. Серверная часть
входит в `v0.3.0` вместе с планом 10; клиент — следующий `app-v0.5.0`.
Ревью: codex (тред `recurring-budgets`, два раунда), 14.09.2026 — см. «Решения после ревью».

## Overview

- Бюджет с `recurring: true` — хвост серии: когда его период кончается, сервер сам создаёт следующий
  (`name`, `amount_minor`, `category_id`, `period`, `series_id` те же, даты — следующий календарный
  период) и переносит флаг на новый. Прошлые инстансы остаются строками истории с тем же `series_id`.
- Никаких фоновых задач: серия достраивается лениво при чтении бюджетов (`GET /budgets`,
  `GET /stats/summary`) до «сегодня» по таймзоне семьи. Двум пользователям этого достаточно.
- `spent` считается из транзакций по `start_date…end_date`, а не копится: новый инстанс показывает
  расход своего месяца, даже если материализовался с опозданием.
- Изменение контракта: `recurring` и `series_id` в `Budget`, `recurring` в `CreateBudgetRequest` и
  `UpdateBudgetRequest`, новый код `409 BUDGET_NOT_TAIL`. Ни `scope`, ни массовых операций: правка хвоста
  и есть правка «будущих месяцев», правка прошлого инстанса — только его.
- Попутно закрывается старая гонка `POST`/`PUT`: overlap-проверка переезжает из памяти сервиса в
  транзакцию репозитория (иначе ручной `POST` и материализация могут вставить два бюджета на один месяц).

## Context (from discovery)

- Домен: `internal/domain/budget/budget.go:23` (`Budget`, `Period`), `ValidatePeriod`; `custom` — период
  без длины.
- Сервис: `internal/services/budget_service.go` — `CreateBudget` (`:114`, overlap-проверка `:129` отдельным
  чтением до `Create` `:148`), `GetBudgetByID` (`:161` → `recalculateAndUpdateSpent` `:570` → полнострочный
  `Update` `:576`), `GetBudgetsPage` (`:186`, `total` и страница из одного массива `:215`), `UpdateBudget`
  (`:221`: читает `:231`, проверяет overlap `:244`, пишет вместе с пересчитанным `spent` `:259–262`),
  `applyBudgetUpdate` (`:271`, даты изменяемы `:281`), `GetActiveBudgets` (`:313`), `UpdateBudgetSpent`
  (`:342`), `spentFor` (`:551`), `budgetPeriodsOverlap` (`:591`, включительно), `sameBudgetScope` (`:605`,
  `NULL` = семейная область, не wildcard). `BudgetRepository` — `:49` (не `interfaces.go`).
  Ещё один писатель `spent`: `transaction_service.go:733`.
- Репозиторий: `internal/infrastructure/budget/budget_repository_sqlite.go` — сортировка новых периодов
  вперёд (`:225`), `GetActiveBudgets` (`:251`, `:263`), `Update` пишет все колонки без версии (`:378`, `:409`),
  `Delete` (`:590`); `BeginTx` не используется. Транзакционные образцы с проверкой внутри tx —
  `FamilyRepository.Bootstrap` и `UserRepository.UpdatePassword`/`Patch` («last active admin»).
- Хендлер: `internal/application/handlers/budgets.go` — `GetBudgets` (`:73`, `active_only` берёт
  `familyToday(ctx, repos.Family)` из `helpers.go:98`), `UpdateBudget` — проверка пустого тела (`:143`),
  `buildBudgetResponse` (`:169`); `dto/budget_dto.go:23,35`.
- Статистика: `stats_service.go:98` → `budgetProgress` → `GetActiveBudgets(ctx, today)`; `today` — по
  `family.Location()` (`:80`).
- Даты: `date.MonthBounds` (`date.go:103`) уже есть; `AddMonths` (`:99`) нормализует как `time.AddDate` и
  не трогается.
- Схема: `migrations/001_consolidated.up.sql:88–110` (`budgets`); `idx_budgets_name_period_active` (`:148`)
  — `(family_id, name, start_date, end_date) WHERE is_active = 1`, **без `category_id`**; `002` пересобирает
  `budgets` со своим списком колонок (`002_…up.sql:9,33,40`) и восстанавливает индексы и триггер (`:43,:49`);
  прод на версии 3. `migrations_test.go:118` — образец через `Migrate(N)`. Тесты прогоняют все `*.up.sql`
  подряд (`testhelpers/sqlite.go:122–126`). SQLite — 3.53.3 (`modernc.org/sqlite v1.57.0`, `go.mod:15`):
  `ALTER TABLE … DROP COLUMN` колонки с собственным `CHECK` проходит (проверено codex на реальных `001–003`).
- Спека: `docs/api/openapi.yaml:612–715` (пять операций `budgets`), `Budget` (`:1362`),
  `CreateBudgetRequest` (`:1391`), `UpdateBudgetRequest` (`:1407`).
- Клиент: kotlinx.serialization с `ignoreUnknownKeys = true` (`core/api/.../net/ApiClient.kt:31,50`) —
  новое поле ответа старому клиенту не мешает; сгенерированный `Budget.kt:88` — required без default,
  поэтому новый клиент на старом сервере не распарсит ответ. `ui/budgets/BudgetEditViewModel.kt`
  (`period` `:67`, пресет конца `:173–184`, удаление по сохранённому id `:219`, `locked` `:277`),
  `BudgetEditScreen.kt` (подтверждение удаления `:178`), `BudgetsScreen.kt`, `BudgetsViewModel.kt`
  (одна страница `:107`, фильтр `TODAY`/`ALL`).

## Development Approach

- **testing approach**: Regular (код, затем тесты, как в планах 09–10).
- Каждая задача закрывается `make fmt && make test && make lint` (0 issues) до следующей.
- Тесты — обязательная часть каждой задачи: новый код, изменённый код, ошибки и границы.
- План обновляется при отклонении от него: `[x]` сразу, ➕ для найденных задач, ⚠️ для блокеров.

## Solution Overview

**Модель — «флаг на хвосте» + `series_id`.** `recurring = 1` стоит только у последнего инстанса серии;
`series_id` (id первого инстанса) — у всех её членов, `NULL` у обычных бюджетов. `series_id` не
адресуется никакой операцией API — он нужен серверу, чтобы отличать «прошлый инстанс серии» от
«обычного бюджета» и находить последний инстанс серии независимо от `name`, которое можно менять.

Правила `recurring` в `PUT /budgets/:id` (только если поле прислано; отсутствие — не трогать):
- `true`: разрешено бюджету вне серии (`series_id := id`) и **последнему живому инстансу** серии
  (`MAX(start_date)` среди `is_active = 1` с тем же `series_id`), если у серии нет хвоста; иначе
  `409 BUDGET_NOT_TAIL`. Возобновление остановленной серии достраивает пропущенные месяцы до «сегодня»
  — то же правило «до today», что и у живой серии.
- `false`: у хвоста снимает флаг; у не-хвоста — `409 BUDGET_NOT_TAIL`, если у серии есть другой хвост
  (устаревшая форма: клиент открыл сентябрьский хвост, другой клиент в октябре вызвал `summary`,
  серия продвинулась), и no-op, если серия уже остановлена.
- Даты членов серии (`series_id IS NOT NULL`) не меняются — `422`, поле `start_date`: иначе перенос
  прошлого инстанса за хвост ломает `MAX(start_date)`.
- Устаревший `DELETE` хвоста удаляет только свою строку, серия продолжается на новом хвосте; клиент
  перечитывает список и видит его со значком. Принятое ограничение: `If-Match` для двух пользователей
  дороже, чем это.

**Календарное выравнивание вместо дрейфа.** У повторяющегося бюджета период совпадает с календарным:
`monthly` — с 1-го по последнее число месяца, `yearly` — с 1 января по 31 декабря, `weekly` — любые
7 дней (шаг +7). Иначе `422 VALIDATION_ERROR`, поле `start_date`/`end_date`; `custom` — `422`, поле
`recurring`. Это ровно потребность владельца («обнуляется первого числа»), шаг серии — `MonthBounds`
следующего месяца, без якоря дня. Клиент при включении тумблера сам подставляет границы.

**Все проверки — внутри транзакции репозитория** (образец — `UserRepository.Patch`): на единственном
соединении с `_txlock=immediate` это единственный способ, чтобы между «проверил» и «записал» не
вклинился другой запрос. Один SQL-предикат `taken(tx, scope, start, end, name, excludeID)` = живой бюджет
той же области (`category_id IS ?`) с пересечением дат включительно **или** живой бюджет с тем же
`name` и датами (индекс имени не знает категорий, и без этой ветки серия с именем, занятым бюджетом
другой категории, вставала бы навсегда). Его используют:
- `Create` — `taken` → `budget.ErrOverlap{Name}` → `409 BUDGET_OVERLAP` (как сейчас, но атомарно);
- `Update(ctx, b, expectRecurring)` — правило хвоста и `taken` при смене дат, затем один `UPDATE … WHERE
  id = ? AND recurring = ? AND is_active = 1`; `rowsAffected = 0` → `ErrNotTail` (хвост продвинулся между
  чтением в сервисе и записью);
- `Advance(ctx, tailID, today)` — достраивает **всю** серию до `today` в одной транзакции:

```
BEGIN IMMEDIATE
  tail := SELECT … WHERE id = ? AND recurring = 1 AND is_active = 1   # нет строки → ErrNotTail
  created := 0
  for cand := tail.Next(); cand.StartDate <= today; cand = cand.Next():
      if created + skipped == maxAdvancePeriods: ROLLBACK; return ErrTooFarBehind
      if taken(cand): skipped++; continue           # занятый период пропускается
      INSERT cand (recurring = 1, series_id = tail.series_id, spent_minor = 0); created++
      UPDATE budgets SET recurring = 0 WHERE id = tail.id; tail = cand
COMMIT; return created
```

Сервис `advanceRecurring(ctx, today)` — один проход по `ListRecurring()`: `ErrNotTail` означает, что
параллельное чтение уже достроило эту серию целиком (его транзакция либо закоммитила всё, либо
откатилась — тогда хвост на месте и `Advance` увидит его сам); `ErrTooFarBehind` и прочие ошибки —
`slog` warn, чтение продолжается, серия остаётся на старом хвосте и видна в списке со значком.
Цикла с перечитыванием не нужно. Вызов — в начале `GetBudgetsPage` и `GetActiveBudgets`, через них
— в `StatsService.Summary`. `today` — по таймзоне семьи. `maxAdvancePeriods = 120`.

**Колонки пишутся адресно** («User writes are column-scoped»): `Update` пишет `name`, `amount_minor`,
`start_date`, `end_date`, `recurring`, `series_id`, `updated_at`; `UpdateSpent(id, spent)` — только
`spent_minor`. Иначе `GET /budgets/:id`, пересчитав `spent`, записывал бы устаревший `recurring = 1`
поверх продвинутого хвоста. Все три писателя `spent` (`recalculateAndUpdateSpent`, `UpdateBudgetSpent`,
`transaction_service.go:733`) переходят на `UpdateSpent`; `UpdateBudget` больше не сохраняет `spent`.
Overlap-проверка в памяти (`budgetPeriodsOverlap`, `ValidateBudgetPeriod`, `validateBudgetPeriodForUpdate`)
удаляется — источник истины один.

**Что не делаем (YAGNI):** дата окончания серии, «применить ко всем прошлым», `scope` у операций, смена
периода у существующего бюджета, `If-Match`, cursor-пагинация (вставка между страницами offset-пагинации
даёт повтор — как и сейчас при `POST` между страницами; клиент читает одну страницу), `category_id` в
индексе имени.

## Решения после ревью

Принято из двух раундов codex: overlap и правило хвоста внутри транзакций `Create`/`Update`/`Advance`,
а не в памяти сервиса; конфликт имени как вторая ветка предиката `taken`; адресные `UPDATE` и три
писателя `spent`; `series_id` + `409 BUDGET_NOT_TAIL` с точной семантикой `true`/`false`/отсутствия
поля; неизменяемые даты членов серии; календарное выравнивание; `Advance` достраивает серию целиком в
одной tx вместо цикла в сервисе; `004` — обычный `ADD COLUMN`/`DROP COLUMN` (`002` пересобирает таблицу
без новых колонок, так что на тестовом пути `004` не дублирует их; `DROP COLUMN` с `CHECK` проходит).

**Занятый период — пропуск (решение владельца 14.09.2026).** Ручной бюджет той же категории (или с тем же именем) на следующий месяц:
- *пропуск* (в плане): серия перешагивает его и продолжается; в ноябре вернётся лимит серии, а не
  ручного октября. Ничего не умирает молча; цена — семантика зависит от момента чтения (удаление ручного
  октября до продвижения в ноябрь достроит октябрь, после — нет).
- *остановка* (codex): флаг снимается, серия кончается на хвосте. Проще объяснить, но одна ручная правка
  тихо убивает серию, и заметить это можно только по пропавшему значку.
Выбран пропуск: серия не должна умирать от одной ручной правки.

## Technical Details

- Схема (`001` + `004_budgets_recurring.{up,down}.sql`): `recurring INTEGER NOT NULL DEFAULT 0
  CHECK (recurring IN (0, 1))`, `series_id TEXT` (без FK: первый инстанс мягко удаляется, но не исчезает).
  Индексы не нужны: хвостов единицы.
- Домен: `Budget.Recurring`, `Budget.SeriesID *uuid.UUID`; `Budget.Next() *Budget`;
  `ValidateRecurring(period, start, end)`; ошибки `ErrRecurringCustom`, `ErrRecurringNotAligned`,
  `ErrSeriesDatesFixed`, `ErrNotTail`, `ErrOverlap` (с именем конфликтующего бюджета — переезжает из
  сервиса, где сейчас `ErrBudgetOverlapExists`).
- Репозиторий: `recurring`, `series_id` в `scan`/`INSERT`/`SELECT`; `taken`; `Create` в tx; `Update(ctx, b,
  expectRecurring bool)` в tx с правилом хвоста; `UpdateSpent`; `ListRecurring`; `Advance`.
- Сервис: `CreateBudgetDTO.Recurring bool`, `UpdateBudgetDTO.Recurring *bool`; `ValidateRecurring` при
  создании и при `PUT` с флагом/датами; `ErrSeriesDatesFixed` при смене дат у члена серии;
  `advanceRecurring` перед чтениями; `today` в `BudgetFilterDTO`.
- API: `Budget.recurring` (required), `Budget.series_id` (nullable uuid), `CreateBudgetRequest.recurring`
  (default false), `UpdateBudgetRequest.recurring`; тело `{recurring: …}` без других полей — валидное
  (`:143`); `409 BUDGET_NOT_TAIL` у `updateBudget`. `BudgetProgress` в `summary` не меняется.
- Клиент: `api-gen`; тумблер «Повторять» (скрыт при `custom`; при включении `start` → 1-е число /
  1 января, `end` → конец, поля дат заблокированы; для `weekly` — только `end`); у члена серии даты
  заблокированы всегда; `recurring` уходит в `PUT` **только если изменён**; значок повтора на карточке;
  подтверждение удаления при `recurring` в копии клиента: «Повторение остановится»; `409 BUDGET_NOT_TAIL`
  → «Бюджет уже продвинулся на следующий период» + перечитать список.

## Implementation Steps

### Task 1: Домен — флаг, серия, календарный шаг

**Files:**
- Modify: `internal/domain/budget/budget.go`, `internal/domain/budget/budget_test.go`

- [x] `Budget.Recurring`, `Budget.SeriesID`; ошибки из Technical Details
- [x] `ValidateRecurring(period, start, end)`: `custom` → `ErrRecurringCustom`; `monthly` — `MonthBounds`,
      `yearly` — 1 января…31 декабря, `weekly` — `DaysBetween == 6`; иначе `ErrRecurringNotAligned`
- [x] `Budget.Next()`: `weekly` +7, `monthly`/`yearly` — границы следующего периода через `MonthBounds`
- [x] тесты `ValidateRecurring` (все периоды, положительные и отрицательные), `Next()` (дек→янв, фев в
      високосный, неделя)
- [x] `make fmt && make test && make lint`

### Task 2: Схема и миграция 004

**Files:**
- Modify: `migrations/001_consolidated.up.sql`
- Create: `migrations/004_budgets_recurring.up.sql`, `migrations/004_budgets_recurring.down.sql`
- Modify: `migrations/CHANGELOG.md`, `migrations/README.md`, `internal/infrastructure/migrations_test.go`

- [x] колонки в `001`; `004.up` — два `ADD COLUMN`; `004.down` — два `DROP COLUMN`
- [x] `migrations_test.go`: `Up()` → колонки есть; `Migrate(3)` → нет; `Up()` снова → есть; на свежей базе
      `Up()` проходит (подтверждает, что `002` убирает колонки из `001` до `004`)
- [x] `README.md`: исключение из правила «`002` = `001`» — колонки после `v0.2.0` живут в `001` и в своей
      миграции, `002` заморожена на схеме `v0.2.0`
- [x] `CHANGELOG.md`: «Plan 12: budgets.recurring, budgets.series_id»
- [x] `make fmt && make test && make lint`

### Task 3: Репозиторий — транзакции, `taken`, адресные UPDATE, `Advance`

**Files:**
- Modify: `internal/infrastructure/budget/budget_repository_sqlite.go`, `budget_repository_test.go`
- Create: `internal/infrastructure/budget/budget_recurring_test.go`
- Modify: `internal/services/budget_service.go` (интерфейс `BudgetRepository`)
- Modify: `internal/testhelpers/sqlite.go` (INSERT-ы фабрик, если перечисляют колонки)

- [x] `recurring`, `series_id` в `scan`/`INSERT`/`SELECT`
- [x] `taken(ctx, tx, …)` — один запрос с двумя ветками (область+даты, имя+даты)
- [x] `Create` — `BeginTx` + `taken` → `budget.ErrOverlap{Name}`
- [x] `Update(ctx, b, expectRecurring)` — `BeginTx`; правило хвоста (`MAX(start_date)`, `SUM(recurring)` по
      `series_id` среди живых); `taken` при смене дат; один `UPDATE` без `spent_minor` с `recurring = ?` в
      `WHERE`; `rowsAffected = 0` → `ErrNotTail`
- [x] `UpdateSpent`, `ListRecurring`, `Advance` по псевдокоду (`maxAdvancePeriods`, `ErrTooFarBehind`)
- [x] тесты: round-trip колонок; `Create` с пересечением → `ErrOverlap`, с тем же именем в другой
      категории → `ErrOverlap`; `Update` не трогает `spent_minor`; `Update` с несовпавшим `expectRecurring`
      → `ErrNotTail`; правило `true`/`false` для хвоста, последнего инстанса, прошлого инстанса; `Advance` за
      3 месяца — три инстанса за один вызов, хвост один; повторный `Advance` по старому id → `ErrNotTail`;
      занятый период (по датам и по имени) пропускается; `today` внутри хвоста → 0; лимит →
      `ErrTooFarBehind` и ни одной строки
- [x] `make fmt && make test && make lint`

➕ Сделано здесь, а не в Task 4, потому что иначе не компилируется смена сигнатуры `Update`:
три писателя `spent` переведены на `UpdateSpent` (`recalculateAndUpdateSpent`, `UpdateBudgetSpent`,
`transaction_service.go`), `BudgetRepositoryForTransactions.Update` заменён на `UpdateSpent`.
Проверки в памяти (`ValidateBudgetPeriod`, `budgetPeriodsOverlap`) пока на месте — их удаляет Task 4.

➕ `ErrBudgetOverlapExists = budget.ErrOverlap` (как `ErrBudgetNameExists`), иначе отказ репозитория
доходил бы до хендлера как `500`. Следствие для контракта: дубль имени в **другой** категории на
пересекающихся датах теперь `409 BUDGET_OVERLAP`, а не `BUDGET_NAME_EXISTS` — это вторая ветка `taken`,
и две интеграционные проверки обновлены. `BUDGET_NAME_EXISTS` с живыми строками больше недостижим
(частичный индекс `WHERE is_active = 1` — подмножество предиката `taken`); маппинг оставлен.

⚠️ Правило «`recurring: false` у не-хвоста при живом хвосте → `409`» в репозиторий не влезает: в
сигнатуре нет признака «поле прислано», а без него переименование прошлого инстанса тоже давало бы
`409`. Остаётся сервису (Task 4) — через `ListRecurring` и сравнение `series_id`.

➕ Фикстуры на прямом `repo.Create` больше не могут класть два бюджета одной области на один период
(`tests/integration/budgets_test.go`, `api_pagination_test.go`) — периоды в них разведены.

### Task 4: Сервис — флаг в create/update, писатели `spent`, материализация

**Files:**
- Modify: `internal/services/budget_service.go`, `budget_service_test.go`
- Modify: `internal/services/transaction_service.go`, `transaction_service_test.go`
- Modify: `internal/services/dto/budget_dto.go`
- Modify: `internal/services/stats_service_test.go` (моки, если ломаются)

- [x] `CreateBudgetDTO.Recurring`, `UpdateBudgetDTO.Recurring *bool`, `BudgetFilterDTO.Today`
- [x] `CreateBudget`: `ValidateRecurring`, `SeriesID = ID` при флаге; overlap-проверка в памяти удалена,
      `ErrOverlap` из репозитория → `ErrBudgetOverlapExists`
- [x] `UpdateBudget`: `ErrSeriesDatesFixed`; `ValidateRecurring` для итоговых дат при флаге; вызов
      `Update(ctx, b, readRecurring)`; `ErrNotTail` наружу; `spent` не сохраняется
- [x] три писателя `spent` → `UpdateSpent`; `ValidateBudgetPeriod`/`validateBudgetPeriodForUpdate` удалены
- [x] `advanceRecurring(ctx, today)` — один проход; вызов в `GetBudgetsPage` и `GetActiveBudgets`;
      `ErrNotTail` молча, остальное — warn
- [x] тесты: серия на три месяца назад → три инстанса; `custom`/невыровненные даты; смена дат у члена
      серии; `ErrNotTail` пробрасывается; `Recurring: true` на бюджете вне серии заводит серию; `today` до
      конца хвоста — ничего; ошибка `Advance` не роняет список; `spent` пишется через `UpdateSpent`
- [x] `make fmt && make test && make lint`

➕ `services.ErrBudgetNotTail = budget.ErrNotTail` (как остальные алиасы) — хендлеру Task 5
незачем импортировать домен ради одного сентинела.

➕ `ValidateBudgetPeriod` убран и из интерфейса `services.BudgetService` (плюс два мока):
проверка жила только ради overlap-а в памяти. `BudgetRepository.GetByPeriod` остался — им
пользуются интеграционные тесты.

➕ Тесты сервиса на семантику пересечения в памяти (общий день, соседняя категория,
`LegacySharedDayPair`) удалены: правило целиком в транзакции репозитория и покрыто там.
Вместо них — проброс `budget.OverlapError` из `Create`.

### Task 5: HTTP — поля, коды ошибок, спека

**Files:**
- Modify: `internal/application/handlers/budgets.go`, `budgets_test.go`, `helpers.go`
- Modify: `docs/api/openapi.yaml`
- Modify: `tests/integration/budgets_test.go`

- [x] `recurring` в `CreateBudgetRequest`/`UpdateBudgetRequest`, `recurring` + `series_id` в `BudgetResponse`;
      проверка пустого тела (`:143`) учитывает `recurring`
- [x] `ErrRecurringCustom`/`ErrRecurringNotAligned`/`ErrSeriesDatesFixed` → `422` с полем; `ErrNotTail` →
      `409 BUDGET_NOT_TAIL`
- [x] `GetBudgets` без `active_only` тоже передаёт `familyToday`
- [x] `openapi.yaml`: схемы, `409` у `updateBudget`, описание ленивой материализации, выравнивания и
      правила хвоста у `recurring`
- [x] интеграционные тесты: `POST` с `recurring` за три месяца назад → `GET /budgets?active_only=true`
      отдаёт текущий месяц с `recurring: true`, тем же `series_id`, `spent_minor` = расход этого месяца;
      прошлые в полном списке с `recurring: false`; `GET /stats/summary` показывает текущий;
      `PUT {recurring:false}` хвоста → `200`, серия больше не растёт; `PUT {recurring:false}` прошлого при
      живом хвосте → `409`; `PUT {recurring:true}` последнего инстанса остановленной серии → `200` и
      достройка; `PUT {start_date}` члена серии → `422`; `custom + recurring` → `422`; `POST` с пересечением
      → `409 BUDGET_OVERLAP` (как раньше)
- [x] `make fmt && make test && make lint`

### Task 6: Android — тумблер, выравнивание дат, значок, `409`

**Files:**
- Modify: `android/core/api/generated/**` (через `make -C android api-gen`)
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModel.kt`,
  `BudgetEditScreen.kt`, `BudgetsScreen.kt`, `BudgetDates.kt`, `android/app/src/main/res/values/strings.xml`
- Modify: `android/gradle/libs.versions.toml` (`appVersionName` 0.5.0, `appVersionCode`)
- Modify: тесты ViewModel в `android/app/src/test/...`

- [x] `make -C android api-gen`, закоммитить generated; `make -C android api-check` зелёный
- [x] `BudgetDates.kt`: `alignedStart(period, date)` — 1-е число / 1 января / без изменений для `weekly`
- [x] `BudgetEditViewModel`: `recurring`; при включении — выравнивание и блокировка дат; при `custom`
      сбрасывается и скрыт; даты члена серии заблокированы; `recurring` в `PUT` только при изменении;
      `409 BUDGET_NOT_TAIL` → сообщение + перезагрузка списка
- [x] `BudgetEditScreen`: `Switch` «Повторять» под периодом; строка «Повторение остановится» в
      подтверждении удаления, когда в копии клиента `recurring`
- [x] `BudgetsScreen`: значок повтора на карточке
- [x] тесты ViewModel: `custom` сбрасывает флаг; выравнивание; флаг в `PUT` только при изменении; `409` →
      ошибка + reload
- [x] `make -C android check`

### Task 7: Verify acceptance criteria

- [x] сценарий владельца: «Продукты, сентябрь, recurring» → 1 октября `active_only` отдаёт октябрьский с
      расходом октября и тем же `series_id`, сентябрьский в полном списке
- [x] ручной бюджет на октябрь заранее → серия перешагивает в ноябрь с лимитом серии
- [x] два клиента: остановка серии со старой формы → `409`, после перечитывания — с нового хвоста → `200`
- [x] `make pre-commit`, `make -C android check`

### Task 8: [Final] Update documentation

- [x] `CLAUDE.md`: пункт в «Conventions» (хвост, `series_id`, материализация при чтении, выравнивание,
      `BUDGET_NOT_TAIL`, `taken` в транзакции, адресные `UPDATE` бюджетов); правка про `002` в «Database &
      migrations» (in-memory overlap-проверки в `CLAUDE.md` не было — удалять нечего)
- [x] `docs/specs/005-api-only-redesign.md`: решение о модели серии — A-14, строка 12 в таблице планов
- [x] `docs/backlog.md`: раздел плана 12 (закрытое и сознательно оставленное), гонка `POST`/`PUT` закрыта
- [x] перенести план в `docs/plans/completed/`

## Post-Completion

- Порядок релиза строго: сервер `v0.3.0` (миграция `004` при старте), затем `app-v0.5.0`. Клиент 0.4.x на
  сервере 0.3.0 работает (`ignoreUnknownKeys`); клиент 0.5.0 на сервере 0.2.x — нет (`recurring` required
  без default в сгенерированной модели). Откат сервера после выхода клиента 0.5.0 ломает клиент.
- Проверка на телефоне: создать повторяющийся бюджет с датами прошлого месяца и убедиться, что
  главная показывает текущий.
