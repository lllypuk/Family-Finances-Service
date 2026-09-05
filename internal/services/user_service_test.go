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

			service := services.NewUserService(userRepo, familyRepo, &MockSessionRevoker{})

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

			service := services.NewUserService(userRepo, familyRepo, &MockSessionRevoker{})
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

			service := services.NewUserService(userRepo, familyRepo, &MockSessionRevoker{})
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

func TestUserService_SetActive(t *testing.T) {
	// Фикстуры создаются на каждый кейс: SetActive меняет IsActive у переданного объекта.
	newUser := func(id uuid.UUID, role user.Role, active bool) *user.User {
		return &user.User{ID: id, Email: id.String() + "@example.com", Role: role, IsActive: active}
	}
	actorID := uuid.New()
	memberID := uuid.New()
	lastAdminID := uuid.New()
	otherAdminID := uuid.New()

	tests := []struct {
		name      string
		userID    uuid.UUID
		active    bool
		setup     func(*MockUserRepository, *MockSessionRevoker)
		wantError error
	}{
		{
			// Правило «себя выключить нельзя» живёт в сервисе, а не в хендлерах.
			name:      "Error - Cannot deactivate self",
			userID:    actorID,
			active:    false,
			setup:     func(*MockUserRepository, *MockSessionRevoker) {},
			wantError: services.ErrCannotDeactivateSelf,
		},
		{
			name:   "Success - Deactivate member revokes sessions",
			userID: memberID,
			active: false,
			setup: func(userRepo *MockUserRepository, sessions *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, memberID).Return(newUser(memberID, user.RoleMember, true), nil)
				userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
					return u.ID == memberID && !u.IsActive
				})).Return(nil)
				sessions.On("RevokeAllSessions", mock.Anything, memberID).Return(nil)
			},
		},
		{
			name:   "Success - Reactivate keeps sessions untouched",
			userID: memberID,
			active: true,
			setup: func(userRepo *MockUserRepository, _ *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, memberID).Return(newUser(memberID, user.RoleMember, false), nil)
				userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
					return u.ID == memberID && u.IsActive
				})).Return(nil)
			},
		},
		{
			name:   "Success - Already in requested state is a no-op",
			userID: memberID,
			active: true,
			setup: func(userRepo *MockUserRepository, _ *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, memberID).Return(newUser(memberID, user.RoleMember, true), nil)
			},
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			active: false,
			setup: func(userRepo *MockUserRepository, _ *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, user.ErrNotFound)
			},
			wantError: services.ErrUserNotFound,
		},
		{
			// Неактивные админы не считаются: семья без активного админа невосстановима.
			name:   "Error - Last active admin",
			userID: lastAdminID,
			active: false,
			setup: func(userRepo *MockUserRepository, _ *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, lastAdminID).
					Return(newUser(lastAdminID, user.RoleAdmin, true), nil)
				userRepo.On("GetAll", mock.Anything).Return([]*user.User{
					newUser(lastAdminID, user.RoleAdmin, true),
					newUser(otherAdminID, user.RoleAdmin, false),
					newUser(memberID, user.RoleMember, true),
				}, nil)
			},
			wantError: services.ErrLastAdmin,
		},
		{
			name:   "Success - Admin deactivated while another active admin remains",
			userID: lastAdminID,
			active: false,
			setup: func(userRepo *MockUserRepository, sessions *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, lastAdminID).
					Return(newUser(lastAdminID, user.RoleAdmin, true), nil)
				userRepo.On("GetAll", mock.Anything).Return([]*user.User{
					newUser(lastAdminID, user.RoleAdmin, true),
					newUser(otherAdminID, user.RoleAdmin, true),
				}, nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
				sessions.On("RevokeAllSessions", mock.Anything, lastAdminID).Return(nil)
			},
		},
		{
			name:   "Error - Session revocation failure surfaces",
			userID: memberID,
			active: false,
			setup: func(userRepo *MockUserRepository, sessions *MockSessionRevoker) {
				userRepo.On("GetByID", mock.Anything, memberID).Return(newUser(memberID, user.RoleMember, true), nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
				sessions.On("RevokeAllSessions", mock.Anything, memberID).Return(errors.New("db down"))
			},
			wantError: errors.New("db down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			sessions := &MockSessionRevoker{}
			tt.setup(userRepo, sessions)

			service := services.NewUserService(userRepo, &MockFamilyRepository{}, sessions)
			err := service.SetActive(context.Background(), tt.userID, tt.active, actorID)

			switch {
			case tt.wantError == nil:
				require.NoError(t, err)
			case errors.Is(tt.wantError, services.ErrCannotDeactivateSelf),
				errors.Is(tt.wantError, services.ErrUserNotFound),
				errors.Is(tt.wantError, services.ErrLastAdmin):
				require.ErrorIs(t, err, tt.wantError)
			default:
				require.ErrorContains(t, err, tt.wantError.Error())
			}

			userRepo.AssertExpectations(t)
			sessions.AssertExpectations(t)
		})
	}
}

func TestUserService_ChangeUserRole(t *testing.T) {
	// ChangeUserRole меняет Role у переданного объекта, поэтому фикстуры создаются
	// заново на каждый кейс: общий указатель делал таблицу зависимой от порядка.
	newUser := func(id uuid.UUID, role user.Role) *user.User {
		return &user.User{ID: id, Role: role, IsActive: true}
	}
	existingUserID := uuid.New()
	onlyAdminID := uuid.New()
	otherAdminID := uuid.New()
	plainMemberID := uuid.New()

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
			userID: existingUserID,
			role:   user.RoleAdmin,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, existingUserID).
					Return(newUser(existingUserID, user.RoleMember), nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
		{
			name:   "Error - Invalid role",
			userID: existingUserID,
			role:   user.Role("invalid"),
			setup: func(_ *MockUserRepository, _ *MockFamilyRepository) {
			},
			wantError: true,
			errorType: services.ErrInvalidRole,
		},
		{
			name:   "Error - User not found",
			userID: uuid.New(),
			role:   user.RoleAdmin,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				userRepo.On("GetByID", mock.Anything, mock.Anything).Return(nil, user.ErrNotFound)
			},
			wantError: true,
			errorType: services.ErrUserNotFound,
		},
		{
			// Понижение последнего админа оставляет семью без того, кто
			// заводит пользователей — тот же запрет, что и на удаление.
			name:   "Error - Demoting the last admin",
			userID: onlyAdminID,
			role:   user.RoleMember,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				admin := newUser(onlyAdminID, user.RoleAdmin)
				userRepo.On("GetByID", mock.Anything, onlyAdminID).Return(admin, nil)
				userRepo.On("GetAll", mock.Anything).
					Return([]*user.User{admin, newUser(plainMemberID, user.RoleMember)}, nil)
			},
			wantError: true,
			errorType: services.ErrLastAdmin,
		},
		{
			name:   "Success - Demoting an admin while another admin remains",
			userID: onlyAdminID,
			role:   user.RoleMember,
			setup: func(userRepo *MockUserRepository, _ *MockFamilyRepository) {
				admin := newUser(onlyAdminID, user.RoleAdmin)
				userRepo.On("GetByID", mock.Anything, onlyAdminID).Return(admin, nil)
				userRepo.On("GetAll", mock.Anything).
					Return([]*user.User{admin, newUser(otherAdminID, user.RoleAdmin)}, nil)
				userRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := &MockUserRepository{}
			familyRepo := &MockFamilyRepository{}
			tt.setup(userRepo, familyRepo)

			service := services.NewUserService(userRepo, familyRepo, &MockSessionRevoker{})
			err := service.ChangeUserRole(context.Background(), tt.userID, tt.role)

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

			service := services.NewUserService(userRepo, familyRepo, &MockSessionRevoker{})
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
