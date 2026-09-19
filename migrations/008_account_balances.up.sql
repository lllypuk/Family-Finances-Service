-- План 20: остаток счёта на конец месяца вместо цифры расходов банка. На свежей базе 001 уже создал таблицу;
-- определение повторяет 001 — схемы живой и свежей базы сравнивает тест. Цифры банка теряются.

CREATE TABLE IF NOT EXISTS account_balances (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    month TEXT NOT NULL,
    balance_minor INTEGER NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (account_id, month),
    CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]')
);

DROP TABLE account_reconciliations;
