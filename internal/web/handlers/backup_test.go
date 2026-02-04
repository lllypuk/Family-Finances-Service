package handlers_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appHandlers "family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/handlers"
)

// setupBackupTest creates common test dependencies for backup handler tests
func setupBackupTest(t *testing.T) (*handlers.BackupHandler, *MockUserService, *MockBackupService) {
	t.Helper()

	mockUserService := new(MockUserService)
	mockFamilyService := new(MockFamilyService)
	mockBackupService := new(MockBackupService)

	repos := &appHandlers.Repositories{}
	srvcs := &services.Services{
		User:   mockUserService,
		Family: mockFamilyService,
		Backup: mockBackupService,
	}

	handler := handlers.NewBackupHandler(repos, srvcs)
	return handler, mockUserService, mockBackupService
}

func TestBackupHandler_BackupPage(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       user.Role
		mockSetup      func(*MockUserService, *MockBackupService)
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:     "Redirect - non-admin cannot access",
			userID:   adminID,
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:           "Redirect - no session",
			userID:         uuid.Nil,
			userRole:       "",
			mockSetup:      func(_ *MockUserService, _ *MockBackupService) {},
			expectedStatus: http.StatusSeeOther,
			expectRedirect: true,
		},
		{
			name:     "Success - displays backup page",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:        adminID,
					Email:     "admin@example.com",
					FirstName: "Admin",
					LastName:  "User",
					Role:      user.RoleAdmin,
				}
				// First call in requireAdmin
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				// Second call to get user details for navigation
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("ListBackups", mock.Anything).Return([]*services.BackupInfo{}, nil).Once()
			},
			expectedStatus: http.StatusOK,
			expectRedirect: false,
		},
		{
			name:     "Error - failed to list backups",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:        adminID,
					Email:     "admin@example.com",
					FirstName: "Admin",
					LastName:  "User",
					Role:      user.RoleAdmin,
				}
				// First call in requireAdmin
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				// Second call to get user details for navigation
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("ListBackups", mock.Anything).Return(nil, errors.New("disk error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, backupSvc := setupBackupTest(t)

			c, rec := newTestContext(http.MethodGet, "/admin/backup", "")
			if tt.userID != uuid.Nil {
				withSession(c, tt.userID, tt.userRole)
			}
			tt.mockSetup(userSvc, backupSvc)

			err := handler.BackupPage(c)

			switch {
			case tt.expectRedirect:
				assert.Equal(t, http.StatusSeeOther, rec.Code)
			case tt.expectedStatus >= 400:
				// For errors, check the error object
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
			userSvc.AssertExpectations(t)
			backupSvc.AssertExpectations(t)
		})
	}
}

func TestBackupHandler_CreateBackup(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		userRole       user.Role
		isHTMX         bool
		mockSetup      func(*MockUserService, *MockBackupService)
		expectedStatus int
		checkResponse  func(*testing.T, error, *echo.HTTPError, echo.Context)
	}{
		{
			name:     "Success - create backup (HTMX)",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				backupInfo := &services.BackupInfo{
					Filename:  "budget_20240103_120000.db",
					Size:      1024000,
					CreatedAt: time.Now(),
				}
				backupSvc.On("CreateBackup", mock.Anything).Return(backupInfo, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Success - create backup (non-HTMX redirect)",
			userRole: user.RoleAdmin,
			isHTMX:   false,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				backupInfo := &services.BackupInfo{
					Filename:  "budget_20240103_120000.db",
					Size:      1024000,
					CreatedAt: time.Now(),
				}
				backupSvc.On("CreateBackup", mock.Anything).Return(backupInfo, nil).Once()
			},
			expectedStatus: http.StatusSeeOther,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Error - backup creation failed (HTMX)",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("CreateBackup", mock.Anything).Return(nil, errors.New("disk full")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Failed to create backup", httpErr.Message)
			},
		},
		{
			name:     "Error - non-admin forbidden",
			userRole: user.RoleMember,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, backupSvc := setupBackupTest(t)

			c, rec := newTestContext(http.MethodPost, "/admin/backup", "")
			withSession(c, adminID, tt.userRole)
			if tt.isHTMX {
				withHTMX(c)
			}
			tt.mockSetup(userSvc, backupSvc)

			err := handler.CreateBackup(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, httpErr, c)
				}
			} else {
				if tt.expectedStatus == http.StatusSeeOther {
					assert.Equal(t, http.StatusSeeOther, rec.Code)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
			backupSvc.AssertExpectations(t)
		})
	}
}

func TestBackupHandler_DownloadBackup(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		filename       string
		userRole       user.Role
		mockSetup      func(*MockUserService, *MockBackupService)
		expectedStatus int
		checkResponse  func(*testing.T, error, *echo.HTTPError)
	}{
		{
			name:     "Error - backup not found",
			filename: "nonexistent.db",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("GetBackup", mock.Anything, "nonexistent.db").
					Return(nil, services.ErrBackupNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Backup not found", httpErr.Message)
			},
		},
		{
			name:     "Error - invalid filename",
			filename: "../../../etc/passwd",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("GetBackup", mock.Anything, "../../../etc/passwd").
					Return(nil, services.ErrInvalidBackupFilename).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Invalid filename", httpErr.Message)
			},
		},
		{
			name:     "Error - non-admin forbidden",
			filename: "backup.db",
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
		{
			name:     "Error - empty filename",
			filename: "",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Filename is required", httpErr.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, backupSvc := setupBackupTest(t)

			c, _ := newTestContext(http.MethodGet, "/admin/backup/download/"+tt.filename, "")
			c.SetParamNames("filename")
			c.SetParamValues(tt.filename)
			withSession(c, adminID, tt.userRole)
			tt.mockSetup(userSvc, backupSvc)

			err := handler.DownloadBackup(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, httpErr)
				}
			}

			userSvc.AssertExpectations(t)
			backupSvc.AssertExpectations(t)
		})
	}
}

func TestBackupHandler_DeleteBackup(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		filename       string
		userRole       user.Role
		isHTMX         bool
		mockSetup      func(*MockUserService, *MockBackupService)
		expectedStatus int
		checkResponse  func(*testing.T, error, *echo.HTTPError, echo.Context)
	}{
		{
			name:     "Success - delete backup (HTMX)",
			filename: "budget_20240101_120000.db",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("DeleteBackup", mock.Anything, "budget_20240101_120000.db").
					Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Success - delete backup (non-HTMX redirect)",
			filename: "budget_20240101_120000.db",
			userRole: user.RoleAdmin,
			isHTMX:   false,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("DeleteBackup", mock.Anything, "budget_20240101_120000.db").
					Return(nil).Once()
			},
			expectedStatus: http.StatusSeeOther,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Error - backup not found",
			filename: "nonexistent.db",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("DeleteBackup", mock.Anything, "nonexistent.db").
					Return(services.ErrBackupNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Backup not found", httpErr.Message)
			},
		},
		{
			name:     "Error - invalid filename",
			filename: "../../../etc/passwd",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("DeleteBackup", mock.Anything, "../../../etc/passwd").
					Return(services.ErrInvalidBackupFilename).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Invalid filename", httpErr.Message)
			},
		},
		{
			name:     "Error - non-admin forbidden",
			filename: "backup.db",
			userRole: user.RoleMember,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, backupSvc := setupBackupTest(t)

			c, rec := newTestContext(http.MethodDelete, "/admin/backup/"+tt.filename, "")
			c.SetParamNames("filename")
			c.SetParamValues(tt.filename)
			withSession(c, adminID, tt.userRole)
			if tt.isHTMX {
				withHTMX(c)
			}
			tt.mockSetup(userSvc, backupSvc)

			err := handler.DeleteBackup(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, httpErr, c)
				}
			} else {
				if tt.expectedStatus == http.StatusSeeOther {
					assert.Equal(t, http.StatusSeeOther, rec.Code)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
			backupSvc.AssertExpectations(t)
		})
	}
}

func TestBackupHandler_RestoreBackup(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		filename       string
		userRole       user.Role
		isHTMX         bool
		mockSetup      func(*MockUserService, *MockBackupService)
		expectedStatus int
		checkResponse  func(*testing.T, error, *echo.HTTPError, echo.Context)
	}{
		{
			name:     "Success - restore backup (HTMX)",
			filename: "budget_20240101_120000.db",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("RestoreBackup", mock.Anything, "budget_20240101_120000.db").
					Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Success - restore backup (non-HTMX redirect)",
			filename: "budget_20240101_120000.db",
			userRole: user.RoleAdmin,
			isHTMX:   false,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("RestoreBackup", mock.Anything, "budget_20240101_120000.db").
					Return(nil).Once()
			},
			expectedStatus: http.StatusSeeOther,
			checkResponse: func(t *testing.T, err error, _ *echo.HTTPError, _ echo.Context) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Error - backup not found",
			filename: "nonexistent.db",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("RestoreBackup", mock.Anything, "nonexistent.db").
					Return(services.ErrBackupNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Backup not found", httpErr.Message)
			},
		},
		{
			name:     "Error - invalid filename",
			filename: "../../../etc/passwd",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("RestoreBackup", mock.Anything, "../../../etc/passwd").
					Return(services.ErrInvalidBackupFilename).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Invalid filename", httpErr.Message)
			},
		},
		{
			name:     "Error - non-admin forbidden",
			filename: "backup.db",
			userRole: user.RoleMember,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, _ *MockBackupService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
		{
			name:     "Error - restore failed",
			filename: "backup.db",
			userRole: user.RoleAdmin,
			isHTMX:   true,
			mockSetup: func(userSvc *MockUserService, backupSvc *MockBackupService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				backupSvc.On("RestoreBackup", mock.Anything, "backup.db").
					Return(errors.New("database locked")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, _ error, httpErr *echo.HTTPError, _ echo.Context) {
				require.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message.(string), "Failed to restore backup")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, backupSvc := setupBackupTest(t)

			c, rec := newTestContext(http.MethodPost, "/admin/backup/restore/"+tt.filename, "")
			c.SetParamNames("filename")
			c.SetParamValues(tt.filename)
			withSession(c, adminID, tt.userRole)
			if tt.isHTMX {
				withHTMX(c)
			}
			tt.mockSetup(userSvc, backupSvc)

			err := handler.RestoreBackup(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, httpErr, c)
				}
			} else {
				if tt.expectedStatus == http.StatusSeeOther {
					assert.Equal(t, http.StatusSeeOther, rec.Code)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, err, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
			backupSvc.AssertExpectations(t)
		})
	}
}
