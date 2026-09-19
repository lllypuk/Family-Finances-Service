# План 19. Журнал незавершённого импорта скриншотов

## Overview

Смерть процесса сейчас теряет импорт целиком: картинки, оплаченный ответ распознавания, правки строк и их
`draft`-UUID — то есть и деньги, и гарантию «без дублей» (`docs/backlog.md:172`, приёмка плана 15). Типичный
случай — ушли в банк за следующим скриншотом, Android убил приложение в фоне.

- Импорт ведёт журнал на диске и восстанавливается из него без сети и без повторного платного вызова.
- Убитый системой процесс возвращается прямо в экран распознавания; после смахивания или падения —
  плашка на «Главной»: «Продолжить / Удалить».
- Только клиент, `0.11.0`. Контракт и сервер не меняются.

Не входит: ждущий второй share (права на его URI умирают с процессом), журнал нескольких импортов сразу,
синхронизация незавершённого импорта между телефонами.

## Context (from discovery)

- `ImportStore.kt` — владение импортом в памяти: `offer` / `claim` / `hold` / `release`, потоки `pending`, `waiting`.
- `ui/recognize/RecognizeViewModel.kt`: `init` :151 (`claim == null` → `recognize_import_lost`), `recognize()`
  (платный вызов один, `retry()` :168 — только по кнопке), `send` / `saveRow` (`draft` = `id` у `POST`,
  `200` — запись уже была), `onCleared()` — `release` + `ImportFiles.discard`.
- `ui/recognize/ImportFiles.kt`: каталог `cacheDir/import/<importId>/` (`root` :99), `prepare` :54,
  `discard` :67, `sweep` :78 — на старте сносит всё, кроме `camera/` (`FamilyFinancesApp.kt:17`).
- `MainActivity.kt`: ветка `AppScreen.Recognize` :694 — модель в store `forms` с ключом `recognize-<importId>`,
  `leave` уводит в «Операции»; маршрутизация по `pending` :181–:224; «Главная» :264.
- `AppScreen.Recognize(importId)` сохраняется `Saver`'ом (`AppScreen.kt:124`): после убийства процесса экран
  восстанавливается сам, а данных под ним нет.
- В `:app` нет ни плагина сериализации, ни своих `@Serializable`-классов; плагин стоит `classpath` в корневом
  `build.gradle.kts:9` и применён в `core/api/build.gradle.kts:7`. Ответ распознавания — сгенерированный
  `RecognizeResult`, его сериализатор доступен из `:core:api`.
- `AndroidManifest.xml:10` — `allowBackup="false"`: журнал и скриншоты из `filesDir` в облачный бэкап не уходят.
  `res/xml/file_paths.xml` отдаёт наружу только `cache-path import/camera/` — перенос корня импорта в `filesDir`
  камеру не задевает.
- Тесты: `ImportStoreTest.kt`, `AppRootImportTest.kt`, `ui/recognize/RecognizeViewModelTest.kt`, `TestFixtures.kt`.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче.
- Каждая задача — `make -C android check`; сервер не трогается, `make test` не нужен.
- **CRITICAL: update this plan file when scope changes during implementation**

## Testing Strategy

- Robolectric: журнал (запись, чтение, повреждённый файл, версия), восстановление модели по фазам, плашка.
- Смерть процесса в тесте — новая модель с тем же `importId` над тем же каталогом при пустом `ImportStore`.
- Compose-тест плашки: «Удалить» необратимо, значит с подтверждением.

## Progress Tracking

- `[x]` сразу по выполнении; новое — `➕`, блокеры — `⚠️`.

## Solution Overview

**Журнал — файл рядом с картинками, в `filesDir`.** `filesDir/import/<importId>/journal.json` и `N.jpg`:
`cacheDir` система чистит сама, а журнал — это оплаченный результат. `camera/` остаётся в `cacheDir` — её
отдаёт `FileProvider`. Запись атомарная: временный файл и `renameTo`.

**Что пишется и когда.** После подготовки картинок — список файлов; перед платным вызовом — отметка
`recognizing`; после ответа — сырой `RecognizeResult` (главное: вызов платный); дальше — при каждой правке
строки, выборе счёта и смене статуса. Справочники не пишутся: они перечитываются с сервера.

**Восстановление по фазе смерти:**

| Умер во время | После восстановления |
|---|---|
| подготовки картинок | импорт потерян, как сейчас: URI уже недоступны, журнал удаляется |
| распознавания (`recognizing`, ответа нет) | отказ с «Повторить», картинки на месте; сам клиент вызов не повторяет — исход платного запроса неизвестен |
| просмотра | экран как был: строки, правки, счёт, статусы; платного вызова нет |
| сохранения строки (`saving`) | строка `Failed` с «Повторить»: тот же `POST` с тем же `draft`, сервер ответит `200` или `201` — дубля нет |

**`discard` уходит из `onCleared`.** Смахивание из «недавних» иногда успевает вызвать `onDestroy` с
`isFinishing`, и `onCleared` снёс бы журнал именно тогда, когда он нужен. Журнал удаляют только явные
события: уход с экрана (`leave`), вытеснение новым share, «Удалить» на плашке, полное сохранение; `onCleared`
только отпускает удержание.

**Два пути возврата.** Системное убийство: `Saver` вернёт `Recognize(importId)`, `claim` ответит `null`, модель
ищет журнал — нашла, восстановилась. Смахивание и падение: сохранённого состояния нет, приложение стартует
на «Главной», и там плашка. Один импорт — один журнал: новый `offer` вытесняет прежний журнал так же, как
сейчас вытесняет невзятое предложение.

**Срок жизни — сутки.** `sweep` на старте удаляет каталоги без журнала, с нечитаемым журналом, с журналом другой
версии формата и старше 24 часов. Банковские скриншоты не должны лежать на диске бессрочно.

**Сериализация — плагин в `:app`.** Свои `@Serializable`-классы журнала плюс сгенерированный `RecognizeResult`
внутри. R8 тесты не видят (`android/CLAUDE.md`, «Подпись и выкладка»): `checkReleaseModelsKept` дополняется
классами журнала, а восстановление проверяется на телефоне релизной сборкой.

## Technical Details

```kotlin
@Serializable
data class ImportJournal(
    val version: Int,              // JOURNAL_VERSION = 1; другая версия — журнал выбрасывается
    val importId: String,
    val createdAt: Long,
    val images: List<JournalImage>, // имя файла или причина отказа; dropped — отдельным полем
    val dropped: Int,
    val recognizing: Boolean,       // платный вызов ушёл, ответа ещё нет
    val result: RecognizeResult?,   // сырой ответ сервера
    val accountId: String?,
    val rows: List<JournalRow>,     // наложение на result.items по индексу
    val savedCount: Int,
)

@Serializable
data class JournalRow(
    val draft: String, val included: Boolean, val date: String?, val categoryId: String?,
    val description: String, val status: String, // pending | saving | saved | failed
)
```

Строка экрана = `result.items[i].toRow(...)` с наложением `JournalRow`; ошибки полей и текст отказа не
пишутся — `Failed` после восстановления получает общий текст «сохранение прервано». `saving` при чтении
превращается в `failed`.

Плашка на «Главной» — над `Totals`, видна и в пустом состоянии (`HomeScreen.kt:104`):

```
┌──────────────────────────────────────────┐
│ Незавершённый импорт                     │
│ 5 строк, сохранено 2 · сегодня 14:20     │
│ [ Продолжить ]              [ Удалить ]  │
└──────────────────────────────────────────┘
```

Журнал в фазе `recognizing` без ответа: «Распознавание прервано · 3 скриншота». «Удалить» — с подтверждением.
Плашки нет, пока открыт экран распознавания того же импорта и пока `ImportStore.pending` непуст.

## Implementation Steps

### Task 1: `ImportJournal` — формат, запись, чтение, уборка

**Files:**
- Modify: `android/app/build.gradle.kts` (плагин сериализации), `ui/recognize/ImportFiles.kt`
- Create: `ui/recognize/ImportJournal.kt`, `app/src/test/…/ui/recognize/ImportJournalTest.kt`
- Modify: `app/src/test/…/ui/recognize/ImportFilesTest.kt` (если есть — иначе создать)

- [ ] `apply(plugin = "org.jetbrains.kotlin.plugin.serialization")` в `:app`; `Json` — свой экземпляр с `ignoreUnknownKeys`, не `ApiClient.json`
- [ ] `ImportFiles.root` → `filesDir/import`; `camera/` остаётся в `cacheDir` и отдаётся `FileProvider`'ом как раньше
- [ ] `ImportJournalStore`: `write(journal)` атомарно, `read(importId)`, `latest()`, `delete(importId)`; нечитаемый файл и чужая версия — `null`
- [ ] `sweep`: удалить каталоги без журнала, с нечитаемым, чужой версии, старше 24 ч; в старом `cacheDir/import` — всё, кроме `camera/`
- [ ] тесты: круг запись→чтение, обрыв записи не портит прежний журнал, мусор в файле, версия, `sweep` по каждому правилу, `latest` при двух каталогах
- [ ] `make -C android check`

### Task 2: Модель пишет журнал и восстанавливается из него

**Files:**
- Modify: `ui/recognize/RecognizeViewModel.kt`, `MainActivity.kt` (:694 — фабрика модели, `leave`), `AppGraph.kt` (журнал в графе)
- Modify: `RecognizeViewModelTest.kt`, `TestFixtures.kt`

- [ ] запись: после `prepare`, перед `api.recognize` (`recognizing = true`), после ответа, в `edit` / `replace` / `onAccountChange` / `saveRow`
- [ ] `init`: `claim == null` → `journal.read(importId)`; есть — восстановить по таблице фаз, нет — `recognize_import_lost`, как сейчас
- [ ] `recognize()` с готовым `result` в журнале платный вызов не делает: грузит справочники и строит `Review`; отказ справочников — `Failure` с «Повторить», который снова идёт этим путём
- [ ] `onCleared` — только `release`; `abandon()` — `discard` каталога с журналом; зовут его `leave` и вытеснение новым share в `AppRoot`; всё сохранено — журнал удаляется сразу
- [ ] восстановленная строка `saving` → `Failed` с «сохранение прервано»; её «Повторить» шлёт тот же `draft`
- [ ] тесты: восстановление из каждой фазы; после восстановления в `Review` к `recognize` нет ни одного запроса; повтор прерванной строки → `200` принимает поля сервера; `abandon` удаляет каталог, `onCleared` — нет; удержание `hold` у восстановленной модели
- [ ] `make -C android check`

### Task 3: Плашка на «Главной»

**Files:**
- Modify: `ui/home/HomeScreen.kt`, `HomeViewModel.kt`, `MainActivity.kt` (:264), `strings.xml`
- Modify: `HomeViewModelTest.kt`, `HomeScreenTest.kt`, `AppRootImportTest.kt`

- [ ] `HomeViewModel` отдаёт сводку `journal.latest()`: строк, сохранено, время, фаза; перечитывается при возврате на «Главную»
- [ ] плашка по макету, видна и в пустом состоянии; «Продолжить» → `AppScreen.Recognize(importId)`; «Удалить» → подтверждение → `delete`
- [ ] новый share при лежащем журнале вытесняет его (каталог удаляется) — как сейчас вытесняется невзятое предложение
- [ ] тесты ViewModel: журнала нет — плашки нет; журнал `recognizing` — свой текст; удаление убирает плашку
- [ ] Compose-тест: «Удалить» спрашивает подтверждение; «Продолжить» зовёт колбэк с `importId`
- [ ] `AppRootImportTest`: «Продолжить» открывает экран, и тот восстанавливается без `offer`
- [ ] `make -C android check`

### Task 4: R8, версия, документация

**Files:**
- Modify: `android/app/build.gradle.kts` (`checkReleaseModelsKept`), `android/gradle/libs.versions.toml`, `android/CLAUDE.md`, `docs/backlog.md`

- [ ] `checkReleaseModelsKept` проверяет и классы журнала; при необходимости — keep-правило в `app/proguard-rules.pro`
- [ ] `appVersionName = "0.11.0"`, `appVersionCode = "11"`
- [ ] `android/CLAUDE.md`, «Распознавание скриншотов»: журнал, таблица фаз, кто удаляет журнал и почему не `onCleared`, срок жизни; абзац «гарантия только в живом процессе» переписать
- [ ] `docs/backlog.md:172` — пункт закрыт
- [ ] `make -C android check`

### Task 5: Verify acceptance criteria

- [ ] все пункты Overview реализованы; `make -C android check`
- [ ] сценарий «убить процесс» на релизной сборке описан в Post-Completion и выполним без правок кода

### Task 6: [Final] Update documentation

- [ ] `CLAUDE.md` «Current direction»: план 19, клиент `0.11.0`
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Проверка на телефоне (релизная сборка, R8):**
1. Импорт → дождаться строк → поправить категорию → `adb shell am kill tech.shatrov.familyfinances` из фона →
   вернуться: экран на месте, правка цела, запросов `recognize` в логе сервера нет.
2. Сохранить две строки из пяти → смахнуть приложение → запуск: плашка «5 строк, сохранено 2» → «Продолжить»
   → досохранить; в операциях нет дублей.
3. Убить во время распознавания → «Распознавание прервано» → «Повторить» → строки.
4. «Удалить» на плашке → каталог `files/import` пуст (`adb shell run-as`).
