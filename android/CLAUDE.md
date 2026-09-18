# CLAUDE.md — Android-клиент

Онлайн-клиент к `/api/v1` (`docs/specs/005-api-only-redesign.md`, решение A-13: живёт в этом же
репозитории). Ни Room, ни очередей, ни синхронизации: нет сети — экран говорит об этом и предлагает
повторить.

## Команды

```bash
make -C android check      # = fmt-check test lint r8-check; офлайновая, генерации в ней нет
make -C android compile    # assembleDebug
make -C android test       # testDebugUnitTest (Robolectric)
make -C android fmt        # ktlintFormat
make -C android r8-check   # R8 без подписи + сверка mapping.txt с моделями :core:api
make -C android apk        # assembleRelease, нужен keystore
make -C android api-gen    # перегенерация клиента из docs/api/openapi.yaml (нужна сеть)
make -C android api-check  # api-gen + git status по каталогу вывода
```

Один тест: `./gradlew :app:testDebugUnitTest --tests '*LoginViewModelTest*'` из `android/`.

Локально нужны JDK 21+ и Android SDK с `android-37.0` и build-tools 37.

## Модули

Два, а не одиннадцать как у соседнего `field-engine`: офлайн-движка нет, делить нечего.

- `:core:api` — единственный модуль, знающий о сети: сгенерированные модели и Retrofit-интерфейсы
  (`generated/kotlin`), транспорт (`net/`), хранилище токена (`auth/`), `ApiGraph` как их
  composition root. Подробности — `core/api/CLAUDE.md`.
- `:app` — Compose-экраны и ViewModel'и (`ui/`), ручная навигация, `AppGraph`, `Session`, тема.

Пакет и namespace — `tech.shatrov.familyfinances`.

Тема и метрики (`ui/theme`) — копия из `core/ui` соседа, а не модуль: экранов мало, второй
потребитель не предвидится, а лишний модуль в Gradle стоит времени сборки.

Иконки — вручную перенесённые контуры Lucide в `ui/AppIcons.kt`: `material-icons-core` не покрывает
нужный набор в одном стиле, а `material-icons-extended` тянет тысячи векторов ради десятка. Следующая
иконка — ещё один `icon(name, path…)`, а не новая зависимость. Иконка приложения адаптивная
(`mipmap-anydpi-v26/ic_launcher.xml` + `drawable/ic_launcher_foreground.xml`), legacy-растров нет:
`minSdk 26` рисует адаптивную везде.

Навигационной библиотеки нет: sealed-класс `AppScreen` и `rememberSaveable`; системную «назад»
каждый неглавный экран ловит `BackHandler`-ом. Четыре корня (главная, операции, категории, бюджеты)
переключает нижняя `AppNavBar`, которую подставляет `AppRoot`, а не экраны: формы её не показывают,
и уход с категорий по вкладке помечает список, главную и бюджеты устаревшими так же, как «назад».
Устаревание держат три флага в `AppRoot` — `listStale`, `homeStale`, `budgetsStale`: экран при
возврате либо перечитывает всё (`refresh`), либо только проверяет смену дня (`revalidate`). Форма
операции взводит все три (`spent` бюджета считает сервер), форма бюджета — `budgetsStale` и
`homeStale`. DI ручной — `AppGraph.create(context)`, без Hilt и
Koin, и граф лежит в `FamilyFinancesApp`, а не в активити: из `onCreate` он пересоздавался бы на
каждый поворот, обнуляя сессию, а пережившие поворот ViewModel остались бы с прошлым клиентом.
По той же причине модели экранов ключуются по `user.id`: они переживают выход, и роль прошлого
пользователя показала бы следующему чужие действия.

Настройки — вложенный хост, а не пятая вкладка: `AppRoot` знает одну ветку
`AppScreen.Settings(page)`, а `ui/settings/SettingsHost.kt` — свой `when` по `SettingsPage` и свою
«назад». Входят с главной иконкой профиля в шапке, где раньше была «Выйти». Модели подразделов
живут в своём store (`ScopedModels` с ключом `settings`, второй экземпляр держит формы), который
`AppRoot` чистит при **каждой** смене страницы и при
уходе из настроек: флагов устаревания внутри настроек нет, повторный заход читает заново. Пока
страница отправляет мутацию, уход из неё заблокирован — очистка store отменила бы корутину,
которую сервер уже мог применить.

`403` в подразделе значит, что роль сняли с другого телефона: модель ставит `forbidden`, хост
закрывает страницу в корень, а тот перечитает сессию и уберёт из списка то, чего больше нет. Текст
такого отказа свой (`settings_forbidden`) — серверный говорит про права, а не про то, что случилось.
Коды `409`, чей серверный текст на экране не годится (`EMAIL_TAKEN`, `LAST_ADMIN`,
`CANNOT_DEACTIVATE_SELF`, `CURRENCY_LOCKED`), переводит карта `ui/settings/SettingsConflicts.kt`
через `toUiError(known)`; остальные показываются как есть.

Корень настроек перечитывает сессию через `AppGraph.refreshSession()`: при любом отказе остаётся
прежняя `Session` (обнулить её значило бы уйти на экран загрузки из-за сети), а поздний ответ не
затирает более свежую публикацию — за этим следит счётчик поколений в `AppGraph`, который двигают
`bootstrap`, `signOut` и `update(user)`/`update(family)`. Последние два кладут в `Session` ответ
`PUT` без бутстрапа; смена `zone`, `currency` или `role` поднимает `epoch`, смена имени или почты —
только `listStale` (в операциях кешируются имена авторов).

`Session` держит роль и валюту: после логина и при каждом старте с сохранённым токеном клиент
делает `GET /me` и `GET /family`. Без валюты главная не форматирует суммы (в `stats/summary` её
нет), без роли категории не прячут удаление от `member`. Имя `Session` занято этим состоянием, поэтому
сгенерированная модель сессии импортируется как `ApiSession` (`ui/settings/SessionsViewModel.kt`).

Токен — один непрозрачный с `expires_at`, пары access/refresh нет: у сервера обновления нет, а
`401` означает «сессия кончилась» и ведёт на экран входа. Единственное исключение — `401` на
`PUT /api/v1/me/password` с `error.code == INVALID_CREDENTIALS`: это неверный текущий пароль, а не
конец сессии. Код читается из копии тела (`response.peekBody`), оригинал достаётся `ApiClient`;
путь целиком из интерцептора не исключён — запрос должен уходить с токеном, а просроченный токен
чиститься. Логин шлёт `device_name = Build.MODEL` (до 64 символов), чтобы список сессий различал
телефоны; сессии, созданные до этого, остаются «Без имени» до перелогина. `409 SETUP_REQUIRED` — состояние сервера,
показывается отдельным текстом, а не как ошибка ввода.

Базовый адрес — `https://ffs.shatrov.tech/` со слэшем и без `/api/v1` (пути в сгенерированных
интерфейсах полные). Живёт в `buildConfigField` модуля `:app` и приходит в `ApiGraph` параметром:
`BuildConfig` у каждого модуля свой, из `:core:api` его не прочитать.

## Общие элементы экранов

Роль, не заданная в `appColorScheme` (`theme/Theme.kt`), берётся из baseline-палитры M3 — фиолетовой:
`secondaryContainer`/`onSecondaryContainer` заданы поэтому, их читают `FilterChip`, `SegmentedButton` и
дорожка `LinearProgressIndicator`. Контраст `elevated` к `canvas` ~1,2:1, поэтому выбранность держит
обводка `action` (`ui/Chips.kt`, `ui/Segments.kt`), а не заливка. Новый M3-компонент сначала проверяют
на его ролях.

Взаимоисключающий выбор из 2–3 вариантов — `SegmentedChoice` (`ui/Segments.kt`): тип операции, фильтр
бюджетов, фильтр типа операций. Чипы (`ChipRow`, `ui/Chips.kt`) остаются там, где вариантов больше
или выбор не бинарный: подписи в сегментах не переносятся, и на 360dp четыре сегмента оставляют под
текст ~33dp — период бюджета поэтому чипами.

Пароль — всегда `SecretField` (`ui/SecretField.kt`), включая вход: переключатель видимости и
`KeyboardType.Password` живут в одном месте, второго такого поля не заводим.

FAB подставляет `WithNavBar` в `MainActivity` поверх содержимого, как и панель вкладок — экраны о нём
не знают; сейчас он только на «Операциях». Список под ним заканчивается `contentPadding`
`Dimens.FAB_CLEARANCE` (88dp = 56 + 2×16), иначе кнопка накрывает последнюю строку и `Retry` подвала.

Фильтры операций и справочник категорий для листа — отдельные `StateFlow` в `TransactionsViewModel`,
а не поля `TransactionsUiState`: ряды фильтров рисуются и на загрузке, и на отказе, а в
`data object Loading` их не втащить. Так же устроен `BudgetsViewModel.filter`.

Признак повторения бюджета — два ресурса: `budget_recurring_toggle` на форме и
`budget_recurring_badge` в `contentDescription` значка строки; одним текстом обе роли не покрываются.

## Правки через `PUT`

Форма шлёт только изменённые поля: `Update*Request` собирается из diff с загруженным объектом, и
без изменений «Сохранить» выключено (`minProperties: 1` в контракте). Непереданные поля не
попадают в тело благодаря `explicitNulls = false` (`core/api/.../net/ApiClient.kt`) — иначе ушли бы
как `null` и затёрли бы данные.

Обратная сторона: `null` на провод не уходит вовсе, поэтому «убрать счёт у операции» нельзя выразить
как `account_id: null` — сервер читает отсутствие поля как «не трогать». Отвязка — отдельный флаг
`clear_account: true` в `UpdateTransactionRequest`, пункт «Без счёта» в листе выбора шлёт его.

`is_active` бюджета клиент не шлёт и не может: поля нет в `UpdateBudgetRequest` контракта. В ответе
он всегда `true` — удалённый бюджет сервер не отдаёт ни списком, ни по id.

Коды `409` бюджета переводит своя карта в `ui/budgets/BudgetEditViewModel.kt` (`budgetConflicts`), и
незнакомый `409` падает там на общий `budget_error_rejected`, а не показывается серверным текстом:
на форме английский ответ сервера не годится ни в одном случае.

## Распознавание скриншотов

Картинки приходят share-интентом (`SEND`/`SEND_MULTIPLE`) или листом `ImportSourceSheet` на «Операциях»
(галерея, камера) и попадают в `AppGraph.imports: ImportStore`. `offer` публикует импорт в `pending`,
`claim(id)` отдаёт его ровно одному `RecognizeViewModel`, второй получает `null` и показывает
`recognize_import_lost`. `MainActivity` предлагает intent из `onCreate` только при
`savedInstanceState == null`, из `onNewIntent` — всегда, и снимает `EXTRA_STREAM`: пересозданная активити
и запуск из «недавних» тот же share второй раз не откроют. `AppRoot` только маршрутизирует по `pending`,
пока открыты формы или настройки — ждёт; лаунчеры галереи и камеры живут в нём же, чтобы результат
камеры пережил смерть процесса.

Второй share. Пока модель готовит, распознаёт или сохраняет, она держит импорт (`hold`/`release`), и
новый не публикуется в `pending` — экран показывает «откроется после сохранения» по `waiting`. Share,
дождавшийся непустого ответа, не отпускает удержание до конца сохранения или ухода с экрана: иначе он
вытеснил бы оплаченный результат, не показав его. Сохранение, пока share ждёт, не отпускает его и при строке с
отказом — до её успешного повтора или ухода. Пришедший уже в просмотре, на отказе или на пустом ответе
вытесняет текущий: `AppRoot` чистит store форм, `onCleared` удаляет файлы. Невзятое предложение вытесняется
следующим `offer`.

Файлы. URI читаются в `init` сразу — права на чужой контент живут не дольше выдавшей их задачи — и
пережимаются в JPEG (`ImportFiles.prepare`, ≤ 5 штук, ≤ 2048 px, ≤ 2 МиБ) в `cacheDir/import/<importId>/`.
`ImportFiles.sweep` на старте приложения удаляет всё, кроме `camera/`: снимок камеры обязан пережить смерть
процесса, пока камера открыта, поэтому он чистится по возрасту (сутки). Наружу через `FileProvider`
(`${applicationId}.files`) отдаётся только `camera/`.

Клиент никогда не повторяет распознавание сам: вызов платный и не идемпотентен. Запрос идёт через
`ApiGraph.recognize` на отдельном клиенте `ApiClient.createLongCall` (read 210 с, без
`retryOnConnectionFailure`), модель зовёт его один раз в `init` и снова — только по «Повторить».
Справочник категорий грузится до платного вызова и «Повторить» его не перечитывает.

Сохранение. Каждая строка получает `draft` UUID — `id` её `POST /transactions`. «Сохранить N» шлёт выбранные
строки по очереди; любой отказ, кроме `422`, включая обрыв после отправки, — `Failed` с «Повторить», который шлёт
тот же `POST` с тем же `draft`: `201` — создано, `200` — запись уже была, и строка принимает поля из ответа.
`422` оставляет строку `Pending` с ошибками полей. `Saved` заперты от правки и повторной отправки, а
`savedCount` взводит три флага устаревания. Гарантия «без дублей» действует только в живом процессе: смерть
процесса теряет импорт вместе с `draft`-UUID и статусами строк, журнала нет.

## Версии

Только через `gradle/libs.versions.toml` — строковых версий в build-файлах не появляется, включая
`compileSdk`/`minSdk`/`jvmTarget`.

AGP 9 несёт встроенный Kotlin более старой версии, поэтому плагин `org.jetbrains.kotlin.android` не
подключается: KGP и serialization идут `classpath` в корневом `build.gradle.kts`, а в модулях —
`apply(plugin = "…")` без версии.

ktlint запускается `JavaExec` на своей конфигурации, а не сторонним плагином: версия тогда живёт в
каталоге, а совместимость с AGP 9 ни от кого не зависит. Нужен shadow-вариант артефакта (в обычном
clikt объявлен не-транзитивно).

## Генерация вне `check`

`generateApiClient` тянет генератор и его зависимости из сети, а `compile` и `check` обязаны
работать офлайн. Поэтому вывод коммитится, а свежесть проверяют отдельная цель `api-check` и
отдельный шаг CI: правка `openapi.yaml` без перегенерации в том же коммите валит `android:api-check`.

## Тесты

Robolectric в `src/test` — в `androidTest` они не попали бы в CI (раннеру нужен `/dev/kvm`).
Сеть — `MockWebServer`, живого сервера в тестах нет. При `compileSdk 37` нужен `@Config(sdk = [36])`
и `unitTests.isIncludeAndroidResources = true`. Каталог maven-репозитория Robolectric выносится
переменной `ROBOLECTRIC_M2_REPO` ради кеша CI: `android-all` весит ~100 МБ.

Compose-тестами покрыты экраны с вводом и экраны, где по клику происходит необратимое (вход, список и
форма транзакции, список и форма бюджета, настройки: корень, профиль, пароль, форма пользователя, семья,
устройства, резервные копии). Главная и категории добавились планом 13: ввода у них по-прежнему нет, но
пустое состояние ведёт кнопкой, а её потерю ViewModel-тест не заметит.

## Подпись и выкладка

Один постоянный keystore, тот же на ноутбуке (`~/.android/ffs-installer.jks`, путь и пароль —
через `Makefile`) и в CI: защищённая файловая переменная `FFS_KEYSTORE` хранит его в base64
(значение переменной — текст), джоба раскодирует её перед сборкой. Смена сертификата даёт
`INSTALL_FAILED_UPDATE_INCOMPATIBLE` и требует сносить приложение вместе с данными; нет keystore —
`assembleRelease` падает, а не подписывается чем попало. В дерево репозитория файл не кладётся.

Тесты минифицированный код не видят: модель, на которую ссылается только generic-сигнатура
Retrofit, R8 однажды удалил, и приложение падало на телефоне при зелёном CI. Поэтому
`core/api/consumer-rules.pro` держит все `@Serializable`-модели, а `checkReleaseModelsKept`
(`make r8-check`, часть `check` и `assembleRelease`) сверяет `mapping.txt` со сгенерированными
моделями — R8 для этого подписи не требует.

Стора не будет: APK ставится с ноутбука. Тег `app-vX.Y.Z` собирает APK в CI и кладёт его в
артефакты на неделю; серверные теги `vX.Y.Z` Android-джоб не запускают.

Релиз приложения: поднять `appVersionCode` и `appVersionName` в `gradle/libs.versions.toml`,
смержить, поставить тег. Тег и `appVersionName` не связаны ничем, кроме рук; `versionCode` меньше
установленного система отвергает, равный — ставит молча поверх, и телефон остаётся на старой сборке.

**Окно несовместимости.** Сервер выкатывается сам с каждого мержа в `main`, телефоны обновляются
руками: ломающая правка контракта роняет установленные APK. Порядок — правка аддитивна, а если
ломает, то сначала APK на оба телефона, потом мерж серверной части.

Обратный случай — новое обязательное поле ответа: сгенерированная модель без значения по умолчанию
не разберёт ответ старого сервера, поэтому такой APK ставят **после** выката сервера (план 12:
`v0.3.0`, затем `app-v0.5.0`). Откат сервера после этого ломает клиент.

Клиент `0.8.0` (план 16) требует сервер `v0.6.0`: без него нет `/accounts` и
`/stats/reconciliation`. Клиенты `0.6`/`0.7` с `v0.6.0` работают — они не шлют `account_id`, и
счёт операции при их правке сохраняется.

## CI

Versions live only in `android/gradle/libs.versions.toml`, `compileSdk`/`minSdk`/`jvmTarget` included.

CI (`.gitlab-ci.yml`): `android:check`, `android:api-check`, `android:apk` — all three in the trailing
`android` stage with `needs: []`, so they start at once and nothing waits for them: a red client check or an
unreachable Maven Central must not hold back the server deploy. All three `extends: .android`,
which overrides `image` **and** replaces the `default:` `before_script` (it would otherwise hand the job
`golang:1.26`, `go version` and the Go cache paths) with the JDK/SDK setup. Rules come from `.android-rules`: the branch `main` and merge
requests with `changes: [android/**/*, docs/api/openapi.yaml]`, plus the tag `app-vX.Y.Z`; a server tag
`vX.Y.Z` is excluded explicitly, because `changes:` is always true in a tag pipeline. Android SDK and
the Gradle/Robolectric caches sit in `/ci-cache/android/*`, like every other cache here — the `cache:`
mechanism is unused. All three share one `GRADLE_USER_HOME`, which Gradle locks: they carry
`resource_group: android` so they never run at the same time — including against the pipeline of another
branch, which is how they first failed. `android:apk` runs **only** on an `app-vX.Y.Z` tag and signs with
the same keystore as the laptop: `FFS_KEYSTORE` is a protected *file* variable holding the **base64** of
the JKS (a CI variable is text; the job decodes it) and `FFS_KEYSTORE_PASSWORD` is protected and masked —
hence `app-v*` is a protected tag, or neither would reach the pipeline.
