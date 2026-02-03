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
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/handlers"
)

// setupAdminTest creates common test dependencies for admin handler tests
func setupAdminTest(t *testing.T) (*handlers.AdminHandler, *MockUserService, *MockInviteService) {
	t.Helper()

	mockUserService := new(MockUserService)
	mockFamilyService := new(MockFamilyService)
	mockInviteService := new(MockInviteService)

	repos := &appHandlers.Repositories{}
	srvcs := &services.Services{
		User:   mockUserService,
		Family: mockFamilyService,
		Invite: mockInviteService,
	}

	handler := handlers.NewAdminHandler(repos, srvcs)
	return handler, mockUserService, mockInviteService
}

func TestAdminHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       user.Role
		mockSetup      func(*MockUserService, uuid.UUID)
		expectRedirect bool
	}{
		{
			name:     "Redirect - non-admin cannot access",
			userID:   uuid.New(),
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService, id uuid.UUID) {
				memberUser := &user.User{
					ID:    id,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, id).Return(memberUser, nil).Once()
			},
			expectRedirect: true,
		},
		{
			name:           "Redirect - no session",
			userID:         uuid.Nil,
			userRole:       "",
			mockSetup:      func(_ *MockUserService, _ uuid.UUID) {},
			expectRedirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, _ := setupAdminTest(t)

			c, rec := newTestContext(http.MethodGet, "/admin/users", "")
			if tt.userID != uuid.Nil {
				withSession(c, tt.userID, tt.userRole)
			}
			tt.mockSetup(userSvc, tt.userID)

			_ = handler.ListUsers(c)

			if tt.expectRedirect {
				assert.Equal(t, http.StatusSeeOther, rec.Code)
			}

			userSvc.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_CreateInvite(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		formData       string
		userRole       user.Role
		mockSetup      func(*MockUserService, *MockInviteService)
		expectedStatus int
		checkResponse  func(*testing.T, *echo.HTTPError, echo.Context)
	}{
		{
			name:     "Success - create valid invite",
			formData: "email=newuser@example.com&role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				invite := &user.Invite{
					ID:        uuid.New(),
					Email:     "newuser@example.com",
					Role:      user.RoleMember,
					Status:    user.InviteStatusPending,
					Token:     "test-token",
					ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
					CreatedAt: time.Now(),
				}
				inviteSvc.On("CreateInvite", mock.Anything, adminID, mock.MatchedBy(func(dto dto.CreateInviteDTO) bool {
					return dto.Email == "newuser@example.com" && dto.Role == "member"
				})).Return(invite, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, c echo.Context) {
				assert.Nil(t, httpErr)
				assert.Equal(t, "inviteCreated", c.Response().Header().Get("Hx-Trigger"))
			},
		},
		{
			name:     "Error - duplicate email",
			formData: "email=existing@example.com&role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("CreateInvite", mock.Anything, adminID, mock.Anything).
					Return(nil, services.ErrEmailAlreadyExists).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "User with this email already exists")
			},
		},
		{
			name:     "Error - pending invite exists",
			formData: "email=pending@example.com&role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("CreateInvite", mock.Anything, adminID, mock.Anything).
					Return(nil, errors.New("pending invite already exists")).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "Pending invite already exists")
			},
		},
		{
			name:     "Error - validation error - missing email",
			formData: "role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "email is required")
			},
		},
		{
			name:     "Error - validation error - missing role",
			formData: "email=test@example.com",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "role is required")
			},
		},
		{
			name:     "Error - validation error - invalid email",
			formData: "email=invalid-email&role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				// Service layer should catch the invalid email
				inviteSvc.On("CreateInvite", mock.Anything, adminID, mock.Anything).
					Return(nil, errors.New("invalid email format")).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "Failed to create invite")
			},
		},
		{
			name:     "Error - validation error - invalid role",
			formData: "email=test@example.com&role=superuser",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message, "invalid role")
			},
		},
		{
			name:     "Error - non-admin forbidden",
			formData: "email=test@example.com&role=member",
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
		{
			name:     "Success - email normalization",
			formData: "email=  NewUser@Example.COM  &role=member",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				invite := &user.Invite{
					ID:        uuid.New(),
					Email:     "newuser@example.com",
					Role:      user.RoleMember,
					Status:    user.InviteStatusPending,
					CreatedAt: time.Now(),
				}
				inviteSvc.On("CreateInvite", mock.Anything, adminID, mock.MatchedBy(func(dto dto.CreateInviteDTO) bool {
					return dto.Email == "newuser@example.com" // Should be normalized
				})).Return(invite, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.Nil(t, httpErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, inviteSvc := setupAdminTest(t)

			c, rec := newTestContext(http.MethodPost, "/admin/invites", tt.formData)
			withSession(c, adminID, tt.userRole)
			withHTMX(c)
			tt.mockSetup(userSvc, inviteSvc)

			err := handler.CreateInvite(c)

			switch {
			case tt.expectedStatus == http.StatusBadRequest:
				// For 400 errors from htmxError, check response code
				require.NoError(t, err) // htmxError doesn't return error
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			case tt.expectedStatus >= 400:
				// For non-400 errors, we expect HTTP errors
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, httpErr, c)
				}
			default:
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
			inviteSvc.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_RevokeInvite(t *testing.T) {
	adminID := uuid.New()
	validInviteID := uuid.New()

	tests := []struct {
		name           string
		inviteID       string
		userRole       user.Role
		mockSetup      func(*MockUserService, *MockInviteService)
		expectedStatus int
		checkResponse  func(*testing.T, *echo.HTTPError, echo.Context)
	}{
		{
			name:     "Success - revoke invite",
			inviteID: validInviteID.String(),
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("RevokeInvite", mock.Anything, validInviteID, adminID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, c echo.Context) {
				assert.Nil(t, httpErr)
				assert.Equal(t, "inviteRevoked", c.Response().Header().Get("Hx-Trigger"))
			},
		},
		{
			name:     "Error - invite not found",
			inviteID: uuid.New().String(),
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("RevokeInvite", mock.Anything, mock.Anything, adminID).
					Return(services.ErrInviteNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Invite not found", httpErr.Message)
			},
		},
		{
			name:     "Error - unauthorized",
			inviteID: validInviteID.String(),
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("RevokeInvite", mock.Anything, validInviteID, adminID).
					Return(services.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Unauthorized", httpErr.Message)
			},
		},
		{
			name:     "Error - invalid invite ID format",
			inviteID: "invalid-uuid",
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				// requireAdmin is called before ID parsing
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Invalid invite ID", httpErr.Message)
			},
		},
		{
			name:     "Error - non-admin forbidden",
			inviteID: validInviteID.String(),
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService, _ *MockInviteService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
		{
			name:     "Error - service error",
			inviteID: validInviteID.String(),
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, inviteSvc *MockInviteService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				inviteSvc.On("RevokeInvite", mock.Anything, validInviteID, adminID).
					Return(errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message.(string), "Failed to revoke invite")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, inviteSvc := setupAdminTest(t)

			c, _ := newTestContext(http.MethodDelete, "/admin/invites/"+tt.inviteID, "")
			c.SetParamNames("id")
			c.SetParamValues(tt.inviteID)
			withSession(c, adminID, tt.userRole)
			withHTMX(c)
			tt.mockSetup(userSvc, inviteSvc)

			err := handler.RevokeInvite(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, httpErr, c)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
			inviteSvc.AssertExpectations(t)
		})
	}
}

func TestAdminHandler_DeleteUser(t *testing.T) {
	adminID := uuid.New()
	validUserID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		sessionUserID  uuid.UUID
		userRole       user.Role
		mockSetup      func(*MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *echo.HTTPError, echo.Context)
	}{
		{
			name:          "Success - delete user",
			userID:        validUserID.String(),
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				userToDelete := &user.User{
					ID:    validUserID,
					Email: "delete@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, validUserID).Return(userToDelete, nil).Once()
				userSvc.On("DeleteUser", mock.Anything, validUserID).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, c echo.Context) {
				assert.Nil(t, httpErr)
				assert.Equal(t, "userDeleted", c.Response().Header().Get("Hx-Trigger"))
			},
		},
		{
			name:          "Error - cannot delete self",
			userID:        adminID.String(),
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Cannot delete yourself", httpErr.Message)
			},
		},
		{
			name:          "Error - user not found",
			userID:        uuid.New().String(),
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				userSvc.On("GetUserByID", mock.Anything, mock.Anything).
					Return(nil, services.ErrUserNotFound).Once()
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "User not found", httpErr.Message)
			},
		},
		{
			name:          "Error - invalid user ID format",
			userID:        "invalid-uuid",
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				// requireAdmin is called before ID parsing
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Invalid user ID", httpErr.Message)
			},
		},
		{
			name:          "Error - non-admin forbidden",
			userID:        validUserID.String(),
			sessionUserID: adminID,
			userRole:      user.RoleMember,
			mockSetup: func(userSvc *MockUserService) {
				memberUser := &user.User{
					ID:    adminID,
					Email: "member@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Admin access required", httpErr.Message)
			},
		},
		{
			name:          "Error - failed to load user",
			userID:        validUserID.String(),
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
				userSvc.On("GetUserByID", mock.Anything, validUserID).
					Return(nil, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Equal(t, "Failed to load user", httpErr.Message)
			},
		},
		{
			name:          "Error - delete service error",
			userID:        validUserID.String(),
			sessionUserID: adminID,
			userRole:      user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@example.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()

				userToDelete := &user.User{
					ID:    validUserID,
					Email: "delete@example.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, validUserID).Return(userToDelete, nil).Once()
				userSvc.On("DeleteUser", mock.Anything, validUserID).
					Return(errors.New("constraint violation")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError, _ echo.Context) {
				assert.NotNil(t, httpErr)
				assert.Contains(t, httpErr.Message.(string), "Failed to delete user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, _ := setupAdminTest(t)

			c, _ := newTestContext(http.MethodDelete, "/admin/users/"+tt.userID, "")
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)
			withSession(c, tt.sessionUserID, tt.userRole)
			withHTMX(c)
			tt.mockSetup(userSvc)

			err := handler.DeleteUser(c)

			if tt.expectedStatus >= 400 {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok, "Expected HTTP error")
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.checkResponse != nil {
					tt.checkResponse(t, httpErr, c)
				}
			} else {
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, nil, c)
				}
			}

			userSvc.AssertExpectations(t)
		})
	}
}
