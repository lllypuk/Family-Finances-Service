# SEC-3: Open URL Redirect в auth.go

## Статус: DONE ✅
## Приоритет: CRITICAL (CodeQL alert, severity: medium)
## Дата завершения: 2026-02-03

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

- ✅ `make test` — все тесты проходят
- ✅ `make lint` — 0 issues
- ✅ Добавлен тест `TestSanitizeRedirectURL` с 24 сценариями
- ⏳ CodeQL alert должен исчезнуть в следующем CI run

## Реализованные изменения

### 1. Добавлена функция `sanitizeRedirectURL()`
- Валидация пустого URL — возвращает "/"
- Нормализация backslashes в forward slashes
- Проверка, что URL начинается с `/` (отклоняет относительные без слеша)
- Проверка, что URL НЕ начинается с `//` (отклоняет protocol-relative)
- Парсинг через `url.Parse()` с проверкой Host и Scheme
- Использование только Path + RawQuery (удаление Fragment и Userinfo)

### 2. Обновлен метод `Login()`
- Удален старый блок валидации (строки 94-105)
- Вызов `sanitizeRedirectURL()` для валидации redirect параметра
- Упрощенный код — меньше дублирования логики

### 3. Тестовое покрытие
- Создан новый файл `internal/web/handlers/auth_test.go`
- Тест `TestSanitizeRedirectURL` с 24 сценариями:
  - Валидные пути (root, nested, с query params)
  - Protocol-relative URLs (`//evil.com`)
  - Absolute URLs (http, https, ftp, mailto, data, javascript)
  - Backslash normalization
  - Fragment и query обработка
  - Newline injection
  - Triple slash (`///evil.com`)

### 4. Защита от атак
- ✅ Open redirect через `//evil.com`
- ✅ Absolute URLs с любыми схемами
- ✅ JavaScript injection (`javascript:alert(1)`)
- ✅ Data URLs
- ✅ Newline injection в заголовках
- ✅ Backslash обход (`\\evil.com`)

### 5. Defense in depth
- Множественные проверки: prefix, parse, Host, Scheme
- Нормализация входных данных
- Явное использование только безопасных компонентов URL (Path + Query)
- Удаление Fragment для предотвращения XSS
