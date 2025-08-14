package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/handlers"
)

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// MockCategoryRepository is a mock implementation of category repository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *category.Category) error {
	args := m.Called(ctx, category)
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

func (m *MockCategoryRepository) GetByType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.CategoryType,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID, categoryType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *category.Category) error {
	args := m.Called(ctx, category)
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
			requestBody: handlers.CreateCategoryRequest{
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
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "Food", response.Data.Name)
				assert.Equal(t, "expense", response.Data.Type)
				assert.Equal(t, "#FF5733", response.Data.Color)
				assert.Equal(t, "food", response.Data.Icon)
			},
		},
		{
			name:        "Error - Invalid JSON",
			requestBody: `{"name": "Food", "type": "expense", "family_id": "invalid-uuid"}`,
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_CATEGORY_ID", response.Error.Code)
			},
		},
		{
			name: "Error - Missing required fields",
			requestBody: handlers.CreateCategoryRequest{
				Name:     "",
				Type:     "expense",
				FamilyID: familyID,
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
			},
		},
		{
			name: "Error - Repository error",
			requestBody: handlers.CreateCategoryRequest{
				Name:     "Food",
				Type:     "expense",
				Color:    "#FF5733",
				Icon:     "food",
				FamilyID: familyID,
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*category.Category")).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repos := &handlers.Repositories{
				Category: mockRepo,
			}
			handler := handlers.NewCategoryHandler(repos)

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

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_GetCategories(t *testing.T) {
	familyID := uuid.New()

	tests := []struct {
		name           string
		queryParams    map[string]string
		mockSetup      func(*MockCategoryRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name: "Success - Get all categories",
			queryParams: map[string]string{
				"family_id": familyID.String(),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				categories := []*category.Category{
					{
						ID:       uuid.New(),
						Name:     "Food",
						Type:     category.CategoryTypeExpense,
						Color:    "#FF5733",
						Icon:     "food",
						FamilyID: familyID,
						IsActive: true,
					},
					{
						ID:       uuid.New(),
						Name:     "Salary",
						Type:     category.CategoryTypeIncome,
						Color:    "#28A745",
						Icon:     "money",
						FamilyID: familyID,
						IsActive: true,
					},
				}
				repo.On("GetByFamilyID", mock.Anything, familyID).Return(categories, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[[]handlers.CategoryResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Len(t, response.Data, 2)
				assert.Equal(t, "Food", response.Data[0].Name)
				assert.Equal(t, "Salary", response.Data[1].Name)
			},
		},
		{
			name:        "Error - Missing family_id",
			queryParams: map[string]string{},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CREATE_FAILED", response.Error.Code)
			},
		},
		{
			name: "Error - Invalid family_id",
			queryParams: map[string]string{
				"family_id": "invalid-uuid",
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "MISSING_FAMILY_ID", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repos := &handlers.Repositories{
				Category: mockRepo,
			}
			handler := handlers.NewCategoryHandler(repos)

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
			name:       "Success - Get category by ID",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				cat := &category.Category{
					ID:       categoryID,
					Name:     "Food",
					Type:     category.CategoryTypeExpense,
					Color:    "#FF5733",
					Icon:     "food",
					FamilyID: familyID,
					IsActive: true,
				}
				repo.On("GetByID", mock.Anything, categoryID).Return(cat, nil)
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
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_FAMILY_ID", response.Error.Code)
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
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repos := &handlers.Repositories{
				Category: mockRepo,
			}
			handler := handlers.NewCategoryHandler(repos)

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
			name:       "Success - Update category",
			categoryID: categoryID.String(),
			requestBody: handlers.UpdateCategoryRequest{
				Name:  stringPtr("Updated Food"),
				Color: stringPtr("#FF6666"),
				Icon:  stringPtr("updated-food"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				existingCategory := &category.Category{
					ID:       categoryID,
					Name:     "Food",
					Type:     category.CategoryTypeExpense,
					Color:    "#FF5733",
					Icon:     "food",
					FamilyID: familyID,
					IsActive: true,
				}
				repo.On("GetByID", mock.Anything, categoryID).Return(existingCategory, nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*category.Category")).Return(nil)
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
				Name: stringPtr("Updated Food"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_CATEGORY_ID", response.Error.Code)
			},
		},
		{
			name:       "Error - Category not found",
			categoryID: categoryID.String(),
			requestBody: handlers.UpdateCategoryRequest{
				Name: stringPtr("Updated Food"),
			},
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("GetByID", mock.Anything, categoryID).Return(nil, errors.New("not found"))
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
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repos := &handlers.Repositories{
				Category: mockRepo,
			}
			handler := handlers.NewCategoryHandler(repos)

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
			name:       "Success - Delete category",
			categoryID: categoryID.String(),
			mockSetup: func(repo *MockCategoryRepository) {
				repo.On("Delete", mock.Anything, categoryID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody: func(t *testing.T, body string) {
				assert.Empty(t, body)
			},
		},
		{
			name:       "Error - Invalid category ID",
			categoryID: "invalid-uuid",
			mockSetup: func(repo *MockCategoryRepository) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_CATEGORY_ID", response.Error.Code)
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
			mockRepo := &MockCategoryRepository{}
			tt.mockSetup(mockRepo)

			repos := &handlers.Repositories{
				Category: mockRepo,
			}
			handler := handlers.NewCategoryHandler(repos)

			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID, nil)
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

			mockRepo.AssertExpectations(t)
		})
	}
}
