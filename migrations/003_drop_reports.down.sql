-- Таблица и индексы дословно из 001 версии 2 — данные не возвращаются, только схема:
-- откат нужен старому образу, который на версии 3 не стартует (no migration found for version 3).

CREATE TABLE IF NOT EXISTS reports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    period TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    data TEXT NOT NULL,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    generated_by TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (type IN ('expenses', 'income', 'budget', 'cash_flow', 'category_breakdown')),
    CHECK (period IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    CHECK (LENGTH(TRIM(name)) > 0),
    CHECK (start_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_reports_family_type ON reports(family_id, type);
CREATE INDEX IF NOT EXISTS idx_reports_generated_by ON reports(generated_by);
