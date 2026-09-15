# План 14 — Метрики Prometheus на служебном порту

Заказчик — репозиторий observability (сессия `observability-d2`, 15.09.2026): Alloy на mini снимает
контейнеры по docker-меткам и ходит по IP контейнера, порт наружу не публикуется. Серверный релиз
`v0.5.0`; контракт `/api/v1` не меняется, клиент не трогается.
Ревью: codex (тред `prometheus-metrics`) и plan-review, 15.09.2026 — см. «Решения после ревью».

## Overview

- Второй слушатель `METRICS_ADDR` (пусто — выключен; в compose `0.0.0.0:9091`) с одним маршрутом
  `GET /metrics`. Не Echo и не за Caddy: `/api/v1` и `/health` остаются на 8080, docker-healthcheck
  не меняется, тест покрытия спеки перебирает `Echo.Routes()` и слушателя не видит
  (`tests/integration/openapi_coverage_test.go:108`).
- Пакет `internal/metrics`: свой `prometheus.Registry` без глобалов (`gochecknoglobals`), экземпляр едет
  параметром из `internal/run.go`; образец — `digital-property-passport/internal/metrics`.
- Метрики `ffs_*` по запросу observability: HTTP, вход, бэкапы, база, состояние. Итоговый список имён и
  меток — в «Technical Details», заказчик его принял.
- В `deploy/docker-compose.yml` у `app`: `METRICS_ADDR`, метки `prometheus.io/scrape` и `prometheus.io/port`.

## Context (from discovery)

- Сборка приложения — `internal/run.go:38` (`NewApplication`), запуск и остановка — `:131` (`Run`), `:162`
  (`shutdown`). Сейчас ошибка `Start` только отменяет контекст (`:147–149`), а `Run` возвращает `shutdown()`,
  который всегда `nil` (`:198`); `HTTPServer.Start` создаёт и присваивает `echo.Server` внутри себя
  (`http_server.go:227`), контекст игнорирует. Версия — `internal/version`.
- `run.go` держит репозитории только как интерфейсы: `infrastructure.NewRepositoriesSQLite(db)` →
  `*handlers.Repositories` (`handlers/repositories.go:8–14`, алиасы `services.*` и `auth.SessionRepository`).
  Метод, добавленный конкретному SQLite-репозиторию, из `run.go` не виден без расширения интерфейса и всех
  его моков (`services/helpers_test.go:65`, `handlers/users_test.go:26`, `auth/service_test.go:25`).
  `*sql.DB` в `run.go` доступен.
- Echo-middleware: `http_server.go:97–124` (`Recover`, `RequestID`, `ContextTimeout`, логирование);
  маршруты `:154`, `AuthHandler` строится в `:134`. Группа с middleware регистрирует catch-all
  `RouteNotFound("")` и `("/*")` (`echo/v4@v4.15.4/group.go:31`), поэтому у неизвестного пути под `/api/v1`
  `c.Path()` = `/api/v1/*`. **Вне группы** несматченный путь получает шаблон ближайшего узла дерева
  (`echo/router.go:741–743`: `currentNode.originalPath`) — `GET /healthz` отчитается как `/health`.
  Тест спеки пропускает маршруты с методом `echo.RouteNotFound` (`openapi_coverage_test.go:109`).
- Статус ответа после `next` не окончателен: ошибку middleware (`401`/`403` от `RequireBearer`) пишет
  `HTTPErrorHandler` уже после всей цепочки — `observability/middleware.go:33–41` берёт код из
  `*echo.HTTPError`, остальные ошибки считает `500`. Хендлеры ошибку после записанного ответа не
  возвращают (`respondError` — обёртка над `c.JSON`). `Recover` без `LogErrorFunc` логирует через
  `c.Logger()`; с ним — только то, что сделает callback, и он обязан вернуть ошибку дальше
  (`middleware/recover.go:103,123`).
- Вход: `handlers/auth.go:47` (`Login`): `Bind`/`Validate` (`:50,:53`), `rate_limited` (`:61`),
  `setup_required` (`:70`), `invalid_credentials` (`:73`), ошибка (`:76`), успех (`:84`).
  `NewAuthHandler` зовут `http_server.go:134` и `handlers/auth_test.go:135`.
- Бэкапы: `services/backup_service.go:184` (`CreateBackup`; temp-файл публикуется `os.Link`+`os.Remove`
  только после успешного `VACUUM`, `:155–161`; `os.Stat` после — `:233`; retention-ошибка не отменяет
  успех, `:253`), `ListBackups` `:263` (только `backup_*.db`, `sweepStaleTemps` `:360`).
  `NewBackupService` зовут `run.go:79`, `cmd/server/backup.go:48`, `testhelpers/integration_server.go:81`,
  `backup_service_security_test.go` (3) и `backup_service_test.go` (16). API-вызов идёт в процессе сервера;
  **cron — отдельный процесс** (`docker compose run --rm --no-deps -T app backup`, `deploy/README.md:92`;
  `cmd/server/backup.go:49`), реестра у него нет.
- База: `infrastructure/sqlite.go:46` (`MaxOpenConns=1`, `_txlock=immediate`), WAL. Скрейп ждёт единственное
  соединение, в том числе за `VACUUM INTO` API-бэкапа.
- Сессии: `auth/session.go:48` (`ExpiryAfter` — минимум скользящего и абсолютного срока уже в `expires_at`),
  `auth/service.go:119` (`Authenticate` сравнивает только `ExpiresAt`), деактивация удаляет сессии в той
  же транзакции (`user_repository_sqlite.go:307`) — `expires_at > now` и есть «живая»; время хранится
  RFC3339 UTC текстом и сравнивается лексикографически (`session_repository_sqlite.go:194,220`).
- Транзакции: `BulkDelete` сервиса (`transaction_service.go:401`) → `DeleteBulk` репозитория
  (`transaction_repository_sqlite.go:602`) возвращает только `RowsAffected`, типов удалённых строк нет.
- Compose: `deploy/docker-compose.yml` сервис `app` — `environment` перекрывает `env_file`, значение из
  `.env` мёртвое (комментарий у `TRUSTED_PROXIES`); оверлей `docker-compose.proxied.yml` переопределяет
  только `environment.TRUSTED_PROXIES` и `networks`, карты сливаются. `docker/Dockerfile:72` `EXPOSE 8080`.
- Тесты: `testhelpers/integration_server.go:58` (`SetupHTTPServer` → `application.NewHTTPServer:105`),
  `NewHTTPServer` в `http_server_test.go` — 18 вызовов; `internal/observability/middleware_test.go` —
  образец теста Echo-middleware.
- Линтер: `depguard` только запрещает (`log` — нельзя вне `main`, `.golangci.yml:212–218`), prometheus
  допустим; `gosec` требует `ReadHeaderTimeout`; `gochecknoglobals` — никакого `promauto` на пакетном
  уровне; `testpackage` для `internal/metrics` не исключён — тесты только `package metrics_test`;
  `mnd` уже знает `prometheus.ExponentialBuckets`/`LinearBuckets` (`:339–341`).

## Development Approach

- **testing approach**: Regular (код, затем тесты в той же задаче)
- каждая задача завершается зелёными `make fmt`, `make test`, `make lint` (0 issues)
- в план вписываются ➕ найденные задачи и ⚠️ блокеры; чекбоксы закрываются сразу

## Solution Overview

**Слушатель.** `METRICS_ADDR` — `ServerConfig.MetricsAddr`; пусто — слушателя нет (локальный запуск и
тесты ничего не открывают). `metrics.NewServer(addr, handler)` — `*http.Server` с `ReadHeaderTimeout`,
`ServeMux` с одним `/metrics`. Отдельный `net/http`, а не второй Echo: ни middleware, ни envelope, ни
спека ему не нужны.

**Запуск и остановка.** Оба `*http.Server` создаются в `NewApplication`, а не внутри `Start` (сейчас
`HTTPServer.Start` собирает `echo.Server` сам — это переезжает в конструктор, `Start` только слушает).
Запуск и ожидание выносятся в `internal.serveAll(ctx, servers...) error` — горутина на сервер,
буферизованный канал ошибок, первая ошибка старта или отмена контекста → остановка всех и возврат этой
ошибки (сейчас теряется). `Run` — сигналы + `serveAll` + `shutdown()`. `shutdown()` гасит оба слушателя
параллельно под одним таймаутом, ждёт обоих, потом закрывает БД: коллекторы ходят в БД, поэтому
`db.Close()` до остановки `/metrics` — гонка. Порядок Echo-middleware тем самым не меняется.

**Реестр и границы пакетов.** `metrics.New(version, logger)` — `prometheus.NewRegistry()` +
Go/process-коллекторы + `ffs_build_info`; инструменты через `promauto.With(reg)` внутри `New`.
`internal/metrics` — лист: импортирует только `prometheus`, `echo` и `internal/version`, ничего из
`services`/`handlers`/`infrastructure`. Интерфейсы наблюдателей объявляет потребитель (как `auth` объявляет
свои репозитории): `services.BackupObserver` и `handlers.LoginObserver` — по одному методу; `metrics`
их реализует, а у потребителя рядом лежит `NopBackupObserver`/`NopLoginObserver` для cron-процесса,
`NewHTTPServer` без метрик и тестов. Коллекторы состояния получают свои узкие интерфейсы из `metrics`
(`BackupLister`, `StateReader`), которые реализуют замыкание в `run.go` и конкретный тип
`infrastructure.NewMetricsReader(db)` — так `metrics` не импортирует `services` ради `BackupInfo`, а
репозиторные интерфейсы и их моки не растут. `Handler()` — `promhttp.HandlerFor(reg,
{ErrorHandling: ContinueOnError, ErrorLog: адаптер slog → promhttp.Logger})`: без этого одна ошибка
коллектора превращает весь `/metrics` в `500` (`promhttp/http.go:481`); `promhttp.Logger` — это
`Println(...any)`, а `log` запрещён `depguard`.

**HTTP.** `metrics.EchoMiddleware(observer)` ставится первым (до `Recover`); in-flight уменьшается в
`defer`. Статус — общий помощник `observability.ResponseStatus(c, err)`, вынесенный из
`LoggingMiddleware` и переиспользованный обоими: из `*echo.HTTPError`, `500` для иной ошибки, иначе
`c.Response().Status`. `route` — `c.Path()`; чтобы вне `/api/v1` несматченный путь не приписывался
соседу (`/healthz` → `/health`), на корне Echo регистрируется `e.RouteNotFound("/*", echo.NotFoundHandler)`
— тогда любой чужой путь получает шаблон `/*`, а тест спеки такие маршруты пропускает. Итого
кардинальность `route` — 38 шаблонов + `/*`, `/api/v1/*`, `/api/v1/users/*`. `method` — из списка
стандартных, остальное → `OTHER` (catch-all матчит любой метод). Паники —
`RecoverWithConfig{LogErrorFunc}`: `Inc`, лог стека в slog (стандартный лог `Recover` при заданном
callback отключается) и `return err` — вернуть `nil` значит подавить `500`.

**Бэкапы.** Счётчик и гистограмма пишутся внутри `CreateBackup`, то есть только для API-вызовов — cron
живёт в другом процессе и получает `NopBackupObserver`. «Свежесть последней копии» и «число файлов»
считает коллектор на скрейпе из каталога (`ListBackups`): он видит и cron. Метрика зовётся
`ffs_backup_latest_file_timestamp_seconds` — это `ModTime` новейшего опубликованного файла, а не
«последний успешный запуск»: удаление новейшего файла сдвинет её назад. Метка `trigger` не вводится —
единственное значение `api` вводило бы в заблуждение. Заказчику этого достаточно.

**Скрейп-коллекторы.** `ffs_db_size_bytes` — `os.Stat` файла БД и `-wal` (метка `file`); пул — стандартный
`collectors.NewDBStatsCollector(db, "budget")` (`go_sql_*`, `db_name`), свои дубликаты не заводятся.
`ffs_sessions_active`, `ffs_users{active}`, `ffs_transactions`, `ffs_setup_complete` — четыре `COUNT`
в `infrastructure.MetricsReader` (один тип, без изменений репозиторных интерфейсов). Каждый `Collect`
работает под своим контекстом с таймаутом (`ScrapeTimeout` = 5 с, экспортирован ради `metrics_test`):
HTTP-таймаут `promhttp` SQL внутри `Collect` не прерывает. Ошибка или таймаут → `NewInvalidMetric` для
этого коллектора, остальные метрики скрейпа целы (`ContinueOnError`), текст — в лог. Задержка за
`VACUUM INTO` принимается: скрейп раз в 15–60 с, `MaxOpenConns=1` не трогаем.

## Решения после ревью

Раунд 1 (codex, `disagree`): приняты — `ContinueOnError`, статус из ошибки, `OTHER` для метода, `defer`
для in-flight, `LogErrorFunc` возвращает ошибку и логирует стек, серверы создаются до горутин, `Run`
возвращает ошибку старта, параллельная остановка до `db.Close()`, контекст с таймаутом в `Collect`,
`invalid_request` во входе, явный ноль для обоих `active`, переименование метрики бэкапа.
`ffs_transactions_total{op,type}` снят: `BulkDelete` не знает типов удалённых строк, а считать по
запрошенным id неточно; вместо него `ffs_transactions` gauge на скрейпе. Оставлены как есть: отдельный
слушатель, `DBStatsCollector`, файловая метрика бэкапа, `ffs_setup_complete` вместо `ffs_families`.
Второй раунд codex не состоялся (лимит), его два вопроса проверены по коду: хендлеры не возвращают
ошибку после записанного ответа; тесты не зависят от `HTTPServer.Start`.

Ответ observability-d2 (15.09.2026): имена приняты, дашборд подогнан; пул берут из `go_sql_*`;
Alloy отбрасывает `com.docker.compose.oneoff=True` — метка у cron не нужна; статус cron-запуска не
нужен — алерт «копия старше 26 ч» строится на `ffs_backup_latest_file_timestamp_seconds`.

plan-review (15.09.2026, «needs revision»): интерфейсы наблюдателей переехали к потребителю + Nop-реализации
(иначе цикл `services` ↔ `metrics` через `BackupInfo` и паника на nil в cron); все вызовы изменённых
конструкторов внесены в списки файлов; счётчики состояния читает отдельный `infrastructure.MetricsReader`
вместо расширения репозиторных интерфейсов и четырёх моков; `/metrics` на Echo — `404` и без токена
(`RequireBearer` висит только на группе), а не `401`; корневой `RouteNotFound("/*")` вместо «`c.Path()`
как есть»; тест `Run` заменён тестом `serveAll`; адаптер `ErrorLog` и общий `ResponseStatus` — отдельными
пунктами; `bad_credentials` → `invalid_credentials`, как в коде (заказчик уведомлён). Предложенные
сокращения (`ffs_users`, `ffs_transactions`, `ffs_setup_complete`, in-flight) не приняты: заказчик уже
построил дашборд на этом списке, а цена — один тип-читатель без моков.

## Technical Details

Итоговый список (принят заказчиком; `compose_project`/`compose_service` добавляет Alloy):

| Метрика | Тип | Метки | Источник |
|---|---|---|---|
| `ffs_build_info` | gauge = 1 | `version`, `go` | `version.String()`, `runtime.Version()` |
| `ffs_http_requests_total` | counter | `method`, `route`, `status` | middleware |
| `ffs_http_request_duration_seconds` | histogram, `DefBuckets` | `method`, `route` | middleware |
| `ffs_http_requests_in_flight` | gauge | — | middleware |
| `ffs_http_panics_total` | counter | — | `Recover.LogErrorFunc` |
| `ffs_login_attempts_total` | counter | `outcome` ∈ `ok`, `invalid_credentials`, `rate_limited`, `setup_required`, `invalid_request`, `error` | `AuthHandler.Login` |
| `ffs_sessions_active` | gauge на скрейпе | — | `COUNT(*) FROM sessions WHERE expires_at > ?` (`formatTime(now)`) |
| `ffs_backups_total` | counter | `outcome` ∈ `ok`, `error` | `CreateBackup` (только API) |
| `ffs_backup_duration_seconds` | histogram, `0.1…120` | — | `CreateBackup` (только API) |
| `ffs_backup_latest_file_timestamp_seconds` | gauge на скрейпе | — | `ModTime` новейшего `backup_*.db`; 0 — файлов нет |
| `ffs_backup_files` | gauge на скрейпе | — | число `backup_*.db` в `BACKUP_DIR` |
| `ffs_db_size_bytes` | gauge на скрейпе | `file` ∈ `main`, `wal` | `os.Stat`; отсутствующий `-wal` → 0 |
| `go_sql_*` | стандартный | `db_name="budget"` | `collectors.NewDBStatsCollector` |
| `ffs_users` | gauge на скрейпе | `active` ∈ `true`, `false` | `COUNT(*) … GROUP BY is_active`, оба значения всегда |
| `ffs_transactions` | gauge на скрейпе | — | `COUNT(*) FROM transactions` |
| `ffs_setup_complete` | gauge на скрейпе, 0/1 | — | `COUNT(*) FROM families` |
| `go_*`, `process_*` | стандартные | — | коллекторы client_golang |

`route` — шаблон Echo: `/api/v1/transactions/:id`, `/health`, `/api/v1/*` (catch-all группы), `/*` (корень).
Оценка рядов: ~38 операций × 14 рядов классической гистограммы ≈ 530 плюс счётчик по статусам.

Не вошло: `ffs_families` (в модели одна семья — заменено на `ffs_setup_complete`), `ffs_db_*` по
`sql.DBStats` (`go_sql_*`), `trigger` у бэкапов, `ffs_transactions_total` (см. «Решения после ревью»).

Конфигурация: `METRICS_ADDR` (`host:port`, пусто — выключено). Compose: `METRICS_ADDR: 0.0.0.0:9091` в
`environment` (задано топологией, как `SERVER_HOST`; из `.env` не переопределяется — `environment`
перекрывает `env_file`), `labels: prometheus.io/scrape: "true"`, `prometheus.io/port: "9091"`; `ports` не
публикуются; Dockerfile получает `EXPOSE 9091` как документацию. `.env.example` не меняется. Cron-контейнер
`compose run` наследует метки, но Alloy отбрасывает `com.docker.compose.oneoff=True` — cron-команда не
меняется.

Зависимость: `github.com/prometheus/client_golang` (v1.24.x, как в dpp); `testutil` из него — в тестах.

## Implementation Steps

### Task 1: Пакет `internal/metrics` — реестр, наблюдатели, слушатель

**Files:**
- Create: `internal/metrics/metrics.go`, `internal/metrics/observers.go`, `internal/metrics/server.go`
- Create: `internal/metrics/metrics_test.go`, `internal/metrics/server_test.go` (`package metrics_test`)
- Modify: `go.mod`, `go.sum`

- [x] `go get github.com/prometheus/client_golang`; `metrics.New(version, logger)`: реестр,
      Go/process-коллекторы, `ffs_build_info`, `Gatherer()`, `Register(collector)`
- [x] адаптер `slog` → `promhttp.Logger` (`Println`); `Handler()` с `ContinueOnError` и этим `ErrorLog`
- [x] реализации наблюдателей на `promauto.With(reg)`: `HTTP()` (для middleware), `Login()` — метод
      `ObserveLogin(outcome string)`, `Backup()` — `ObserveBackup(outcome string, d time.Duration)`
      (имена и метки — из таблицы; интерфейсы под них объявят потребители в Task 3)
- [x] `metrics.NewServer(addr, handler)` → `*http.Server` с `ReadHeaderTimeout`, mux только с `/metrics`
- [x] тесты: `build_info` с версией, `Handler()` отдаёт text-format и `200` при коллекторе, отдающем
      `NewInvalidMetric`; счётчики/гистограммы через `testutil.ToFloat64`/`CollectAndCount`; два `New()`
      не конфликтуют; `NewServer` на занятом порту возвращает ошибку `ListenAndServe`
- [x] `make fmt && make test && make lint`

### Task 2: Echo-middleware, паники, корневой catch-all

**Files:**
- Create: `internal/metrics/echo.go`, `internal/metrics/echo_test.go`
- Modify: `internal/observability/middleware.go`, `internal/observability/middleware_test.go`
- Modify: `internal/application/http_server.go`, `internal/application/http_server_test.go`

- [x] `observability.ResponseStatus(c, err) int` вынести из `LoggingMiddleware` (`:33–41`), тест на три ветки
- [x] `metrics.EchoMiddleware(HTTPObserver)`: in-flight с `defer`, длительность, `requests_total` по
      `c.Path()` и `ResponseStatus`; метод из списка, иначе `OTHER`; ошибка возвращается дальше
- [x] `application.Config.Metrics *metrics.Metrics` (nil — без middleware); middleware первым;
      `Recover` → `RecoverWithConfig{LogErrorFunc}`: `panics.Inc()`, slog со стеком, `return err`;
      `e.RouteNotFound("/*", echo.NotFoundHandler)` на корне
- [x] тесты middleware: шаблон в `route`, `/api/v1/*` на неизвестный путь в группе, `/*` на `/healthz`,
      `401` от middleware-ошибки попадает в `status="401"`, паника → счётчик и `500` JSON-envelope,
      in-flight возвращается к 0 после паники; `TestOpenAPISpec_*` зелёные
- [x] `make fmt && make test && make lint`
- ➕ корневой `RouteNotFound("/*")` перехватывает и чужой метод: `POST /health` — `404 NOT_FOUND`
      вместо `405`, как это уже было внутри `/api/v1` из-за catch-all группы. Ожидания поправлены в
      `http_server_test.go` и `tests/integration/api_error_envelope_test.go`; спека 405 не описывала.

### Task 3: Вход и бэкапы

**Files:**
- Modify: `internal/application/handlers/auth.go`, `internal/application/handlers/auth_test.go`,
  `internal/application/http_server.go`
- Modify: `internal/services/backup_service.go`, `internal/services/backup_service_test.go`,
  `internal/services/backup_service_security_test.go`, `internal/run.go`, `cmd/server/backup.go`,
  `internal/testhelpers/integration_server.go`
- Create: `internal/metrics/backup_collector.go`, `internal/metrics/backup_collector_test.go`

- [x] `handlers.LoginObserver` (`ObserveLogin(outcome)`) + `handlers.NopLoginObserver`;
      `NewAuthHandler(authService, limiter, logger, observer)`; по одному `ObserveLogin` на каждой ветке
      `Login`, включая `invalid_request` на `Bind`/`Validate`; `http_server.go:134` передаёт
      `Config.Metrics.Login()` или `NopLoginObserver{}`
- [x] `services.BackupObserver` (`ObserveBackup(outcome, d)`) + `services.NopBackupObserver`;
      `NewBackupService(db, dbPath, backupDir, keep, logger, observer)`; `outcome` и длительность вокруг
      всего `CreateBackup` (ошибка каталога и `os.Stat` — тоже `error`; retention-ошибка успех не
      отменяет); `cmd/server/backup.go`, `testhelpers` и тесты передают `NopBackupObserver{}`, `run.go` —
      пока тоже (реестр появится в Task 5)
- [x] `metrics.BackupLister` (`ListBackupTimes(ctx) ([]time.Time, error)`) и
      `metrics.NewBackupDirCollector(lister)` — `ffs_backup_latest_file_timestamp_seconds`,
      `ffs_backup_files`; ошибка → `NewInvalidMetric`
- [x] тесты: исходы входа (успех, неверный пароль, rate limit, setup required, битый JSON) через
      счётчик из `metrics.New`; исходы бэкапа; коллектор на пустом и непустом списке и на ошибке
- [x] `make fmt && make test && make lint`

### Task 4: Состояние на скрейпе — база, сессии, пользователи, транзакции

**Files:**
- Create: `internal/infrastructure/metrics_reader_sqlite.go`, `internal/infrastructure/metrics_reader_sqlite_test.go`
- Create: `internal/metrics/state_collector.go`, `internal/metrics/state_collector_test.go`

- [x] `infrastructure.NewMetricsReader(db)`: `ActiveSessions(ctx, now)`, `UsersByActive(ctx) (active,
      inactive int, err)`, `Transactions(ctx)`, `SetupComplete(ctx)`; тесты на in-memory БД
      (`testhelpers.SetupSQLiteTestDB`, фабрики): просроченная сессия не считается, оба `active` при
      пустой таблице — нули
- [x] `metrics.StateReader` (те же четыре метода) и `metrics.NewStateCollector(reader, dbPath)`:
      `ffs_db_size_bytes{file}`, `ffs_sessions_active`, `ffs_users{active}`, `ffs_transactions`,
      `ffs_setup_complete`; каждый `Collect` под `context.WithTimeout(ScrapeTimeout)`;
      `metrics.New` регистрирует `collectors.NewDBStatsCollector(db, "budget")` через `RegisterDB(db)`
- [x] тесты коллектора с фейковым `StateReader`: `-wal` отсутствует → 0, ошибка ридера → `/metrics`
      всё равно `200` без этой метрики, ридер, ждущий `ctx.Done()`, → то же за `ScrapeTimeout`
- [x] `make fmt && make test && make lint`

### Task 5: Конфигурация, сборка и остановка в `run.go`, интеграционный тест

**Files:**
- Modify: `internal/config.go`, `internal/config_test.go`, `internal/run.go`,
  `internal/application/http_server.go`, `internal/testhelpers/integration_server.go`
- Create: `internal/run_test.go` (только `serveAll`), `tests/integration/metrics_test.go`

- [x] `METRICS_ADDR` → `ServerConfig.MetricsAddr`; `Validate` принимает `host:port` через `net.SplitHostPort`
- [x] `HTTPServer`: `*http.Server` собирается в конструкторе, `Start` только слушает; `serveAll(ctx,
      servers...)` — горутина на сервер, канал ошибок, первая ошибка → остановка остальных и возврат
- [x] `run.go`: `metrics.New` до сервисов, `Login()`/`Backup()` в хендлер и сервис, `RegisterDB`,
      `BackupDirCollector` (замыкание над `ListBackups`) и `StateCollector` после БД; оба сервера созданы
      до `Run`; `Run` = сигналы + `serveAll` + `shutdown()`, возвращает ошибку старта
- [x] `shutdown()`: оба `Shutdown` параллельно под одним контекстом, ждать обоих, потом `db.Close()`
- [x] `TestServer.Metrics` в `testhelpers` — тот же экземпляр, что в сервере
- [x] интеграционный тест: после `POST /auth/login` (успех + неверный пароль) и `GET /api/v1/transactions`
      `Handler()` отдаёт `ffs_login_attempts_total{outcome="ok"} 1`, строку `route="/api/v1/transactions"`,
      `ffs_build_info`; `GET /metrics` на Echo — `404` и без токена, и с ним
- [x] тест `serveAll`: два сервера на `127.0.0.1:0`, отмена контекста → оба остановлены; занятый порт у
      второго → ошибка возвращается, первый остановлен
- [x] `make fmt && make test && make lint`

### Task 6: Deploy и документация выката

**Files:**
- Modify: `deploy/docker-compose.yml`, `docker/Dockerfile`, `deploy/README.md`, `deploy/CLAUDE.md`

- [x] `app.environment.METRICS_ADDR: 0.0.0.0:9091` с комментарием, что это топология и из `.env` не
      меняется; `labels` `prometheus.io/scrape`, `prometheus.io/port`; `EXPOSE 9091`; dev-compose без
      изменений (выключено)
- [x] `make compose-config` (все три раскладки) зелёный
- [x] `deploy/README.md`: раздел «Monitoring» — что слушает, как проверить с хоста
      (`docker exec family-budget-app wget -qO- http://127.0.0.1:9091/metrics | head`), что cron-бэкап
      виден только через `ffs_backup_latest_file_timestamp_seconds`; `deploy/CLAUDE.md` — одна строка

### Task 7: Verify acceptance criteria
- [x] все метрики из таблицы присутствуют в выводе `/metrics` локального запуска с `METRICS_ADDR=127.0.0.1:9091`
      (`run-local`, один login, один бэкап через API, один запрос без токена, один `GET /healthz`)
- [x] с пустым `METRICS_ADDR` порт не слушается, `make test` не открывает сокетов кроме `127.0.0.1:0`
- [x] `make fmt && make test && make lint` — 0 issues; `make compose-config`

### Task 8: [Final] Update documentation
- [x] `CLAUDE.md`: шаг сборки в `NewApplication`, `METRICS_ADDR`, где живут метрики и интерфейсы
      наблюдателей, корневой `RouteNotFound`, что `Run` возвращает ошибку старта; `docs/tech_stack.md`
- [x] `docs/backlog.md`: снять запись о мониторинге, `docs/specs/005-api-only-redesign.md`: строка плана 14
      — записи о мониторинге в бэклоге не было, вместо неё раздел «Метрики закрыты планом 14» с тем,
      что сознательно не сделано
- [x] переместить план в `docs/plans/completed/`

## Post-Completion

- Релиз `v0.5.0`, выкат на mini по `deploy/README.md`; проверить с хоста
  `curl http://$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' family-budget-app):9091/metrics`.
- Сообщить сессии `observability-d2` о выкате (`ListAgents` → `SendMessage`): она проверит
  `up{job="apps", compose_project="ffs"}` и панели; алерты — на стороне observability.
