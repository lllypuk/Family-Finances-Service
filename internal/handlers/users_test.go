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

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/handlers"
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

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestUserHandler_CreateUser(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserRepository)
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
			mockSetup: func(repo *MockUserRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
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
			name:        "Error - Invalid request body",
			requestBody: `{"invalid": json}`, // Invalid JSON syntax to trigger bind error
			mockSetup: func(repo *MockUserRepository) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "INVALID_REQUEST", response.Error.Code)
			},
		},
		{
			name: "Error - Missing required fields",
			requestBody: handlers.CreateUserRequest{
				Email: "test@example.com",
				// Missing password, name, etc.
			},
			mockSetup: func(repo *MockUserRepository) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.UserResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
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
			mockSetup: func(repo *MockUserRepository) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[handlers.UserResponse]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.Errors)
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
			mockSetup: func(repo *MockUserRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.ErrorResponse
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "CREATE_FAILED", response.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			repositories := &handlers.Repositories{User: mockRepo}
			handler := handlers.NewUserHandler(repositories)

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
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetUserByID(t *testing.T) {
	userID := uuid.New()
	familyID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:   "Success - User found",
			userID: userID.String(),
			mockSetup: func(repo *MockUserRepository) {
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
				repo.On("GetByID", mock.Anything, userID).Return(user, nil)
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
			mockSetup: func(repo *MockUserRepository) {
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
			mockSetup: func(repo *MockUserRepository) {
				repo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("not found"))
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
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			repositories := &handlers.Repositories{User: mockRepo}
			handler := handlers.NewUserHandler(repositories)

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
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	userID := uuid.New()
	familyID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		mockSetup      func(*MockUserRepository)
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
			mockSetup: func(repo *MockUserRepository) {
				existingUser := &user.User{
					ID:        userID,
					Email:     "test@example.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      user.RoleMember,
					FamilyID:  familyID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				updatedUser := *existingUser
				updatedUser.FirstName = "Jane"
				updatedUser.LastName = "Smith"

				repo.On("GetByID", mock.Anything, userID).Return(existingUser, nil)
				repo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(nil)
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
			mockSetup: func(repo *MockUserRepository) {
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
			mockSetup: func(repo *MockUserRepository) {
				repo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("not found"))
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
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			repositories := &handlers.Repositories{User: mockRepo}
			handler := handlers.NewUserHandler(repositories)

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
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserHandler_DeleteUser(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockUserRepository)
		expectedStatus int
		expectedBody   func(t *testing.T, body string)
	}{
		{
			name:   "Success - User deleted",
			userID: userID.String(),
			mockSetup: func(repo *MockUserRepository) {
				repo.On("Delete", mock.Anything, userID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body string) {
				var response handlers.APIResponse[map[string]string]
				err := json.Unmarshal([]byte(body), &response)
				require.NoError(t, err)
				assert.Equal(t, "User deleted successfully", response.Data["message"])
			},
		},
		{
			name:   "Error - Invalid UUID",
			userID: "invalid-uuid",
			mockSetup: func(repo *MockUserRepository) {
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
			mockSetup: func(repo *MockUserRepository) {
				repo.On("Delete", mock.Anything, userID).Return(errors.New("not found"))
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
			mockRepo := &MockUserRepository{}
			tt.mockSetup(mockRepo)

			repositories := &handlers.Repositories{User: mockRepo}
			handler := handlers.NewUserHandler(repositories)

			// Create request
			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, nil)
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
			mockRepo.AssertExpectations(t)
		})
	}
}
