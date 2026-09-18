-- План 17: позиции капитала и снимки их стоимости. Обе таблицы новые, пересборок нет; на свежей базе
-- 001 уже создал всё ниже. Определения повторяют 001 — схемы живой и свежей базы сравнивает тест.

CREATE TABLE IF NOT EXISTS holdings (
    id TEXT PRIMARY KEY,
    family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL,
    side TEXT NOT NULL CHECK (side IN ('asset', 'liability')),
    kind TEXT NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (LENGTH(TRIM(name)) > 0),
    UNIQUE (family_id, name_key)
);

CREATE TABLE IF NOT EXISTS holding_values (
    holding_id TEXT NOT NULL REFERENCES holdings(id) ON DELETE CASCADE,
    date TEXT NOT NULL,
    value_minor INTEGER NOT NULL CHECK (value_minor >= 0),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (holding_id, date),
    CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]')
);

CREATE INDEX IF NOT EXISTS idx_holdings_family_archived ON holdings(family_id, is_archived);

CREATE TRIGGER IF NOT EXISTS update_holdings_updated_at
AFTER UPDATE ON holdings
BEGIN
    UPDATE holdings SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
