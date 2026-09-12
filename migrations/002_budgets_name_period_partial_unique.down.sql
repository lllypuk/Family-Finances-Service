-- Возврат к табличному UNIQUE. Откат падает, если в таблице уже есть мягко удалённая строка с именем
-- и периодом живого бюджета, — ровно та ситуация, ради которой UNIQUE и сняли.

CREATE TABLE budgets_rebuilt (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    spent_minor INTEGER NOT NULL DEFAULT 0,
    period TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (family_id, name, start_date, end_date),
    CHECK (period IN ('weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (amount_minor > 0),
    CHECK (spent_minor >= 0),
    CHECK (start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date > start_date),
    CHECK (is_active IN (0, 1))
);

INSERT INTO budgets_rebuilt (id, name, amount_minor, spent_minor, period, start_date, end_date,
                             category_id, family_id, is_active, created_at, updated_at)
SELECT id, name, amount_minor, spent_minor, period, start_date, end_date,
       category_id, family_id, is_active, created_at, updated_at
FROM budgets;

DROP TABLE budgets;
ALTER TABLE budgets_rebuilt RENAME TO budgets;

CREATE INDEX IF NOT EXISTS idx_budgets_family_active ON budgets(family_id, is_active);
CREATE INDEX IF NOT EXISTS idx_budgets_category_id ON budgets(category_id);
CREATE INDEX IF NOT EXISTS idx_budgets_family_period ON budgets(family_id, start_date, end_date);

CREATE TRIGGER IF NOT EXISTS update_budgets_updated_at
AFTER UPDATE ON budgets
BEGIN
    UPDATE budgets SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
