# [SIMPLE-002] 🗄️ Миграция PostgreSQL → SQLite

## Информация о задаче

- **Приоритет**: Высокий
- **Риск**: Высокий (требует тщательного тестирования)
- **Статус**: ⏳ Не начато
- **Фаза**: 2 из 3 (Упрощение проекта для Self-Hosted)
- **Зависит от**: [SIMPLE-001] ✅ Завершено

## Цель

Заменить PostgreSQL на SQLite для упрощения self-hosted развёртывания в одном контейнере.

## Описание

Второй этап упрощения проекта - миграция с PostgreSQL (pgx/v5) на SQLite:
- Заменить драйвер БД на `modernc.org/sqlite` (Pure Go, без CGO)
- Переписать 4 миграции для совместимости с SQLite
- Адаптировать 6 репозиториев (изменить SQL синтаксис)
- Перенести триггерную логику из PostgreSQL в Go код
- Обновить конфигурацию и тесты

**Что требует изменения:**
- PostgreSQL-специфичные конструкции: ENUM, UUID, JSONB, триггеры
- SQL синтаксис: плейсхолдеры `$1` → `?`, RETURNING clause
- Настройки подключения: connection pool, WAL mode, foreign keys

---

## PostgreSQL → SQLite: Таблица замен

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
| Плейсхолдеры | `$1, $2, $3` | `?, ?, ?` |
| RETURNING | `INSERT ... RETURNING id` | `last_insert_rowid()` или SELECT |

---

## План выполнения

### 2.1. Добавление SQLite драйвера

**Файл**: `go.mod`

**Добавить зависимость:**
```bash
go get modernc.org/sqlite
```

**Почему `modernc.org/sqlite`:**
- Pure Go (CGO_ENABLED=0)
- Совместимость с Alpine/scratch образами
- Упрощённая сборка без C компилятора

### 2.2. Создание SQLite подключения

**Новый файл**: `internal/infrastructure/sqlite.go`

**Основной функционал:**
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

**Важные настройки:**
- `_journal_mode=WAL` - Write-Ahead Logging для производительности
- `_foreign_keys=ON` - Включение внешних ключей
- `_busy_timeout=5000` - Таймаут при конфликтах записи
- `MaxOpenConns=1` - SQLite ограничение на одного писателя

### 2.3. Переписывание миграций

**Текущие файлы:**
- `migrations/001_initial_schema.up.sql` (275+ строк)
- `migrations/002_fix_budget_trigger.up.sql`
- `migrations/003_performance_indexes.up.sql`
- `migrations/004_fix_budget_alerts_schema.up.sql`

**Изменения в 001_initial_schema.up.sql:**

#### До (PostgreSQL):
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TYPE user_role AS ENUM ('admin', 'member', 'child');
CREATE TYPE transaction_type AS ENUM ('income', 'expense');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role user_role NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### После (SQLite):
```sql
-- UUID генерируется в Go коде
-- ENUM заменяется на CHECK constraint

CREATE TABLE users (
    id TEXT PRIMARY KEY,  -- UUID как TEXT
    role TEXT NOT NULL CHECK(role IN ('admin', 'member', 'child')),
    email TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в SQLite
CREATE TRIGGER update_users_updated_at
AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

**Основные изменения:**
- Удалить `CREATE EXTENSION`
- Удалить `CREATE TYPE` (заменить на CHECK)
- `UUID` → `TEXT`
- `VARCHAR` → `TEXT`
- `TIMESTAMP` → `DATETIME`
- `NOW()` → `CURRENT_TIMESTAMP`
- Создать триггеры для `updated_at`

### 2.4. Перенос триггерной логики в Go

**Триггеры для переноса:**

1. **update_updated_at_column** → SQLite триггер (простой)
2. **update_budget_spent** → Go код в BudgetRepository
3. **check_budget_alerts** → Go код в BudgetRepository

**Пример переноса update_budget_spent:**

**Файл**: `internal/infrastructure/budget/budget_repository.go`

```go
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

// Вызывать после создания/обновления транзакции:
func (r *TransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
    // ... создание транзакции ...

    // Пересчитать бюджет если транзакция связана с категорией
    if tx.CategoryID != nil {
        budget, err := r.budgetRepo.FindByCategory(ctx, *tx.CategoryID)
        if err == nil && budget != nil {
            _ = r.budgetRepo.RecalculateSpent(ctx, budget.ID)
        }
    }

    return nil
}
```

### 2.5. Адаптация репозиториев

**Файлы для изменения:**

| Репозиторий | Файл | Основные изменения |
|-------------|------|-------------------|
| User | `internal/infrastructure/user/user_repository.go` | `$1` → `?`, UUID.String(), RETURNING |
| Family | `internal/infrastructure/user/family_repository.go` | `$1` → `?`, UUID.String(), RETURNING |
| Category | `internal/infrastructure/category/category_repository.go` | WITH RECURSIVE (проверить), `$1` → `?` |
| Transaction | `internal/infrastructure/transaction/transaction_repository.go` | JSONB → JSON, `$1` → `?`, RETURNING |
| Budget | `internal/infrastructure/budget/budget_repository.go` | Добавить RecalculateSpent, `$1` → `?` |
| Report | `internal/infrastructure/report/report_repository.go` | JSONB → JSON, `$1` → `?` |

**Основные паттерны изменений:**

#### Плейсхолдеры:
```go
// До (PostgreSQL):
query := `INSERT INTO users (id, email) VALUES ($1, $2)`
db.ExecContext(ctx, query, user.ID, user.Email)

// После (SQLite):
query := `INSERT INTO users (id, email) VALUES (?, ?)`
db.ExecContext(ctx, query, user.ID.String(), user.Email)
```

#### RETURNING clause:
```go
// До (PostgreSQL):
query := `INSERT INTO users (...) VALUES (...) RETURNING id`
var id uuid.UUID
err := db.QueryRowContext(ctx, query, ...).Scan(&id)

// После (SQLite):
query := `INSERT INTO users (id, ...) VALUES (?, ...)`
id := uuid.New()
_, err := db.ExecContext(ctx, query, id.String(), ...)
```

#### UUID обработка:
```go
// До (PostgreSQL):
var id uuid.UUID
err := row.Scan(&id, ...)

// После (SQLite):
var idStr string
err := row.Scan(&idStr, ...)
id, _ := uuid.Parse(idStr)
```

### 2.6. Обновление конфигурации

**Файл**: `internal/config.go`

#### До (PostgreSQL):
```go
type DatabaseConfig struct {
    URI             string
    Name            string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}
```

#### После (SQLite):
```go
type DatabaseConfig struct {
    Path string // /data/budget.db
}
```

**Переменные окружения:**

#### Удалить:
```
POSTGRESQL_URI
POSTGRESQL_DATABASE
DB_MAX_OPEN_CONNS
DB_MAX_IDLE_CONNS
DB_CONN_MAX_LIFETIME
```

#### Добавить:
```
DATABASE_PATH=/data/budget.db
```

### 2.7. Обновление тестов

**Файл**: `internal/testhelpers/postgresql.go` → `sqlite.go`

#### До (PostgreSQL с testcontainers):
```go
func SetupPostgreSQLTestDB(t *testing.T) *sql.DB {
    // Запуск testcontainer с PostgreSQL
    // ...
}
```

#### После (SQLite in-memory):
```go
func SetupSQLiteTestDB(t *testing.T) *sql.DB {
    // Используем in-memory для тестов
    db, err := sql.Open("sqlite", ":memory:?_foreign_keys=ON")
    require.NoError(t, err)

    // Применить миграции
    err = runMigrations(db)
    require.NoError(t, err)

    t.Cleanup(func() {
        db.Close()
    })

    return db
}
```

**Преимущества SQLite для тестов:**
- ✅ Мгновенный старт (без Docker)
- ✅ Изолированная БД для каждого теста
- ✅ Автоматическая очистка (in-memory)
- ✅ Упрощение CI/CD

### 2.8. Обновление go.mod

**Добавить:**
```
modernc.org/sqlite
```

**Удалить:**
```
github.com/jackc/pgx/v5
github.com/jackc/pgx/v5/pgxpool
github.com/testcontainers/testcontainers-go/modules/postgres
```

---

## Чеклист выполнения

### Подготовка
- [ ] 2.1. Добавить `modernc.org/sqlite` в go.mod
- [ ] 2.2. Создать `internal/infrastructure/sqlite.go`
- [ ] 2.3. Обновить `internal/config.go` (DatabaseConfig)

### Миграции
- [ ] 2.4. Переписать `001_initial_schema.up.sql`:
  - [ ] Удалить CREATE EXTENSION
  - [ ] Заменить ENUM на CHECK constraints
  - [ ] UUID → TEXT, TIMESTAMP → DATETIME
  - [ ] Создать триггеры для updated_at
- [ ] 2.5. Переписать `002_fix_budget_trigger.up.sql`
- [ ] 2.6. Переписать `003_performance_indexes.up.sql`
- [ ] 2.7. Переписать `004_fix_budget_alerts_schema.up.sql`

### Адаптация репозиториев
- [ ] 2.8. Адаптировать UserRepository:
  - [ ] `$1` → `?`
  - [ ] UUID.String() для параметров
  - [ ] RETURNING → отдельный SELECT/uuid.New()
- [ ] 2.9. Адаптировать FamilyRepository
- [ ] 2.10. Адаптировать CategoryRepository (проверить WITH RECURSIVE)
- [ ] 2.11. Адаптировать TransactionRepository (JSONB → JSON)
- [ ] 2.12. Адаптировать BudgetRepository:
  - [ ] Добавить метод RecalculateSpent
  - [ ] Интегрировать вызов из TransactionRepository
- [ ] 2.13. Адаптировать ReportRepository (JSONB → JSON)

### Перенос триггерной логики
- [ ] 2.14. Реализовать update_budget_spent в Go
- [ ] 2.15. Реализовать check_budget_alerts в Go
- [ ] 2.16. Создать SQLite триггеры для updated_at

### Обновление тестов
- [ ] 2.17. Создать `internal/testhelpers/sqlite.go`
- [ ] 2.18. Обновить все репозиторные тесты для SQLite
- [ ] 2.19. Удалить testcontainers PostgreSQL зависимости
- [ ] 2.20. Обновить интеграционные тесты

### Очистка и проверка
- [ ] 2.21. Удалить `internal/infrastructure/postgresql.go` (если есть)
- [ ] 2.22. Обновить `.env.example` (DATABASE_PATH)
- [ ] 2.23. Запустить `go mod tidy`
- [ ] 2.24. Запустить `make fmt`
- [ ] 2.25. Запустить `make test` - **все тесты должны пройти**
- [ ] 2.26. Запустить `make lint` - **0 ошибок**
- [ ] 2.27. Запустить `make build` - успешная сборка

### Функциональное тестирование
- [ ] 2.28. Создать тестовую БД SQLite
- [ ] 2.29. Проверить создание пользователя
- [ ] 2.30. Проверить создание семьи
- [ ] 2.31. Проверить категории (WITH RECURSIVE)
- [ ] 2.32. Проверить транзакции
- [ ] 2.33. Проверить бюджеты и пересчёт spent
- [ ] 2.34. Проверить отчёты
- [ ] 2.35. Проверить все API endpoints
- [ ] 2.36. Проверить web интерфейс

---

## Ожидаемый результат

После выполнения задачи:
- ✅ SQLite заменил PostgreSQL во всём проекте
- ✅ Все миграции совместимы с SQLite
- ✅ 6 репозиториев адаптированы (новый SQL синтаксис)
- ✅ Триггерная логика перенесена в Go код
- ✅ Тесты используют in-memory SQLite
- ✅ Удалены pgx и testcontainers зависимости
- ✅ Конфигурация обновлена (DATABASE_PATH)
- ✅ Все тесты проходят
- ✅ Линтер возвращает 0 ошибок
- ✅ Приложение работает с SQLite БД

---

## Риски и митигация

| Риск | Вероятность | Митигация |
|------|-------------|-----------|
| Несовместимость SQL запросов | Высокая | Тщательное тестирование каждого репозитория с SQLite |
| WITH RECURSIVE не работает | Средняя | Протестировать на CategoryRepository, возможно переписать логику |
| Потеря данных при миграции | Средняя | Создать скрипт экспорта/импорта PostgreSQL → SQLite |
| Performance issues с SQLite | Низкая | WAL mode, правильные индексы, оптимизация запросов |
| Concurrent writes проблемы | Средняя | MaxOpenConns=1, тестирование concurrent операций |
| JSON вместо JSONB медленнее | Низкая | Для small-scale приложения не критично |

---

## Следующие шаги

После завершения этой задачи переходим к:
- **[SIMPLE-003]** Единый контейнер (Фаза 3)

---

## Справочные материалы

**SQLite документация:**
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
- [SQLite Foreign Keys](https://www.sqlite.org/foreignkeys.html)
- [SQLite Triggers](https://www.sqlite.org/lang_createtrigger.html)

**Драйвер modernc.org/sqlite:**
- [GitHub Repository](https://gitlab.com/cznic/sqlite)
- Pure Go, CGO_ENABLED=0 совместимость

**Миграция данных:**
- Потребуется скрипт для экспорта из PostgreSQL и импорта в SQLite
- Можно использовать CSV или JSON как промежуточный формат

---

**Создано**: 2026-01-24
**Обновлено**: 2026-01-24
**Ответственный**: TBD
