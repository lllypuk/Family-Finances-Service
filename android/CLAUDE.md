# CLAUDE.md — Android-клиент

Онлайн-клиент к `/api/v1` (`docs/specs/005-api-only-redesign.md`, решение A-13: живёт в этом же
репозитории). Ни Room, ни очередей, ни синхронизации: нет сети — экран говорит об этом и предлагает
повторить.

## Команды

```bash
make -C android check      # = fmt-check test lint; офлайновая, генерации в ней нет
make -C android compile    # assembleDebug
make -C android test       # testDebugUnitTest (Robolectric)
make -C android fmt        # ktlintFormat
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

Навигационной библиотеки нет: sealed-класс `AppScreen` и `rememberSaveable`. DI ручной —
`AppGraph.create(context)`, без Hilt и Koin.

`Session` держит роль и валюту: после логина и при каждом старте с сохранённым токеном клиент
делает `GET /me` и `GET /family`. Без валюты главная не форматирует суммы (в `stats/summary` её
нет), без роли категории не прячут удаление от `member`.

Токен — один непрозрачный с `expires_at`, пары access/refresh нет: у сервера обновления нет, а
`401` означает «сессия кончилась» и ведёт на экран входа. `409 SETUP_REQUIRED` — состояние сервера,
показывается отдельным текстом, а не как ошибка ввода.

Базовый адрес — `https://ffs.shatrov.tech/` со слэшем и без `/api/v1` (пути в сгенерированных
интерфейсах полные). Живёт в `buildConfigField` модуля `:app` и приходит в `ApiGraph` параметром:
`BuildConfig` у каждого модуля свой, из `:core:api` его не прочитать.

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

Compose-тестами покрыты экраны с вводом (вход, список и форма транзакции); главная и категории —
через ViewModel: там нет ввода, а разметка меняется чаще логики.

## Подпись и выкладка

Один постоянный keystore, тот же на ноутбуке (`~/.android/ffs-installer.jks`, путь и пароль —
через `Makefile`) и в защищённой файловой переменной CI. Смена сертификата даёт
`INSTALL_FAILED_UPDATE_INCOMPATIBLE` и требует сносить приложение вместе с данными; нет keystore —
`assembleRelease` падает, а не подписывается чем попало. В дерево репозитория файл не кладётся.

Стора не будет: APK ставится с ноутбука. Тег `app-v*` собирает APK в CI; серверные теги `vX.Y.Z`
Android-джоб не запускают.

**Окно несовместимости.** Сервер выкатывается сам с каждого мержа в `main`, телефоны обновляются
руками: ломающая правка контракта роняет установленные APK. Порядок — правка аддитивна, а если
ломает, то сначала APK на оба телефона, потом мерж серверной части.
