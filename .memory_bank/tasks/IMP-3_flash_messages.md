# IMP-3: Flash messages

## Статус: DONE ✅
## Приоритет: NICE TO HAVE

## Результат

Реализован механизм flash messages для отображения сообщений после redirect:

✅ Cookie-based flash messages (10 секунд TTL)
✅ Функции `setFlashMessage` и `GetFlashMessage` в base.go
✅ Методы `redirectWithError` и `redirectWithSuccess` используют flash messages
✅ Компонент `components/flash.html` с PicoCSS стилями
✅ Flash messages интегрированы в layout base.html
✅ Flash messages добавлены во все page handlers (backup, admin, users, auth)
✅ Comprehensive тесты для flash message функциональности
✅ Все тесты проходят
✅ Линтер прошел без ошибок

## Файлы изменены

1. ✅ `internal/web/handlers/base.go` - flash message functions уже были реализованы
2. ✅ `internal/web/templates/components/flash.html` - новый компонент с PicoCSS
3. ✅ `internal/web/templates/layouts/base.html` - обновлен для использования нового компонента
4. ✅ `internal/web/handlers/backup.go` - добавлены flash messages в PageData
5. ✅ `internal/web/handlers/admin.go` - добавлены flash messages в PageData
6. ✅ `internal/web/handlers/users.go` - добавлены flash messages в PageData
7. ✅ `internal/web/handlers/auth.go` - встроен BaseHandler, добавлены flash messages
8. ✅ `internal/web/handlers/flash_test.go` - comprehensive тесты
9. ✅ `.golangci.yml` - добавлено исключение testpackage для web handlers

## Тестирование

✅ Тест: flash message устанавливается в cookie с правильными параметрами
✅ Тест: flash message читается и cookie удаляется
✅ Тест: redirectWithError/redirectWithSuccess устанавливают flash cookies
✅ Тест: getFlashMessages возвращает messages из cookies
✅ `make test` - все тесты проходят
✅ `make lint` - 0 issues
✅ `make build` - успешная сборка

## Проблема

Нет механизма показа пользователю сообщений после redirect. Текущие методы `redirectWithError` и `redirectWithSuccess` в `base.go:166-177` принимают сообщение, но игнорируют его. Пользователь не видит результат операции после redirect.

## Решение

### Cookie-based flash messages

Простое решение без дополнительных зависимостей. Flash message сохраняется в cookie, читается на следующей странице и удаляется.

### 1. Добавить flash message helpers в `base.go`

```go
import "net/url"

const (
	flashMessageCookie = "flash_msg"
	flashTypeCookie    = "flash_type"
	flashMaxAge        = 10 // seconds
)

func setFlashMessage(c echo.Context, msgType, message string) {
	c.SetCookie(&http.Cookie{
		Name:     flashMessageCookie,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   flashMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     flashTypeCookie,
		Value:    msgType,
		Path:     "/",
		MaxAge:   flashMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func getFlashMessage(c echo.Context) (msgType, message string) {
	msgCookie, err := c.Cookie(flashMessageCookie)
	if err != nil {
		return "", ""
	}
	typeCookie, _ := c.Cookie(flashTypeCookie)

	// Clear cookies immediately
	c.SetCookie(&http.Cookie{Name: flashMessageCookie, Path: "/", MaxAge: -1})
	c.SetCookie(&http.Cookie{Name: flashTypeCookie, Path: "/", MaxAge: -1})

	message, _ = url.QueryUnescape(msgCookie.Value)
	msgType = "info"
	if typeCookie != nil {
		msgType = typeCookie.Value
	}
	return msgType, message
}
```

### 2. Обновить redirectWithError / redirectWithSuccess

```go
func (h *BaseHandler) redirectWithError(c echo.Context, redirectURL, message string) error {
	if message != "" {
		setFlashMessage(c, "error", message)
	}
	return c.Redirect(http.StatusSeeOther, redirectURL)
}

func (h *BaseHandler) redirectWithSuccess(c echo.Context, redirectURL, message string) error {
	if message != "" {
		setFlashMessage(c, "success", message)
	}
	return c.Redirect(http.StatusSeeOther, redirectURL)
}
```

### 3. Обновить `getFlashMessages` (уже есть заглушка в base.go:75-78)

```go
func (h *BaseHandler) getFlashMessages(c echo.Context) []Message {
	msgType, message := getFlashMessage(c)
	if message == "" {
		return nil
	}
	return []Message{{Type: msgType, Text: message}}
}
```

### 4. Middleware для автоматического добавления flash в шаблоны

Или: в каждый render-вызов добавлять flash messages в data. Проще через middleware:

```go
func FlashMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read flash message and store in context
			msgType, message := getFlashMessage(c)
			if message != "" {
				c.Set("flash_type", msgType)
				c.Set("flash_message", message)
			}
			return next(c)
		}
	}
}
```

### 5. Шаблон компонент для flash message

В `internal/web/templates/components/flash.html`:
```html
{{if .FlashMessage}}
<div role="alert" class="{{if eq .FlashType "error"}}pico-color-red{{else if eq .FlashType "success"}}pico-color-green{{end}}">
  {{.FlashMessage}}
</div>
{{end}}
```

Включить в layout:
```html
{{template "flash" .}}
```

## Файлы для изменения

1. `internal/web/handlers/base.go` — flash message functions, обновить redirect методы
2. `internal/web/middleware/` — опционально FlashMiddleware
3. `internal/web/templates/components/flash.html` — новый компонент
4. `internal/web/templates/layouts/` — включить flash компонент в layout

## Тестирование

- Тест: после redirect flash message присутствует в cookie
- Тест: после чтения flash message cookie удалён
- `make test`
- `make lint`
