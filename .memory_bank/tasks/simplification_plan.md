# План упрощения проекта для Self-Hosted развёртывания

## Цель

Упростить проект для self-hosted развёртывания в **одном контейнере**:
- Заменить PostgreSQL на SQLite
- Удалить мониторинг (Prometheus, Grafana, Jaeger, ELK)
- Минимизировать зависимости и инфраструктуру

---

## Текущее состояние

| Компонент | Статус | Сложность удаления |
|-----------|--------|-------------------|
| PostgreSQL (pgx/v5) | 6 репозиториев, 4 миграции | **Высокая** |
| Prometheus/Grafana | Метрики HTTP, БД, бизнес | Средняя |
| Jaeger (OTLP) | Трейсинг в Echo middleware | Низкая |
| postgres-exporter | 449 строк кастомных запросов | Низкая |
| testcontainers | Интеграционные тесты | Средняя |

---

## Фаза 1: Удаление мониторинга

**Приоритет**: Высокий
**Риск**: Низкий
**Статус**: ✅ **ЗАВЕРШЕНО** (2026-01-24)

### 1.1. Удаление Prometheus метрик

**Файлы для удаления/изменения:**

| Файл | Действие |
|------|----------|
| `internal/observability/metrics.go` | Удалить полностью |
| `internal/observability/tracing.go` | Удалить полностью |
| `internal/run.go` | Убрать инициализацию metrics/tracing |
| `internal/application/http_server.go` | Убрать `/metrics` endpoint, otelecho middleware |

**Что остаётся:**
- `internal/observability/logging.go` — базовое логирование (slog)
- `internal/observability/health.go` — health checks (`/health`, `/ready`, `/live`)

### 1.2. Удаление инфраструктуры мониторинга

**Удалить полностью:**

```
monitoring/                           # ~1000+ строк конфигов
├── prometheus/
│   ├── prometheus.yml
│   ├── alert_rules.yml
│   └── recording_rules.yml
├── grafana/
│   ├── dashboards/
│   └── provisioning/
├── alertmanager/
│   └── alertmanager.yml
└── postgres_exporter/
    └── queries.yaml
```

### 1.3. Обновление Docker Compose

**Файл**: `docker/docker-compose.yml`

**Удалить сервисы:**
- `prometheus`
- `alertmanager`
- `grafana`
- `jaeger`
- `node-exporter`
- `postgres-exporter`

**Удалить профили:**
- `observability`

### 1.4. Обновление Makefile

**Удалить цели:**
- `observability-up`
- `observability-down`
- `full-up` (или переделать)
- Команды связанные с Prometheus/Grafana

### 1.5. Обновление конфигурации

**Файл**: `internal/config.go`

**Удалить переменные:**
```go
PROMETHEUS_PORT
GRAFANA_PORT
JAEGER_UI_PORT
JAEGER_COLLECTOR_PORT
NODE_EXPORTER_PORT
POSTGRES_EXPORTER_PORT
ALERTMANAGER_PORT
ENABLE_METRICS
ENABLE_TRACING
```

### 1.6. Обновление go.mod

**Удалить зависимости:**
```
go.opentelemetry.io/otel
go.opentelemetry.io/otel/exporters/otlp/otlptrace
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go.opentelemetry.io/otel/sdk
go.opentelemetry.io/otel/trace
go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho
github.com/prometheus/client_golang
```

---

## Фаза 2: Миграция PostgreSQL → SQLite

**Приоритет**: Высокий
**Риск**: Высокий (требует тщательного тестирования)
**Статус**: ⏳ Не начато

### 2.1. PostgreSQL-специфичные конструкции для замены

| Конструкция | PostgreSQL | SQLite замена |
|-------------|------------|---------------|
| ENUM типы | `CREATE TYPE user_role AS ENUM (...)` | `TEXT CHECK(column IN (...))` |
| UUID генерация | `uuid_generate_v4()` | Go: `github.com/google/uuid` |
| Текущее время | `NOW()` | `CURRENT_TIMESTAMP` |
| JSONB | `JSONB` | `JSON` (TEXT) |
| Триггерные функции | PL/pgSQL | Go код или SQLite триггеры |
| GIN индексы | `CREATE INDEX ... USING GIN` | Обычные индексы |
| Регулярные выражения | `~ '^[a-zA-Z0-9...]+'` | Валидация в Go |
| Extensions | `uuid-ossp`, `pg_stat_statements` | Не нужны |

### 2.2. Выбор SQLite драйвера

**Рекомендация**: `modernc.org/sqlite`

| Драйвер | CGO | Преимущества |
|---------|-----|--------------|
| `modernc.org/sqlite` | Не требует | Pure Go, CGO_ENABLED=0, scratch образ |
| `github.com/mattn/go-sqlite3` | Требует | Быстрее, но сложнее сборка |

### 2.3. Создание SQLite подключения

**Новый файл**: `internal/infrastructure/sqlite.go`

```go
package infrastructure

import (
    "database/sql"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

type SQLiteConnection struct {
    db *sql.DB
}

func NewSQLiteConnection(dbPath string) (*SQLiteConnection, error) {
    // Создать директорию если не существует
    dir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }

    // Подключение с WAL mode для лучшей производительности
    dsn := dbPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000"
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }

    // SQLite настройки для продакшена
    db.SetMaxOpenConns(1) // SQLite не поддерживает несколько писателей
    db.SetMaxIdleConns(1)

    return &SQLiteConnection{db: db}, nil
}
```

### 2.4. Переписать миграции

**Текущие файлы:**
- `migrations/001_initial_schema.up.sql` (275+ строк)
- `migrations/002_fix_budget_trigger.up.sql`
- `migrations/003_performance_indexes.up.sql`
- `migrations/004_fix_budget_alerts_schema.up.sql`

**Изменения в 001_initial_schema.up.sql:**

```sql
-- PostgreSQL (удалить):
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TYPE user_role AS ENUM ('admin', 'member', 'child');

-- SQLite (заменить на):
-- UUID генерируется в Go коде
-- ENUM заменяется на CHECK constraint:
CREATE TABLE users (
    id TEXT PRIMARY KEY,  -- UUID как TEXT
    role TEXT NOT NULL CHECK(role IN ('admin', 'member', 'child')),
    email TEXT UNIQUE NOT NULL,
    -- ...
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в SQLite:
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 2.5. Перенос триггерной логики в Go

**Триггеры для переноса:**

1. **update_updated_at_column** → SQLite триггер (простой)
2. **update_budget_spent** → Go код в BudgetRepository
3. **check_budget_alerts** → Go код в BudgetRepository

**Пример переноса update_budget_spent:**

```go
// internal/infrastructure/budget/budget_repository.go

func (r *BudgetRepository) RecalculateSpent(ctx context.Context, budgetID uuid.UUID) error {
    query := `
        UPDATE budgets
        SET spent = (
            SELECT COALESCE(SUM(amount), 0)
            FROM transactions
            WHERE category_id = budgets.category_id
            AND date BETWEEN budgets.start_date AND budgets.end_date
        )
        WHERE id = ?
    `
    _, err := r.db.ExecContext(ctx, query, budgetID.String())
    return err
}
```

### 2.6. Адаптация репозиториев

**Файлы для проверки/изменения:**

| Репозиторий | Файл | Изменения |
|-------------|------|-----------|
| User | `internal/infrastructure/user/user_repository.go` | `$1` → `?`, UUID как TEXT |
| Family | `internal/infrastructure/user/family_repository.go` | `$1` → `?`, UUID как TEXT |
| Category | `internal/infrastructure/category/category_repository.go` | WITH RECURSIVE → проверить совместимость |
| Transaction | `internal/infrastructure/transaction/transaction_repository.go` | JSONB → JSON, `$1` → `?` |
| Budget | `internal/infrastructure/budget/budget_repository.go` | Добавить вызов RecalculateSpent |
| Report | `internal/infrastructure/report/report_repository.go` | JSONB → JSON |

**Основные изменения в SQL:**
- Placeholder: `$1, $2, $3` → `?, ?, ?`
- RETURNING: `RETURNING id` → отдельный SELECT или `last_insert_rowid()`
- UUID: `uuid_generate_v4()` → генерация в Go
- NOW(): `NOW()` → `CURRENT_TIMESTAMP` или `datetime('now')`

### 2.7. Обновление тестов

**Файл**: `internal/testhelpers/postgresql.go` → `sqlite.go`

```go
func SetupSQLiteTestDB(t *testing.T) *sql.DB {
    // Используем in-memory для тестов
    db, err := sql.Open("sqlite", ":memory:?_foreign_keys=ON")
    require.NoError(t, err)

    // Применить миграции
    // ...

    t.Cleanup(func() {
        db.Close()
    })

    return db
}
```

### 2.8. Обновление конфигурации

**Файл**: `internal/config.go`

```go
// Было (PostgreSQL):
type DatabaseConfig struct {
    URI             string
    Name            string
    MaxOpenConns    int
    MaxIdleConns    int
    // ...
}

// Стало (SQLite):
type DatabaseConfig struct {
    Path string // /data/budget.db
}
```

**Переменные окружения:**
```
# Было:
POSTGRESQL_URI=postgres://...
POSTGRESQL_DATABASE=family_budget

# Стало:
DATABASE_PATH=/data/budget.db
```

---

## Фаза 3: Единый контейнер

**Приоритет**: Средний
**Риск**: Низкий
**Статус**: ⏳ Не начато

### 3.1. Новый Dockerfile

**Файл**: `docker/Dockerfile`

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Кэширование зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Сборка
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /family-budget-service ./cmd/server

# Final stage
FROM alpine:latest

# Добавить ca-certificates и tzdata
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник
COPY --from=builder /family-budget-service .

# Копируем миграции
COPY migrations ./migrations

# Копируем статику и шаблоны
COPY internal/web/static ./internal/web/static
COPY internal/web/templates ./internal/web/templates

# Создаём директорию для данных
RUN mkdir -p /data && chown -R 65534:65534 /data

# Non-root user
USER 65534

# Volume для SQLite
VOLUME ["/data"]

# Переменные по умолчанию
ENV DATABASE_PATH=/data/budget.db
ENV SERVER_PORT=8080
ENV LOG_LEVEL=info

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/family-budget-service"]
```

### 3.2. Упрощённый docker-compose.yml

```yaml
version: '3.8'

services:
  family-budget:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    container_name: family-budget
    ports:
      - "8080:8080"
    volumes:
      - budget_data:/data
    environment:
      - DATABASE_PATH=/data/budget.db
      - SERVER_PORT=8080
      - LOG_LEVEL=info
      - SESSION_SECRET=${SESSION_SECRET:-change-me-in-production}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  budget_data:
```

### 3.3. Упрощённый Makefile

**Удалить цели:**
- Все postgres-* команды
- observability-* команды
- migrate-* команды (встроить в приложение)

**Добавить:**
```makefile
# Сборка и запуск
build:
	CGO_ENABLED=0 go build -o ./build/family-budget-service ./cmd/server

run-local:
	DATABASE_PATH=./data/budget.db \
	SESSION_SECRET=dev-secret \
	LOG_LEVEL=debug \
	go run ./cmd/server

# Docker
docker-build:
	docker build -t family-budget-service -f docker/Dockerfile .

docker-run:
	docker run -d -p 8080:8080 -v budget_data:/data family-budget-service

# Backup/Restore
backup:
	cp ./data/budget.db ./backups/budget_$(date +%Y%m%d_%H%M%S).db

restore:
	cp $(BACKUP_FILE) ./data/budget.db
```

### 3.4. Автоматические миграции при старте

**Файл**: `cmd/server/main.go` или `internal/run.go`

```go
func (a *App) Run(ctx context.Context) error {
    // Автоматически применить миграции при старте
    if err := a.runMigrations(); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    // Запуск HTTP сервера
    // ...
}
```

---

## Итоговая архитектура

```
┌─────────────────────────────────────┐
│     Self-Hosted Docker Container    │
├─────────────────────────────────────┤
│  family-budget-service              │
│  ├─ HTTP Server (Echo) :8080        │
│  ├─ SQLite DB (/data/budget.db)     │
│  ├─ Web UI (HTMX + PicoCSS)         │
│  ├─ Logging (slog → stdout)         │
│  └─ Health checks (/health)         │
│                                     │
│  Volume: /data (SQLite + backups)   │
└─────────────────────────────────────┘
```

---

## Чеклист задач

### Фаза 1: Удаление мониторинга ✅ **ЗАВЕРШЕНО**
- [x] 1.1. Удалить `internal/observability/metrics.go` ✅
- [x] 1.2. Удалить `internal/observability/tracing.go` ✅
- [x] 1.3. Обновить `internal/observability/observability.go` (убрать InitMetrics/InitTracing) ✅
- [x] 1.4. Обновить `internal/application/http_server.go` (убрать /metrics, otelecho, PrometheusMiddleware) ✅
- [x] 1.5. Обновить `internal/observability/middleware.go` (убрать PrometheusMiddleware, MetricsHandler) ✅
- [x] 1.6. Удалить папку `monitoring/` ✅
- [x] 1.7. Обновить `docker/docker-compose.yml` (удалить 6 observability сервисов) ✅
- [x] 1.8. Обновить `Makefile` (удалить observability команды) ✅
- [x] 1.9. `internal/config.go` - не требовал изменений ✅
- [x] 1.10. Обновить `go.mod` (удалить 6 зависимостей otel/prometheus) ✅
- [x] 1.11. Исправить тесты `internal/application/http_server_test.go` ✅
- [x] 1.12. Запустить `make test && make lint` - **0 ошибок, все тесты прошли** ✅

### Фаза 2: Миграция на SQLite
- [ ] 2.1. Добавить `modernc.org/sqlite` в go.mod
- [ ] 2.2. Создать `internal/infrastructure/sqlite.go`
- [ ] 2.3. Переписать миграции для SQLite
- [ ] 2.4. Перенести триггерную логику в Go
- [ ] 2.5. Адаптировать UserRepository
- [ ] 2.6. Адаптировать FamilyRepository
- [ ] 2.7. Адаптировать CategoryRepository
- [ ] 2.8. Адаптировать TransactionRepository
- [ ] 2.9. Адаптировать BudgetRepository
- [ ] 2.10. Адаптировать ReportRepository
- [ ] 2.11. Обновить `internal/config.go`
- [ ] 2.12. Обновить тесты
- [ ] 2.13. Удалить pgx зависимости из go.mod
- [ ] 2.14. Полное тестирование всех endpoints

### Фаза 3: Единый контейнер
- [ ] 3.1. Обновить Dockerfile
- [ ] 3.2. Упростить docker-compose.yml
- [ ] 3.3. Упростить Makefile
- [ ] 3.4. Добавить авто-миграции при старте
- [ ] 3.5. Обновить .env.example
- [ ] 3.6. Обновить README.md
- [ ] 3.7. Обновить CLAUDE.md
- [ ] 3.8. Финальное тестирование

---

## Риски и митигация

| Риск | Вероятность | Митигация |
|------|-------------|-----------|
| Несовместимость SQL запросов | Высокая | Тщательное тестирование каждого репозитория |
| Потеря данных при миграции | Средняя | Скрипт миграции данных pg→sqlite |
| Performance issues с SQLite | Низкая | WAL mode, правильные индексы |
| Concurrent writes в SQLite | Средняя | Сериализация записи, connection pool = 1 |

---

## Документация для обновления

После завершения обновить:
- [ ] `README.md` — новые инструкции развёртывания
- [ ] `CLAUDE.md` — убрать PostgreSQL команды, добавить SQLite
- [ ] `.env.example` — новые переменные окружения
- [ ] `.memory_bank/tech_stack.md` — обновить стек технологий

---

**Создано**: 2026-01-24
**Статус**: Планирование
**Ответственный**: TBD
