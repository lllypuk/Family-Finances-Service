# План 17. Активы, пассивы и чистый капитал

## Overview

Семья хочет видеть, чем владеет и что должна, и как меняется разница. Значения вводятся вручную
снимками по датам (раз в месяц, квартира — раз в год); операции на них не влияют.

- Справочник позиций: актив или пассив, вид, архив.
- Снимки стоимости по датам с историей.
- `GET /stats/net-worth` — месячный ряд «активы / пассивы / капитал».
- Сервер `v0.7.0`, клиент `0.9.0`. Идёт после плана 16, от него не зависит ничем, кроме номера миграции
  и расширения `CURRENCY_LOCKED`.

Не входит: связь позиций со счетами плана 16 (два независимых списка — карта с остатком заводится
дважды, это принято), другие валюты, автоматическая переоценка, проценты и графики платежей по кредитам.

## Context (from discovery)

- Образец справочника — счета плана 16 (`internal/domain/account`, …); образец ряда —
  `StatsService.Monthly` (`internal/services/stats_service.go:142`, потолок `monthlyMaxMonths` :30,
  «сегодня» из `family.Location()` :148), DTO `services/dto/stats_dto.go:50`, handler `handlers/stats.go:53`,
  маршруты `http_server.go:273`.
- Месяц: `date.MonthKey`, `MonthBounds`, `AddMonths`, `ParseMonth` (план 16).
- `money.MaxAmount` = 99 999 999 999 (`internal/domain/money/money.go:18`) — это и `Money.maximum` в спеке.
- Валюта: `services/family_service.go` — после плана 16 блокируется операциями и сверками.
- Стенд собирается вручную: `internal/testhelpers/integration_server.go:88`, `:122`; `CleanTables` —
  `internal/testhelpers/sqlite.go:78`.
- Клиент: `AppScreen.kt`, `MainActivity.kt`, `ui/AppNavBar.kt` (четыре вкладки: Главная, Операции, Категории,
  Бюджеты), `ui/AppIcons.kt`, `ApiGraph.kt`.
- Решения обсуждены с codex: тред `accounts-reconciliation-holdings`.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Сервер (1–5), затем клиент (6–8). Правка спеки — в одном коммите с маршрутом и `make -C android api-gen`.
- Серверная задача — `make fmt`, `make test`, `make lint` (0 issues); клиентская — `make -C android check`.
- **CRITICAL: update this plan file when scope changes during implementation**
- Существующий контракт не меняется: только новые маршруты.

## Testing Strategy

- Unit: свёртка ряда — чистая функция, таблично; сервисы на моках; handler'ы через `principalContext`.
- Репозитории и миграция — in-memory SQLite.
- Интеграция — `testhelpers.SetupHTTPServer`: роли, коды, покрытие спеки.
- Клиент — Robolectric-тесты ViewModel'ей. E2E в проекте нет.

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

**Знак даёт `side`, а не число.** `value_minor >= 0` всегда; `side` после создания не меняется — иначе
вся история позиции меняет знак задним числом.

**Перенос вперёд без срока давности.** Значение позиции на дату = последний снимок с `date <=` этой даты.
Квартиру переоценивают раз в год, и она не должна выпадать из капитала через месяц. Позиция до первого
снимка в сумму не входит: ряд растёт ступенькой по мере заведения позиций, это честно.

**Архив — только видимость.** Архивная позиция остаётся в ряду по своим снимкам и продолжает переноситься
вперёд; архив прячет её из списка и из ввода. Прекращение стоимости — это снимок `0` («продали», «погасили»).
Фильтр по `is_archived` в ряду переписал бы прошлое. Клиент при архивации позиции с ненулевым значением
предлагает снимок `0` на сегодня; устаревшую позицию от закрытой отличает дата `current`.

**Будущих снимков нет, но `current` всё равно отсекается.** `date` позже «сегодня» в `family.Location()` —
`422`. Запрета на записи мало: после смены зоны семьи назад через полночь сегодняшний снимок окажется
завтрашним, список показал бы его, а ряд — нет. Поэтому `current` = последний снимок с `date <= today`
семьи, и он всегда равен вкладу позиции в последнюю корзину. Телефон в зоне восточнее семьи получит отказ
на своё локальное «сегодня» — это следствие правила; клиент считает «сегодня» по `session.zone`.

**Ряд читает не всю историю, и одним запросом.** Последний снимок каждой позиции строго до `from`
(начальное состояние) `UNION ALL` снимки внутри `[from, to]`; свёртка по месяцам — в Go. Два отдельных
чтения могли бы увидеть разные состояния при параллельной правке истории, а `BeginTx` ради чтения взял бы
write-lock (`_txlock=immediate`). Начальное состояние — коррелированным подзапросом
`… AND v.date = (SELECT x.date … WHERE x.date < ? ORDER BY x.date DESC LIMIT 1)`: он идёт по PK
`(holding_id, date)`; bare-column с `MAX(date)` тоже корректен, но держится на особенности SQLite.
Фильтра по архиву нет ни в одной ветке.

**Корзина — состояние на `min(конец месяца, to)`.** Корзины идут от первого числа месяца `from` шагом
`AddMonths` по первым числам (31-е нормализуется); начальное состояние берётся строго до исходного `from`,
снимок ровно на `from` — уже изменение. Снимок **заменяет** значение позиции, а не прибавляется. `to` по
умолчанию — сегодня в зоне семьи; позже сегодня — `422` с `field: to`, и это проверка только `NetWorth`:
у `Monthly` такого запрета нет, общий помощник границ его не получает.

**Итог на экране — из ряда, не из списка.** Клиент берёт капитал из последней корзины `net-worth`: список
пагинирован и без архивных, а архивные в капитале участвуют.

**Итоги — не `Money`.** `assets_minor` и `liabilities_minor` — суммы нескольких значений и могут превысить
`Money.maximum`: в спеке это `int64` без `maximum` (`minimum: 0` у активов и пассивов, у `net_minor` —
без `minimum`; сам `Money` знаковый и так). В Go остаётся `money.Minor`, в Kotlin — тот же `Long`, третьего
денежного типа не появляется. Переполнение `int64` требует ~9·10⁷ позиций по `MaxAmount`; проверки нет —
это практическое ограничение одной семьи, не гарантия. Старые `stats` остаются под `Money` — не трогаем.

**Удаление — каскадом, и только admin.** `DELETE /holdings/:id` сносит снимки: это исправление ошибочно
заведённой позиции. Проданное имущество — снимок `0` и архив.

## Technical Details

Схема (`001` + `006`; обе таблицы новые, `006.up` — `CREATE TABLE IF NOT EXISTS`, пересборок нет):

```sql
CREATE TABLE holdings (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('asset', 'liability')),
    kind TEXT NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (LENGTH(TRIM(name)) > 0),
    UNIQUE (family_id, name_key)
);
CREATE TABLE holding_values (
    holding_id TEXT NOT NULL REFERENCES holdings(id) ON DELETE CASCADE,
    date TEXT NOT NULL,
    value_minor INTEGER NOT NULL CHECK (value_minor >= 0),
    updated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (holding_id, date),
    CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')
);
```

`kind` проверяется в домене, не в SQL: активы — `cash`, `deposit`, `investment`, `property`, `vehicle`,
`other`; пассивы — `mortgage`, `loan`, `credit_card`, `other`. Вид не из списка своей стороны — `422`.
В спеке `kind` — **строка с перечнем в описании, не `enum`**: закрытый Kotlin-enum уронил бы установленный
APK на первом же новом виде; неизвестный вид клиент рисует как `other`. `side` — `enum`, он не растёт.
`kind` меняется, `side` — нет: поля `side` в `UpdateHoldingRequest` просто нет, присланное игнорируется
(как `user_id` в `CreateTransactionRequest`) — ловить лишний ключ пришлось бы мимо стандартного декодера.

API (`financeAccess`, кроме помеченного):

| Маршрут | Заметки |
|---|---|
| `GET /holdings` | `?archived=true`; у каждой позиции `current {date, value_minor}` или `null`; порядок `name_key, id`; `meta.pagination` |
| `POST /holdings` | `{id?, name, side, kind}`; идемпотентен по `id` через `respondClientID`; `409 HOLDING_NAME_EXISTS` |
| `PUT /holdings/:id` | `{name?, kind?, is_archived?}` |
| `DELETE /holdings/:id` | `adminOnly`; снимки уходят каскадом |
| `GET /holdings/:id/values` | история, новые сверху; `meta.pagination` |
| `PUT /holdings/:id/values/:date` | `{value_minor}` — указатель, `0 … MaxAmount`; upsert; будущая дата — `422`; `updated_by` из principal |
| `DELETE /holdings/:id/values/:date` | `404`, если снимка нет |
| `GET /stats/net-worth?from&to` | по умолчанию 12 месяцев до сегодня; шире 120 корзин — `422`, как у `monthly` |

```json
{"from": "2025-10-01", "to": "2026-09-18", "months": [
  {"month": "2026-09", "assets_minor": 1250000000, "liabilities_minor": 640000000, "net_minor": 610000000}]}
```

`CURRENCY_LOCKED`: в `FamilyRepository.HasMonetaryData` (план 16) добавляется третий `EXISTS` — снимки через
`holdings.family_id`. Считается наличие строки, нулевые и архивные тоже.

### Экран «Капитал» (решение от 18.09.2026)

Пятая вкладка нижней панели — `AppTab.NET_WORTH` после `BUDGETS` (`ui/AppNavBar.kt:18`); пять — предел
панели, шестой не будет. Иконка — `landmark` в том же штриховом наборе `AppIcons`. Макет текстовый, Figma
догоняет позже.

```
Капитал                                          Лист снимка
┌──────────────────────────────────────────┐     ┌────────────────────────────┐
│ 6 100 000,00 ₽                           │     │ Вклад Сбер                 │
│ активы 12 500 000 · пассивы 6 400 000    │     │ Сумма   [ 850 000,00 ] ₽   │
│ ▁▂▂▃▃▄▅▅▆▆▇█   12 мес                    │     │ Дата    [ 18.09.2026 ]     │
└──────────────────────────────────────────┘     │ [ Сохранить ]   История ›  │
Активы                                           └────────────────────────────┘
┌──────────────────────────────────────────┐
│ Квартира           11 000 000,00         │     Пусто:
│ недвижимость · 12.01.2026                │     ┌────────────────────────────┐
├──────────────────────────────────────────┤     │ Здесь будет капитал семьи  │
│ Вклад Сбер            850 000,00         │     │ [ Добавить актив ]         │
│ вклад · 01.09.2026                       │     │ [ Добавить пассив ]        │
└──────────────────────────────────────────┘     └────────────────────────────┘
Пассивы
┌──────────────────────────────────────────┐
│ Ипотека             6 400 000,00         │
│ ипотека · 01.09.2026                     │
└──────────────────────────────────────────┘
В архиве (2) ˅                         FAB +
```

- Шапка — последняя корзина `net-worth`; график — `net_minor` за 12 месяцев, без осей, тап ничего не делает.
- Тап по строке — лист снимка; «История» в нём — экран истории; долгий тап или иконка в истории — правка
  позиции. Позиция без снимка показывает «нет значения» вместо суммы.
- Дата снимка старше 45 дней — приглушённая подпись «давно не обновлялось»; у `property` и `vehicle` порог —
  год.
- FAB — новая позиция: сначала `SegmentedChoice` «Актив / Пассив», затем имя и вид.

## Implementation Steps

### Task 1: Миграция `006`

**Files:**
- Modify: `migrations/001_consolidated.{up,down}.sql`, `internal/testhelpers/sqlite.go` (`CleanTables`)
- Create: `migrations/006_holdings.{up,down}.sql`
- Modify: `internal/infrastructure/migrations_test.go`

- [ ] `001`: обе таблицы и индекс `holdings(family_id, is_archived)`; в `001.down` — `DROP … IF EXISTS`
- [ ] `006.up`: `CREATE TABLE IF NOT EXISTS` обеих, `CREATE INDEX IF NOT EXISTS`; `006.down`: `holding_values`, затем `holdings`
- [ ] `CleanTables`: `holding_values` → `holdings`, обе до `users` и `families`
- [ ] тест: `Up → Migrate(5) → Up` — таблицы есть, данные плана 16 целы; схема свежей базы и пути обновления совпадает по `table_info` / `foreign_key_list` / `index_list`
- [ ] `make fmt && make test && make lint`

### Task 2: Позиции — домен, репозиторий, сервис, маршруты

**Files:**
- Create: `internal/domain/holding/holding.go`, `holding_test.go`
- Create: `internal/infrastructure/holding/holding_repository_sqlite.go`, `holding_repository_test.go`
- Create: `internal/services/holding_service.go`, `holding_service_test.go`
- Create: `internal/application/handlers/holdings.go`, `holdings_test.go`, `tests/integration/holdings_test.go`
- Modify: `services/interfaces.go`, `services/container.go`, `infrastructure/repositories_sqlite.go`, `handlers/repositories.go`, `handlers/types.go`, `handlers/errors.go`, `application/http_server.go`, `testhelpers/integration_server.go`, `testhelpers/factories.go`
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] домен: `Holding`, `Side`, `Kind`, `ValidKind(side, kind)`; имя — через `names.Key` (план 16)
- [ ] репозиторий: CRUD; `List(includeArchived, today)` — одна строка на позицию, `LEFT JOIN` на последний снимок с `date <= today` (по `date`, не по `updated_at`), `ORDER BY name_key, id`; нарушение `name_key` → `ErrNameExists`
- [ ] сервис и handler: `respondClientID` до сервиса, `respondList` + `pageSlice` (`total` — до среза)
- [ ] спека: `listHoldings`, `createHolding`, `updateHolding`, `deleteHolding`; схемы `Holding`, `HoldingCurrent`, запросы; `side` — enum, `kind` — строка
- [ ] unit-тесты: `ValidKind` по обеим сторонам, уникальность имени, архив в списке
- [ ] интеграция: CRUD, повтор `POST` → `200`, `mortgage` у актива → `422`, `side` в `PUT` игнорируется, `member` не удаляет → `403`
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 3: Снимки стоимости

**Files:**
- Modify: `internal/domain/holding/holding.go` (`Value`), репозиторий, сервис, `handlers/holdings.go`, их тесты, `tests/integration/holdings_test.go`
- Modify: `infrastructure/user/family_repository_sqlite.go` (`HasMonetaryData`) и его тест, `handlers/errors.go:87` (текст `CURRENCY_LOCKED` говорит только об операциях)
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] репозиторий: `UpsertValue`, `DeleteValue`, `ListValues(holdingID, page)` с `COUNT`, `HasValues(familyID)`
- [ ] сервис: позиция существует; `0 … MaxAmount`; дата не позже сегодня в `family.Location()` → иначе `422` с `field: date` («сегодня» приходит параметром в чистую проверку — для тестов); архивной позиции можно писать и удалять любые снимки: история правится, и удаление закрывающего `0` вернёт её вклад в капитал — это цена правки истории
- [ ] handler'ы трёх маршрутов; `:date` разбирает `date.Parse`, мусор — `400`
- [ ] третий `EXISTS` в `HasMonetaryData`
- [ ] спека: `listHoldingValues`, `putHoldingValue`, `deleteHoldingValue`, схемы `HoldingValue`, `HoldingValueRequest`
- [ ] тесты: upsert дважды → одна строка, `current` — самый поздний по дате, а не последний записанный; `0` принимается; завтра → `422`; `DELETE` позиции уносит снимки; смена валюты при одном нулевом снимке → `409`
- [ ] тесты `current` в репозитории: удалён последний снимок → предыдущий; удалён единственный → `null`; удалён непоследний → без изменений; снимок с датой позже `today` в `current` не попадает
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 4: `GET /stats/net-worth`

**Files:**
- Create: `internal/services/net_worth.go` (свёртка), `net_worth_test.go`, `internal/services/stats_period.go` (границы), `stats_period_test.go`
- Modify: `internal/services/stats_service.go`, `stats_service_test.go:57` (фабрика `NewStatsService`), `services/dto/stats_dto.go`, `handlers/stats.go` (`respondStatsError` :57 — новая ошибка иначе станет `500`), `handlers/stats_test.go:24` (`MockStatsService`), `http_server.go`, репозиторий позиций (`SeriesValues`), `services/interfaces.go`, `container.go`, `testhelpers/integration_server.go`, `services/helpers_test.go`
- Create: `tests/integration/net_worth_test.go`
- Modify: `docs/api/openapi.yaml`; `make -C android api-gen`

- [ ] репозиторий: `SeriesValues(familyID, from, to)` — один `SELECT … UNION ALL` из Solution Overview, строки с `side`, по возрастанию даты; `EXPLAIN QUERY PLAN` — поиск по PK, не скан
- [ ] `statsPeriod(today, from, to)` вынести из `stats_service.go:148-161` без изменения поведения: отсутствующие границы подставляются **независимо** (только `to` → `from` всё равно от сегодня), дефолт — начало текущего месяца минус 11 → сегодня, не `today.AddMonths(-12)`; существующий тест `stats_service_test.go:482` остаётся зелёным
- [ ] чистая `foldNetWorth(rows, from, to)` по правилам корзины из Solution Overview
- [ ] `StatsService.NetWorth`: `statsPeriod` + своя проверка `to > today` → `422` с `field: to`; шире 120 → `ErrStatsPeriodTooLong`
- [ ] спека: `getNetWorthStats`; `assets_minor`, `liabilities_minor`, `net_minor` — `int64` без `maximum`, `net_minor` без `minimum`
- [ ] тесты свёртки: перенос через пустые месяцы; позиция до первого снимка не входит; два снимка в месяце → поздний; снимок `0` обнуляет вклад; архивная остаётся; начальное состояние до `from`; `from` посреди месяца; снимок ровно на `from`; ровно в последний день месяца; `from = to`; `to` в середине месяца отсекает поздний снимок; пустая история → корзины с нулями; пассивы больше активов → отрицательный `net`; сумма двух позиций выше `MaxAmount`
- [ ] интеграция: ряд на стенде под admin и под member (`financeAccess` пускает обоих), без токена `401`, период 121 месяц → `422`, `to` завтра → `422`
- [ ] `make fmt && make test && make lint`; `make -C android check`

### Task 5: Документация сервера

**Files:**
- Modify: `CLAUDE.md`, `migrations/README.md`, `migrations/CHANGELOG.md`, `docs/api/README.md`

- [ ] `CLAUDE.md`: таблицы; «Conventions» — знак от `side`, перенос вперёд, архив не вычёркивает прошлое, итоги вне `Money.maximum`, `CURRENCY_LOCKED` от трёх источников; `CleanTables`
- [ ] `migrations/*`: `006`, при откате теряется вся история позиций

### Task 6: Клиент — `ApiGraph`, список позиций, ввод снимка

**Files:**
- Modify: `android/core/api/…/ApiGraph.kt`, `AppScreen.kt` (+ `Saver`), `MainActivity.kt`, `ui/AppNavBar.kt` (`AppTab.NET_WORTH`), `ui/AppIcons.kt` (`Landmark`), `strings.xml`
- Create: `ui/networth/NetWorthScreen.kt`, `NetWorthViewModel.kt`, `HoldingEditScreen.kt`, `HoldingEditViewModel.kt`, `ValueSheet.kt` + тесты

- [ ] вкладка `NET_WORTH` и экран по макету из Technical Details; подписи пяти вкладок помещаются на ширине 360dp (проверка в превью и на телефоне)
- [ ] экран: итог капитала — из последней корзины `net-worth`, не суммой списка; группы «Активы» и «Пассивы» с `current` и датой снимка; сворачиваемая группа архивных (`?archived=true`) — она объясняет разницу между итогом и видимым списком; пустое состояние с действием
- [ ] создание и правка позиции: сторона выбирается только при создании, вид — из списка своей стороны
- [ ] ввод снимка по тапу: сумма и дата; «сегодня» и потолок пикера — `LocalDate.now(session.zone)` (`Session.kt:20`), не `LocalDate.now()` как в `TransactionEditViewModel.kt:53`
- [ ] архивация позиции с ненулевым `current` предлагает сначала снимок `0`
- [ ] тесты ViewModel: загрузка, создание, снимок, `HOLDING_NAME_EXISTS`, сценарий архивации
- [ ] `make -C android check`

### Task 7: Клиент — история и график

**Files:**
- Create: `ui/networth/HoldingHistoryScreen.kt`, `HoldingHistoryViewModel.kt`, `NetWorthChart.kt` + тесты
- Modify: `NetWorthScreen.kt`, `AppScreen.kt` (+ `Saver`), `MainActivity.kt`

- [ ] история позиции: список снимков, правка и удаление
- [ ] график ряда `net-worth` за 12 месяцев — свой Canvas в шапке (экрана «Обзор» в приложении нет, переиспользовать нечего); отрицательный капитал — ось нуля внутри графика, ряд из одной точки — точка, не линия
- [ ] тесты ViewModel: пагинация истории, удаление снимка обновляет `current`
- [ ] `make -C android check`

### Task 8: Документация клиента

**Files:**
- Modify: `android/CLAUDE.md`

- [ ] требование сервера `v0.7.0`; итоги ряда — `Long`, не модель `Money`

### Task 9: Verify acceptance criteria

- [ ] все пункты Overview реализованы
- [ ] `make fmt && make test && make lint` — 0 issues; `make compose-config`
- [ ] `make -C android check`; `make -C android api-check` — на чистом дереве после коммита
- [ ] копия прод-базы: `migrate --to 6` → `--to 5` → `--to 6`, `PRAGMA foreign_key_check`

### Task 10: [Final] Update documentation

- [ ] `CLAUDE.md` «Current direction»: план 17, версии
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Figma:** вкладка и экраны «Капитал» — в файл дизайна после возврата лимита MCP или локальным плагином.

**Ручная проверка:** завести реальные позиции, снимки за два месяца, сверить итог с ручным расчётом.

**Выкатка:** `006` только создаёт таблицы — рисков для существующих данных нет. Теги `v0.7.0` и `app-v0.9.0`
после проверки на телефоне.

**Метрики:** `ffs_holdings` в `StateReader` не добавляем — список метрик согласован с observability,
новая метрика — отдельный запрос туда.
