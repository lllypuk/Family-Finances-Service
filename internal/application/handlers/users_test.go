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

func (m *MockUserService) GetUsersByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
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

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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
				FamilyID:  uuid.New(),
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				testUser := &user.User{
					ID:        uuid.New(),
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
					FamilyID:  uuid.New(),
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
				assert.NotEqual(t, uuid.Nil, response.Data.FamilyID)
				assert.Equal(t, "member", response.Data.Role)
			},
		},
		{
			name: "Error - Service validation failed",
			requestBody: handlers.CreateUserRequest{
				Email:     "invalid-email", // Invalid email format
				Password:  "123",           // Too short password
				FirstName: "",              // Empty first name
				LastName:  "Doe",
				FamilyID:  uuid.New(),
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				service.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserDTO")).
					Return(nil, services.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
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
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				service.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserDTO")).
					Return(nil, services.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
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
				FamilyID:  uuid.New(),
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				service.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserDTO")).
					Return(nil, services.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			},
		},
		{
			name: "Error - Repository creation fails",
			requestBody: handlers.CreateUserRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				FamilyID:  uuid.New(),
				Role:      "member",
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
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
			handler := handlers.NewUserHandler(repositories, mockService)

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
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				user := &user.User{
					ID:        userID,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
					FamilyID:  familyID,
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
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
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
			handler := handlers.NewUserHandler(repositories, mockService)

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
				FirstName: stringPtr("UpdatedName"),
				LastName:  stringPtr("UpdatedLastName"),
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				updatedUser := &user.User{
					ID:        userID,
					Email:     "test@example.com",
					FirstName: "UpdatedName",
					LastName:  "UpdatedLastName",
					Role:      user.RoleMember,
					FamilyID:  familyID,
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
				FirstName: stringPtr("UpdatedName"),
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
				FirstName: stringPtr("UpdatedName"),
			},
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
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
			handler := handlers.NewUserHandler(repositories, mockService)

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

func TestUserHandler_DeleteUser(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserService, uuid.UUID)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:   "Success - User deleted",
			userID: userID.String(),
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				service.On("DeleteUser", mock.Anything, userID, familyID).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody: func(t *testing.T, body string) {
				// No content expected for 204 No Content
				assert.Empty(t, body)
			},
		},
		{
			name:   "Error - Invalid UUID",
			userID: "invalid-uuid",
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
			mockSetup: func(service *MockUserService, familyID uuid.UUID) {
				service.On("DeleteUser", mock.Anything, userID, familyID).Return(errors.New("database error"))
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
			familyUUID := uuid.New()
			e := echo.New()
			mockService := &MockUserService{}
			tt.mockSetup(mockService, familyUUID)

			repositories := &handlers.Repositories{}
			handler := handlers.NewUserHandler(repositories, mockService)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID+"?family_id="+familyUUID.String(), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Execute
			err := handler.DeleteUser(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.expectedBody(t, rec.Body.String())
			mockService.AssertExpectations(t)
		})
	}
}
