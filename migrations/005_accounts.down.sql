-- Откат нужен старому образу: на версии 5 он не стартует. Теряются счета, сверки и привязка операций
-- к счетам; сами операции остаются.

DROP TABLE account_reconciliations;
DROP TRIGGER update_accounts_updated_at;
-- DROP COLUMN отказывает на индексированной колонке — индекс снимается первым.
DROP INDEX idx_transactions_account_date;
ALTER TABLE transactions DROP COLUMN account_id;
DROP TABLE accounts;
