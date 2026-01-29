package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services/dto"
)

var (
	ErrCategoryNotFound            = errors.New("category not found")
	ErrCategoryNameExists          = errors.New("category with this name already exists in the same scope")
	ErrParentCategoryNotFound      = errors.New("parent category not found")
	ErrParentCategoryWrongFamily   = errors.New("parent category must belong to the same family")
	ErrParentCategoryWrongType     = errors.New("parent category must be of the same type")
	ErrMaxHierarchyLevels          = errors.New("cannot create more than 2 levels of category hierarchy")
	ErrCategorySelfParent          = errors.New("category cannot be its own parent")
	ErrCategoriesDifferentTypes    = errors.New("parent and child categories must be of the same type")
	ErrCategoriesDifferentFamilies = errors.New("parent and child categories must belong to the same family")
)

// CategoryRepository defines the data access operations for categories
type CategoryRepository interface {
	Create(ctx context.Context, category *category.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error)
	GetByType(ctx context.Context, familyID uuid.UUID, categoryType category.Type) ([]*category.Category, error)
	Update(ctx context.Context, category *category.Category) error
	Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error
}

// CategoryUsageChecker defines operations needed for category usage checks
type CategoryUsageChecker interface {
	IsCategoryUsed(ctx context.Context, categoryID uuid.UUID) (bool, error)
}

// categoryService implements CategoryService interface
type categoryService struct {
	categoryRepo CategoryRepository
	familyRepo   FamilyRepository
	usageChecker CategoryUsageChecker
	validator    *validator.Validate
}

// NewCategoryService creates a new CategoryService instance
func NewCategoryService(
	categoryRepo CategoryRepository,
	familyRepo FamilyRepository,
	usageChecker CategoryUsageChecker,
) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		familyRepo:   familyRepo,
		usageChecker: usageChecker,
		validator:    validator.New(),
	}
}

// CreateCategory creates a new category with business logic validation
func (s *categoryService) CreateCategory(ctx context.Context, req dto.CreateCategoryDTO) (*category.Category, error) {
	// Validate DTO
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate family exists
	if _, err := s.familyRepo.Get(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFamilyNotFound, err)
	}

	// Validate parent category if specified
	if req.ParentID != nil {
		parentCategory, err := s.categoryRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrParentCategoryNotFound, err)
		}

		// Parent must belong to the same family
		if parentCategory.FamilyID != req.FamilyID {
			return nil, ErrParentCategoryWrongFamily
		}

		// Parent must be of the same type
		if parentCategory.Type != req.Type {
			return nil, ErrParentCategoryWrongType
		}

		// Parent cannot be a subcategory itself (max 2 levels)
		if parentCategory.IsSubcategory() {
			return nil, ErrMaxHierarchyLevels
		}
	}

	// Check for duplicate category name within family and type
	existingCategories, err := s.categoryRepo.GetByType(ctx, req.FamilyID, req.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing categories: %w", err)
	}

	for _, existing := range existingCategories {
		if existing.Name == req.Name && existing.ParentID == req.ParentID {
			return nil, ErrCategoryNameExists
		}
	}

	// Create new category
	newCategory := &category.Category{
		ID:        uuid.New(),
		Name:      req.Name,
		Type:      req.Type,
		Color:     req.Color,
		Icon:      req.Icon,
		ParentID:  req.ParentID,
		FamilyID:  req.FamilyID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if createErr := s.categoryRepo.Create(ctx, newCategory); createErr != nil {
		return nil, fmt.Errorf("failed to create category: %w", createErr)
	}

	return newCategory, nil
}

// GetCategoryByID retrieves a category by its ID
func (s *categoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	cat, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCategoryNotFound, err)
	}

	return cat, nil
}

// GetCategoriesByFamily retrieves categories for a family with optional type filter
func (s *categoryService) GetCategoriesByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	typeFilter *category.Type,
) ([]*category.Category, error) {
	// Validate family exists
	if _, err := s.familyRepo.Get(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFamilyNotFound, err)
	}

	var categories []*category.Category
	var err error

	if typeFilter != nil {
		categories, err = s.categoryRepo.GetByType(ctx, familyID, *typeFilter)
	} else {
		categories, err = s.categoryRepo.GetByFamilyID(ctx, familyID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	// Filter only active categories
	var activeCategories []*category.Category
	for _, cat := range categories {
		if cat.IsActive {
			activeCategories = append(activeCategories, cat)
		}
	}

	return activeCategories, nil
}

// UpdateCategory updates an existing category
func (s *categoryService) UpdateCategory(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateCategoryDTO,
) (*category.Category, error) {
	// Validate DTO
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing category
	existingCategory, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCategoryNotFound, err)
	}

	// Check if category name already exists (if name is being updated)
	if req.Name != nil && *req.Name != existingCategory.Name {
		existingCategories, checkErr := s.categoryRepo.GetByType(ctx, existingCategory.FamilyID, existingCategory.Type)
		if checkErr != nil {
			return nil, fmt.Errorf("failed to check existing categories: %w", checkErr)
		}

		for _, existing := range existingCategories {
			if existing.ID != id && existing.Name == *req.Name && existing.ParentID == existingCategory.ParentID {
				return nil, ErrCategoryNameExists
			}
		}
	}

	// Apply updates
	if req.Name != nil {
		existingCategory.Name = *req.Name
	}
	if req.Color != nil {
		existingCategory.Color = *req.Color
	}
	if req.Icon != nil {
		existingCategory.Icon = *req.Icon
	}
	existingCategory.UpdatedAt = time.Now()

	if updateErr := s.categoryRepo.Update(ctx, existingCategory); updateErr != nil {
		return nil, fmt.Errorf("failed to update category: %w", updateErr)
	}

	return existingCategory, nil
}

// DeleteCategory performs soft delete of a category
func (s *categoryService) DeleteCategory(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Get existing category
	existingCategory, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCategoryNotFound, err)
	}

	// Check if category is used in transactions
	isUsed, err := s.CheckCategoryUsage(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check category usage: %w", err)
	}

	if isUsed {
		// Soft delete - mark as inactive
		existingCategory.IsActive = false
		existingCategory.UpdatedAt = time.Now()

		if updateErr := s.categoryRepo.Update(ctx, existingCategory); updateErr != nil {
			return fmt.Errorf("failed to soft delete category: %w", updateErr)
		}
	} else {
		// Hard delete if not used
		if deleteErr := s.categoryRepo.Delete(ctx, id, familyID); deleteErr != nil {
			return fmt.Errorf("failed to hard delete category: %w", deleteErr)
		}
	}

	// Also handle subcategories
	subcategories, err := s.getSubcategories(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get subcategories: %w", err)
	}

	for _, subcat := range subcategories {
		if deleteErr := s.DeleteCategory(ctx, subcat.ID, familyID); deleteErr != nil {
			return fmt.Errorf("failed to delete subcategory %s: %w", subcat.ID, deleteErr)
		}
	}

	return nil
}

// GetCategoryHierarchy returns categories organized in a hierarchical structure
func (s *categoryService) GetCategoryHierarchy(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	// Get all categories for the family
	categories, err := s.GetCategoriesByFamily(ctx, familyID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Separate parent categories (those without ParentID)
	var parentCategories []*category.Category
	for _, cat := range categories {
		if cat.ParentID == nil {
			parentCategories = append(parentCategories, cat)
		}
	}

	return parentCategories, nil
}

// ValidateCategoryHierarchy validates that adding a parent-child relationship won't create cycles
func (s *categoryService) ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error {
	if categoryID == parentID {
		return ErrCategorySelfParent
	}

	// Get parent category
	parentCategory, err := s.categoryRepo.GetByID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrParentCategoryNotFound, err)
	}

	// Check if parent is already a subcategory (max 2 levels)
	if parentCategory.IsSubcategory() {
		return ErrMaxHierarchyLevels
	}

	// Get the category being updated
	cat, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCategoryNotFound, err)
	}

	// Categories must be of the same type
	if cat.Type != parentCategory.Type {
		return ErrCategoriesDifferentTypes
	}

	// Categories must belong to the same family
	if cat.FamilyID != parentCategory.FamilyID {
		return ErrCategoriesDifferentFamilies
	}

	return nil
}

// CheckCategoryUsage checks if a category is used in any transactions
func (s *categoryService) CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	return s.usageChecker.IsCategoryUsed(ctx, categoryID)
}

// getSubcategories is a helper method to get all subcategories of a parent category
func (s *categoryService) getSubcategories(ctx context.Context, parentID uuid.UUID) ([]*category.Category, error) {
	// Get parent category to know family and type
	parent, err := s.categoryRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}

	allCategories, err := s.categoryRepo.GetByFamilyID(ctx, parent.FamilyID)
	if err != nil {
		return nil, err
	}

	var subcategories []*category.Category
	for _, cat := range allCategories {
		if cat.ParentID != nil && *cat.ParentID == parentID {
			subcategories = append(subcategories, cat)
		}
	}

	return subcategories, nil
}

// CreateDefaultCategories creates default categories for a newly created family
func (s *categoryService) CreateDefaultCategories(ctx context.Context, familyID uuid.UUID) error {
	// Default expense categories
	expenseCategories := []struct {
		name  string
		color string
		icon  string
	}{
		{"Продукты", "#FF6B6B", "🛒"},
		{"Транспорт", "#4ECDC4", "🚗"},
		{"Коммунальные услуги", "#45B7D1", "🏠"},
		{"Развлечения", "#F7DC6F", "🎬"},
		{"Здоровье", "#BB8FCE", "🏥"},
		{"Одежда", "#85C1E9", "👕"},
		{"Образование", "#F8C471", "📚"},
		{"Прочее", "#AEB6BF", "📦"},
	}

	// Default income categories
	incomeCategories := []struct {
		name  string
		color string
		icon  string
	}{
		{"Зарплата", "#58D68D", "💰"},
		{"Бонус", "#76D7C4", "🎁"},
		{"Фриланс", "#F9E79F", "💻"},
		{"Инвестиции", "#D2B4DE", "📈"},
		{"Прочий доход", "#A9DFBF", "💵"},
	}

	// Create expense categories
	for _, cat := range expenseCategories {
		categoryDTO := dto.CreateCategoryDTO{
			Name:     cat.name,
			Type:     category.TypeExpense,
			Color:    cat.color,
			Icon:     cat.icon,
			FamilyID: familyID,
		}

		_, err := s.CreateCategory(ctx, categoryDTO)
		if err != nil {
			return fmt.Errorf("failed to create expense category %s: %w", cat.name, err)
		}
	}

	// Create income categories
	for _, cat := range incomeCategories {
		categoryDTO := dto.CreateCategoryDTO{
			Name:     cat.name,
			Type:     category.TypeIncome,
			Color:    cat.color,
			Icon:     cat.icon,
			FamilyID: familyID,
		}

		_, err := s.CreateCategory(ctx, categoryDTO)
		if err != nil {
			return fmt.Errorf("failed to create income category %s: %w", cat.name, err)
		}
	}

	return nil
}
