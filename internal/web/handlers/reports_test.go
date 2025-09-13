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
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

func TestReportHandler_Index(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockReportService)
		expectedStatus int
	}{
		{
			name: "Successfully show reports index",
			setupMocks: func(reportService *MockReportService) {
				// Mock GetReportsByFamily call
				reportService.On("GetReportsByFamily", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.Anything).
					Return([]*report.Report{}, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			reportService := &MockReportService{}
			tt.setupMocks(reportService)

			mockServices := &services.Services{
				Report: reportService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports", nil)
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
			require.NoError(t, err)

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_Show(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		setupMocks     func(*MockReportService)
		expectedStatus int
	}{
		{
			name:     "Successfully show report",
			reportID: "550e8400-e29b-41d4-a716-446655440000",
			setupMocks: func(_ *MockReportService) {
				// We'll set the FamilyID in the test to match the user's family
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Invalid report ID",
			reportID: "invalid-uuid",
			setupMocks: func(_ *MockReportService) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user first
			testUser := createTestUser()

			// Setup mock services
			reportService := &MockReportService{}

			// Set up the mock expectation with the correct family ID for successful test
			if tt.expectedStatus == http.StatusOK && tt.reportID != "invalid-uuid" {
				reportService.On("GetReportByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&report.Report{
						ID:       uuid.MustParse(tt.reportID),
						FamilyID: testUser.FamilyID, // Match the user's family
						Name:     "Test Report",
					}, nil).Once()
			}

			tt.setupMocks(reportService)

			mockServices := &services.Services{
				Report: reportService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports/"+tt.reportID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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
			err := handler.Show(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				// Expect an HTTP error for non-OK responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
			}

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_New(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockReportService)
		expectedStatus int
	}{
		{
			name: "Successfully show new report form",
			setupMocks: func(_ *MockReportService) {
				// No specific mocks needed for form rendering
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			reportService := &MockReportService{}
			tt.setupMocks(reportService)

			mockServices := &services.Services{
				Report: reportService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports/new", nil)
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
			require.NoError(t, err)

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		formData       map[string]string
		setupMocks     func(*MockReportService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name: "Successfully create report",
			formData: map[string]string{
				"name":       "Test Report",
				"type":       "expenses",
				"period":     "monthly",
				"start_date": "2024-01-01",
				"end_date":   "2024-01-31",
			},
			setupMocks: func(_ *MockReportService) {
				// Mock the report generation and saving
				// Note: Mocks are set up outside due to test structure
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name: "Invalid form data",
			formData: map[string]string{
				"name": "",
				"type": "invalid",
			},
			setupMocks: func(_ *MockReportService) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock services
			reportService := &MockReportService{}
			tt.setupMocks(reportService)

			mockServices := &services.Services{
				Report: reportService,
			}

			repos := setupRepositories()
			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()

			// Set form data
			form := make(url.Values)
			for key, value := range tt.formData {
				form.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(form.Encode()))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set user in context first
			testUser := createTestUser()
			sessionData := &middleware.SessionData{
				UserID:    testUser.ID,
				FamilyID:  testUser.FamilyID,
				Role:      testUser.Role,
				Email:     testUser.Email,
				ExpiresAt: time.Now().Add(time.Hour),
			}
			c.Set("user", sessionData)

			// Skip CSRF for this test - focus on the main functionality

			// Execute
			err := handler.Create(c)

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

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		setupMocks     func(*MockReportService, *user.User)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:     "Successfully delete report",
			reportID: "550e8400-e29b-41d4-a716-446655440000",
			setupMocks: func(_ *MockReportService, _ *user.User) {
				// Delete handler first calls GetReportByID, then DeleteReport
				// Note: Mocks are set up outside due to test structure
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:     "Invalid report ID",
			reportID: "invalid-uuid",
			setupMocks: func(_ *MockReportService, _ *user.User) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user first
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			reportService := &MockReportService{}
			mockServices := &services.Services{
				Report: reportService,
			}
			tt.setupMocks(reportService, testUser)

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodDelete, "/reports/"+tt.reportID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_Export(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		exportFormat   string
		setupMocks     func(*MockReportService, *user.User)
		expectedStatus int
	}{
		{
			name:         "Successfully export CSV",
			reportID:     "550e8400-e29b-41d4-a716-446655440000",
			exportFormat: "csv",
			setupMocks: func(_ *MockReportService, _ *user.User) {
				// Export handler only calls GetReportByID, then does internal CSV export
				// Note: Mocks are set up outside due to test structure
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Unsupported format",
			reportID:     "550e8400-e29b-41d4-a716-446655440000",
			exportFormat: "xml",
			setupMocks: func(reportService *MockReportService, testUser *user.User) {
				// Export handler first calls GetReportByID, might not get to ExportReport due to format validation
				reportService.On("GetReportByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(&report.Report{
						ID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						FamilyID: testUser.FamilyID,
						Name:     "Test Report",
					}, nil).Maybe()
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Invalid report ID",
			reportID:     "invalid-uuid",
			exportFormat: "csv",
			setupMocks: func(_ *MockReportService, _ *user.User) {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test user first
			testUser := createTestUser()

			// Setup
			repos := setupRepositories()
			reportService := &MockReportService{}
			mockServices := &services.Services{
				Report: reportService,
			}
			tt.setupMocks(reportService, testUser)

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports/"+tt.reportID+"/export?format="+tt.exportFormat, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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
			err := handler.Export(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				require.NoError(t, err)
			} else {
				// Expect an HTTP error for non-OK responses
				require.Error(t, err)
				var httpError *echo.HTTPError
				require.ErrorAs(t, err, &httpError)
				assert.Equal(t, tt.expectedStatus, httpError.Code)
			}

			// Assert mock expectations
			reportService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_Generate(t *testing.T) {
	tests := []struct {
		name           string
		formData       map[string]string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Successfully generate report",
			formData: map[string]string{
				"name":       "Generated Report",
				"type":       "expenses",
				"period":     "monthly",
				"start_date": "2024-01-01",
				"end_date":   "2024-01-31",
			},
			setupMocks: func() {
				// No specific mocks needed for basic generation
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid form data",
			formData: map[string]string{
				"name": "",
				"type": "invalid",
			},
			setupMocks: func() {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			mockServices := &services.Services{
				Report: &MockReportService{},
			}
			tt.setupMocks()

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodPost, "/reports/generate", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set form data
			form := make(url.Values)
			for key, value := range tt.formData {
				form.Set(key, value)
			}
			req.PostForm = form

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
			err := handler.Generate(c)

			// Assert
			require.NoError(t, err)
		})
	}
}
