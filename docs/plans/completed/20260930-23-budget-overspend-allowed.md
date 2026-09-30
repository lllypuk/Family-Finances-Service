# План 23. Расход сверх бюджета сохраняется

## Overview

Сервер отвечает `422 VALIDATION_ERROR` («transaction would exceed budget limit») на создание и правку расхода,
который выводит категорию за лимит бюджета. Бюджет — план, а покупка уже случилась: записать её нельзя, пока
кто-то не поднимет лимит. Найдено 30.09.2026 при импорте истории из xlsx — сентябрьский «На продукты»
(40 000 ₽) отклонил операцию на 1 201,93 ₽; импорт прошёл только после подъёма лимита до 40 389,68 ₽.

План: проверка снимается с создания и с правки операции, заодно уходит `409 BUDGET_BELOW_SPENT` на правке бюджета; перерасход виден в бюджете (`remaining_minor < 0`,
`utilization > 1`), как контракт уже и описывает (`Budget.remaining_minor`: «отрицательное при перерасходе»).

Сервер `v0.10.1`. Контракт меняется в двух местах: код `BUDGET_BELOW_SPENT` больше не возвращается, а `spent_minor` в `Budget` и `BudgetProgress` теряет
`Money.maximum` (сумма расходов больше не ограничена лимитом бюджета). Релиз клиента не нужен: модели держат
`Long`/`Double` без проверки диапазона, переименование перерасходованного бюджета уже покрыто тестом
(`BudgetEditViewModelTest.kt:234`).

**Не делаем:** уведомления о превышении, «мягкое» предупреждение в ответе `POST /transactions`, настройку
«жёсткий/мягкий лимит» на бюджете, правки клиента — полоса прогресса заполняется одинаково при 100 % и 150 %
(`coerceIn` в `BudgetsScreen.kt:190`, `HomeScreen.kt:483`), размер перерасхода виден по красному цвету и двум
суммам; отдельная строка остатка — отдельной задачей, если понадобится.

## Context (from discovery)

- Проверка пришла в `TransactionService` коммитом `c498bd5`, в `BudgetService` — `d41d131` (оба 29.08.2025, до
  API-only); решения о ней в `docs/specs` и `docs/plans` нет. `docs/product_brief.md:79–83` называет и
  «категориальные лимиты: ограничения по типам трат», и «исключения для экстренных трат», и уведомления —
  запрет из него не следует, но и не исключён; решение снять его принято 30.09.2026.
- `internal/services/transaction_service.go`: `ErrInsufficientBudget` (29), вызов `ValidateTransactionLimits` в
  `CreateTransaction` (194), сама функция (655), `validateUpdatedTransactionLimits` (≈686–717) и её вызов в правке.
  `findBudgetByCategory` (794) **остаётся**: им пользуется `updateBudgetSpent` (785).
- `ValidateTransactionLimits` объявлен в интерфейсе (`internal/services/interfaces.go:147`) и в двух моках:
  `internal/services/helpers_test.go:460`, `internal/application/http_server_test.go:250`; у `CheckBudgetLimits` —
  `interfaces.go:170`, `helpers_test.go:547`, `http_server_test.go:341`.
- `internal/services/budget_service.go:430` `CheckBudgetLimits` + `ErrInsufficientBudgetFunds` — в интерфейсе
  `internal/services/interfaces.go:170`, вне тестов не вызывается.
- `internal/application/handlers/transactions.go:111,116,482` — ветки `errors.Is(err, services.ErrInsufficientBudget)`.
- Тесты, закрепляющие отказ: `internal/services/transaction_service_test.go:114–141, ≈542`; искать также в
  `internal/services/budget_service_test.go`, `internal/services/helpers_test.go` (моки интерфейса),
  `tests/integration/`.
- `409 BUDGET_BELOW_SPENT` на `PUT /budgets/:id` (`budget_service.go:24, 275`; `handlers/budgets.go:212`,
  `handlers/errors.go:43–44, 104`) отклоняет любую переданную сумму ниже потраченного, в том числе подъём лимита
  (потрачено 45 000 из 40 000, новые 42 000 → `409`). Решение 30.09.2026: отказ снимается совсем — бюджет с
  суммой ниже потраченного есть просто перерасходованный бюджет. Тесты на отказ:
  `budget_service_test.go:363, 758, 806`, `handlers/budgets_test.go:239`, `tests/integration/budgets_test.go:923`.
  Клиентская строка `BUDGET_BELOW_SPENT` (`BudgetEditViewModel.kt:45`) остаётся — она нужна против старого сервера.
- Инварианта `spent <= amount` в хранении нет: `syncSpent`, `UpdateSpent`, `recalculateAndUpdateSpent`,
  `materializeNext` пишут вычисленную сумму без сравнения с лимитом; CHECK в `001` и `002` перерасход допускают.
- Схема: `Money` без `minimum`, `utilization` без `maximum` — отрицательный остаток и > 100 % / > 1 допустимы.
  `spent_minor` наследует `Money.maximum` = 99 999 999 999, а сумма расходов потолка не имеет.

## Development Approach

- **testing approach**: Regular (код, затем тесты)
- каждая задача заканчивается тестами; `make fmt`, `make test`, `make lint` (0 issues) перед сдачей
- план правится, если объём меняется

## Testing Strategy

- **unit**: сервисные тесты на создание и правку расхода сверх лимита — операция сохранена, `spent_minor`
  бюджета больше `amount_minor`
- **integration** (`tests/integration`): `POST /transactions` сверх лимита → `201`; `GET /budgets` показывает
  отрицательный `remaining_minor`; `PUT /budgets/:id` перерасходованного бюджета без `amount_minor` → `200`
- **android**: unit-тесты экранов бюджета на перерасход, если отрисовку придётся править

## Progress Tracking

- выполненное — `[x]` сразу; новое — с ➕, блокеры — с ⚠️

## Solution Overview

Удаление, а не флаг: отказа не остаётся ни в сервисе операций, ни в интерфейсе бюджетного сервиса. Учёт
`spent_minor` не трогаем — он и сейчас пересчитывается из операций и лимитом не ограничен.

## Technical Details

- Уходят: `ErrInsufficientBudget`, `ValidateTransactionLimits`, `validateUpdatedTransactionLimits`,
  `CheckBudgetLimits`, `ErrInsufficientBudgetFunds`, ветки обработчика, методы интерфейсов и моков.
- `spent_minor` в `Budget` и `BudgetProgress` — `int64` без `Money.maximum`, как суммы в `StatsNetWorth`;
  в Go остаётся `money.Minor`. Клиентские модели перегенерируются, диффа в типах быть не должно (`Long`).

## What Goes Where

- **Implementation Steps** — код, тесты, документация в этом репозитории
- **Post-Completion** — релиз, выкат, возврат лимита в проде

## Implementation Steps

### Task 1: Снять проверку с создания и правки операции

**Files:**
- Modify: `internal/services/transaction_service.go`
- Modify: `internal/services/interfaces.go`
- Modify: `internal/application/handlers/transactions.go`
- Modify: `internal/services/transaction_service_test.go`
- Modify: `internal/services/helpers_test.go`
- Modify: `internal/application/http_server_test.go`

- [x] убрать вызов `ValidateTransactionLimits` из `CreateTransaction` и вызов `validateUpdatedTransactionLimits` из правки
- [x] удалить обе функции и `ErrInsufficientBudget`; `findBudgetByCategory` не трогать
- [x] убрать `ValidateTransactionLimits` из интерфейса и из обоих моков — иначе сборка не пройдёт
- [x] убрать ветки `ErrInsufficientBudget` в `handlers/transactions.go` (111, 116, 482)
- [x] переписать тесты на отказ (`transaction_service_test.go:114–141, ≈542`) в тесты «расход сверх лимита сохранён» — создание и правка
- [x] ➕ `tests/integration/transactions_test.go`: `TestTransactionAPI_UpdateCountsAgainstBudgetOnce` ждал `422` на правку сверх лимита — теперь `200` и `spent_minor` = новой сумме
- [x] `make test` — зелёный

### Task 2: Убрать `CheckBudgetLimits` из бюджетного сервиса

**Files:**
- Modify: `internal/services/budget_service.go`
- Modify: `internal/services/interfaces.go`
- Modify: `internal/services/budget_service_test.go`
- Modify: `internal/services/helpers_test.go`
- Modify: `internal/application/http_server_test.go`
- Modify: `internal/infrastructure/budget/budget_repository_sqlite.go`

- [x] удалить `CheckBudgetLimits`, `ErrInsufficientBudgetFunds` и метод из интерфейса
- [x] убрать метод из обоих моков и удалить его тесты
- [x] поправить комментарии, ссылающиеся на проверку лимита: `budget_service.go:286`, `budget_repository_sqlite.go:793`
- [x] `make test`, `make lint` — зелёные
- [x] ➕ удалить `isBudgetActiveOnDate` — без `CheckBudgetLimits` не используется (`unused`)

### Task 3: Снять `BUDGET_BELOW_SPENT` с правки бюджета

**Files:**
- Modify: `internal/services/budget_service.go`
- Modify: `internal/services/dto/budget_dto.go`
- Modify: `internal/application/handlers/budgets.go`
- Modify: `internal/application/handlers/errors.go`
- Modify: `internal/services/budget_service_test.go`
- Modify: `internal/application/handlers/budgets_test.go`

- [x] убрать сравнение `*req.AmountMinor < spent` в `UpdateBudget` (275); `spentFor` и запись `SpentMinor` остаются
- [x] удалить `ErrBudgetAlreadyExceeded` (сервис и `dto/budget_dto.go:19`, если больше не используется), ветку обработчика, `ErrCodeBudgetBelowSpent` и `ErrMessageBudgetBelowSpent`
- [x] переписать тесты на отказ (`budget_service_test.go:363, 758, 806`, `budgets_test.go:239`) в тесты «сумма ниже потраченного сохранена»
- [x] `make test`, `make lint` — зелёные
- [x] ➕ `tests/integration/budgets_test.go:867`: тест на `409` переписан в `TestBudgetAPI_UpdateAmountBelowSpent_Saved` уже здесь — без константы `ErrCodeBudgetBelowSpent` пакет не собирается

### Task 4: Спецификация — `spent_minor` без потолка, без `BUDGET_BELOW_SPENT`

**Files:**
- Modify: `docs/api/openapi.yaml`
- Modify (перегенерация): `android/core/api/generated/`

- [x] `Budget.spent_minor` и `BudgetProgress.spent_minor` — `int64` без `Money.maximum`
- [x] убрать `BUDGET_BELOW_SPENT` из перечней кодов (`openapi.yaml:28, 1073, 1395`)
- [x] описание `utilization` в обеих схемах и абзац про единицы (`openapi.yaml:17`): значение бывает больше 100 / больше 1
- [x] перегенерировать клиентские модели, убедиться, что типы не изменились
- [x] `make test` (тесты покрытия спецификации), `make -C android test` — зелёные

### Task 5: Интеграционные тесты на перерасход

**Files:**
- Modify: `tests/integration/budgets_test.go`

- [x] `POST /transactions` сверх лимита → `201`; `GET /budgets` → точные `spent_minor`, `remaining_minor < 0`, `utilization > 100`
- [x] `PUT /transactions/:id` с суммой сверх лимита → `200`, `spent_minor` бюджета равен новой сумме
- [x] `PUT /budgets/:id` перерасходованного бюджета: только `name` → `200`; только `recurring: false` → `200`; `amount_minor` ниже потраченного → `200` (тест на `409` переписан в задаче 3)
- [x] повторяющийся бюджет с перерасходом: следующий период материализуется, его `spent_minor` считается с нуля
- [x] `GET /stats/summary` → `budgets[].utilization > 1` для этого бюджета
- [x] `make test` — зелёный

### Task 6: Update documentation

- [x] `CLAUDE.md`, «Conventions»: бюджет не ограничивает запись операций, `BUDGET_BELOW_SPENT` убрать из перечня кодов (304); «Fractions come in two units» — значения бывают выше 1 / 100; перечень планов и релизов
- [x] `docs/patterns/api_standards.md:62`: убрать `BUDGET_BELOW_SPENT`
- [x] `docs/patterns/error_handling.md`: убрать `BUSINESS_BUDGET_EXCEEDED`, если код нигде не возвращается
- [x] `docs/product_brief.md:79–83`: лимит — ориентир, запись расхода он не блокирует
- [x] ➕ `README.md:115`: убрать `BUDGET_BELOW_SPENT` из перечня кодов бюджета

### Task 7: [Final] Verify acceptance criteria

- [x] расход сверх лимита создаётся и правится без ошибки; в коде нет `exceed budget limit` и `BUDGET_BELOW_SPENT` (остались намеренно: клиентская строка против старого сервера и упоминание в `CLAUDE.md`)
- [x] `make fmt`, `make test`, `make lint` — 0 issues
- [x] перенести план в `docs/plans/completed/`

## Post-Completion

- тег `v0.10.1`, выкат на mini;
- в проде бюджет «На продукты» 30.09.2026 поднят с 40 000 ₽ до 40 389,68 ₽ ради импорта; серия повторяющаяся —
  проверить сумму октябрьского экземпляра и вернуть 40 000 ₽, если она унаследовала поднятую;
- ручная проверка с телефона: расход сверх лимита сохраняется, бюджет показывает перерасход.
