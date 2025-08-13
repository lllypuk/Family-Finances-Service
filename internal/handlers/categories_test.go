package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/category"
)

// MockCategoryRepository is a mock implementation of category repository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, c *category.Category) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByType(ctx context.Context, familyID uuid.UUID, categoryType category.CategoryType) ([]*category.Category, error) {
	args := m.Called(ctx, familyID, categoryType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, c *category.Category) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCategoryHandler_CreateCategory(t *testing.T) {
	familyID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name: "Success - Valid category creation",
			requestBody: CreateCategoryRequest{
				Name:     "Food",
				Type:     "expense",
				Color:    "#FF5733",
				Icon:     "food",
				FamilyID: familyID,
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Food", response.Data.Name)
				assert.Equal(t, "expense", response.Data.Type)
				assert.Equal(t, "#FF5733", response.Data.Color)
				assert.Equal(t, "food", response.Data.Icon)
				assert.Equal(t, familyID, response.Data.FamilyID)
				assert.True(t, response.Data.IsActive)
			},
		},
		{
			name: "Error - Missing required fields",
			requestBody: CreateCategoryRequest{
				Name: "Food",
				// Missing required fields
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[interface{}]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
			},
		},
		{
			name: "Error - Invalid color format",
			requestBody: CreateCategoryRequest{
				Name:     "Food",
				Type:     "expense",
				Color:    "invalid-color",
				Icon:     "food",
				FamilyID: familyID,
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[interface{}]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
			},
		},
		{
			name: "Error - Repository creation fails",
			requestBody: CreateCategoryRequest{
				Name:     "Food",
				Type:     "expense",
				Color:    "#FF5733",
				Icon:     "food",
				FamilyID: familyID,
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*category.Category")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "CREATE_FAILED", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repositories := &Repositories{Category: mockRepo}
			handler := NewCategoryHandler(repositories)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(jsonBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handler.CreateCategory(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_GetCategories(t *testing.T) {
	familyID := uuid.New()
	categories := []*category.Category{
		{
			ID:        uuid.New(),
			Name:      "Food",
			Type:      category.CategoryTypeExpense,
			Color:     "#FF5733",
			Icon:      "food",
			FamilyID:  familyID,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	tests := []struct {
		name           string
		queryParams    map[string]string
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name: "Success - Get all categories by family",
			queryParams: map[string]string{
				"family_id": familyID.String(),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("GetByFamilyID", mock.Anything, familyID).Return(categories, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[[]CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Len(t, response.Data, 1)
				assert.Equal(t, "Food", response.Data[0].Name)
			},
		},
		{
			name: "Success - Get categories by type",
			queryParams: map[string]string{
				"family_id": familyID.String(),
				"type":      "expense",
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("GetByType", mock.Anything, familyID, category.CategoryTypeExpense).Return(categories, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[[]CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Len(t, response.Data, 1)
			},
		},
		{
			name:        "Error - Missing family_id",
			queryParams: map[string]string{},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "MISSING_FAMILY_ID", response.Error.Code)
			},
		},
		{
			name: "Error - Invalid family_id format",
			queryParams: map[string]string{
				"family_id": "invalid-uuid",
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "INVALID_FAMILY_ID", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repositories := &Repositories{Category: mockRepo}
			handler := NewCategoryHandler(repositories)

			// Create request with query params
			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handler.GetCategories(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_GetCategoryByID(t *testing.T) {
	categoryID := uuid.New()
	familyID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Category found",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				cat := &category.Category{
					ID:        categoryID,
					Name:      "Food",
					Type:      category.CategoryTypeExpense,
					Color:     "#FF5733",
					Icon:      "food",
					FamilyID:  familyID,
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				repo.On("GetByID", mock.Anything, categoryID).Return(cat, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, categoryID, response.Data.ID)
				assert.Equal(t, "Food", response.Data.Name)
			},
		},
		{
			name:       "Error - Invalid UUID format",
			categoryID: "invalid-uuid",
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock setup needed for UUID validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("GetByID", mock.Anything, categoryID).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repositories := &Repositories{Category: mockRepo}
			handler := NewCategoryHandler(repositories)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/categories/"+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.GetCategoryByID(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_UpdateCategory(t *testing.T) {
	categoryID := uuid.New()
	familyID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		requestBody    interface{}
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Category updated",
			categoryID: categoryID.String(),
			requestBody: UpdateCategoryRequest{
				Name:  stringPtr("Updated Food"),
				Color: stringPtr("#FF6644"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				existingCategory := &category.Category{
					ID:        categoryID,
					Name:      "Food",
					Type:      category.CategoryTypeExpense,
					Color:     "#FF5733",
					Icon:      "food",
					FamilyID:  familyID,
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				repo.On("GetByID", mock.Anything, categoryID).Return(existingCategory, nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Updated Food", response.Data.Name)
				assert.Equal(t, "#FF6644", response.Data.Color)
			},
		},
		{
			name:       "Error - Invalid UUID",
			categoryID: "invalid-uuid",
			requestBody: UpdateCategoryRequest{
				Name: stringPtr("Updated Food"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock needed for UUID validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			requestBody: UpdateCategoryRequest{
				Name: stringPtr("Updated Food"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("GetByID", mock.Anything, categoryID).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repositories := &Repositories{Category: mockRepo}
			handler := NewCategoryHandler(repositories)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/categories/"+tt.categoryID, bytes.NewReader(jsonBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.UpdateCategory(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_DeleteCategory(t *testing.T) {
	categoryID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Category deleted",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Delete", mock.Anything, categoryID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response APIResponse[interface{}]
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				dataMap, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Category deleted successfully", dataMap["message"])
			},
		},
		{
			name:       "Error - Invalid UUID",
			categoryID: "invalid-uuid",
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock needed for UUID validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Delete", mock.Anything, categoryID).Return(errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				assert.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repositories := &Repositories{Category: mockRepo}
			handler := NewCategoryHandler(repositories)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.DeleteCategory(c)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockRepo.AssertExpectations(t)
		})
	}
}