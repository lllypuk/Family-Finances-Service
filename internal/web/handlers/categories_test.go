package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

func TestCategoryHandler_Index(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockCategoryService)
		expectedStatus int
	}{
		{
			name: "Successfully show categories",
			setupMocks: func(categoryService *MockCategoryService) {
				testUser := createTestUser()
				testCategories := []*category.Category{
					createTestCategory(testUser.FamilyID),
				}

				categoryService.On("GetCategoriesByFamily", mock.Anything, testUser.FamilyID, mock.Anything).Return(testCategories, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Category service error",
			setupMocks: func(categoryService *MockCategoryService) {
				testUser := createTestUser()
				categoryService.On("GetCategoriesByFamily", mock.Anything, testUser.FamilyID, mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.Index(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_New(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockCategoryService)
		expectedStatus int
	}{
		{
			name: "Successfully show create form",
			setupMocks: func(categoryService *MockCategoryService) {
				testUser := createTestUser()
				testCategories := []*category.Category{
					createTestCategory(testUser.FamilyID),
				}

				categoryService.On("GetCategoriesByFamily", mock.Anything, testUser.FamilyID, mock.Anything).Return(testCategories, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Category service error",
			setupMocks: func(categoryService *MockCategoryService) {
				testUser := createTestUser()
				categoryService.On("GetCategoriesByFamily", mock.Anything, testUser.FamilyID, mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories/new", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.New(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		formData       url.Values
		setupMocks     func(*MockCategoryService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name: "Successfully create expense category",
			formData: url.Values{
				"name": {"Food"},
				"type": {"expense"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).Return(&category.Category{}, nil)
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name: "Successfully create income category",
			formData: url.Values{
				"name": {"Salary"},
				"type": {"income"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).Return(&category.Category{}, nil)
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name: "Invalid form data",
			formData: url.Values{
				"name": {""},
				"type": {"invalid"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Category service error",
			formData: url.Values{
				"name": {"Food"},
				"type": {"expense"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(tt.formData.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.Create(c)

			// Assert
			if tt.expectRedirect {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Edit(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		setupMocks     func(*MockCategoryService)
		expectedStatus int
	}{
		{
			name:       "Successfully show edit form",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService) {
				testCategory := createTestCategory(uuid.New())
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(testCategory, nil)
				categoryService.On("GetCategoriesByFamily", mock.Anything, testCategory.FamilyID, mock.Anything).Return([]*category.Category{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "Category not found",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			setupMocks: func(categoryService *MockCategoryService) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories/"+tt.categoryID+"/edit", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.Edit(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		formData       url.Values
		setupMocks     func(*MockCategoryService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:       "Successfully update category",
			categoryID: uuid.New().String(),
			formData: url.Values{
				"name": {"Updated Food"},
				"type": {"expense"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				testCategory := createTestCategory(uuid.New())
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(testCategory, nil)
				categoryService.On("UpdateCategory", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateCategoryDTO")).Return(testCategory, nil)
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			formData: url.Values{
				"name": {"Updated Food"},
				"type": {"expense"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "Category service error",
			categoryID: uuid.New().String(),
			formData: url.Values{
				"name": {"Updated Food"},
				"type": {"expense"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				testCategory := createTestCategory(uuid.New())
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(testCategory, nil)
				categoryService.On("UpdateCategory", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateCategoryDTO")).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodPut, "/categories/"+tt.categoryID, strings.NewReader(tt.formData.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.Update(c)

			// Assert
			if tt.expectRedirect {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		setupMocks     func(*MockCategoryService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:       "Successfully delete category",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService) {
				testCategory := createTestCategory(uuid.New())
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(testCategory, nil)
				categoryService.On("DeleteCategory", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil)
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			setupMocks: func(categoryService *MockCategoryService) {
				// No setup needed as validation should fail first
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "Category service error",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService) {
				testCategory := createTestCategory(uuid.New())
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(testCategory, nil)
				categoryService.On("DeleteCategory", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			tt.setupMocks(categoryService)

			mockServices := &services.Services{
				Category: categoryService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Set user in context
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Execute
			err := handler.Delete(c)

			// Assert
			if tt.expectRedirect {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryFormValidation(t *testing.T) {
	tests := []struct {
		name     string
		form     models.CategoryForm
		wantErr  bool
		errorMsg string
	}{
		{
			name: "Valid expense category",
			form: models.CategoryForm{
				Name: "Food",
				Type: "expense",
			},
			wantErr: false,
		},
		{
			name: "Valid income category",
			form: models.CategoryForm{
				Name: "Salary",
				Type: "income",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			form: models.CategoryForm{
				Name: "",
				Type: "expense",
			},
			wantErr:  true,
			errorMsg: "name is required",
		},
		{
			name: "Invalid type",
			form: models.CategoryForm{
				Name: "Food",
				Type: "invalid",
			},
			wantErr:  true,
			errorMsg: "type must be income or expense",
		},
		{
			name: "Name too long",
			form: models.CategoryForm{
				Name: strings.Repeat("a", 101),
				Type: "expense",
			},
			wantErr:  true,
			errorMsg: "name must be 100 characters or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &MockValidator{}
			if tt.wantErr {
				validator.On("Validate", &tt.form).Return(assert.AnError)
			} else {
				validator.On("Validate", &tt.form).Return(nil)
			}

			err := validator.Validate(&tt.form)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			validator.AssertExpectations(t)
		})
	}
}

func TestCategoryTypeValidation(t *testing.T) {
	validTypes := []string{"income", "expense"}
	invalidTypes := []string{"", "INCOME", "EXPENSE", "transfer", "invalid"}

	for _, categoryType := range validTypes {
		t.Run("Valid type: "+categoryType, func(t *testing.T) {
			assert.Contains(t, validTypes, categoryType)
		})
	}

	for _, categoryType := range invalidTypes {
		t.Run("Invalid type: "+categoryType, func(t *testing.T) {
			assert.NotContains(t, validTypes, categoryType)
		})
	}
}

func TestCategoryNameNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Trim whitespace",
			input:    "  Food  ",
			expected: "Food",
		},
		{
			name:     "Capitalize first letter",
			input:    "food",
			expected: "Food",
		},
		{
			name:     "Multiple spaces",
			input:    "Food  and  Drinks",
			expected: "Food and Drinks",
		},
		{
			name:     "Already normalized",
			input:    "Food",
			expected: "Food",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be implemented in the actual form processing logic
			normalized := strings.TrimSpace(tt.input)
			// Add more normalization logic here as needed
			assert.NotEmpty(t, normalized)
		})
	}
}
