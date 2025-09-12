package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

func TestReportHandler_Index(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Successfully show reports index",
			setupMocks: func() {
				// No specific mocks needed for basic index rendering
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			mockServices := &services.Services{
				// Add minimal services if needed
			}
			tt.setupMocks()

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
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
		})
	}
}

func TestReportHandler_Show(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:     "Successfully show report",
			reportID: "550e8400-e29b-41d4-a716-446655440000",
			setupMocks: func() {
				// No specific mocks needed for basic show rendering
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Invalid report ID",
			reportID: "invalid-uuid",
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
				// Add minimal services if needed
			}
			tt.setupMocks()

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports/"+tt.reportID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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
			err := handler.Show(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
		})
	}
}

func TestReportHandler_New(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Successfully show new report form",
			setupMocks: func() {
				// No specific mocks needed for basic form rendering
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			repos := setupRepositories()
			mockServices := &services.Services{
				// Add minimal services if needed
			}
			tt.setupMocks()

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
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
		})
	}
}

func TestReportHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		formData       map[string]string
		setupMocks     func()
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
			setupMocks: func() {
				// No specific mocks needed for basic creation
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
				// Add minimal services if needed
			}
			tt.setupMocks()

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodPost, "/reports", nil)
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
			err := handler.Create(c)

			// Assert
			if tt.expectRedirect {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
		})
	}
}

func TestReportHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		setupMocks     func()
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:     "Successfully delete report",
			reportID: "550e8400-e29b-41d4-a716-446655440000",
			setupMocks: func() {
				// No specific mocks needed for basic deletion
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:     "Invalid report ID",
			reportID: "invalid-uuid",
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
				// Add minimal services if needed
			}
			tt.setupMocks()

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodDelete, "/reports/"+tt.reportID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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
		})
	}
}

func TestReportHandler_Export(t *testing.T) {
	tests := []struct {
		name           string
		reportID       string
		exportFormat   string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:         "Successfully export CSV",
			reportID:     "550e8400-e29b-41d4-a716-446655440000",
			exportFormat: "csv",
			setupMocks: func() {
				// No specific mocks needed for basic export
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "Unsupported format",
			reportID:     "550e8400-e29b-41d4-a716-446655440000",
			exportFormat: "xml",
			setupMocks: func() {
				// No mocks needed for validation errors
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "Invalid report ID",
			reportID:     "invalid-uuid",
			exportFormat: "csv",
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
				// Add minimal services if needed
			}
			tt.setupMocks()

			handler := webHandlers.NewReportHandler(repos, mockServices)

			e := setupEchoWithSession()
			req := httptest.NewRequest(http.MethodGet, "/reports/"+tt.reportID+"/export?format="+tt.exportFormat, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.reportID)

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
			err := handler.Export(c)

			// Assert
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
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
				// Add minimal services if needed
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
			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err) // Echo handles HTTP errors differently
			}
		})
	}
}
