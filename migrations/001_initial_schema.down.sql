-- Family Budget Service - Drop Initial Schema

-- Set search path
SET search_path TO family_budget, public;

-- Drop triggers
DROP TRIGGER IF EXISTS check_budget_alerts_on_update ON budgets;
DROP TRIGGER IF EXISTS update_budget_spent_on_transaction ON transactions;
DROP TRIGGER IF EXISTS update_budgets_updated_at ON budgets;
DROP TRIGGER IF EXISTS update_transactions_updated_at ON transactions;
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_families_updated_at ON families;

-- Drop functions
DROP FUNCTION IF EXISTS check_budget_alerts();
DROP FUNCTION IF EXISTS update_budget_spent();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_user_sessions_expires;
DROP INDEX IF EXISTS idx_user_sessions_user_id;
DROP INDEX IF EXISTS idx_user_sessions_token;

DROP INDEX IF EXISTS idx_reports_data_gin;
DROP INDEX IF EXISTS idx_reports_cached;
DROP INDEX IF EXISTS idx_reports_date_range;
DROP INDEX IF EXISTS idx_reports_generated_by;
DROP INDEX IF EXISTS idx_reports_family_type;

DROP INDEX IF EXISTS idx_budget_alerts_triggered;
DROP INDEX IF EXISTS idx_budget_alerts_budget_id;

DROP INDEX IF EXISTS idx_budgets_family_period;
DROP INDEX IF EXISTS idx_budgets_date_range;
DROP INDEX IF EXISTS idx_budgets_category_id;
DROP INDEX IF EXISTS idx_budgets_family_active;

DROP INDEX IF EXISTS idx_transactions_tags_gin;
DROP INDEX IF EXISTS idx_transactions_family_date_type;
DROP INDEX IF EXISTS idx_transactions_date_range;
DROP INDEX IF EXISTS idx_transactions_type_family;
DROP INDEX IF EXISTS idx_transactions_category_id;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_transactions_family_date;

DROP INDEX IF EXISTS idx_categories_family_active;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_family_type;

DROP INDEX IF EXISTS idx_users_role_family;
DROP INDEX IF EXISTS idx_users_email_active;
DROP INDEX IF EXISTS idx_users_family_id;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS budget_alerts;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS families;

-- Drop types
DROP TYPE IF EXISTS report_period;
DROP TYPE IF EXISTS report_type;
DROP TYPE IF EXISTS budget_period;
DROP TYPE IF EXISTS category_type;
DROP TYPE IF EXISTS transaction_type;
DROP TYPE IF EXISTS user_role;

-- Drop schema (only if empty)
DROP SCHEMA IF EXISTS family_budget;

-- Drop extensions (only if not used by other databases)
-- DROP EXTENSION IF EXISTS "pg_stat_statements";
-- DROP EXTENSION IF EXISTS "uuid-ossp";