package services_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func TestUserService_CreateUser(t *testing.T) {
	familyID := uuid.New()
	family := &user.Family{ID: familyID, Name: "Test Family"}

	tests := []struct {
		name      string
		dto       dto.CreateUserDTO
		setup     func(*MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name: "Success - Create valid user",
			dto: dto.CreateUserDTO{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "password123",
				Role:      user.RoleMember,
			},
			setup: func(userRepo *MockUserRepository, familyRepo *MockFamilyRepository) {
				// Family exists
				familyRepo.On("Get", mock.Anything).Return(family, nil)

				// Email doesn't exist
				userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("not found"))

				// Create succeeds
				userRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
		{
			name: "Error - Invalid email",
			dto: dto.CreateUserDTO{
				Email:     "invalid-email",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "password123",
				Role:      user.RoleMember,
			},
			setup:     func(*MockUserRepository, *MockFamilyRepository) {},
			wantError: true,
			errorType: services.ErrValidationFailed,
		},
		{
			name: "Error - Missing required fields",
			dto: dto.CreateUserDTO{
				Email: "test@example.com",
			},
			setup: func(_ *MockUserRepository, _ *MockFamilyRepository) {
			},
			wantError: true,
			errorType: services.ErrValidationFailed,
		},
		{
			name: "Error - Family not found",
			dto: dto.CreateUserDTO{
				Email:     "test@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "password123",
				Role:      user.RoleMember,
			},
			setup: func(_ *MockUserRepository, fr *MockFamilyRepository) {
				fr.On("Get", mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrFamilyNotFound,
		},
		{
			name: "Error - Email already exists",
			dto: dto.CreateUserDTO{
				Email:     "existing@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "password123",
				Role:      user.RoleMember,
			},
			setup: func(ur *MockUserRepository, fr *MockFamilyRepository) {
				fr.On("Get", mock.Anything).Return(family, nil)
				ur.On("GetByEmail", mock.Anything, "existing@example.com").Return(&user.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
				}, nil)
			},
			wantError: true,
			errorType: services.ErrEmailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			service := services.NewUserService(userRepo, familyRepo)

			result, err := service.CreateUser(context.Background(), tt.dto)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.dto.Email, result.Email)
				assert.Equal(t, tt.dto.FirstName, result.FirstName)
				assert.Equal(t, tt.dto.LastName, result.LastName)
				assert.Equal(t, tt.dto.Role, result.Role)

				// Check password is hashed
				assert.NotEqual(t, tt.dto.Password, result.Password)
				err = bcrypt.CompareHashAndPassword([]byte(result.Password), []byte(tt.dto.Password))
				require.NoError(t, err, "Password should be properly hashed")
			}

			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	tests := []struct {
		name         string
		userID       uuid.UUID
		setup        func(*MockUserRepository, *MockFamilyRepository)
		wantError    bool
		errorType    error
		notErrorType error
	}{
		{
			name:   "Success - User found",
			userID: uuid.New(),
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				expectedUser := &user.User{
					ID:        uuid.New(),
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
				}
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(expectedUser, nil)
			},
			wantError: false,
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, user.ErrNotFound)
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
		{
			// Сбой инфраструктуры не должен выглядеть как «пользователя нет»:
			// один SQLITE_BUSY отвечал бы 404 вместо 500.
			name:   "Error - infrastructure failure is not a not-found",
			userID: uuid.New(),
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, errors.New("database is locked"))
			},
			wantError:    true,
			notErrorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			service := services.NewUserService(userRepo, familyRepo)
			result, err := service.GetUserByID(context.Background(), tt.userID)

			if tt.notErrorType != nil {
				require.Error(t, err)
				require.NotErrorIs(t, err, tt.notErrorType)
			}

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	existingUser := &user.User{
		ID:        uuid.New(),
		Email:     "old@example.com",
		FirstName: "OldFirst",
		LastName:  "OldLast",
		Role:      user.RoleMember,
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		dto       dto.UpdateUserDTO
		setup     func(*MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name:   "Success - Update user fields",
			userID: existingUser.ID,
			dto: dto.UpdateUserDTO{
				FirstName: new("NewFirst"),
				LastName:  new("NewLast"),
			},
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
		{
			name:   "Success - Update email",
			userID: existingUser.ID,
			dto: dto.UpdateUserDTO{
				Email: new("new@example.com"),
			},
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
				userRepo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, errors.New("not found"))
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			dto: dto.UpdateUserDTO{
				FirstName: new("NewFirst"),
			},
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, user.ErrNotFound)
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
		{
			name:   "Error - Email already exists",
			userID: existingUser.ID,
			dto: dto.UpdateUserDTO{
				Email: new("existing@example.com"),
			},
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
				userRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(&user.User{
					ID:    uuid.New(),
					Email: "existing@example.com",
				}, nil)
			},
			wantError: true,
			errorType: services.ErrEmailAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			service := services.NewUserService(userRepo, familyRepo)
			result, err := service.UpdateUser(context.Background(), tt.userID, tt.dto)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, result)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_PatchUser(t *testing.T) {
	actorID := uuid.New()
	memberID := uuid.New()
	lastAdminID := uuid.New()

	tests := []struct {
		name      string
		userID    uuid.UUID
		role      *user.Role
		active    *bool
		setup     func(*MockUserRepository)
		wantError error
	}{
		{
			// Самодеактивация отбивается раньше проверки «последний админ»: репозиторий не зовём,
			// поэтому роль из того же запроса тоже не применяется.
			name:      "Error - Cannot deactivate self",
			userID:    actorID,
			role:      ptr(user.RoleMember),
			active:    ptr(false),
			setup:     func(*MockUserRepository) {},
			wantError: services.ErrCannotDeactivateSelf,
		},
		{
			name:      "Error - Invalid role",
			userID:    memberID,
			role:      ptr(user.Role("invalid")),
			setup:     func(*MockUserRepository) {},
			wantError: services.ErrInvalidRole,
		},
		{
			name:   "Success - Role only",
			userID: memberID,
			role:   ptr(user.RoleAdmin),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, memberID, ptr(user.RoleAdmin), (*bool)(nil)).Return(nil)
			},
		},
		{
			name:   "Success - Activity only",
			userID: memberID,
			active: ptr(false),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, memberID, (*user.Role)(nil), ptr(false)).Return(nil)
			},
		},
		{
			name:   "Success - Both fields in one write",
			userID: memberID,
			role:   ptr(user.RoleMember),
			active: ptr(false),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, memberID, ptr(user.RoleMember), ptr(false)).Return(nil)
			},
		},
		{
			name:   "Success - Self role change is allowed",
			userID: actorID,
			role:   ptr(user.RoleMember),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, actorID, ptr(user.RoleMember), (*bool)(nil)).Return(nil)
			},
		},
		{
			// Проверку «последний активный админ» делает репозиторий в транзакции.
			name:   "Error - Last active admin",
			userID: lastAdminID,
			role:   ptr(user.RoleMember),
			active: ptr(false),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, lastAdminID, ptr(user.RoleMember), ptr(false)).
					Return(user.ErrLastAdmin)
			},
			wantError: services.ErrLastAdmin,
		},
		{
			name:   "Error - User not found",
			userID: memberID,
			active: ptr(true),
			setup: func(userRepo *MockUserRepository) {
				userRepo.On("Patch", mock.Anything, memberID, (*user.Role)(nil), ptr(true)).
					Return(fmt.Errorf("user with id x: %w", user.ErrNotFound))
			},
			wantError: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			tt.setup(userRepo)

			service := services.NewUserService(userRepo, &MockFamilyRepository{})
			err := service.PatchUser(context.Background(), tt.userID, tt.role, tt.active, actorID)

			if tt.wantError == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantError)
			}

			userRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_ValidateUserAccess(t *testing.T) {
	user1 := &user.User{ID: uuid.New()}
	user2 := &user.User{ID: uuid.New()}

	tests := []struct {
		name            string
		userID          uuid.UUID
		resourceOwnerID uuid.UUID
		setup           func(*MockUserRepository, *MockFamilyRepository)
		wantError       bool
		errorType       error
	}{
		{
			name:            "Success - Same family access",
			userID:          user1.ID,
			resourceOwnerID: user2.ID,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, user1.ID).Return(user1, nil)
				userRepo.On("GetByID", mock.Anything, user2.ID).Return(user2, nil)
			},
			wantError: false,
		},
		{
			name:            "Error - Requesting user not found",
			userID:          uuid.New(),
			resourceOwnerID: user2.ID,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, user.ErrNotFound).Once()
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			service := services.NewUserService(userRepo, familyRepo)
			err := service.ValidateUserAccess(context.Background(), tt.userID, tt.resourceOwnerID)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}
