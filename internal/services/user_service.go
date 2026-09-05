package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	ErrFamilyNotFound = errors.New("family not found")
	// ErrUserNotFound — тот же sentinel, что и user.ErrNotFound: хендлеры и auth.Service
	// проверяют его через errors.Is, не завися от пакета services.
	ErrUserNotFound       = user.ErrNotFound
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrValidationFailed   = errors.New("validation failed")
	// ErrLastAdmin — попытка деактивировать или понизить последнего активного администратора.
	// Модель однофамильная: без администратора некому заводить пользователей и
	// открытой регистрации нет, поэтому состояние невосстановимо. Тот же sentinel, что
	// и user.ErrLastAdmin: проверку делает репозиторий в транзакции.
	ErrLastAdmin = user.ErrLastAdmin
	// ErrCannotDeactivateSelf — администратор пытается деактивировать собственную учётную запись.
	ErrCannotDeactivateSelf = errors.New("cannot deactivate yourself")
)

// UserRepository defines the data access operations for users
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetAll(ctx context.Context) ([]*user.User, error)
	// Update пишет только профиль (email, имя, фамилия).
	Update(ctx context.Context, user *user.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	// UpdateRole и SetActive — одиночные записи с атомарной проверкой последнего активного
	// администратора: user.ErrLastAdmin, если запись оставила бы семью без него.
	UpdateRole(ctx context.Context, id uuid.UUID, role user.Role) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}

// SessionRevoker отзывает bearer-сессии пользователя; реализуется *auth.Service.
type SessionRevoker interface {
	RevokeAllSessions(ctx context.Context, userID uuid.UUID) error
}

// FamilyRepository defines the data access operations for the single family
type FamilyRepository interface {
	Create(ctx context.Context, family *user.Family) error
	Get(ctx context.Context) (*user.Family, error)
	Update(ctx context.Context, family *user.Family) error
	Exists(ctx context.Context) (bool, error)
	// Bootstrap — семья, категории и админ одной транзакцией; повтор → user.ErrFamilyExists.
	Bootstrap(ctx context.Context, family *user.Family, categories []*category.Category, admin *user.User) error
}

// userService implements UserService interface
type userService struct {
	userRepo   UserRepository
	familyRepo FamilyRepository
	sessions   SessionRevoker
	validator  *validator.Validate
}

// NewUserService creates a new UserService instance
func NewUserService(userRepo UserRepository, familyRepo FamilyRepository, sessions SessionRevoker) UserService {
	return &userService{
		userRepo:   userRepo,
		familyRepo: familyRepo,
		sessions:   sessions,
		validator:  newValidator(),
	}
}

// CreateUser creates a new user with validation and password hashing
func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Verify the single family exists
	_, err := s.familyRepo.Get(ctx)
	if err != nil {
		return nil, ErrFamilyNotFound
	}

	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user entity
	newUser := &user.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
		return nil, fmt.Errorf("failed to create user: %w", createErr)
	}

	return newUser, nil
}

// GetUserByID retrieves a user by ID.
//
// Сбой инфраструктуры (таймаут контекста, SQLITE_BUSY) НЕ схлопывается в
// ErrUserNotFound: вызывающему нужно отличать удалённого пользователя (404)
// от временно недоступной БД (500).
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	foundUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	if foundUser == nil {
		return nil, ErrUserNotFound
	}

	return foundUser, nil
}

// GetUsers retrieves all users of the single family
func (s *userService) GetUsers(ctx context.Context) ([]*user.User, error) {
	// Verify the single family exists
	_, err := s.familyRepo.Get(ctx)
	if err != nil {
		return nil, ErrFamilyNotFound
	}

	// Get all users (single family model)
	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}

// UpdateUser updates an existing user
func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Get existing user
	existingUser, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if email is being changed and already exists
	if req.Email != nil && *req.Email != existingUser.Email {
		if emailUser, emailErr := s.userRepo.GetByEmail(ctx, *req.Email); emailErr == nil && emailUser != nil {
			return nil, ErrEmailAlreadyExists
		}
	}

	// Update fields
	if req.FirstName != nil {
		existingUser.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		existingUser.LastName = *req.LastName
	}
	if req.Email != nil {
		existingUser.Email = *req.Email
	}
	existingUser.UpdatedAt = time.Now()

	// Save to database
	if updateErr := s.userRepo.Update(ctx, existingUser); updateErr != nil {
		return nil, fmt.Errorf("failed to update user: %w", updateErr)
	}

	return existingUser, nil
}

// SetActive включает или выключает пользователя. Деактивация отзывает все его сессии;
// себя и последнего активного администратора выключить нельзя.
func (s *userService) SetActive(ctx context.Context, id uuid.UUID, active bool, actorID uuid.UUID) error {
	if !active && id == actorID {
		return ErrCannotDeactivateSelf
	}

	existingUser, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	if existingUser.IsActive == active {
		return nil
	}

	if err = s.userRepo.SetActive(ctx, id, active); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if !active {
		if err = s.sessions.RevokeAllSessions(ctx, id); err != nil {
			return fmt.Errorf("failed to revoke sessions: %w", err)
		}
	}

	return nil
}

// ChangeUserRole changes a user's role; понижение последнего активного администратора — ErrLastAdmin.
func (s *userService) ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error {
	if !s.isValidRole(role) {
		return ErrInvalidRole
	}

	if err := s.userRepo.UpdateRole(ctx, userID, role); err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}

// ValidateUserAccess validates if a user has access to a resource
func (s *userService) ValidateUserAccess(ctx context.Context, userID, resourceOwnerID uuid.UUID) error {
	// Verify both users exist
	_, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = s.GetUserByID(ctx, resourceOwnerID)
	if err != nil {
		return err
	}

	// Single family model - all users are in the same family
	return nil
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	foundUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if foundUser == nil {
		return nil, ErrUserNotFound
	}

	return foundUser, nil
}

// isValidRole checks if a role is valid
func (s *userService) isValidRole(role user.Role) bool {
	return role == user.RoleAdmin || role == user.RoleMember || role == user.RoleChild
}
