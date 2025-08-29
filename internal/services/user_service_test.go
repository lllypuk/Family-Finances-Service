package services_test

import (
	"context"
	"errors"
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
				FamilyID:  uuid.New(),
			},
			setup: func(userRepo *MockUserRepository, familyRepo *MockFamilyRepository) {
				// Family exists
				familyRepo.On("GetByID", mock.Anything, mock.Anything).Return(&user.Family{
					ID:   uuid.New(),
					Name: "Test Family",
				}, nil)

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
				FamilyID:  uuid.New(),
			},
			setup:     func(*MockUserRepository, *MockFamilyRepository) {},
			wantError: true,
			errorType: services.ErrValidationFailed,
		},
		{
			name: "Error - Missing required fields",
			dto: dto.CreateUserDTO{
				Email: "test@example.com",
				// Missing FirstName, LastName, Password, Role, FamilyID
			},
			setup: func(ur *MockUserRepository, fr *MockFamilyRepository) {
				// No setup needed for validation test
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
				FamilyID:  uuid.New(),
			},
			setup: func(ur *MockUserRepository, fr *MockFamilyRepository) {
				// Family doesn't exist
				fr.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
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
				FamilyID:  uuid.New(),
			},
			setup: func(ur *MockUserRepository, fr *MockFamilyRepository) {
				// Family exists
				fr.On("GetByID", mock.Anything, mock.Anything).Return(&user.Family{
					ID:   uuid.New(),
					Name: "Test Family",
				}, nil)
				// User with same email already exists
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
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			result, err := service.CreateUser(context.Background(), tt.dto)

			// Assert
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
				assert.Equal(t, tt.dto.FamilyID, result.FamilyID)

				// Check password is hashed
				assert.NotEqual(t, tt.dto.Password, result.Password)
				err = bcrypt.CompareHashAndPassword([]byte(result.Password), []byte(tt.dto.Password))
				require.NoError(t, err, "Password should be properly hashed")
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	tests := []struct {
		name      string
		userID    uuid.UUID
		setup     func(*MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
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
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			result, err := service.GetUserByID(context.Background(), tt.userID)

			// Assert
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

			// Verify mocks
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
		FamilyID:  uuid.New(),
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
				FirstName: stringPtr("NewFirst"),
				LastName:  stringPtr("NewLast"),
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
				Email: stringPtr("new@example.com"),
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
				FirstName: stringPtr("NewFirst"),
			},
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
		{
			name:   "Error - Email already exists",
			userID: existingUser.ID,
			dto: dto.UpdateUserDTO{
				Email: stringPtr("existing@example.com"),
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
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			result, err := service.UpdateUser(context.Background(), tt.userID, tt.dto)

			// Assert
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

			// Verify mocks
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	existingUser := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		setup     func(*MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name:   "Success - Delete user",
			userID: existingUser.ID,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
				userRepo.On("Delete", mock.Anything, existingUser.ID).Return(nil)
			},
			wantError: false,
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			err := service.DeleteUser(context.Background(), tt.userID)

			// Assert
			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_ChangeUserRole(t *testing.T) {
	existingUser := &user.User{
		ID:   uuid.New(),
		Role: user.RoleMember,
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		role      user.Role
		setup     func(*MockUserRepository, *MockFamilyRepository)
		wantError bool
		errorType error
	}{
		{
			name:   "Success - Change role to admin",
			userID: existingUser.ID,
			role:   user.RoleAdmin,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUser.ID).Return(existingUser, nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
		{
			name:   "Error - Invalid role",
			userID: existingUser.ID,
			role:   user.Role("invalid"),
			setup: func(_ *MockUserRepository, _ *MockFamilyRepository) {
				// No expectations since role validation happens first
			},
			wantError: true,
			errorType: services.ErrInvalidRole,
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			role:   user.RoleAdmin,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found"))
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			err := service.ChangeUserRole(context.Background(), tt.userID, tt.role)

			// Assert
			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_ValidateUserAccess(t *testing.T) {
	familyID := uuid.New()
	user1 := &user.User{
		ID:       uuid.New(),
		FamilyID: familyID,
	}
	user2 := &user.User{
		ID:       uuid.New(),
		FamilyID: familyID,
	}
	userFromAnotherFamily := &user.User{
		ID:       uuid.New(),
		FamilyID: uuid.New(),
	}

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
			name:            "Error - Different family access",
			userID:          user1.ID,
			resourceOwnerID: userFromAnotherFamily.ID,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, user1.ID).Return(user1, nil)
				userRepo.On("GetByID", mock.Anything, userFromAnotherFamily.ID).Return(userFromAnotherFamily, nil)
			},
			wantError: true,
			errorType: services.ErrUnauthorized,
		},
		{
			name:            "Error - Requesting user not found",
			userID:          uuid.New(),
			resourceOwnerID: user2.ID,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found")).Once()
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			// Create service
			service := services.NewUserService(userRepo, familyRepo)

			// Execute
			err := service.ValidateUserAccess(context.Background(), tt.userID, tt.resourceOwnerID)

			// Assert
			if tt.wantError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify mocks
			userRepo.AssertExpectations(t)
			familyRepo.AssertExpectations(t)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
