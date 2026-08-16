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

- [ ] табличный тест `TestAPIAuth_AnonymousRequestsRejected`: GET по `/api/v1/{users,categories,transactions,budgets,reports}` → ожидание `401`
- [ ] кейсы записи: POST/PUT/DELETE **с валидным CSRF-токеном** и без сессии → ожидание `401` (без токена ожидается 403 — см. Technical Details)
- [ ] воспроизвести сценарий обхода из аудита: получить анонимный CSRF-токен с `/login`, попытаться создать транзакцию
- [ ] запустить и **убедиться, что тест падает** (сейчас будет 200/201/204) — зафиксировать вывод в комментарии к тесту
- [ ] не переходить к задаче 3, пока падение не воспроизведено

### Task 3: Middleware RequireAPIAuth с ответом 401 JSON

**Files:**
- Modify: `internal/web/middleware/auth.go`
- Modify: `internal/web/middleware/auth_test.go`

- [ ] добавить `RequireAPIAuth()` — при отсутствии сессии `401` и JSON-ошибка, без редиректа
- [ ] формат ответа согласовать с `internal/application/handlers/errors.go`
- [ ] при валидной сессии класть `SessionData` в контекст под ключом `"user"`, как `RequireAuth`
- [ ] написать тесты: нет сессии → 401 и корректный JSON; валидная сессия → `next` вызван
- [ ] написать тест: `Content-Type: application/json`, а не HTML
- [ ] `make test` и `make lint` — 0 issues перед задачей 4

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

- [ ] навесить `webmw.RequireAPIAuth()` на группу `api` (`http_server.go:152`)
- [ ] проверить, что `/health` и веб-маршруты не задеты
- [ ] обновить ~70 вызовов `/api/v1` в 7 тестовых файлах: добавить сессию через `LoginAs` из задачи 1 (в т.ч. явное ожидание 200 в `http_server_test.go:759`)
- [ ] прогнать тест из задачи 2 — **теперь он обязан проходить**
- [ ] написать тест: аутентифицированный запрос по-прежнему получает 200
- [ ] `make test` и `make lint` — 0 issues перед задачей 5

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

- [ ] написать падающий тест: запрос с чужим `user_id` в теле создаёт запись от имени владельца сессии
- [ ] удалить поле `UserID` из `CreateTransactionRequest` (`types.go:105`) и `CreateReportRequest` (`types.go:183`)
- [ ] заменить `req.UserID` на значение из `GetUserFromContext(c)` в `transactions.go` **строки 83 и 130** и в `reports.go`
- [ ] удалить мёртвый `dto.CreateTransactionAPIRequest` вместе с его тестом (ссылок из production-кода нет)
- [ ] написать тесты: подмена `user_id` игнорируется; запись создаётся с ID из сессии
- [ ] `make test` и `make lint` — 0 issues перед задачей 6

### Task 6: Ролевые проверки на разрушающих маршрутах API

Без этого S-01 закрыт наполовину: любой аутентифицированный пользователь, включая
роль `child`, сможет вызвать `DELETE /api/v1/users/:id`. В вебе такие маршруты
закрыты `RequireAdmin` (`internal/web/web.go:130`), в API — нет.

**Files:**
- Modify: `internal/application/http_server.go`
- Modify: `internal/web/middleware/auth.go`
- Create: `tests/integration/api_roles_test.go`

- [ ] написать падающий тест: пользователь с ролью `child` вызывает `DELETE /api/v1/users/:id` → ожидание `403`
- [ ] добавить API-вариант проверки роли (401/403 JSON вместо редиректа), переиспользуя `hasRequiredRole`
- [ ] закрыть `RequireAdmin`-эквивалентом: `POST/PUT/DELETE /api/v1/users`, `DELETE /api/v1/categories/:id`
- [ ] сверить набор маршрутов с ролевой моделью веба, чтобы поведение не расходилось
- [ ] написать тесты для каждой роли: admin → 200/204, member и child → 403
- [ ] `make test` и `make lint` — 0 issues перед задачей 7

### Task 7: Перевыпуск CSRF-токена при логине (S-02)

**Files:**
- Modify: `internal/web/middleware/csrf.go`
- Modify: `internal/web/handlers/auth.go`
- Modify: `internal/web/handlers/auth_security_test.go`

- [ ] написать падающий тест: CSRF-токен, полученный анонимно, недействителен после успешного входа
- [ ] добавить `RegenerateCSRFToken(c)` в `csrf.go` — серверного session ID не существует, хранилище cookie-based (`session.go:38`), поэтому перевыпускается именно токен
- [ ] вызывать `ClearSession` перед `SetSessionData` в `AuthHandler.Login`, затем генерировать новый токен
- [ ] написать тест: после логина токен изменился, старый отвергается с 403
- [ ] написать тест: форма после логина содержит новый токен — вход через UI не сломан
- [ ] `Logout` уже очищает токен через `ClearSession` (`session.go:113-127`) — только покрыть тестом, кода не трогать
- [ ] `make test` и `make lint` — 0 issues перед задачей 8

### Task 8: RequireSetup не трогает статику и /health (U-01)

**Files:**
- Modify: `internal/web/middleware/setup.go`
- Create: `internal/web/middleware/setup_test.go`

- [ ] перейти с `c.Path()` на `c.Request().URL.Path` — `c.Path()` возвращает шаблон маршрута (`/static*`), а для незарегистрированного `/favicon.ico` вернёт не то
- [ ] пропускать без проверок: префикс `/static/`, `/health`, `/favicon.ico`
- [ ] написать тесты: при незавершённой настройке `/static/css/pico.min.css` → 200, `/health` → 200, `/favicon.ico` → не редирект
- [ ] написать тест: при незавершённой настройке любой другой путь по-прежнему → 302 на `/setup`
- [ ] написать тест: после завершения настройки `/setup` → 302 на `/login` (поведение не изменилось)
- [ ] `make test` и `make lint` — 0 issues перед задачей 9

### Task 9: Кэшировать IsSetupComplete (S-05)

**Files:**
- Modify: `internal/web/middleware/setup.go`
- Modify: `internal/web/middleware/setup_test.go`

- [ ] завести `atomic.Bool` **внутри замыкания** `RequireSetup` — не пакетную переменную (`gochecknoglobals` включён)
- [ ] после первого `true` в БД не обращаться
- [ ] сохранить graceful degradation: при ошибке БД запрос пропускается, как сейчас
- [ ] написать тест со счётчиком вызовов: второй запрос не дёргает `IsSetupComplete`
- [ ] написать тест: до завершения настройки кэш не «залипает» на `false`
- [ ] `make test` и `make lint` — 0 issues перед задачей 10

### Task 10: Починить секреты в compose-файлах (D-01, S-04)

**Files:**
- Modify: `docker/docker-compose.yml`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/docker-compose.nginx.yml`
- Modify: `deploy/docker-compose.caddy.yml`
- Modify: `deploy/docker-compose.minimal.yml`
- Modify: `.env.example`
- Modify: `deploy/.env.production.example`

- [ ] добавить `CSRF_SECRET: ${CSRF_SECRET:?CSRF_SECRET is required}` в `docker/docker-compose.yml`
- [ ] заменить `${CSRF_SECRET:-}` на `:?` в prod/nginx/caddy — пустая строка не должна проходить как «задано»
- [ ] то же для `SESSION_SECRET` в `prod.yml:14` — там тоже нет `:?`
- [ ] убрать `ENVIRONMENT=development` и значения `INSECURE_*` из `minimal.yml` (S-04)
- [ ] добавить отсутствующий `CSRF_SECRET` в корневой `.env.example` (сейчас есть только `SESSION_SECRET`, строка 26)
- [ ] решить вопрос расположения `.env`: compose v2 ищет его рядом с первым `-f`, то есть `docker/.env`, а `README.md:91` учит класть в корень — добавить `env_file` или описать явно
- [ ] проверить: без секретов `docker compose config` падает с внятным сообщением; с секретами контейнер поднимается и `/health` → 200

### Task 11: Сборка образа из исходников (D-02, D-03, D-04)

**Files:**
- Modify: `deploy/docker-compose.minimal.yml`
- Modify: `deploy/docker-compose.caddy.yml`
- Modify: `deploy/docker-compose.prod.yml`
- Modify: `deploy/scripts/install.sh`
- Modify: `deploy/scripts/upgrade.sh`
- Modify: `deploy/README.md`

- [ ] перевести `minimal.yml` и `caddy.yml` на `build: {context: .., dockerfile: docker/Dockerfile}`
- [ ] `prod.yml` переводить **только вместе** с правкой скриптов: `install.sh:156` кладёт compose в `/opt/family-budget` без исходников, а `install.sh:256` и `upgrade.sh:239` делают `docker compose pull` — для build-only сервиса это не сработает
- [ ] в `install.sh` клонировать репозиторий в `$INSTALL_DIR/src` и заменить `pull` на `build`; то же в `upgrade.sh`
- [ ] согласовать UID: `docker/Dockerfile` создаёт `/data` от `nobody` (65534), `prod.yml:39` запускает от `1000:1000` — иначе SQLite не откроет БД на bind-mount (D-03)
- [ ] поправить ссылку на несуществующий `docs/tasks/002-reverse-proxy-config.md` в `install.sh:310` (D-04)
- [ ] снизить требование «минимум 2 ГБ RAM» в `deploy/README.md` — по замерам хватает 128-256 МБ
- [ ] добавить в `Makefile` или CI шаг `docker compose -f <файл> config -q` для всех пяти файлов, чтобы D-01 не воспроизвёлся при следующей правке

### Task 12: Тест реального рендеринга страниц-списков

Падающий тест, фиксирующий U-02. Ключевое требование: тест обязан проверять данные,
которые собирает **хендлер**, а не собранные вручную в самом тесте — иначе он
пройдёт при любом контракте и регрессию не поймает.

**Files:**
- Create: `tests/integration/web_pages_test.go`
- Modify: `internal/web/handlers/testhelpers_test.go`

- [ ] написать тест через полный сервер из задачи 1: залогиниться, запросить `/transactions`, проверить наличие пунктов меню в HTML
- [ ] распространить на `/categories`, `/budgets`, `/reports`
- [ ] запустить и **убедиться, что тесты падают** для всех четырёх страниц
- [ ] пометить `MockRenderer` комментарием о том, что он не проверяет шаблоны, чтобы им не закрывали будущие регрессии
- [ ] не переходить к задаче 13, пока падение не воспроизведено

### Task 13: Контракт данных — транзакции

**Files:**
- Modify: `internal/web/handlers/transactions.go`
- Modify: `internal/web/handlers/transactions_helpers.go`
- Modify: `internal/web/handlers/transactions_test.go`

- [ ] заменить `map[string]any{"PageData": …}` на встроенную структуру по образцу `dashboard.go:110` (3 места в `transactions.go`, 1 в `transactions_helpers.go`)
- [ ] встроенные поля ставить первыми и отделять пустой строкой (`embeddedstructfieldcheck`)
- [ ] проверить, что `{{.PageData.X}}` в шаблонах продолжает работать — имя встроенного поля остаётся `PageData`
- [ ] прогнать тест из задачи 12 — страница `/transactions` обязана пройти
- [ ] написать тест на данные хендлера: `CurrentUser` доступен в корне контекста
- [ ] `make test` и `make lint` — 0 issues перед задачей 14

### Task 14: Контракт данных — категории и бюджеты

**Files:**
- Modify: `internal/web/handlers/categories.go`
- Modify: `internal/web/handlers/budgets.go`
- Modify: `internal/web/handlers/categories_test.go`
- Modify: `internal/web/handlers/budgets_test.go`

- [ ] заменить map-контракт на встроенную структуру (5 мест в `categories.go`, 6 в `budgets.go`)
- [ ] следить за `dupl`: почти одинаковые анонимные структуры в production-коде линтер может зацепить — при срабатывании вынести общий тип
- [ ] прогнать тест из задачи 12 — `/categories` и `/budgets` обязаны пройти
- [ ] написать тесты на данные хендлеров для обеих страниц
- [ ] `make test` и `make lint` — 0 issues перед задачей 15

### Task 15: Контракт данных — отчёты и русские заголовки

**Files:**
- Modify: `internal/web/handlers/reports.go`
- Modify: `internal/web/handlers/transactions.go`
- Modify: `internal/web/handlers/categories.go`
- Modify: `internal/web/handlers/budgets.go`
- Modify: `internal/web/handlers/reports_test.go`

- [ ] заменить map-контракт на встроенную структуру (4 места в `reports.go`)
- [ ] перевести заголовки: `Transactions`→`Транзакции`, `Budgets`→`Бюджеты`, `Categories`→`Категории`, `Reports`→`Отчёты`, а также формы `New/Edit …` (U-05)
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
