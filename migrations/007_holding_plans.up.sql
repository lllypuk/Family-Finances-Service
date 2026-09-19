-- План 18: месячный план позиции. Таблица новая, holdings не пересобирается; на свежей базе 001 уже создал
-- её. Определение повторяет 001 — схемы живой и свежей базы сравнивает тест.

CREATE TABLE IF NOT EXISTS holding_plans (
    holding_id TEXT PRIMARY KEY REFERENCES holdings(id) ON DELETE CASCADE,
    monthly_income_minor INTEGER NOT NULL DEFAULT 0 CHECK (monthly_income_minor >= 0),
    monthly_expense_minor INTEGER NOT NULL DEFAULT 0 CHECK (monthly_expense_minor >= 0),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (monthly_income_minor > 0 OR monthly_expense_minor > 0)
);
