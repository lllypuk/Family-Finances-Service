-- План 10: /reports снесён — отчёт в приложении стал экраном поверх статистики, а не сохранённой
-- строкой с JSON. На свежей базе 001 таблицу уже не создаёт, и все три шага — no-op.

DROP INDEX IF EXISTS idx_reports_generated_by;
DROP INDEX IF EXISTS idx_reports_family_type;
DROP TABLE IF EXISTS reports;
