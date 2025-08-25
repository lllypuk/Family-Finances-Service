package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

// MockFamilyRepository is a mock implementation of family repository
type MockFamilyRepository struct {
	mock.Mock
}

func (m *MockFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyRepository) Update(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockUserRepositoryFamily is a mock implementation of user repository for family tests
type MockUserRepositoryFamily struct {
	mock.Mock
}

func (m *MockUserRepositoryFamily) Create(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryFamily) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryFamily) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryFamily) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepositoryFamily) Update(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryFamily) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// setupFamilyHandler creates a new family handler with mock repositories
func setupFamilyHandler() (*handlers.FamilyHandler, *MockFamilyRepository, *MockUserRepositoryFamily) {
	mockFamilyRepo := &MockFamilyRepository{}
	mockUserRepo := &MockUserRepositoryFamily{}
	repositories := &handlers.Repositories{
		Family: mockFamilyRepo,
		User:   mockUserRepo,
	}
	handler := handlers.NewFamilyHandler(repositories)
	return handler, mockFamilyRepo, mockUserRepo
}

// createValidFamilyRequest creates a valid family request for testing
func createValidFamilyRequest() handlers.CreateFamilyRequest {
	return handlers.CreateFamilyRequest{
		Name:     "Test Family",
		Currency: "USD",
	}
}

func TestFamilyHandler_CreateFamily_Success(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	req := createValidFamilyRequest()
	mockFamilyRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.Family")).Return(nil)

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateFamily(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response handlers.APIResponse[handlers.FamilyResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, req.Name, response.Data.Name)
	assert.Equal(t, req.Currency, response.Data.Currency)
	assert.NotEqual(t, uuid.Nil, response.Data.ID)
	assert.False(t, response.Data.CreatedAt.IsZero())
	assert.False(t, response.Data.UpdatedAt.IsZero())

	mockFamilyRepo.AssertExpectations(t)
}

func TestFamilyHandler_CreateFamily_InvalidRequest(t *testing.T) {
	handler, _, _ := setupFamilyHandler()

	tests := []struct {
		name        string
		requestBody any
		expectedMsg string
	}{
		{
			name:        "Invalid JSON",
			requestBody: "invalid json",
			expectedMsg: "Invalid request body",
		},
		{
			name: "Missing name",
			requestBody: map[string]any{
				"currency": "USD",
			},
			expectedMsg: "",
		},
		{
			name: "Empty name",
			requestBody: map[string]any{
				"name":     "",
				"currency": "USD",
			},
			expectedMsg: "",
		},
		{
			name: "Name too short",
			requestBody: map[string]any{
				"name":     "A",
				"currency": "USD",
			},
			expectedMsg: "",
		},
		{
			name: "Name too long",
			requestBody: map[string]any{
				"name":     "This is a very long family name that exceeds the maximum allowed length for family names in the system which should be around 100 characters",
				"currency": "USD",
			},
			expectedMsg: "",
		},
		{
			name: "Missing currency",
			requestBody: map[string]any{
				"name": "Test Family",
			},
			expectedMsg: "",
		},
		{
			name: "Invalid currency length",
			requestBody: map[string]any{
				"name":     "Test Family",
				"currency": "US",
			},
			expectedMsg: "",
		},
		{
			name: "Invalid currency format",
			requestBody: map[string]any{
				"name":     "Test Family",
				"currency": "USDD",
			},
			expectedMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			// Act
			err = handler.CreateFamily(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestFamilyHandler_CreateFamily_RepositoryError(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	req := createValidFamilyRequest()
	mockFamilyRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.Family")).
		Return(errors.New("database error"))

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateFamily(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "CREATE_FAILED", response.Error.Code)
}

func TestFamilyHandler_GetFamilyByID_Success(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	expectedFamily := &user.Family{
		ID:        familyID,
		Name:      "Test Family",
		Currency:  "EUR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockFamilyRepo.On("GetByID", mock.Anything, familyID).Return(expectedFamily, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.FamilyResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedFamily.ID, response.Data.ID)
	assert.Equal(t, expectedFamily.Name, response.Data.Name)
	assert.Equal(t, expectedFamily.Currency, response.Data.Currency)
	assert.WithinDuration(t, expectedFamily.CreatedAt, response.Data.CreatedAt, time.Second)
	assert.WithinDuration(t, expectedFamily.UpdatedAt, response.Data.UpdatedAt, time.Second)

	mockFamilyRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyByID_InvalidID(t *testing.T) {
	handler, _, _ := setupFamilyHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.GetFamilyByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_ID", response.Error.Code)
}

func TestFamilyHandler_GetFamilyByID_NotFound(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	mockFamilyRepo.On("GetByID", mock.Anything, familyID).Return(nil, errors.New("not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "FAMILY_NOT_FOUND", response.Error.Code)

	mockFamilyRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_Success(t *testing.T) {
	handler, _, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	expectedMembers := []*user.User{
		{
			ID:        uuid.New(),
			Email:     "admin@family.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      user.RoleAdmin,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "member@family.com",
			FirstName: "Jane",
			LastName:  "Doe",
			Role:      user.RoleMember,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "child@family.com",
			FirstName: "Little",
			LastName:  "Doe",
			Role:      user.RoleChild,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedMembers, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String()+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.UserResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Data, 3)

	// Check admin user
	assert.Equal(t, expectedMembers[0].ID, response.Data[0].ID)
	assert.Equal(t, expectedMembers[0].Email, response.Data[0].Email)
	assert.Equal(t, expectedMembers[0].FirstName, response.Data[0].FirstName)
	assert.Equal(t, expectedMembers[0].LastName, response.Data[0].LastName)
	assert.Equal(t, string(expectedMembers[0].Role), response.Data[0].Role)
	assert.Equal(t, expectedMembers[0].FamilyID, response.Data[0].FamilyID)

	// Check member user
	assert.Equal(t, expectedMembers[1].ID, response.Data[1].ID)
	assert.Equal(t, string(expectedMembers[1].Role), response.Data[1].Role)

	// Check child user
	assert.Equal(t, expectedMembers[2].ID, response.Data[2].ID)
	assert.Equal(t, string(expectedMembers[2].Role), response.Data[2].Role)

	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_InvalidID(t *testing.T) {
	handler, _, _ := setupFamilyHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/invalid/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_ID", response.Error.Code)
}

func TestFamilyHandler_GetFamilyMembers_RepositoryError(t *testing.T) {
	handler, _, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(nil, errors.New("database error"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String()+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "FETCH_FAILED", response.Error.Code)

	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_EmptyFamily(t *testing.T) {
	handler, _, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return([]*user.User{}, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String()+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.UserResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Empty(t, response.Data)

	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_CurrencyValidation(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	validCurrencies := []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "CNY", "SEK", "NZD"}

	for _, currency := range validCurrencies {
		t.Run(fmt.Sprintf("Valid currency: %s", currency), func(t *testing.T) {
			req := createValidFamilyRequest()
			req.Currency = currency

			mockFamilyRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *user.Family) bool {
				return f.Currency == currency
			})).Return(nil).Once()

			body, err := json.Marshal(req)
			require.NoError(t, err)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err = handler.CreateFamily(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestFamilyHandler_FamilyNameValidation(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	validNames := []string{
		"The Smiths",
		"Johnson Family",
		"Brown-Wilson",
		"Familie Müller",
		"Famille Dupont",
		"Семья Петров",
		"familia García",
		"A very long family name but still within the limit of one hundred characters maximum length",
	}

	for _, name := range validNames {
		t.Run(fmt.Sprintf("Valid name: %s", name), func(t *testing.T) {
			req := createValidFamilyRequest()
			req.Name = name

			mockFamilyRepo.On("Create", mock.Anything, mock.MatchedBy(func(f *user.Family) bool {
				return f.Name == name
			})).Return(nil).Once()

			body, err := json.Marshal(req)
			require.NoError(t, err)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err = handler.CreateFamily(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestFamilyHandler_RoleHierarchy_InMembersList(t *testing.T) {
	handler, _, mockUserRepo := setupFamilyHandler()

	// Arrange - Create family with different role types
	familyID := uuid.New()
	expectedMembers := []*user.User{
		{
			ID:        uuid.New(),
			Email:     "admin@family.com",
			FirstName: "Admin",
			LastName:  "User",
			Role:      user.RoleAdmin,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "member@family.com",
			FirstName: "Member",
			LastName:  "User",
			Role:      user.RoleMember,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "child@family.com",
			FirstName: "Child",
			LastName:  "User",
			Role:      user.RoleChild,
			FamilyID:  familyID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedMembers, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String()+"/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(familyID.String())

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.UserResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Data, 3)

	// Verify all roles are present
	roles := make(map[string]bool)
	for _, member := range response.Data {
		roles[member.Role] = true
	}

	assert.True(t, roles["admin"], "Admin role should be present")
	assert.True(t, roles["member"], "Member role should be present")
	assert.True(t, roles["child"], "Child role should be present")

	mockUserRepo.AssertExpectations(t)
}

// Benchmark tests for performance validation
func BenchmarkFamilyHandler_CreateFamily(b *testing.B) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Setup mock to return nil for all calls
	mockFamilyRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.Family")).Return(nil)

	req := createValidFamilyRequest()
	body, _ := json.Marshal(req)

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodPost, "/families", bytes.NewBuffer(body))
		httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.CreateFamily(c)
	}
}

func BenchmarkFamilyHandler_GetFamilyMembers(b *testing.B) {
	handler, _, mockUserRepo := setupFamilyHandler()

	// Setup mock to return empty slice for all calls
	mockUserRepo.On("GetByFamilyID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return([]*user.User{}, nil)

	familyID := uuid.New()

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodGet, "/families/"+familyID.String()+"/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)
		c.SetParamNames("id")
		c.SetParamValues(familyID.String())

		handler.GetFamilyMembers(c)
	}
}
