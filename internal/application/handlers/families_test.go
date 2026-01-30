package handlers_test

import (
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
)

// MockFamilyRepository is a mock implementation of family repository
type MockFamilyRepository struct {
	mock.Mock
}

func (m *MockFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) Get(ctx context.Context) (*user.Family, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyRepository) Update(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) Exists(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
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

func TestFamilyHandler_GetFamily_Success(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	expectedFamily := &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "EUR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockFamilyRepo.On("Get", mock.Anything).Return(expectedFamily, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetFamily(c)

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

func TestFamilyHandler_GetFamily_NotFound(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	mockFamilyRepo.On("Get", mock.Anything).Return(nil, errors.New("not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetFamily(c)

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
	handler, mockFamilyRepo, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}
	expectedMembers := []*user.User{
		{
			ID:        uuid.New(),
			Email:     "admin@family.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      user.RoleAdmin,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "member@family.com",
			FirstName: "Jane",
			LastName:  "Doe",
			Role:      user.RoleMember,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "child@family.com",
			FirstName: "Little",
			LastName:  "Doe",
			Role:      user.RoleChild,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedMembers, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

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

	// Check member user
	assert.Equal(t, expectedMembers[1].ID, response.Data[1].ID)
	assert.Equal(t, string(expectedMembers[1].Role), response.Data[1].Role)

	// Check child user
	assert.Equal(t, expectedMembers[2].ID, response.Data[2].ID)
	assert.Equal(t, string(expectedMembers[2].Role), response.Data[2].Role)

	mockFamilyRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_FamilyNotFound(t *testing.T) {
	handler, mockFamilyRepo, _ := setupFamilyHandler()

	// Arrange
	mockFamilyRepo.On("Get", mock.Anything).Return(nil, errors.New("not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "FAMILY_NOT_FOUND", response.Error.Code)

	mockFamilyRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_RepositoryError(t *testing.T) {
	handler, mockFamilyRepo, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(nil, errors.New("database error"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "FETCH_FAILED", response.Error.Code)

	mockFamilyRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_GetFamilyMembers_EmptyFamily(t *testing.T) {
	handler, mockFamilyRepo, mockUserRepo := setupFamilyHandler()

	// Arrange
	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return([]*user.User{}, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetFamilyMembers(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.UserResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Empty(t, response.Data)

	mockFamilyRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

func TestFamilyHandler_RoleHierarchy_InMembersList(t *testing.T) {
	handler, mockFamilyRepo, mockUserRepo := setupFamilyHandler()

	// Arrange - Create family with different role types
	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}
	expectedMembers := []*user.User{
		{
			ID:        uuid.New(),
			Email:     "admin@family.com",
			FirstName: "Admin",
			LastName:  "User",
			Role:      user.RoleAdmin,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "member@family.com",
			FirstName: "Member",
			LastName:  "User",
			Role:      user.RoleMember,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Email:     "child@family.com",
			FirstName: "Child",
			LastName:  "User",
			Role:      user.RoleChild,

			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedMembers, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

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

	mockFamilyRepo.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
}

// Benchmark tests for performance validation
func BenchmarkFamilyHandler_GetFamilyMembers(b *testing.B) {
	handler, mockFamilyRepo, mockUserRepo := setupFamilyHandler()

	familyID := uuid.New()
	family := &user.Family{
		ID:       familyID,
		Name:     "Test Family",
		Currency: "USD",
	}

	// Setup mocks
	mockFamilyRepo.On("Get", mock.Anything).Return(family, nil)
	mockUserRepo.On("GetByFamilyID", mock.Anything, familyID).Return([]*user.User{}, nil)

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodGet, "/api/v1/family/members", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.GetFamilyMembers(c)
	}
}
