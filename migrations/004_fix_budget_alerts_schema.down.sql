-- Migration 004 Down: Revert budget_alerts trigger schema fix

-- Drop the fixed triggers and functions
DROP TRIGGER IF EXISTS check_budget_alerts_on_update ON family_budget.budgets;
DROP TRIGGER IF EXISTS update_budget_spent_on_transaction ON family_budget.transactions;
DROP FUNCTION IF EXISTS family_budget.check_budget_alerts();
DROP FUNCTION IF EXISTS family_budget.update_budget_spent();

-- Recreate original functions without schema prefix (this will cause the schema issue again)
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
    ) WHERE family_id = COALESCE(NEW.family_id, OLD.family_id);

    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Recreate original triggers
CREATE TRIGGER check_budget_alerts_on_update
    AFTER UPDATE OF spent ON budgets
    FOR EACH ROW EXECUTE FUNCTION check_budget_alerts();

CREATE TRIGGER update_budget_spent_on_transaction
    AFTER INSERT OR UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_budget_spent();
