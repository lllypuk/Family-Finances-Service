package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
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

// MockFamilyService provides mock implementation for services.FamilyService
type MockFamilyService struct {
	mock.Mock
}

func (m *MockFamilyService) SetupFamily(ctx context.Context, req dto.SetupFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) GetFamily(ctx context.Context) (*user.Family, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) UpdateFamily(ctx context.Context, req dto.UpdateFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) IsSetupComplete(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func familyRequest(t *testing.T, service *MockFamilyService, method, body string) *httptest.ResponseRecorder {
	t.Helper()

	handler := handlers.NewFamilyHandler(service)

	e := echo.New()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, "/api/v1/family", reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if method == http.MethodGet {
		require.NoError(t, handler.GetFamily(c))
	} else {
		require.NoError(t, handler.UpdateFamily(c))
	}

	return rec
}

func testFamily() *user.Family {
	return &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "EUR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestFamilyHandler_GetFamily_Success(t *testing.T) {
	service := &MockFamilyService{}
	family := testFamily()
	service.On("GetFamily", mock.Anything).Return(family, nil)

	rec := familyRequest(t, service, http.MethodGet, "")

	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.FamilyResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, family.ID, response.Data.ID)
	assert.Equal(t, family.Name, response.Data.Name)
	assert.Equal(t, family.Currency, response.Data.Currency)

	service.AssertExpectations(t)
}

func TestFamilyHandler_GetFamily_NotFound(t *testing.T) {
	service := &MockFamilyService{}
	service.On("GetFamily", mock.Anything).Return(nil, services.ErrFamilyNotFound)

	rec := familyRequest(t, service, http.MethodGet, "")

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "FAMILY_NOT_FOUND", response.Error.Code)

	service.AssertExpectations(t)
}

func TestFamilyHandler_UpdateFamily_Success(t *testing.T) {
	service := &MockFamilyService{}
	family := testFamily()
	family.Name = "Renamed Family"
	service.On("UpdateFamily", mock.Anything, mock.MatchedBy(func(req dto.UpdateFamilyDTO) bool {
		return req.Name != nil && *req.Name == "Renamed Family" && req.Currency == nil
	})).Return(family, nil)

	rec := familyRequest(t, service, http.MethodPut, `{"name":"Renamed Family"}`)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.FamilyResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "Renamed Family", response.Data.Name)

	service.AssertExpectations(t)
}

func TestFamilyHandler_UpdateFamily_ValidationError(t *testing.T) {
	service := &MockFamilyService{}

	rec := familyRequest(t, service, http.MethodPut, `{"currency":"EURO"}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)

	service.AssertNotCalled(t, "UpdateFamily", mock.Anything, mock.Anything)
}

// Пустое тело — minProperties: 1 в контракте: менять нечего, сервис не зовём.
func TestFamilyHandler_UpdateFamily_EmptyBody(t *testing.T) {
	service := &MockFamilyService{}

	rec := familyRequest(t, service, http.MethodPut, `{}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)

	service.AssertNotCalled(t, "UpdateFamily", mock.Anything, mock.Anything)
}

func TestFamilyHandler_UpdateFamily_NotFound(t *testing.T) {
	service := &MockFamilyService{}
	service.On("UpdateFamily", mock.Anything, mock.Anything).Return(nil, services.ErrFamilyNotFound)

	rec := familyRequest(t, service, http.MethodPut, `{"name":"Renamed Family"}`)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "FAMILY_NOT_FOUND", response.Error.Code)

	service.AssertExpectations(t)
}
