# План 06 — Android-клиент

Первый план после переезда на GitLab. Клиент живёт в `android/` этого же репозитория
([spec 005](../specs/005-api-only-redesign.md), решение A-13) и строится по образцу соседнего
проекта `shatrov.tech/field-engine`, из которого берётся всё, кроме офлайна.

## Overview

- Онлайн-клиент к `/api/v1`: без Room, без очередей, без синхронизации. Нет сети — экран говорит
  об этом и предлагает повторить.
- v1 закрывает ежедневный сценарий: вход, главная, транзакции, категории. Бюджеты, отчёты,
  пользователи, сессии и бэкапы — следующим планом.
- Модели и типизированные интерфейсы генерируются из `docs/api/openapi.yaml`: контракт в
  репозитории один, вендоренной копии и пина нет (A-13).
- Собирается и проверяется в том же пайплайне; APK ставится с ноутбука, стора не будет.

## Context (from discovery)

Из `field-engine` (проверено агентом 07.09.2026, пути — там):

- Версии только через `gradle/libs.versions.toml`. AGP `9.3.2`, Kotlin `2.3.21`, KSP `2.3.11`,
  `compileSdk`/`targetSdk` 37, `minSdk` 26, `jvmTarget` 17, Compose BOM `2026.08.00`,
  kotlinx-serialization `1.11.0`, OkHttp `5.5.0`, Robolectric `4.16.1`, openapi-generator-cli
  `7.24.0`.
- AGP 9 несёт встроенный Kotlin: плагин `org.jetbrains.kotlin.android` не подключается, KGP,
  serialization и KSP идут `classpath` в корневом `build.gradle.kts`, в модулях — `apply(plugin=…)`
  без версии.
- UI — Compose + Material3. Навигационной библиотеки нет: sealed-класс экранов и `rememberSaveable`.
- DI ручной: `*Graph.create(context)` как composition root, без Hilt и Koin.
- Токен: `TokenVault`-шов и `KeystoreTokenVault` — AES/GCM, ключ в Android Keystore, шифротекст в
  приватном `SharedPreferences`, запись через `commit()` с проверкой. Чистые части (упаковка
  шифротекста, перевыпуск ключа) покрыты тестами без Keystore.
- **Генерация не входит ни в `compile`, ни в `check`**: она тянет артефакты из сети, а обе цели
  обязаны работать офлайн. Сверка свежести — отдельная цель `api-check` и отдельный шаг джобы.
- Тесты — Robolectric в `src/test` (в `androidTest` они не попали бы в CI, там нет `/dev/kvm`),
  протокол — `okhttp-mockwebserver`. Нужны `unitTests.isIncludeAndroidResources = true` и
  `@Config(sdk=[36])` при `compileSdk 37`; каталог maven-репозитория Robolectric выносится
  переменной ради кеша.
- Релизная сборка подписана явным `signingConfig`, а не «как-нибудь».
- ktlint запускается внешним бинарём и в CI не гоняется вовсе.

Из этого репозитория:

- Контракт — `docs/api/openapi.yaml`, 25 путей, ~40 операций; парность роутов и операций держит
  `tests/integration/openapi_coverage_test.go`. Схемы тел не сверяются — ограничение известно.
- Конверт ответа свой, не RFC 9457: `{data|error:{code,message,details[]},meta:{…}}`. У `/health`
  конверта нет вовсе — это голая схема `Health`.
- Главная собирается одним запросом: `GET /api/v1/stats/summary`. **Валюты в нём нет** — ни в
  `StatsSummary`, ни в `PeriodTotals`; она только в `Family`. Значит форматирование сумм требует
  ещё и `GET /api/v1/family`.
- Роль пользователя приходит из `GET /api/v1/me` и решает, показывать ли удаление категорий.
- Деньги — `amount_minor` (целое в минимальных единицах), даты — `YYYY-MM-DD` без времени и зоны.
- `.gitlab-ci.yml`: **каждый мерж в `main` собирает образ и выкатывает прод**; тег `vX.Y.Z` делает
  то же самое с именем версии. Правил `changes:` у Go-джоб нет сознательно (A-13). Раннер один,
  docker executor. Блок `default:` задаёт `image: golang:1.26` и `before_script` с каталогами
  Go-кеша — любая Android-джоба обязана переопределить оба. Кеши лежат в смонтированном раннером
  `/ci-cache`, механизм `cache:` в этом репозитории не используется.
- Прод настроен 07.09.2026: семья и админ созданы, `/health` отвечает `setup_complete: true`.

Прогон генератора 7.24.0 на нашей спеке (07.09.2026, проверено, не предположение):

- `amount_minor` → `Long`, `date` → `LocalDate`, `created_at` → `OffsetDateTime`, `id` → `UUID`;
  все помечены `@Contextual` и требуют зарегистрированных сериализаторов. `meta.timestamp` есть в
  каждом ответе конверта, поэтому без них не разберётся даже логин.
- `Category.parent_id` (`oneOf: [uuid, null]`) даёт нормальный `UUID? = null` — обёртки нет.
- Пути в интерфейсах полные (`@POST("api/v1/auth/login")`), значит базовый адрес — корень хоста со
  слэшем на конце, без `/api/v1`.
- По умолчанию методы возвращают `Call<T>`, а пакет — `org.openapitools.client`, вывод —
  `<out>/src/main/kotlin`. Всё это меняется параметрами, которых в `docs/api/README.md` нет.
- Инлайновые конверты **в путях** генератор именует детерминированно по `operationId`
  (`ListTransactions200Response`, `Login200Response`). А шесть схем внутри `components/responses`
  (`UserOk`, `FamilyOk`, `CategoryOk`, `TransactionOk`, `BudgetOk`, `ReportOk`) стали
  `InlineObject`…`InlineObject5` — имена позиционные, и седьмая такая схема их сдвинет.

Локально: JDK 25, Android SDK с `android-37.0` и build-tools 37 в `~/Android/Sdk`, node для
генератора.

## Development Approach

- **testing approach**: Regular.
- Каждая задача заканчивается зелёным `make -C android check`. Эта цель офлайновая: формат, тесты,
  линт. Генерация в неё не входит.
- Тесты — Robolectric в `src/test`, сеть — `MockWebServer`. Живого сервера в тестах нет.
- Плагины и версии — только через каталог; строковых версий в build-файлах не появляется.

## Testing Strategy

- **unit**: разбор конверта ошибок, сериализаторы дат, пагинация, упаковка шифротекста токена,
  состояния ViewModel каждого экрана.
- **Compose**: экраны с формой и списком — вход, список транзакций, форма транзакции. Главная и
  категории проверяются через ViewModel: там нет ввода, а разметка меняется чаще логики.
- **свежесть генерации**: не тест, а цель `make -C android api-check` и шаг джобы: генератор
  тянет артефакты из сети, а `check` обязан работать офлайн.
- **e2e**: нет. Инструментальные тесты в CI не гоняются, эмулятор в пайплайне не поднимается.

## Progress Tracking

- `[x]` по завершении; ➕ новые задачи; ⚠️ блокеры.

## Solution Overview

Два модуля вместо одиннадцати у соседа: офлайн-движка нет, и делить нечего.

- `:core:api` — сгенерированные модели и интерфейсы, транспорт (OkHttp, `Authorization`, разбор
  конверта), хранилище токена. Единственный модуль, знающий о сети.
- `:app` — Compose-экраны, ViewModel'и, ручная навигация, composition root `AppGraph`, тема.

Тему и метрики забираем из `core/ui` соседа копией, а не модулем: экранов мало, второй потребитель
не предвидится, а лишний модуль в Gradle стоит времени сборки.

Токен хранится в Keystore по схеме соседа, но пара `access`/`refresh` схлопывается в один
непрозрачный токен с `expires_at`: у сервера обновления нет, а `401` означает «сессия кончилась» и
ведёт на экран входа. Отдельный `409 SETUP_REQUIRED` показывается текстом «сервис ещё не настроен» —
это состояние сервера, а не ошибка клиента.

**Бутстрап сессии.** После логина и при каждом старте с сохранённым токеном клиент делает `GET /me`
и `GET /family` и кладёт роль и валюту в `AppGraph`. Без этого главная не умеет форматировать
суммы, а категории — прятать удаление от роли `member`.

**Окно несовместимости.** Сервер выкатывается сам с каждого мержа в `main`, а телефоны обновляются
руками. Ломающая правка контракта роняет установленные APK, пока их не обновят. Порядок: правка
аддитивна, а если ломает — сначала APK на оба телефона, потом мерж серверной части.

## Technical Details

- Каталог: `android/` в корне репозитория; `settings.gradle.kts`, `gradle/libs.versions.toml`,
  враппер Gradle, `.editorconfig`, `Makefile` с целями `compile`, `test`, `lint`, `fmt`,
  `fmt-check`, `check`, `apk`, `api-gen`, `api-check`.
- Пакет: `tech.shatrov.familyfinances`, namespace модуля равен пакету.
- ktlint подключается **Gradle-плагином**, а не внешним бинарём: версия тогда идёт через каталог, а
  CI получает его вместе с проектом. У соседа он внешний, и поэтому в его пайплайне формат не
  проверяется вовсе — повторять это не будем.
- Генерация: `JavaExec` в `core/api/build.gradle.kts`, конфигурация `apiGenerator`. Вход —
  `rootProject.layout.projectDirectory.file("../docs/api/openapi.yaml")`: корень Gradle-проекта —
  `android/`, поэтому `..` выходит в репозиторий (относительный `../../docs` из каталога модуля
  указывал бы на `android/docs`, которого нет). Вход объявляется `inputs.file(…)` с
  `PathSensitivity.RELATIVE`, выход — `outputs.dir(…)`; каталог вывода очищается перед запуском,
  иначе удалённая из спеки схема остаётся закоммиченным мусором.
- Параметры генератора целиком, а не «как в README»:
  `library=jvm-retrofit2,serializationLibrary=kotlinx_serialization,dateLibrary=java8`,
  `useCoroutines=true` (иначе `Call<T>` вместо `suspend`),
  `useResponseAsReturnType=true` (иначе тело ошибки недоступно),
  `packageName/apiPackage/modelPackage=tech.shatrov.familyfinances.core.api`,
  `sourceFolder=kotlin` (иначе вывод ляжет в `generated/src/main/kotlin`),
  `--global-property apis,models,apiDocs=false,modelDocs=false,apiTests=false,modelTests=false`
  (иначе рядом окажутся чужие `build.gradle`, `README.md` и `docs/`).
- Сериализаторы `LocalDate`, `OffsetDateTime` и `UUID` регистрируются в `SerializersModule` того
  `Json`, что отдан конвертеру Retrofit.
- Базовый адрес — `https://ffs.shatrov.tech/` со слэшем и **без** `/api/v1`, один и тот же для
  отладочной и релизной сборки: отдельного тестового хоста не будет (решение владельца 07.09.2026).
  Значение живёт в `buildConfigField` модуля `:app` (`buildConfig = true`) и приходит в `ApiGraph`
  параметром — `BuildConfig` у каждого модуля свой, и читать его из `:core:api` нельзя. Поле
  заводится сразу, чтобы отладочная сборка могла смотреть в другой адрес, если это когда-нибудь
  понадобится.
- Подпись APK: один постоянный keystore, тот же на ноутбуке и в защищённой файловой переменной CI;
  путь задаётся переменной, в дерево репозитория файл не кладётся. Android не ставит неподписанный
  пакет, а смена сертификата даёт `INSTALL_FAILED_UPDATE_INCOMPATIBLE` при обновлении поверх. Если
  keystore недоступен — сборка падает, а не подписывается чем попало.
- CI: джобы переопределяют `image` и `before_script: []` (иначе наследуют `golang:1.26` и Go-кеш),
  ставят `git`, `curl`, `unzip`, держат SDK и кеши Gradle/Robolectric в `/ci-cache/android/*` —
  как остальной кеш этого репозитория, без механизма `cache:`.
- Правила джоб: `changes:` в тег-пайплайнах всегда истинно, поэтому серверный тег `vX.Y.Z` иначе
  запускал бы и Android. Нужны явные `if`: ветка `main` с `changes`, тег `app-v*` — без.

## Implementation Steps

### Task 1: Каркас Gradle, подпись и релизная сборка

**Files:**
- Create: `android/settings.gradle.kts`, `android/build.gradle.kts`, `android/gradle.properties`,
  `android/gradle/libs.versions.toml`, `android/.editorconfig`, `android/Makefile`,
  `android/gradlew`, `android/gradle/wrapper/*`, `android/app/build.gradle.kts`,
  `android/app/src/main/AndroidManifest.xml`,
  `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `.dockerignore`, `.gitignore`

- [x] каталог версий по образцу соседа: Compose BOM, Material3, kotlinx-serialization, OkHttp,
      Retrofit, Robolectric, ktlint-плагин; Room, WorkManager и CameraX не переносятся
- [x] корневой `build.gradle.kts` с `classpath` KGP/serialization и `apply(plugin=…)` без версий
- [x] модуль `:app`: Compose, Material3, `minSdk 26`, `compileSdk 37`, `buildConfig = true`,
      `testOptions { unitTests.isIncludeAndroidResources = true }`, пустой `MainActivity`
- [x] `android.permission.INTERNET` в манифесте — без него первый запрос отвечает отказом
      разрешения, а не сетевой ошибкой
- [x] `signingConfig` из переменной с путём к keystore; отсутствие файла роняет сборку
- [x] `Makefile`: `compile`, `test`, `lint`, `fmt`, `fmt-check`, `check` (= `fmt-check test lint`),
      `apk`; `JAVA_HOME` не прибивать к чужому пути, брать из окружения
- [x] `.dockerignore` получает `android/`; `.gitignore` — `android/.gradle`, `android/build`,
      `android/local.properties`, `*.apk`, `*.jks`, `*.keystore`
- [x] `make -C android check` и `make -C android apk` — зелёные (R8 на пустом приложении ловится
      здесь, а не после восьми задач кода)

### Task 2: Имена конвертов ответа в контракте

**Files:**
- Modify: `docs/api/openapi.yaml`, `docs/api/README.md`

- [x] шесть схем внутри `components/responses` (`UserOk`, `FamilyOk`, `CategoryOk`, `TransactionOk`,
      `BudgetOk`, `ReportOk`) вынести в `components/schemas` и подключить через `$ref`: сейчас
      генератор даёт им позиционные имена `InlineObject`…`InlineObject5`, и седьмая такая схема
      сдвинет нумерацию в уже закоммиченном коде. Конверты в путях трогать не нужно — они уже
      именуются по `operationId`
- [x] в `docs/api/README.md` заменить команду генератора на полный набор параметров из «Technical
      Details»: без них пакет и раскладка вывода не те, что объявлены здесь
- [x] `make test` — зелёный (тест покрытия роутов имена схем не проверяет, но правка не должна
      задеть операции)

### Task 3: Модуль `:core:api` и генерация клиента

**Files:**
- Create: `android/core/api/build.gradle.kts`, `android/core/api/CLAUDE.md`
- Create: `android/core/api/generated/kotlin/**` (результат генератора, коммитится)
- Modify: `android/settings.gradle.kts`, `android/gradle/libs.versions.toml`, `android/Makefile`

- [x] `JavaExec`-задача `generateApiClient` на конфигурации `apiGenerator`; вход и выход объявлены
      через `inputs.file`/`outputs.dir`, каталог вывода очищается перед запуском
- [x] полный набор параметров генератора (см. «Technical Details»)
- [x] подключить каталог вывода как `sourceSets["main"].kotlin.srcDir`
- [x] цели `api-gen` и `api-check` в `Makefile`: вторая запускает генерацию и падает, если
      `git -C <корень> status --porcelain -- android/core/api/generated` непусто (`git diff` новых
      файлов не видит). В `check` не входит: генератору нужна сеть
- [x] `make -C android api-check` — зелёный; `make -C android check` — зелёный

### Task 4: Транспорт и разбор конверта

**Files:**
- Create: `android/core/api/src/main/kotlin/.../net/ApiClient.kt`, `net/ApiFailure.kt`,
  `net/ApiSerializers.kt`, `net/ErrorEnvelope.kt`,
  `android/core/api/src/main/kotlin/.../ApiGraph.kt`
- Create: `android/core/api/src/test/kotlin/.../net/ApiFailureTest.kt`, `net/ApiClientTest.kt`,
  `net/ApiSerializersTest.kt`

- [ ] `ApiClient`: OkHttp + Retrofit, базовый адрес параметром, `Json` с `SerializersModule` для
      `LocalDate`, `OffsetDateTime`, `UUID`
- [ ] `ApiFailure` и разбор `Response<T>.errorBody()` в одной точке на все операции: `details[]`,
      `Retry-After`, `SETUP_REQUIRED`, сеть, `5xx`
- [ ] хелпер, снимающий конверт: вызов возвращает `data` или бросает `ApiFailure`
- [ ] тесты на MockWebServer: успех, 401, 403, 404, 422 с деталями, 429 с `Retry-After`, 409,
      обрыв сети
- [ ] тест: ответ с `meta.timestamp` и календарной датой разбирается (ловит незарегистрированный
      сериализатор — иначе падает первый же запрос)
- [ ] `make -C android check` — зелёный

### Task 5: Хранилище токена

**Files:**
- Create: `android/core/api/src/main/kotlin/.../auth/TokenVault.kt`,
  `auth/KeystoreTokenVault.kt`, `auth/CipherText.kt`
- Create: `android/core/api/src/test/kotlin/.../auth/CipherTextTest.kt`,
  `auth/InMemoryTokenVault.kt`, `auth/TokenInterceptorTest.kt`
- Modify: `android/core/api/src/main/kotlin/.../net/ApiClient.kt`

- [ ] `TokenVault`-шов и `KeystoreTokenVault` по образцу соседа: AES/GCM, ключ в Keystore,
      `commit()` с проверкой, хранится токен и `expires_at`, `toString()` замаскирован
- [ ] интерцептор подставляет `Authorization: Bearer` из хранилища
- [ ] `401` очищает хранилище и поднимает событие «сессия кончилась»
- [ ] тесты упаковки шифротекста и перевыпуска ключа — без Keystore, на чистых функциях
      (боевая реализация иначе остаётся вовсе непокрытой)
- [ ] тест интерцептора на реализации в памяти: заголовок подставлен, после 401 хранилище пусто
- [ ] `make -C android check` — зелёный

### Task 6: Вход, навигация и бутстрап сессии

**Files:**
- Create: `android/app/src/main/kotlin/.../AppGraph.kt`, `AppScreen.kt`, `Session.kt`, `theme/*`,
  `ui/login/LoginScreen.kt`, `ui/login/LoginViewModel.kt`
- Create: `android/app/src/test/kotlin/.../ui/login/LoginViewModelTest.kt`,
  `ui/login/LoginScreenTest.kt`, `SessionTest.kt`
- Modify: `android/app/src/main/kotlin/.../MainActivity.kt`, `android/app/build.gradle.kts`

- [ ] тема и метрики копией из `field-engine/core/ui`, без офлайн-компонентов
- [ ] `AppScreen` sealed + `rememberSaveable`-навигация; старт — вход или главная по наличию токена
      и непросроченному `expires_at`
- [ ] бутстрап сессии: `GET /me` и `GET /family` после логина и при старте с токеном, роль и валюта
      складываются в `AppGraph`; отказ бутстрапа возвращает на вход
- [ ] `LoginViewModel`: разные тексты для `INVALID_CREDENTIALS`, `RATE_LIMITED` с ожиданием и
      `SETUP_REQUIRED`
- [ ] выход из аккаунта: `POST /auth/logout`, очистка хранилища, возврат на экран входа
- [ ] тесты ViewModel и бутстрапа: успех, неверный пароль, лимит, сервис не настроен, протухший
      токен на старте
- [ ] Compose-тест экрана входа: поля, отключённая кнопка, показ ошибки
- [ ] `make -C android check` — зелёный

### Task 7: Главная

**Files:**
- Create: `android/app/src/main/kotlin/.../ui/home/HomeScreen.kt`, `ui/home/HomeViewModel.kt`,
  `ui/format/Money.kt`, `ui/format/Dates.kt`
- Create: `android/app/src/test/kotlin/.../ui/home/HomeViewModelTest.kt`, `ui/format/MoneyTest.kt`

- [ ] `HomeViewModel` на одном `GET /stats/summary`, период — текущий месяц; валюта берётся из
      сессии (в самой сводке её нет)
- [ ] форматирование денег из минорных единиц, без `Double` в коде
- [ ] карточки: доходы и расходы, дельты, топ категорий, прогресс бюджетов, последние транзакции
- [ ] состояния: загрузка, пусто, отказ сети с повтором
- [ ] тесты ViewModel на MockWebServer и тесты форматирования (ноль, минус, округление)
- [ ] `make -C android check` — зелёный

### Task 8: Список транзакций

**Files:**
- Create: `android/app/src/main/kotlin/.../ui/transactions/TransactionsScreen.kt`,
  `ui/transactions/TransactionsViewModel.kt`, `ui/transactions/Filters.kt`
- Create: `android/app/src/test/kotlin/.../ui/transactions/TransactionsViewModelTest.kt`,
  `ui/transactions/TransactionsScreenTest.kt`

- [ ] список с постраничной догрузкой по `meta.pagination`
- [ ] фильтры (период, тип, категория) уходят в query, а не фильтруются на клиенте
- [ ] группировка по дате, категория и автор в строке
- [ ] тесты: первая страница, догрузка, пустой ответ, отказ на второй странице
- [ ] Compose-тест: пустое состояние и список
- [ ] `make -C android check` — зелёный

### Task 9: Создание, правка и удаление транзакции

**Files:**
- Create: `android/app/src/main/kotlin/.../ui/transactions/TransactionEditScreen.kt`,
  `ui/transactions/TransactionEditViewModel.kt`
- Create: `android/app/src/test/kotlin/.../ui/transactions/TransactionEditViewModelTest.kt`,
  `ui/transactions/TransactionEditScreenTest.kt`

- [ ] форма: сумма целым, дата календарём, тип, категория, описание
- [ ] `POST` с клиентским UUID — повтор после обрыва не создаёт второй записи (идемпотентность
      сервера); `PUT` и `DELETE` для правки и удаления
- [ ] ошибки валидации ложатся под поля по `error.details[].field`
- [ ] тесты: создание, повтор того же UUID, правка, удаление, 422 с деталями
- [ ] Compose-тест формы: ошибка под полем, блокировка кнопки при пустой сумме
- [ ] `make -C android check` — зелёный

### Task 10: Категории

**Files:**
- Create: `android/app/src/main/kotlin/.../ui/categories/CategoriesScreen.kt`,
  `ui/categories/CategoriesViewModel.kt`, `ui/categories/CategoryEditScreen.kt`
- Create: `android/app/src/test/kotlin/.../ui/categories/CategoriesViewModelTest.kt`

- [ ] список с делением на доходные и расходные, вложенность по `parent_id`
- [ ] создание и правка; удаление показывается только роли `admin` из сессии — кнопки нет, а не
      ловим `403`
- [ ] тесты: список, создание, правка, скрытие удаления у роли `member`
- [ ] `make -C android check` — зелёный

### Task 11: Джобы CI

**Files:**
- Modify: `.gitlab-ci.yml`

- [ ] `android:check`: свой `image` и `before_script: []` (иначе наследуется `golang:1.26` и
      Go-кеш), установка `git`, `curl`, `unzip`, SDK и кеши в `/ci-cache/android/*`,
      запуск `make -C android check`
- [ ] `android:api-check`: отдельным шагом, генерация и сверка каталога вывода
- [ ] `android:apk`: `assembleRelease` с keystore из защищённой файловой переменной, артефакт на
      неделю
- [ ] правила: ветка `main` с `changes: [android/**/*, docs/api/openapi.yaml]` и тег `app-v*`
      отдельным `if` — в тег-пайплайнах `changes` всегда истинно, и серверный тег иначе запускал бы
      Android-джобы
- [ ] проверить: правка только под `internal/` Android-джобы не запускает, правка контракта —
      запускает, тег `v0.1.0` — нет

### Task 12: Verify acceptance criteria

- [ ] `make -C android check`, `make -C android api-check`, `make test` (сервер) — зелёные
- [ ] APK ставится на телефон, вход по боевому домену проходит, транзакция создаётся и видна на
      главной; предпосылка выполнена — семья и админ созданы на проде 07.09.2026
- [ ] повторная установка APK поверх предыдущего проходит без удаления приложения (тот же ключ)
- [ ] выключенная сеть даёт понятное сообщение, а не пустой экран
- [ ] удаление сессии через `DELETE /auth/sessions/{id}` уводит клиента на экран входа
- [ ] правка `openapi.yaml` без перегенерации валит `android:api-check`

### Task 13: [Final] Update documentation

- [ ] `android/CLAUDE.md` — карта модулей, правила версий, что генерация вне `check`
- [ ] `README.md`: раздел про приложение и сборку APK
- [ ] `CLAUDE.md`: модули клиента, генерация, правила Android-джоб
- [ ] `docs/tech_stack.md`: раздел про клиента
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

**На стороне владельца:**
- Завести постоянный keystore, положить его на ноутбук и в защищённую файловую переменную CI.
  Потерять его — значит потом сносить приложение с телефонов вместе с настройками.
- Поставить APK на оба телефона через `adb install`.
- Проверить работу снаружи домашней сети.

**Отладочного хоста не будет** (решение владельца): обе сборки ходят в `ffs.shatrov.tech`, и
правки с телефона разработчика видит второй пользователь сразу. Отсюда практическое следствие для
работы над экранами: тестовые данные придётся заводить и убирать руками, а не сбрасывать базу.

**Следующим планом:** бюджеты, отчёты с экспортом, управление пользователями, сессии, бэкапы.
