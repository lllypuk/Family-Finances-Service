# SEC-3: Open URL Redirect в auth.go

## Статус: TODO
## Приоритет: CRITICAL (CodeQL alert, severity: medium)

## Проблема

В `internal/web/handlers/auth.go:94-113` метод `Login`:

```go
redirectTo := c.QueryParam("redirect")
if redirectTo == "" {
	redirectTo = "/"
} else {
	redirectTo = strings.ReplaceAll(redirectTo, "\\", "/")
	parsed, parsErr := url.Parse(redirectTo)
	if parsErr != nil || parsed.IsAbs() || parsed.Host != "" {
		redirectTo = "/"
	}
}
// ...
return c.Redirect(http.StatusFound, redirectTo)
```

CodeQL alert: `go/unvalidated-url-redirection` на строке 113.

Текущая валидация уже проверяет `parsed.IsAbs()` и `parsed.Host != ""`, но CodeQL считает это недостаточным. Проблемы:
1. Protocol-relative URL `//evil.com` — `url.Parse("//evil.com")` даёт `Host: "evil.com"`, IsAbs: false — текущий код должен ловить это через `parsed.Host != ""`, но CodeQL не доверяет этой проверке
2. URL с нестандартными схемами или кодированием

## Решение

Усилить валидацию с явной проверкой на `/` и запрет `//`:

```go
redirectTo := c.QueryParam("redirect")
redirectTo = sanitizeRedirectURL(redirectTo)
```

Добавить функцию `sanitizeRedirectURL`:

```go
// sanitizeRedirectURL validates and sanitizes a redirect URL to prevent open redirect attacks.
// Only relative paths starting with "/" are allowed. Protocol-relative URLs and absolute URLs are rejected.
func sanitizeRedirectURL(rawURL string) string {
	if rawURL == "" {
		return "/"
	}

	// Normalize backslashes
	rawURL = strings.ReplaceAll(rawURL, "\\", "/")

	// Must start with exactly one slash (reject "//evil.com" and "https://evil.com")
	if !strings.HasPrefix(rawURL, "/") || strings.HasPrefix(rawURL, "//") {
		return "/"
	}

	// Parse to catch any remaining edge cases
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host != "" || parsed.Scheme != "" {
		return "/"
	}

	// Use only the path + query (strip any fragment or userinfo)
	result := parsed.Path
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}

	if result == "" {
		return "/"
	}

	return result
}
```

### Обновить метод Login

```go
func (h *AuthHandler) Login(c echo.Context) error {
	// ... form binding and validation unchanged ...

	// Определяем куда перенаправить после входа
	redirectTo := sanitizeRedirectURL(c.QueryParam("redirect"))

	// Если это HTMX запрос, возвращаем redirect header
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", redirectTo)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, redirectTo)
}
```

Удалить старый блок кода (строки 94-105).

## Файлы для изменения

1. `internal/web/handlers/auth.go` — добавить `sanitizeRedirectURL()`, обновить `Login()`

## Тестирование

Добавить unit-тесты для `sanitizeRedirectURL`:

```go
func TestSanitizeRedirectURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"valid path", "/dashboard", "/dashboard"},
		{"valid path with query", "/page?id=1", "/page?id=1"},
		{"protocol relative", "//evil.com", "/"},
		{"absolute http", "http://evil.com", "/"},
		{"absolute https", "https://evil.com/path", "/"},
		{"backslash", "\\evil.com", "/"},
		{"double backslash", "\\\\evil.com", "/"},
		{"no leading slash", "evil.com", "/"},
		{"javascript scheme", "javascript:alert(1)", "/"},
		{"data scheme", "data:text/html,<h1>hi</h1>", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRedirectURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

- `make test`
- `make lint`
