# План 08 — Профиль и настройки в Android-клиенте

Продолжение [плана 07](completed/20260911-07-android-budgets.md). Закрывает остаток контракта:
`PUT /me`, `PUT /me/password`, `GET/DELETE /auth/sessions`, а для admin — `/users`, `PUT /family`,
`/backups`. Отчёты и `downloadBackup` остаются в [бэклоге](../backlog.md).

## Overview

- Экран «Настройки» с подразделами: профиль (имя, почта), смена пароля, сессии; для admin —
  пользователи, семья, бэкапы. «Выйти» переезжает из шапки главной вниз этого экрана, а на её
  месте — иконка профиля (решение владельца 11.09.2026).
- Второй пользователь семьи получает свой пароль и свои сессии без ssh на сервер; admin заводит и
  чинит пользователей с телефона.
- Контракт и сервер не меняются. Меняется транспорт: `TokenInterceptor` перестаёт считать
  `401 INVALID_CREDENTIALS` концом сессии, логин шлёт `device_name`.

## Context (from discovery)

- Корень: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt` (`AppRoot`,
  445 строк; флаги `listStale/homeStale/budgetsStale`, `epoch`, `forms: FormModels` создаётся в
  `AppRoot` через `viewModel { }`, `onForm`; `LaunchedEffect(screen, session)` на `:103` уводит на
  `Loading`, как только `session == null`), `AppScreen.kt` (`AppScreenSaver`, `restore` разбирает
  ключ деструктуризацией фиксированной арности), `AppGraph.kt` (`bootstrap()` обнуляет `Session`
  при любом отказе — `:37`; `signOut()`), `Session.kt` (`zone` с подменой неизвестной зоны на зону
  телефона, `isAdmin`, `currency`).
- Шапка главной с «Выйти»: `ui/home/HomeScreen.kt:66`. Образцы форм: `ui/budgets/BudgetEditViewModel.kt`
  (`PUT` из diff, `fieldErrors`, `done`), уход с формы во время `submitting` запрещён
  (`MainActivity.kt:361`); список одной страницей: `ui/categories/CategoriesViewModel.kt`.
- Транспорт: `core/api/.../auth/TokenInterceptor.kt:31` — любой `401` не на пути логина →
  `tokens.clearIf(token)` + `onSessionExpired()`. Тесты — `core/api/src/test/.../auth/TokenInterceptorTest.kt`.
  `ApiClient.kt`: `unwrap` для ответов с телом, `send` только для `Response<Unit>`;
  `READ_TIMEOUT_SECONDS = 30`.
- Сгенерировано и не подключено: `BackupsApi` (нет в `ApiGraph`; `createBackup` возвращает
  `Response<CreateBackup201Response>` — то есть `unwrap`); `MeApi.changePassword/updateCurrentUser`,
  `AuthApi.listSessions/revokeSession`, `UsersApi.*`, `FamilyApi.updateFamily` есть в графе, но не
  вызываются. Сгенерированная модель `core.api.Session` конфликтует по имени с `Session` приложения —
  импортировать с алиасом `ApiSession`.
- `LoginViewModel.kt:98` не шлёт `device_name`. `MainActivity.kt` создаёт `LoginViewModel(graph.api)`.
- `ui/format/Dates.kt` — только `LocalDate`; форматтера для `OffsetDateTime` нет, `DatesTest` тоже.
  `TestFixtures.kt` уже содержит `USERS_OK` с двумя пользователями.
- Коды `409`, различимые клиентом: `EMAIL_TAKEN`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`,
  `CURRENCY_LOCKED` (`internal/application/handlers/errors.go`). `PUT /me/password` с неверным
  текущим паролем — `401 INVALID_CREDENTIALS` (`handlers/me.go:97`), сессия жива.

Сервер (проверено 11.09.2026 вместе с codex):

- Смена своего пароля сохраняет текущую сессию (`internal/auth/service.go:182`); админская
  установка пароля отзывает все, включая свою, если цель — сам админ (`:199`, `uuid.Nil`).
  Хеш пишется до отзыва сессий (`:211`, `:214`), сбой отзыва — `500` (`handlers/me.go:100`,
  `users.go:244`): ошибка после записи оставляет новый пароль. Деактивация так же пишется до
  отзыва сессий (`user_service.go:222`).
- Политика пароля — 10…72 **байта** (`internal/auth/password.go:25`), не символов.
- `PATCH /users/:id` с двумя полями применяет роль, затем активность; `409` второго шага не
  откатывает первый (`handlers/users.go:203`). Себя деактивировать нельзя (`user_service.go:209`),
  свою роль понизить — можно (`:236`), после чего сервер отвечает `403` на admin-роуты со
  следующего запроса (`auth/service.go:138`). `PUT /users/:id` отдаёт `409 EMAIL_TAKEN`
  (`user_service.go:181`, `handlers/users.go:39`), которого нет у этой операции в `openapi.yaml`.
- `PUT /family`: валюта меняется, пока в семье нет транзакций (`family_service.go:103`), иначе
  `409 CURRENCY_LOCKED`; таймзона проверяется `time.LoadLocation`.
- `POST /backups` синхронный: `VACUUM INTO`, потом retention (`backup_service.go:155`, `:244`;
  ошибки удаления старых пропускаются — `:390`); серверный `WriteTimeout` 15 с
  (`internal/config.go:20`), клиентский read 30 с. Длительность `VACUUM` на проде не измерена;
  если она превысит таймаут, файл появится, а ответ не дойдёт, и повтор создаст второй файл.
- Списки пагинированы (`limit` ≤ 200, `meta.pagination.total` есть — `helpers.go:208`); сессии
  сортируются новыми вперёд (`session_repository_sqlite.go:174`), число сессий не ограничено
  (`auth/service.go:94`). `RevokeSession` чужой или исчезнувшей — `404` (`handlers/auth.go:155`).

## Решения

Раздел заменяет Solution Overview / Technical Details шаблона.

- **Вход через иконку в шапке главной**, не пятая вкладка: настройки нужны реже вкладок, у member
  раздел почти пуст. Цена — с других вкладок в настройки через главную.
- **Вложенный хост.** В `AppRoot` одна ветка `AppScreen.Settings(page: SettingsPage)`; страницы,
  их `when` и `BackHandler` — в `ui/settings/SettingsHost.kt`. `SettingsPage`: `Root`, `Profile`,
  `Password`, `Sessions`, `Users`, `UserEdit(id: UUID?)`, `UserPassword(id)`, `Family`, `Backups`;
  у каждой страницы `visit: UUID` — идентичность захода, как `draft` у форм. «Назад»: из
  `UserEdit`/`UserPassword` → `Users`, из подраздела → `Root`, из `Root` → главная. Ключ в
  `AppScreenSaver` фиксированной арности: `settings:<page>:<id или пусто>:<visit>`. После смерти
  процесса бутстрап ведёт на главную, как и у форм сейчас — принято.
- **Модели подразделов** живут в `SettingsModels` (`ViewModelStoreOwner` по образцу `FormModels`,
  создаётся в `AppRoot` через `viewModel { }` и передаётся в хост параметром, как `forms`).
  Модель ключуется `"<page>-<visit>"`; store чистится при **каждой** смене страницы и при уходе с
  `AppScreen.Settings`, так что повторный заход на `Profile` всегда получает новую модель, а
  списки перечитываются при входе — флагов устаревания внутри настроек нет. Уход со страницы
  (назад, пункт меню) **заблокирован, пока `submitting`**: очистка store отменила бы мутацию,
  которую сервер уже мог применить.
- **`Root` перечитывает сессию** новым `AppGraph.refreshSession(): ApiFailure?`: оба `GET`
  выполняются, прежняя `Session` остаётся при любом отказе, новая публикуется только если за это
  время не было другой публикации (счётчик поколений в `AppGraph`; `bootstrap`, `signOut`,
  `update*` его двигают). Настоящий `401` по-прежнему заканчивает сессию через интерцептор;
  защита `session == null → Loading` в `AppRoot` не трогается. Результат перечитки идёт через тот
  же `onSessionChanged(before, after)`, что и ответы `PUT`. `403` в любом подразделе —
  `UiError.Resource(settings_forbidden)` и возврат в `Root`.
- **Обновление `Session` без бутстрапа**: `AppGraph.update(user)` и `update(family)` из ответа
  `PUT`. `AppRoot` в `onSessionChanged` при смене `zone`, `currency` или `role` делает `epoch++`
  (пересоздание моделей вкладок при следующем заходе; старые остаются в store до конца активити —
  так уже при повторном входе) и взводит все три флага; смена имени или почты — `listStale` (кеш
  имён авторов в операциях). Правка **другого** пользователя `Session` не меняет — хост отдаёт
  отдельный `onUsersChanged` → `listStale`. Правка своей записи через `Users` тоже обновляет
  `Session`, не только при смене роли.
- **`401` в интерцепторе**: для `PUT /api/v1/me/password` ответ `401` с разобранным
  `error.code == INVALID_CREDENTIALS` — не конец сессии; читается копия тела через
  `response.peekBody(limit)`, оригинал остаётся `ApiClient`. `UNAUTHORIZED` и нечитаемое тело —
  по-прежнему `clearIf` + событие. Путь целиком не исключается: запрос должен уходить с токеном,
  а просроченный токен — чиститься.
- **`device_name = Build.MODEL`**, обрезанный до 64 символов, пустой не шлётся; параметр
  `LoginViewModel` со значением по умолчанию, чтобы `MainActivity` не менялся. Старые сессии без
  имени показываются как «Без имени»; в строке — даты создания и последнего использования в зоне
  семьи (с пометкой «обновляется не чаще раза в час»); текущая помечена и не отзывается — для
  неё «Выйти».
- **Списки одной страницей** (`limit = 200`): пользователей два, сессий единицы, бэкапы режет
  retention. Цена видима: если `meta.pagination.total` больше полученного, внизу «Показаны первые
  200 из N»; полнота списка и попадание текущей сессии на страницу не утверждаются.
- **Пароли**: длина считается в байтах UTF-8, как на сервере. После `204` своей смены — поля
  чистятся, текст «Пароль изменён, другие устройства разлогинены». `Network`, `Malformed` и `5xx`
  после отправки — «Результат неизвестен: если ошибка повторится, войдите с новым паролем», без
  автоповтора. У админской установки тот же класс отказов, текст про целевого пользователя:
  «Результат неизвестен: попросите пользователя войти с новым паролем».
- **Пользователи**: список, форма создания (почта, пароль, имя, фамилия, роль), правка имени и
  почты (`PUT` из diff), роль и активность — два отдельных `PATCH` с одним полем, установка
  пароля — отдельная страница. Для себя скрыты деактивация и установка пароля (своя смена — в
  «Пароль»); понижение своей роли разрешено: после `200` — `update(user)`, выход в `Root`.
  `409` переводятся по коду: `EMAIL_TAKEN`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`; последний админ
  по загруженному списку не вычисляется — решает сервер. После любой правки и после **ошибки**
  деактивации пользователь перечитывается `GET /users/:id`: `is_active` мог уже измениться.
  `409` у `PUT /users/:id` отсутствует в спеке — клиент обрабатывает код, спека дописывается
  отдельно (бэклог), чтобы не тянуть перегенерацию в этот план.
- **Семья**: название, валюта и таймзона — текстовые поля, `PUT` из diff. Валюта не прячется:
  сервер ответит `409 CURRENCY_LOCKED`, текст переводится по коду. Таймзона без клиентской
  проверки и без пикера: меняется примерно никогда, `422` от `time.LoadLocation` ложится под поле.
- **Бэкапы**: список (`limit = 200`), «Создать» (`unwrap`, после `201` список перечитывается —
  retention мог удалить старые), удаление с подтверждением. После `Network`/`Malformed`/`5xx` на
  создании — «Результат неизвестен», список перечитывается, повтор только руками. Скачивания нет:
  на телефоне файлу базы делать нечего, восстановление — по ssh (A-11).
- **Перевод кодов**: перегрузка `ApiFailure.toUiError(known: Map<String, Int> = emptyMap())` —
  известный `code` → `UiError.Resource`, остальное как раньше; существующие вызовы без аргументов
  не меняются. Общая карта для `409` семьи и пользователей.

## Development Approach

- **testing approach**: Regular — код, затем тесты в той же задаче; конвенция — Compose-тесты для
  экранов с вводом, ViewModel-тесты через `MockWebServer`, форматтеры с JVM-тестами
  (`android/CLAUDE.md`, `ui/format/MoneyTest.kt`).
- Каждая задача закрывается зелёным `make -C android check`; заглушки из Task 2 заменяются в
  задачах 3–8, и каждая из них компилируется сама.
- Правки в план — сразу при отклонении от объёма.

## Testing Strategy

- Транспорт: `TokenInterceptorTest` — `401 INVALID_CREDENTIALS` на `/me/password` не чистит
  хранилище и не поднимает событие; `401 UNAUTHORIZED` там же — чистит; `401` без тела — чистит.
- `AppGraphTest`: `refreshSession` при отказе сохраняет `Session`; поздний результат не
  перезаписывает более новую публикацию; `update(user)` меняет только пользователя.
- ViewModel: по тесту на модель подраздела через `MockWebServer` с проверкой тела/query.
- Compose (экраны с вводом): `SettingsRootScreenTest`, `ProfileScreenTest`, `PasswordScreenTest`,
  `UserEditScreenTest`, `FamilyScreenTest`. Сессии и бэкапы — без ввода, только ViewModel.
- Форматтеры: `DatesTest` (`formatDateTime` в зоне семьи), `SizesTest`.
- `AppScreenSaverTest` — все `SettingsPage`.
- e2e нет.

## Progress Tracking

- `[x]` — сделано; `➕` — найдено по ходу; `⚠️` — блокер.

## Implementation Steps

### Task 1: Транспорт — `401` при смене пароля, `device_name`, `BackupsApi`, перевод кодов

**Files:**
- Modify: `android/core/api/src/main/kotlin/tech/shatrov/familyfinances/core/api/auth/TokenInterceptor.kt`
- Modify: `android/core/api/src/test/kotlin/tech/shatrov/familyfinances/core/api/auth/TokenInterceptorTest.kt`
- Modify: `android/core/api/src/main/kotlin/tech/shatrov/familyfinances/core/api/ApiGraph.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/login/LoginViewModel.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/login/LoginViewModelTest.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/UiError.kt`

- [x] `TokenInterceptor`: `401` на `PUT /api/v1/me/password` с `error.code == INVALID_CREDENTIALS`
      (копия тела `peekBody(PEEK_LIMIT)`, разбор `ErrorEnvelope`) — пропускается без `clearIf`;
      любой другой `401` — как сейчас
- [x] тесты интерцептора: три случая из Testing Strategy; `unauthorizedClearsVaultAndRaisesEvent`
      не сломан
- [x] `ApiGraph.backups: BackupsApi`
- [x] `LoginViewModel(api, deviceName: String? = Build.MODEL.take(64).ifBlank { null })`; тест на
      тело логина с `device_name` и без него при `null`
- [x] `UiError.kt`: перегрузка `toUiError(known)` со значением по умолчанию
- [x] `make -C android check` — зелёный

### Task 2: Каркас настроек — `AppScreen.Settings`, хост, корень, вход с главной

**Files:**
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/AppScreen.kt`
- Modify: `android/app/src/test/kotlin/tech/shatrov/familyfinances/AppScreenSaverTest.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/AppGraph.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/AppGraphTest.kt`
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/SettingsPage.kt`
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/SettingsHost.kt`
- Create: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/settings/SettingsRootScreen.kt`
- Create: `android/app/src/test/kotlin/tech/shatrov/familyfinances/ui/settings/SettingsRootScreenTest.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/home/HomeScreen.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/MainActivity.kt`
- Modify: `android/app/src/main/kotlin/tech/shatrov/familyfinances/ui/AppIcons.kt`
- Modify: `android/app/src/main/res/values/strings.xml`

- [x] `SettingsPage` (sealed, девять страниц, у каждой `visit`) и `AppScreen.Settings(page)`;
      `AppScreenSaver` с ключом фиксированной арности; `AppScreenSaverTest` — все страницы
- [x] `AppGraph`: счётчик поколений, `refreshSession()`, `update(user)`, `update(family)`;
      `AppGraphTest` по Testing Strategy
- [x] `SettingsModels` (копия `FormModels`) создаётся в `AppRoot`, чистится при смене `page` и при
      `screen !is Settings`
- [x] `SettingsHost(graph, session, models, page, onPageChange, onLeave, onSessionChanged,
      onUsersChanged, onSignedOut)`: `when` по странице, `BackHandler` по правилам из «Решений»,
      уход заблокирован при `submitting` текущей страницы; на этой задаче реальная только `Root`,
      остальные восемь — заглушка `Centered { Text(page.name) }` без ссылок на будущие классы
- [x] `SettingsRootScreen`: карточка пользователя (имя, почта, роль), карточка семьи (название,
      валюта, таймзона), пункты «Профиль», «Пароль», «Сессии», для admin — «Пользователи», «Семья»,
      «Бэкапы»; внизу «Выйти» с подтверждением. При входе (`LaunchedEffect(visit)`) —
      `refreshSession()` с индикатором поверх карточек; отказ — текст и «Повторить», карточки с
      прежней `Session` остаются; успех — `onSessionChanged(before, after)`
- [x] главная: иконка профиля (`AppIcons.User`, контур Lucide) вместо `LogOut` →
      `AppScreen.Settings(Root)`
- [x] `AppRoot`: ветка `Settings`; `onSessionChanged`: `zone`/`currency`/`role` → `epoch++` и все
      три флага, имя/почта → `listStale`; `onUsersChanged` → `listStale`; `onSignedOut` → `Login`
- [x] `SettingsRootScreenTest`: у member нет admin-пунктов, у admin есть; «Выйти» → подтверждение →
      колбэк; отказ перечитки оставляет карточки
- [x] `make -C android check` — зелёный

### Task 3: Профиль и смена пароля

**Files:**
- Create: `ui/settings/ProfileViewModel.kt`, `ui/settings/ProfileScreen.kt`
- Create: `ui/settings/PasswordViewModel.kt`, `ui/settings/PasswordScreen.kt`
- Create: тесты `ProfileViewModelTest`, `ProfileScreenTest`, `PasswordViewModelTest`, `PasswordScreenTest`
- Modify: `ui/settings/SettingsHost.kt` (заглушки `Profile`, `Password` → экраны), `strings.xml`

- [x] `ProfileViewModel`: поля из `Session.user`, `PUT /me` из diff (`UpdateUserRequest`), `409
      EMAIL_TAKEN` → перевод, `422` → под поля; после `200` — `update(user)`, `onSessionChanged`, `done`
- [x] `PasswordViewModel`: текущий, новый, повтор; `canSubmit` — байтовая длина 10…72 и совпадение;
      `send { changePassword }`; `401 INVALID_CREDENTIALS` → ошибка под «Текущий пароль»;
      `Network`/`Malformed`/`5xx` → текст о неизвестном результате; успех — поля чистятся, текст
- [x] экраны по образцу `BudgetEditScreen`; подключение в хост
- [x] тесты: тело `PUT /me` из одного поля; `401` не поднимает `sessionExpired` (реальный
      `ApiGraph` над `MockWebServer`); `500` → неизвестный результат; «ёёёёё» (10 байт) проходит,
      9 байт — нет
- [x] `make -C android check` — зелёный

### Task 4: Сессии и формат времени

**Files:**
- Modify: `ui/format/Dates.kt` (`formatDateTime(OffsetDateTime, zone)`), Create: `ui/format/DatesTest.kt`
- Create: `ui/settings/SessionsViewModel.kt`, `ui/settings/SessionsScreen.kt`, `SessionsViewModelTest`
- Modify: `ui/settings/SettingsHost.kt`, `strings.xml`, `TestFixtures.kt` (`SESSIONS_OK`)

- [x] `formatDateTime` в зоне семьи; `DatesTest` — одна и та же метка в двух зонах даёт разные строки
- [x] `SessionsViewModel(api, zone)`: `listSessions(limit = 200)`, текущая с пометкой; строка: имя
      устройства или «Без имени», создана, последняя активность; «Показаны первые 200 из N» при
      усечении
- [x] «Отозвать» с подтверждением у всех, кроме текущей; `404` → перечитать без ошибки
- [x] тесты: query, `404` → refresh, у текущей нет действия, усечение по `total`
- [x] `make -C android check` — зелёный

### Task 5: Пользователи — список

**Files:**
- Create: `ui/settings/UsersViewModel.kt`, `ui/settings/UsersScreen.kt`, `UsersViewModelTest`
- Modify: `ui/settings/SettingsHost.kt`, `strings.xml`, `TestFixtures.kt` (`USERS_TRUNCATED`)
- ➕ Modify: `ui/settings/SettingsHeader.kt` — слот под кнопку справа от заголовка («плюс» списка)

- [x] `listUsers(limit = 200)`, строка: имя, почта, роль, «неактивен», «это вы»; усечение по `total`
- [x] переходы в `UserEdit(null)` и `UserEdit(id)` — пока на заглушку из Task 2
- [x] тесты: query, отметка себя, усечение
- [x] `make -C android check` — зелёный

### Task 6: Пользователи — форма, роль/активность, пароль

**Files:**
- Create: `ui/settings/UserEditViewModel.kt`, `ui/settings/UserEditScreen.kt`
- Create: `ui/settings/UserPasswordViewModel.kt`, `ui/settings/UserPasswordScreen.kt`
- Create: `UserEditViewModelTest`, `UserEditScreenTest`, `UserPasswordViewModelTest`
- Modify: `ui/settings/SettingsHost.kt`, `strings.xml`, `PasswordScreen.kt` (`SecretField` стал
  `internal` — то же поле у создания и у админской установки), `SettingsConflicts.kt`,
  `TestFixtures.kt` (`USER_ADMIN_OK`, `USER_INACTIVE_OK` вместо `USER_OK`: `MEMBER_OK` уже
  подходит под `GET /users/:id`)

- [x] `UserEditViewModel`: создание — `CreateUserRequest` (пароль в байтах 10…72); правка —
      `getUser(id)` при входе, `PUT` из diff; отдельные действия «Сделать админом/участником»
      (`PATCH {role}`), «Деактивировать/Активировать» (`PATCH {is_active}`) с подтверждением; после
      каждого `2xx` и после ошибки деактивации — перечитать `getUser(id)`; для себя скрыты
      деактивация и «Задать пароль»; своя запись после `200` — `update(user)`, `onSessionChanged`;
      понижение своей роли — `done` в `Root`; чужая запись — `onUsersChanged`
- [x] `409` → `EMAIL_TAKEN`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF` по-русски; `422` — под поля
- [x] `UserPasswordViewModel`: `send { setUserPassword }`, успех — назад в `UserEdit` с текстом
      «Пароль задан, пользователь разлогинен везде» (➕ текст держит хост: store страницы к моменту
      перехода уже очищен); `Network`/`5xx` — неизвестный результат про целевого пользователя
- [x] `UserEditScreenTest`: у своей записи нет деактивации; ошибка под полем; подтверждение перед
      `PATCH`
- [x] тесты моделей: `PATCH` с одним полем; `409 LAST_ADMIN` → `Resource`; понижение своей роли →
      `onSessionChanged`; ошибка деактивации → повторный `GET`
- [x] `make -C android check` — зелёный

### Task 7: Семья

**Files:**
- Create: `ui/settings/FamilyViewModel.kt`, `ui/settings/FamilyScreen.kt`, `FamilyViewModelTest`, `FamilyScreenTest`
- Modify: `ui/settings/SettingsHost.kt`, `strings.xml`

- [x] поля из `Session.family` (название, валюта, таймзона — текстовые); `PUT` из diff
      (`UpdateFamilyRequest`); `409 CURRENCY_LOCKED` → перевод; `422` под поля (в том числе
      неизвестная зона); после `200` — `update(family)`, `onSessionChanged`, `done`
- [x] тесты: тело из одного поля; `409` → `Resource`; `422 timezone` под полем; кнопка выключена без diff
- [x] `make -C android check` — зелёный

### Task 8: Бэкапы и формат размера

**Files:**
- Create: `ui/format/Sizes.kt` (`formatBytes`), `ui/format/SizesTest.kt`
- Create: `ui/settings/BackupsViewModel.kt`, `ui/settings/BackupsScreen.kt`, `BackupsViewModelTest`
- Modify: `ui/settings/SettingsHost.kt`, `strings.xml`, `TestFixtures.kt` (`BACKUPS_OK`)

- [x] `formatBytes` («12,3 МБ»), тест на границы КБ/МБ
- [x] `BackupsViewModel(api, zone)`: `listBackups(limit = 200)` — имя, размер, дата в зоне семьи;
      «Создать» — `unwrap { createBackup() }`, индикатор, после `201` перечитать;
      `Network`/`Malformed`/`5xx` → «Результат неизвестен» и перечитать; усечение по `total`
- [x] «Удалить» с подтверждением (`send`); `404` → перечитать
- [x] тесты: успех → POST и GET; обрыв → текст и GET; удаление; усечение
- [x] `make -C android check` — зелёный

### Task 9: Verify acceptance criteria

- [x] руками на debug-сборке против `ffs.shatrov.tech` под admin (skipped — ручная проверка на устройстве): переименовать **другого**
      пользователя — в операциях его подпись обновилась; сменить свой пароль с неверным текущим —
      остались в приложении; с верным — второй телефон разлогинен; при выключенной сети открыть
      настройки — карточки на месте, приложение не ушло на экран загрузки; создать бэкап, увидеть
      в списке, удалить
- [x] под member (второй телефон) (skipped — ручная проверка на устройстве): нет admin-пунктов; своя новая сессия с именем устройства
- [x] `make -C android check`, `make -C android compile` — зелёные; `api-check` не нужен

### Task 10: [Final] Update documentation

- [ ] `android/CLAUDE.md`: вход в настройки, вложенный хост и `SettingsModels` (очистка на смене
      страницы, блокировка ухода при `submitting`), `refreshSession` и поколения, исключение для
      `401 INVALID_CREDENTIALS`, `device_name`, алиас `ApiSession`
- [ ] `android/gradle/libs.versions.toml`: `appVersionCode` 2 → 3, `appVersionName` 0.2.0 → 0.3.0
- [ ] `docs/backlog.md`: снять «Экран Профиль/Настройки», оставить отчёты и `bulkDelete`; добавить
      серверные находки из «Context»: `409` у `PUT /users/:id` не в спеке; пароль и деактивация
      записываются до отзыва сессий (нужна одна транзакция); `POST /backups` против
      `SERVER_WRITE_TIMEOUT` 15 с — измерить `VACUUM` на проде, при необходимости поднять таймаут
      роута или сделать операцию асинхронной; `PATCH` двух полей без отката
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

**Ручная проверка:** оба телефона после `app-v0.3.0`; у второго пользователя первая сессия
создана до `device_name` и останется «Без имени» до перелогина.

**Сервер (в бэклог):** см. Task 10 — четыре находки, ни одна не блокирует клиент.
