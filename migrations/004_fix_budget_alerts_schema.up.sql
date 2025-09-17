-- Migration 004: Fix budget_alerts trigger schema references

-- Drop existing triggers and functions
DROP TRIGGER IF EXISTS check_budget_alerts_on_update ON family_budget.budgets;
DROP TRIGGER IF EXISTS update_budget_spent_on_transaction ON family_budget.transactions;
DROP FUNCTION IF EXISTS check_budget_alerts();
DROP FUNCTION IF EXISTS update_budget_spent();

-- Recreate budget alerts function with proper schema references
CREATE OR REPLACE FUNCTION family_budget.check_budget_alerts()
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
            SELECT * FROM family_budget.budget_alerts
            WHERE budget_id = NEW.id AND NOT is_triggered
        LOOP
            IF usage_percentage >= alert_record.threshold_percentage THEN
                UPDATE family_budget.budget_alerts
                SET is_triggered = true, triggered_at = NOW()
                WHERE id = alert_record.id;
            END IF;
        END LOOP;
    END IF;

    RETURN NEW;
END;
$$ language 'plpgsql';

-- Recreate budget spent update function with proper schema references
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
    ) WHERE family_id = COALESCE(NEW.family_id, OLD.family_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Recreate triggers
CREATE TRIGGER check_budget_alerts_on_update
    AFTER UPDATE OF spent ON family_budget.budgets
    FOR EACH ROW EXECUTE FUNCTION family_budget.check_budget_alerts();

CREATE TRIGGER update_budget_spent_on_transaction
    AFTER INSERT OR UPDATE OR DELETE ON family_budget.transactions
    FOR EACH ROW EXECUTE FUNCTION family_budget.update_budget_spent();
