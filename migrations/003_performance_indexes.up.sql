-- Migration 003: Performance optimization indexes (SQLite)

-- Index for monthly summary queries (GetMonthlySummary)
-- SQLite doesn't support function-based indexes like PostgreSQL's EXTRACT
-- So we create a simpler index that SQLite can use effectively
CREATE INDEX IF NOT EXISTS idx_transactions_monthly_summary
ON transactions (family_id, date, category_id, type, amount);

-- Index for complex transaction filtering
-- Partial indexes in SQLite use WHERE clause
CREATE INDEX IF NOT EXISTS idx_transactions_complex_filter
ON transactions (family_id, type, date, amount)
WHERE type IS NOT NULL;

-- Index for transaction summary calculations
CREATE INDEX IF NOT EXISTS idx_transactions_summary_calc
ON transactions (family_id, date, type, amount);

-- Index for category hierarchy operations (GetCategoryChildren)
-- SQLite supports recursive CTEs with WITH RECURSIVE
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

-- SQLite doesn't support ANALYZE with table names in the same way as PostgreSQL
-- Instead, we run ANALYZE without arguments to update statistics for all tables
ANALYZE;
