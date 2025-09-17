-- Fix trigger function to use qualified schema names
CREATE OR REPLACE FUNCTION family_budget.update_budget_spent()
RETURNS TRIGGER AS $$
BEGIN
    -- Update spent amount for all budgets that include this transaction's category
    UPDATE family_budget.budgets SET spent = (
        SELECT COALESCE(SUM(t.amount), 0)
        FROM family_budget.transactions t
        WHERE t.type = 'expense'
        AND t.family_id = family_budget.budgets.family_id
        AND t.date BETWEEN family_budget.budgets.start_date AND family_budget.budgets.end_date
        AND (family_budget.budgets.category_id IS NULL OR t.category_id = family_budget.budgets.category_id)
    )
    WHERE family_id = COALESCE(NEW.family_id, OLD.family_id)
    AND (category_id IS NULL OR category_id = COALESCE(NEW.category_id, OLD.category_id));

    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Re-create the trigger
DROP TRIGGER IF EXISTS update_budget_spent_on_transaction ON family_budget.transactions;
CREATE TRIGGER update_budget_spent_on_transaction
    AFTER INSERT OR UPDATE OR DELETE ON family_budget.transactions
    FOR EACH ROW EXECUTE FUNCTION family_budget.update_budget_spent();