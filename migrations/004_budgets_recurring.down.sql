-- Откат нужен старому образу: на версии 4 он не стартует (no migration found for version 4).
-- Данные серий теряются, схема возвращается к версии 3 (промежуточной: v0.2.0 — это версия 2,
-- до неё нужен ещё --to 2).

ALTER TABLE budgets DROP COLUMN series_id;
ALTER TABLE budgets DROP COLUMN recurring;
