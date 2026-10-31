-- Сессии живут до отзыва. Истёкшие строки ждали логина: без DELETE они бы ожили.
DELETE FROM sessions WHERE expires_at < strftime('%Y-%m-%dT%H:%M:%SZ', 'now');
UPDATE sessions SET expires_at = '9999-12-31T23:59:59Z';
