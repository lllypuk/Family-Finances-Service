-- План 12: периодические бюджеты. Хвост серии (recurring = 1) сервер продлевает при чтении,
-- series_id связывает инстансы одной серии. FK на series_id нет: первый инстанс мягко удаляется.
-- 002 пересобирает budgets по схеме v0.2.0, то есть без этих колонок, — на свежей базе 004 не
-- дублирует то, что создала 001, а возвращает колонки после пересборки.

ALTER TABLE budgets ADD COLUMN recurring INTEGER NOT NULL DEFAULT 0 CHECK (recurring IN (0, 1));
ALTER TABLE budgets ADD COLUMN series_id TEXT;
