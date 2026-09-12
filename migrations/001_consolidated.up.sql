-- Family Budget Service — consolidated schema.
-- Единственная миграция: до первого релиза схема меняется правкой этого файла,
-- а базы пересоздаются (`make db-reset`) — golang-migrate хранит только номер версии.

PRAGMA foreign_keys = ON;

-- ==============================================================================
-- Tables
-- ==============================================================================

CREATE TABLE IF NOT EXISTS families (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    -- timezone: IANA-зона семьи, по ней считаются границы периодов (A-06)
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    -- singleton: ровно одна семья на инсталляцию (A-02) — вторая INSERT падает на UNIQUE
    singleton INTEGER NOT NULL DEFAULT 1 CHECK (singleton = 1) UNIQUE,

    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (LENGTH(currency) = 3 AND currency = UPPER(currency)),
    CHECK (LENGTH(TRIM(timezone)) > 0)
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    last_login DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (role IN ('admin', 'member')),
    CHECK (email LIKE '%_@__%.__%'),
    CHECK (LENGTH(TRIM(first_name)) > 0),
    CHECK (LENGTH(TRIM(last_name)) > 0),
    CHECK (LENGTH(TRIM(password_hash)) > 0),
    CHECK (is_active IN (0, 1))
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    description TEXT,
    parent_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (type IN ('income', 'expense')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (id != parent_id),
    CHECK (is_active IN (0, 1))
    -- Уникальность имени — частичный индекс idx_categories_unique_active ниже: табличный UNIQUE
    -- не различал бы мягко удалённые строки, и имя подкатегории оставалось бы занятым навсегда.
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    amount_minor INTEGER NOT NULL,
    description TEXT NOT NULL,
    date TEXT NOT NULL,
    type TEXT NOT NULL,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    tags TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (type IN ('income', 'expense')),
    CHECK (amount_minor > 0),
    CHECK (LENGTH(TRIM(description)) > 0),
    CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')
);

CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    spent_minor INTEGER NOT NULL DEFAULT 0,
    period TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (period IN ('weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (amount_minor > 0),
    CHECK (spent_minor >= 0),
    CHECK (start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    -- Строго больше: бюджет на один день бессмыслен, и то же требует budget.ValidatePeriod.
    -- У отчётов ниже стоит >=, там однодневный период допустим.
    CHECK (end_date > start_date),
    CHECK (is_active IN (0, 1))
);

CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    period TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    data TEXT NOT NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    generated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (type IN ('expenses', 'income', 'budget', 'cash_flow', 'category_breakdown')),
    CHECK (period IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date >= start_date)
);

-- Только хеш токена; срок продлевается активностью (см. internal/auth/session.go)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    device_name TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    last_used_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);

-- ==============================================================================
-- Indexes
-- ==============================================================================

CREATE INDEX IF NOT EXISTS idx_users_family_id ON users(family_id);
CREATE INDEX IF NOT EXISTS idx_users_email_active ON users(email, is_active);

CREATE INDEX IF NOT EXISTS idx_categories_family_type ON categories(family_id, type);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_family_active ON categories(family_id, is_active);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_unique_active
    ON categories(family_id, name, type, parent_id) WHERE is_active = 1;

CREATE INDEX IF NOT EXISTS idx_transactions_family_date ON transactions(family_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_family_type_date ON transactions(family_id, type, date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);

CREATE INDEX IF NOT EXISTS idx_budgets_family_active ON budgets(family_id, is_active);
CREATE INDEX IF NOT EXISTS idx_budgets_category_id ON budgets(category_id);
CREATE INDEX IF NOT EXISTS idx_budgets_family_period ON budgets(family_id, start_date, end_date);
-- Частичный UNIQUE вместо табличного: мягко удалённая строка остаётся в таблице, и общий
-- UNIQUE запрещал бы создать бюджет с тем же именем и периодом заново — конфликт с бюджетом,
-- которого клиент уже не видит (GetByID и все списки фильтруют is_active).
CREATE UNIQUE INDEX IF NOT EXISTS idx_budgets_name_period_active
    ON budgets(family_id, name, start_date, end_date) WHERE is_active = 1;

CREATE INDEX IF NOT EXISTS idx_reports_family_type ON reports(family_id, type);
CREATE INDEX IF NOT EXISTS idx_reports_generated_by ON reports(generated_by);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- ==============================================================================
-- Triggers: updated_at
-- ==============================================================================

CREATE TRIGGER IF NOT EXISTS update_families_updated_at
AFTER UPDATE ON families
BEGIN
    UPDATE families SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_users_updated_at
AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_categories_updated_at
AFTER UPDATE ON categories
BEGIN
    UPDATE categories SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_transactions_updated_at
AFTER UPDATE ON transactions
BEGIN
    UPDATE transactions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_budgets_updated_at
AFTER UPDATE ON budgets
BEGIN
    UPDATE budgets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

ANALYZE;
