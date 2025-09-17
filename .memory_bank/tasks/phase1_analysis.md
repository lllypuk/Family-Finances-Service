# Phase 1: PostgreSQL Migration Analysis - ЗАВЕРШЕНО ✅

## Исходная схема (до миграции)

### Коллекции и структуры данных

#### 1. Users Collection
```go
type User struct {
    ID        uuid.UUID `bson:"_id"`
    Email     string    `bson:"email"`          // Unique index
    Password  string    `bson:"password"`       // bcrypt hash
    FirstName string    `bson:"first_name"`
    LastName  string    `bson:"last_name"`
    Role      Role      `bson:"role"`           // "admin", "member", "child"
    FamilyID  uuid.UUID `bson:"family_id"`
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}
```

#### 2. Families Collection
```go
type Family struct {
    ID        uuid.UUID `bson:"_id"`
    Name      string    `bson:"name"`
    Currency  string    `bson:"currency"`       // USD, RUB, EUR
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}
```

#### 3. Categories Collection
```go
type Category struct {
    ID        uuid.UUID  `bson:"_id"`
    Name      string     `bson:"name"`
    Type      Type       `bson:"type"`           // "income", "expense"
    Color     string     `bson:"color"`          // #FF5733
    Icon      string     `bson:"icon"`
    ParentID  *uuid.UUID `bson:"parent_id,omitempty"`  // Hierarchy support
    FamilyID  uuid.UUID  `bson:"family_id"`
    IsActive  bool       `bson:"is_active"`
    CreatedAt time.Time  `bson:"created_at"`
    UpdatedAt time.Time  `bson:"updated_at"`
}
```

#### 4. Transactions Collection
```go
type Transaction struct {
    ID          uuid.UUID `bson:"_id"`
    Amount      float64   `bson:"amount"`
    Type        Type      `bson:"type"`           // "income", "expense"
    Description string    `bson:"description"`
    CategoryID  uuid.UUID `bson:"category_id"`
    UserID      uuid.UUID `bson:"user_id"`
    FamilyID    uuid.UUID `bson:"family_id"`
    Date        time.Time `bson:"date"`
    Tags        []string  `bson:"tags"`          // Array for search
    CreatedAt   time.Time `bson:"created_at"`
    UpdatedAt   time.Time `bson:"updated_at"`
}
```

#### 5. Budgets Collection
```go
type Budget struct {
    ID         uuid.UUID  `bson:"_id"`
    Name       string     `bson:"name"`
    Amount     float64    `bson:"amount"`         // Limit
    Spent      float64    `bson:"spent"`          // Current spent
    Period     Period     `bson:"period"`         // "weekly", "monthly", "yearly", "custom"
    CategoryID *uuid.UUID `bson:"category_id,omitempty"`
    FamilyID   uuid.UUID  `bson:"family_id"`
    StartDate  time.Time  `bson:"start_date"`
    EndDate    time.Time  `bson:"end_date"`
    IsActive   bool       `bson:"is_active"`
    CreatedAt  time.Time  `bson:"created_at"`
    UpdatedAt  time.Time  `bson:"updated_at"`
}

type Alert struct {
    ID          uuid.UUID  `bson:"_id"`
    BudgetID    uuid.UUID  `bson:"budget_id"`
    Threshold   float64    `bson:"threshold"`      // Percentage (50, 80, 100)
    IsTriggered bool       `bson:"is_triggered"`
    TriggeredAt *time.Time `bson:"triggered_at,omitempty"`
    CreatedAt   time.Time  `bson:"created_at"`
}
```

#### 6. Reports Collection
```go
type Report struct {
    ID          uuid.UUID `bson:"_id"`
    Name        string    `bson:"name"`
    Type        Type      `bson:"type"`           // "expenses", "income", "budget", etc.
    Period      Period    `bson:"period"`         // "daily", "weekly", "monthly", etc.
    FamilyID    uuid.UUID `bson:"family_id"`
    UserID      uuid.UUID `bson:"user_id"`
    StartDate   time.Time `bson:"start_date"`
    EndDate     time.Time `bson:"end_date"`
    Data        Data      `bson:"data"`           // Complex embedded document
    GeneratedAt time.Time `bson:"generated_at"`
}

type Data struct {
    TotalIncome       float64                 `bson:"total_income"`
    TotalExpenses     float64                 `bson:"total_expenses"`
    NetIncome         float64                 `bson:"net_income"`
    CategoryBreakdown []CategoryReportItem    `bson:"category_breakdown"`
    DailyBreakdown    []DailyReportItem       `bson:"daily_breakdown"`
    TopExpenses       []TransactionReportItem `bson:"top_expenses"`
    BudgetComparison  []BudgetComparisonItem  `bson:"budget_comparison"`
}
```

### Существующие индексы
- **users.email**: Unique index для предотвращения дублирования
- Возможные составные индексы на:
  - transactions: (family_id, date)
  - budgets: (family_id, is_active)
  - categories: (family_id, type)

## Проектирование PostgreSQL схемы

### Основные таблицы

```sql
-- Families table (parent table for multi-tenancy)
CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL CHECK (length(currency) = 3),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Users table with foreign key to families
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'member', 'child')),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Categories table with self-referencing foreign key for hierarchy
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('income', 'expense')),
    color VARCHAR(7) CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    icon VARCHAR(100),
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Transactions table with JSONB for tags
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    type VARCHAR(10) NOT NULL CHECK (type IN ('income', 'expense')),
    description TEXT NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    user_id UUID NOT NULL REFERENCES users(id),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    tags JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Budgets table
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    spent DECIMAL(15,2) DEFAULT 0 CHECK (spent >= 0),
    period VARCHAR(20) NOT NULL CHECK (period IN ('weekly', 'monthly', 'yearly', 'custom')),
    category_id UUID REFERENCES categories(id),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT check_date_range CHECK (end_date > start_date),
    CONSTRAINT check_spent_not_exceed_amount CHECK (spent <= amount)
);

-- Budget alerts table (normalized from embedded document)
CREATE TABLE budget_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    threshold DECIMAL(5,2) NOT NULL CHECK (threshold > 0 AND threshold <= 100),
    is_triggered BOOLEAN DEFAULT false,
    triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Reports table with JSONB for complex data
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('expenses', 'income', 'budget', 'cash_flow', 'category_break')),
    period VARCHAR(20) NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    data JSONB NOT NULL,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT check_report_date_range CHECK (end_date >= start_date)
);
```

### Индексы для производительности

```sql
-- Core indexes for family-based isolation
CREATE INDEX idx_users_family_id ON users(family_id);
CREATE INDEX idx_categories_family_id ON categories(family_id);
CREATE INDEX idx_transactions_family_id ON transactions(family_id);
CREATE INDEX idx_budgets_family_id ON budgets(family_id);
CREATE INDEX idx_reports_family_id ON reports(family_id);

-- Query optimization indexes
CREATE INDEX idx_transactions_date ON transactions(date);
CREATE INDEX idx_transactions_category_id ON transactions(category_id);
CREATE INDEX idx_transactions_family_date ON transactions(family_id, date);
CREATE INDEX idx_budgets_active ON budgets(family_id, is_active) WHERE is_active = true;

-- Hierarchy support for categories
CREATE INDEX idx_categories_parent_id ON categories(parent_id) WHERE parent_id IS NOT NULL;

-- JSONB indexes
CREATE INDEX idx_transactions_tags ON transactions USING GIN(tags);
CREATE INDEX idx_reports_data ON reports USING GIN(data);

-- Text search index for transactions
CREATE INDEX idx_transactions_description ON transactions USING GIN(to_tsvector('english', description));
```

## Выявленные особенности миграции

### 1. Изменения в типах данных
- **float64 → DECIMAL(15,2)**: Точная арифметика для денежных сумм
- **[]string → JSONB**: Теги транзакций
- **embedded documents → JSONB**: Данные отчетов
- **time.Time → TIMESTAMP WITH TIME ZONE**: Лучшая поддержка временных зон

### 2. Нормализация данных
- **Budget Alerts**: Отдельная таблица вместо встроенного документа
- **Constraints**: Добавлены проверки на уровне БД
- **Foreign Keys**: Строгие связи между таблицами

### 3. Новые возможности PostgreSQL
- **CHECK constraints**: Валидация на уровне БД
- **CASCADE deletes**: Автоматическая очистка связанных данных
- **GIN indexes**: Полнотекстовый поиск и JSONB
- **Partial indexes**: Оптимизация для активных бюджетов

## Последние версии библиотек (из context7)

### PGX (Native Driver) - Основной выбор
```go
// go.mod
github.com/jackc/pgx/v5 v5.7.2
github.com/jackc/pgx/v5/pgxpool v5.7.2
github.com/jackc/pgx/v5/stdlib v5.7.2  // database/sql compatibility
```

### Migrations
```go
// go.mod
github.com/golang-migrate/migrate/v4 v4.18.3
```

### Установка миграционного CLI
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Рекомендуемый стек для миграции

### Выбор: Прямые SQL запросы с PGX
**Рекомендация: PGX без ORM**
- Максимальный контроль над SQL запросами
- Лучшая производительность без ORM overhead
- Явные SQL запросы для лучшей читаемости
- Полное использование PostgreSQL возможностей
- Простота отладки и профилирования запросов

### Структура проекта после миграции
```
internal/
├── infrastructure/
│   ├── postgresql.go           # PGX connection pool
│   ├── migrations/             # SQL миграции
│   │   ├── 000001_init.up.sql
│   │   ├── 000001_init.down.sql
│   │   ├── 000002_indexes.up.sql
│   │   └── 000002_indexes.down.sql
│   ├── queries/                # SQL файлы для сложных запросов
│   │   ├── user_queries.sql
│   │   ├── transaction_queries.sql
│   │   └── report_queries.sql
│   └── repositories/
│       ├── user_repository.go      # Pure SQL + pgx implementation
│       ├── family_repository.go    # Raw SQL queries
│       ├── category_repository.go  # Hierarchical SQL queries
│       ├── transaction_repository.go # Analytical SQL
│       ├── budget_repository.go    # Complex aggregations
│       └── report_repository.go    # Advanced PostgreSQL analytics
```

## План реализации Phase 1 ✅

- [x] Проанализирована исходная схема
- [x] Получены последние версии библиотек через context7
- [x] Спроектирована PostgreSQL схема с нормализацией
- [x] Определены необходимые индексы
- [x] Выбран технологический стек (PGX + golang-migrate, без ORM)
- [x] Создан план структуры проекта с прямыми SQL запросами

**Готово к переходу к Phase 2: Настройка PostgreSQL окружения**

## Преимущества выбранного подхода

### Прямые SQL запросы vs ORM
1. **Производительность**: Нет overhead от ORM слоя
2. **Контроль**: Полный контроль над генерируемыми запросами
3. **PostgreSQL features**: Прямое использование CTE, window functions, JSONB операций
4. **Отладка**: Простота профилирования и оптимизации запросов
5. **Читаемость**: Явные SQL запросы легче понимать и поддерживать

### Архитектурные принципы
- Сохранение Clean Architecture с Repository pattern
- Изоляция SQL запросов в отдельных файлах для сложной логики
- Использование prepared statements для безопасности
- Connection pooling через pgxpool для производительности
