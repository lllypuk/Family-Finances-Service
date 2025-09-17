package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	ErrFamilyNotFound     = errors.New("family not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrValidationFailed   = errors.New("validation failed")
)

// UserRepository defines the data access operations for users
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error)
	Update(ctx context.Context, user *user.User) error
	Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error
}

// FamilyRepository defines the data access operations for families
type FamilyRepository interface {
	Create(ctx context.Context, family *user.Family) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error)
	Update(ctx context.Context, family *user.Family) error
	Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error
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
		validator:  validator.New(),
	}
}

// CreateUser creates a new user with validation and password hashing
func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Validate family exists
	if err := s.validateFamilyExists(ctx, req.FamilyID); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user entity
	newUser := &user.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		FamilyID:  req.FamilyID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if createErr := s.userRepo.Create(ctx, newUser); createErr != nil {
		return nil, fmt.Errorf("failed to create user: %w", createErr)
	}

	return newUser, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	foundUser, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if foundUser == nil {
		return nil, ErrUserNotFound
	}

	return foundUser, nil
}

// GetUsersByFamily retrieves all users in a family
func (s *userService) GetUsersByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	// Validate family exists
	if err := s.validateFamilyExists(ctx, familyID); err != nil {
		return nil, err
	}

	users, err := s.userRepo.GetByFamilyID(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by family: %w", err)
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

// DeleteUser deletes a user by ID
func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Check if user exists
	if _, err := s.GetUserByID(ctx, id); err != nil {
		return err
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, id, familyID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
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
	requestingUser, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	resourceOwner, err := s.GetUserByID(ctx, resourceOwnerID)
	if err != nil {
		return err
	}

	// Users can access resources in their own family
	if requestingUser.FamilyID == resourceOwner.FamilyID {
		return nil
	}

	return ErrUnauthorized
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

// validateFamilyExists validates that a family exists
func (s *userService) validateFamilyExists(ctx context.Context, familyID uuid.UUID) error {
	if s.familyRepo == nil {
		return nil // Skip validation if family repository is not available
	}

	family, err := s.familyRepo.GetByID(ctx, familyID)
	if err != nil {
		return ErrFamilyNotFound
	}
	if family == nil {
		return ErrFamilyNotFound
	}

	return nil
}

// isValidRole checks if a role is valid
func (s *userService) isValidRole(role user.Role) bool {
	return role == user.RoleAdmin || role == user.RoleMember || role == user.RoleChild
}
