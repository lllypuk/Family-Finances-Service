package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/account"
)

type mockAccountService struct {
	mock.Mock
}

func (m *mockAccountService) Create(ctx context.Context, id *uuid.UUID, name string) (*account.Account, error) {
	args := m.Called(ctx, id, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *mockAccountService) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *mockAccountService) List(ctx context.Context, includeArchived bool) ([]*account.Account, error) {
	args := m.Called(ctx, includeArchived)
	return args.Get(0).([]*account.Account), args.Error(1)
}

func (m *mockAccountService) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	archived *bool,
) (*account.Account, error) {
	args := m.Called(ctx, id, name, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *mockAccountService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func errorCode(t *testing.T, body []byte) handlers.ErrorResponse {
	t.Helper()

	var resp handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(body, &resp))

	return resp
}

func TestAccountHandler_ListAccounts_ArchivedParam(t *testing.T) {
	svc := &mockAccountService{}
	svc.On("List", mock.Anything, true).Return([]*account.Account{{ID: uuid.New(), Name: "Сбер"}}, nil)
	h := handlers.NewAccountHandler(svc)

	c, rec := principalContext(http.MethodGet, "/api/v1/accounts?archived=true", "", nil)
	require.NoError(t, h.ListAccounts(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp handlers.APIResponse[[]handlers.AccountResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "Сбер", resp.Data[0].Name)

	c, rec = principalContext(http.MethodGet, "/api/v1/accounts?archived=yes", "", nil)
	require.NoError(t, h.ListAccounts(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, "archived", errorCode(t, rec.Body.Bytes()).Error.Details[0].Field)
}

func TestAccountHandler_CreateAccount_RepeatByIDSkipsService(t *testing.T) {
	svc := &mockAccountService{}
	id := uuid.New()
	svc.On("GetByID", mock.Anything, id).Return(&account.Account{ID: id, Name: "Сбер", IsArchived: true}, nil)
	h := handlers.NewAccountHandler(svc)

	c, rec := principalContext(http.MethodPost, "/api/v1/accounts", `{"id":"`+id.String()+`","name":"Другое"}`, nil)
	require.NoError(t, h.CreateAccount(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
}

func TestAccountHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"name exists", account.ErrNameExists, http.StatusConflict, handlers.ErrCodeAccountNameExists},
		{"blank name", account.ErrNameEmpty, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"not found", account.ErrNotFound, http.StatusNotFound, handlers.ErrCodeAccountNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAccountService{}
			svc.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tt.err)
			h := handlers.NewAccountHandler(svc)

			c, rec := principalContext(http.MethodPut, "/", `{"name":" "}`, nil)
			c.SetParamNames("id")
			c.SetParamValues(uuid.NewString())
			require.NoError(t, h.UpdateAccount(c))
			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.code, errorCode(t, rec.Body.Bytes()).Error.Code)
		})
	}
}

func TestAccountHandler_UpdateAccount_EmptyBody(t *testing.T) {
	h := handlers.NewAccountHandler(&mockAccountService{})

	c, rec := principalContext(http.MethodPut, "/", `{}`, nil)
	c.SetParamNames("id")
	c.SetParamValues(uuid.NewString())
	require.NoError(t, h.UpdateAccount(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestAccountHandler_DeleteAccount(t *testing.T) {
	svc := &mockAccountService{}
	inUse := uuid.New()
	svc.On("Delete", mock.Anything, inUse).Return(account.ErrInUse)
	h := handlers.NewAccountHandler(svc)

	c, rec := principalContext(http.MethodDelete, "/", "", nil)
	c.SetParamNames("id")
	c.SetParamValues(inUse.String())
	require.NoError(t, h.DeleteAccount(c))
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, handlers.ErrCodeAccountInUse, errorCode(t, rec.Body.Bytes()).Error.Code)

	c, rec = principalContext(http.MethodDelete, "/", "", nil)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")
	require.NoError(t, h.DeleteAccount(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
