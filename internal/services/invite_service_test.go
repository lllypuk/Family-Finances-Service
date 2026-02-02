package services_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// MockInviteRepository is a mock implementation of InviteRepository
type MockInviteRepository struct {
	mock.Mock
}

func (m *MockInviteRepository) Create(invite *user.Invite) error {
	args := m.Called(invite)
	return args.Error(0)
}

func (m *MockInviteRepository) GetByToken(token string) (*user.Invite, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Invite), args.Error(1)
}

func (m *MockInviteRepository) GetByID(id uuid.UUID) (*user.Invite, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Invite), args.Error(1)
}

func (m *MockInviteRepository) GetByFamily(familyID uuid.UUID) ([]*user.Invite, error) {
	args := m.Called(familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.Invite), args.Error(1)
}

func (m *MockInviteRepository) GetPendingByEmail(email string) ([]*user.Invite, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.Invite), args.Error(1)
}

func (m *MockInviteRepository) Update(invite *user.Invite) error {
	args := m.Called(invite)
	return args.Error(0)
}

func (m *MockInviteRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockInviteRepository) DeleteExpired() error {
	args := m.Called()
	return args.Error(0)
}

func TestInviteService_CreateInvite(t *testing.T) {
	familyID := uuid.New()
	creatorID := uuid.New()
	family := &user.Family{ID: familyID, Name: "Test Family"}
	creator := &user.User{
		ID:        creatorID,
		Email:     "admin@example.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      user.RoleAdmin,
	}

	tests := []struct {
		name      string
		creatorID uuid.UUID
		dto       dto.CreateInviteDTO
		setup     func(*MockInviteRepository, *MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name:      "Success - Create valid invite",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "newuser@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				// Creator is admin
				ur.On("GetByID", mock.Anything, creatorID).Return(creator, nil)
				// Family exists
				fr.On("Get", mock.Anything).Return(family, nil)
				// User doesn't exist
				ur.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, errors.New("not found"))
				// No pending invites
				ir.On("GetPendingByEmail", "newuser@example.com").Return([]*user.Invite{}, nil)
				// Create succeeds
				ir.On("Create", mock.AnythingOfType("*user.Invite")).Return(nil)
			},
			wantError: false,
		},
		{
			name:      "Error - Creator not found",
			creatorID: uuid.New(),
			dto: dto.CreateInviteDTO{
				Email: "newuser@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(_ *MockInviteRepository, ur *MockUserRepository, _ *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
		{
			name:      "Error - Creator is not admin",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "newuser@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(_ *MockInviteRepository, ur *MockUserRepository, _ *MockFamilyRepository) {
				nonAdmin := &user.User{
					ID:        creatorID,
					Email:     "member@example.com",
					FirstName: "Member",
					LastName:  "User",
					Role:      user.RoleMember,
				}
				ur.On("GetByID", mock.Anything, creatorID).Return(nonAdmin, nil)
			},
			wantError: true,
			errorType: services.ErrUnauthorized,
		},
		{
			name:      "Error - Family not found",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "newuser@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(_ *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, creatorID).Return(creator, nil)
				fr.On("Get", mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrFamilyNotFound,
		},
		{
			name:      "Error - Email already exists",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "existing@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(_ *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, creatorID).Return(creator, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				existingUser := &user.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
					Role:  user.RoleMember,
				}
				ur.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			wantError: true,
			errorType: services.ErrEmailAlreadyExists,
		},
		{
			name:      "Error - Pending invite already exists",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "pending@example.com",
				Role:  string(user.RoleMember),
			},
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, creatorID).Return(creator, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				ur.On("GetByEmail", mock.Anything, "pending@example.com").Return(nil, errors.New("not found"))
				pendingInvite := &user.Invite{
					ID:       uuid.New(),
					Email:    "pending@example.com",
					Status:   user.InviteStatusPending,
					FamilyID: familyID,
				}
				ir.On("GetPendingByEmail", "pending@example.com").Return([]*user.Invite{pendingInvite}, nil)
			},
			wantError: true,
		},
		{
			name:      "Error - Invalid role",
			creatorID: creatorID,
			dto: dto.CreateInviteDTO{
				Email: "newuser@example.com",
				Role:  "invalid_role",
			},
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, creatorID).Return(creator, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				ur.On("GetByEmail", mock.Anything, "newuser@example.com").Return(nil, errors.New("not found"))
				ir.On("GetPendingByEmail", "newuser@example.com").Return([]*user.Invite{}, nil)
			},
			wantError: true,
			errorType: services.ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo, userRepo, familyRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			invite, err := service.CreateInvite(context.Background(), tt.creatorID, tt.dto)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
				assert.Nil(t, invite)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, invite)
				assert.Equal(t, tt.dto.Email, invite.Email)
				assert.NotEmpty(t, invite.Token)
				assert.Equal(t, user.InviteStatusPending, invite.Status)
			}

			inviteRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestInviteService_GetInviteByToken(t *testing.T) {
	validToken := "valid-token-123"
	expiredToken := "expired-token-456"
	invalidToken := "invalid-token-789"

	validInvite := &user.Invite{
		ID:        uuid.New(),
		Token:     validToken,
		Email:     "user@example.com",
		Status:    user.InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	expiredInvite := &user.Invite{
		ID:        uuid.New(),
		Token:     expiredToken,
		Email:     "expired@example.com",
		Status:    user.InviteStatusPending,
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	}

	tests := []struct {
		name      string
		token     string
		setup     func(*MockInviteRepository)
		wantError bool
		errorType error
	}{
		{
			name:  "Success - Get valid invite",
			token: validToken,
			setup: func(ir *MockInviteRepository) {
				ir.On("GetByToken", validToken).Return(validInvite, nil)
			},
			wantError: false,
		},
		{
			name:  "Error - Invite not found",
			token: invalidToken,
			setup: func(ir *MockInviteRepository) {
				ir.On("GetByToken", invalidToken).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrInviteNotFound,
		},
		{
			name:  "Error - Invite expired",
			token: expiredToken,
			setup: func(ir *MockInviteRepository) {
				ir.On("GetByToken", expiredToken).Return(expiredInvite, nil)
				ir.On("Update", mock.AnythingOfType("*user.Invite")).Return(nil)
			},
			wantError: true,
			errorType: services.ErrInviteExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			invite, err := service.GetInviteByToken(context.Background(), tt.token)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, invite)
				assert.Equal(t, tt.token, invite.Token)
			}

			inviteRepo.AssertExpectations(t)
		})
	}
}

func TestInviteService_AcceptInvite(t *testing.T) {
	validToken := "valid-token-123"
	familyID := uuid.New()

	validInvite := &user.Invite{
		ID:        uuid.New(),
		FamilyID:  familyID,
		Token:     validToken,
		Email:     "user@example.com",
		Role:      user.RoleMember,
		Status:    user.InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	tests := []struct {
		name      string
		token     string
		dto       dto.AcceptInviteDTO
		setup     func(*MockInviteRepository, *MockUserRepository)
		wantError bool
		errorType error
	}{
		{
			name:  "Success - Accept valid invite",
			token: validToken,
			dto: dto.AcceptInviteDTO{
				Email:    "user@example.com",
				Name:     "John Doe",
				Password: "password123",
			},
			setup: func(ir *MockInviteRepository, ur *MockUserRepository) {
				ir.On("GetByToken", validToken).Return(validInvite, nil)
				ur.On("GetByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("not found"))
				ur.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
				ir.On("Update", mock.AnythingOfType("*user.Invite")).Return(nil)
			},
			wantError: false,
		},
		{
			name:  "Error - Email mismatch",
			token: validToken,
			dto: dto.AcceptInviteDTO{
				Email:    "wrong@example.com",
				Name:     "John Doe",
				Password: "password123",
			},
			setup: func(ir *MockInviteRepository, _ *MockUserRepository) {
				ir.On("GetByToken", validToken).Return(validInvite, nil)
			},
			wantError: true,
		},
		{
			name:  "Error - User already exists",
			token: validToken,
			dto: dto.AcceptInviteDTO{
				Email:    "user@example.com",
				Name:     "John Doe",
				Password: "password123",
			},
			setup: func(ir *MockInviteRepository, ur *MockUserRepository) {
				// Create a fresh invite for this test
				testInvite := &user.Invite{
					ID:        uuid.New(),
					FamilyID:  familyID,
					Token:     validToken,
					Email:     "user@example.com",
					Role:      user.RoleMember,
					Status:    user.InviteStatusPending,
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				ir.On("GetByToken", validToken).Return(testInvite, nil)
				existingUser := &user.User{
					ID:    uuid.New(),
					Email: "user@example.com",
				}
				ur.On("GetByEmail", mock.Anything, "user@example.com").Return(existingUser, nil)
			},
			wantError: true,
			errorType: services.ErrEmailAlreadyExists,
		},
		{
			name:  "Error - Invite not found",
			token: "invalid-token",
			dto: dto.AcceptInviteDTO{
				Email:    "user@example.com",
				Name:     "John Doe",
				Password: "password123",
			},
			setup: func(ir *MockInviteRepository, _ *MockUserRepository) {
				ir.On("GetByToken", "invalid-token").Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrInviteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo, userRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			newUser, err := service.AcceptInvite(context.Background(), tt.token, tt.dto)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
				assert.Nil(t, newUser)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, newUser)
				assert.Equal(t, tt.dto.Email, newUser.Email)
				assert.NotEmpty(t, newUser.Password)
			}

			inviteRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestInviteService_RevokeInvite(t *testing.T) {
	inviteID := uuid.New()
	revokerID := uuid.New()
	familyID := uuid.New()

	admin := &user.User{
		ID:   revokerID,
		Role: user.RoleAdmin,
	}

	member := &user.User{
		ID:   revokerID,
		Role: user.RoleMember,
	}

	family := &user.Family{
		ID:   familyID,
		Name: "Test Family",
	}

	pendingInvite := &user.Invite{
		ID:       inviteID,
		FamilyID: familyID,
		Status:   user.InviteStatusPending,
	}

	acceptedInvite := &user.Invite{
		ID:       inviteID,
		FamilyID: familyID,
		Status:   user.InviteStatusAccepted,
	}

	tests := []struct {
		name      string
		inviteID  uuid.UUID
		revokerID uuid.UUID
		setup     func(*MockInviteRepository, *MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name:      "Success - Revoke pending invite",
			inviteID:  inviteID,
			revokerID: revokerID,
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, revokerID).Return(admin, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				ir.On("GetByID", inviteID).Return(pendingInvite, nil)
				ir.On("Update", mock.AnythingOfType("*user.Invite")).Return(nil)
			},
			wantError: false,
		},
		{
			name:      "Error - Revoker not admin",
			inviteID:  inviteID,
			revokerID: revokerID,
			setup: func(_ *MockInviteRepository, ur *MockUserRepository, _ *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, revokerID).Return(member, nil)
			},
			wantError: true,
			errorType: services.ErrUnauthorized,
		},
		{
			name:      "Error - Cannot revoke accepted invite",
			inviteID:  inviteID,
			revokerID: revokerID,
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, revokerID).Return(admin, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				ir.On("GetByID", inviteID).Return(acceptedInvite, nil)
			},
			wantError: true,
		},
		{
			name:      "Error - Invite not found",
			inviteID:  uuid.New(),
			revokerID: revokerID,
			setup: func(ir *MockInviteRepository, ur *MockUserRepository, fr *MockFamilyRepository) {
				ur.On("GetByID", mock.Anything, revokerID).Return(admin, nil)
				fr.On("Get", mock.Anything).Return(family, nil)
				ir.On("GetByID", mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrInviteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo, userRepo, familyRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			err := service.RevokeInvite(context.Background(), tt.inviteID, tt.revokerID)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			inviteRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestInviteService_ListFamilyInvites(t *testing.T) {
	familyID := uuid.New()

	invites := []*user.Invite{
		{
			ID:        uuid.New(),
			FamilyID:  familyID,
			Email:     "user1@example.com",
			Status:    user.InviteStatusPending,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		{
			ID:        uuid.New(),
			FamilyID:  familyID,
			Email:     "user2@example.com",
			Status:    user.InviteStatusPending,
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
		},
	}

	tests := []struct {
		name      string
		familyID  uuid.UUID
		setup     func(*MockInviteRepository)
		wantError bool
	}{
		{
			name:     "Success - List family invites",
			familyID: familyID,
			setup: func(ir *MockInviteRepository) {
				ir.On("GetByFamily", familyID).Return(invites, nil)
				// Mock update for expired invite
				ir.On("Update", mock.AnythingOfType("*user.Invite")).Return(nil).Maybe()
			},
			wantError: false,
		},
		{
			name:     "Error - Repository error",
			familyID: familyID,
			setup: func(ir *MockInviteRepository) {
				ir.On("GetByFamily", familyID).Return(nil, errors.New("database error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			result, err := service.ListFamilyInvites(context.Background(), tt.familyID)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			inviteRepo.AssertExpectations(t)
		})
	}
}

func TestInviteService_DeleteExpiredInvites(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*MockInviteRepository)
		wantError bool
	}{
		{
			name: "Success - Delete expired invites",
			setup: func(ir *MockInviteRepository) {
				ir.On("DeleteExpired").Return(nil)
			},
			wantError: false,
		},
		{
			name: "Error - Repository error",
			setup: func(ir *MockInviteRepository) {
				ir.On("DeleteExpired").Return(errors.New("database error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inviteRepo := new(MockInviteRepository)
			userRepo := new(MockUserRepository)
			familyRepo := new(MockFamilyRepository)

			tt.setup(inviteRepo)

			service := services.NewInviteService(inviteRepo, userRepo, familyRepo, slog.Default())
			err := service.DeleteExpiredInvites(context.Background())

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			inviteRepo.AssertExpectations(t)
		})
	}
}
