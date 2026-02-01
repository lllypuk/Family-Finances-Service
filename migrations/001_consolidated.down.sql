-- Family Budget Service - Consolidated DOWN Migration
-- This file combines all migrations in REVERSE chronological order

-- ==============================================================================
-- Migration 005 Down: Drop invites table
-- ==============================================================================

-- Drop trigger
DROP TRIGGER IF EXISTS update_invites_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_invites_expires_at;
DROP INDEX IF EXISTS idx_invites_status;
DROP INDEX IF EXISTS idx_invites_email;
DROP INDEX IF EXISTS idx_invites_family_id;
DROP INDEX IF EXISTS idx_invites_token;

-- Drop invites table
DROP TABLE IF EXISTS invites;

-- ==============================================================================
-- Migration 004 Down: Revert budget_alerts trigger schema fix
-- ==============================================================================
-- Note: This migration is no longer needed for SQLite as trigger logic
-- has been moved to Go code in BudgetRepository.
-- This file is kept for migration versioning compatibility.
-- No operations needed

-- ==============================================================================
-- Migration 003 Down: Remove performance optimization indexes
-- ==============================================================================

DROP INDEX IF EXISTS idx_budgets_active_lookup;
DROP INDEX IF EXISTS idx_transactions_budget_calc;
DROP INDEX IF EXISTS idx_transactions_pagination;
DROP INDEX IF EXISTS idx_categories_hierarchy;
DROP INDEX IF EXISTS idx_transactions_summary_calc;
DROP INDEX IF EXISTS idx_transactions_complex_filter;
DROP INDEX IF EXISTS idx_transactions_monthly_summary;

-- ==============================================================================
-- Migration 002 Down: Revert budget trigger fix
-- ==============================================================================
-- Note: This migration is no longer needed for SQLite as trigger logic
-- has been moved to Go code in BudgetRepository.
-- This file is kept for migration versioning compatibility.
-- No operations needed

-- ==============================================================================
-- Migration 001 Down: Drop Initial Schema
-- ==============================================================================

-- Drop triggers
DROP TRIGGER IF EXISTS update_budgets_updated_at;
DROP TRIGGER IF EXISTS update_transactions_updated_at;
DROP TRIGGER IF EXISTS update_categories_updated_at;
DROP TRIGGER IF EXISTS update_users_updated_at;
DROP TRIGGER IF EXISTS update_families_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_user_sessions_expires;
DROP INDEX IF EXISTS idx_user_sessions_user_id;
DROP INDEX IF EXISTS idx_user_sessions_token;

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
