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
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// MockUserRepository is a mock implementation of user repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
	return args.Error(0)
}

// MockUserService provides mock implementation for UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUsers(ctx context.Context) ([]*user.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) SetActive(ctx context.Context, id uuid.UUID, active bool, actorID uuid.UUID) error {
	args := m.Called(ctx, id, active, actorID)
	return args.Error(0)
}

func (m *MockUserService) ChangeUserRole(ctx context.Context, id uuid.UUID, newRole user.Role) error {
	args := m.Called(ctx, id, newRole)
	return args.Error(0)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) ValidateUserAccess(ctx context.Context, userID uuid.UUID, targetFamilyID uuid.UUID) error {
	args := m.Called(ctx, userID, targetFamilyID)
	return args.Error(0)
}

func TestUserHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		mockSetup      func(*MockUserService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name: "Success - Valid user creation",
			requestBody: handlers.CreateUserRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				testUser := &user.User{
					ID:        uuid.New(),
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				service.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserDTO")).Return(testUser, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.UserResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "test@example.com", response.Data.Email)
				assert.Equal(t, "John", response.Data.FirstName)
				assert.Equal(t, "Doe", response.Data.LastName)
				assert.Equal(t, "member", response.Data.Role)
			},
		},
		{
			name: "Error - Request validation failed",
			requestBody: handlers.CreateUserRequest{
				Email:     "invalid-email", // Invalid email format
				Password:  "123",           // Too short password
				FirstName: "",              // Empty first name
				LastName:  "Doe",
				Role:      "member",
			},
			mockSetup:      func(_ *MockUserService, _ uuid.UUID) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
		{
			name: "Error - Missing required fields",
			requestBody: handlers.CreateUserRequest{
				Email: "test@example.com",
				// Missing password, name, etc.
			},
			mockSetup:      func(_ *MockUserService, _ uuid.UUID) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
		{
			name: "Error - Invalid email format",
			requestBody: handlers.CreateUserRequest{
				Email:     "invalid-email",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "member",
			},
			mockSetup:      func(_ *MockUserService, _ uuid.UUID) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
		{
			name: "Error - Unknown role",
			requestBody: handlers.CreateUserRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "superadmin",
			},
			mockSetup:      func(_ *MockUserService, _ uuid.UUID) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
				require.Len(t, response.Error.Details, 1)
				assert.Equal(t, "role", response.Error.Details[0].Field)
			},
		},
		{
			name: "Error - Repository creation fails",
			requestBody: handlers.CreateUserRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				service.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserDTO")).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockService := &MockUserService{}
			tt.mockSetup(mockService, uuid.New())

			repositories := &handlers.Repositories{}
			handler := handlers.NewUserHandler(repositories, mockService, newFakeAuthService())

			// Create request
			var reqBody []byte
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, _ = json.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			err := handler.CreateUser(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetUserByID(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:   "Success - User found",
			userID: userID.String(),
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				user := &user.User{
					ID:        userID,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				service.On("GetUserByID", mock.Anything, userID).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.UserResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, userID, response.Data.ID)
				assert.Equal(t, "test@example.com", response.Data.Email)
			},
		},
		{
			name:   "Error - Invalid UUID format",
			userID: "invalid-uuid",
			mockSetup: func(_ *MockUserService, _ uuid.UUID) {
				// No mock setup needed for UUID validation error
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
			name:   "Error - User not found",
			userID: userID.String(),
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				service.On("GetUserByID", mock.Anything, userID).Return(nil, services.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockService := &MockUserService{}
			tt.mockSetup(mockService, uuid.New())

			repositories := &handlers.Repositories{}
			handler := handlers.NewUserHandler(repositories, mockService, newFakeAuthService())

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Execute
			err := handler.GetUserByID(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		requestBody    any
		mockSetup      func(*MockUserService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:   "Success - User updated",
			userID: userID.String(),
			requestBody: handlers.UpdateUserRequest{
				FirstName: new("UpdatedName"),
				LastName:  new("UpdatedLastName"),
			},
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				updatedUser := &user.User{
					ID:        userID,
					Email:     "test@example.com",
					FirstName: "UpdatedName",
					LastName:  "UpdatedLastName",
					Role:      user.RoleMember,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				service.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("dto.UpdateUserDTO")).
					Return(updatedUser, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.UserResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "UpdatedName", response.Data.FirstName)
				assert.Equal(t, "UpdatedLastName", response.Data.LastName)
			},
		},
		{
			name:   "Error - Invalid UUID",
			userID: "invalid-uuid",
			requestBody: handlers.UpdateUserRequest{
				FirstName: new("UpdatedName"),
			},
			mockSetup: func(_ *MockUserService, _ uuid.UUID) {
				// No mock needed for UUID validation error
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
			name:   "Error - User not found",
			userID: userID.String(),
			requestBody: handlers.UpdateUserRequest{
				FirstName: new("UpdatedName"),
			},
			mockSetup: func(service *MockUserService, _ uuid.UUID) {
				service.On("UpdateUser", mock.Anything, userID, mock.AnythingOfType("dto.UpdateUserDTO")).
					Return(nil, services.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockService := &MockUserService{}
			tt.mockSetup(mockService, uuid.New())

			repositories := &handlers.Repositories{}
			handler := handlers.NewUserHandler(repositories, mockService, newFakeAuthService())

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewReader(jsonBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Execute
			err := handler.UpdateUser(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}

func listUsersRequest(t *testing.T, service *MockUserService) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	handler := handlers.NewUserHandler(&handlers.Repositories{}, service, newFakeAuthService())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, handler.GetUsers(e.NewContext(req, rec)))

	return rec
}

// patchUserRequest прогоняет PATCH /api/v1/users/:id от имени сессии actorID (uuid.Nil — без сессии).
func patchUserRequest(
	t *testing.T, service *MockUserService, targetID, body string, actorID uuid.UUID,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	handler := handlers.NewUserHandler(&handlers.Repositories{}, service, newFakeAuthService())

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+targetID, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(targetID)
	if actorID != uuid.Nil {
		withSessionUser(c, actorID)
	}

	require.NoError(t, handler.PatchUser(c))

	return rec
}

func setUserPasswordRequest(
	t *testing.T, svc *fakeAuthService, targetID, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	handler := handlers.NewUserHandler(&handlers.Repositories{}, &MockUserService{}, svc)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+targetID+"/password", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(targetID)

	require.NoError(t, handler.SetUserPassword(c))

	return rec
}

func TestUserHandler_GetUsers_Success(t *testing.T) {
	service := &MockUserService{}
	users := []*user.User{
		{ID: uuid.New(), Email: "admin@family.com", FirstName: "A", LastName: "Admin", Role: user.RoleAdmin},
		{ID: uuid.New(), Email: "member@family.com", FirstName: "M", LastName: "Member", Role: user.RoleMember},
	}
	service.On("GetUsers", mock.Anything).Return(users, nil)

	rec := listUsersRequest(t, service)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Data, 2)
	assert.Equal(t, users[0].Email, response.Data[0].Email)
	assert.Equal(t, "member", response.Data[1].Role)

	service.AssertExpectations(t)
}

func TestUserHandler_GetUsers_ServiceError(t *testing.T) {
	service := &MockUserService{}
	service.On("GetUsers", mock.Anything).Return(nil, errors.New("database down"))

	rec := listUsersRequest(t, service)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)

	service.AssertExpectations(t)
}

func TestUserHandler_PatchUser_ChangesRole(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	service.On("ChangeUserRole", mock.Anything, targetID, user.RoleMember).Return(nil)
	service.On("GetUserByID", mock.Anything, targetID).Return(&user.User{
		ID:    targetID,
		Email: "member@family.com",
		Role:  user.RoleMember,
	}, nil)

	rec := patchUserRequest(t, service, targetID.String(), `{"role":"member"}`, uuid.New())

	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "member", response.Data.Role)

	service.AssertExpectations(t)
}

// Понижение единственного администратора отбивает сервис; API отдаёт 409.
func TestUserHandler_PatchUser_LastAdminRejected(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	service.On("ChangeUserRole", mock.Anything, targetID, user.RoleMember).Return(services.ErrLastAdmin)

	rec := patchUserRequest(t, service, targetID.String(), `{"role":"member"}`, uuid.New())

	assert.Equal(t, http.StatusConflict, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "LAST_ADMIN", response.Error.Code)

	service.AssertExpectations(t)
}

func TestUserHandler_PatchUser_InvalidRole(t *testing.T) {
	service := &MockUserService{}

	rec := patchUserRequest(t, service, uuid.New().String(), `{"role":"owner"}`, uuid.New())

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)

	service.AssertNotCalled(t, "ChangeUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_PatchUser_EmptyBody(t *testing.T) {
	service := &MockUserService{}

	rec := patchUserRequest(t, service, uuid.New().String(), `{}`, uuid.New())

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	service.AssertNotCalled(t, "ChangeUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_PatchUser_InvalidID(t *testing.T) {
	service := &MockUserService{}

	rec := patchUserRequest(t, service, "not-a-uuid", `{"role":"member"}`, uuid.New())

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INVALID_ID", response.Error.Code)

	service.AssertNotCalled(t, "ChangeUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_PatchUser_UserNotFound(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	service.On("ChangeUserRole", mock.Anything, targetID, user.RoleChild).Return(services.ErrUserNotFound)

	rec := patchUserRequest(t, service, targetID.String(), `{"role":"child"}`, uuid.New())

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)

	service.AssertExpectations(t)
}

func TestUserHandler_PatchUser_Deactivates(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	actorID := uuid.New()
	service.On("SetActive", mock.Anything, targetID, false, actorID).Return(nil)
	service.On("GetUserByID", mock.Anything, targetID).Return(&user.User{
		ID: targetID, Email: "member@family.com", Role: user.RoleMember, IsActive: false,
	}, nil)

	rec := patchUserRequest(t, service, targetID.String(), `{"is_active":false}`, actorID)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.False(t, response.Data.IsActive)

	service.AssertNotCalled(t, "ChangeUserRole", mock.Anything, mock.Anything, mock.Anything)
	service.AssertExpectations(t)
}

// Роль и активность в одном запросе применяются по очереди, в этом порядке.
func TestUserHandler_PatchUser_RoleAndActive(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	service.On("ChangeUserRole", mock.Anything, targetID, user.RoleMember).Return(nil)
	service.On("SetActive", mock.Anything, targetID, true, mock.Anything).Return(nil)
	service.On("GetUserByID", mock.Anything, targetID).Return(&user.User{
		ID: targetID, Email: "member@family.com", Role: user.RoleMember, IsActive: true,
	}, nil)

	rec := patchUserRequest(t, service, targetID.String(), `{"role":"member","is_active":true}`, uuid.New())

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	service.AssertExpectations(t)
}

// Самодеактивация отбивается сервисом; API отдаёт 409 CANNOT_DEACTIVATE_SELF.
func TestUserHandler_PatchUser_SelfDeactivationRejected(t *testing.T) {
	service := &MockUserService{}
	selfID := uuid.New()
	service.On("SetActive", mock.Anything, selfID, false, selfID).Return(services.ErrCannotDeactivateSelf)

	rec := patchUserRequest(t, service, selfID.String(), `{"is_active":false}`, selfID)

	assert.Equal(t, http.StatusConflict, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "CANNOT_DEACTIVATE_SELF", response.Error.Code)
	service.AssertExpectations(t)
}

func TestUserHandler_PatchUser_DeactivateLastAdminRejected(t *testing.T) {
	service := &MockUserService{}
	targetID := uuid.New()
	service.On("SetActive", mock.Anything, targetID, false, mock.Anything).Return(services.ErrLastAdmin)

	rec := patchUserRequest(t, service, targetID.String(), `{"is_active":false}`, uuid.New())

	assert.Equal(t, http.StatusConflict, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "LAST_ADMIN", response.Error.Code)
	service.AssertExpectations(t)
}

func TestUserHandler_PatchUser_NoSession(t *testing.T) {
	service := &MockUserService{}

	rec := patchUserRequest(t, service, uuid.New().String(), `{"is_active":false}`, uuid.Nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	service.AssertNotCalled(t, "SetActive", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUserHandler_SetUserPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newFakeAuthService()
		targetID := uuid.New()

		rec := setUserPasswordRequest(t, svc, targetID.String(), `{"new_password":"battery-staple-2026"}`)

		assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
		require.NotNil(t, svc.adminSet)
		assert.Equal(t, targetID, svc.adminSet.userID)
		assert.Equal(t, "battery-staple-2026", svc.adminSet.next)
	})

	t.Run("user not found", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.adminErr = user.ErrNotFound

		rec := setUserPasswordRequest(t, svc, uuid.New().String(), `{"new_password":"battery-staple-2026"}`)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		var response handlers.ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "USER_NOT_FOUND", response.Error.Code)
	})

	// Политика пароля проверяется валидатором до сервиса.
	t.Run("weak password", func(t *testing.T) {
		svc := newFakeAuthService()

		rec := setUserPasswordRequest(t, svc, uuid.New().String(), `{"new_password":"short"}`)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Nil(t, svc.adminSet)

		var response handlers.ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.NotEmpty(t, response.Error.Details)
		assert.Equal(t, "new_password", response.Error.Details[0].Field)
	})

	t.Run("invalid id", func(t *testing.T) {
		rec := setUserPasswordRequest(t, newFakeAuthService(), "nope", `{"new_password":"battery-staple-2026"}`)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("storage failure", func(t *testing.T) {
		svc := newFakeAuthService()
		svc.adminErr = errors.New("disk on fire")

		rec := setUserPasswordRequest(t, svc, uuid.New().String(), `{"new_password":"battery-staple-2026"}`)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
