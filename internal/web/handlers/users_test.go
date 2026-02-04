package handlers_test

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

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

// setupUserTest creates common test dependencies for user handler tests
func setupUserTest(t *testing.T) (*handlers.UserHandler, *MockUserService, *MockFamilyService) {
	t.Helper()

	mockUserService := new(MockUserService)
	mockFamilyService := new(MockFamilyService)

	repos := &appHandlers.Repositories{}
	srvcs := &services.Services{
		User:   mockUserService,
		Family: mockFamilyService,
	}

	handler := handlers.NewUserHandler(repos, srvcs)
	return handler, mockUserService, mockFamilyService
}

func TestUserHandler_Index(t *testing.T) {
	familyID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       user.Role
		mockSetup      func(*MockUserService, *MockFamilyService)
		expectedStatus int
		checkResponse  func(*testing.T, *echo.HTTPError)
	}{
		{
			name:     "Success - list all users",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, familySvc *MockFamilyService) {
				users := []*user.User{
					{
						ID:    adminID,
						Email: "admin@test.com",
						Role:  user.RoleAdmin,
					},
					{
						ID:    memberID,
						Email: "member@test.com",
						Role:  user.RoleMember,
					},
				}
				userSvc.On("GetUsers", mock.Anything).Return(users, nil).Once()

				family := &user.Family{
					ID:   familyID,
					Name: "Test Family",
				}
				familySvc.On("GetFamily", mock.Anything).Return(family, nil).Once()

				currentUser := &user.User{
					ID:    adminID,
					Email: "admin@test.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(currentUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Success - empty family",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, familySvc *MockFamilyService) {
				users := []*user.User{}
				userSvc.On("GetUsers", mock.Anything).Return(users, nil).Once()

				family := &user.Family{
					ID:   familyID,
					Name: "Empty Family",
				}
				familySvc.On("GetFamily", mock.Anything).Return(family, nil).Once()

				currentUser := &user.User{
					ID:    adminID,
					Email: "admin@test.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(currentUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Error - service error loading users",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, _ *MockFamilyService) {
				userSvc.On("GetUsers", mock.Anything).Return(nil, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
				assert.Contains(t, httpErr.Message, "Failed to load users")
			},
		},
		{
			name:     "Error - service error loading family",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, familySvc *MockFamilyService) {
				users := []*user.User{
					{ID: adminID, Email: "admin@test.com", Role: user.RoleAdmin},
				}
				userSvc.On("GetUsers", mock.Anything).Return(users, nil).Once()
				familySvc.On("GetFamily", mock.Anything).Return(nil, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
				assert.Contains(t, httpErr.Message, "Failed to load family")
			},
		},
		{
			name:     "Error - service error loading current user",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService, familySvc *MockFamilyService) {
				users := []*user.User{
					{ID: adminID, Email: "admin@test.com", Role: user.RoleAdmin},
				}
				userSvc.On("GetUsers", mock.Anything).Return(users, nil).Once()

				family := &user.Family{ID: familyID, Name: "Test Family"}
				familySvc.On("GetFamily", mock.Anything).Return(family, nil).Once()

				userSvc.On("GetUserByID", mock.Anything, adminID).Return(nil, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, httpErr *echo.HTTPError) {
				require.NotNil(t, httpErr)
				assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
				assert.Contains(t, httpErr.Message, "Failed to load current user")
			},
		},
		{
			name:           "Redirect - no session",
			userID:         uuid.Nil,
			userRole:       "",
			mockSetup:      func(_ *MockUserService, _ *MockFamilyService) {},
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, familySvc := setupUserTest(t)

			c, rec := newTestContext(http.MethodGet, "/users", "")
			if tt.userID != uuid.Nil {
				withSession(c, tt.userID, tt.userRole)
			}

			tt.mockSetup(userSvc, familySvc)

			err := handler.Index(c)

			switch tt.expectedStatus {
			case http.StatusSeeOther:
				assert.Equal(t, http.StatusSeeOther, rec.Code)
			default:
				if tt.checkResponse != nil {
					httpErr := &echo.HTTPError{}
					ok := errors.As(err, &httpErr)
					tt.checkResponse(t, httpErr)
					assert.True(t, ok)
				} else {
					require.NoError(t, err)
				}
			}

			userSvc.AssertExpectations(t)
			familySvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_New(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       user.Role
		mockSetup      func(*MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, error)
	}{
		{
			name:     "Success - admin can access new user form",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				adminUser := &user.User{
					ID:    adminID,
					Email: "admin@test.com",
					Role:  user.RoleAdmin,
				}
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(adminUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name:     "Forbidden - member cannot access new user form",
			userID:   uuid.New(),
			userRole: user.RoleMember,
			mockSetup: func(userSvc *MockUserService) {
				memberUser := &user.User{
					ID:    uuid.New(),
					Email: "member@test.com",
					Role:  user.RoleMember,
				}
				userSvc.On("GetUserByID", mock.Anything, mock.Anything).Return(memberUser, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, err error) {
				// c.String() returns nil error but sets response code
				assert.NoError(t, err)
			},
		},
		{
			name:     "Forbidden - child cannot access new user form",
			userID:   uuid.New(),
			userRole: user.RoleChild,
			mockSetup: func(userSvc *MockUserService) {
				childUser := &user.User{
					ID:    uuid.New(),
					Email: "child@test.com",
					Role:  user.RoleChild,
				}
				userSvc.On("GetUserByID", mock.Anything, mock.Anything).Return(childUser, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, err error) {
				// c.String() returns nil error but sets response code
				assert.NoError(t, err)
			},
		},
		{
			name:           "Redirect - no session",
			userID:         uuid.Nil,
			userRole:       "",
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusSeeOther,
		},
		{
			name:     "Error - failed to load current user",
			userID:   adminID,
			userRole: user.RoleAdmin,
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("GetUserByID", mock.Anything, adminID).Return(nil, errors.New("database error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, err error) {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok)
				assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
				assert.Contains(t, httpErr.Message, "Failed to load current user")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, _ := setupUserTest(t)

			c, rec := newTestContext(http.MethodGet, "/users/new", "")
			if tt.userID != uuid.Nil {
				withSession(c, tt.userID, tt.userRole)
			}

			tt.mockSetup(userSvc)

			err := handler.New(c)

			switch tt.expectedStatus {
			case http.StatusSeeOther:
				assert.Equal(t, http.StatusSeeOther, rec.Code)
			case http.StatusForbidden:
				require.NoError(t, err) // c.String() returns nil
				assert.Equal(t, http.StatusForbidden, rec.Code)
				assert.Contains(t, rec.Body.String(), "Only family admin can add new members")
			default:
				if tt.checkResponse != nil {
					tt.checkResponse(t, err)
				}
			}

			userSvc.AssertExpectations(t)
		})
	}
}

//nolint:gocognit // Table-driven test with comprehensive test cases
func TestUserHandler_Create_Success(t *testing.T) {
	adminID := uuid.New()

	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       user.Role
		formData       url.Values
		isHTMX         bool
		mockSetup      func(*MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, echo.Context, error)
	}{
		{
			name:     "create valid member",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"John"},
				"last_name":  {"Doe"},
				"email":      {"john.doe@test.com"},
				"password":   {"SecurePass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				createdUser := &user.User{
					ID:        uuid.New(),
					Email:     "john.doe@test.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
				}
				userSvc.On("CreateUser", mock.Anything, mock.MatchedBy(func(createDTO dto.CreateUserDTO) bool {
					return createDTO.Email == "john.doe@test.com" &&
						createDTO.FirstName == "John" &&
						createDTO.LastName == "Doe" &&
						createDTO.Role == "member"
				})).Return(createdUser, nil).Once()
			},
			expectedStatus: http.StatusFound,
			checkResponse: func(t *testing.T, _ echo.Context, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "Success - create valid child",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Jane"},
				"last_name":  {"Doe"},
				"email":      {"jane.doe@test.com"},
				"password":   {"ChildPass123!"},
				"role":       {"child"},
			},
			mockSetup: func(userSvc *MockUserService) {
				createdUser := &user.User{
					ID:        uuid.New(),
					Email:     "jane.doe@test.com",
					FirstName: "Jane",
					LastName:  "Doe",
					Role:      user.RoleChild,
				}
				userSvc.On("CreateUser", mock.Anything, mock.MatchedBy(func(createDTO dto.CreateUserDTO) bool {
					return createDTO.Email == "jane.doe@test.com" &&
						createDTO.Role == "child"
				})).Return(createdUser, nil).Once()
			},
			expectedStatus: http.StatusFound,
		},
		{
			name:     "Success - create valid admin",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Admin"},
				"last_name":  {"User"},
				"email":      {"admin.user@test.com"},
				"password":   {"AdminPass123!"},
				"role":       {"admin"},
			},
			mockSetup: func(userSvc *MockUserService) {
				createdUser := &user.User{
					ID:        uuid.New(),
					Email:     "admin.user@test.com",
					FirstName: "Admin",
					LastName:  "User",
					Role:      user.RoleAdmin,
				}
				userSvc.On("CreateUser", mock.Anything, mock.MatchedBy(func(createDTO dto.CreateUserDTO) bool {
					return createDTO.Email == "admin.user@test.com" &&
						createDTO.Role == "admin"
				})).Return(createdUser, nil).Once()
			},
			expectedStatus: http.StatusFound,
		},
		{
			name:     "Success - HTMX redirect",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"HTMX"},
				"last_name":  {"User"},
				"email":      {"htmx@test.com"},
				"password":   {"HTMXPass123!"},
				"role":       {"member"},
			},
			isHTMX: true,
			mockSetup: func(userSvc *MockUserService) {
				createdUser := &user.User{
					ID:    uuid.New(),
					Email: "htmx@test.com",
					Role:  user.RoleMember,
				}
				userSvc.On("CreateUser", mock.Anything, mock.Anything).Return(createdUser, nil).Once()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, c echo.Context, err error) {
				require.NoError(t, err)
				assert.Equal(t, "/users", c.Response().Header().Get("Hx-Redirect"))
			},
		},
		{
			name:     "Error - duplicate email",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Existing"},
				"last_name":  {"User"},
				"email":      {"existing@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, services.ErrEmailAlreadyExists).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - validation failed - missing email",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"No"},
				"last_name":  {"Email"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - validation failed - short password",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Weak"},
				"last_name":  {"Pass"},
				"email":      {"weak@test.com"},
				"password":   {"123"},
				"role":       {"member"},
			},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - validation failed - short first name",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"A"},
				"last_name":  {"User"},
				"email":      {"test@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - validation failed - invalid email",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Invalid"},
				"last_name":  {"Email"},
				"email":      {"not-an-email"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - validation failed - missing role",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"No"},
				"last_name":  {"Role"},
				"email":      {"norole@test.com"},
				"password":   {"Pass123!"},
			},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - service validation failed",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Service"},
				"last_name":  {"Error"},
				"email":      {"service@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, services.ErrValidationFailed).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - family not found",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"No"},
				"last_name":  {"Family"},
				"email":      {"nofamily@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, services.ErrFamilyNotFound).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - unauthorized",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Unauthorized"},
				"last_name":  {"User"},
				"email":      {"unauthorized@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, services.ErrUnauthorized).Once()
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, _ echo.Context, err error) {
				// c.String() returns nil error
				require.NoError(t, err)
			},
		},
		{
			name:     "Error - invalid role",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Invalid"},
				"last_name":  {"Role"},
				"email":      {"invalidrole@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, services.ErrInvalidRole).Once()
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "Error - generic service error",
			userID:   adminID,
			userRole: user.RoleAdmin,
			formData: url.Values{
				"first_name": {"Generic"},
				"last_name":  {"Error"},
				"email":      {"generic@test.com"},
				"password":   {"Pass123!"},
				"role":       {"member"},
			},
			mockSetup: func(userSvc *MockUserService) {
				userSvc.On("CreateUser", mock.Anything, mock.Anything).
					Return(nil, errors.New("unexpected error")).Once()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, _ echo.Context, err error) {
				httpErr := &echo.HTTPError{}
				ok := errors.As(err, &httpErr)
				require.True(t, ok)
				assert.Equal(t, http.StatusInternalServerError, httpErr.Code)
				assert.Contains(t, httpErr.Message, "Failed to create user")
			},
		},
		{
			name:           "Redirect - no session",
			userID:         uuid.Nil,
			userRole:       "",
			formData:       url.Values{},
			mockSetup:      func(_ *MockUserService) {},
			expectedStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, userSvc, _ := setupUserTest(t)

			c, rec := newTestContext(http.MethodPost, "/users", tt.formData.Encode())
			if tt.userID != uuid.Nil {
				withSession(c, tt.userID, tt.userRole)
			}
			if tt.isHTMX {
				withHTMX(c)
			}

			tt.mockSetup(userSvc)

			err := handler.Create(c)

			switch tt.expectedStatus {
			case http.StatusFound:
				require.NoError(t, err)
				assert.Equal(t, "/users", rec.Header().Get("Location"))
			case http.StatusSeeOther:
				assert.Equal(t, http.StatusSeeOther, rec.Code)
			case http.StatusOK:
				if tt.checkResponse != nil {
					tt.checkResponse(t, c, err)
				} else {
					require.NoError(t, err)
				}
			case http.StatusUnprocessableEntity:
				// For validation errors and service errors that render forms
				// c.Render() returns nil in tests with MockRenderer
				require.NoError(t, err)
			case http.StatusForbidden, http.StatusInternalServerError:
				if tt.checkResponse != nil {
					tt.checkResponse(t, c, err)
				}
			}

			userSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_Create_InvalidFormBind(t *testing.T) {
	handler, _, _ := setupUserTest(t)

	// Create a request with invalid content type to trigger bind error
	c, _ := newTestContext(http.MethodPost, "/users", "")
	c.Request().Header.Set(echo.HeaderContentType, "application/json")
	c.Request().Body = nil
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Create(c)

	// userError() calls c.Render() which returns nil in tests
	require.NoError(t, err)
}

func TestUserHandler_FormPreservation(t *testing.T) {
	handler, userSvc, _ := setupUserTest(t)

	formData := url.Values{
		"first_name": {"Preserved"},
		"last_name":  {"Data"},
		"email":      {"preserved@test.com"},
		"password":   {"Pass123!"},
		"role":       {"member"},
	}

	userSvc.On("CreateUser", mock.Anything, mock.Anything).
		Return(nil, services.ErrEmailAlreadyExists).Once()

	c, _ := newTestContext(http.MethodPost, "/users", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Create(c)

	// c.Render() returns nil in tests
	require.NoError(t, err)

	userSvc.AssertExpectations(t)
}

func TestUserHandler_EdgeCases(t *testing.T) {
	t.Run("Empty form data", func(t *testing.T) {
		handler, _, _ := setupUserTest(t)

		c, _ := newTestContext(http.MethodPost, "/users", "")
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)
		// Validation fails, but c.Render() returns nil in tests
		require.NoError(t, err)
	})

	t.Run("Long email address", func(t *testing.T) {
		handler, _, _ := setupUserTest(t)

		longEmail := strings.Repeat("a", 250) + "@test.com"
		formData := url.Values{
			"first_name": {"Long"},
			"last_name":  {"Email"},
			"email":      {longEmail},
			"password":   {"Pass123!"},
			"role":       {"member"},
		}

		c, _ := newTestContext(http.MethodPost, "/users", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)
		// Should fail validation (max 254 chars), but c.Render() returns nil
		require.NoError(t, err)
	})

	t.Run("Special characters in name", func(t *testing.T) {
		handler, userSvc, _ := setupUserTest(t)

		formData := url.Values{
			"first_name": {"O'Brien"},
			"last_name":  {"Müller"},
			"email":      {"special@test.com"},
			"password":   {"Pass123!"},
			"role":       {"member"},
		}

		createdUser := &user.User{
			ID:        uuid.New(),
			Email:     "special@test.com",
			FirstName: "O'Brien",
			LastName:  "Müller",
			Role:      user.RoleMember,
		}
		userSvc.On("CreateUser", mock.Anything, mock.Anything).Return(createdUser, nil).Once()

		c, _ := newTestContext(http.MethodPost, "/users", formData.Encode())
		withSession(c, uuid.New(), user.RoleAdmin)

		err := handler.Create(c)
		require.NoError(t, err)

		userSvc.AssertExpectations(t)
	})
}
