# План 03 — Bearer-аутентификация и удаление веб-слоя

Третий план перехода на API-only ([spec 005](../specs/005-api-only-redesign.md), решения
A-01…A-04, A-09, A-12). После него сервис отвечает только JSON, а в конфиге не остаётся секретов.

## Overview

- Новый пакет `internal/auth`: токены, таблица `sessions`, bearer-middleware, ролевые гейты,
  лимитер логина. Он заменяет `internal/web/middleware` и `internal/application/handlers/api_auth.go`.
- Роуты `/auth/*`, `/me`, `/me/password`, `PATCH /users/:id`, `PUT /users/:id/password`.
- Bootstrap семьи и сброс пароля — подкоманды CLI, одной транзакцией.
- `internal/web/**`, шаблоны, статика, `echo-contrib`, `gorilla/sessions`, `RequireSetup`,
  CSRF и cookie-конфиг удаляются; `testhelpers` переводится на bearer.

## Context (from discovery)

- Сессии сейчас: `internal/web/middleware/session.go` (cookie store), `auth.go`
  (`RequireAuth`, `RevalidateSessionUser`, `RequireRole`), `csrf.go`, `setup.go`.
  API-обёртки: `internal/application/handlers/api_auth.go`; сессию читают
  `transactions.go:47-57` и `users.go:248-256` (только `UserID`).
- Проверка пароля — только в `internal/web/handlers/auth.go:129`; хеширование в трёх сервисах
  (`user_service.go:88`, `family_service.go:79`, `invite_service.go:290`), cost 10.
- `SetupFamily` (`internal/services/family_service.go:44-92`) — три операции без транзакции.
  Единственный репозиторий с транзакциями — `transaction_repository_sqlite.go`; соединение к SQLite
  одно (`internal/infrastructure/sqlite.go:46`), поэтому `BEGIN … COMMIT` через `*sql.DB` безопасен.
- Открытие БД и миграции — внутри `NewApplication` (`internal/run.go:61-73`); CLI сейчас — только
  `-health-check` (`cmd/server/main.go:26-38`).
- `testhelpers.LoginAs` подписывает cookie без БД (`integration_server.go:159-189`);
  `Apply` ставит cookie + `X-Csrf-Token`; 76 вызовов в `tests/integration`.
- Веб-специфичные интеграционные тесты: `auth_csrf_test.go`, `error_page_test.go`, `invites_test.go`,
  `login_links_test.go`, `web_pages_test.go`, `web_render_test.go`,
  `web_transactions_pagination_test.go`; смешанные: `api_auth_test.go:56-140,363-380`,
  `session_revalidation_test.go`.
- Echo не настраивает `IPExtractor` (`http_server.go:82-116`); `CORS()` включён с `*`.
- `.golangci.yml:456-473` — исключения для `internal/web/handlers` и `internal/web/models`.

## Development Approach

- **testing approach**: Regular.
- Задачи 1–6 добавляют bearer-путь рядом с cookie-путём: оба работают, `make test` зелёный
  после каждой. Задача 7 переключает тесты на bearer, задача 8 удаляет веб. Так удаление —
  один коммит с заранее зелёными тестами.
- Тесты — отдельными пунктами в каждой задаче.

## Testing Strategy

- unit: `internal/auth/*_test.go` (токены, лимитер, middleware через `httptest`),
  репозиторий сессий на `testhelpers.SetupSQLiteTestDB`.
- integration: `tests/integration/auth_test.go` — полный цикл login → запрос → logout → 401;
  `testhelpers.LoginAs` выдаёт токен через `auth.Service`, `Apply` ставит `Authorization`.
- Роуты — в `openapi.yaml` (тест покрытия из плана 01).

## Progress Tracking

- `[x]` по завершении; ➕ новые задачи; ⚠️ блокеры.

## Solution Overview

`internal/auth` не зависит ни от `web`, ни от `application`; его импортируют `application/handlers`
и CLI. Контекстный ключ и `GetUserFromContext` переезжают сюда — обратное ребро
`application → web` из `docs/backlog.md` исчезает.

Проверка токена — один запрос `SELECT … FROM sessions s JOIN users u ON … WHERE s.token_hash = ?`,
который возвращает срок сессии, `is_active` и роль. `last_used_at` обновляется, только если
прошёл час с прошлого обновления (запись в SQLite сериализует чтения).

## Technical Details

- Таблица (добавить в конец `001_consolidated.up.sql`, `DROP` — в начало `.down.sql`; план 04
  перепишет файл целиком):
  `sessions(id TEXT PK, user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE, token_hash TEXT NOT NULL UNIQUE, device_name TEXT NOT NULL, created_at, last_used_at, expires_at DATETIME NOT NULL)`
  + индекс по `user_id`. Старую `user_sessions` удалить.
- Токен: 32 байта `crypto/rand`, `base64.RawURLEncoding`; в БД `hex(sha256)`. Срок: `expires_at =
  last_used_at + 30d`, но не позже `created_at + 180d`. Константы — в `internal/auth/token.go`.
- Пароль: bcrypt cost 12; политика 10…72 байта — одна функция `auth.ValidatePassword`, её
  используют DTO создания пользователя, CLI и смена пароля. Неизвестный email → сравнение с
  фиктивным хешем (константа, сгенерированная при старте) → `401 INVALID_CREDENTIALS`.
- Ответ логина: `{token, user: {id, email, first_name, last_name, role}}`.
- Лимитер: `internal/auth/ratelimit.go` — sliding window в памяти с мьютексом; per-IP 10 попыток
  за 5 минут, per-email 20 за час; успешный логин сбрасывает счётчик email; `429` + `Retry-After`;
  уборка старых записей при каждом N-м вызове. IP — `echo.ExtractIPFromXFFHeader(TrustOptions)`
  с `TRUSTED_PROXIES` (CIDR через запятую, по умолчанию пусто = доверять только `RemoteAddr`).
- Лог неудачи: `logger.WarnContext(ctx, "login failed", "email", email, "ip", ip, "reason", …)`.
- Роуты: публичная группа `/api/v1/auth` (`POST /login`); защищённая группа `/api/v1` с
  `auth.RequireBearer(service)` — `POST /auth/logout`, `GET /auth/sessions`, `DELETE /auth/sessions/:id`,
  `GET|PUT /me`, `PUT /me/password`; admin — `PATCH /users/:id {is_active|role}`,
  `PUT /users/:id/password {new_password}`.
- До setup: `/health` → `checks.setup: false` и поле `setup_complete`; логин → `409 SETUP_REQUIRED`.
- CLI (`cmd/server/main.go`, `flag.NewFlagSet` на подкоманду): `setup --family --currency --timezone
  --email --first-name --last-name --password-stdin`, `reset-password --email --password-stdin`.
  Общая функция `internal.OpenDatabase(cfg)` (БД + миграции) для сервера и CLI.
- Конфиг после чистки: `SERVER_*`, `DATABASE_PATH`, `LOG_*`, `ENVIRONMENT`, `BACKUP_DIR`,
  `TRUSTED_PROXIES`. `Validate()` теряет проверки секретов.

## Implementation Steps

### Task 1: Домен и репозиторий сессий

**Files:**
- Create: `internal/auth/session.go`, `internal/infrastructure/auth/session_repository_sqlite.go`,
  `internal/infrastructure/auth/session_repository_test.go`
- Modify: `migrations/001_consolidated.up.sql`, `001_consolidated.down.sql`,
  `internal/testhelpers/sqlite.go` (`CleanTables`), `internal/infrastructure/repositories_sqlite.go`

- [x] таблица `sessions` в миграции, `user_sessions` удалить; `CleanTables` обновить (заодно добавить отсутствующую там `invites`)
- [x] `make db-reset` (удаляет `./data/budget.db*`) и правило в `migrations/README.md` + `CLAUDE.md`: golang-migrate хранит только номер версии, правка уже применённой `001` на существующей БД — no-op (`Up()` → `ErrNoChange`), тестовый путь этого не покажет; до первого релиза схема меняется переписыванием `001`, локальные и серверные БД пересоздаются
- [x] `auth.Session` (поля из «Technical Details») и `auth.SessionRepository`: `Create`, `FindByTokenHash` (JOIN users → `Session` + `user.User`), `Touch(id, at)`, `Delete(id)`, `DeleteByUser(userID, exceptID)`, `ListByUser`, `DeleteExpired(now)`
- [x] реализация на SQLite по образцу `internal/infrastructure/user/user_repository_sqlite.go`
- [x] тесты репозитория: создание и поиск, неизвестный хеш → `ErrSessionNotFound`, `DeleteByUser` сохраняет исключение, `DeleteExpired` удаляет только просроченные (CASCADE не тестировать — приложение пользователей не удаляет, см. задачу 5)
- [x] `make test` — зелёный (обе ветки миграций: golang-migrate и `testhelpers`)

### Task 2: `auth.Service` — выдача, проверка и отзыв токенов

**Files:**
- Create: `internal/auth/token.go`, `internal/auth/service.go`, `internal/auth/password.go`,
  `internal/auth/service_test.go`, `internal/auth/password_test.go`
- Modify: `internal/services/user_service.go`, `family_service.go`, `internal/services/dto/user_dto.go`

- [x] направление зависимостей: `internal/auth` не импортирует `internal/services`. Ему нужны три узких интерфейса, объявленных в `internal/auth`: `SessionRepository` (задача 1), `UserLookup` (`GetByEmail`, `GetByID`, `UpdatePassword`) и `SetupChecker` (`Exists`); их реализуют репозитории из `internal/infrastructure/user`. `internal/services` импортирует `auth` только ради `HashPassword`/`ValidatePassword`
- [x] `GenerateToken() (plain, hash string)`, `HashToken(plain)`; константы `IdleTTL = 30d`, `AbsoluteTTL = 180d`, `TouchInterval = 1h`
- [x] `auth.HashPassword` (cost 12), `auth.ComparePassword`, `auth.ValidatePassword` (10…72 байта); сервисы и DTO переходят на них, `min=6` заменить
- [x] `Service.Login(ctx, email, password, device) (token, *user.User, error)`: фиктивный хеш для неизвестного email — поле `Service.dummyHash`, вычисляется в `NewService` (`gochecknoglobals` запрещает package-level var), `ErrInvalidCredentials`, `ErrSetupRequired` (семьи нет), неактивный пользователь → `ErrInvalidCredentials`
- [x] `Service.Authenticate(ctx, token) (*Principal, error)`: истёк → удалить и `ErrUnauthorized`; `Touch`, только если `now - last_used_at > TouchInterval`
- [x] `Logout`, `ListSessions`, `RevokeSession(userID, id)`, `ChangePassword(userID, current, new, keepSessionID)` — проверка `current`, новый хеш, `DeleteByUser(except)`; `AdminSetPassword(userID, new)` — отзыв всех сессий
- [x] тесты: успех, неверный пароль, неизвестный email (одинаковый ответ), просроченный токен, touch не пишет в БД раньше интервала (мок репозитория считает вызовы), смена пароля с неверным `current` → ошибка и сессии целы
- [x] `make test` — зелёный

### Task 3: Middleware и лимитер

**Files:**
- Create: `internal/auth/middleware.go`, `internal/auth/middleware_test.go`,
  `internal/auth/ratelimit.go`, `internal/auth/ratelimit_test.go`
- Modify: `internal/application/handlers/api_auth.go` (временно делегирует), `internal/config.go`,
  `internal/application/http_server.go`

- [x] `auth.ContextKey`, `auth.Principal{UserID, Role, Email}`, `auth.FromContext(c)`; `RequireBearer(service)` — `401 UNAUTHORIZED` без заголовка / с плохим токеном; `RequireRole(roles...)` — `403 FORBIDDEN`
- [x] `RateLimiter` (окно, лимиты, `Allow(ip, email) (retryAfter time.Duration, ok bool)`, `Reset(email)`); `TRUSTED_PROXIES` → `e.IPExtractor`
- [x] `api_auth.go`: `RequireAPIAuth`/`RequireAPIActiveUser`/`RequireAPIRole` принимают и cookie, и bearer (проверить `Authorization` первым) — чтобы оба пути жили до задачи 8
- [x] тесты middleware через `httptest`: нет заголовка, мусор, валидный токен → principal в контексте, роль не та → 403
- [x] тесты лимитера: 11-я попытка с IP → блок, `Retry-After` > 0, окно сдвигается, `Reset` снимает блок по email, IP из XFF учитывается только от доверенного прокси
- [x] `make test` — зелёный

### Task 4: Роуты `/auth/*` и `/me`

**Files:**
- Create: `internal/application/handlers/auth.go`, `auth_test.go`, `me.go`, `me_test.go`,
  `tests/integration/auth_test.go`
- Modify: `internal/application/http_server.go`, `internal/application/handlers/types.go`,
  `internal/observability/health.go`, `internal/run.go`, `docs/api/openapi.yaml`

- [x] публичная группа `/api/v1/auth`: `POST /login` — лимитер → `Service.Login` → `200 {token, user}`; `401 INVALID_CREDENTIALS`, `409 SETUP_REQUIRED`, `429`
- [x] защищённые: `POST /auth/logout`, `GET /auth/sessions` (текущая помечена `current: true`), `DELETE /auth/sessions/:id` (чужая → 404), `GET /me`, `PUT /me {first_name, last_name, email}`, `PUT /me/password`
- [x] `/health`: `setup_complete` через `FamilyService.IsSetupComplete`
- [x] тесты обработчиков (успех, ошибки, коды) и интеграционный цикл: login → `GET /me` → смена пароля → старый токен другой сессии 401, текущий жив → logout → 401
- [x] `openapi.yaml` обновлён; `make fmt && make test && make lint` — зелёные

### Task 5: Администрирование пользователей

**Files:**
- Modify: `internal/domain/user/user.go`, `internal/infrastructure/user/user_repository_sqlite.go`,
  `user_repository_test.go`, `internal/services/user_service.go`, `user_service_test.go`,
  `internal/application/handlers/users.go`, `users_test.go`, `tests/integration/users_test.go`,
  `docs/api/openapi.yaml`

- [x] `user.User.IsActive` в домене; репозиторий читает и пишет колонку, фильтр `is_active = 1` убрать из `GetByID`/`GetByEmail`/`GetAll` (`user_repository_sqlite.go:144,188,234`) — активность проверяет `auth`, список показывает всех с полем `is_active`
- [x] удаление пользователя сейчас — soft delete `is_active = 0` (`Delete`, `:315`), то есть то же, что деактивация, а транзакции ссылаются на пользователя `ON DELETE RESTRICT`. Поэтому: `DELETE /api/v1/users/:id` и `UserService.DeleteUser` удалить, `tests/integration/api_users_delete_test.go` и веб-тесты удаления переписать на `PATCH`; `Delete` в репозитории удалить
- [x] `UserService.SetActive(id, active, actorID)`: нельзя себя, нельзя последнего активного админа; деактивация → `sessions.DeleteByUser`
- [x] `PATCH /api/v1/users/:id {is_active?, role?}`, `PUT /api/v1/users/:id/password` (admin)
- [x] тесты: деактивированный не логинится и его токен → 401, самодеактивация → 409, последний админ → 409 `LAST_ADMIN`, `GET /users` показывает неактивного с `is_active: false`
- [x] `openapi.yaml` обновлён (`DELETE /users/{id}` убран); `make fmt && make test && make lint` — зелёные

### Task 6: CLI `setup` и `reset-password`, транзакционный bootstrap

**Files:**
- Create: `cmd/server/setup.go`, `cmd/server/setup_test.go`, `internal/bootstrap.go`,
  `internal/bootstrap_test.go`
- Modify: `cmd/server/main.go`, `internal/run.go`, `internal/services/family_service.go`,
  `family_service_test.go`, `internal/infrastructure/user/family_repository_sqlite.go`,
  `migrations/001_consolidated.up.sql`

- [ ] `internal.OpenDatabase(cfg) (*sql.DB, error)` — открытие + миграции; `NewApplication` использует её
- [ ] `families.singleton INTEGER NOT NULL DEFAULT 1 CHECK (singleton = 1) UNIQUE` в миграции; `FamilyRepository.Bootstrap(ctx, family, categories, admin)` — одна транзакция `BEGIN IMMEDIATE`; `SetupFamily` вызывает её; ошибка UNIQUE → `ErrFamilyAlreadyExists`
- [ ] подкоманды в `main.go`: `setup` (флаги из «Technical Details», пароль из stdin, `ValidatePassword`), `reset-password`; без подкоманды — сервер, как сейчас
- [ ] тесты: `Bootstrap` при сбое на админе не оставляет семью; два `Bootstrap` подряд → второй `ErrFamilyAlreadyExists`; CLI — парсинг флагов и чтение пароля из stdin (через `io.Reader`)
- [ ] `make test` — зелёный

### Task 7: `testhelpers` на bearer, интеграционные тесты без cookie

**Files:**
- Modify: `internal/testhelpers/integration_server.go`, `tests/integration/helpers_test.go`,
  `tests/integration/api_auth_test.go`, `session_revalidation_test.go`, `test_server_test.go`,
  все `tests/integration/*_test.go` с `Apply`

- [ ] `AuthSession{Token}`, `Apply` ставит `Authorization: Bearer`; `LoginAs` выдаёт токен через `auth.Service` тестового сервера — сигнатура меняется с `LoginAs(t, u)` (`integration_server.go:162`) на `LoginAs(t, ts, u)`, вызовы `:222,:242` и в `tests/integration` обновить; CLAUDE.md уже ошибочно описывает трёхаргументную форму
- [ ] `api_auth_test.go`: проверки cookie/CSRF заменить на bearer (нет заголовка → 401, мусор → 401, отозванный → 401); `session_revalidation_test.go`: смена роли / деактивация видна на следующем запросе
- [ ] удалить `TestSessionSecret`, `randomCSRFToken`, зависимость от `gorilla/sessions` в хелпере
- [ ] `make test` — зелёный при обоих путях аутентификации

### Task 8: Удаление веб-слоя и чистка конфига

**Files:**
- Delete: `internal/web/**`, `internal/application/handlers/api_auth.go` (+ тест),
  `.claude/skills/test-frontend/`, `tests/integration/{auth_csrf,error_page,invites,login_links,web_pages,web_render,web_transactions_pagination}_test.go`
- Modify: `internal/application/http_server.go`, `http_server_test.go`, `http_server_internal_test.go`,
  `internal/application/handlers/transactions.go`, `transactions_test.go`, `users.go`, `users_test.go`,
  `internal/run.go`, `internal/config.go`, `config_test.go`, `internal/services/dto/mappers.go`,
  `mappers_test.go`, `internal/testhelpers/integration_server.go`, `docker/Dockerfile`,
  `docker/docker-compose.yml`, `Makefile`, `.golangci.yml`, `go.mod`
  (`deploy/*.yml` и `deploy/.env.production.example` не трогать — их целиком заменяет план 05;
  до него они требуют `SESSION_SECRET`/`CSRF_SECRET`, которые приложение уже игнорирует)

- [ ] до удаления: `transactions.go:53` и `users.go:250` с `middleware.GetUserFromContext` перевести на `auth.FromContext`; полный список импортёров вне `internal/web` — `grep -rl 'family-budget-service/internal/web' --include='*.go' . | grep -v '^./internal/web/'` (9 файлов) — должен опустеть
- [ ] `http_server.go`: убрать `web.NewWebServer`, `TemplatesDir`/`StaticDir`, `WebServerInitError`; `HTTPErrorHandler` — JSON в едином envelope для любых ошибок (в том числе 404 неизвестного пути); `CORS()` убрать
- [ ] `run.go`: убрать проверку веб-инициализации и `CookieSecure`; `config.go`: удалить `Web.*` кроме нового `TrustedProxies`/`BackupDir`, упростить `Validate()`; `config_test.go` обновить
- [ ] `dto/mappers.go`: удалить функции форм (`FromCreateUserForm`, `FromSetupForm`) и импорт `internal/web/models`
- [ ] удалить файлы из списка; `go mod tidy` (уходят `echo-contrib`, `gorilla/sessions`); `.golangci.yml:456-473` — исключения для `internal/web` убрать
- [ ] `Dockerfile`: убрать `COPY` шаблонов и статики, комментарий про `--spider`/`/login`; `docker/docker-compose.yml`: убрать `SESSION_SECRET`, `CSRF_SECRET`, `COOKIE_SECURE`, добавить `TRUSTED_PROXIES`; `Makefile compose-config` (`:140-158`): убрать цикл «стартует без секретов» и `COMPOSE_VALIDATE_ENV` для секретов — список `DEPLOY_COMPOSE_FILES` сокращает план 05
- [ ] `http_server_test.go`: тесты регистрации роутов — только API; неизвестный путь → 404 JSON
- [ ] `make fmt && make test && make lint` — зелёные; `docker build` собирается, `/health` внутри контейнера 200

### Task 9: Verify acceptance criteria

- [ ] `grep -r "internal/web" --include='*.go'` пуст; `grep -r SESSION_SECRET` вне `docs/plans/completed` пуст
- [ ] ручной прогон на `make run-local`: `setup` через CLI → `curl` login → `GET /me` → `PUT /me/password` → старый токен 401 → 11 неверных логинов → 429
- [ ] `make pre-commit` зелёный; покрытие не ниже, чем до плана (веб-тесты удалены, значит смотреть на `internal/auth` и `handlers` отдельно)

### Task 10: [Final] Update documentation

- [ ] `CLAUDE.md`: переписать «Architecture» (нет web, есть `internal/auth`, CLI), «Two HTTP surfaces» → одна, удалить «Templates», «Frontend rules», `/test-frontend`; «Known rough edges» — убрать S-03 и секреты; «Testing» — `LoginAs` выдаёт bearer
- [ ] `README.md`: убрать разделы про веб-интерфейс, admin panel, HTMX/PicoCSS, cookie-конфиг; Quick Start — CLI `setup` и `curl`-пример логина
- [ ] `docs/product_brief.md`, `docs/tech_stack.md`: раздел Frontend удалить, «Безопасность» — bearer + лимитер; `docs/guides/testing_strategy.md` — абзац про сессии и CSRF
- [ ] `docs/backlog.md`: закрыть S-03, `CSRF_SECRET`, `echo-contrib`, долг `application → web`; `docs/specs/README.md` — отметить S-03 закрытой, 003 — «устарел, UI удалён»; `docs/specs/003-ui-ux-audit.md` удалить
- [ ] `.claude/skills/README.md`: убрать `/test-frontend`
- [ ] переместить план в `docs/plans/completed/`

## Post-Completion

**Manual verification:** на телефоне через любой REST-клиент (или `curl` с мобильного) — логин
по `https://`, токен в заголовке, `GET /transactions`.

**Owner decision (spec 005, открытые вопросы):** если выбрана ротация refresh-токенов —
добавить задачу между 2 и 3: колонка `sessions.family_id`, `POST /auth/refresh`, отзыв всей
семьи при повторном использовании.
