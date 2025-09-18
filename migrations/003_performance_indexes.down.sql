-- Remove performance optimization indexes

DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_monthly_summary;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_complex_filter;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_summary_calc;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_categories_hierarchy;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_tags_gin;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_pagination;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_transactions_budget_calc;
DROP INDEX CONCURRENTLY IF EXISTS family_budget.idx_budgets_active_lookup;