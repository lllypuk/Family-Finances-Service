# План 07 — Бюджеты в Android-клиенте

Продолжение [плана 06](completed/20260907-06-android-client.md): из «следующего плана» берутся
только бюджеты. Отчёты, профиль, пользователи, сессии и бэкапы — в [бэклоге](../backlog.md).

## Overview

- Четвёртая вкладка «Бюджеты»: список с прогрессом, форма создания, правка, удаление.
  Всё, что даёт контракт `/api/v1/budgets` — кроме `is_active` (см. «Решения»).
- Главная уже рисует `StatsSummary.budgets[]`; вкладка нужна, потому что бюджеты смотрят и без
  операций, а секция на главной пропадает, когда бюджетов нет.
- Контракт и сервер не меняются: перегенерации клиента и окна несовместимости APK нет.

## Context (from discovery)

- Корень навигации: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
  (`AppRoot`, флаги `listStale`/`homeStale`, `forms: FormModels`, `onForm`), `AppScreen.kt`
  (`AppScreenSaver`), `ui/AppNavBar.kt` (`AppTab`), `ui/AppIcons.kt`.
- Образцы: форма — `ui/transactions/TransactionEditViewModel.kt` + `TransactionEditScreen.kt`
  (`draft`-UUID в ключе экрана, `done`, `fieldErrors`, `DatePickerSheet`); список с одной
  страницей — `ui/categories/CategoriesViewModel.kt` (`limit = 200`); строка прогресса —
  `ui/home/HomeScreen.kt:BudgetRow`.
- Клиент уже сгенерирован: `core/api/generated/kotlin/.../BudgetsApi.kt`, `Budget.kt`,
  `BudgetPeriod.kt`, `CreateBudgetRequest.kt`, `UpdateBudgetRequest.kt`. В `ApiGraph` нет поля
  `budgets`.
- Тесты: Robolectric + `MockWebServer`, фикстуры в `src/test/.../TestFixtures.kt`
  (`enqueueJson`, `liveToken`, `FakeTokenVault`); Compose-тесты для экранов с вводом.

Сервер (проверено 11.09.2026, вместе с codex):

- `listBudgets` без `active_only` и с `active_only=false` отдаёт только `is_active = 1`
  (`internal/infrastructure/budget/budget_repository_sqlite.go:221`); `active_only=true` — ещё и
  «сегодня в зоне семьи» внутри дат (`handlers/budgets.go:271`). `DELETE` — тот же `is_active = 0`
  (`budget_repository_sqlite.go:587`). Выключенный через `PUT` бюджет исчезает из всех списков.
- `period` к датам не привязан: проверяется только `end_date > start_date`
  (`services/dto/budget_dto.go:170`, `services/budget_service.go:287`).
- `PUT` сначала пересчитывает `spent` по старым датам, потом проверяет новую сумму против него
  (`budget_service.go:233`, `:266`): переотправка неизменённой суммы перерасходованного бюджета
  даёт 422.
- Бизнес-отказы (пересечение периодов, имя занято, сумма меньше потраченного) — `422
  VALIDATION_ERROR` с `details[{field: "body"}]` и тем же кодом (`handlers/budgets.go:355`,
  `helpers.go:139`): по-русски их не различить.
- `Budget.utilization` — проценты 0…100, `BudgetProgress.utilization` на главной — доля 0…1
  (`docs/api/openapi.yaml`).
- `spent_minor` считается только по расходным операциям и только по точному `category_id`, без
  потомков (`budget_service.go:552`, `transaction_repository_sqlite.go:781`).
- Ответ `POST` приходит со `spent_minor = 0`, ответ `PUT` — с расходом по старым датам: после
  сохранения список перечитывается.

## Решения

- **`is_active` клиент не трогает.** Выключить — значит потерять бюджет без пути назад; на сервере
  это либо баг контракта, либо недоделанный фильтр — пункт в бэклоге. Отключённые сервером
  бюджеты клиенту не приходят, поэтому строка «неактивен» не нужна.
- **Форма — отдельный экран** `AppScreen.BudgetEdit(id, draft)` со своей `BudgetEditViewModel`
  (решение владельца 11.09.2026): черновик с UUID в ключе экрана, как у транзакций.
- **Даты видны обе и правятся обе.** Пресет периода только подставляет конец при создании:
  `weekly` = начало + 6 дней, `monthly` = + 1 месяц − 1 день, `yearly` = + 1 год − 1 день,
  `custom` — не трогает. При правке период только для чтения и конец не пересчитывается.
- **`PUT` шлёт только изменённые поля**, `UpdateBudgetRequest` собирается из diff с загруженным
  бюджетом; без изменений кнопка «Сохранить» выключена (`minProperties: 1`). Незаполненные поля
  не уходят благодаря `explicitNulls = false` в `core/api/.../net/ApiClient.kt:32` — иначе ушёл
  бы и `is_active: null`. Период и категория при правке только для чтения не по выбору UX, а
  потому что их нет в `UpdateBudgetRequest`.
- **Бизнес-422 без поля** показывается одним русским текстом
  (`budget_error_rejected`: «Не удалось сохранить: название занято, даты пересекаются с другим
  бюджетом или сумма меньше уже потраченного»). Для этого в `UiError` добавляется вариант с
  ресурсом строки.
- **Категория — только расходная и без вложенности в выборе:** список показывает все категории
  типа `expense` (родители и дети), потому что `spent` считается по точному `category_id`.
  Пункт «Все категории» = `category_id` не передаётся. У существующего бюджета категория
  показывается как есть, даже если она доходная или удалена.
- **Фильтр списка:** чипы «На сегодня» (`active_only=true`, по умолчанию) и «Все периоды».
  Пустой список «на сегодня» подсказывает переключить фильтр: будущий бюджет после создания
  иначе выглядит потерянным.
- **Одна страница** (`limit = 200`), как у категорий: бюджетов у семьи единицы.
- **Устаревание:** третий флаг `budgetsStale` рядом с `listStale`/`homeStale`. Его взводят
  форма операции (`spent`), уход с категорий (названия) и форма бюджета; форма бюджета взводит
  ещё `homeStale`.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче; конвенция клиента —
  Compose-тесты для экранов с вводом, ViewModel-тесты через `MockWebServer` для остального.
- Каждая задача закрывается зелёным `make -C android check` до следующей.
- Правки в план — сразу при отклонении от объёма.

## Testing Strategy

- ViewModel: `BudgetsViewModelTest`, `BudgetEditViewModelTest` — `MockWebServer`, проверка тела и
  query запросов через `RecordedRequest`.
- Compose (Robolectric): `BudgetsScreenTest` (пустой список, строка, фильтр, тап), 
  `BudgetEditScreenTest` (пресет периода подставляет конец, кнопка выключена без изменений,
  ошибка под полем).
- Чистые функции дат (`endOf`) — отдельный JVM-тест без Robolectric.
- e2e нет: живого сервера в тестах нет (android/CLAUDE.md).

## Progress Tracking

- `[x]` — сделано; `➕` — найдено по ходу; `⚠️` — блокер.

## Implementation Steps

### Task 1: Точки входа — API, экраны, вкладка, строки

**Files:**
- Modify: `android/core/api/src/main/kotlin/tech/shatrov/familyfinances/core/api/ApiGraph.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/AppScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppNavBar.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppIcons.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/UiError.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/TestFixtures.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/AppScreenSaverTest.kt`

- [x] `ApiGraph.budgets: BudgetsApi` через `client.create`, как соседние
- [x] `AppScreen.Budgets` и `AppScreen.BudgetEdit(id: UUID?, draft: UUID)`; ключи в `AppScreenSaver`
      по образцу `TransactionEdit`
- [x] `AppTab.BUDGETS` четвёртой вкладкой с новой иконкой в `AppIcons` (тем же `icon(name, path)`);
      KDoc «Три корня приложения» в `AppNavBar.kt` → четыре
- [x] оба `when` в `MainActivity.kt` (`AppRoot` по экрану и `AppTab.screen`) — неисчерпываемый
      `when` в Kotlin 2.3 не компилируется: временные ветки `BUDGETS -> AppScreen.Budgets`,
      `Budgets`/`BudgetEdit` → заглушка `Centered { Text(...) }`, которую заменит Task 6
- [x] `UiError.Resource(@StringRes id)` и его ветка в `message(res)`
- [x] строки: `budgets_title`, `budgets_add`, `budgets_empty_today`, `budgets_empty`,
      `budgets_filter_today`, `budgets_filter_all`, `budgets_all_categories`,
      `budget_new_title`, `budget_edit_title`, `budget_name`, `budget_amount`, `budget_period`,
      `budget_period_weekly|monthly|yearly|custom`, `budget_category`, `budget_start`, `budget_end`,
      `budget_save`, `budget_delete`, `budget_delete_confirm`, `budget_error_rejected`,
      `budget_progress` («%1$s из %2$s», без общего с главной — тексты разойдутся)
- [x] фикстуры: `BUDGET_OK`, `BUDGETS_LIST` (два бюджета: с категорией и общий, один перерасходован)
- [x] `AppScreenSaverTest`: save → restore для `Budgets` и `BudgetEdit` (с `id` и без) даёт тот
      же экран; тестов на `AppScreenSaver` в репозитории до этого не было
- [x] `make -C android check` — зелёный

➕ Панели вкладок (`ui/AppNavBar.kt`, `ui/AppIcons.kt`) на ветке не было: они жили в неслитой
  `android-bottom-nav`, на которую опирался раздел «Context». Ветка влита в эту перед задачей.

### Task 2: BudgetsViewModel — список с фильтром

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsViewModel.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsViewModelTest.kt`

- [x] `BudgetsUiState`: `Loading`, `Failure(error)`, `Ready(rows, filter)`; `BudgetRow(budget,
      categoryName: String?, level: BudgetLevel)` — `level ∈ OK | NEAR | OVER` по сырому
      `utilization` (проценты): `OVER` при > 100, `NEAR` при ≥ `NEAR_LIMIT = 80.0` — дубль
      серверного `BudgetAlertNearLimit` (`dto/budget_dto.go:131`), у `Budget` флагов
      `is_over_budget`/`is_near_limit` нет, в отличие от `BudgetProgress` на главной
- [x] `refresh()`: `listBudgets(limit = 200, activeOnly = if (filter == TODAY) true else null)` и
      `listCategories(limit = 200)` для имён; ошибка любого из двух — `Failure`
- [x] `onFilterChange(filter)` перечитывает список; фильтр переживает поворот (в состоянии модели)
- [x] `revalidate()` по образцу `HomeViewModel.revalidate` (`ui/home/HomeViewModel.kt:58`):
      «сегодня» считает сервер, и ответ на `active_only=true`, полученный вчера, после возврата
      из фона устарел — модель запоминает день запроса в зоне семьи и перечитывает при смене
- [x] тесты: сегодня → `active_only=true` в query, все → без параметра; имя категории и «Все
      категории»; `level` для 150 % = `OVER`, для 85 % = `NEAR`; `revalidate` в тот же день не
      шлёт запрос; сеть → `Failure(Network)`
- [x] `make -C android check` — зелёный

### Task 3: BudgetsScreen — список

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreen.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreenTest.kt`

- [x] шапка с заголовком и `Plus`, чипы фильтра из `ui/Chips.kt`, `LazyColumn` строк:
      название, категория или «Все категории», `formatPeriod(start, end)`, «потрачено из лимита»,
      `LinearProgressIndicator` (`utilization / 100`, обрезано в 0…1) с цветами как у
      `HomeScreen.BudgetRow` по `level` (`OVER` — `expense`, `NEAR` — `warning`)
- [x] пустое состояние: на «сегодня» — текст с подсказкой про «Все периоды»; на «все» — просто
      «Бюджетов пока нет»; `Failure` — текст и «Повторить»
- [x] Compose-тесты: пустой список, строка показывает имя/период/суммы, тап по чипу и по строке
      доходят до колбэков
- [x] `make -C android check` — зелёный

### Task 4: Даты периода и BudgetEditViewModel

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetDates.kt`
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModel.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetDatesTest.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModelTest.kt`

- [x] `endOf(period, start): LocalDate?` — `custom` даёт `null`; 30 сентября monthly → 29 октября,
      31 декабря yearly → 30 декабря следующего года (тест на оба)
- [x] `BudgetEditUiState`: `name`, `amount` (строка, `parseAmountMinor`), `period`, `categoryId`,
      `start`, `end`, `categories` (только `expense`), `loaded: Budget?`, `editing`, `loading`,
      `submitting`, `error`, `fieldErrors`, `done`; `canSubmit` = имя ≥ 2, сумма > 0,
      `end > start`, не отправляется, а при правке — ещё и есть diff
- [x] `load()`: категории, при правке — `getBudget(id)`; создание: `start = today`,
      `period = monthly`, `end = endOf(monthly, today)`
- [x] `onPeriodChange`/`onStartChange` при создании подставляют `end` через `endOf`, если период
      не `custom`; при правке — только меняют своё поле; `onEndChange` не трогает период
- [x] `onSubmit`: `POST` c `id = draft`; `PUT` — `UpdateBudgetRequest` только из изменённых полей;
      после успеха `done = true`
- [x] `onDelete`: `deleteBudget(id)` → `done`
- [x] ошибки: `details` с полями формы (`name`, `amount_minor`, `start_date`, `end_date`,
      а на `POST` ещё `category_id`, `period`) — под поля; всё остальное с `field = "body"` —
      `UiError.Resource(R.string.budget_error_rejected)`; прочее — `toUiError()`
- [x] тесты: тело `POST` содержит `id` черновика и не содержит `category_id` при «все категории»;
      `PUT` после смены только имени — тело из одного поля; перерасходованный бюджет с
      неизменённой суммой сохраняет имя без 422; 422 с `field=body` → `Resource`; 422 с
      `field=name` → `fieldErrors["name"]`; удаление → `done`
- [x] `make -C android check` — зелёный

### Task 5: BudgetEditScreen — форма

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreen.kt`
  (вынести `DatePickerSheet` в `ui/DatePickerSheet.kt`, если он `private`)
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreenTest.kt`

- [ ] поля: название, сумма, период (чипы; при правке — текст без выбора), категория (выпадающий
      список как у транзакции, первым «Все категории»; при правке — текст), начало и конец —
      две кнопки с датой и общим `DatePickerSheet`
- [ ] кнопка «Сохранить» по `canSubmit`; «Удалить» с подтверждением только при правке
      (`AlertDialog`, как у транзакции); ошибка формы над кнопкой, ошибки полей под полями
- [ ] Compose-тесты: выбор `weekly` меняет текст конца; при правке кнопка выключена, пока поле не
      изменено; `fieldErrors["name"]` виден под полем; тап «Удалить» → подтверждение → колбэк
- [ ] `make -C android check` — зелёный

### Task 6: Подключение в AppRoot и устаревание

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`

- [ ] `budgetsStale` рядом с `listStale`/`homeStale`; `onForm` учитывает `BudgetEdit`
- [ ] `AppScreen.Budgets` вместо заглушки из Task 1: модель с ключом `budgets-${user.id}-$epoch`,
      `LifecycleResumeEffect` по `budgetsStale` — `refresh`, иначе `revalidate`, как у главной и
      операций; `BackHandler` на главную; `WithNavBar(AppTab.BUDGETS)`; `onCreate`/`onOpen` →
      `BudgetEdit`
- [ ] `AppScreen.BudgetEdit`: модель в `forms` с ключом `budget-${draft}`; `done` взводит
      `budgetsStale` и `homeStale` и ведёт на `Budgets`; уход во время `submitting` запрещён
- [ ] `TransactionEdit.done` и уход с категорий взводят `budgetsStale`
- [ ] тестов на `AppRoot` нет и стенда для него тоже (`MainActivityTest` — один дымовой тест);
      навигация проверяется руками в Task 7, новый стенд ради одной вкладки не строится
- [ ] `make -C android check` — зелёный

### Task 7: Verify acceptance criteria

- [ ] сценарий руками на debug-сборке против `ffs.shatrov.tech`: создать месячный бюджет с
      категорией, увидеть его на вкладке и на главной, добавить операцию в категорию — прогресс
      сдвинулся, переименовать, удалить (тестовые данные убрать руками — базы для отладки нет)
- [ ] будущий бюджет: создан → на «сегодня» пуст, подсказка ведёт на «Все периоды»
- [ ] `make -C android check` и `make -C android compile` — зелёные; `make -C android api-check`
      не нужен (контракт не менялся)

### Task 8: [Final] Update documentation

- [ ] `android/CLAUDE.md`: «четыре корня» вместо трёх, `budgetsStale`, правило «PUT из diff» и
      почему `is_active` не показывается
- [ ] `android/gradle/libs.versions.toml`: `appVersionCode` 1 → 2, `appVersionName` 0.1.0 → 0.2.0 —
      равный `versionCode` система ставит молча, и телефон остаётся на старой сборке
- [ ] `docs/backlog.md`: серверные пункты из этого плана уже добавлены при его создании —
      сверить, что ничего не закрылось по ходу
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Ручная проверка:** оба телефона после установки APK — бюджеты видны member-у, удаление
доступно обоим (контракт: admin и member).

**Figma** (`figma-app-design` в памяти): в файле есть `BudgetRow`/`ProgressBar`, экрана
«Бюджеты» и формы нет — дорисовать при следующем заходе в Figma, лимит MCP-вызовов на месяц
исчерпан 09.09.2026.

**Сервер (в бэклоге, не в этом плане):** семантика `is_active` в `listBudgets`/`PUT`,
различимые коды бизнес-422 для бюджетов, граничный день в двух бюджетах одной категории.
