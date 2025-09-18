package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	ErrFamilyAlreadyExists = errors.New("family already exists")
)

// familyService implements FamilyService interface
type familyService struct {
	familyRepo      FamilyRepository
	categoryService CategoryService
	validator       *validator.Validate
}

// NewFamilyService creates a new FamilyService instance
func NewFamilyService(familyRepo FamilyRepository, categoryService CategoryService) FamilyService {
	return &familyService{
		familyRepo:      familyRepo,
		categoryService: categoryService,
		validator:       validator.New(),
	}
}

// CreateFamily creates a new family with validation
func (s *familyService) CreateFamily(ctx context.Context, req dto.CreateFamilyDTO) (*user.Family, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Create family entity
	newFamily := &user.Family{
		ID:        uuid.New(),
		Name:      req.Name,
		Currency:  req.Currency,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := s.familyRepo.Create(ctx, newFamily); err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	// Create default categories for the new family
	if err := s.categoryService.CreateDefaultCategories(ctx, newFamily.ID); err != nil {
		return nil, fmt.Errorf("failed to create default categories: %w", err)
	}

	return newFamily, nil
}

// GetFamilyByID retrieves a family by ID
func (s *familyService) GetFamilyByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	family, err := s.familyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrFamilyNotFound
	}
	if family == nil {
		return nil, ErrFamilyNotFound
	}

	return family, nil
}

// UpdateFamily updates an existing family
func (s *familyService) UpdateFamily(ctx context.Context, id uuid.UUID, req dto.UpdateFamilyDTO) (*user.Family, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Get existing family
	existingFamily, err := s.GetFamilyByID(ctx, id)
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

// DeleteFamily deletes a family by ID
func (s *familyService) DeleteFamily(ctx context.Context, id uuid.UUID) error {
	// Check if family exists
	if _, err := s.GetFamilyByID(ctx, id); err != nil {
		return err
	}

	// Delete family
	if err := s.familyRepo.Delete(ctx, id, id); err != nil {
		return fmt.Errorf("failed to delete family: %w", err)
	}

	return nil
}
