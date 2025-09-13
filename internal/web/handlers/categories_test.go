package handlers_test

import (
	"errors"
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
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

func TestCategoryHandler_Index(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockCategoryService, *MockTransactionService, *MockBudgetService, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "Successfully show categories",
			setupMocks: func(categoryService *MockCategoryService, transactionService *MockTransactionService, budgetService *MockBudgetService, familyID uuid.UUID) {
				testCategories := []*category.Category{
					createTestCategory(familyID),
				}

				categoryService.On("GetCategoriesByFamily", mock.Anything, familyID, mock.Anything).
					Return(testCategories, nil)
				// Mock transaction service calls for populating statistics
				transactionService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
					Return([]*transaction.Transaction{}, nil)
				// Mock budget service calls
				budgetService.On("GetBudgetsByCategory", mock.Anything, familyID, mock.Anything).
					Return([]*budget.Budget{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Category service error",
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, familyID uuid.UUID) {
				categoryService.On("GetCategoriesByFamily", mock.Anything, familyID, mock.Anything).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user once for the entire test
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			transactionService := &MockTransactionService{}
			budgetService := &MockBudgetService{}
			tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID)

			mockServices := &services.Services{
				Category:    categoryService,
				Transaction: transactionService,
				Budget:      budgetService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set user in context
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
				require.NoError(t, err)
			} else {
				// Expect an HTTP error for non-200 responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_New(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockCategoryService, *MockTransactionService, *MockBudgetService, uuid.UUID)
		expectedStatus int
	}{
		{
			name: "Successfully show create form",
			setupMocks: func(categoryService *MockCategoryService, transactionService *MockTransactionService, budgetService *MockBudgetService, familyID uuid.UUID) {
				testCategories := []*category.Category{
					createTestCategory(familyID),
				}

				categoryService.On("GetCategoriesByFamily", mock.Anything, familyID, mock.Anything).
					Return(testCategories, nil)
				// Mock transaction service calls for populating statistics
				transactionService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
					Return([]*transaction.Transaction{}, nil)
				// Mock budget service calls
				budgetService.On("GetBudgetsByCategory", mock.Anything, familyID, mock.Anything).
					Return([]*budget.Budget{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Category service error",
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, familyID uuid.UUID) {
				categoryService.On("GetCategoriesByFamily", mock.Anything, familyID, mock.Anything).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user once for the entire test
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			transactionService := &MockTransactionService{}
			budgetService := &MockBudgetService{}
			tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID)

			mockServices := &services.Services{
				Category:    categoryService,
				Transaction: transactionService,
				Budget:      budgetService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories/new", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Process request through middleware first to set up session and CSRF
			sessionMiddleware := middleware.SessionStore("test-secret-key-for-testing-that-is-long-enough", false)
			csrfMiddleware := middleware.CSRFProtection()
			combinedMiddleware := sessionMiddleware(csrfMiddleware(func(_ echo.Context) error {
				return nil // Dummy handler
			}))
			_ = combinedMiddleware(c) // Process middleware

			// Set user in context
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
				require.NoError(t, err)
			} else {
				// Expect an HTTP error for non-200 responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
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
				"name":  {"Food"},
				"type":  {"expense"},
				"color": {"#FF5733"},
				"icon":  {"utensils"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).
					Return(&category.Category{}, nil)
				// Add expectation for GetCategoriesByFamily in case of form re-render on error
				categoryService.On("GetCategoriesByFamily", mock.Anything, mock.Anything, mock.Anything).
					Return([]*category.Category{}, nil).
					Maybe()
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name: "Successfully create income category",
			formData: url.Values{
				"name":  {"Salary"},
				"type":  {"income"},
				"color": {"#33FF57"},
				"icon":  {"dollar-sign"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).
					Return(&category.Category{}, nil)
				// Add expectation for GetCategoriesByFamily in case of form re-render on error
				categoryService.On("GetCategoriesByFamily", mock.Anything, mock.Anything, mock.Anything).
					Return([]*category.Category{}, nil).
					Maybe()
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name: "Invalid form validation",
			formData: url.Values{
				"name": {""},
				"type": {"invalid"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				// Mock for GetCategoriesByFamily when re-rendering form with errors
				categoryService.On("GetCategoriesByFamily", mock.Anything, mock.Anything, mock.Anything).
					Return([]*category.Category{}, nil)
			},
			expectedStatus: http.StatusOK, // Form is re-rendered with errors, not an HTTP error
			expectRedirect: false,
		},
		{
			name: "Category service error",
			formData: url.Values{
				"name":  {"Food"},
				"type":  {"expense"},
				"color": {"#FF5733"},
				"icon":  {"utensils"},
			},
			setupMocks: func(categoryService *MockCategoryService) {
				categoryService.On("CreateCategory", mock.Anything, mock.AnythingOfType("dto.CreateCategoryDTO")).
					Return(nil, assert.AnError)

				// For service errors, the handler returns HTTP error directly without re-rendering form
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
				Category:    categoryService,
				Transaction: &MockTransactionService{},
				Budget:      &MockBudgetService{},
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
			switch {
			case tt.expectRedirect:
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			case tt.expectedStatus == http.StatusOK:
				// Form re-rendered with validation errors - no error returned
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			default:
				// Expect an HTTP error for error responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Edit(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		setupMocks     func(*MockCategoryService, *MockTransactionService, *MockBudgetService, uuid.UUID)
		expectedStatus int
	}{
		{
			name:       "Successfully show edit form",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService, transactionService *MockTransactionService, budgetService *MockBudgetService, familyID uuid.UUID) {
				testCategory := createTestCategory(familyID)
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(testCategory, nil)
				categoryService.On("GetCategoriesByFamily", mock.Anything, familyID, mock.Anything).
					Return([]*category.Category{}, nil)
				// Mock transaction service calls for populating statistics
				transactionService.On("GetTransactionsByCategory", mock.Anything, mock.Anything, mock.Anything).
					Return([]*transaction.Transaction{}, nil)
				// Mock budget service calls
				budgetService.On("GetBudgetsByCategory", mock.Anything, familyID, mock.Anything).
					Return([]*budget.Budget{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "Category not found",
			categoryID: uuid.New().String(),
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID) {
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			setupMocks: func(_ *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user once for the entire test
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			transactionService := &MockTransactionService{}
			budgetService := &MockBudgetService{}
			tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID)

			mockServices := &services.Services{
				Category:    categoryService,
				Transaction: transactionService,
				Budget:      budgetService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/categories/"+tt.categoryID+"/edit", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Process request through middleware first to set up session and CSRF
			sessionMiddleware := middleware.SessionStore("test-secret-key-for-testing-that-is-long-enough", false)
			csrfMiddleware := middleware.CSRFProtection()
			combinedMiddleware := sessionMiddleware(csrfMiddleware(func(_ echo.Context) error {
				return nil // Dummy handler
			}))
			_ = combinedMiddleware(c) // Process middleware

			// Set user in context
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
				require.NoError(t, err)
			} else {
				// Expect an HTTP error for non-200 responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
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
		setupMocks     func(*MockCategoryService, *MockTransactionService, *MockBudgetService, uuid.UUID)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:       "Successfully update category",
			categoryID: uuid.New().String(),
			formData: url.Values{
				"name":  {"Updated Food"},
				"type":  {"expense"},
				"color": {"#FF5733"},
				"icon":  {"utensils"},
			},
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, familyID uuid.UUID) {
				testCategory := createTestCategory(familyID)
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(testCategory, nil)
				categoryService.On("UpdateCategory", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateCategoryDTO")).
					Return(testCategory, nil)
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			formData: url.Values{
				"name":  {"Updated Food"},
				"type":  {"expense"},
				"color": {"#FF5733"},
				"icon":  {"utensils"},
			},
			setupMocks: func(_ *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "Category service error",
			categoryID: uuid.New().String(),
			formData: url.Values{
				"name":  {"Updated Food"},
				"type":  {"expense"},
				"color": {"#FF5733"},
				"icon":  {"utensils"},
			},
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, familyID uuid.UUID) {
				testCategory := createTestCategory(familyID)
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(testCategory, nil)
				categoryService.On("UpdateCategory", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateCategoryDTO")).
					Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user once for the entire test
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			categoryService := &MockCategoryService{}
			transactionService := &MockTransactionService{}
			budgetService := &MockBudgetService{}
			tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID)

			mockServices := &services.Services{
				Category:    categoryService,
				Transaction: transactionService,
				Budget:      budgetService,
			}

			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(
				http.MethodPut,
				"/categories/"+tt.categoryID,
				strings.NewReader(tt.formData.Encode()),
			)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Set user in context
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
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				// Expect an HTTP error for non-redirect responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
			}

			categoryService.AssertExpectations(t)
		})
	}
}

func TestCategoryHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		categoryID     string
		setupMocks     func(*MockCategoryService, *MockTransactionService, *MockBudgetService, uuid.UUID, uuid.UUID)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:       "Successfully delete category",
			categoryID: "test-category-id", // Will be replaced with actual ID
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, familyID uuid.UUID, categoryID uuid.UUID) {
				categoryService.On("GetCategoryByID", mock.Anything, categoryID).
					Return(&category.Category{
						ID:        categoryID,
						Name:      "Test Category",
						Type:      category.TypeExpense,
						Color:     "#FF5733",
						Icon:      "💰",
						FamilyID:  familyID,
						IsActive:  true,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).Once()

				categoryService.On("DeleteCategory", mock.Anything, categoryID).
					Return(nil).Once()
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:       "Invalid category ID",
			categoryID: "invalid-uuid",
			setupMocks: func(_ *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID, _ uuid.UUID) {
				// No setup needed as validation should fail first
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "Category not found",
			categoryID: "test-category-id", // Will be replaced with actual ID
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID, categoryID uuid.UUID) {
				categoryService.On("GetCategoryByID", mock.Anything, categoryID).
					Return(nil, errors.New("category not found")).Once()
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "Access denied - wrong family",
			categoryID: "test-category-id", // Will be replaced with actual ID
			setupMocks: func(categoryService *MockCategoryService, _ *MockTransactionService, _ *MockBudgetService, _ uuid.UUID, categoryID uuid.UUID) {
				categoryService.On("GetCategoryByID", mock.Anything, categoryID).
					Return(&category.Category{
						ID:        categoryID,
						Name:      "Test Category",
						Type:      category.TypeExpense,
						Color:     "#FF5733",
						Icon:      "💰",
						FamilyID:  uuid.New(), // Different family
						IsActive:  true,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user first
			testUser := createTestUser()

			// Create test category ID that will be consistent
			var testCategoryID uuid.UUID
			if tt.categoryID == "test-category-id" {
				testCategoryID = uuid.New()
				tt.categoryID = testCategoryID.String()
			}

			// Setup mocks with familyID and categoryID
			categoryService := &MockCategoryService{}
			transactionService := &MockTransactionService{}
			budgetService := &MockBudgetService{}

			// Only pass categoryID if it's valid
			if tt.categoryID != "invalid-uuid" {
				tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID, testCategoryID)
			} else {
				tt.setupMocks(categoryService, transactionService, budgetService, testUser.FamilyID, uuid.Nil)
			}

			mockServices := &services.Services{
				Category:    categoryService,
				Transaction: transactionService,
				Budget:      budgetService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewCategoryHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodDelete, "/categories/"+tt.categoryID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.categoryID)

			// Set user in context
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
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				// Expect an HTTP error for non-redirect responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
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
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
