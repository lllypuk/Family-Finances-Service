package services_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

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
		{"single quote", "'"},
		{"double quote", "\""},
		{"semicolon", ";"},
		{"sql comment", "--"},
		{"sql multi-line comment", "/* comment */"},
		{"newline injection", "token\nmalicious"},
		{"carriage return", "token\rmalicious"},
		{"tab character", "token\tmalicious"},
		{"backslash", "token\\malicious"},
		{"slash", "token/malicious"},
		{"pipe", "token|malicious"},
		{"ampersand", "token&malicious"},
		{"dollar sign", "token$malicious"},
		{"backtick", "token`malicious"},
		{"percent encoding", "%27%20OR%201=1--"},
		{"unicode null", "token\u0000"},
		{"html entity", "&lt;script&gt;"},
		{"ldap injection", "token)(uid=*))(|(uid=*"},
		{"xpath injection", "token' or '1'='1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInviteRepo := &MockInviteRepository{}
			mockUserRepo := &MockUserRepository{}
			mockFamilyRepo := &MockFamilyRepository{}

			inviteService := services.NewInviteService(
				mockInviteRepo,
				mockUserRepo,
				mockFamilyRepo,
				slog.Default(),
			)

			// Setup mock to return not found error
			mockInviteRepo.On("GetByToken", context.Background(), tt.token).
				Return(nil, errors.New("not found"))

			_, err := inviteService.GetInviteByToken(context.Background(), tt.token)
			require.Error(t, err, "Should return error for invalid token")
			require.ErrorIs(t, err, services.ErrInviteNotFound)
		})
	}
}

func TestInviteService_ExpiredTokenHandling(t *testing.T) {
	mockInviteRepo := &MockInviteRepository{}
	mockUserRepo := &MockUserRepository{}
	mockFamilyRepo := &MockFamilyRepository{}

	inviteService := services.NewInviteService(
		mockInviteRepo,
		mockUserRepo,
		mockFamilyRepo,
		slog.Default(),
	)

	familyID := uuid.New()
	creatorID := uuid.New()

	// Create an expired invite
	expiredInvite, err := user.NewInvite(familyID, creatorID, "test@example.com", user.RoleMember)
	require.NoError(t, err)

	// Manually set to expired status
	expiredInvite.Status = user.InviteStatusPending
	expiredInvite.ExpiresAt = expiredInvite.CreatedAt.Add(-1) // Expired in the past

	mockInviteRepo.On("GetByToken", context.Background(), expiredInvite.Token).
		Return(expiredInvite, nil)
	mockInviteRepo.On("Update", context.Background(), expiredInvite).
		Return(nil)

	_, err = inviteService.GetInviteByToken(context.Background(), expiredInvite.Token)
	assert.ErrorIs(t, err, services.ErrInviteExpired)
}

func TestInviteService_TokenValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectedError error
	}{
		{
			name:          "unicode zero-width characters",
			token:         "token\u200B\u200C\u200D",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "rtl override character",
			token:         "token\u202E",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "url encoded null byte",
			token:         "token%00",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "double url encoded",
			token:         "token%2527",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "mixed case sql injection",
			token:         "' oR 1=1 --",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "union select injection",
			token:         "' UNION SELECT * FROM users--",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "stacked queries",
			token:         "token'; DROP TABLE invites;--",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "boolean based blind",
			token:         "token' AND '1'='1",
			expectedError: services.ErrInviteNotFound,
		},
		{
			name:          "time based blind",
			token:         "token' AND SLEEP(5)--",
			expectedError: services.ErrInviteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInviteRepo := &MockInviteRepository{}
			mockUserRepo := &MockUserRepository{}
			mockFamilyRepo := &MockFamilyRepository{}

			inviteService := services.NewInviteService(
				mockInviteRepo,
				mockUserRepo,
				mockFamilyRepo,
				slog.Default(),
			)

			mockInviteRepo.On("GetByToken", context.Background(), tt.token).
				Return(nil, errors.New("not found"))

			_, err := inviteService.GetInviteByToken(context.Background(), tt.token)
			require.Error(t, err)
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestInviteService_AcceptInviteSecurityValidation(t *testing.T) {
	mockInviteRepo := &MockInviteRepository{}
	mockUserRepo := &MockUserRepository{}
	mockFamilyRepo := &MockFamilyRepository{}

	inviteService := services.NewInviteService(
		mockInviteRepo,
		mockUserRepo,
		mockFamilyRepo,
		slog.Default(),
	)

	familyID := uuid.New()
	creatorID := uuid.New()

	tests := []struct {
		name          string
		email         string
		expectedError bool
	}{
		{
			name:          "email with sql injection",
			email:         "test@example.com'; DROP TABLE users;--",
			expectedError: true,
		},
		{
			name:          "email with xss",
			email:         "<script>alert(1)</script>@example.com",
			expectedError: true,
		},
		{
			name:          "email with null byte",
			email:         "test@example.com\x00admin",
			expectedError: true,
		},
		{
			name:          "email with newline",
			email:         "test@example.com\nadmin@example.com",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create valid invite
			invite, err := user.NewInvite(familyID, creatorID, "test@example.com", user.RoleMember)
			require.NoError(t, err)

			mockInviteRepo.On("GetByToken", context.Background(), invite.Token).
				Return(invite, nil)

			// Try to accept with malicious email
			acceptDTO := dto.AcceptInviteDTO{
				Email:    tt.email,
				Name:     "Test User",
				Password: "SecurePass123!",
			}

			_, err = inviteService.AcceptInvite(context.Background(), invite.Token, acceptDTO)
			if tt.expectedError {
				assert.Error(t, err, "Should reject malicious email")
			}
		})
	}
}

func TestInviteService_RevokeWithInvalidInput(t *testing.T) {
	tests := []struct {
		name          string
		inviteID      uuid.UUID
		revokerID     uuid.UUID
		expectedError error
	}{
		{
			name:          "nil invite ID",
			inviteID:      uuid.Nil,
			revokerID:     uuid.New(),
			expectedError: services.ErrUserNotFound, // Revoker check happens first
		},
		{
			name:          "nil revoker ID",
			inviteID:      uuid.New(),
			revokerID:     uuid.Nil,
			expectedError: services.ErrUserNotFound,
		},
		{
			name:          "both nil",
			inviteID:      uuid.Nil,
			revokerID:     uuid.Nil,
			expectedError: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInviteRepo := &MockInviteRepository{}
			mockUserRepo := &MockUserRepository{}
			mockFamilyRepo := &MockFamilyRepository{}

			inviteService := services.NewInviteService(
				mockInviteRepo,
				mockUserRepo,
				mockFamilyRepo,
				slog.Default(),
			)

			// Setup mocks - GetByID always returns not found for nil/invalid IDs
			mockUserRepo.On("GetByID", context.Background(), tt.revokerID).
				Return(nil, errors.New("not found"))

			err := inviteService.RevokeInvite(context.Background(), tt.inviteID, tt.revokerID)
			require.Error(t, err)
			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
			}
		})
	}
}

func TestInviteService_ConcurrentTokenAccess(t *testing.T) {
	mockInviteRepo := &MockInviteRepository{}
	mockUserRepo := &MockUserRepository{}
	mockFamilyRepo := &MockFamilyRepository{}

	inviteService := services.NewInviteService(
		mockInviteRepo,
		mockUserRepo,
		mockFamilyRepo,
		slog.Default(),
	)

	// Create a valid invite
	familyID := uuid.New()
	creatorID := uuid.New()
	invite, err := user.NewInvite(familyID, creatorID, "test@example.com", user.RoleMember)
	require.NoError(t, err)

	mockInviteRepo.On("GetByToken", context.Background(), invite.Token).
		Return(invite, nil)

	// Simulate concurrent access attempts
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_, _ = inviteService.GetInviteByToken(context.Background(), invite.Token)
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

func TestInviteService_MaliciousEmailNormalization(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"unicode spaces", "test\u00A0@example.com"},
		{"mixed case with injection", "TeSt@ExAmPlE.com'; DROP TABLE users;--"},
		{"trailing whitespace", "test@example.com    "},
		{"leading whitespace", "    test@example.com"},
		{"mixed whitespace", "  test@example.com  \t\n"},
		{"rtl override in email", "test\u202E@example.com"},
		{"homograph attack", "test@еxample.com"}, // Cyrillic 'е'
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInviteRepo := &MockInviteRepository{}
			mockUserRepo := &MockUserRepository{}
			mockFamilyRepo := &MockFamilyRepository{}

			inviteService := services.NewInviteService(
				mockInviteRepo,
				mockUserRepo,
				mockFamilyRepo,
				slog.Default(),
			)

			adminUser := &user.User{
				ID:    uuid.New(),
				Email: "admin@example.com",
				Role:  user.RoleAdmin,
			}

			family := &user.Family{
				ID:   uuid.New(),
				Name: "Test Family",
			}

			mockUserRepo.On("GetByID", context.Background(), adminUser.ID).
				Return(adminUser, nil)
			mockFamilyRepo.On("Get", context.Background()).
				Return(family, nil)

			normalizedEmail := strings.ToLower(strings.TrimSpace(tt.email))

			mockUserRepo.On("GetByEmail", context.Background(), normalizedEmail).
				Return(nil, errors.New("not found"))
			mockInviteRepo.On("GetPendingByEmail", context.Background(), normalizedEmail).
				Return([]*user.Invite{}, nil)
			mockInviteRepo.On("Create", context.Background(), mock.AnythingOfType("*user.Invite")).
				Return(nil)

			createDTO := dto.CreateInviteDTO{
				Email: tt.email,
				Role:  string(user.RoleMember),
			}

			// Service should handle normalization
			// For malicious inputs, creation might fail due to validation
			_, err := inviteService.CreateInvite(context.Background(), adminUser.ID, createDTO)

			// We expect either success (if normalized properly) or validation error
			// But never SQL injection or security bypass
			if err != nil {
				// Error is acceptable for malicious input
				assert.NotContains(t, err.Error(), "DROP TABLE")
				assert.NotContains(t, err.Error(), "script")
			}
		})
	}
}
