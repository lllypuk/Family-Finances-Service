# BUG-003: 403 Forbidden при создании резервной копии

## Приоритет: ВЫСОКИЙ

## Описание проблемы

При попытке создать резервную копию базы данных через админ-панель возвращается ошибка 403 Forbidden.

## Воспроизведение

1. Войти в систему как администратор
2. Перейти на страницу бэкапов (`/admin/backup`)
3. Нажать кнопку "Создать резервную копию"
4. Получить ошибку 403

## Лог сервера

```json
{
  "method": "POST",
  "path": "/admin/backup/create",
  "status": 403,
  "duration": 100343
}
```

## Анализ

Возможные причины 403 ошибки:

1. **CSRF токен** - не передаётся или невалидный
2. **Проблема с сессией** - на странице backup пропадает меню авторизованного пользователя (видно "Вход" и "Регистрация"
   вместо имени)
3. **Проверка роли** - middleware RequireAdmin() не распознаёт пользователя как админа
4. **Cookie** - сессионная кука не передаётся на POST запрос

## Локация бага

- **Endpoint**: `POST /admin/backup/create`
- **Handler**: `internal/web/handlers/backup.go` -> `CreateBackup`
- **Middleware**: `internal/web/middleware/auth.go` -> `RequireAdmin`
- **Routes**: `internal/web/web.go` строки 146-150

## Задачи по исправлению

1. [ ] Проверить, что CSRF токен передаётся в форме бэкапа
2. [ ] Проверить шаблон `internal/web/templates/admin/backup.html` на наличие CSRF
3. [ ] Проверить middleware `RequireAdmin()` - корректно ли проверяется роль
4. [ ] Проверить, что сессия корректно читается на странице backup
5. [ ] Добавить логирование в middleware для диагностики
6. [ ] Написать интеграционный тест на создание бэкапа
7. [ ] Запустить `make lint` и `make test`

## Проверка CSRF в шаблоне

Убедиться, что в форме есть:

```html

<form hx-post="/admin/backup/create" ...>
    <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
    <!-- или -->
    {{ .CSRFField }}
</form>
```

Или HTMX заголовок:

```html

<button hx-post="/admin/backup/create"
        hx-headers='{"X-CSRF-Token": "{{ .CSRFToken }}"}'>
```

## Связанные файлы

- `internal/web/handlers/backup.go`
- `internal/web/templates/admin/backup.html`
- `internal/web/middleware/auth.go`
- `internal/web/middleware/csrf.go`
- `internal/web/web.go`

## Дополнительная проблема

На странице `/admin/backup` пропадает навигация авторизованного пользователя - показывается "Вход" и "Регистрация"
вместо имени пользователя. Это может быть связано с той же проблемой сессии.

## Критерии приёмки

- [ ] Резервные копии создаются успешно
- [ ] Навигация отображается корректно на странице backup
- [ ] CSRF защита работает
- [ ] Добавлен интеграционный тест
- [ ] Все тесты проходят
- [ ] Линтер не выдаёт ошибок
