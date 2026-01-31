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
	ErrFamilyAlreadyExists = errors.New("family already exists")
)

// familyService implements FamilyService interface
type familyService struct {
	familyRepo      FamilyRepository
	userRepo        UserRepository
	categoryService CategoryService
	validator       *validator.Validate
}

// NewFamilyService creates a new FamilyService instance
func NewFamilyService(
	familyRepo FamilyRepository,
	userRepo UserRepository,
	categoryService CategoryService,
) FamilyService {
	return &familyService{
		familyRepo:      familyRepo,
		userRepo:        userRepo,
		categoryService: categoryService,
		validator:       validator.New(),
	}
}

// SetupFamily creates the initial family with admin user (bootstrap only)
func (s *familyService) SetupFamily(ctx context.Context, req dto.SetupFamilyDTO) (*user.Family, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Check if family already exists
	exists, err := s.familyRepo.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check family existence: %w", err)
	}
	if exists {
		return nil, ErrFamilyAlreadyExists
	}

	// Create family entity
	newFamily := &user.Family{
		ID:        uuid.New(),
		Name:      req.FamilyName,
		Currency:  req.Currency,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save family to database
	if err = s.familyRepo.Create(ctx, newFamily); err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	// Create default categories for the new family
	if err = s.categoryService.CreateDefaultCategories(ctx); err != nil {
		return nil, fmt.Errorf("failed to create default categories: %w", err)
	}

	// Create admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	adminUser := user.NewUser(req.Email, req.FirstName, req.LastName, user.RoleAdmin)
	adminUser.Password = string(hashedPassword)

	if err = s.userRepo.Create(ctx, adminUser); err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	return newFamily, nil
}

// GetFamily retrieves the single family
func (s *familyService) GetFamily(ctx context.Context) (*user.Family, error) {
	family, err := s.familyRepo.Get(ctx)
	if err != nil {
		return nil, ErrFamilyNotFound
	}
	if family == nil {
		return nil, ErrFamilyNotFound
	}

	return family, nil
}

// UpdateFamily updates the single family settings
func (s *familyService) UpdateFamily(ctx context.Context, req dto.UpdateFamilyDTO) (*user.Family, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Get existing family
	existingFamily, err := s.GetFamily(ctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		existingFamily.Name = *req.Name
	}
	if req.Currency != nil {
		existingFamily.Currency = *req.Currency
	}
	existingFamily.UpdatedAt = time.Now()

	// Save to database
	if updateErr := s.familyRepo.Update(ctx, existingFamily); updateErr != nil {
		return nil, fmt.Errorf("failed to update family: %w", updateErr)
	}

	return existingFamily, nil
}

// IsSetupComplete checks if the initial setup has been done
func (s *familyService) IsSetupComplete(ctx context.Context) (bool, error) {
	return s.familyRepo.Exists(ctx)
}
