-- Performance optimization indexes based on benchmark analysis

-- Index for monthly summary queries (GetMonthlySummary)
-- This supports WHERE family_id + date EXTRACT + GROUP BY category/type operations
CREATE INDEX IF NOT EXISTS idx_transactions_monthly_summary
ON family_budget.transactions (family_id, EXTRACT(YEAR FROM date), EXTRACT(MONTH FROM date), category_id, type, amount);

-- Index for complex transaction filtering
-- This supports complex filters with family_id, type, date ranges, and amounts
CREATE INDEX IF NOT EXISTS idx_transactions_complex_filter
ON family_budget.transactions (family_id, type, date, amount)
WHERE type IS NOT NULL;

-- Index for transaction summary calculations
-- This supports WHERE family_id + date range operations with type filtering
CREATE INDEX IF NOT EXISTS idx_transactions_summary_calc
ON family_budget.transactions (family_id, date, type, amount);

-- Index for category hierarchy operations (GetCategoryChildren)
-- This supports recursive CTE queries for parent-child relationships
CREATE INDEX IF NOT EXISTS idx_categories_hierarchy
ON family_budget.categories (family_id, parent_id, id)
WHERE is_active = true;

-- Index for JSONB tag filtering
-- This supports filtering transactions by tags using GIN index
CREATE INDEX IF NOT EXISTS idx_transactions_tags_gin
ON family_budget.transactions USING GIN (tags);

-- Index for pagination operations
-- This supports ORDER BY date DESC with LIMIT/OFFSET
CREATE INDEX IF NOT EXISTS idx_transactions_pagination
ON family_budget.transactions (family_id, date DESC, id);

-- Index for budget trigger performance
-- This supports the budget spent calculation trigger
CREATE INDEX IF NOT EXISTS idx_transactions_budget_calc
ON family_budget.transactions (family_id, category_id, type, date, amount)
WHERE type = 'expense';

-- Partial index for active budgets lookup
CREATE INDEX IF NOT EXISTS idx_budgets_active_lookup
ON family_budget.budgets (family_id, category_id, start_date, end_date)
WHERE is_active = true;

-- Analyze tables after index creation to update statistics
ANALYZE family_budget.transactions;
ANALYZE family_budget.categories;
ANALYZE family_budget.budgets;