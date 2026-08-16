# Устранение блокеров развёртывания на self-hosted мини-сервере

> Ревизия 2 (после автоматического ревью). Изменения против ревизии 1: добавлена
> задача 1 — починка тестового хелпера, без неё задачи 2, 4 и 12 нереализуемы;
> задача 5 перенацелена на настоящие request-структуры; задача 11 больше не ломает
> `install.sh`; уточнены ожидаемые коды 401 против 403; добавлены ролевые проверки
> API и D-03.

## Overview

Цель — безопасно поднять Family Finances Service на домашнем мини-сервере и начать
им пользоваться. Сейчас это невозможно: REST API отдаёт и принимает данные без
аутентификации, `docker compose` не стартует, а образ, на который ссылаются все
production-конфиги, никогда не публиковался.

План закрывает находки аудита ([docs/specs/](../specs/README.md)):

| ID | Находка | Приоритет | Задачи |
|---|---|---|---|
| [S-01](../specs/002-security-audit.md#s-01) | `/api/v1/*` — полный анонимный CRUD (подтверждено live: 200/201/204) | P0 | 2-6 |
| [D-01](../specs/004-deployment-readiness.md#d-01) | `docker-compose.yml` не стартует — нет `CSRF_SECRET` | P0 | 10 |
| [D-02](../specs/004-deployment-readiness.md#d-02) | Образ в GHCR не существует, тегов ноль | P0 | 11 |
| [U-01](../specs/003-ui-ux-audit.md#u-01) | `RequireSetup` редиректит `/static` и `/health` → setup без CSS | P1 | 8 |
| [S-02](../specs/002-security-audit.md#s-02) | CSRF-токен не перевыпускается при логине | P1 | 7 |
| [U-02](../specs/003-ui-ux-audit.md#u-02) | Навигация пропадает на страницах-списках | P1 | 12-15 |
| [S-04](../specs/002-security-audit.md#s-04) | Небезопасные значения по умолчанию в `minimal.yml` | P2 | 10 |
| [S-05](../specs/002-security-audit.md#s-05) | `IsSetupComplete` — запрос в БД на каждый HTTP-запрос | P2 | 9 |
| [U-05](../specs/003-ui-ux-audit.md#u-05) | Английские заголовки, мёртвая ссылка `/register` | P2 | 15, 16 |
| [D-03](../specs/004-deployment-readiness.md#d-03) | Несоответствие UID: `nobody` в образе против `1000:1000` в compose | P2 | 11 |
| [D-04](../specs/004-deployment-readiness.md#d-04) | Ссылка на несуществующий `docs/tasks/...` в `install.sh` | P3 | 11 |

**Ключевые решения** (приняты при планировании):

- **Аутентификация API — через сессию.** `RequireAPIAuth` на группу `/api/v1`.
  API-токены для внешних интеграций — отдельный план, когда появится потребитель.
- **Docker — сборка на месте**, но только там, где это не ломает `install.sh`.
  Тег `v0.1.0` ставим последним шагом, когда блокеры закрыты.
- **Тесты — TDD для задач безопасности (2-7)**, обычный порядок для остальных.

**Вне объёма** (сознательно): rate limiting на логин ([S-03](../specs/002-security-audit.md#s-03)),
слой макета и мобильное меню ([U-03](../specs/003-ui-ux-audit.md#u-03),
[U-04](../specs/003-ui-ux-audit.md#u-04)), редизайн ([003](../specs/003-ui-ux-audit.md)),
генерация отчётов в API. Это следующие планы — они не мешают безопасно поднять сервис.

## Context (from discovery)

**Проверено на живом коде, а не предположено:**

- **Тестовый сервер поднимается без веб-слоя.** `testhelpers.SetupHTTPServer` вызывает
  `application.NewHTTPServer`, который создаёт веб-сервер с путём `"internal/web/templates"`
  относительно cwd (`internal/application/http_server.go:114`). При `go test ./tests/integration/`
  cwd = `tests/integration`, `ParseGlob` не находит шаблоны, `NewWebServer` возвращает
  ошибку, а `http_server.go:118-126` её **молча глотает** (`obsService` в тестах nil).
  Экспериментально подтверждено:

  ```
  GET  /login                        → 404   ← веб-маршрутов нет
  POST /api/v1/transactions без CSRF → 400   ← CSRF middleware не зарегистрирован
  ```

  Следствие: интеграционные тесты проверяют приложение **без** `SessionStore`,
  `CSRFProtection` и веб-маршрутов. Это первопричина того, что S-01 и U-02 прошли
  мимо тестов, и это надо чинить до всего остального.

- **Реальные request-структуры API — не в `dto`.** Хендлеры биндят
  `CreateTransactionRequest` (`internal/application/handlers/types.go:105`) и
  `CreateReportRequest` (`types.go:183`), оба с `UserID … validate:"required"`.
  А `dto.CreateTransactionAPIRequest` (`api_mappers.go:128`) — **мёртвый код**:
  ссылки только из самого файла и его теста.

- **`req.UserID` в `transactions.go` встречается дважды** — строки 83
  (`createTransactionViaService`) и 130 (`buildTransaction`).

- **Порядок middleware.** `CSRFProtection` регистрируется глобально через `e.Use`
  (`internal/web/web.go:53`) и отрабатывает **раньше** group-middleware группы
  `/api/v1`. Значит в проде анонимный POST без токена вернёт 403, а не 401 — это
  надо учесть в ожиданиях тестов.

- **`install.sh` не имеет исходников на сервере.** Строка 156 копирует
  `docker-compose.prod.yml` в `/opt/family-budget/docker-compose.yml`, строка 256
  делает `docker compose pull`; `upgrade.sh:239` — `pull app`. Перевод `prod.yml`
  на `build:` без правки скриптов сломает единственный рабочий путь установки.

- **Цикла импортов нет:** `internal/web/middleware` тянет только `echo` и
  `domain/user`, поэтому `internal/application` может его импортировать.

- **`RequireAuth()` нельзя переиспользовать как есть** — при отсутствии сессии он
  делает 302 на `/login`; API нужен 401 JSON.

- **`RequireSetup` — функция-фабрика** (`internal/web/middleware/setup.go:17`),
  структуры нет. И она читает `c.Path()` — это **шаблон маршрута**, а не URL
  (для статики Echo регистрирует `/static*`, а для незарегистрированного
  `/favicon.ico` вернёт не то, что ожидается).

## Development Approach

- **testing approach**: TDD для задач 2-7 (безопасность), regular для остальных
- завершать задачу полностью перед переходом к следующей
- **каждая задача обязана содержать новые/обновлённые тесты**
- **все тесты должны проходить перед началом следующей задачи**
- после каждой задачи: `make fmt`, `make test`, `make lint` — линтер обязан выдать 0 issues
- обновлять этот файл при изменении объёма

**Замечания по линтеру** (`.golangci.yml`, строгий):
- `gochecknoglobals` включён — кэш в задаче 9 делать `atomic.Bool` **внутри замыкания**,
  а не пакетной переменной; отдельный тип и `sync.RWMutex` избыточны.
- `testpackage` отключён только для `internal/web/handlers/`, `internal/services/dto/`,
  `tests/`. Новые тесты в `internal/web/middleware` и `internal/web` обязаны быть
  `middleware_test` / `web_test`.
- `embeddedstructfieldcheck`: встроенные поля идут первыми и отделяются пустой
  строкой (образец — `dashboard.go:110`).
- `dupl` может зацепить пять почти одинаковых анонимных структур в задачах 13-15
  (в тестах `dupl` отключён, в production-коде — нет).

## Testing Strategy

- **unit-тесты**: обязательны в каждой задаче
- **интеграционные**: `tests/integration/` через `testhelpers.SetupHTTPServer(t)` —
  **после задачи 1**, которая делает этот хелпер пригодным
- **шаблоны**: задача 12 вводит тест, проверяющий данные **хендлера**, а не
  собранные вручную в тесте — иначе регрессия U-02 не ловится
- **e2e**: в проекте нет Playwright/Cypress; ручная проверка вынесена в Post-Completion

## Progress Tracking

- отмечать выполненное `[x]` сразу
- новые задачи — с префиксом ➕
- блокеры — с префиксом ⚠️

## Solution Overview

**Задача 1 — предусловие для всего остального.** Пока тестовый сервер поднимается
без веб-слоя, ни дыру воспроизвести, ни существующие тесты аутентифицировать нечем.

Дальше три направления:

1. **Безопасность API** (2-7). `RequireAPIAuth` с ответом 401 JSON на группе
   `/api/v1`; `user_id` из сессии; ролевые проверки на разрушающих маршрутах;
   перевыпуск CSRF-токена при логине.
2. **Первый запуск** (8-9). `RequireSetup` перестаёт трогать статику и `/health`,
   результат проверки кэшируется.
3. **Развёртывание** (10-11). Секреты в compose, сборка из исходников, UID.

Задачи 12-16 — контракт шаблонов и чистка ссылок.

## Technical Details

**Политика CSRF для API.** `CSRFProtection` остаётся глобальным и продолжает
действовать на `/api/v1`. Это осознанно: при сессионной аутентификации CSRF
обязателен, иначе любой сторонний сайт сможет действовать от имени залогиненного
пользователя. Практическое следствие, которое надо записать в `deploy/README.md`:
программный клиент API обязан сначала сделать GET за токеном, затем слать его в
`X-Csrf-Token`. Ожидаемые коды для анонимного запроса:

| Запрос | Код | Кто вернул |
|---|---|---|
| GET без сессии | 401 | `RequireAPIAuth` |
| POST/PUT/DELETE без CSRF-токена | 403 | глобальный `CSRFProtection` |
| POST/PUT/DELETE с валидным токеном, без сессии | 401 | `RequireAPIAuth` |

**Новый middleware.** В `internal/web/middleware/auth.go` рядом с `RequireAuth`:
`RequireAPIAuth()` отвечает 401 и JSON в формате, который уже используют API-хендлеры
(`handlers/errors.go`, `types.go`). При валидной сессии кладёт `SessionData` в
контекст под тем же ключом `"user"`.

**Источник user_id.** В `handlers/transactions.go` (строки 83 и 130) и
`handlers/reports.go` — вместо `req.UserID` брать из сессии. Обратите внимание:
`GetUserFromContext` возвращает `(*SessionData, error)`, а не значение.
Поле `UserID` из request-структур **удаляется** (а не просто теряет
`validate:"required"`) — иначе клиент продолжит его слать, а сервер молча
игнорировать.

**Кэш настройки.** `atomic.Bool` внутри замыкания `RequireSetup`: после первого
`true` в БД не ходим. Обратный переход невозможен — семью нельзя удалить через UI.

**Контракт шаблонов.** 19 мест с `"PageData": pageData` (budgets — 6, categories — 5,
reports — 4, transactions — 3, transactions_helpers — 1). При встраивании `*PageData`
имя поля остаётся `PageData`, поэтому существующие `{{.PageData.X}}` в шаблонах
**продолжат работать** — массово править шаблоны не нужно.

## What Goes Where

- **Implementation Steps** — всё, что делается в этом репозитории
- **Post-Completion** — выкат на mini, ручная проверка в браузере, тег релиза

## Implementation Steps

### Task 1: Починить тестовый хелпер — сервер с полным веб-слоем

Предусловие для задач 2, 4 и 12. Сейчас тесты проверяют приложение без сессий,
CSRF и веб-маршрутов.

**Files:**
- Modify: `internal/testhelpers/integration_server.go`
- Modify: `internal/application/http_server.go`

- [x] разрешать путь к шаблонам устойчиво к cwd (`runtime.Caller` или поиск `go.mod` вверх), передавать его в конфиг сервера
- [x] перестать молча глотать ошибку инициализации веб-сервера в `http_server.go:118` — логировать всегда, а в тестовом хелпере падать через `t.Fatalf`
- [x] добавить хелпер `LoginAs(t, ts, user)`, возвращающий cookie сессии и CSRF-токен для последующих запросов
- [x] написать тест-страж: в тестовом сервере зарегистрированы `/login` (не 404) и CSRF-middleware (POST без токена → 403)
- [x] прогнать существующие интеграционные тесты — убедиться, что появление CSRF их не сломало, поправить где нужно
- [x] `make test` и `make lint` — 0 issues перед задачей 2

### Task 2: Регрессионный тест — анонимный доступ к API запрещён

TDD: тест пишется первым и **обязан упасть**, зафиксировав дыру S-01.

**Files:**
- Create: `tests/integration/api_auth_test.go`

- [x] табличный тест `TestAPIAuth_AnonymousRequestsRejected`: GET по `/api/v1/{users,categories,transactions,budgets,reports}` → ожидание `401`
- [x] кейсы записи: POST/PUT/DELETE **с валидным CSRF-токеном** и без сессии → ожидание `401` (без токена ожидается 403 — см. Technical Details)
- [x] воспроизвести сценарий обхода из аудита: получить анонимный CSRF-токен с `/login`, попытаться создать транзакцию
- [x] запустить и **убедиться, что тест падает** (сейчас будет 200/201/204) — зафиксировать вывод в комментарии к тесту
- [x] не переходить к задаче 3, пока падение не воспроизведено

⚠️ Красная фаза зафиксирована: 15 упавших подтестов (GET → 200, POST → 201,
PUT → 200, DELETE → 204), `WriteWithoutCSRFToken` → 403 как и ожидалось. Вывод
записан в шапке `tests/integration/api_auth_test.go`. Тест закрыт `t.Skip`
(константа `apiAuthSkipReason`), чтобы `make test` оставался зелёным для задач 3
и 4; **в задаче 4 skip обязан быть снят** — иначе дыра останется непроверенной.

### Task 3: Middleware RequireAPIAuth с ответом 401 JSON

**Files:**
- Modify: `internal/web/middleware/auth.go`
- Modify: `internal/web/middleware/auth_test.go`

- [x] добавить `RequireAPIAuth()` — при отсутствии сессии `401` и JSON-ошибка, без редиректа
- [x] формат ответа согласовать с `internal/application/handlers/errors.go`
- [x] при валидной сессии класть `SessionData` в контекст под ключом `"user"`, как `RequireAuth`
- [x] написать тесты: нет сессии → 401 и корректный JSON; валидная сессия → `next` вызван
- [x] написать тест: `Content-Type: application/json`, а не HTML
- [x] `make test` и `make lint` — 0 issues перед задачей 4

ℹ️ Формат ответа продублирован структурами `apiErrorResponse`/`apiErrorDetail`/`apiErrorMeta`
внутри `internal/web/middleware/auth.go`, а не взят импортом из
`internal/application/handlers`: импорт развернул бы направление зависимостей
(web → application) ради трёх полей. Тело ответа —
`{"error":{"code":"UNAUTHORIZED","message":"Authentication required"},"meta":{…,"version":"v1"}}`.

### Task 4: Включить аутентификацию на группе /api/v1

**Files:**
- Modify: `internal/application/http_server.go`
- Modify: `internal/application/http_server_test.go`
- Modify: `internal/application/handlers/families_test.go`
- Modify: `tests/integration/transactions_test.go`
- Modify: `tests/integration/budgets_test.go`
- Modify: `tests/integration/reports_test.go`
- Modify: `tests/integration/users_test.go`
- Modify: `tests/integration/categories_test.go`

- [x] навесить `webmw.RequireAPIAuth()` на группу `api` (`http_server.go:152`)
- [x] проверить, что `/health` и веб-маршруты не задеты
- [x] обновить ~70 вызовов `/api/v1` в 7 тестовых файлах: добавить сессию через `LoginAs` из задачи 1 (в т.ч. явное ожидание 200 в `http_server_test.go:759`)
- [x] снять `t.Skip(apiAuthSkipReason)` в `tests/integration/api_auth_test.go` и прогнать тест из задачи 2 — **теперь он обязан проходить**
- [x] написать тест: аутентифицированный запрос по-прежнему получает 200
- [x] `make test` и `make lint` — 0 issues перед задачей 5

ℹ️ Вызовы `/api/v1` в `tests/integration/*` уже несли сессию с задачи 1
(`testServer.Auth(t).Apply(req)`), править их не пришлось.
`internal/application/handlers/families_test.go` дёргает хендлеры напрямую, минуя
роутер, — middleware его не задевает. А сервер в `http_server_test.go` собран на
mock-репозиториях без веб-слоя, поэтому сессии в нём нет в принципе: ожидания
`/api/v1/categories` и `/api/v1/nonexistent` переведены на 401 с комментарием, а
аутентифицированный путь (200/201) проверяется на полном стеке —
`TestAPIAuth_AuthenticatedRequestsAllowed` в `tests/integration/api_auth_test.go`.
Побочный эффект: несуществующий путь внутри группы тоже отдаёт 401 (Echo вешает
group-middleware на catch-all маршрут группы) — так анонимный клиент не различает
существующие и несуществующие маршруты API.

### Task 5: Брать user_id из сессии, а не из тела запроса

Закрывает вторую половину S-01: без этого аутентифицированный пользователь может
писать данные от чужого имени.

**Files:**
- Modify: `internal/application/handlers/types.go`
- Modify: `internal/application/handlers/transactions.go`
- Modify: `internal/application/handlers/reports.go`
- Modify: `internal/application/handlers/transactions_test.go`
- Modify: `internal/services/dto/api_mappers.go`
- Modify: `internal/services/dto/api_mappers_test.go`

- [x] написать падающий тест: запрос с чужим `user_id` в теле создаёт запись от имени владельца сессии
- [x] удалить поле `UserID` из `CreateTransactionRequest` (`types.go:105`) и `CreateReportRequest` (`types.go:183`)
- [x] заменить `req.UserID` на значение из `GetUserFromContext(c)` в `transactions.go` **строки 83 и 130** и в `reports.go`
- [x] удалить мёртвый `dto.CreateTransactionAPIRequest` вместе с его тестом (ссылок из production-кода нет)
- [x] написать тесты: подмена `user_id` игнорируется; запись создаётся с ID из сессии
- [x] `make test` и `make lint` — 0 issues перед задачей 6

ℹ️ ID сессии достаётся хелпером `sessionUserID(c)` в
`internal/application/handlers/helpers.go` (обёртка над
`middleware.GetUserFromContext`), отказ отдаёт `respondUnauthorized` с кодом
`ErrCodeUnauthorized` = `UNAUTHORIZED` — тот же код, что и у `RequireAPIAuth`.
Проверка сессии стоит **до** `Bind`/валидации: анонимный клиент не должен
получать подсказки о формате тела. `reports.go` `req.UserID` нигде не читал
(маршрут отвечает 501), поэтому там добавлена только проверка сессии — задел на
момент, когда генерация отчётов появится. Регрессия на полном стеке —
`TestAPIAuth_BodyUserIDIgnored` в `tests/integration/api_auth_test.go`: проверяет
и ответ, и то, что легло в БД.

### Task 6: Ролевые проверки на разрушающих маршрутах API

Без этого S-01 закрыт наполовину: любой аутентифицированный пользователь, включая
роль `child`, сможет вызвать `DELETE /api/v1/users/:id`. В вебе такие маршруты
закрыты `RequireAdmin` (`internal/web/web.go:130`), в API — нет.

**Files:**
- Modify: `internal/application/http_server.go`
- Modify: `internal/web/middleware/auth.go`
- Create: `tests/integration/api_roles_test.go`

- [x] написать падающий тест: пользователь с ролью `child` вызывает `DELETE /api/v1/users/:id` → ожидание `403`
- [x] добавить API-вариант проверки роли (401/403 JSON вместо редиректа), переиспользуя `hasRequiredRole`
- [x] закрыть `RequireAdmin`-эквивалентом: `POST/PUT/DELETE /api/v1/users`, `DELETE /api/v1/categories/:id`
- [x] сверить набор маршрутов с ролевой моделью веба, чтобы поведение не расходилось
- [x] написать тесты для каждой роли: admin → 200/204, member и child → 403
- [x] `make test` и `make lint` — 0 issues перед задачей 7

ℹ️ Добавлены `RequireAPIRole(roles…)` и ярлыки `RequireAPIAdmin` /
`RequireAPIAdminOrMember` в `internal/web/middleware/auth.go`: нет сессии — 401,
роль не подходит — 403, тело в обоих случаях `apiErrorResponse` (общий хелпер
`respondAPIError`, код `FORBIDDEN` / `Insufficient permissions`).
Сверка с вебом дала на один шаг больше, чем перечислено в задаче: раз финансовые
разделы веба закрыты `RequireAdminOrMember` (`web.go:158-208`), те же группы
`/api/v1/{categories,transactions,budgets,reports}` закрыты
`RequireAPIAdminOrMember` — иначе роль `child` через API делает то, что ей
запрещено в UI. `GET /api/v1/users/:id` намеренно оставлен всем
аутентифицированным (в задаче названы только POST/PUT/DELETE).
Единственное осознанное расхождение с вебом: `DELETE /api/v1/categories/:id`
требует админа, тогда как в UI категорию удаляет и `member`, — так предписано
задачей, и через API удаление необратимо и без подтверждения.
Тесты: `tests/integration/api_roles_test.go` (матрица ролей на полном стеке,
красная фаза зафиксирована в шапке файла) и юнит-тесты `RequireAPIRole` в
`internal/web/middleware/auth_test.go`. Хелпер `TestServer.AuthAs(t, role)`
добавлен в `internal/testhelpers/integration_server.go` — заводит пользователя с
нужной ролью в **той же** семье.

### Task 7: Перевыпуск CSRF-токена при логине (S-02)

**Files:**
- Modify: `internal/web/middleware/csrf.go`
- Modify: `internal/web/handlers/auth.go`
- Modify: `internal/web/handlers/auth_security_test.go`

- [x] написать падающий тест: CSRF-токен, полученный анонимно, недействителен после успешного входа
- [x] добавить `RegenerateCSRFToken(c)` в `csrf.go` — серверного session ID не существует, хранилище cookie-based (`session.go:38`), поэтому перевыпускается именно токен
- [x] вызывать `ClearSession` перед `SetSessionData` в `AuthHandler.Login`, затем генерировать новый токен
- [x] написать тест: после логина токен изменился, старый отвергается с 403
- [x] написать тест: форма после логина содержит новый токен — вход через UI не сломан
- [x] `Logout` уже очищает токен через `ClearSession` (`session.go:113-127`) — только покрыть тестом, кода не трогать
- [x] `make test` и `make lint` — 0 issues перед задачей 8

⚠️ Пришлось тронуть ещё один файл — `internal/web/middleware/session.go`. `session.Get`
в рамках одного запроса возвращает **тот же** объект сессии, а `ClearSession`
выставляет на нём `Options.MaxAge = -1`. Без восстановления MaxAge связка
`ClearSession` → `SetSessionData` отдавала бы клиенту cookie на удаление, то есть
вход через UI ломался бы наглухо. Поэтому `SetSessionData` теперь восстанавливает
`MaxAge = SessionTimeout` перед сохранением; на `Logout` это не влияет —
там `ClearSession` вызывается последним.

ℹ️ Красная фаза зафиксирована: с откатанным `Login` подтесты
`TestLogin_SessionFixation/AnonymousTokenRejected` (403 против фактических 200) и
`/NewTokenIssuedAndAccepted` («после входа обязан выдаваться новый токен») падают.
Тесты в трёх слоях: юнит на middleware (`internal/web/middleware/csrf_test.go` —
`TestRegenerateCSRFToken_*`), настоящий обработчик входа на реальных
`SessionStore`+`CSRFProtection` со стаб-репозиторием
(`internal/web/handlers/auth_security_test.go` — заглушка `TestLogin_SessionFixation`
заменена рабочим тестом, добавлен `TestLogout_ClearsCSRFToken`) и полный стек с
настоящими шаблонами (➕ `tests/integration/auth_csrf_test.go`: токен из формы
`/admin/users` после входа отличается от анонимного и принимается сервером).
Проба «принимается ли токен» на полном стеке — `POST /login`: глобальный
`CSRFProtection` отрабатывает раньше маршрутного `RedirectIfAuthenticated`,
поэтому негодный токен даёт 403, а годный — 302, и состояние не мутируется.
Замечание для будущих тестов: за один запрос сессия сохраняется несколько раз,
клиент применяет `Set-Cookie` по порядку — брать надо **последнюю** cookie.
Про `Logout`: хранилище cookie-based, поэтому «отзыв» токена означает, что
выданная на выходе cookie пуста; переигранная старая cookie по-прежнему валидна
(ограничение модели, а не регрессия).

### Task 8: RequireSetup не трогает статику и /health (U-01)

**Files:**
- Modify: `internal/web/middleware/setup.go`
- Create: `internal/web/middleware/setup_test.go`

- [x] перейти с `c.Path()` на `c.Request().URL.Path` — `c.Path()` возвращает шаблон маршрута (`/static*`), а для незарегистрированного `/favicon.ico` вернёт не то
- [x] пропускать без проверок: префикс `/static/`, `/health`, `/favicon.ico`
- [x] написать тесты: при незавершённой настройке `/static/css/pico.min.css` → 200, `/health` → 200, `/favicon.ico` → не редирект
- [x] написать тест: при незавершённой настройке любой другой путь по-прежнему → 302 на `/setup`
- [x] написать тест: после завершения настройки `/setup` → 302 на `/login` (поведение не изменилось)
- [x] `make test` и `make lint` — 0 issues перед задачей 9

ℹ️ Исключения вынесены в `isSetupExempt(path)` (`setup.go`): точные `/health`,
`/favicon.ico`, `/static` плюс префикс `/static/`; пути-константы (`mnd`/`goconst`),
без пакетных переменных (`gochecknoglobals` — таблица-map потребовала бы `nolint`).
Проверка исключений стоит **до** обращения к `IsSetupComplete`, поэтому статика и
`/health` вообще не ходят в БД — это же поведение пригодится задаче 9.
Тесты — `internal/web/middleware/setup_test.go` (`package middleware_test`),
запросы идут через `e.ServeHTTP` с зарегистрированным маршрутом `/static*`, чтобы
`c.Path()` реально возвращал шаблон: на старом коде `/static/css/pico.min.css`
дал бы 302. Заглушка `stubSetupChecker` считает вызовы — задел под задачу 9.

### Task 9: Кэшировать IsSetupComplete (S-05)

**Files:**
- Modify: `internal/web/middleware/setup.go`
- Modify: `internal/web/middleware/setup_test.go`

- [x] завести `atomic.Bool` **внутри замыкания** `RequireSetup` — не пакетную переменную (`gochecknoglobals` включён)
- [x] после первого `true` в БД не обращаться
- [x] сохранить graceful degradation: при ошибке БД запрос пропускается, как сейчас
- [x] написать тест со счётчиком вызовов: второй запрос не дёргает `IsSetupComplete`
- [x] написать тест: до завершения настройки кэш не «залипает» на `false`
- [x] `make test` и `make lint` — 0 issues перед задачей 10

ℹ️ Кэш — `var setupComplete atomic.Bool` в теле `RequireSetup` (одна на middleware,
общая для всех запросов), логика вынесена в свободную функцию
`checkSetupCached(ctx, checker, cache) (complete, checked bool)`: инлайн-версия
подняла `gocognit` до 24 при пороге 20. Второе возвращаемое значение отличает
«проверили» от «БД недоступна» — так ошибка не попадает в кэш и не путается с
`false`. Кэшируется **только** `true`; ошибка и `false` не кэшируются, поэтому
переход «настройка завершена» виден без перезапуска процесса.
Тесты в `setup_test.go`: `TestRequireSetup_CompleteResultCached` (второй запрос
не увеличивает `checker.calls`, ветка `/setup` → `/login` работает из кэша),
`TestRequireSetup_IncompleteResultNotCached` (два редиректа — два обращения,
после `complete = true` кэш включается), а
`TestRequireSetup_CheckerError_GracefulDegradation` расширен проверкой, что
ошибка не кэшируется.

### Task 10: Починить секреты в compose-файлах (D-01, S-04)

**Files:**
- Modify: `docker/docker-compose.yml`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/docker-compose.nginx.yml`
- Modify: `deploy/docker-compose.caddy.yml`
- Modify: `deploy/docker-compose.minimal.yml`
- Modify: `.env.example`
- Modify: `deploy/.env.production.example`

- [x] добавить `CSRF_SECRET: ${CSRF_SECRET:?CSRF_SECRET is required}` в `docker/docker-compose.yml`
- [x] заменить `${CSRF_SECRET:-}` на `:?` в prod/nginx/caddy — пустая строка не должна проходить как «задано»
- [x] то же для `SESSION_SECRET` в `prod.yml:14` — там тоже нет `:?`
- [x] убрать `ENVIRONMENT=development` и значения `INSECURE_*` из `minimal.yml` (S-04)
- [x] добавить отсутствующий `CSRF_SECRET` в корневой `.env.example` (сейчас есть только `SESSION_SECRET`, строка 26)
- [x] решить вопрос расположения `.env`: compose v2 ищет его рядом с первым `-f`, то есть `docker/.env`, а `README.md:91` учит класть в корень — добавить `env_file` или описать явно
- [x] проверить: без секретов `docker compose config` падает с внятным сообщением; с секретами контейнер поднимается и `/health` → 200

ℹ️ **`env_file` не подходит** — он наполняет окружение контейнера, а падает
*интерполяция* `${VAR:?…}`, которая читает только shell-окружение и `.env` из
project directory. Поэтому выбран `--project-directory .`: он возвращает project
directory в корень репозитория, где и лежит `.env` (как учит README), и заодно
резолвит относительные пути (`DATA_DIR`) от корня. Прописан в `Makefile`
(переменная `DOCKER_COMPOSE`, заодно `docker-compose` v1 → `docker compose` v2),
в README и в шапке `docker/docker-compose.yml`. Следствие: `build.context`
в `docker/docker-compose.yml` пришлось поменять с `..` на `.` — он тоже
резолвится от project directory.

⚠️ Сообщение об ошибке в `:?` **не может содержать `: `** — compose парсит YAML до
интерполяции и валится на `mapping values are not allowed`. Формулировка:
``VAR is required - generate one with `openssl rand -base64 32` ``.

➕ Попутно найдено и починено во всех пяти файлах: healthcheck `wget --spider`
шлёт **HEAD**, а зарегистрирован только `GET /health` — HEAD проваливается в
редирект `/login` ⇄ `/setup`, wget выходит с кодом 8 и контейнер **навсегда
`unhealthy`** (проверено вживую). Заменено на `wget -O /dev/null` (обычный GET);
после правки контейнер становится `healthy` за ~9 секунд.

Проверено вживую (docker 29.6.2, compose v5.4.0):
- все 5 файлов без секретов и с пустыми строками → `config -q` даёт rc=1 и
  «required variable … is missing a value: …»;
- все 5 файлов с секретами → rc=0;
- `docker compose --project-directory . -f docker/docker-compose.yml up -d --build`
  → сборка проходит, `/health` → 200, `/` → 302 `/setup`, healthcheck `healthy`.

### Task 11: Сборка образа из исходников (D-02, D-03, D-04)

**Files:**
- Modify: `deploy/docker-compose.minimal.yml`
- Modify: `deploy/docker-compose.caddy.yml`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/docker-compose.nginx.yml` (➕ тот же мёртвый образ)
- Modify: `deploy/scripts/install.sh`
- Modify: `deploy/scripts/upgrade.sh`
- Modify: `deploy/scripts/lib/common.sh` (➕ там же живёт проверка RAM)
- Modify: `deploy/README.md`
- Modify: `deploy/.env.production.example`
- Modify: `docker/Dockerfile`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

- [x] перевести `minimal.yml` и `caddy.yml` на `build: {context: .., dockerfile: docker/Dockerfile}`
- [x] `prod.yml` переводить **только вместе** с правкой скриптов: `install.sh:156` кладёт compose в `/opt/family-budget` без исходников, а `install.sh:256` и `upgrade.sh:239` делают `docker compose pull` — для build-only сервиса это не сработает
- [x] в `install.sh` клонировать репозиторий в `$INSTALL_DIR/src` и заменить `pull` на `build`; то же в `upgrade.sh`
- [x] согласовать UID: `docker/Dockerfile` создаёт `/data` от `nobody` (65534), `prod.yml:39` запускает от `1000:1000` — иначе SQLite не откроет БД на bind-mount (D-03)
- [x] поправить ссылку на несуществующий `docs/tasks/002-reverse-proxy-config.md` в `install.sh:310` (D-04)
- [x] снизить требование «минимум 2 ГБ RAM» в `deploy/README.md` — по замерам хватает 128-256 МБ
- [x] добавить в `Makefile` или CI шаг `docker compose -f <файл> config -q` для всех пяти файлов, чтобы D-01 не воспроизвёлся при следующей правке

ℹ️ **Контекст сборки — переменная, а не фиксированный `..`.** `context: ${BUILD_CONTEXT:-..}`
во всех четырёх `deploy/*.yml`. Фиксированный `..` работает только когда файл
запускают на месте, из `deploy/`; `install.sh` кладёт compose в `/opt/family-budget`,
где `..` — это `/opt`. Значение по умолчанию сохраняет запуск из репозитория,
а `install.sh` пишет `BUILD_CONTEXT=./src` в `config/.env`. `image:` намеренно
не задан: у build-only сервиса `docker compose pull` не должен даже пытаться идти
в реестр.

ℹ️ **UID (D-03) согласован в сторону 1000, а не 65534.** `docker/Dockerfile` теперь
заводит пользователя `app` = 1000:1000 (`USER 1000:1000`) и отдаёт ему `/data`,
`/backups`, `/logs`; `prod.yml` со своим `user: "1000:1000"` совпал с образом без
правок. `install.sh` делает `chown 1000:1000` на смонтированные каталоги
**после** общего `chown -R $APP_USER` — системный `$APP_USER` имеет UID < 1000 и
для bind-mount не годится. Заодно в Dockerfile исправлен тот же `wget --spider`,
что и в compose-файлах (задача 10): HEALTHCHECK в самом образе оставался сломанным.

ℹ️ **`upgrade.sh` переведён с тегов образа на git-ref.** «Версия» — коммит в
`$INSTALL_DIR/src`: `--version` принимает тег/ветку/коммит (`latest` → `main` для
совместимости), `pull_new_version` стал `build_new_version` (fetch + checkout +
`compose build app`), `version.txt` хранит SHA, откат делает checkout этого SHA и
пересборку. `APP_VERSION` больше не экспортируется — подставлять его некуда.

➕ **`nginx.yml` тоже переведён на `build`**, хотя в задаче не назван: он ссылался
на тот же несуществующий образ, и без правки D-02 воспроизводился бы для
Option 2 из `deploy/README.md`.

➕ **Порог RAM живёт в `deploy/scripts/lib/common.sh`, а не в README** — правка
только документации оставила бы `install.sh` падать на 2 ГБ. Порог снижен до
**512 МБ**, а не до 128-256 МБ: 128-256 МБ — это потребление работающего сервиса,
но `install.sh` теперь **собирает образ на месте**, и компилятору Go нужен запас.
Оба числа записаны в README и в `--help`.

➕ `install_git()` в `common.sh` — исходники теперь обязательны, а `git` на чистой
машине может отсутствовать (ставится через apt/dnf по `$OS`).

Проверено вживую (docker 29.6.2, compose v5.4.0):
- `make compose-config` — все 5 файлов rc=0; с пустым `CSRF_SECRET` каждый из
  четырёх `deploy/*.yml` даёт rc=1 и внятное сообщение;
- `deploy/docker-compose.prod.yml` с `BUILD_CONTEXT` на корень репозитория:
  `docker compose build app` проходит, контейнер становится `healthy`,
  `id` внутри — `uid=1000(app) gid=1000(app)`, SQLite создал `budget.db` на
  bind-mount, `/health` → 200, `/` → 302, `/static/css/pico.min.css` → 200;
- встроенный fallback-compose из `install.sh` извлечён и проверен `compose config`
  — `context` резолвится в `./src`, `user: 1000:1000` на месте;
- `bash -n` на всех трёх изменённых скриптах; shellcheck (`-S warning`) — новых
  предупреждений нет, одно старое (SC2046) ушло.

⚠️ Полный прогон `install.sh` на чистом хосте не выполнялся (нужен root, systemd,
поддерживаемый дистрибутив и правка firewall) — проверены синтаксис, извлечённые
из скрипта артефакты и все docker-команды, которые он выполняет.

### Task 12: Тест реального рендеринга страниц-списков

Падающий тест, фиксирующий U-02. Ключевое требование: тест обязан проверять данные,
которые собирает **хендлер**, а не собранные вручную в самом тесте — иначе он
пройдёт при любом контракте и регрессию не поймает.

**Files:**
- Create: `tests/integration/web_pages_test.go`
- Modify: `internal/web/handlers/testhelpers_test.go`

- [x] написать тест через полный сервер из задачи 1: залогиниться, запросить `/transactions`, проверить наличие пунктов меню в HTML
- [x] распространить на `/categories`, `/budgets`, `/reports`
- [x] запустить и **убедиться, что тесты падают** для всех четырёх страниц
- [x] пометить `MockRenderer` комментарием о том, что он не проверяет шаблоны, чтобы им не закрывали будущие регрессии
- [x] не переходить к задаче 13, пока падение не воспроизведено

⚠️ Красная фаза зафиксирована: `TestWebPages_NavigationRendered` падает на всех
четырёх страницах, вывод записан в шапке `tests/integration/web_pages_test.go`.
Подтесты закрыты `t.Skip` (константы `webPagesSkipTransactions`,
`webPagesSkipCategories`, `webPagesSkipBudgets`, `webPagesSkipReports`), чтобы
`make test` оставался зелёным; **каждая из задач 13-15 обязана снять свой skip**
— соответствующие пункты добавлены в их чек-листы.

ℹ️ Проверяется не только набор ссылок, но и пользовательская часть шапки (имя
владельца сессии + `action="/logout"`) — иначе `/reports` прошёл бы и в красной
фазе: `pages/reports/index.html` рисует пункты меню **безусловно**, без
`{{if .CurrentUser}}`. Ассерты бьют по вырезанному блоку `<nav>…</nav>`, а не по
всей странице, чтобы совпадение с ссылками в теле страницы не давало ложного
зелёного.

⚠️ Следствие для задачи 15: `/reports` закрывается не только контрактом данных,
но и правкой самого `pages/reports/index.html` — блока с именем пользователя и
формой выхода там нет вовсе (в отличие от transactions/categories/budgets, где
он есть, но скрыт ложным `{{if .CurrentUser}}`).

### Task 13: Контракт данных — транзакции

**Files:**
- Modify: `internal/web/handlers/transactions.go`
- Modify: `internal/web/handlers/transactions_helpers.go`
- Modify: `internal/web/handlers/transactions_test.go`

- [x] заменить `map[string]any{"PageData": …}` на встроенную структуру по образцу `dashboard.go:110` (3 места в `transactions.go`, 1 в `transactions_helpers.go`)
- [x] встроенные поля ставить первыми и отделять пустой строкой (`embeddedstructfieldcheck`)
- [x] проверить, что `{{.PageData.X}}` в шаблонах продолжает работать — имя встроенного поля остаётся `PageData`
- [x] снять `t.Skip(webPagesSkipTransactions)` в `tests/integration/web_pages_test.go` и удалить константу
- [x] прогнать тест из задачи 12 — страница `/transactions` обязана пройти
- [x] написать тест на данные хендлера: `CurrentUser` доступен в корне контекста
- [x] `make test` и `make lint` — 0 issues перед задачей 14

ℹ️ Сборка `*PageData` вынесена в общий хелпер `BaseHandler.buildPageData(c, title)`
(`internal/web/handlers/base.go`): заголовок, flash-сообщения, CSRF-токен и
`CurrentUser`. Имени и фамилии в `middleware.SessionData` нет, поэтому они
дочитываются через `services.User.GetUserByID` — так же, как это делает
`DashboardHandler`. Хелпер положен в `base.go`, а не в `transactions.go`,
чтобы задачи 14-15 переиспользовали его и не породили пять почти одинаковых
блоков, на которые ругнётся `dupl` (замечание из «Development Approach»).

ℹ️ Побочно закрыт латентный баг из задачи 7: `{{.CSRFToken}}` в форме выхода
рендерился пустым, потому что хендлеры не клали токен в данные страницы.
`buildPageData` кладёт его в `PageData.CSRFToken`, откуда он промотируется в
корень контекста; проверено на полном стеке — `/transactions` отдаёт
`name="_token" value="…"` с непустым токеном. Прежние ключи `tplKeyCSRFToken`
в `New`/`Edit` транзакций больше не нужны.

ℹ️ Тест на контракт данных — `TestTransactionHandler_PageDataContract`
(`internal/web/handlers/transactions_test.go`): подтесты `Index`, `New`,
`FormWithErrors`. Он не сверяет структуру полем-по-полю, а **исполняет**
пробный шаблон `{{if .CurrentUser}}…{{.PageData.Title}}{{end}}` по данным,
которые хендлер отдал рендереру — то есть ловит ровно ту ошибку, что дала U-02.
Для этого в `internal/web/handlers/testhelpers_test.go` добавлен
`capturingRenderer` (в отличие от `MockRenderer` он данные запоминает) и
`newCapturingContext`; задачи 14-15 могут переиспользовать оба.

### Task 14: Контракт данных — категории и бюджеты

**Files:**
- Modify: `internal/web/handlers/categories.go`
- Modify: `internal/web/handlers/budgets.go`
- Modify: `internal/web/handlers/categories_test.go`
- Modify: `internal/web/handlers/budgets_test.go`

- [x] заменить map-контракт на встроенную структуру (5 мест в `categories.go`, 6 в `budgets.go`)
- [x] следить за `dupl`: почти одинаковые анонимные структуры в production-коде линтер может зацепить — при срабатывании вынести общий тип
- [x] снять `t.Skip(webPagesSkipCategories)` и `t.Skip(webPagesSkipBudgets)` в `tests/integration/web_pages_test.go` и удалить константы
- [x] прогнать тест из задачи 12 — `/categories` и `/budgets` обязаны пройти
- [x] написать тесты на данные хендлеров для обеих страниц
- [x] `make test` и `make lint` — 0 issues перед задачей 15

ℹ️ Общий тип заведён **сразу**, не дожидаясь `dupl`: `categoryFormData`
(`categories.go`) и `budgetFormData` (`budgets.go`) обслуживают по две страницы
формы каждый. `*PageData` собирается хелпером `BaseHandler.buildPageData` из
задачи 13, а для форм с ошибками добавлен `BaseHandler.formPageData(c, title,
errors)` (`base.go`) — он же подставляет общее сообщение
`formValidationMessage`; `transactions_helpers.go` переведён на него, чтобы
литерал не размножался (`goconst`).

⚠️ **Ключевое отличие структуры от map:** обращение к отсутствующему полю
структуры — **ошибка исполнения** шаблона, тогда как у map это молча давало
`<no value>`. Поэтому пришлось доложить в контракт всё, что шаблоны реально
читают, иначе страницы отдавали бы 500:
- `pages/categories/index.html` — `IncomeCount`/`ExpenseCount`/
  `WithSubcategoriesCount` (считает новый `countCategoriesByKind`; раньше в
  карточках сводки печаталось `<no value>`) и `Pagination` (пагинации на
  странице нет — поле `any`/nil, `{{if .Pagination}}` ложно, как и было);
- `pages/categories/new.html` — `{{if .Form}}`, поэтому `Form` в
  `categoryFormData` указатель: `nil` на пустой форме, `&form` на возврате
  с ошибками;
- `pages/budgets/new.html` — `{{.DefaultForm.Name}}` без `{{if}}`; на map
  возврат формы бюджета с ошибками валидации **падал**, теперь
  `renderBudgetFormWithErrors` кладёт и `DefaultForm`, и `BudgetID`
  (у него появились параметры `errors` и `budgetID`);
- `pages/budgets/show.html` — `{{if .Transactions}}`; поле оставлено nil, как
  было с map: строки таблицы читают `.FormattedDate` и `.UserName`, которых
  у `webModels.TransactionSummary` нет.

➕ Попутно переведён `CategoryHandler.Show` (`pages/categories/show.html`) —
в задаче он не назван, потому что в нём вообще не было `PageData`, но шапка
там та же и U-02 воспроизводился один в один.

⚠️ **`BudgetHandler.Alerts` осознанно оставлен на map — это 5 из 6 названных
мест в `budgets.go`.** `pages/budgets/alerts.html` написан под данные, которых
в коде нет вовсе: `.OverBudgetAlerts`/`.WarningAlerts`/`.ExpiredAlerts`/
`.Settings`/`.WarningCount`/`.NormalCount`/`.OverBudgetCount`, а карточки
читают `.OverspentFormatted`, `.SpentFormatted`, `.DaysExpired` — таких полей
нет ни у `BudgetAlertVM`, ни у `BudgetProgressVM`. Страница падает уже сейчас
(`.Settings.WarningThreshold` по отсутствующему ключу), и перевод на структуру
потребовал бы новой view-модели, то есть переписывания страницы. Причина
записана комментарием в самом хендлере.

ℹ️ Тесты на данные хендлеров — `TestCategoryHandler_PageDataContract` и
`TestBudgetHandler_PageDataContract` (подтесты `Index`, `New`, `Edit`, `Show`,
`FormWithErrors`) по образцу задачи 13: исполняют пробный шаблон по данным,
которые хендлер отдал рендереру. Пробный шаблон и `renderWith` переехали в
`testhelpers_test.go` как `pageDataProbe` + `renderPageData()` — с тремя
наборами тестов `unparam` начал ругаться на параметры, которые всегда получают
одно и то же значение (заодно `newCapturingContext` потерял параметр `body`).
`buildPageData` ходит в `UserService`, которого в `setupCategoryHandler`/
`setupBudgetHandler` не было, — добавлены варианты `…WithUser`, а старые
хелперы делегируют им и вешают разрешающий `.Maybe()`.

ℹ️ Побочно закрыт тот же латентный баг, что и в задаче 13: `{{.CSRFToken}}`
в форме выхода на `/categories` и `/budgets` рендерился пустым — теперь токен
приходит из `PageData`. Дашборд (`/`) по-прежнему затронут, но он ничей
чекбокс. Осиротевшие константы `tplKeyCategories`, `tplKeyFilters`,
`tplKeyCategory`, `tplKeyCategoryOptions`, `tplKeyDefaultColors`,
`tplKeyDefaultIcons` удалены из `template_keys.go`.

### Task 15: Контракт данных — отчёты и русские заголовки

**Files:**
- Modify: `internal/web/handlers/reports.go`
- Modify: `internal/web/handlers/transactions.go`
- Modify: `internal/web/handlers/categories.go`
- Modify: `internal/web/handlers/budgets.go`
- Modify: `internal/web/handlers/reports_test.go`
- Modify: `internal/web/templates/pages/reports/index.html` (➕ в шапке нет блока пользователя — см. задачу 12)

- [ ] заменить map-контракт на встроенную структуру (4 места в `reports.go`)
- [ ] перевести заголовки: `Transactions`→`Транзакции`, `Budgets`→`Бюджеты`, `Categories`→`Категории`, `Reports`→`Отчёты`, а также формы `New/Edit …` (U-05)
- [ ] добавить в `pages/reports/index.html` блок `{{if .CurrentUser}}` с именем пользователя и формой выхода по образцу `pages/transactions/index.html` — одного контракта данных для `/reports` мало
- [ ] снять `t.Skip(webPagesSkipReports)` в `tests/integration/web_pages_test.go` и удалить константу
- [ ] прогнать тест из задачи 12 — `/reports` обязан пройти
- [ ] расширить тест из задачи 12: заголовок страницы на русском
- [ ] `make test` и `make lint` — 0 issues перед задачей 16

### Task 16: Убрать мёртвую ссылку /register

**Files:**
- Modify: `internal/web/templates/pages/login.html`
- Modify: `internal/web/templates/components/nav.html`
- Modify: `internal/web/templates/pages/dashboard.html`
- Delete: `internal/web/templates/pages/register.html`

- [ ] убрать ссылку «Создать семейный аккаунт» с `/login` — регистрация идёт по инвайтам
- [ ] убрать пункт «Регистрация» из `nav.html` и `dashboard.html`
- [ ] удалить осиротевший `pages/register.html` (проверено: `layouts/auth.html:41-42` ветвится только на `login_content`/`setup_content`, удаление безопасно)
- [ ] написать тест: страница `/login` не содержит ссылок на несуществующие маршруты
- [ ] `make test` и `make lint` — 0 issues перед задачей 17

### Task 17: Verify acceptance criteria

- [ ] анонимный GET ко всем `/api/v1/*` → 401
- [ ] анонимный POST/PUT/DELETE с валидным CSRF-токеном → 401; без токена → 403
- [ ] подмена `user_id` в теле запроса игнорируется
- [ ] роль `child` не может удалять пользователей и категории через API → 403
- [ ] на чистой БД: `/setup` открывается **со стилями**, `/health` → 200 до завершения настройки
- [ ] навигация присутствует на `/transactions`, `/categories`, `/budgets`, `/reports`; заголовки по-русски
- [ ] `docker compose -f docker/docker-compose.yml up` поднимается с секретами и падает с внятной ошибкой без них
- [ ] полный прогон `make test` — все пакеты зелёные; `make lint` — 0 issues
- [ ] покрытие не ниже исходных 61.9%: `make test-coverage`

### Task 18: [Final] Обновить документацию

- [ ] отметить закрытые находки в `docs/specs/002-security-audit.md` и `004-deployment-readiness.md`
- [ ] обновить «API Readiness» в `README.md` — API требует аутентификации и CSRF-токена
- [ ] обновить `CLAUDE.md`: убрать предупреждение «группа `/api/v1` регистрируется без auth middleware», описать починенный тестовый хелпер
- [ ] описать в `deploy/README.md` первый запуск через SSH-туннель и порядок работы программного клиента с CSRF
- [ ] поставить тег `v0.1.0`, убедиться, что `docker.yml` опубликовал multi-arch образ
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

*Требует ручных действий вне репозитория*

**Ручная проверка перед выкатом:**
- сценарий в браузере на чистой БД: setup → вход → категория → транзакция → список
- проверить на телефоне — навигация останется неудобной ([U-03](../specs/003-ui-ux-audit.md#u-03) вне объёма)
- снять бэкап и **выполнить восстановление** — бэкап без проверенного восстановления бэкапом не является

**Выкат на mini:**
- первый запуск только в локальную сеть или через SSH-туннель: `ssh -L 8080:127.0.0.1:8080 mini`
- публичный домен и Caddy — только после закрытия задач 2-6
- внешний мониторинг `/health`
- проверить fail2ban на реальных логах — эффективность jail'а не подтверждена ([S-03](../specs/002-security-audit.md#s-03))

**Следующие планы:**
- rate limiting на логин в приложении (S-03)
- слой макета: оживить `base.html`, убрать 20 копий вёрстки (U-04), мобильное меню (U-03)
- дизайн-направление из [003](../specs/003-ui-ux-audit.md)
- API-токены для внешних интеграций
- генерация отчётов через API (сейчас `POST /api/v1/reports` → 501)
