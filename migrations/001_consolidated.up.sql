-- Family Budget Service - Consolidated UP Migration
-- This file combines all migrations in chronological order

-- ==============================================================================
-- Migration 001: Initial Schema (SQLite)
-- ==============================================================================

-- Enable foreign keys (must be set per connection)
PRAGMA foreign_keys = ON;

-- Create families table
CREATE TABLE IF NOT EXISTS families (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (LENGTH(currency) = 3 AND currency = UPPER(currency))
);

-- Create users table
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

    -- Constraints
    CHECK (role IN ('admin', 'member', 'child')),
    CHECK (email LIKE '%_@__%.__%'),
    CHECK (LENGTH(TRIM(first_name)) > 0),
    CHECK (LENGTH(TRIM(last_name)) > 0),
    CHECK (LENGTH(TRIM(password_hash)) > 0),
    CHECK (is_active IN (0, 1))
);

-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    parent_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (type IN ('income', 'expense')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (id != parent_id),
    CHECK (is_active IN (0, 1)),
    UNIQUE (family_id, name, type, parent_id)
);

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    amount REAL NOT NULL,
    description TEXT NOT NULL,
    date DATE NOT NULL,
    type TEXT NOT NULL,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    tags TEXT DEFAULT '[]',
    receipt_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (type IN ('income', 'expense')),
    CHECK (amount > 0),
    CHECK (LENGTH(TRIM(description)) > 0),
    CHECK (date >= '1900-01-01')
);

-- Create budgets table
CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    amount REAL NOT NULL,
    spent REAL DEFAULT 0,
    period TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (period IN ('weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (amount > 0),
    CHECK (spent >= 0),
    CHECK (end_date > start_date),
    CHECK (is_active IN (0, 1)),
    UNIQUE (family_id, name, start_date, end_date)
);

-- Create budget_alerts table
CREATE TABLE IF NOT EXISTS budget_alerts (
    id TEXT PRIMARY KEY,
    budget_id TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    threshold_percentage INTEGER NOT NULL,
    is_triggered INTEGER DEFAULT 0,
    triggered_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (threshold_percentage > 0 AND threshold_percentage <= 100),
    CHECK (is_triggered IN (0, 1)),
    UNIQUE (budget_id, threshold_percentage)
);

-- Create reports table
CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    period TEXT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    data TEXT NOT NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    generated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    is_cached INTEGER DEFAULT 0,
    cache_expires_at DATETIME,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (type IN ('expenses', 'income', 'budget', 'cash_flow', 'category_breakdown')),
    CHECK (period IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (end_date >= start_date),
    CHECK (is_cached IN (0, 1))
);

-- Create user_sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id TEXT PRIMARY KEY,
    session_token TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CHECK (LENGTH(TRIM(session_token)) > 0),
    CHECK (expires_at > created_at)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_family_id ON users(family_id);
CREATE INDEX IF NOT EXISTS idx_users_email_active ON users(email, is_active);
CREATE INDEX IF NOT EXISTS idx_users_role_family ON users(role, family_id);

CREATE INDEX IF NOT EXISTS idx_categories_family_type ON categories(family_id, type);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_family_active ON categories(family_id, is_active);

CREATE INDEX IF NOT EXISTS idx_transactions_family_date ON transactions(family_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type_family ON transactions(type, family_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date_range ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_family_date_type ON transactions(family_id, date, type);

CREATE INDEX IF NOT EXISTS idx_budgets_family_active ON budgets(family_id, is_active);
CREATE INDEX IF NOT EXISTS idx_budgets_category_id ON budgets(category_id);
CREATE INDEX IF NOT EXISTS idx_budgets_date_range ON budgets(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_budgets_family_period ON budgets(family_id, start_date, end_date);

CREATE INDEX IF NOT EXISTS idx_budget_alerts_budget_id ON budget_alerts(budget_id);
CREATE INDEX IF NOT EXISTS idx_budget_alerts_triggered ON budget_alerts(is_triggered, triggered_at);

CREATE INDEX IF NOT EXISTS idx_reports_family_type ON reports(family_id, type);
CREATE INDEX IF NOT EXISTS idx_reports_generated_by ON reports(generated_by);
CREATE INDEX IF NOT EXISTS idx_reports_date_range ON reports(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_reports_cached ON reports(is_cached, cache_expires_at);

CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);

-- Create triggers for updated_at
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

-- ==============================================================================
-- Migration 002: Fix budget trigger (SQLite)
-- ==============================================================================
-- Note: This migration is no longer needed for SQLite as trigger logic
-- has been moved to Go code in BudgetRepository.
-- This file is kept for migration versioning compatibility.
-- No operations needed

-- ==============================================================================
-- Migration 003: Performance optimization indexes (SQLite)
-- ==============================================================================

-- Index for monthly summary queries (GetMonthlySummary)
CREATE INDEX IF NOT EXISTS idx_transactions_monthly_summary
ON transactions (family_id, date, category_id, type, amount);

-- Index for complex transaction filtering
CREATE INDEX IF NOT EXISTS idx_transactions_complex_filter
ON transactions (family_id, type, date, amount)
WHERE type IS NOT NULL;

-- Index for transaction summary calculations
CREATE INDEX IF NOT EXISTS idx_transactions_summary_calc
ON transactions (family_id, date, type, amount);

-- Index for category hierarchy operations (GetCategoryChildren)
CREATE INDEX IF NOT EXISTS idx_categories_hierarchy
ON categories (family_id, parent_id, id)
WHERE is_active = 1;

-- Index for pagination operations
CREATE INDEX IF NOT EXISTS idx_transactions_pagination
ON transactions (family_id, date DESC, id);

-- Index for budget trigger performance
CREATE INDEX IF NOT EXISTS idx_transactions_budget_calc
ON transactions (family_id, category_id, type, date, amount)
WHERE type = 'expense';

-- Partial index for active budgets lookup
CREATE INDEX IF NOT EXISTS idx_budgets_active_lookup
ON budgets (family_id, category_id, start_date, end_date)
WHERE is_active = 1;

-- Update statistics for query optimizer
ANALYZE;

-- ==============================================================================
-- Migration 004: Fix budget_alerts trigger schema references (SQLite)
-- ==============================================================================
-- Note: This migration is no longer needed for SQLite as trigger logic
-- has been moved to Go code in BudgetRepository.
-- This file is kept for migration versioning compatibility.
-- No operations needed

-- ==============================================================================
-- Migration 005: Create invites table
-- ==============================================================================

-- Create invites table for user invitation system
CREATE TABLE IF NOT EXISTS invites (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'member', 'child')),
    token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'accepted', 'expired', 'revoked')),
    expires_at DATETIME NOT NULL,
    accepted_at DATETIME,
    accepted_by TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (accepted_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_invites_token ON invites(token);
CREATE INDEX IF NOT EXISTS idx_invites_family_id ON invites(family_id);
CREATE INDEX IF NOT EXISTS idx_invites_email ON invites(email);
CREATE INDEX IF NOT EXISTS idx_invites_status ON invites(status);
CREATE INDEX IF NOT EXISTS idx_invites_expires_at ON invites(expires_at);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_invites_updated_at
AFTER UPDATE ON invites
FOR EACH ROW
BEGIN
    UPDATE invites SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
