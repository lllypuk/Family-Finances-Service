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
- **Sessions**: gorilla/sessions v1.4.0
- **Password Hashing**: golang.org/x/crypto/bcrypt
- **Testing**: testify v1.10.0 + in-memory SQLite

### Frontend (Web Interface)
- **Framework**: HTMX v2.0.4+ для dynamic updates
- **CSS**: PicoCSS v2.1.1+ minimalist framework
- **Templates**: Go Templates с layout system
- **Static Assets**: Echo static middleware
- **PWA**: Service Worker ready

### Инфраструктура
- **Контейнеризация**: Docker & Docker Compose
- **Multi-platform**: linux/amd64, linux/arm64
- **CI/CD**: GitHub Actions (ci, docker, security, release)
- **Registry**: не используется — образ собирается локально из `docker/Dockerfile`
  (релизов нет, в GHCR ничего не опубликовано; см. [004-deployment-readiness.md](specs/004-deployment-readiness.md#d-02))
- **Security Scanning**: CodeQL, Semgrep, TruffleHog, OSV Scanner

### Документация API
- **Спецификация**: OpenAPI 3.0 (планируется)
- **Генерация**: go generate (в development)
- **UI**: Swagger UI (планируется)
- **Тестирование**: HTTP тесты

## 🗂️ Структура проекта

```
Family-Finances-Service/
├── cmd/                    # Точки входа приложения
│   └── server/            # HTTP сервер
├── internal/              # Приватный код приложения
│   ├── domain/           # Domain entities и бизнес-логика
│   ├── application/      # Application layer с интерфейсами
│   ├── infrastructure/   # Реализация репозиториев (SQLite)
│   ├── config.go         # Конфигурация приложения
│   └── run.go           # Bootstrap приложения
├── generated/             # Автогенерированный код (OpenAPI)
├── docs/                 # Документация проекта (specs, plans, guides, patterns)
├── docker-compose.yml    # Docker окружение
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
- **Схема**: Сессии с HTTP-only cookies
- **Роли**: Admin, Member, Child
- **CSRF защита**: Токены в формах

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
github.com/gorilla/sessions        # Session management
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
- **XSS**: Content Security Policy
- **CORS**: Настроенные CORS политики Echo
- **Rate Limiting**: ⬜ не реализован в приложении. Ограничение запросов есть только в
  конфигурациях nginx/Caddy и fail2ban; `docker-compose.minimal.yml` и native systemd им не покрыты
  (находка [S-03](specs/002-security-audit.md#s-03) открыта)

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
- [ ] Переход на API-only для Android: планы 01–05 в `docs/plans/`, решения в
  [specs/005-api-only-redesign.md](specs/005-api-only-redesign.md) — удаление раздела
  «Frontend» выше, bearer-аутентификация, деньги в минимальных единицах, один compose с Caddy
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
