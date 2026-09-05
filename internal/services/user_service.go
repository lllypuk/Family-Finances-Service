package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	ErrFamilyNotFound = errors.New("family not found")
	// ErrUserNotFound — тот же sentinel, что и user.ErrNotFound: слой middleware
	// проверяет его через errors.Is, не импортируя пакет services.
	ErrUserNotFound       = user.ErrNotFound
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrValidationFailed   = errors.New("validation failed")
	// ErrLastAdmin — попытка удалить или понизить последнего администратора семьи.
	// Модель однофамильная: без администратора некому выпускать инвайты и
	// открытой регистрации нет, поэтому состояние невосстановимо.
	ErrLastAdmin = errors.New("cannot delete the last admin")
	// ErrCannotDeleteSelf — пользователь пытается удалить собственную учётную запись.
	ErrCannotDeleteSelf = errors.New("cannot delete yourself")
)

// UserRepository defines the data access operations for users
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetAll(ctx context.Context) ([]*user.User, error)
	Update(ctx context.Context, user *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// FamilyRepository defines the data access operations for the single family
type FamilyRepository interface {
	Create(ctx context.Context, family *user.Family) error
	Get(ctx context.Context) (*user.Family, error)
	Update(ctx context.Context, family *user.Family) error
	Exists(ctx context.Context) (bool, error)
}

// userService implements UserService interface
type userService struct {
	userRepo   UserRepository
	familyRepo FamilyRepository
	validator  *validator.Validate
}

// NewUserService creates a new UserService instance
func NewUserService(userRepo UserRepository, familyRepo FamilyRepository) UserService {
	return &userService{
		userRepo:   userRepo,
		familyRepo: familyRepo,
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
// ErrUserNotFound: вызывающему нужно отличать удалённого пользователя от
// временно недоступной БД — middleware перепроверки сессии в первом случае
// гасит cookie, а во втором обязано вернуть 500 и сессию не трогать.
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

// DeleteUser deletes a user by ID.
// actorID — владелец сессии, от имени которого выполняется удаление: правило
// «себя удалить нельзя» — бизнес-правило, а не деталь конкретной поверхности,
// поэтому оно живёт здесь, а веб и API только раскладывают ErrCannotDeleteSelf
// по своим форматам ответа (как уже сделано для ErrLastAdmin).
func (s *userService) DeleteUser(ctx context.Context, id, actorID uuid.UUID) error {
	// Самоудаление: администратор мгновенно теряет и сессию, и доступ к
	// консоли, а в однофамильной модели вернуть его некому.
	if id == actorID {
		return ErrCannotDeleteSelf
	}

	// Check if user exists
	existingUser, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Последний администратор не удаляется ни через API, ни через веб:
	// иначе семья остаётся без администратора навсегда (инвайты выпускает
	// только он, открытой регистрации нет).
	if err = s.ensureNotLastAdmin(ctx, existingUser); err != nil {
		return err
	}

	// Delete user
	if err = s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ensureNotLastAdmin возвращает ErrLastAdmin, если удаляемый пользователь —
// единственный оставшийся администратор.
func (s *userService) ensureNotLastAdmin(ctx context.Context, target *user.User) error {
	if target.Role != user.RoleAdmin {
		return nil
	}

	users, err := s.userRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	for _, u := range users {
		if u.Role == user.RoleAdmin && u.ID != target.ID {
			return nil
		}
	}

	return ErrLastAdmin
}

// ChangeUserRole changes a user's role
func (s *userService) ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error {
	// Validate role first
	if !s.isValidRole(role) {
		return ErrInvalidRole
	}

	// Get existing user
	existingUser, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Понижение — такая же потеря администратора, как и удаление: семья
	// остаётся без того, кто выпускает инвайты и заводит пользователей.
	if role != user.RoleAdmin {
		if err = s.ensureNotLastAdmin(ctx, existingUser); err != nil {
			return err
		}
	}

	// Update role
	existingUser.Role = role
	existingUser.UpdatedAt = time.Now()

	// Save to database
	if updateErr := s.userRepo.Update(ctx, existingUser); updateErr != nil {
		return fmt.Errorf("failed to update user role: %w", updateErr)
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
