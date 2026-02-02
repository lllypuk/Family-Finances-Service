# TEST-1: Тесты для web handlers

## Статус: TODO
## Приоритет: IMPORTANT

## Проблема

Нет тестового покрытия для новых web-хендлеров:
- `internal/web/handlers/admin.go` — AdminHandler (ListUsers, CreateInvite, RevokeInvite, DeleteUser)
- `internal/web/handlers/backup.go` — BackupHandler (BackupPage, CreateBackup, DownloadBackup, DeleteBackup, RestoreBackup)

## Решение

### Подход: Table-driven тесты с Echo test utilities

Использовать `httptest.NewRecorder` + `echo.New()` для тестирования хендлеров. Мокировать сервисы через интерфейсы.

### 1. Создать файл `internal/web/handlers/admin_test.go`

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name           string
		sessionRole    string
		expectedStatus int
	}{
		{
			name:           "admin can view users",
			sessionRole:    "admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-admin gets redirected",
			sessionRole:    "member",
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:           "no session gets redirected",
			sessionRole:    "",
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup echo context with mock session
			// Setup mock services
			// Call handler
			// Assert status code
		})
	}
}

func TestAdminHandler_CreateInvite(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		role           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid invite creation",
			email:          "test@example.com",
			role:           "member",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "duplicate email",
			email:          "existing@example.com",
			role:           "member",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "already exists",
		},
		{
			name:           "invalid email",
			email:          "not-an-email",
			role:           "member",
			expectedStatus: http.StatusBadRequest,
		},
	}
	// ... implementation
}

func TestAdminHandler_RevokeInvite(t *testing.T) {
	// Test cases: valid revoke, not found, already accepted, unauthorized
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	// Test cases: valid delete, self-delete attempt, not found, unauthorized
}
```

### 2. Создать файл `internal/web/handlers/backup_test.go`

```go
package handlers_test

func TestBackupHandler_BackupPage(t *testing.T) {
	// Test cases: admin access, non-admin redirect, list error
}

func TestBackupHandler_CreateBackup(t *testing.T) {
	// Test cases: success, error, HTMX vs regular request
}

func TestBackupHandler_DownloadBackup(t *testing.T) {
	// Test cases: valid download, not found, invalid filename
}

func TestBackupHandler_DeleteBackup(t *testing.T) {
	// Test cases: success, not found, invalid filename, HTMX vs regular
}

func TestBackupHandler_RestoreBackup(t *testing.T) {
	// Test cases: success, not found, invalid filename, HTMX vs regular
}
```

### 3. Вспомогательные функции для тестов

Создать test helpers для:
- Создания echo.Context с mock session
- Создания mock Services struct
- Настройки CSRF token
- Проверки HTMX-specific поведения

```go
func newTestContext(method, path string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func withSession(c echo.Context, userID uuid.UUID, role user.Role) {
	// Set session data in context
}

func withHTMX(c echo.Context) {
	c.Request().Header.Set("Hx-Request", "true")
}
```

## Файлы для создания

1. `internal/web/handlers/admin_test.go`
2. `internal/web/handlers/backup_test.go`
3. Возможно `internal/web/handlers/testhelpers_test.go` для общих хелперов

## Тестирование

- `make test` — новые тесты проходят
- `make test-coverage` — покрытие увеличилось
- `make lint` — 0 issues
