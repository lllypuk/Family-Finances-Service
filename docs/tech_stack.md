# Технологический паспорт - Family Finances Service

## 🏗️ Архитектурный обзор

### Общая архитектура
- **Тип**: Self-hosted сервис (один Docker-образ ~50MB)
- **Стиль**: RESTful API + Clean Architecture
- **Развертывание**: Один Docker-контейнер
- **База данных**: SQLite (встроенная, без внешних зависимостей)

### Архитектурные принципы
- **Clean Architecture**: Разделение на слои (Domain, Use Cases, Interface Adapters, Infrastructure)
- **Dependency Inversion**: Зависимости направлены внутрь к бизнес-логике
- **Single Responsibility**: Каждый компонент имеет одну ответственность
- **API First**: API проектируется до реализации

## 💻 Основной технологический стек

### Backend
- **Язык**: Go 1.26+
- **Framework**: Echo Web Framework v4.15.4
- **База данных**: SQLite (modernc.org/sqlite — pure Go, без CGO)
- **Валидация**: go-playground/validator v10.27.0
- **UUID**: google/uuid v1.6.0 для идентификаторов
- **Аутентификация**: `internal/auth` — bearer-токены, таблица `sessions`, лимитер логина
- **Password Hashing**: golang.org/x/crypto/bcrypt, cost 12
- **Testing**: testify v1.10.0 + in-memory SQLite

### Клиент
- Android-приложение, генерируется из `docs/api/openapi.yaml`; HTML сервис не отдаёт

### Инфраструктура
- **Контейнеризация**: Docker & Docker Compose
- **Multi-platform**: linux/amd64, linux/arm64
- **CI/CD**: GitHub Actions (ci, docker, security, release)
- **Registry**: не используется — образ собирается локально из `docker/Dockerfile`
  (релизов нет, в GHCR ничего не опубликовано; см. [004-deployment-readiness.md](specs/004-deployment-readiness.md#d-02))
- **Security Scanning**: CodeQL, Semgrep, TruffleHog, OSV Scanner

### Документация API
- **Спецификация**: OpenAPI 3.1 — `docs/api/openapi.yaml`, роут без описания валит `make test`
- **Тестирование**: интеграционные HTTP-тесты в `tests/integration/`

## 🗂️ Структура проекта

```
Family-Finances-Service/
├── cmd/                    # Точки входа приложения
│   └── server/            # HTTP сервер + подкоманды `setup`, `reset-password`
├── internal/              # Приватный код приложения
│   ├── domain/           # Domain entities и бизнес-логика
│   ├── auth/             # Bearer-токены, сессии, middleware, лимитер логина
│   ├── application/      # Echo, JSON error handler, хендлеры /api/v1
│   ├── services/         # Бизнес-логика
│   ├── infrastructure/   # Реализация репозиториев (SQLite), миграции
│   ├── bootstrap.go      # OpenDatabase, Setup, ResetPassword — общие для сервера и CLI
│   ├── config.go         # Конфигурация приложения
│   └── run.go           # Bootstrap приложения
├── migrations/            # 001_consolidated.{up,down}.sql
├── docs/                 # Документация проекта (specs, plans, guides, patterns, api)
├── docker/               # Dockerfile, docker-compose.yml
└── Makefile              # Автоматизация задач
```

## 🔧 Инструменты разработки

### Сборка и зависимости
- **Менеджер пакетов**: Go Modules
- **Сборка**: Make + Dockerfile
- **Линтеры**: golangci-lint
- **Форматирование**: gofmt, goimports

### Тестирование
- **Unit тесты**: testing пакет Go
- **Mocking**: testify/mock
- **Integration тесты**: in-memory SQLite (без Docker, мгновенный запуск)
- **Coverage**: go test -cover

### Observability
- **Логирование**: slog (structured logging)
- **Health checks**: /health эндпоинт

## 🗄️ База данных

### SQLite конфигурация
- **Driver**: modernc.org/sqlite (pure Go, без CGO)
- **Хранение**: Один файл `./data/budget.db`
- **Миграции**: Консолидированные (up/down), выполняются автоматически при старте
- **Подход**: Единая миграция в двух файлах (`001_consolidated.up/down.sql`)
- **Бэкапы**: Копирование файла базы данных

### Дизайн БД
- **Подход**: Relational database (SQLite)
- **Schema**: Строгая типизация с foreign keys
- **Индексы**: B-tree индексы для оптимизации
- **WAL mode**: Для конкурентного чтения
- **Миграции**: Консолидированный подход (все изменения в одном файле)

### Основные таблицы
```sql
families       -- Семейные профили
users          -- Пользователи (члены семей)
transactions   -- Финансовые транзакции
categories     -- Категории доходов/расходов
budgets        -- Бюджеты и планы
reports        -- Сгенерированные отчеты
invites        -- Приглашения пользователей
sessions       -- Bearer-сессии (хеш токена, устройство, сроки)
budget_alerts  -- Оповещения о бюджете
```

### Управление миграциями
- **Файлы**: `migrations/001_consolidated.up.sql` и `migrations/001_consolidated.down.sql`
- **Автоматизация**: Миграции применяются при старте приложения через `golang-migrate`
- **Добавление изменений**: Редактирование консолидированных файлов напрямую
- **Откат**: Полный откат через `down.sql` (удаление всех объектов)
- **Идемпотентность**: Использование `IF NOT EXISTS` / `IF EXISTS`

### Особенности SQLite
- **Встроенная БД**: Не требует отдельного сервера
- **Pure Go**: Нет зависимости от CGO/C-компилятора
- **Foreign Keys**: Для referential integrity
- **Single-tenant**: Один экземпляр обслуживает одну семью

## 🌐 API Design

### REST принципы
- **Ресурсно-ориентированный**: /families/{id}/transactions
- **HTTP методы**: GET, POST, PUT, DELETE
- **Статус-коды**: Стандартные HTTP коды
- **Content-Type**: application/json

### Аутентификация и авторизация
- **Схема**: `Authorization: Bearer <token>`; токен непрозрачный, в БД — SHA-256; 30 дней без
  активности, не дольше 180 дней. Cookie и CSRF нет
- **Роли**: Admin, Member, Child; проверяются на каждом запросе из БД, а не из токена
- **Bootstrap**: семьи и первого администратора — только CLI `setup`; логин до него — `409 SETUP_REQUIRED`
- **Лимитер логина**: 10 попыток с IP за 5 минут, 20 на email за час, `429` + `Retry-After`;
  IP из `X-Forwarded-For` только от `TRUSTED_PROXIES`

## 🚀 DevOps и развертывание

### Локальная разработка
```bash
# Запуск всех сервисов
make docker-up

# Запуск приложения локально
make run-local

# Форматирование и линтинг
make fmt && make lint

# Тестирование
make test
```

### Среды
- **Development**: `make run-local` (localhost:8080, SQLite)
- **Production**: Docker-контейнер (~50MB Alpine-based image)

### Мониторинг
- **Healthcheck**: /health эндпоинт
- **Logging**: Structured JSON logs (slog)

## 📦 Зависимости

### Основные Go модули
```go
github.com/labstack/echo/v4        # Web framework
modernc.org/sqlite                 # SQLite driver (pure Go)
github.com/google/uuid             # UUID generation
golang.org/x/crypto                # bcrypt
github.com/golang-migrate/migrate  # Migrations
```

### Dev зависимости
```go
github.com/stretchr/testify       # Testing utilities
```

## 🔒 Безопасность

### Принципы
- **Defense in Depth**: Многоуровневая защита
- **Least Privilege**: Минимальные права доступа
- **Data Encryption**: Шифрование в покое и в движении
- **Input Validation**: Валидация всех входных данных

### Реализация
- **SQL Injection**: Параметризованные запросы и валидация данных
- **Пароли**: bcrypt cost 12, политика 10…72 байта одна для API и CLI; неизвестный email
  сравнивается с фиктивным хешем, ответ одинаков
- **Rate Limiting**: лимитер логина в приложении (`internal/auth/ratelimit.go`), не зависит от reverse proxy
  ([S-03](specs/002-security-audit.md#s-03) закрыта)
- **Секретов в конфиге нет**: токены случайные и хранятся хешем, подписывать нечего

## 📈 Производительность

### Целевые метрики
- **Response Time**: < 200ms (95th percentile)
- **Throughput**: > 1000 RPS
- **Availability**: 99.9%
- **Recovery Time**: < 1 минута

### Оптимизации
- **SQLite**: Индексы, WAL mode, prepared statements
- **Compression**: gzip middleware Echo
- **Profiling/metrics**: нет — ни pprof, ни `/metrics`; только `/health` и JSON-лог

## 🔄 Планы развития

### Ближайшие обновления
- [ ] Переход на API-only для Android: планы 01–03 выполнены (bearer, удаление веб-слоя), 04–05 в
  `docs/plans/` — деньги в минимальных единицах, один compose с Caddy; решения в
  [specs/005-api-only-redesign.md](specs/005-api-only-redesign.md)
- [ ] Улучшение аналитики и отчетов

### Среднесрочные планы
- [ ] Офлайн-синхронизация мобильного клиента (A-10: revision + tombstones)
- [ ] Расширенная система уведомлений

### Долгосрочная перспектива
- [ ] AI-рекомендации по оптимизации трат

## 📚 Полезные ресурсы

### Документация
- [Go Documentation](https://golang.org/doc/)
- [Echo Framework](https://echo.labstack.com/guide/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)

### Лучшие практики
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

*Последнее обновление: 2025*
*Технический лидер: Development Team*
*Частота ревизий: ежемесячно*
