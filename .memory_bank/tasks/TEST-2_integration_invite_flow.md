# TEST-2: Интеграционные тесты для invite flow

## Статус: TODO
## Приоритет: IMPORTANT

## Проблема

Нет end-to-end тестов для полного цикла приглашений:
1. Админ создаёт invite
2. Пользователь получает ссылку с токеном
3. Пользователь регистрируется через invite
4. Invite помечается как accepted
5. Новый пользователь может войти

## Решение

### Создать файл `internal/services/invite_integration_test.go`

```go
//go:build integration

package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

func TestInviteFlow_FullCycle(t *testing.T) {
	// Setup: in-memory SQLite + repos + services
	db := testhelpers.NewTestDB(t)
	// ... setup repos and services

	ctx := context.Background()

	// Step 1: Create family and admin user (via setup)
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	admin, err := familyService.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	// Step 2: Admin creates invite
	createDTO := dto.CreateInviteDTO{
		Email: "newuser@test.com",
		Role:  "member",
	}
	invite, err := inviteService.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)
	assert.Equal(t, user.InviteStatusPending, invite.Status)
	assert.NotEmpty(t, invite.Token)

	// Step 3: Get invite by token
	fetchedInvite, err := inviteService.GetInviteByToken(ctx, invite.Token)
	require.NoError(t, err)
	assert.Equal(t, "newuser@test.com", fetchedInvite.Email)

	// Step 4: Accept invite
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "newuser@test.com",
		Name:     "New User",
		Password: "newpassword123",
	}
	newUser, err := inviteService.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.NoError(t, err)
	assert.Equal(t, "newuser@test.com", newUser.Email)
	assert.Equal(t, user.RoleMember, newUser.Role)

	// Step 5: Verify invite is now accepted
	_, err = inviteService.GetInviteByToken(ctx, invite.Token)
	assert.Error(t, err) // Should fail — invite no longer valid

	// Step 6: Verify user can be found
	foundUser, err := userService.GetUserByEmail(ctx, "newuser@test.com")
	require.NoError(t, err)
	assert.Equal(t, newUser.ID, foundUser.ID)
}

func TestInviteFlow_ExpiredInvite(t *testing.T) {
	// Setup
	// Create invite with past expiration
	// Attempt to accept — should get ErrInviteExpired
}

func TestInviteFlow_RevokedInvite(t *testing.T) {
	// Setup
	// Create invite, then revoke
	// Attempt to accept — should get ErrInviteRevoked
}

func TestInviteFlow_DuplicateEmail(t *testing.T) {
	// Setup with existing user
	// Create invite with same email
	// Should get ErrEmailAlreadyExists
}

func TestInviteFlow_EmailMismatch(t *testing.T) {
	// Create invite for user@a.com
	// Try to accept with user@b.com
	// Should fail with email mismatch error
}

func TestInviteFlow_DoubleAccept(t *testing.T) {
	// Create and accept invite
	// Try to accept again
	// Should get ErrInviteAlreadyUsed
}
```

## Файлы для создания

1. `internal/services/invite_integration_test.go`

## Зависимости

- Работающий `testhelpers.NewTestDB` для in-memory SQLite
- Все репозитории и сервисы доступны для инициализации в тесте

## Тестирование

- `make test-integration`
- `make test`
- `make lint`
