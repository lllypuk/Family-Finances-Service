-- Revert to original trigger function without schema qualification
DROP TRIGGER IF EXISTS update_budget_spent_on_transaction ON family_budget.transactions;

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