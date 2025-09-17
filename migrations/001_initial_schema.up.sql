-- Family Budget Service - Initial Schema Migration

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create schema
CREATE SCHEMA IF NOT EXISTS family_budget;

-- Set search path
SET search_path TO family_budget, public;

-- Create enum types
CREATE TYPE user_role AS ENUM ('admin', 'member', 'child');
CREATE TYPE transaction_type AS ENUM ('income', 'expense');
CREATE TYPE category_type AS ENUM ('income', 'expense');
CREATE TYPE budget_period AS ENUM ('weekly', 'monthly', 'yearly', 'custom');
CREATE TYPE report_type AS ENUM ('expenses', 'income', 'budget', 'cash_flow', 'category_breakdown');
CREATE TYPE report_period AS ENUM ('daily', 'weekly', 'monthly', 'yearly', 'custom');

-- Create families table
CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT families_name_not_empty CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT families_currency_valid CHECK (LENGTH(currency) = 3 AND currency = UPPER(currency))
);

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'member',
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_first_name_not_empty CHECK (LENGTH(TRIM(first_name)) > 0),
    CONSTRAINT users_last_name_not_empty CHECK (LENGTH(TRIM(last_name)) > 0),
    CONSTRAINT users_password_hash_not_empty CHECK (LENGTH(TRIM(password_hash)) > 0)
);

-- Create categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type category_type NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT categories_name_not_empty CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT categories_no_self_reference CHECK (id != parent_id),
    CONSTRAINT categories_unique_name_per_family_type UNIQUE (family_id, name, type, parent_id)
);

-- Create transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    amount DECIMAL(15,2) NOT NULL,
    description TEXT NOT NULL,
    date DATE NOT NULL,
    type transaction_type NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    tags JSONB DEFAULT '[]'::jsonb,
    receipt_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_description_not_empty CHECK (LENGTH(TRIM(description)) > 0),
    CONSTRAINT transactions_date_reasonable CHECK (date >= '1900-01-01' AND date <= CURRENT_DATE + INTERVAL '1 year'),
    CONSTRAINT transactions_tags_is_array CHECK (jsonb_typeof(tags) = 'array')
);

-- Create budgets table
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    spent DECIMAL(15,2) DEFAULT 0,
    period budget_period NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT budgets_name_not_empty CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT budgets_amount_positive CHECK (amount > 0),
    CONSTRAINT budgets_spent_non_negative CHECK (spent >= 0),
    CONSTRAINT budgets_date_range_valid CHECK (end_date > start_date),
    CONSTRAINT budgets_unique_name_per_family_period UNIQUE (family_id, name, start_date, end_date)
);

-- Create budget_alerts table
CREATE TABLE budget_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    budget_id UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    threshold_percentage INTEGER NOT NULL,
    is_triggered BOOLEAN DEFAULT false,
    triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT budget_alerts_threshold_valid CHECK (threshold_percentage > 0 AND threshold_percentage <= 100),
    CONSTRAINT budget_alerts_unique_threshold_per_budget UNIQUE (budget_id, threshold_percentage)
);

-- Create reports table
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    type report_type NOT NULL,
    period report_period NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    data JSONB NOT NULL,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    generated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    is_cached BOOLEAN DEFAULT false,
    cache_expires_at TIMESTAMP WITH TIME ZONE,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT reports_name_not_empty CHECK (LENGTH(TRIM(name)) > 0),
    CONSTRAINT reports_date_range_valid CHECK (end_date >= start_date),
    CONSTRAINT reports_data_is_object CHECK (jsonb_typeof(data) = 'object')
);

-- Create user_sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT user_sessions_token_not_empty CHECK (LENGTH(TRIM(session_token)) > 0),
    CONSTRAINT user_sessions_expires_future CHECK (expires_at > created_at)
);

-- Create indexes for performance
CREATE INDEX idx_users_family_id ON users(family_id);
CREATE INDEX idx_users_email_active ON users(email, is_active);
CREATE INDEX idx_users_role_family ON users(role, family_id);

CREATE INDEX idx_categories_family_type ON categories(family_id, type);
CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_family_active ON categories(family_id, is_active);

CREATE INDEX idx_transactions_family_date ON transactions(family_id, date DESC);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_category_id ON transactions(category_id);
CREATE INDEX idx_transactions_type_family ON transactions(type, family_id);
CREATE INDEX idx_transactions_date_range ON transactions(date);
CREATE INDEX idx_transactions_family_date_type ON transactions(family_id, date, type);

-- GIN indexes for JSONB fields
CREATE INDEX idx_transactions_tags_gin ON transactions USING GIN (tags);

CREATE INDEX idx_budgets_family_active ON budgets(family_id, is_active);
CREATE INDEX idx_budgets_category_id ON budgets(category_id);
CREATE INDEX idx_budgets_date_range ON budgets(start_date, end_date);
CREATE INDEX idx_budgets_family_period ON budgets(family_id, start_date, end_date);

CREATE INDEX idx_budget_alerts_budget_id ON budget_alerts(budget_id);
CREATE INDEX idx_budget_alerts_triggered ON budget_alerts(is_triggered, triggered_at);

CREATE INDEX idx_reports_family_type ON reports(family_id, type);
CREATE INDEX idx_reports_generated_by ON reports(generated_by);
CREATE INDEX idx_reports_date_range ON reports(start_date, end_date);
CREATE INDEX idx_reports_cached ON reports(is_cached, cache_expires_at);

-- GIN index for reports data
CREATE INDEX idx_reports_data_gin ON reports USING GIN (data);

CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at);

-- Create triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_families_updated_at BEFORE UPDATE ON families FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_transactions_updated_at BEFORE UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_budgets_updated_at BEFORE UPDATE ON budgets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create trigger for budget spent calculation
CREATE OR REPLACE FUNCTION update_budget_spent()
RETURNS TRIGGER AS $$
BEGIN
    -- Update spent amount for all budgets that include this transaction's category
    UPDATE budgets SET spent = (
        SELECT COALESCE(SUM(t.amount), 0)
        FROM transactions t
        WHERE t.type = 'expense'
        AND t.family_id = budgets.family_id
        AND t.date BETWEEN budgets.start_date AND budgets.end_date
        AND (budgets.category_id IS NULL OR t.category_id = budgets.category_id)
    )
    WHERE family_id = COALESCE(NEW.family_id, OLD.family_id)
    AND (category_id IS NULL OR category_id = COALESCE(NEW.category_id, OLD.category_id));

    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

CREATE TRIGGER update_budget_spent_on_transaction
    AFTER INSERT OR UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_budget_spent();

-- Create trigger for budget alerts
CREATE OR REPLACE FUNCTION check_budget_alerts()
RETURNS TRIGGER AS $$
DECLARE
    alert_record RECORD;
    usage_percentage DECIMAL(5,2);
BEGIN
    -- Calculate usage percentage
    IF NEW.amount > 0 THEN
        usage_percentage := (NEW.spent / NEW.amount * 100)::DECIMAL(5,2);

        -- Check all alerts for this budget
        FOR alert_record IN
            SELECT * FROM budget_alerts
            WHERE budget_id = NEW.id AND NOT is_triggered
        LOOP
            IF usage_percentage >= alert_record.threshold_percentage THEN
                UPDATE budget_alerts
                SET is_triggered = true, triggered_at = NOW()
                WHERE id = alert_record.id;
            END IF;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER check_budget_alerts_on_update
    AFTER UPDATE OF spent ON budgets
    FOR EACH ROW EXECUTE FUNCTION check_budget_alerts();