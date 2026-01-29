package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Mock usage checker for category service tests
type MockCategoryUsageChecker struct {
	mock.Mock
}

func (m *MockCategoryUsageChecker) IsCategoryUsed(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).(bool), args.Error(1)
}

func setupCategoryService() (services.CategoryService, *MockCategoryRepository, *MockFamilyRepository, *MockCategoryUsageChecker) {
	mockCategoryRepo := &MockCategoryRepository{}
	mockFamilyRepo := &MockFamilyRepository{}
	mockUsageChecker := &MockCategoryUsageChecker{}
	service := services.NewCategoryService(mockCategoryRepo, mockFamilyRepo, mockUsageChecker)
	return service, mockCategoryRepo, mockFamilyRepo, mockUsageChecker
}

func TestCategoryService_CreateCategory_Success(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	req := dto.CreateCategoryDTO{
		Name:     "Test Category",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "test-icon",
		FamilyID: familyID,
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByType", mock.Anything, familyID, category.TypeExpense).Return([]*category.Category{}, nil)
	mockCategoryRepo.On("Create", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)

	// Execute
	result, err := service.CreateCategory(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.Name, result.Name)
	assert.Equal(t, req.Type, result.Type)
	assert.Equal(t, req.Color, result.Color)
	assert.Equal(t, req.Icon, result.Icon)
	assert.Equal(t, req.FamilyID, result.FamilyID)
	assert.True(t, result.IsActive)
	assert.NotEqual(t, uuid.Nil, result.ID)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_FamilyNotFound(t *testing.T) {
	service, _, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	req := dto.CreateCategoryDTO{
		Name:     "Test Category",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "test-icon",
		FamilyID: familyID,
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(nil, errors.New("family not found"))

	// Execute
	result, err := service.CreateCategory(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, services.ErrFamilyNotFound)

	mockFamilyRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_DuplicateName(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	existingCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Test Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	req := dto.CreateCategoryDTO{
		Name:     "Test Category", // Same name as existing
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "test-icon",
		FamilyID: familyID,
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByType", mock.Anything, familyID, category.TypeExpense).
		Return([]*category.Category{existingCategory}, nil)

	// Execute
	result, err := service.CreateCategory(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, services.ErrCategoryNameExists)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_WithParent_Success(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	parentID := uuid.New()

	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	parentCategory := &category.Category{
		ID:       parentID,
		Name:     "Parent Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	req := dto.CreateCategoryDTO{
		Name:     "Child Category",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "test-icon",
		ParentID: &parentID,
		FamilyID: familyID,
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByID", mock.Anything, parentID).Return(parentCategory, nil)
	mockCategoryRepo.On("GetByType", mock.Anything, familyID, category.TypeExpense).Return([]*category.Category{}, nil)
	mockCategoryRepo.On("Create", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)

	// Execute
	result, err := service.CreateCategory(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, &parentID, result.ParentID)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_CreateCategory_WithParent_WrongFamily(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	parentID := uuid.New()
	otherFamilyID := uuid.New()

	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	parentCategory := &category.Category{
		ID:       parentID,
		Name:     "Parent Category",
		Type:     category.TypeExpense,
		FamilyID: otherFamilyID, // Different family!
		IsActive: true,
	}

	req := dto.CreateCategoryDTO{
		Name:     "Child Category",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "test-icon",
		ParentID: &parentID,
		FamilyID: familyID,
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByID", mock.Anything, parentID).Return(parentCategory, nil)

	// Execute
	result, err := service.CreateCategory(context.Background(), req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, services.ErrParentCategoryWrongFamily)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoryByID_Success(t *testing.T) {
	service, mockCategoryRepo, _, _ := setupCategoryService()

	categoryID := uuid.New()
	expectedCategory := &category.Category{
		ID:       categoryID,
		Name:     "Test Category",
		Type:     category.TypeExpense,
		FamilyID: uuid.New(),
		IsActive: true,
	}

	// Setup mocks
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(expectedCategory, nil)

	// Execute
	result, err := service.GetCategoryByID(context.Background(), categoryID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedCategory, result)

	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoryByID_NotFound(t *testing.T) {
	service, mockCategoryRepo, _, _ := setupCategoryService()

	categoryID := uuid.New()

	// Setup mocks
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(nil, errors.New("category not found"))

	// Execute
	result, err := service.GetCategoryByID(context.Background(), categoryID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, services.ErrCategoryNotFound)

	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoriesByFamily_Success(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	expectedCategories := []*category.Category{
		{
			ID:       uuid.New(),
			Name:     "Category 1",
			Type:     category.TypeExpense,
			FamilyID: familyID,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Name:     "Category 2",
			Type:     category.TypeExpense,
			FamilyID: familyID,
			IsActive: true,
		},
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedCategories, nil)

	// Execute
	result, err := service.GetCategoriesByFamily(context.Background(), familyID, nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedCategories, result)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_GetCategoriesByFamily_WithTypeFilter(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	typeFilter := category.TypeExpense
	expectedCategories := []*category.Category{
		{
			ID:       uuid.New(),
			Name:     "Expense Category",
			Type:     category.TypeExpense,
			FamilyID: familyID,
			IsActive: true,
		},
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByType", mock.Anything, familyID, typeFilter).Return(expectedCategories, nil)

	// Execute
	result, err := service.GetCategoriesByFamily(context.Background(), familyID, &typeFilter)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedCategories, result)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_UpdateCategory_Success(t *testing.T) {
	service, mockCategoryRepo, _, _ := setupCategoryService()

	categoryID := uuid.New()
	existingCategory := &category.Category{
		ID:        categoryID,
		Name:      "Old Name",
		Type:      category.TypeExpense,
		Color:     "#FF0000",
		Icon:      "old-icon",
		FamilyID:  uuid.New(),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	newName := "New Name"
	newColor := "#00FF00"
	req := dto.UpdateCategoryDTO{
		Name:  &newName,
		Color: &newColor,
	}

	// Setup mocks
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(existingCategory, nil)
	mockCategoryRepo.On("GetByType", mock.Anything, existingCategory.FamilyID, existingCategory.Type).
		Return([]*category.Category{}, nil)
	mockCategoryRepo.On("Update", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)

	// Execute
	result, err := service.UpdateCategory(context.Background(), categoryID, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, result.Name)
	assert.Equal(t, newColor, result.Color)
	assert.Equal(t, existingCategory.Icon, result.Icon) // Should remain unchanged

	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_SoftDelete(t *testing.T) {
	service, mockCategoryRepo, _, mockUsageChecker := setupCategoryService()

	categoryID := uuid.New()
	existingCategory := &category.Category{
		ID:       categoryID,
		Name:     "Test Category",
		Type:     category.TypeExpense,
		FamilyID: uuid.New(),
		IsActive: true,
	}

	// Setup mocks - category is used in transactions
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(existingCategory, nil)
	mockUsageChecker.On("IsCategoryUsed", mock.Anything, categoryID).Return(true, nil)
	mockCategoryRepo.On("GetByFamilyID", mock.Anything, existingCategory.FamilyID).Return([]*category.Category{}, nil)
	mockCategoryRepo.On("Update", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)

	// Execute
	err := service.DeleteCategory(context.Background(), categoryID, existingCategory.FamilyID)

	// Assert
	require.NoError(t, err)

	mockCategoryRepo.AssertExpectations(t)
	mockUsageChecker.AssertExpectations(t)
}

func TestCategoryService_DeleteCategory_HardDelete(t *testing.T) {
	service, mockCategoryRepo, _, mockUsageChecker := setupCategoryService()

	categoryID := uuid.New()
	existingCategory := &category.Category{
		ID:       categoryID,
		Name:     "Test Category",
		Type:     category.TypeExpense,
		FamilyID: uuid.New(),
		IsActive: true,
	}

	// Setup mocks - category is not used in transactions
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(existingCategory, nil)
	mockUsageChecker.On("IsCategoryUsed", mock.Anything, categoryID).Return(false, nil)
	mockCategoryRepo.On("GetByFamilyID", mock.Anything, existingCategory.FamilyID).Return([]*category.Category{}, nil)
	mockCategoryRepo.On("Delete", mock.Anything, categoryID, existingCategory.FamilyID).Return(nil)

	// Execute
	err := service.DeleteCategory(context.Background(), categoryID, existingCategory.FamilyID)

	// Assert
	require.NoError(t, err)

	mockCategoryRepo.AssertExpectations(t)
	mockUsageChecker.AssertExpectations(t)
}

func TestCategoryService_GetCategoryHierarchy_Success(t *testing.T) {
	service, mockCategoryRepo, mockFamilyRepo, _ := setupCategoryService()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	parentCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Parent Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	childCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Child Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		ParentID: &parentCategory.ID,
		IsActive: true,
	}

	allCategories := []*category.Category{parentCategory, childCategory}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockCategoryRepo.On("GetByFamilyID", mock.Anything, familyID).Return(allCategories, nil)

	// Execute
	result, err := service.GetCategoryHierarchy(context.Background(), familyID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1) // Only parent categories
	assert.Equal(t, parentCategory.ID, result[0].ID)

	mockFamilyRepo.AssertExpectations(t)
	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_ValidateCategoryHierarchy_Success(t *testing.T) {
	service, mockCategoryRepo, _, _ := setupCategoryService()

	categoryID := uuid.New()
	parentID := uuid.New()
	familyID := uuid.New()

	categoryObj := &category.Category{
		ID:       categoryID,
		Name:     "Child Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	parentCategory := &category.Category{
		ID:       parentID,
		Name:     "Parent Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	// Setup mocks
	mockCategoryRepo.On("GetByID", mock.Anything, parentID).Return(parentCategory, nil)
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(categoryObj, nil)

	// Execute
	err := service.ValidateCategoryHierarchy(context.Background(), categoryID, parentID)

	// Assert
	require.NoError(t, err)

	mockCategoryRepo.AssertExpectations(t)
}

func TestCategoryService_ValidateCategoryHierarchy_SameCategory(t *testing.T) {
	service, _, _, _ := setupCategoryService()

	categoryID := uuid.New()

	// Execute
	err := service.ValidateCategoryHierarchy(context.Background(), categoryID, categoryID)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrCategorySelfParent)
}

func TestCategoryService_ValidateCategoryHierarchy_DifferentTypes(t *testing.T) {
	service, mockCategoryRepo, _, _ := setupCategoryService()

	categoryID := uuid.New()
	parentID := uuid.New()
	familyID := uuid.New()

	categoryObj := &category.Category{
		ID:       categoryID,
		Name:     "Income Category",
		Type:     category.TypeIncome,
		FamilyID: familyID,
		IsActive: true,
	}

	parentCategory := &category.Category{
		ID:       parentID,
		Name:     "Expense Category",
		Type:     category.TypeExpense,
		FamilyID: familyID,
		IsActive: true,
	}

	// Setup mocks
	mockCategoryRepo.On("GetByID", mock.Anything, parentID).Return(parentCategory, nil)
	mockCategoryRepo.On("GetByID", mock.Anything, categoryID).Return(categoryObj, nil)

	// Execute
	err := service.ValidateCategoryHierarchy(context.Background(), categoryID, parentID)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, services.ErrCategoriesDifferentTypes)

	mockCategoryRepo.AssertExpectations(t)
}
