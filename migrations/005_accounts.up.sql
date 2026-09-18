-- План 16: счета и сверки. account_id в transactions добавляется пересборкой, а не ADD COLUMN, чтобы
-- колонка стояла там же, где в 001 (после family_id): схема живой и свежей базы совпадает по cid.
-- Определение transactions ниже заморожено на схеме этой версии; колонка, добавленная позже, живёт
-- в 001 и в своей NNN — как с budgets в 002. На свежей базе всё это пересоздаёт уже готовое.
-- Своего BEGIN нет: golang-migrate выполняет файл в своей транзакции.

CREATE TABLE IF NOT EXISTS accounts (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (LENGTH(TRIM(name)) > 0),
    UNIQUE (family_id, name_key)
);

CREATE TABLE IF NOT EXISTS account_reconciliations (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    month TEXT NOT NULL,
    bank_expense_minor INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (account_id, month),
    CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
    CHECK (bank_expense_minor >= 0)
);

CREATE TRIGGER IF NOT EXISTS update_accounts_updated_at
AFTER UPDATE ON accounts
BEGIN
    UPDATE accounts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TABLE transactions_rebuilt (
    id TEXT PRIMARY KEY,
    amount_minor INTEGER NOT NULL,
    description TEXT NOT NULL,
    date TEXT NOT NULL,
    type TEXT NOT NULL,
    category_id TEXT NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    account_id TEXT REFERENCES accounts(id) ON DELETE RESTRICT,
    tags TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (type IN ('income', 'expense')),
    CHECK (amount_minor > 0),
    CHECK (LENGTH(TRIM(description)) > 0),
    CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')
);

-- account_id не копируется: на версии 4 колонки нет, а на свежей базе таблица пуста.
INSERT INTO transactions_rebuilt (id, amount_minor, description, date, type, category_id, user_id,
                                  family_id, account_id, tags, created_at, updated_at)
SELECT id, amount_minor, description, date, type, category_id, user_id,
       family_id, NULL, tags, created_at, updated_at
FROM transactions;

-- Вместе с таблицей уходят её индексы и триггер updated_at — ниже они создаются заново.
DROP TABLE transactions;
ALTER TABLE transactions_rebuilt RENAME TO transactions;

CREATE INDEX IF NOT EXISTS idx_transactions_family_date ON transactions(family_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_family_type_date ON transactions(family_id, type, date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_account_date ON transactions(account_id, date);

CREATE TRIGGER IF NOT EXISTS update_transactions_updated_at
AFTER UPDATE ON transactions
BEGIN
    UPDATE transactions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
