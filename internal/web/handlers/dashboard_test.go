package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

func TestDashboardHandler_Dashboard(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockTransactionService, *MockCategoryService, *MockBudgetService)
		expectedStatus int
	}{
		{
			name: "Successfully show dashboard",
			setupMocks: func(transactionService *MockTransactionService, categoryService *MockCategoryService, budgetService *MockBudgetService) {
				// Mock Transaction service calls
				transactionService.On("GetTransactionsByFamily", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*transaction.Transaction{}, nil).
					Maybe()

				// Mock Category service calls
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&category.Category{
						ID:   uuid.New(),
						Name: "Test Category",
						Type: category.TypeExpense,
					}, nil).Maybe()

				// Mock Budget service calls
				budgetService.On("GetActiveBudgets", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*budget.Budget{}, nil).Maybe()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			transactionService := &MockTransactionService{}
			categoryService := &MockCategoryService{}
			budgetService := &MockBudgetService{}

			tt.setupMocks(transactionService, categoryService, budgetService)

			mockServices := &services.Services{
				Transaction: transactionService,
				Category:    categoryService,
				Budget:      budgetService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewDashboardHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
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
			err := handler.Dashboard(c)

			// Assert
			require.NoError(t, err)

			// Assert mock expectations
			transactionService.AssertExpectations(t)
			categoryService.AssertExpectations(t)
			budgetService.AssertExpectations(t)
		})
	}
}

func TestDashboardHandler_DashboardStats(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockTransactionService, *MockCategoryService, *MockBudgetService)
		expectedStatus int
	}{
		{
			name: "Successfully get dashboard stats",
			setupMocks: func(transactionService *MockTransactionService, categoryService *MockCategoryService, budgetService *MockBudgetService) {
				// Mock Transaction service calls
				transactionService.On("GetTransactionsByFamily", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*transaction.Transaction{}, nil).
					Maybe()

				// Mock Category service calls
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&category.Category{
						ID:   uuid.New(),
						Name: "Test Category",
						Type: category.TypeExpense,
					}, nil).Maybe()

				// Mock Budget service calls
				budgetService.On("GetActiveBudgets", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*budget.Budget{}, nil).Maybe()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			transactionService := &MockTransactionService{}
			categoryService := &MockCategoryService{}
			budgetService := &MockBudgetService{}

			tt.setupMocks(transactionService, categoryService, budgetService)

			mockServices := &services.Services{
				Transaction: transactionService,
				Category:    categoryService,
				Budget:      budgetService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewDashboardHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
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
			err := handler.DashboardStats(c)

			// Assert
			require.NoError(t, err)

			// Assert mock expectations
			transactionService.AssertExpectations(t)
			categoryService.AssertExpectations(t)
			budgetService.AssertExpectations(t)
		})
	}
}

func TestDashboardHandler_RecentTransactions(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockTransactionService, *MockCategoryService, *MockBudgetService)
		expectedStatus int
	}{
		{
			name: "Successfully get recent transactions",
			setupMocks: func(transactionService *MockTransactionService, categoryService *MockCategoryService, budgetService *MockBudgetService) {
				// Mock Transaction service calls
				transactionService.On("GetTransactionsByFamily", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*transaction.Transaction{}, nil).
					Maybe()

				// Mock Category service calls
				categoryService.On("GetCategoryByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&category.Category{
						ID:   uuid.New(),
						Name: "Test Category",
						Type: category.TypeExpense,
					}, nil).Maybe()

				// Mock Budget service calls
				budgetService.On("GetActiveBudgets", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*budget.Budget{}, nil).Maybe()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			transactionService := &MockTransactionService{}
			categoryService := &MockCategoryService{}
			budgetService := &MockBudgetService{}

			tt.setupMocks(transactionService, categoryService, budgetService)

			mockServices := &services.Services{
				Transaction: transactionService,
				Category:    categoryService,
				Budget:      budgetService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewDashboardHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/dashboard/transactions", nil)
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
			err := handler.RecentTransactions(c)

			// Assert
			require.NoError(t, err)

			// Assert mock expectations
			transactionService.AssertExpectations(t)
			categoryService.AssertExpectations(t)
			budgetService.AssertExpectations(t)
		})
	}
}
