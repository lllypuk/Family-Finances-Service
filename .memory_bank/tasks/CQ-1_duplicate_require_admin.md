# CQ-1: Дублирование requireAdmin

## Статус: DONE ✅
## Приоритет: IMPORTANT

## Проблема

Два почти идентичных метода `requireAdmin`:

**AdminHandler** (`internal/web/handlers/admin.go:36-53`):
```go
func (h *AdminHandler) requireAdmin(c echo.Context) (*user.User, error) {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return nil, errors.New("failed to load user")
	}
	if currentUser.Role != user.RoleAdmin {
		return nil, errors.New("admin access required")
	}
	return currentUser, nil
}
```

**BackupHandler** (`internal/web/handlers/backup.go:28-45`):
```go
func (h *BackupHandler) requireAdmin(c echo.Context) error {
	session, err := middleware.GetSessionData(c)
	// ... identical logic, but returns error instead of (*user.User, error)
}
```

Разница: AdminHandler возвращает `(*user.User, error)`, BackupHandler возвращает только `error`.

## Решение

Вынести в `BaseHandler` два варианта:

### 1. Добавить методы в BaseHandler (`internal/web/handlers/base.go`)

```go
// requireAdmin checks if the current user is an admin and returns the user.
func (h *BaseHandler) requireAdmin(c echo.Context) (*user.User, error) {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return nil, errors.New("unauthorized")
	}

	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return nil, errors.New("failed to load user")
	}

	if currentUser.Role != user.RoleAdmin {
		return nil, errors.New("admin access required")
	}

	return currentUser, nil
}

// requireAdminAccess checks if the current user is an admin (without returning user).
func (h *BaseHandler) requireAdminAccess(c echo.Context) error {
	_, err := h.requireAdmin(c)
	return err
}
```

### 2. Обновить AdminHandler (`internal/web/handlers/admin.go`)

Удалить метод `requireAdmin` (строки 36-53). AdminHandler встраивает `*BaseHandler`, поэтому `h.requireAdmin(c)` будет вызывать метод BaseHandler автоматически.

### 3. Обновить BackupHandler (`internal/web/handlers/backup.go`)

Удалить метод `requireAdmin` (строки 28-45). Заменить вызовы:

```go
// Было:
if err := h.requireAdmin(c); err != nil {

// Стало:
if err := h.requireAdminAccess(c); err != nil {
```

Обновить в методах: `BackupPage`, `CreateBackup`, `DownloadBackup`, `DeleteBackup`, `RestoreBackup`.

## Файлы для изменения

1. `internal/web/handlers/base.go` — добавить `requireAdmin()` и `requireAdminAccess()`
2. `internal/web/handlers/admin.go` — удалить дублирующий `requireAdmin()`
3. `internal/web/handlers/backup.go` — удалить дублирующий `requireAdmin()`, заменить вызовы на `requireAdminAccess()`

## Тестирование

- `make test` — все тесты проходят
- `make lint` — 0 issues
