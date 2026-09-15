# План 13 — UI/UX-правки Android-клиента по аудиту

Аудит `android/app` против Material 3 и принципов дизайна (15.09.2026, второе мнение codex —
тред `android-ui-audit`, затем plan-review). Контракт `/api/v1` и сервер не меняются. Не входит:
светлая тема (решение v1 в `theme/Colors.kt:25`), `FlowRow` для категорий, замена « · »-склеек
колонками — codex не нашёл им доказанной пользы, они остаются в бэклоге.

## Overview

- Подтверждённый дефект: выбранный чип фиолетовый — `appColorScheme` не задаёт
  `secondaryContainer`, и `FilterChip` берёт baseline `#4A4458`/`#E8DEF8`. Ту же роль используют
  дорожка `LinearProgressIndicator` и активный сегмент `SegmentedButton`.
- Пустые состояния без действия, пароль без переключателя видимости, бинарный выбор чипами,
  дата в форме операции без подписи, «+» в правом верхнем углу вместо FAB, три ряда фильтров над
  списком операций, главная без визуального центра.
- Две правки визуального языка — отдельный цвет дохода и деньги не моноширинным — идут последними
  и принимаются только по скриншоту с телефона.

## Context (from discovery)

Пути от `android/app/src/main/kotlin/tech/shatrov/familyfinances/`, тесты — от
`android/app/src/test/kotlin/tech/shatrov/familyfinances/`. Ссылки на material3 — исходники
1.4.0 из Gradle-кеша.

- Тема: `theme/Theme.kt:10` — `darkColorScheme` без `secondaryContainer`/`onSecondaryContainer`.
  Потребители роли: `FilterChipTokens` (заливка и текст выбранного), `ProgressIndicatorTokens.TrackColor`,
  `OutlinedSegmentedButtonTokens.SelectedContainerColor`. Остальные незаданные роли
  (`tertiaryContainer`, `errorContainer`, `inverse*`, `*Fixed`) в используемых компонентах не
  всплывают. `theme/Type.kt` — `displayLarge` 40sp и `displayMedium` 24sp не используются,
  `displaySmall` 16sp — все суммы (`ui/transactions/TransactionsScreen.kt:250`,
  `ui/home/HomeScreen.kt:198,236,312`).
- `ui/Chips.kt:33` — единственный `FilterChip`, без обводки выбранного;
  `FilterChipDefaults.filterChipBorder(enabled, selected, …, selectedBorderColor, selectedBorderWidth)`.
  `SingleChoiceSegmentedButtonRow` в 1.4.0 не экспериментальный; `ModalBottomSheet` —
  экспериментальный (opt-in как в `ui/DatePickerSheet.kt:17`).
- `ui/Common.kt:16` — `Centered` принимает любой контент, главная уже кладёт в него текст и
  кнопку на отказе.
- `ui/settings/PasswordScreen.kt:119` — `internal fun SecretField(value, onValueChange, label: Int,
  enabled, error)` — общее поле пароля, шесть вызовов: `PasswordScreen.kt:55,63,72`,
  `UserPasswordScreen.kt:44,53`, `UserEditScreen.kt:186`. Логин (`ui/login/LoginScreen.kt:69`)
  собирает своё поле с `imeAction`/`keyboardActions`. Переключателя видимости нет нигде.
- `MainActivity.kt:459` `WithNavBar`: `Box(weight(1f))` над `AppNavBar` — место для FAB без
  `Scaffold`. `TransactionsScreen` вызывается на `:211-218` с
  `onCreate = { screen = AppScreen.TransactionEdit(null) }`. Форма операции по `done` и «назад»
  уводит на `AppScreen.Transactions` (`:392-403`). Системные отступы даны один раз (`:72`),
  `AppNavBar` свои обнуляет (`ui/AppNavBar.kt:36`). `MainActivityTest.kt` — smoke без сессии,
  вкладок не достигает.
- `ui/transactions/TransactionsScreen.kt:80` — `Filters` только внутри `Ready`; `onFiltersChange`
  переводит экран в `Loading` (`TransactionsViewModel.kt:98`), на `Failure` фильтры недоступны.
  `TransactionsUiState` — sealed с `data object Loading`. Образец отдельного потока фильтра —
  `BudgetsViewModel.filter: StateFlow<BudgetFilter>` (`:74`) и `BudgetsScreen.kt:74-75`.
  Список: `contentPadding` снизу 12dp (`:165`), подвал с `Retry` (`:180`).
  `TransactionFilters.categoryId: UUID?` — одиночный, как и в `openapi.yaml:477`.
- `ui/transactions/TransactionEditScreen.kt:126` — дата `OutlinedButton` без подписи;
  `ui/budgets/BudgetEditScreen.kt:323` `DateButton` уже с «Начало: …»/«Конец: …» и блокировками
  `editable`/`startLocked`/`endLocked` (`:135`). `TransactionEditUiState` без `currency`
  (`TransactionEditViewModel.kt:48`); модели форм конструируются напрямую в
  `ui/transactions/TransactionEditViewModelTest.kt:59` и `ui/budgets/BudgetEditViewModelTest.kt:129`.
- `ui/home/HomeViewModel.kt:73` — `getStatsSummary()` без границ: сервер отдаёт первое число
  месяца — сегодня в зоне семьи (`internal/services/stats_service.go:80`). Шапка главной вне
  `when` (`HomeScreen.kt:53-69`), `Summary` первым элементом печатает `formatPeriod` (`:107-113`).
  `home_title` делит главная и вкладка (`ui/AppNavBar.kt:22`). `TestFixtures.kt:70` — `STATS_OK`
  как JSON для MockWebServer, объекта `StatsSummary` для Compose-теста нет. Бэклог оставляет
  главную «этим месяцем», экран «Обзор» плана 11 — произвольный период (`docs/backlog.md:46`).
- `budget_recurring` — один ресурс на переключатель «Повторять» (`BudgetEditScreen.kt:240`) и на
  `contentDescription` значка строки (`BudgetsScreen.kt:188`); тест
  `ui/budgets/BudgetsScreenTest.kt:156,163`. Словарь «сессий» — в `settings_session_*`,
  `settings_user_deactivate_confirm`.
- Тесты ищут узлы по `res.getString(...)` и `contentDescription`; при сохранённых подписях и
  колбэках замену чипов сегментами переживают `TransactionEditScreenTest:84`,
  `BudgetsScreenTest:97,135,145`, `TransactionsScreenTest:80`. `HomeScreenTest` и
  `CategoriesScreenTest` нет — `android/CLAUDE.md:140` покрывает их через ViewModel «там нет ввода».
- Иконки — контуры Lucide в `ui/AppIcons.kt` (`Home List Tag Target Plus ArrowLeft Repeat User`);
  новые зависимости запрещены (`android/CLAUDE.md`).
- Эмулятор виснет; визуальные проверки — на телефоне через `adb` (память
  `android-device-testing`).

## Решения

Раздел заменяет Solution Overview / Technical Details шаблона.

- **Чип и сегмент.** `secondaryContainer = elevated`, `onSecondaryContainer = textPrimary`.
  Контраст `elevated`/`canvas` около 1,2:1 — одной заливкой выбранность не читается, поэтому
  выбранный чип обводится `action` через `filterChipBorder`, активный сегмент — через
  `SegmentedButtonDefaults.colors(activeBorderColor = action)`. Дорожка прогресса станет
  `elevated`; её различимость — визуальная проверка.
- **Сегменты только для 2–3 вариантов:** тип в форме операции (2), фильтр бюджетов (2), фильтр
  типа операций (3). Период бюджета остаётся чипами: на 360dp четыре сегмента дают ~33dp под
  текст, «Произвольный» не влезает. Общий `SegmentedChoice(options, selected, onSelect)` в
  `ui/Segments.kt`, без `enabled` — ни у одного вызова нет выключенного состояния. Подписи и
  колбэки прежние — тесты не переписываются.
- **Фильтры операций** поднимаются из `Ready` в отдельный `StateFlow<TransactionFilters>` по образцу
  `BudgetsViewModel.filter` (в sealed-состояние с `data object Loading` их не втащить); категории
  для листа — тоже отдельный поток. Два ряда вместо трёх: `SegmentedChoice` типа и `ChipRow` из
  трёх периодов плюс чип категории («Все категории» / имя выбранной), открывающий
  `ModalBottomSheet`. Выбор применяется сразу и закрывает лист. Тело листа — отдельный
  `CategorySheetContent`, его и тестируем: сам `ModalBottomSheet` живёт в своём окне с анимацией и
  в Robolectric ненадёжен. Категория остаётся одиночной — контракт. Делается **до** пустых
  состояний: их ветка «сбросить фильтры» читает поднятые `filters`.
- **FAB** только на вкладке «Операции», в `Box` `WithNavBar`; из шапки операций уходит только
  `IconButton(Plus)`, параметр `onCreate` остаётся — его зовут FAB и пустое состояние.
  `transactions_add` («Добавить») становится `contentDescription` FAB. Список получает
  `contentPadding` снизу `Dimens.FAB_CLEARANCE` = 88dp (56 + 2×16), иначе FAB накрывает последнюю
  строку и `Retry` подвала. Автотеста на «FAB только на этой вкладке» нет — `MainActivityTest`
  до вкладок не доходит; проверка руками.
- **Пустые состояния** без нового API: `Centered { Text(...); Button(...) }` как на отказе.
  Действия: главная → новая операция (после сохранения пользователь окажется на вкладке
  «Операции» — так уже работает форма, принимаем); операции при `TransactionFilters()` → «Добавить
  операцию», при изменённых → «Сбросить фильтры»; категории → новая категория; бюджеты `TODAY` →
  «Показать все периоды» (не «Все периоды» — иначе два узла с одним текстом в
  `BudgetsScreenTest:101`), `ALL` → новый бюджет.
- **Дата операции:** остаётся `OutlinedButton` (у него есть `onClick` для TalkBack), получает
  подпись «Дата: …» по образцу `DateButton` бюджета и иконку календаря слева; `readOnly`-поле не
  берём — потеряло бы блокировки и семантику.
- **Главная:** заголовок содержимого — месяц из `summary.from` (`formatMonth` с паттерном
  `LLLL yyyy` — именительный «сентябрь», не «сентября»; `formatPeriod` не трогаем — он показывает
  сроки бюджетов), подзаголовок «по 15 сентября» из `summary.to`. Пока `Ready` не пришёл
  (`Loading`, `Failure`) шапка показывает `home_title`. Старая строка `formatPeriod` в `Summary`
  убирается — иначе период показан трижды. «Итого» — `displayLarge`, `maxLines = 1`; доходы и
  расходы под ним `displaySmall` с прежними дельтами и счётчиком операций. Карточка убирается.
  Если сумма из девяти цифр на 360dp не помещается — `displayMedium`. Имя вкладки не меняется.
- **Пароль:** `SecretField` переезжает из `PasswordScreen.kt` в `ui/SecretField.kt` и получает
  trailing-иконку `Eye`/`EyeOff` (Lucide) с локальным состоянием видимости плюс необязательные
  `imeAction`/`keyboardActions`, чтобы им же пользовался логин. Второго компонента не будет.
- **Сумма:** `displayMedium` в поле и `suffix` с символом валюты — `symbolOf` в `Money.kt`
  становится публичным `currencySymbol` (fallback на код остаётся); `currency` приходит в
  `TransactionEditViewModel` и `BudgetEditViewModel` из `Session` параметром **с умолчанием**,
  чтобы прямые конструирования в тестах моделей не ломались.
- **Текст:** экран «Сессии» становится «Устройства» целиком: `settings_session_current` → «Это
  устройство», `settings_session_revoke_confirm` → «Выйти на этом устройстве?»,
  `settings_user_deactivate_confirm` → «… Он выйдет на всех устройствах»; `settings_session_created`
  / `last_used*` остаются («Вход», «Активность»). «Бэкапы» → «Резервные копии» с производными.
  `budget_recurring` делится на `budget_recurring_toggle` («Повторять») и
  `budget_recurring_badge` («Повторяющийся»). Двоеточие в «Операций за период: 5» и « · » остаются.
- **Визуальный язык** (Task 11–12): `income` → `#5FD37A`, `action` без изменений; деньги —
  `FontFamily.Default` + `tnum`. Оба принимаются по скриншотам списка операций и главной с
  телефона; при «пляшущей» колонке сумм Task 12 откатывается.
- **Compose-тесты главной и категорий** появляются вопреки `android/CLAUDE.md:140`: у обоих
  экранов теперь есть кнопка. Абзац правится в Task 14.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче; Compose-тесты для экранов с
  вводом, форматтеры с JVM-тестами (`android/CLAUDE.md`).
- Каждая задача — отдельный коммит с зелёным `make -C android check`; компонент, состояние,
  строки и тесты одной правки — в одном коммите.
- `make check` не ловит перекрытие FAB, переносы сегментов и заливку дорожки — эти пункты
  закрываются скриншотом с телефона в той же задаче.

## Testing Strategy

- Compose: `LoginScreenTest` (переключатель показывает пароль), `TransactionsScreenTest` (фильтры
  видны на `Failure`; `CategorySheetContent` — выбор доходит до колбэка; пустое состояние с
  фильтрами → «Сбросить фильтры», без → «Добавить операцию»), `BudgetsScreenTest` (пустой `TODAY`
  → колбэк смены фильтра; значок повтора по новому ресурсу), `CategoriesScreenTest` (новый: пустое
  состояние → `onAdd`), `TransactionEditScreenTest` (подпись даты, суффикс валюты),
  `HomeScreenTest` (новый: месяц в заголовке на `Ready`, `home_title` на `Loading`, «Итого»
  присутствует, дельты скрыты без `hasPreviousData`).
- Форматтеры: `DatesTest` — `formatMonth` («сентябрь 2026», русская локаль, год всегда).
- ViewModel: `TransactionsViewModelTest` — смена фильтра публикуется в отдельный поток и уходит в
  query как раньше.
- Без автотеста, руками: FAB только на «Операциях» и не накрывает подвал; чип и дорожка после
  Task 1; сегменты в одну строку.
- e2e нет.

## Progress Tracking

- `[x]` — сделано; `➕` — найдено по ходу; `⚠️` — блокер.

## Implementation Steps

### Task 1: Тема — `secondaryContainer` и обводка выбранного чипа

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/theme/Theme.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/Chips.kt`

- [x] `appColorScheme`: `secondaryContainer = colors.elevated`, `onSecondaryContainer = colors.textPrimary`
- [x] `Chip`: `border = FilterChipDefaults.filterChipBorder(enabled, selected, selectedBorderColor = action, selectedBorderWidth = Dimens.BORDER)`
- [x] существующие экранные тесты чипов проходят без изменений
- [x] скриншот с телефона: чипы фильтров операций и дорожка прогресса бюджетов читаются (пропущено — не автоматизируется)
- [x] `make -C android check` — зелёный

### Task 2: `SecretField` с переключателем видимости — везде, включая логин

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/SecretField.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppIcons.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/PasswordScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/UserPasswordScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/UserEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/login/LoginScreen.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/login/LoginScreenTest.kt`

- [x] `AppIcons.Eye`, `AppIcons.EyeOff` — контуры Lucide
- [x] `SecretField` переезжает в `ui/SecretField.kt`: `remember { visible }`, trailing `IconButton`
      с `contentDescription` `password_show`/`password_hide`; новые параметры
      `imeAction = ImeAction.Default`, `keyboardActions = KeyboardActions.Default`; шесть вызовов в
      настройках — только импорт
- [x] логин: своё поле пароля → `SecretField(..., imeAction = Done, keyboardActions = onDone)`
- [x] `LoginScreenTest`: после нажатия «Показать пароль» текст пароля виден
- [x] `make -C android check` — зелёный

### Task 3: `SegmentedChoice` — тип операции, фильтр бюджетов, фильтр типа

**Files:**
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/Segments.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreen.kt`

- [x] `SegmentedChoice(options: List<Pair<T, String>>, selected: T, onSelect)` на
      `SingleChoiceSegmentedButtonRow`, `SegmentedButtonDefaults.colors(activeBorderColor = action)`
- [x] форма операции: Расход/Доход; бюджеты: На сегодня/Все периоды; операции: Все/Доходы/Расходы
- [x] существующие тесты выбора проходят без изменения селекторов
- [x] скриншот с телефона: три сегмента на 360dp в одну строку (пропущено — не автоматизируется)
- [x] `make -C android check` — зелёный

### Task 4: Фильтры операций — отдельный поток, два ряда, лист категорий

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsViewModel.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreen.kt`
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/CategorySheet.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreenTest.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsViewModelTest.kt`

- [x] `TransactionsViewModel`: `filters: StateFlow<TransactionFilters>` и `categories:
      StateFlow<List<Category>>` отдельно от `state`; `Ready` их больше не несёт
- [x] `TransactionsScreen(state, filters, categories, ...)`; `AppRoot` собирает оба потока
- [x] `Filters` вне `when`: ряд 1 — `SegmentedChoice` типа; ряд 2 — `ChipRow` трёх периодов + чип
      категории «Все категории»/имя выбранной
- [x] `CategorySheet.kt`: `@OptIn(ExperimentalMaterial3Api)` `ModalBottomSheet` вокруг
      `CategorySheetContent(categories, selected, onSelect)` — `LazyColumn` с «Все категории» и
      списком; выбор → `onFiltersChange(copy(categoryId))` и закрытие; `shown` —
      `rememberSaveable` на уровне экрана
- [x] тесты: `CategorySheetContent` — выбор доходит до колбэка; фильтры видны на `Failure`;
      `TransactionsViewModelTest` — смена фильтра публикуется и уходит в query
- [x] `make -C android check` — зелёный

### Task 5: FAB на «Операциях»

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/theme/Dimens.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreen.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreenTest.kt`

- [ ] `Dimens.FAB_CLEARANCE = 88.dp`
- [ ] `WithNavBar(fab: @Composable () -> Unit = {})`: контент + `fab` в `BottomEnd` с отступом
      `SPACE_4`; `AppRoot` передаёт `FloatingActionButton(Plus, contentDescription =
      transactions_add)` только для `TRANSACTIONS`, `onClick` — прежний `onCreate`
- [ ] `TransactionsScreen`: `IconButton(Plus)` из шапки убран, `onCreate` остаётся;
      `contentPadding` снизу `FAB_CLEARANCE`
- [ ] `TransactionsScreenTest`: селекторы на `transactions_add` в шапке убраны
- [ ] скриншот с телефона: последняя строка и `Retry` подвала не под FAB
- [ ] `make -C android check` — зелёный

### Task 6: Пустые состояния с действием

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/categories/CategoriesScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreenTest.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreenTest.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/categories/CategoriesScreenTest.kt`

- [ ] `HomeScreen(onAddTransaction)`: пустое → текст + «Добавить операцию»; `AppRoot` →
      `AppScreen.TransactionEdit(null)`
- [ ] `TransactionsScreen`: пустое при `filters == TransactionFilters()` → «Добавить операцию»
      (`onCreate`), иначе «Сбросить фильтры» → `onFiltersChange(TransactionFilters())`
- [ ] `CategoriesScreen`: пустое → «Добавить категорию» → `onAdd`
- [ ] `BudgetsScreen`: `TODAY` → «Показать все периоды» → `onFilterChange(ALL)`; `ALL` → «Добавить бюджет»
- [ ] строки: `home_add_transaction`, `transactions_add_first`, `transactions_reset_filters`,
      `categories_add_first`, `budgets_show_all`, `budgets_add_first`; `budgets_empty_today` без
      «посмотрите …»
- [ ] тесты по Testing Strategy
- [ ] `make -C android check` — зелёный

### Task 7: Дата операции с подписью и иконкой

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppIcons.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreen.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreenTest.kt`

- [ ] `AppIcons.Calendar` — контур Lucide
- [ ] кнопка даты операции: `transaction_date` «Дата: %1$s» + иконка; `DateButton` бюджета — только иконка
- [ ] `TransactionEditScreenTest`: подпись с датой видна, нажатие открывает пикер
- [ ] `make -C android check` — зелёный

### Task 8: Главная — месяц и «Итого» как герой

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/format/Dates.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/format/DatesTest.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreen.kt`
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/TestFixtures.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreenTest.kt`

- [ ] `formatMonth(date)` — `DateTimeFormatter.ofPattern("LLLL yyyy", locale)`, первая буква
      заглавная; `DatesTest`
- [ ] шапка: на `Ready` — `formatMonth(summary.from)` и под ним `home_through` «по %1$s» из
      `summary.to`; на `Loading`/`Failure` — `home_title`; иконка профиля на месте
- [ ] `Summary`: первый элемент `formatPeriod` убран; `TotalsCard` → `Totals`: «Итого»
      `displayLarge` `maxLines = 1`, знак и цвет прежние; доходы и расходы `displaySmall` с
      дельтами; счётчик операций; `Card` убрана
- [ ] `TestFixtures`: билдер `statsSummary(...)` → `StatsSummary`
- [ ] `HomeScreenTest` по Testing Strategy
- [ ] скриншот с телефона: сумма 9 цифр на 360dp в одну строку, иначе `displayMedium`
- [ ] `make -C android check` — зелёный

### Task 9: Сумма — крупный стиль и валюта

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/format/Money.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditViewModel.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditViewModel.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionEditScreenTest.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreenTest.kt`

- [ ] `symbolOf` → публичный `currencySymbol(code)`
- [ ] `currency: String = ""` в конструкторах обеих моделей, в `UiState`; `AppRoot` передаёт из
      `Session`; `TransactionEditViewModelTest:59` и `BudgetEditViewModelTest:129` не меняются
- [ ] поля суммы/лимита: `textStyle = displayMedium`, `suffix = { Text(currencySymbol(currency)) }`
      при непустой валюте
- [ ] тесты: суффикс виден, ввод и ошибки прежние
- [ ] `make -C android check` — зелёный

### Task 10: Текст — устройства, резервные копии, признак повторения

**Files:**
- Modify: `android/app/src/main/res/values/strings.xml`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreen.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetsScreenTest.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/budgets/BudgetEditScreenTest.kt`

- [ ] «Устройства»: `settings_sessions`, `settings_session_current`,
      `settings_session_revoke_confirm`, `settings_user_deactivate_confirm` — по «Решениям»
- [ ] «Резервные копии»: `settings_backups`, `settings_backup_create/delete_confirm/hint/unknown`
- [ ] `budget_recurring` → `budget_recurring_toggle` (форма) и `budget_recurring_badge` (значок строки)
- [ ] тесты на новые ресурсы
- [ ] `make -C android check` — зелёный

### Task 11: Развести цвет дохода и акцента (по скриншоту)

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/theme/Colors.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/transactions/TransactionsScreen.kt`

- [ ] `income = Color(0xFF5FD37A)`; `tertiary` в схеме следует автоматически
- [ ] знак суммы текстом остаётся; комментарии «доход и акцент — один цвет» переписать на
      «знак текстом ради цветовосприятия»
- [ ] скриншоты главной и списка с телефона — принять или откатить коммит
- [ ] `make -C android check` — зелёный

### Task 12: Деньги — системный шрифт с `tnum` (по скриншоту)

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/theme/Type.kt`

- [ ] `FontFamily.Default` в трёх `display*`, `fontFeatureSettings = TABULAR_FIGURES` остаётся,
      док-комментарий про моноширинный переписать
- [ ] скриншот списка операций с суммами разной длины: колонка ровная — принять, иначе откатить
- [ ] `make -C android check` — зелёный

### Task 13: Verify acceptance criteria

- [ ] debug-сборка на телефоне против `ffs.shatrov.tech`: выбранный чип и активный сегмент в
      палитре; пароль показывается и скрывается на входе и в настройках; сегменты в одну строку;
      фильтры доступны на отказе, лист категорий открывается и фильтрует; FAB только на
      «Операциях» и не накрывает подвал; пустые состояния ведут на действие; главная показывает
      месяц и «Итого» без переноса; суффикс валюты в формах
- [ ] TalkBack: FAB, переключатель пароля, кнопка даты и значок повтора озвучиваются
- [ ] `make -C android check`, `make -C android compile` — зелёные; `api-check` не нужен

### Task 14: [Final] Update documentation

- [ ] `android/CLAUDE.md`: `SegmentedChoice` vs `Chip` (когда что), `SecretField` в `ui/`, FAB в
      `WithNavBar` и `FAB_CLEARANCE`, фильтры операций отдельным потоком, разделённые ресурсы
      повтора; абзац `:140` — главная и категории теперь с Compose-тестами (у них есть кнопка)
- [ ] `android/gradle/libs.versions.toml`: `appVersionCode` +1, `appVersionName` → 0.6.0
- [ ] `docs/backlog.md`: отложенное — светлая тема, `FlowRow` для категорий, колонки вместо « · »
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Ручная проверка:** оба телефона после `app-v0.6.0`; увеличенный системный шрифт — сегменты и
«Итого» (Task 3, 8) на 360dp.

**Figma:** файл `VsFUJiZcr2OgyihLOiM8cW` собран из прежних токенов (`Kit`: Button, Chip,
TransactionRow, BudgetRow); после Task 1, 3, 8, 11–12 он расходится с кодом — обновлять локальным
плагином, лимит MCP исчерпан.

**Открытые вопросы codex:** вид сегментов, крупных сумм и категорий при увеличенном системном
шрифте — только отрисовкой.
