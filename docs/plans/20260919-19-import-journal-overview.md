# План 19. Журнал незавершённого импорта и экран «Обзор»

## Overview

Две клиентские доработки одним релизом `0.11.0`; контракт и сервер не меняются.

**Журнал импорта.** Смерть процесса сейчас теряет импорт целиком: картинки, оплаченный ответ распознавания,
правки строк и их `draft`-UUID — и деньги, и гарантию «без дублей» (`docs/backlog.md:172`). Типичный случай —
ушли в банк за следующим скриншотом, Android убил приложение в фоне.
- Импорт ведёт журнал на диске и восстанавливается из него без повторного платного вызова.
- Убитый системой процесс возвращается прямо в экран распознавания; после смахивания или падения — плашка на
  «Главной»: «Продолжить / Удалить».

**«Обзор».** «Главная» показывает только текущий месяц, а `GET /stats/summary?from&to` и `GET /stats/monthly`
(план 10) экрана не имеют — это незакрытая половина плана 11.
- Период чипами, итоги с дельтой, столбики дохода и расхода за 12 месяцев, категории.
- Тап по категории — операции этой категории за период; тап по столбику — период «этот месяц».

Не входит: ждущий второй share (права на его URI умирают с процессом), несколько журналов сразу, произвольный
диапазон дат, мультивыбор и массовое удаление операций, бюджеты и последние операции за период — сервер считает
их на сегодня.

## Context (from discovery)

Импорт:
- `ImportStore.kt` — владение импортом в памяти: `offer` / `claim` / `hold` / `release`, потоки `pending`, `waiting`.
- `ui/recognize/RecognizeViewModel.kt`: `init` :151 (`claim == null` → `recognize_import_lost`), `recognize()` —
  платный вызов один, `retry()` :168 только по кнопке; `saveRow` (`draft` = `id` у `POST`, `200` — запись уже
  была); `onCleared()` :205 — `release` + `ImportFiles.discard`; правки строк (`edit`, `replace`) синхронные, а
  описание правится на каждое нажатие (`RecognizeScreen.kt:364`).
- `ui/recognize/ImportFiles.kt`: `root` :99 = `cacheDir/import`, общий для `prepare` :54, `discard` :67, `sweep` :78
  (зовётся один раз, `FamilyFinancesApp.kt:17`) и `cameraUri` :92; `ImportImage.uri: Uri` не nullable (:20-32),
  экран читает только `file` и причину отказа. `res/xml/file_paths.xml` отдаёт наружу только
  `cache-path import/camera/`. `AndroidManifest.xml:10` — `allowBackup="false"`.
- `RecognizedTransaction` помечает `date` и `categoryId` как `@Contextual`; сериализаторы — в публичном
  `apiSerializersModule` (`core/api/src/main/.../net/ApiSerializers.kt:69`). В `:app` плагина сериализации нет;
  он стоит `classpath` в корневом `build.gradle.kts:9`, `kotlinx-serialization-json` приходит транзитивно.
- **Восстановленный экран сейчас не выживает:** `MainActivity.kt:171-175` при `session == null` заменяет любой
  экран на `Loading`, а после бутстрапа `:224` ведёт в `pending → Recognize`, иначе на «Главную». `Saver` вернёт
  `Recognize(importId)`, и маршрутизация тут же его затрёт.
- `forms` чистится без ведома модели при `sessionExpired` (`MainActivity.kt:177`), `signOut` из бутстрапа (:232)
  и из настроек (:638). Вытеснение новым share — `LaunchedEffect` в `AppRoot` (:204-208), ссылки на модель там нет.
- «Главная»: `HomeScreen.kt` — `when (state)` :92, пустое состояние :109-118, `Totals` — элемент `LazyColumn`
  внутри `Summary` (:143); `HomeViewModel` (:62) знает только `api`, `currency`, `zone`.

«Обзор»:
- `StatsSummary` (`internal/services/dto/stats_dto.go:16`): `current`, `previous`, `has_previous_data`, дельты-доли,
  `expense_categories` / `income_categories` (`share` 0…1); `budgets` (`stats_service.go:110`) и `recent` (:115)
  от периода не зависят. Предыдущий период — той же длины вплотную перед выбранным (`previousPeriod`, :396).
  Категории группируются по `category_id` самой операции, без свёртки к родителю
  (`transaction_repository_sqlite.go:829`) — расшифровка по точному `categoryId` сходится со строкой.
- `getStatsMonthly()` без параметров — 12 календарных месяцев по сегодня в зоне семьи, считает сервер.
- Фильтр операций: `today` — в зоне семьи (`TransactionsViewModel.kt:71`); KDoc `Filters.kt:19-22` про «дату
  телефона» устарел. Периоды `ALL / THIS_MONTH / PREV_MONTH / MONTH`; `MONTH` рисуется статичным чипом
  (`TransactionsScreen.kt:171`).
- `AppScreen.kt`: `saveKey` :193-203 пишет **шесть** полей через `:` (`period:month:type:accountId:unassigned:
  reconciliation`), категория намеренно не сохраняется (:192); `restoreTransactions` :205-219; чужой ключ безопасен —
  `restore` обёрнут в `runCatching` → `Loading` (:130).
- «Открыт снаружи»: `drillFrom = current.reconciliation` и `dropDrill` (`MainActivity.kt:192`, `:315-328`) —
  уход со списка сбрасывает внешний фильтр только при `reconciliation != null`.
- Флаги устаревания: каждый потребляет и гасит ровно один `LifecycleResumeEffect` (`MainActivity.kt:272`, `:304`,
  `:407`, `:441`); взводятся формой операции (:750) и распознаванием (:709).
- Маршрутизация share-импорта — `when` с `else -> Unit` (`MainActivity.kt:185-212`): пропущенный экран молчит.
- График-прецедент: `ui/networth/NetWorthChart.kt` — Canvas, геометрия чистой функцией (`chartGeometry` :29).

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Задачи 1–4 — журнал, 5–7 — «Обзор»; задача 2 даёт возврат экрана после бутстрапа, на который опирается 7.
- Каждая задача — `make -C android check`; сервер не трогается.
- **CRITICAL: update this plan file when scope changes during implementation**

## Testing Strategy

- Robolectric: журнал, восстановление модели по фазам, модель «Обзора», геометрия графика, `AppScreenSaverTest`.
- Уровень `AppRoot` (`AppRootImportTest`, `StateRestorationTester`): возврат экрана после смерти процесса,
  чистка журнала при выходе, сброс внешнего фильтра — ViewModel-тесты этого не видят.
- Compose-тесты: плашка («Удалить» с подтверждением), экран «Обзора».
- Станут красными и правятся в своих задачах: `RecognizeViewModelTest.clearingModelDiscardsFilesAndReleasesHold`
  (:444, путь `cacheDir/import`), `ImportFilesTest` (:121, :154).

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

### Журнал импорта

**Файл рядом с картинками, в `filesDir`.** `filesDir/import/<importId>/journal.json` и `N.jpg`: `cacheDir`
система чистит сама, а журнал — оплаченный результат. У `ImportFiles` становится два корня: `filesRoot` для
импортов и прежний `cacheRoot` для `camera/` — иначе `FileProvider` потерял бы камеру.

**Писатель один, вне главного потока.** `Channel(CONFLATED)` и один сборщик на `Dispatchers.IO`: правки строк
сливаются, запись атомарная (временный файл + `renameTo`). Две записи идут вне слияния и **ожидаются** до смены
фазы — список файлов после `prepare` и сырой ответ после платного вызова: потерянная запись ответа — это
потерянные деньги.

**Восстановление по состоянию журнала:**

| В журнале | После восстановления |
|---|---|
| журнала нет (умер в `prepare`) | импорт потерян, как сейчас: URI уже недоступны |
| картинки есть, `recognizing = false`, ответа нет | обычный первый вызов `recognize` — денег ещё не тратили |
| `recognizing = true`, ответа нет | отказ с «Повторить», картинки на месте; сам клиент вызов не повторяет — исход платного запроса неизвестен |
| ответ есть | экран как был: строки, правки, счёт, статусы; платного вызова нет, справочники перечитываются с сервера |
| строка в `saving` | строка `Failed` «сохранение прервано» с «Повторить»: тот же `POST` с тем же `draft`, `200` или `201` — дубля нет |

**Журнал удаляют только явные события,** не `onCleared`: смахивание иногда успевает вызвать `onDestroy` с
`isFinishing`, и журнал исчез бы именно тогда, когда нужен. События: уход с экрана (`leave`), «Удалить» на
плашке, полное сохранение, **выход из аккаунта и `401`** (`deleteAll` — журнал с банковскими данными одного
пользователя не должен достаться следующему), новый импорт (`deleteOthers(keep)` в `init` его модели — инвариант
«один журнал» держится там, а не в `AppRoot`, у которого нет ссылки на модель).

**Возврат экрана после бутстрапа.** Экран, стоявший до ухода в `Loading`, запоминается (`rememberSaveable`) и
после успешного бутстрапа возвращается в порядке `pending → Recognize` ▸ запомненный ▸ «Главная». Запоминаются
только экраны, которым не нужна живая модель формы: `Recognize(importId)` — если журнал есть — и `Overview(period)`.

**Срок жизни — сутки, и на чтении тоже.** `sweep` бывает только на старте, а процесс живёт днями: `latest()` и
`read()` сами не отдают журнал старше 24 часов.

**Сериализация — плагин в `:app`,** свой `Json { ignoreUnknownKeys = true; serializersModule =
apiSerializersModule }` — без модуля `RecognizeResult` с `@Contextual`-полями не сериализуется. `checkReleaseModelsKept`
не расширяется: он сверяет каталог сгенерированных моделей по правилу «имя файла = имя класса», а
`encodeToString(journal)` разрешает сериализатор статически, R8 его не удалит. Восстановление проверяется на
телефоне релизной сборкой.

### «Обзор»

**Вход — строка «Обзор ›» под итогами «Главной»,** не вкладка: панель полна.

| Чип | Границы (зона семьи) |
|---|---|
| Этот месяц | 1-е число … сегодня |
| Прошлый месяц | полный прошлый месяц |
| 3 месяца | 1-е число месяца два месяца назад … сегодня |
| Год | 1-е число месяца одиннадцать месяцев назад … сегодня — равно умолчанию `monthly` |
| (столбик) | полный выбранный месяц; для текущего — по сегодня |

**График всегда за 12 месяцев, период на нём подсвечен.** `monthly` запрашивается **без параметров** — границы
считает сервер, и сумма столбиков сходится с чипом «Год» без гонки на границе суток. `summary(from, to)` — на
каждую смену периода. Тап по столбику выбирает месяц.

**Дельта — «к предыдущему периоду той же длины», и только при `has_previous_data`.** Для «этого месяца» 19-го
числа это 19 дней до 1-го, а не прошлый месяц целиком; подпись говорит именно это.

**Расшифровка несёт те же даты, что ушли в `summary`.** Новый `TransactionPeriod.RANGE` с `from` / `to` —
состояние «открыт снаружи», как `MONTH`: в ряду фильтров статичный чип с диапазоном. Категория при этом обязана
сохраняться в `Saver` — иначе после поворота список покажет чужие суммы.

**Свой флаг `overviewStale`.** Флаг гасит тот, кто его прочёл: общий с «Главной» `homeStale` «Обзор» съел бы,
и «Главная» осталась бы со старыми итогами.

## Technical Details

```kotlin
@Serializable
data class ImportJournal(
    val version: Int,               // JOURNAL_VERSION = 1; другая версия — журнал выбрасывается
    val importId: String,
    val updatedAt: Long,            // время последней записи: срок жизни и подпись плашки
    val images: List<JournalImage>, // строка URI (только ключ, права мертвы) + имя файла или причина отказа
    val dropped: Int,
    val recognizing: Boolean,
    val result: RecognizeResult?,
    val accountId: String?,         // при восстановлении сверяется со списком счетов, как lastAccount (:222)
    val rows: List<JournalRow>,     // наложение на result.items по индексу
    val savedCount: Int,
)
@Serializable
data class JournalRow(
    val draft: String, val included: Boolean, val date: String?, val categoryId: String?,
    val description: String, val status: String, // pending | saving | saved | failed
)
```

Ошибки полей и текст отказа не пишутся. Плашка рисуется в `HomeScreen` **над** `when (state)` — при любом
состоянии сводки, журнал локальный:

```
┌──────────────────────────────────────────┐
│ Незавершённый импорт                     │
│ 5 строк, сохранено 2 · сегодня 14:20     │
│ [ Продолжить ]              [ Удалить ]  │
└──────────────────────────────────────────┘
```

`recognizing` без ответа — «Распознавание прервано · 3 скриншота». Плашка скрыта, пока `ImportStore.pending` или
`waiting` непусты: это условие собирается в ветке `AppScreen.Home` (`MainActivity.kt:264`) и приходит в экран
параметром, `HomeViewModel` знает только про журнал.

```
Обзор                                  ‹ назад
( Этот месяц ) ( Прошлый ) ( 3 мес ) ( Год )

Остаток                          +84 300,00 ₽
Доход            312 000,00        +4 %
Расход           227 700,00       −11 %
к предыдущему периоду той же длины

 ▂▃ ▃▄ ▂▃ ▄▅ ▃▃ ▅▆ ▄▄ ▃▅ ▄▆ ▅▅ ▄▆ █▇      выбранные месяцы — в полную силу,
 о  н  д  я  ф  м  а  м  и  и  а  с       остальные приглушены

Расходы по категориям
│ Продукты            68 400,00      30 % ›│
│ Ипотека             62 000,00      27 % ›│
Доходы по категориям
│ Зарплата           290 000,00      93 % ›│
```

- Столбики парные, общий масштаб по максимуму ряда; нулевой месяц — нулевая высота. Геометрия — `barGeometry(months)`.
- Категории — все от сервера, по убыванию; свёртки «Прочее» нет.
- Пустой период: итоги нулями, вместо списков «За период операций нет», график остаётся.
- Ошибка `summary` не стирает график и чипы: отказ на месте итогов с «Повторить».

`Saver` списка операций — десять полей: `period:month:type:categoryId:accountId:unassigned:from:to:reconciliation:
overview`, чтение по индексу; строка другой длины уходит в `Loading` через существующий `runCatching`.
`OverviewPeriod.saveKey()` двоеточий не содержит (`MONTH-2026-09`).

## Implementation Steps

### Task 1: `ImportJournal` — формат, запись, чтение, уборка

**Files:**
- Modify: `android/app/build.gradle.kts`, `ui/recognize/ImportFiles.kt`, `app/src/test/…/ui/recognize/ImportFilesTest.kt`
- Create: `ui/recognize/ImportJournal.kt`, `app/src/test/…/ui/recognize/ImportJournalTest.kt`

- [x] плагин сериализации в `:app` и явный `implementation(libs.kotlinx.serialization.json)`; `Json` с `apiSerializersModule`
- [x] `ImportFiles`: `filesRoot` = `filesDir/import` для импортов, `cacheRoot` = `cacheDir/import` для `camera/`; `cameraUri` и его ветка `sweep` остаются на `cacheRoot`; устаревший комментарий `:134` про `discard` модели поправить
- [x] `ImportJournalStore`: `write` атомарно, `read(importId)`, `latest()`, `delete`, `deleteOthers(keep)`, `deleteAll()`; нечитаемый файл, чужая версия и возраст больше 24 ч — `null`
- [x] `sweep`: каталоги без журнала, с нечитаемым, чужой версии, старше 24 ч; в старом `cacheDir/import` — всё, кроме `camera/`
- [x] тесты: круг запись→чтение с непустым `result` (`date`, `category_id`); обрыв записи не портит прежний журнал; мусор; версия; `latest` не отдаёт вчерашний; `sweep` по каждому правилу; `cameraUri` по-прежнему под `cacheDir`
- [x] `make -C android check`

### Task 2: Модель пишет журнал и восстанавливается; экран возвращается после бутстрапа

**Files:**
- Modify: `ui/recognize/RecognizeViewModel.kt`, `MainActivity.kt` (:171-175, :177, :204-208, :220-237, :638, :694), `AppGraph.kt`
- Modify: `RecognizeViewModelTest.kt`, `AppRootImportTest.kt`, `TestFixtures.kt`

- [x] писатель: `Channel(CONFLATED)` + сборщик на `Dispatchers.IO`; записи после `prepare` и после ответа — ожидаемые, вне слияния; `recognizing = true` пишется до вызова
- [x] `init`: новый импорт — `deleteOthers(keep = importId)`; `claim == null` → `read(importId)`; восстановление по таблице; `JournalImage` → `ImportImage` через сохранённую строку URI
- [x] `recognize()` с готовым `result` платный вызов не делает: справочники → `Review`; их отказ — `Failure` с «Повторить» тем же путём; `accountId` из журнала, которого нет в списке счетов, сбрасывается
- [x] `onCleared` — только `release`; `abandon()` удаляет каталог, его зовёт `leave`; всё сохранено — журнал удаляется сразу
- [x] `signOut` (оба места) и `sessionExpired` зовут `deleteAll()`
- [x] `AppRoot`: экран перед уходом в `Loading` запоминается (`rememberSaveable` + `AppScreenSaver`); после бутстрапа — `pending` ▸ запомненный ▸ «Главная»; запоминается `Recognize` только при существующем журнале
- [x] тесты модели: каждое состояние таблицы; после восстановления с ответом к `recognize` нет запросов; повтор прерванной строки → `200` принимает поля сервера; `abandon` удаляет каталог, `onCleared` — нет; удалённый счёт сбрасывается
- [x] тесты `AppRoot`: смерть процесса на `Recognize` с журналом → после бутстрапа снова `Recognize`; без журнала → «Главная»; после `signOut` каталога `files/import` нет
- [x] `make -C android check`
- ➕ `JournalRow` получил `dateAssumed`, `amountMinor`, `type`: без них подтверждённый год и поля сервера после `200` терялись бы при восстановлении
- ➕ `ImportJournalStore` помнит удалённые в процессе id: запись, начатая до `delete`/`deleteAll`, журнал не возвращает; `recognizing` остаётся взведённым и после отказа вызова

### Task 3: Плашка на «Главной»

**Files:**
- Modify: `ui/home/HomeScreen.kt`, `HomeViewModel.kt`, `MainActivity.kt` (:264), `strings.xml`
- Modify: `HomeViewModelTest.kt`, `HomeScreenTest.kt`, `AppRootImportTest.kt`

- [x] `HomeViewModel` отдаёт сводку `latest()`: строк, сохранено, `updatedAt`, фаза; перечитывается при возврате на «Главную»
- [x] плашка над `when (state)`; скрыта параметром при непустых `pending` / `waiting`; «Продолжить» → `AppScreen.Recognize(importId)`; «Удалить» → подтверждение → `delete`
- [x] тесты ViewModel: журнала нет — плашки нет; `recognizing` — свой текст; удаление убирает плашку
- [x] Compose-тест: плашка видна на загрузке, отказе и пустом состоянии; «Удалить» спрашивает подтверждение
- [x] `AppRootImportTest`: «Продолжить» открывает экран, и тот восстанавливается без `offer`
- [x] `make -C android check`

### Task 4: Документация журнала

**Files:**
- Modify: `android/CLAUDE.md`, `docs/backlog.md`

- [x] «Распознавание скриншотов»: журнал, таблица состояний, кто удаляет журнал и почему не `onCleared`, выход и `401`, срок жизни на чтении, два корня `ImportFiles`; абзац «гарантия только в живом процессе» переписать
- [x] навигация: возврат запомненного экрана после бутстрапа и какие экраны запоминаются
- [x] `docs/backlog.md:172` — пункт закрыт

### Task 5: Диапазон дат и категория в фильтре операций

**Files:**
- Modify: `ui/transactions/Filters.kt`, `TransactionsViewModel.kt`, `TransactionsScreen.kt`, `AppScreen.kt`
- Modify: тесты фильтра, `TransactionsViewModelTest.kt`, `AppScreenSaverTest.kt`

- [ ] `TransactionPeriod.RANGE`, поля `from` / `to`; `dateFrom` / `dateTo`; `withPeriod` их сбрасывает; фабрика `TransactionFilters.overview(type, categoryId, from, to)` рядом с `reconciliation`
- [ ] `RANGE` — статичный чип с диапазоном, как `MONTH` (`TransactionsScreen.kt:171`)
- [ ] `Saver`: поля `categoryId`, `from`, `to` (пока восемь, `overview` — в задаче 7), чтение по индексу; комментарий `AppScreen.kt:192` про несохраняемую категорию убрать
- [ ] KDoc `Filters.kt:19-22`: границы — в зоне семьи
- [ ] тесты: `RANGE` шлёт серверу обе даты; смена чипа периода сбрасывает диапазон; `Saver` — круг с диапазоном и категорией
- [ ] `make -C android check`

### Task 6: Модель «Обзора»

**Files:**
- Create: `ui/overview/OverviewPeriod.kt`, `ui/overview/OverviewViewModel.kt`, тесты к обоим
- Modify: `TestFixtures.kt` (`STATS_MONTHLY_OK`, `STATS_SUMMARY_RANGE_OK`)

- [ ] `OverviewPeriod`: четыре чипа и `Month(YearMonth)`; `bounds(today)` по таблице; `saveKey()` без `:`
- [ ] модель: `today` из `session.zone`; `getStatsMonthly()` без параметров — один раз и при `refresh`; `summary(from, to)` — на смену периода, прежний запрос отменяется
- [ ] состояние: ряд, итоги, дельты (`null` без `has_previous_data`), категории обоих типов, раздельные ошибки ряда и итогов; `revalidate` при смене дня
- [ ] тесты: границы каждого периода на 1-е, 31-е и 29 февраля; «Год» = первая и последняя корзина `monthly`; смена периода шлёт один `summary` и не трогает `monthly`; быстрая двойная смена — ответ последней; ошибка `summary` сохраняет ряд
- [ ] `make -C android check`

### Task 7: Экран, график, вход и возврат

**Files:**
- Create: `ui/overview/OverviewScreen.kt`, `ui/overview/MonthlyBars.kt`, `OverviewScreenTest.kt`, `MonthlyBarsTest.kt`
- Modify: `AppScreen.kt`, `MainActivity.kt`, `ui/home/HomeScreen.kt`, `strings.xml`, `AppScreenSaverTest.kt`, `HomeScreenTest.kt`, `AppRootImportTest.kt`

- [ ] `barGeometry(months)`: общий масштаб, нулевой месяц — нулевая высота, ряд из нулей не делит на ноль
- [ ] `MonthlyBars`: парные столбики цветами дохода и расхода, месяцы вне периода приглушены, тап выбирает месяц; `contentDescription` пары — месяц и обе суммы
- [ ] экран по макету: `ChipRow`, итоги в стиле `Totals`, подпись дельты, списки категорий (`groupedRow`), пустой период, ошибка итогов на месте
- [ ] `AppScreen.Overview(period)` и поле `overview: OverviewPeriod?` у `AppScreen.Transactions` — десятое поле `Saver`'а; `Overview` входит в запоминаемые экраны задачи 2
- [ ] `MainActivity`: ветка экрана; `BackHandler` на «Главную»; модель `overview-${user.id}-$epoch`; «открыт снаружи» = `reconciliation != null || overview != null` для `drillFrom` и `dropDrill` (:192, :315-328); «назад» из расшифровки — в «Обзор» с тем же периодом; ветка в `when` share-импорта (:185-212), как у `Home`
- [ ] пятый флаг `overviewStale`: взводится там же, где `homeStale` (:709, :750), гасится `LifecycleResumeEffect` «Обзора»
- [ ] вход: строка «Обзор ›» под `Totals` на «Главной» (в пустом состоянии её нет)
- [ ] тесты: геометрия; Compose — чип зовёт модель, тап по категории и по столбику зовут колбэки с верными аргументами, пустой период, ошибка итогов не прячет график
- [ ] тесты `AppRoot` и `Saver`: «назад» из расшифровки → «Обзор» с тем же периодом; вкладка «Операции» после расшифровки открывается без фильтра; после «Обзора» «Главная» всё ещё перечитывается; share на «Обзоре» открывает распознавание; круг `Transactions` с `RANGE` + категорией + `overview`; `Overview` с каждым видом периода
- [ ] `make -C android check`

### Task 8: Версия и документация

**Files:**
- Modify: `android/gradle/libs.versions.toml`, `android/CLAUDE.md`, `docs/backlog.md`, `CLAUDE.md`

- [ ] `appVersionName = "0.11.0"`, `appVersionCode = "11"`
- [ ] `android/CLAUDE.md`: «Обзор» — не корень панели; `monthly` без параметров и почему; `RANGE` — состояние «открыт снаружи», расшифровка несёт даты `summary`; дельта — к отрезку той же длины; **пять** флагов устаревания
- [ ] `docs/backlog.md:46`: «Обзор» закрыт, мультивыбор и `bulkDeleteTransactions` остаются
- [ ] `CLAUDE.md`: «Releases» — фактические теги (`v0.8.0`, `app-v0.10.0`; `v0.5.0`–`v0.7.0` и `app-v0.6.0`–`app-v0.9.0` не нарезались: серверный тег выкатывает образ, на старый коммит его ставить нельзя); «Current direction» — план 19
- [ ] `make -C android check`

### Task 9: Verify acceptance criteria

- [ ] все пункты Overview реализованы; `make -C android check`
- [ ] сценарии Post-Completion выполнимы на релизной сборке без правок кода

### Task 10: [Final] Update documentation

- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Журнал, на телефоне (релизная сборка, R8):**
1. Импорт → дождаться строк → поправить категорию → из фона `adb shell am kill tech.shatrov.familyfinances` →
   вернуться: экран распознавания на месте, правка цела, запросов `recognize` в логе сервера нет.
2. Сохранить две строки из пяти → смахнуть → запуск: плашка «5 строк, сохранено 2» → «Продолжить» → досохранить;
   дублей в операциях нет.
3. Убить во время распознавания → «Распознавание прервано» → «Повторить» → строки.
4. «Удалить» на плашке и выход из аккаунта → каталог `files/import` пуст (`adb shell run-as`).

**«Обзор»:** итоги «Этого месяца» совпадают с «Главной»; у «Года» сумма столбиков равна итогам; тап по категории
за «3 месяца» открывает операции, сумма которых равна строке; «назад» возвращает в «Обзор» с тем же периодом;
поворот и убийство процесса сохраняют период.

**Не проверено codex** (лимит 19.09.2026): можно ли не платить второй раз за повтор распознавания после смерти
посреди вызова — сервер результат не хранит; осмысленна ли дельта у «Года».
