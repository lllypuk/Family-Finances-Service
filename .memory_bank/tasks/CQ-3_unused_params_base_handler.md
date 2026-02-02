# CQ-3: Неиспользуемые параметры в BaseHandler

## Статус: DONE ✅
## Приоритет: IMPORTANT

## Проблема

В `internal/web/handlers/base.go`:

```go
// строка 166
func (h *BaseHandler) redirectWithError(c echo.Context, url, _ string) error {
// TODO: Add flash message support for error messages
return c.Redirect(http.StatusSeeOther, url)
}

// строка 173
func (h *BaseHandler) redirectWithSuccess(c echo.Context, url, _ string) error {
// TODO: Add flash message support for success messages
return c.Redirect(http.StatusSeeOther, url)
}

```

Параметр `message` принимается, но игнорируется (`_`). Это вводит в заблуждение — вызывающий код передаёт сообщения, которые никуда не попадают.

## Решение

### Вариант A: Реализовать flash messages (рекомендуется)

Использовать cookie-based flash messages (простое решение без дополнительных зависимостей):

```go
const flashCookieName = "flash_message"
const flashTypeCookieName = "flash_type"

// setFlashMessage sets a flash message in a cookie
func setFlashMessage(c echo.Context, msgType, message string) {
	c.SetCookie(&http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   10, // 10 seconds
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     flashTypeCookieName,
		Value:    msgType,
		Path:     "/",
		MaxAge:   10,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetFlashMessage reads and clears the flash message
func GetFlashMessage(c echo.Context) (string, string) {
	msgCookie, err := c.Cookie(flashCookieName)
	if err != nil {
		return "", ""
	}
	typeCookie, _ := c.Cookie(flashTypeCookieName)

	// Clear cookies
	c.SetCookie(&http.Cookie{Name: flashCookieName, Path: "/", MaxAge: -1})
	c.SetCookie(&http.Cookie{Name: flashTypeCookieName, Path: "/", MaxAge: -1})

	message, _ := url.QueryUnescape(msgCookie.Value)
	msgType := "info"
	if typeCookie != nil {
		msgType = typeCookie.Value
	}

	return msgType, message
}

func (h *BaseHandler) redirectWithError(c echo.Context, redirectURL, message string) error {
	setFlashMessage(c, "error", message)
	return c.Redirect(http.StatusSeeOther, redirectURL)
}

func (h *BaseHandler) redirectWithSuccess(c echo.Context, redirectURL, message string) error {
	setFlashMessage(c, "success", message)
	return c.Redirect(http.StatusSeeOther, redirectURL)
}
```

Затем в шаблонах добавить чтение flash message (через middleware или в data каждой страницы).

### Вариант B: Удалить неиспользуемый параметр (проще)

Если flash messages пока не нужны, убрать параметр message:

```go
func (h *BaseHandler) redirectWithError(c echo.Context, url string) error {
	return c.Redirect(http.StatusSeeOther, url)
}

func (h *BaseHandler) redirectWithSuccess(c echo.Context, url string) error {
	return c.Redirect(http.StatusSeeOther, url)
}
```

И обновить все вызовы (убрать третий аргумент):
- `admin.go` — `h.redirectWithError(c, "/")`
- `backup.go` — `h.redirectWithError(c, "/admin/backup")`, `h.redirectWithSuccess(c, "/admin/backup")`

## Файлы для изменения

### Вариант A:
1. `internal/web/handlers/base.go` — реализовать flash messages
2. Все шаблоны/middleware — добавить чтение flash messages

### Вариант B:
1. `internal/web/handlers/base.go` — убрать параметр message
2. `internal/web/handlers/admin.go` — обновить вызовы
3. `internal/web/handlers/backup.go` — обновить вызовы

## Тестирование

- `make test`
- `make lint`
