-- Family Budget Service — откат консолидированной схемы: всё в обратном порядке.

DROP TRIGGER IF EXISTS update_invites_updated_at;
DROP TRIGGER IF EXISTS update_budgets_updated_at;
DROP TRIGGER IF EXISTS update_transactions_updated_at;
DROP TRIGGER IF EXISTS update_categories_updated_at;
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP TRIGGER IF EXISTS update_families_updated_at;

DROP INDEX IF EXISTS idx_sessions_user_id;

DROP INDEX IF EXISTS idx_invites_family_id;
DROP INDEX IF EXISTS idx_invites_token;

DROP INDEX IF EXISTS idx_reports_generated_by;
DROP INDEX IF EXISTS idx_reports_family_type;

DROP INDEX IF EXISTS idx_budgets_family_period;
DROP INDEX IF EXISTS idx_budgets_category_id;
DROP INDEX IF EXISTS idx_budgets_family_active;

DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_transactions_category_id;
DROP INDEX IF EXISTS idx_transactions_family_type_date;
DROP INDEX IF EXISTS idx_transactions_family_date;

DROP INDEX IF EXISTS idx_categories_family_active;
DROP INDEX IF EXISTS idx_categories_parent_id;
DROP INDEX IF EXISTS idx_categories_family_type;

DROP INDEX IF EXISTS idx_users_email_active;
DROP INDEX IF EXISTS idx_users_family_id;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS families;
