# TEST-3: Security тесты

## Статус: TODO
## Приоритет: IMPORTANT

## Проблема

Нет негативных тестов на:
- Path traversal в backup service
- Невалидные/подделанные invite токены
- SQL injection attempts
- Open redirect attempts

## Решение

### 1. Создать `internal/services/backup_service_security_test.go`

```go
package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/services"
)

func TestBackupService_PathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  error
	}{
		{
			name:     "valid filename",
			filename: "backup_20240101_120000000.db",
			wantErr:  nil, // or ErrBackupNotFound if file doesn't exist
		},
		{
			name:     "directory traversal with ../",
			filename: "../../../etc/passwd",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "directory traversal with encoded",
			filename: "..%2F..%2Fetc%2Fpasswd",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "absolute path",
			filename: "/etc/passwd",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "null byte injection",
			filename: "backup_20240101_120000000.db\x00.txt",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "hidden file",
			filename: ".backup_20240101_120000000.db",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "double dots in name",
			filename: "backup..20240101_120000000.db",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "spaces in filename",
			filename: "backup 20240101_120000000.db",
			wantErr:  services.ErrInvalidBackupFilename,
		},
		{
			name:     "sql injection in filename",
			filename: "backup'; DROP TABLE users;--.db",
			wantErr:  services.ErrInvalidBackupFilename,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTestBackupService(t)

			// Test GetBackup
			_, err := svc.GetBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}

			// Test DeleteBackup
			err = svc.DeleteBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}

			// Test RestoreBackup
			err = svc.RestoreBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}

			// Test GetBackupFilePath
			path := svc.GetBackupFilePath(tt.filename)
			if tt.wantErr != nil {
				assert.Empty(t, path)
			}
		})
	}
}
```

### 2. Создать `internal/services/invite_security_test.go`

```go
package services_test

func TestInviteService_InvalidTokens(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"too short token", "abc"},
		{"sql injection", "' OR 1=1 --"},
		{"xss attempt", "<script>alert(1)</script>"},
		{"null bytes", "valid\x00token"},
		{"very long token", strings.Repeat("a", 10000)},
		{"unicode injection", "тест\u0000токен"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := inviteService.GetInviteByToken(context.Background(), tt.token)
			assert.Error(t, err)
		})
	}
}
```

### 3. Создать/дополнить `internal/web/handlers/auth_test.go`

```go
package handlers_test

func TestSanitizeRedirectURL(t *testing.T) {
	// Тесты из SEC-3 task
}

func TestLogin_OpenRedirect(t *testing.T) {
	redirectTests := []struct {
		name        string
		redirectURL string
		expected    string
	}{
		{"protocol relative", "//evil.com", "/"},
		{"absolute url", "https://evil.com", "/"},
		{"javascript", "javascript:alert(1)", "/"},
		{"data uri", "data:text/html,<h1>hi</h1>", "/"},
		{"backslash", "\\\\evil.com", "/"},
		{"valid local", "/dashboard", "/dashboard"},
	}

	for _, tt := range redirectTests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup login request with redirect param
			// Assert redirect goes to expected URL, not attacker URL
		})
	}
}
```

## Файлы для создания

1. `internal/services/backup_service_security_test.go`
2. `internal/services/invite_security_test.go`
3. `internal/web/handlers/auth_security_test.go` (или дополнить существующий auth_test.go)

## Тестирование

- `make test` — все тесты проходят
- `make lint` — 0 issues
