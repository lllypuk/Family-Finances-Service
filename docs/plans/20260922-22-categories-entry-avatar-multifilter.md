# План 22. Категории: вход без вкладки, аватар со значком и цветом, фильтр по нескольким категориям

## Overview

Категории занимают вкладку нижней панели, хотя правят их раз в месяц; значок и цвет вводятся на форме и
нигде, кроме точки в справочнике, не видны; фильтр операций умеет одну категорию. План:

- вкладка `CATEGORIES` снимается, панель — четыре корня; вход — строка «Категории» в настройках и пункт
  «Управлять категориями…» там, где категорию выбирают;
- значок (эмодзи) и цвет становятся аватаром категории в строках операций, листах выбора, справочнике,
  топе главной и «Обзоре»;
- `GET /transactions` принимает несколько `category_id`, фильтр на «Операциях» — мультивыбор.

Сервер `v0.10.0` (аддитивно), клиент `0.14.0`, один MR. Порядок выката: сначала сервер — старый сервер
берёт только первый `category_id` (`c.QueryParam`) и **молча** сужает фильтр до одной категории.

**Не делаем:** сетку эмодзи, фильтр по категориям в бюджетах и «Обзоре», раскрытие детей родителя в
фильтре, серверное дерево категорий, отдельный экран формы категории (она — состояние
`CategoriesViewModel.editor`, так и остаётся), потолок числа `category_id` (ограничен числом категорий семьи).

## Context (from discovery)

- Навигация: `ui/AppNavBar.kt` — `AppTab` из пяти корней (`CATEGORIES` → `AppIcons.Tag`);
  `AppScreen.kt` — `data object Categories`, `TransactionEdit(back)`, `AppScreenSaver` (`back` —
  последний ключ; `TRANSACTIONS_KEY_FIELDS` = 10, поле 3 — `categoryId`); `MainActivity.kt` (`AppRoot`):
  `AppScreen.Categories` в `when` (235, 450–493 — там же рисуется форма из `model.editor`), FAB только через
  `WithNavBar(fab)` (469–473, 933–946), `AppTab.CATEGORIES → Categories` (963), уход с вкладки взводит
  `listStale`/`homeStale`/`overviewStale`/`budgetsStale` (168–173); **стор форм** `forms` чистится
  `LaunchedEffect(onForm)` при любом экране вне `TransactionEdit|BudgetEdit|HoldingEdit|HoldingHistory|Recognize`
  (183–189) — уход в справочник поверх формы убил бы черновик; обработчик ждущего импорта (220–248) с
  `Categories` уходит в `Recognize`, с форм — `else -> Unit`; `resumable()` (968–975) `Categories` не включает.
- Справочник: `ui/categories/{CategoriesScreen,CategoriesViewModel,CategoryEditScreen,CategoryDot,CategoryColor}.kt`;
  у `CategoriesScreen` нет «назад» (голый `Text` в шапке, 43–50; образец — `ReconciliationScreen.kt:79`);
  значок — свободное текстовое поле (`CategoryEditScreen.kt:131`), серверные дефолты — эмодзи
  (`internal/services/category_service.go:340`), в старых данных бывают `default`/`food`.
- Выбор категории: `ui/transactions/CategorySheet.kt` (`CategorySheet` + `CategorySheetContent`,
  одиночный, выбор через `sheetState.hide()` 38–43) — фильтр `TransactionsScreen.kt:112` и распознавание
  `RecognizeScreen.kt:211`; формы операции и бюджета — `ChipRow` (`TransactionEditScreen.kt:121`,
  `BudgetEditScreen.kt:295–308`; строка 119 там — период), листа нет.
- Загрузка справочника в формах: `TransactionEditViewModel.load()` (126, он же `onRetry`) через `filled()`
  (253–282) **перезаписывает** поля при `existing != null` и сбрасывает `accountId` у новой;
  `RecognizeViewModel.loadCatalogs()` (303–314) тоже сбрасывает `accountId`. Узкого «перечитать только
  категории» нет.
- Строки: `TransactionsScreen.kt:298 TransactionItem` (`TransactionRow.categoryName: String?`,
  `TransactionsViewModel.kt:238`), `HomeScreen.kt:426 CategoryRow(CategoryShare)`,
  `OverviewScreen.kt:192 categories(...)` — без полос, процент текстом; `CategoryShare.color`/`icon`
  необязательны в спеке (2186–2193) и сгенерированы `String?`.
- Фильтр: `ui/transactions/Filters.kt` `TransactionFilters.categoryId: UUID?` (у `reconciliation()`
  категории нет); сгенерированный `listTransactions(categoryId: UUID?)`, генератор `jvm-retrofit2` с
  `CollectionFormats.kt` уже подключён (`core/api/build.gradle.kts:86–110`); спека `openapi.yaml:778`;
  сервер — `handlers/transactions.go:224` (`c.QueryParam`), `buildTransactionServiceFilter` (346–363) →
  `dto.TransactionFilterDTO` → `convertDTOFilterToRepoFilter` (`transaction_service.go:845–852`) →
  `transaction.Filter.CategoryID *uuid.UUID` (`domain/transaction/transaction.go:38`) → SQL `category_id = ?`
  (`transaction_repository_sqlite.go:391–396`); прецедент `IN (?, …)` с `//nolint:gosec` — там же 648–659;
  `transaction_service.go:479` (`GetTransactionsByCategory`, зовётся только из моков);
  `budget_service.go:536` — чужой фильтр. Тест хендлера мокает сервис (`transactions_test.go:507`),
  сквозного теста фильтра нет. Переименование ломает `domain/transaction/transaction_test.go:148,162` и
  `services/dto/transaction_dto_test.go:86,99`.
- Настройки: `ui/settings/SettingsRootScreen.kt` (`SettingsItem`), `SettingsHost` отдаёт наружу
  `onLeave`/`onSessionChanged`/`onUsersChanged`/`onSignedOut` — навигации за пределы хоста нет.
- Тесты: `AppScreenSaverTest`, `AppRootOverviewTest`, `ui/AppNavBarTest`, `ui/transactions/*Test`
  (в т.ч. `TransactionEditScreenTest`), `ui/categories/*Test`, `ui/home/*Test`, `ui/overview/*Test`,
  `ui/recognize/RecognizeScreenTest`, `ui/settings/SettingsRootScreenTest`; серверные —
  `handlers/transactions_test.go`, `infrastructure/transaction/transaction_repository_test.go`,
  `tests/integration/transactions_test.go`; `openapi_coverage_test.go` параметры не проверяет;
  `make -C android api-check` требует перегенерации клиента в том же MR.

## Development Approach

- **testing approach**: Regular.
- задача закрывается целиком, включая тесты; сервер — `make fmt && make test && make lint` (0 issues),
  клиент — `make -C android check`.
- **CRITICAL: в каждой задаче с кодом есть тесты**, успех и отказ (Task 7 — только номер версии).
- **CRITICAL: план обновляется при смене объёма.**

## Testing Strategy

- **сервер**: хендлер (один/два/дубль/невалидный `category_id`), репозиторий (`IN`, пустой срез),
  интеграционный тест сквозь сервис.
- **клиент**: unit — графема аватара, `TransactionFilters` ↔ запрос ↔ `Saver`, подпись чипа,
  `reloadCategories` не трогает поля; Compose — лист в мультирежиме, строки с аватаром, вход из настроек,
  возврат из `Categories(back)` в форму с живым черновиком.
- e2e нет; телефон — Post-Completion.

## Progress Tracking

- `[x]` сразу, ➕ — найденные задачи, ⚠️ — блокеры.

## Solution Overview

1. **Вход.** `AppScreen.Categories(back: AppScreen = Settings())`. `AppTab` — четыре. Строка «Категории» в
   корне настроек рядом со «Счетами» → `SettingsHost.onOpenCategories` → `Categories(back = Settings())`.
   Пункт «Управлять категориями…» — внизу `CategorySheetContent` (фильтр «Операций», распознавание; лист
   сперва `hide()`, как при выборе) и текстовой кнопкой у заголовка «Категория» на формах операции и бюджета
   (там `ChipRow`) → `Categories(back = текущий экран)`. «Назад» из справочника — в `back` плюс четыре флага,
   что взводил уход с вкладки, плюс новый `categoriesStale`.
   **Справочник поверх формы не убивает черновик:** `onForm` в `AppRoot` расширяется на
   `Categories`, чей `back` — форма (`TransactionEdit`/`BudgetEdit`/`Recognize`), и обработчик ждущего импорта
   в этом случае ведёт себя как на форме (`Unit`). Форма, поймав `categoriesStale`, зовёт **узкий**
   `reloadCategories()`, который меняет только `categories` — `load()`/`loadCatalogs()` не годятся, они
   затирают поля.
2. **Аватар.** `CategoryAvatar(icon, color, name, size)` — единственное место, где значок и цвет
   становятся картинкой; `CategoryDot` удаляется. Цвет остаётся на данных, акцент интерфейса — `action`.
3. **Мультифильтр.** Контракт аддитивен: массив `category_id` с `explode: true` — один id на проводе не
   отличим от прежнего. Клиент держит `Set<UUID>`, в запрос — в порядке справочника; лист фильтра —
   чекбоксы с «Готово».

## Technical Details

### Аватар

```kotlin
// ui/categories/CategoryAvatar.kt
@Composable fun CategoryAvatar(icon: String, color: String, name: String, size: Dp, modifier: Modifier = Modifier)
fun categoryGlyph(icon: String, name: String): String
```

- круг `size`, заливка `tint.copy(alpha = 0.18f)`, кольцо `1.dp` цветом `tint`; `tint =
  parseCategoryColor(color, outline)`; `contentDescription = name`;
- `categoryGlyph`: первый графемный кластер `icon.trim()` через **`android.icu.text.BreakIterator`**
  (`java.text` не знает GB11 и режет ZWJ; в Robolectric `android.icu` — настоящая ICU4J). Кластер —
  эмодзи, если его длина в `Char` > 1 (`1️⃣`, ZWJ-семьи, модификаторы кожи) или первый code point не
  ASCII-буква/цифра; иначе (пусто, `default`, `food`) — первый символ `name` в верхнем регистре цветом `tint`;
- `Dimens.AVATAR_S = 32.dp` (строки операций, топ главной, «Обзор»), `AVATAR_M = 36.dp` (листы,
  справочник — `ROW_COMPACT` = 40dp, 40dp аватар не оставил бы воздуха); текст `titleMedium`;
- `CategoryShare.icon/color` — `orEmpty()`;
- `TransactionRow.categoryName: String?` → `category: Category?`; без категории — аватар с `?` цветом
  `outline`; строка-кандидат распознавания — аватар выбранной категории; чипы форм операции и бюджета —
  префикс `"$glyph $name"`;
- форма категории: поле значка обрезает ввод до одного кластера (`categoryGlyph`-срез в `onIconChange`),
  рядом — превью `CategoryAvatar(AVATAR_M)`.

### Навигация

```kotlin
data class Categories(val back: AppScreen = Settings()) : AppScreen
```

`AppScreenSaver`: `KEY_CATEGORIES` + `back` последним (его ключ содержит двоеточия); старый бандл
`Categories` без `back` читается как `Settings()`. `AppRoot`: `Categories` без `WithNavBar` —
`CategoriesScreen(onBack)` получает шапку с `ArrowLeft` (образец `ReconciliationScreen`) и свой FAB
(`Scaffold.floatingActionButton`), `BackHandler` → `back` + `listStale`/`homeStale`/`overviewStale`/
`budgetsStale`/`categoriesStale`. `onForm` = прежние формы **или** `Categories` с `back` формой; ждущий
импорт при этом не перехватывает экран. Ветка формы гасит `categoriesStale` и зовёт
`TransactionEditViewModel.reloadCategories()` / `BudgetEditViewModel.reloadCategories()` /
`RecognizeViewModel.reloadCategories()` — новые методы, пишущие только `categories` в состояние.

### Мультифильтр

Спека:

```yaml
- name: category_id
  in: query
  description: Несколько значений — операции любой из категорий
  style: form
  explode: true
  schema: { type: array, items: { type: string, format: uuid } }
```

Сервер: `handlers/transactions.go` — `for _, raw := range c.QueryParams()["category_id"]` → UUID или
`writeInvalidQueryParam`; дубли схлопываются. `dto.TransactionFilterDTO.CategoryIDs []uuid.UUID`
`validate:"omitempty,dive,required"` и `transaction.Filter.CategoryIDs`; `GetTransactionsByCategory`
кладёт `[]uuid.UUID{id}`; репозиторий — `category_id IN (?, …)` (`ValidateUUID` на каждом, плейсхолдеры
как в 648–659); пустой срез — без условия.

Клиент: `make -C android api-gen` → `listTransactions(categoryId: List<UUID>?)`.
`TransactionFilters.categoryIds: Set<UUID> = emptySet()`; в запрос — отсортированные по порядку справочника
(`Set` хранит порядок вставки, тесты с `MockWebServer` сравнивают query). `Saver`: поле 3 — id через `,`,
`TRANSACTIONS_KEY_FIELDS` остаётся 10, старый бандл с одним uuid читается как набор из одного.
`Overview` → `setOf(id)`. `CategorySheetContent(multi, selectedIds, onDone)`: `Checkbox` в строках, «Все»
очищает, «Готово» закреплена снизу (лист с 20+ категориями прокручивается) и отдаёт набор; одиночный
режим (распознавание) не меняется. Чип: пусто — «Все категории», одна — `path`, больше — «Продукты +2».

## Implementation Steps

### Task 1: Сервер — массив `category_id` в `GET /transactions`

**Files:**
- Modify: `docs/api/openapi.yaml`
- Modify: `internal/application/handlers/transactions.go`
- Modify: `internal/services/dto/transaction_dto.go`, `internal/services/transaction_service.go`
- Modify: `internal/domain/transaction/transaction.go`
- Modify: `internal/infrastructure/transaction/transaction_repository_sqlite.go`
- Modify: `internal/application/handlers/transactions_test.go`
- Modify: `internal/domain/transaction/transaction_test.go`, `internal/services/dto/transaction_dto_test.go`
- Modify: `internal/infrastructure/transaction/transaction_repository_test.go`
- Modify: `tests/integration/transactions_test.go`

- [x] спека по «Мультифильтр»
- [x] `CategoryIDs []uuid.UUID` в домене и dto (с тегом `validate`); `GetTransactionsByCategory`,
      `buildTransactionServiceFilter`, `convertDTOFilterToRepoFilter` переведены
- [x] хендлер: `QueryParams()["category_id"]`, дубли схлопнуты
- [x] репозиторий: `category_id IN (…)`, пустой срез — без условия
- [x] тесты хендлера: один id (как раньше), два, дубль → один, невалидный (прежний код ответа)
- [x] тесты репозитория: `IN` по двум категориям возвращает обе и только их; пустой срез = все
- [x] интеграционный: два `category_id` сквозь сервис → операции обеих категорий и только они
- [x] `make fmt && make test && make lint` — 0 issues

### Task 2: Клиент — фильтр `categoryIds` и мультивыбор в листе

**Files:**
- Modify: `android/core/api/generated/**` (через `make -C android api-gen`)
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/{Filters,TransactionsViewModel,TransactionsScreen,CategorySheet}.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/AppScreen.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/{TransactionsViewModelTest,TransactionsScreenTest}.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/AppScreenSaverTest.kt`

- [x] `make -C android api-gen`; `git status` по каталогу генерации — только `listTransactions`
- [x] `TransactionFilters.categoryIds: Set<UUID>`; `Overview` → `setOf(id)`; `Saver` поле 3 через `,`
- [x] `CategorySheetContent(multi)`: чекбоксы, «Все», закреплённая «Готово»; фильтр — мультирежим,
      распознавание — одиночный
- [x] подпись чипа: «Все категории» / `path` / «Имя +N»
- [x] `TransactionsViewModelTest`: два id уходят двумя `category_id` в порядке справочника; пустой набор — без
      параметра
- [x] `TransactionsScreenTest`: отметить две → «Готово» → `onFiltersChange` с обоими; «Все» сбрасывает
- [x] `AppScreenSaverTest`: набор id туда и обратно, пустой набор, старая строка с одним uuid
- [x] `make -C android check` зелёный

### Task 3: `CategoryAvatar` и поле значка на форме

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/categories/CategoryAvatar.kt`
- Delete: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/categories/CategoryDot.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/theme/Dimens.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/categories/{CategoriesScreen,CategoryEditScreen,CategoriesViewModel}.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/categories/CategoryAvatarTest.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/categories/{CategoriesScreenTest,CategoriesViewModelTest}.kt`

- [x] `categoryGlyph` + `CategoryAvatar` по «Аватар»; `Dimens.AVATAR_S/M`
- [x] справочник: `CategoryAvatar(AVATAR_M)` вместо `CategoryDot`; `CategoryDot.kt` удалён
- [x] форма: ввод значка обрезается до одного кластера, превью аватара рядом с полем
- [x] `CategoryAvatarTest`: `🛒` → `🛒`; `👨‍👩‍👧` и `1️⃣` целиком; `default`/`food`/пусто → `П` для «Продукты»;
      `ё` → `Ё`; непарсибельный цвет не падает
- [x] `CategoriesViewModelTest`: `onIconChange("🛒🚗")` оставляет `🛒`
- [x] `CategoriesScreenTest`: аватар с `contentDescription` имени в строке
- [x] `make -C android check` зелёный

### Task 4: Аватар в строках операций, распознавании, главной и «Обзоре»

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/{TransactionsViewModel,TransactionsScreen,CategorySheet,TransactionEditScreen}.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/recognize/RecognizeScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/overview/OverviewScreen.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/{transactions/TransactionsScreenTest,transactions/TransactionsViewModelTest,home/HomeScreenTest,overview/OverviewScreenTest,recognize/RecognizeScreenTest}.kt`

- [x] `TransactionRow.category: Category?`; `TransactionItem` — `AVATAR_S` слева, без категории — `?` цветом `outline`
- [x] `CategorySheetContent` — `AVATAR_M`; чипы форм операции и бюджета — префикс-эмодзи (`categoryChipLabel`; буква-заглушка в чипе повторила бы имя — её нет)
- [x] `RecognizeScreen`: аватар у строки-кандидата
- [x] `HomeScreen.CategoryRow`, `OverviewScreen.categories` — `AVATAR_S` из `CategoryShare.icon/color` с `orEmpty()`
- [x] экранные тесты: аватар в строке операции, в топе главной (и без `icon`/`color` в ответе), в «Обзоре»,
      в кандидате; `TransactionsViewModelTest` — `category` в строке
- [x] `make -C android check` зелёный

### Task 5: Вкладка снята, `Categories(back)`, вход из настроек

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppNavBar.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/AppScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt` (`AppRoot`)
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/categories/CategoriesScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/{SettingsRootScreen,SettingsHost}.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/{AppScreenSaverTest,AppRootOverviewTest}.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/{AppNavBarTest,categories/CategoriesScreenTest,settings/SettingsRootScreenTest}.kt`

- [x] `AppTab` без `CATEGORIES`; `AppScreen.Categories(back)`; `Saver` с `back` последним, старый бандл → `Settings()`
- [x] `CategoriesScreen(onBack)`: шапка с `ArrowLeft`, FAB внутри экрана
- [x] `AppRoot`: `Categories` без панели, «назад» → `back` + `listStale`/`homeStale`/`overviewStale`/
      `budgetsStale`/`categoriesStale`; `AppTab.CATEGORIES`-ветки удалены
- [x] `SettingsRootScreen`: строка «Категории» после «Счетов»; `SettingsHost.onOpenCategories`
- [x] `AppNavBarTest`: четыре вкладки; `AppScreenSaverTest`: `Categories(back = Settings())` и с `back =
      TransactionEdit(...)` туда-обратно; `CategoriesScreenTest`: «назад» и FAB
- [x] новый `AppRootCategoriesTest`: настройки → справочник → «назад» → настройки,
      флаги взведены
- [x] `SettingsRootScreenTest`: тап «Категории» → `onOpenCategories`
- [x] `make -C android check` зелёный

### Task 6: «Управлять категориями…» из выбора и перечитка справочника

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt` (`AppRoot`)
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/{CategorySheet,TransactionsScreen,TransactionEditScreen,TransactionEditViewModel}.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/{BudgetEditScreen,BudgetEditViewModel}.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/recognize/{RecognizeScreen,RecognizeViewModel}.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/{TransactionsScreenTest,TransactionEditScreenTest,TransactionEditViewModelTest}.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/{BudgetEditScreenTest,BudgetEditViewModelTest}.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/recognize/{RecognizeScreenTest,RecognizeViewModelTest}.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/AppRootOverviewTest.kt` (или `AppRootCategoriesTest`)

- [x] пункт внизу `CategorySheetContent` (фильтр, распознавание) — сперва `hide()`, потом навигация;
      текстовая кнопка у «Категория» на формах операции и бюджета → `Categories(back = текущий экран)`
- [x] `AppRoot`: `onForm` включает `Categories` с `back`-формой; ждущий импорт при этом — `Unit`;
      ветка формы гасит `categoriesStale` и зовёт `reloadCategories()`
- [x] `reloadCategories()` в трёх моделях — только `categories`; платный вызов распознавания не повторяется
- [x] `*ViewModelTest`: после `reloadCategories()` введённые сумма/описание/счёт на месте, список категорий
      новый; отказ сети оставляет прежний список
- [x] `*ScreenTest`: пункт/кнопка зовут навигацию; лист скрыт до перехода
- [x] `AppRoot*Test`: форма с введённой суммой → справочник → «назад» → та же модель, сумма на месте,
      `reloadCategories` вызван; share во время справочника поверх формы не открывает `Recognize`
- [x] `make -C android check` зелёный

### Task 7: Версия клиента

**Files:**
- Modify: `android/gradle/libs.versions.toml`

- [x] `appVersionCode = 14`, `appVersionName = "0.14.0"` (тестов нет — только номер)
- [x] `make -C android apk` собирается

### Task 8: Verify acceptance criteria
- [x] панель — четыре вкладки; категории открываются из настроек и из «Управлять…», «назад» возвращает туда,
      откуда пришли, черновик формы цел, новая категория видна в листе/чипах без перезапуска
- [x] аватар: эмодзи и цвет видны в операциях, листах, справочнике, главной, «Обзоре»; `default`/`food` — буква
- [x] фильтр по двум категориям показывает операции обеих; чип «Имя +1»; переживает поворот
- [x] `make fmt && make test && make lint` (0 issues), `make -C android check`, `make -C android api-check`

### Task 9: [Final] Update documentation
- [ ] `android/CLAUDE.md`: «Модули» (четыре корня, `Categories(back)`, `onForm` со справочником поверх формы,
      `categoriesStale` и узкий `reloadCategories`), «Общие элементы» (`CategoryAvatar` вместо `CategoryDot`,
      `android.icu` для графем), «Версии» (клиент `0.14.0` требует `v0.10.0` для мультифильтра — старый
      сервер молча берёт первый `category_id`)
- [ ] корневой `CLAUDE.md`: `category_id` как массив в «One HTTP surface», теги `v0.10.0`/`app-v0.14.0` в
      «Current direction»
- [ ] `docs/api/README.md`, если там перечислены фильтры
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Выкат**: мерж → тег `v0.10.0` (сервер выкатится сам, `/health` → `version`), затем тег `app-v0.14.0`,
APK на оба телефона. Клиент `0.13.0` с `v0.10.0` работает без изменений.

**Ручная проверка на телефоне**: эмодзи с модификаторами кожи и ZWJ в аватаре 32dp; чипы форм с префиксом на
360dp; лист фильтра с 20+ категориями — «Готово» видна без прокрутки.

**Figma** (`VsFUJiZcr2OgyihLOiM8cW`): `NavBar` — четыре вкладки, компонент `CategoryAvatar`, `TransactionRow`
с аватаром — через локальный dev-плагин (лимит MCP исчерпан).
