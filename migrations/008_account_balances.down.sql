-- Откат нужен старому образу: на версии 8 он не стартует. Теряются все остатки; account_reconciliations
-- возвращается пустой, в виде из 005.

DROP TABLE account_balances;

CREATE TABLE account_reconciliations (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    month TEXT NOT NULL,
    bank_expense_minor INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (account_id, month),
    CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
    CHECK (bank_expense_minor >= 0)
);
