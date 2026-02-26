package handlers_test

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
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Helper function for creating string pointers
//
//go:fix inline
func stringPtr(s string) *string {
	return new(s)
}

// MockCategoryService is a mock implementation of CategoryService
type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) CreateCategory(
	ctx context.Context,
	req dto.CreateCategoryDTO,
) (*category.Category, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) GetCategories(
	ctx context.Context,
	typeFilter *category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryService) UpdateCategory(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateCategoryDTO,
) (*category.Category, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryService) GetCategoryHierarchy(
	ctx context.Context,
) ([]*category.Category, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryService) ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error {
	args := m.Called(ctx, categoryID, parentID)
	return args.Error(0)
}

func (m *MockCategoryService) CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCategoryService) CreateDefaultCategories(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestCategoryHandler_CreateCategory(t *testing.T) {
	// Define a consistent family ID for all test cases
	testFamilyID := uuid.New()

	tests := []struct {
		name           string
		requestBody    any
		mockSetup      func(*MockCategoryService, uuid.UUID)
		expectedStatus int
		expectedBody   func(*testing.T, string)
	}{
		{
			name: "Success - Valid category creation",
			requestBody: handlers.CreateCategoryRequest{
				Name:  "Food",
				Type:  "expense",
				Color: "#FF5733",
				Icon:  "food",
			},
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				expectedCategory := &category.Category{
					ID:        uuid.New(),
					Name:      "Food",
					Type:      category.TypeExpense,
					Color:     "#FF5733",
					Icon:      "food",
					IsActive:  true,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				service.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).
					Return(expectedCategory, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "Food", response.Data.Name)
				assert.Equal(t, "expense", response.Data.Type)
				assert.Equal(t, "#FF5733", response.Data.Color)
				assert.Equal(t, "food", response.Data.Icon)
				assert.True(t, response.Data.IsActive)
			},
		},
		{
			name: "Error - Invalid JSON",
			requestBody: `{
				"name": "Food",
				"type": "expense",
				"color": "#FF5733",
				"icon": "food",
			}`,
			mockSetup: func(_ *MockCategoryService, _ uuid.UUID) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
			},
		},
		{
			name: "Error - Validation failure",
			requestBody: handlers.CreateCategoryRequest{
				Name: "",
				Type: "expense",
			},
			mockSetup: func(_ *MockCategoryService, _ uuid.UUID) {
				// No service call expected since validation fails at handler level
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[any]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
				assert.Equal(t, "VALIDATION_ERROR", response.Errors[0].Code)
			},
		},
		{
			name: "Error - Service error",
			requestBody: handlers.CreateCategoryRequest{
				Name:  "Food",
				Type:  "expense",
				Color: "#FF5733",
				Icon:  "food",
			},
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				service.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CREATE_FAILED", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockCategoryService{}
			tt.mockSetup(mockService, testFamilyID)

			repos := &handlers.Repositories{}
			handler := handlers.NewCategoryHandler(repos, mockService)

			e := echo.New()
			var req *http.Request
			if str, ok := tt.requestBody.(string); ok {
				req = httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBufferString(str))
			} else {
				body, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPost, "/categories", bytes.NewBuffer(body))
			}
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handler.CreateCategory(c)

			// Assert
			if tt.expectedStatus < 400 {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_GetCategories(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		mockSetup      func(*MockCategoryService)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:        "Success - Get all categories",
			queryParams: map[string]string{},
			mockSetup: func(service *MockCategoryService) {
				categories := []*category.Category{
					{
						ID:       uuid.New(),
						Name:     "Food",
						Type:     category.TypeExpense,
						Color:    "#FF5733",
						Icon:     "food",
						IsActive: true,
					},
					{
						ID:       uuid.New(),
						Name:     "Salary",
						Type:     category.TypeIncome,
						Color:    "#28A745",
						Icon:     "money",
						IsActive: true,
					},
				}
				service.On("GetCategories", mock.Anything, (*category.Type)(nil)).
					Return(categories, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[[]dto.CategoryAPIResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Len(t, response.Data, 2)
				assert.Equal(t, "Food", response.Data[0].Name)
				assert.Equal(t, "Salary", response.Data[1].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockCategoryService{}
			tt.mockSetup(mockService)

			repos := &handlers.Repositories{}
			handler := handlers.NewCategoryHandler(repos, mockService)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/categories", nil)

			// Add query parameters
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
			if tt.expectedStatus < 400 {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_GetCategoryByID(t *testing.T) {
	categoryID := uuid.New()
	testFamilyID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		mockSetup      func(*MockCategoryService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Get category by ID",
			categoryID: categoryID.String(),
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				cat := &category.Category{
					ID:       categoryID,
					Name:     "Food",
					Type:     category.TypeExpense,
					Color:    "#FF5733",
					Icon:     "food",
					IsActive: true,
				}
				service.On("GetCategoryByID", mock.Anything, categoryID).Return(cat, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "Food", response.Data.Name)
				assert.Equal(t, "expense", response.Data.Type)
			},
		},
		{
			name:       "Error - Invalid category ID",
			categoryID: "invalid-uuid",
			mockSetup: func(_ *MockCategoryService, _ uuid.UUID) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				service.On("GetCategoryByID", mock.Anything, categoryID).Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockCategoryService{}
			tt.mockSetup(mockService, testFamilyID)

			repos := &handlers.Repositories{}
			handler := handlers.NewCategoryHandler(repos, mockService)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/categories/"+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.GetCategoryByID(c)

			// Assert
			if tt.expectedStatus < 400 {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_UpdateCategory(t *testing.T) {
	categoryID := uuid.New()
	testFamilyID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		requestBody    any
		mockSetup      func(*MockCategoryService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Update category",
			categoryID: categoryID.String(),
			requestBody: handlers.UpdateCategoryRequest{
				Name:  new("Updated Food"),
				Color: new("#FF6666"),
				Icon:  new("updated-food"),
			},
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				updatedCategory := &category.Category{
					ID:       categoryID,
					Name:     "Updated Food",
					Type:     category.TypeExpense,
					Color:    "#FF6666",
					Icon:     "updated-food",
					IsActive: true,
				}
				service.On("UpdateCategory", mock.Anything, categoryID, mock.AnythingOfType("dto.UpdateCategoryDTO")).
					Return(updatedCategory, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "Updated Food", response.Data.Name)
				assert.Equal(t, "#FF6666", response.Data.Color)
				assert.Equal(t, "updated-food", response.Data.Icon)
			},
		},
		{
			name:       "Error - Invalid category ID",
			categoryID: "invalid-uuid",
			requestBody: handlers.UpdateCategoryRequest{
				Name: new("Updated Food"),
			},
			mockSetup: func(_ *MockCategoryService, _ uuid.UUID) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			requestBody: handlers.UpdateCategoryRequest{
				Name: new("Updated Food"),
			},
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				service.On("UpdateCategory", mock.Anything, categoryID, mock.AnythingOfType("dto.UpdateCategoryDTO")).
					Return(nil, services.ErrCategoryNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockCategoryService{}
			tt.mockSetup(mockService, testFamilyID)

			repos := &handlers.Repositories{}
			handler := handlers.NewCategoryHandler(repos, mockService)

			e := echo.New()
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/categories/"+tt.categoryID, bytes.NewBuffer(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.UpdateCategory(c)

			// Assert
			if tt.expectedStatus < 400 {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_DeleteCategory(t *testing.T) {
	categoryID := uuid.New()
	testFamilyID := uuid.New()

	tests := []struct {
		name           string
		categoryID     string
		mockSetup      func(*MockCategoryService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:       "Success - Delete category",
			categoryID: categoryID.String(),
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				service.On("DeleteCategory", mock.Anything, categoryID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody: func(t *testing.T, body string) {
				// Should be empty for 204 No Content (can be null or empty with newline)
				assert.True(
					t,
					body == "" || body == "null" || body == "null\n",
					"Expected empty body or 'null', got: '%s'",
					body,
				)
			},
		},
		{
			name:       "Error - Invalid category ID",
			categoryID: "invalid-uuid",
			mockSetup: func(_ *MockCategoryService, _ uuid.UUID) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			mockSetup: func(service *MockCategoryService, _ uuid.UUID) {
				service.On("DeleteCategory", mock.Anything, categoryID).
					Return(services.ErrCategoryNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CATEGORY_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &MockCategoryService{}
			tt.mockSetup(mockService, testFamilyID)

			repos := &handlers.Repositories{}
			handler := handlers.NewCategoryHandler(repos, mockService)

			e := echo.New()
			familyID := testFamilyID.String()
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID+"?family_id="+familyID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Execute
			err := handler.DeleteCategory(c)

			// Assert
			if tt.expectedStatus < 400 {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())

			mockService.AssertExpectations(t)
		})
	}
}
