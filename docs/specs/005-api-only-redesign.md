# 005 — Переход на API-only бэкенд для Android-приложения

> Дата: 4 сентября 2026, коммит `2e42580`. Аудит кода плюс второе мнение codex
> (тред `api-only-auth`). Это документ решений: планы в `docs/plans/` ссылаются
> на него по номерам `A-xx`.

## Цель и рамки

- Сервис становится JSON API без HTML-интерфейса. Единственный клиент — Android-приложение,
  которое пишется следом.
- Один инстанс = одна семья. Пользователей двое (владелец и супруга), обе роли известны заранее.
- Развёртывание: домашний мини-сервер, домен `ffs.shatrov.tech`, Caddy + Let's Encrypt.
- Сервис ещё не используется: продовых данных нет, схему БД и любые слои можно переписать с нуля.

## Что мешает мобильному клиенту сейчас

Факты проверены по коду; ссылки — на текущее состояние `main`.

### Аутентификация

| # | Факт | Где |
|---|---|---|
| 1 | JSON-логина нет. Единственный вход — HTML-форма `POST /login`; клиенту пришлось бы парсить страницу ради CSRF-токена и хранить cookie jar | `internal/web/handlers/auth.go:106` |
| 2 | CSRF-middleware и `RequireSetup` зарегистрированы глобально и действуют на `/api/v1`: анонимная запись → 403, до setup любой API-вызов → 302 на `/setup` | `internal/web/web.go:65,98` |
| 3 | Cookie-сессия живёт ровно 24 ч (константа), отозвать её с сервера нельзя; `SESSION_TIMEOUT`, `CSRF_SECRET`, `COOKIE_HTTP_ONLY`, `COOKIE_SAME_SITE` читаются, но ни на что не влияют | `internal/web/middleware/session.go:21`, `internal/config.go:99-104` |
| 4 | Таблица `user_sessions` есть в миграции, но кодом не используется; хранит сырой токен | `migrations/001_consolidated.up.sql:153` |
| 5 | bcrypt cost 10, политика пароля `min=6`, смены и сброса пароля нет нигде | `internal/services/user_service.go:88`, `internal/services/dto/user_dto.go:16` |
| 6 | Неизвестный email отвечает раньше bcrypt — тайминг выдаёт существование пользователя | `internal/web/handlers/auth.go:124-135` |
| 7 | Rate limiting в приложении отсутствует (S-03). Внешняя защита не работает: fail2ban ловит `POST /login … 401`, а неуспешный логин отдаёт 422; директива `rate_limit` в Caddyfile — плагин, которого нет в стоковом `caddy:2-alpine` | `deploy/fail2ban/family-budget.conf`, `deploy/caddy/Caddyfile.template:53` |
| 8 | `internal/application/handlers` импортирует `internal/web/middleware` — API-слой зависит от веб-слоя | `internal/application/handlers/api_auth.go:11` |

### Полнота API

Есть только в веб-интерфейсе, в `/api/v1` отсутствует:

| Функция | Где живёт логика | Судьба |
|---|---|---|
| Логин, логаут, setup, инвайты | `internal/web/handlers/auth.go`, `admin.go` | заменяется A-01…A-03 |
| Дашборд (сводка месяца, дельты, обзор бюджетов, инсайты по категориям) | `internal/web/handlers/dashboard.go` (1063 строки, сервиса нет) | переносится в `StatsService` |
| Генерация отчётов и CSV | `internal/web/handlers/reports.go:240,533`; `POST /api/v1/reports` отдаёт 501 | переносится в `ReportService` |
| Список пользователей | `UserService.GetUsers` есть, роут нет | добавляется |
| Бэкапы (создать, список, скачать, удалить, восстановить) | `BackupService` есть, только веб-роуты | API без restore |
| Массовое удаление транзакций | цикл в `internal/web/handlers/transactions.go:381` | добавляется |
| Алерты бюджетов | заглушки без персистентности (`budgets.go:766,801`) | удаляется вместе с таблицей `budget_alerts` |
| Плановые отчёты, прогноз, бенчмарки | заглушки в `ReportService` | удаляются из интерфейса |
| `FamilyHandler` | написан и протестирован, роут не зарегистрирован | регистрируется |

### Контракт

| Факт | Где |
|---|---|
| Деньги — `float64` в домене и JSON, `REAL` в SQLite | `transactions.amount`, `budgets.amount/spent` |
| Три формы ошибок: `error{code,message,details}`, `errors[{field,message,code}]` для валидации, inline-литералы `ResponseMeta` в `users.go`/`categories.go` | `internal/application/handlers/helpers.go:31,74` |
| Пагинация только у транзакций (`limit/offset`, без `total`); документация обещает `page/page_size` | `internal/application/handlers/transactions.go:345`, `docs/patterns/api_standards.md` |
| `Transaction.Date` — `time.Time` RFC3339 в JSON, `DATE` в SQLite; у семьи нет часового пояса | `internal/domain/transaction/transaction.go:10` |
| `POST` генерирует UUID на сервере — сетевой ретрай создаёт дубль | `internal/services/transaction_service.go:149` |
| Валюту семьи можно сменить при наличии операций — история молча «переименуется» | `internal/services/family_service.go:107` |
| `/health` отдаёт версию `1.0.0`, захардкоженную в `internal/run.go:51` | |

### Инварианты, которые код не держит

- «Ровно одна семья»: `families` без ограничения на число строк; `SetupFamily` создаёт семью,
  категории и админа тремя операциями без транзакции — после сбоя посередине повторный setup
  невозможен (`ErrFamilyAlreadyExists`), а семья остаётся без админа.
- «Всегда есть админ»: удаление последнего админа запрещено, понижение роли — нет
  (`internal/services/user_service.go:246`).
- `users.is_active` есть в БД (всегда `1`), в домене поля нет; деактивация = удаление.

## Решения

### A-01. Аутентификация: непрозрачные bearer-токены с хранением на сервере

- Таблица `sessions(id, user_id, token_hash UNIQUE, device_name, created_at, last_used_at, expires_at)`.
  Токен — 32 байта `crypto/rand`, base64url, отдаётся клиенту один раз; в БД только SHA-256.
- Время жизни: скользящее 30 дней без активности, абсолютный потолок 180 дней от выдачи.
  `last_used_at` обновляется не чаще раза в час — SQLite с одним соединением, запись на каждый
  запрос сериализовала бы чтения.
- Проверка — один `SELECT … FROM sessions JOIN users` по хешу: срок, `is_active`, роль читаются
  из БД на каждом запросе (как сейчас делает `RevalidateSessionUser`).
- Endpoints: `POST /api/v1/auth/login {email, password, device_name}` → `{token, user}`;
  `POST /auth/logout`; `GET /auth/sessions`; `DELETE /auth/sessions/:id`; `GET|PUT /me`;
  `PUT /me/password {current_password, new_password}` — отзывает все сессии, кроме текущей.
  Логин — в отдельной публичной группе; всё остальное под bearer-middleware.
- Секретов в конфиге не остаётся: `SESSION_SECRET`, `CSRF_SECRET`, `COOKIE_*`, `SESSION_TIMEOUT`
  удаляются. Пакет `internal/auth` заменяет `internal/web/middleware` и закрывает долг из
  `docs/backlog.md` (обратное ребро `application → web`).
- Пароль: bcrypt cost 12, длина 10…72 байта. Неизвестный email проходит сравнение с фиктивным
  хешем, ответ одинаковый (`401 INVALID_CREDENTIALS`).

**Расхождение с codex.** Codex рекомендует пару access + refresh с ротацией: повторное
использование старого refresh-токена выявляет клонирование (RFC 9700 §4.14.2). Единый токен
этого не даёт — украденный токен живёт до отзыва или потолка. Цена ротации: ещё один endpoint,
колонка `family_id` в `sessions`, `Authenticator` в OkHttp. Для двух пользователей выбран единый
токен: отзыв мгновенный, поверхность меньше, состояние на клиенте — одна строка. Переход на
ротацию не ломает контракт: `login` начнёт отдавать два поля, остальное не меняется.

### A-02. Bootstrap — только CLI на сервере

`family-budget-service setup --family "…" --timezone Europe/Moscow --currency RUB --email … --password-stdin`.
HTTP-endpoint `/setup` и `RequireSetup` удаляются: до setup `/health` отдаёт `setup_complete: false`,
логин — `409 SETUP_REQUIRED`. Нет гонки «кто первый вызовет setup на публичном домене» и нет
одноразового секрета в конфиге.

Setup выполняется одной транзакцией; единственность семьи держит схема:
`families.singleton INTEGER NOT NULL DEFAULT 1 CHECK (singleton = 1) UNIQUE`. Открытие БД и
миграции выносятся из `NewApplication` в общую функцию, которой пользуются и сервер, и CLI.

Тем же путём — `reset-password --email …` для забытого пароля админа.

### A-03. Второй пользователь — без инвайтов

Админ создаёт пользователя существующим `POST /api/v1/users` с паролем и сообщает его лично;
пользователь меняет пароль через `PUT /me/password`. Система инвайтов (`internal/domain/user/invite.go`,
`internal/services/invite_service.go`, репозиторий, таблица `invites`, ~1000 строк с тестами)
удаляется: минус неаутентифицированный endpoint регистрации, минус токен в URL.

### A-04. Роли и активность

- Роли `admin | member`; `child` удаляется из домена, CHECK-ограничения и DTO.
- `admin`: пользователи, бэкапы, семья. `member`: всё финансовое. Обе роли видят все данные семьи.
- `is_active` появляется в домене; `PATCH /users/:id {is_active}` — деактивация отзывает сессии.
- Инвариант «есть хотя бы один активный admin» проверяется при удалении, понижении и деактивации.
- Ограничение «не больше двух пользователей» в код не закладывается — это конфигурация, не инвариант.

### A-05. Деньги — целые в минимальных единицах

- Домен, БД (`INTEGER`) и JSON: `amount_minor` (int64, копейки). Имя поля с суффиксом — чтобы
  клиент не спутал с рублями. Валюта — у семьи, ISO 4217, только валюты с двумя знаками.
- Целыми остаются суммы, остатки, `spent`; проценты, утилизация, ставки — `float64` с явным
  приведением. Средние (`AVG` в SQL, `total / days`) округляются до минимальной единицы
  half-up; правило живёт в одном месте (`internal/domain/money`).
- Смена валюты семьи запрещена, если есть транзакции (`409 CURRENCY_LOCKED`).

### A-06. Даты и часовой пояс

- Дата операции — календарная, `YYYY-MM-DD` в JSON и `TEXT` в SQLite. Без времени.
- У семьи появляется `timezone` (IANA); «сегодня» и границы периодов считаются в нём.
- Служебные метки (`created_at`, `updated_at`, `expires_at`) — RFC3339 UTC.

### A-07. Идентификаторы и идемпотентность

`POST /transactions`, `/budgets`, `/categories` принимают необязательный `id` (UUID v4),
сгенерированный клиентом. Повтор с тем же `id` → `200` с существующей записью. Так ретрай
после обрыва сети не создаёт дубль.

### A-08. Пагинация и ошибки

- Все списки: `limit` (по умолчанию 50, максимум 200) и `offset`; ответ несёт
  `meta.pagination {limit, offset, total}`.
- Одна форма ошибки: `{"error": {"code", "message", "details": [{field, message, code}]}, "meta"}`.
  Валидация — `422 VALIDATION_ERROR`. Прочие коды — в `docs/patterns/api_standards.md`.
- `docs/api/openapi.yaml` — единственное описание контракта; тест проверяет, что каждый
  зарегистрированный роут `/api/v1/*` описан.

### A-09. Защита от перебора — в приложении

Лимитер на `POST /auth/login`: per-IP (10 попыток / 5 минут) и per-email (20 / час), `429` с
`Retry-After`, состояние в памяти (инстанс один). IP берётся из `X-Forwarded-For` только от
доверенных прокси (`TRUSTED_PROXIES`, по умолчанию — сеть compose). Каждая неудача — структурированная
запись `slog` с email и IP. Закрывает S-03; nginx-конфиги и fail2ban удаляются.

### A-10. Офлайн-синхронизация — вне v1

Клиент v1 работает онлайн. Настоящий sync требует монотонной `revision`, tombstones для удалений и
курсора `(revision, id)` — `updated_since` по `updated_at` из триггеров теряет удаления и близкие
правки. Если потребность появится, это отдельный spec; A-07 уже даёт идемпотентность.

### A-11. Развёртывание

Одна топология: `deploy/docker-compose.yml` = приложение + `caddy:2-alpine`. Caddyfile без
`rate_limit` и `/static`, CSP для JSON, HSTS. Бэкапы — CLI `backup` по cron хоста в `BACKUP_DIR`
(один механизм вместо трёх: Go-сервис, `backup.sh`, Makefile). Restore — только по ssh.
nginx, certbot, fail2ban, `minimal`/`prod`/`nginx` compose-варианты и native systemd удаляются.

### A-12. Что удаляется

`internal/web/**` (шаблоны, статика, рендерер, cookie-сессии, CSRF, HTMX-партиалы),
`echo-contrib`, `gorilla/sessions`, инвайты, `budget_alerts`, `user_sessions`, заглушки плановых
отчётов/прогноза/бенчмарков, роль `child`, `.claude/skills/test-frontend`, `docs/specs/003-ui-ux-audit.md`
(после удаления UI — история).

## Открытые вопросы владельцу

Планы написаны под вариант по умолчанию; смена ответа меняет одну-две задачи.

| Вопрос | По умолчанию | Альтернатива |
|---|---|---|
| Единый токен или access + refresh с ротацией (A-01) | единый | ротация: +1 endpoint, +колонка, клонирование обнаруживается |
| Офлайн-режим в v1 (A-10) | нет | revision + tombstones в схему до старта Android |
| CSV-экспорт отчётов | оставить (`GET /reports/:id/export`) | удалить вместе с `ExportReport` |
| Restore из бэкапа через API | нет, только ssh | `POST /backups/:name/restore` под admin |

## Планы

Порядок обязателен: каждый план оставляет `make test` и `make lint` зелёными, веб-интерфейс
живёт до плана 03, чтобы логику дашборда и отчётов переносить, а не переписывать.

Схема меняется правкой `001_consolidated` на месте, а не новыми миграциями. golang-migrate
хранит только номер версии, поэтому на уже мигрированной БД такая правка — no-op; до первого
релиза локальные и серверные БД пересоздаются (`make db-reset`, появляется в плане 03).

| # | План | Что даёт |
|---|---|---|
| 01 | [api-contract](../plans/20260904-01-api-contract.md) | `openapi.yaml`, тест покрытия роутов, версия сборки, `api_standards.md` |
| 02 | [api-completeness](../plans/20260904-02-api-completeness.md) | `StatsService`, генерация отчётов, бэкапы, список пользователей, единый envelope и пагинация |
| 03 | [bearer-auth-web-removal](../plans/20260904-03-bearer-auth-web-removal.md) | `internal/auth`, `/auth/*`, `/me`, CLI setup, лимитер, удаление `internal/web`, чистка конфига |
| 04 | [schema-and-money](../plans/20260904-04-schema-and-money.md) | переписанная `001_consolidated`, `amount_minor`, календарные даты, часовой пояс, удаление инвайтов и `child` |
| 05 | [deploy-ffs](../plans/20260904-05-deploy-ffs.md) | compose + Caddy для `ffs.shatrov.tech`, CLI `backup`, чистка `deploy/` |

Android-приложение стоит начинать после плана 04: до него контракт денег и дат не совпадает с
`openapi.yaml`.
