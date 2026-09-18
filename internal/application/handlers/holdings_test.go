package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
)

type mockHoldingService struct {
	mock.Mock
}

func (m *mockHoldingService) Create(
	ctx context.Context,
	id *uuid.UUID,
	name string,
	side holding.Side,
	kind holding.Kind,
) (*holding.Holding, error) {
	args := m.Called(ctx, id, name, side, kind)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*holding.Holding), args.Error(1)
}

func (m *mockHoldingService) GetByID(ctx context.Context, id uuid.UUID) (*holding.Holding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*holding.Holding), args.Error(1)
}

func (m *mockHoldingService) List(ctx context.Context, includeArchived bool) ([]*holding.Holding, error) {
	args := m.Called(ctx, includeArchived)
	return args.Get(0).([]*holding.Holding), args.Error(1)
}

func (m *mockHoldingService) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	kind *holding.Kind,
	archived *bool,
) (*holding.Holding, error) {
	args := m.Called(ctx, id, name, kind, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*holding.Holding), args.Error(1)
}

func (m *mockHoldingService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockHoldingService) PutValue(
	ctx context.Context,
	id uuid.UUID,
	day date.Date,
	value money.Minor,
) (*holding.Value, error) {
	args := m.Called(ctx, id, day, value)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*holding.Value), args.Error(1)
}

func (m *mockHoldingService) DeleteValue(ctx context.Context, id uuid.UUID, day date.Date) error {
	return m.Called(ctx, id, day).Error(0)
}

func (m *mockHoldingService) ListValues(
	ctx context.Context,
	id uuid.UUID,
	limit, offset int,
) ([]*holding.Value, int, error) {
	args := m.Called(ctx, id, limit, offset)
	return args.Get(0).([]*holding.Value), args.Int(1), args.Error(2)
}

func TestHoldingHandler_ListHoldings_CurrentAndPagination(t *testing.T) {
	svc := &mockHoldingService{}
	svc.On("List", mock.Anything, false).Return([]*holding.Holding{
		{ID: uuid.New(), Name: "Вклад", Side: holding.SideAsset, Kind: holding.KindDeposit,
			Current: &holding.Value{Date: date.New(2026, 9, 1), ValueMinor: 500}},
		{ID: uuid.New(), Name: "Ипотека", Side: holding.SideLiability, Kind: holding.KindMortgage},
	}, nil)
	h := handlers.NewHoldingHandler(svc)

	c, rec := principalContext(http.MethodGet, "/api/v1/holdings?limit=1&offset=1", "", nil)
	require.NoError(t, h.ListHoldings(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []map[string]any      `json:"data"`
		Meta handlers.ResponseMeta `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "Ипотека", resp.Data[0]["name"])
	assert.Contains(t, resp.Data[0], "current", "без снимков — явный null")
	assert.Nil(t, resp.Data[0]["current"])
	require.NotNil(t, resp.Meta.Pagination)
	assert.Equal(t, 2, resp.Meta.Pagination.Total, "total — до среза")

	c, rec = principalContext(http.MethodGet, "/api/v1/holdings?archived=1x", "", nil)
	require.NoError(t, h.ListHoldings(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHoldingHandler_CreateHolding_RepeatByIDSkipsService(t *testing.T) {
	svc := &mockHoldingService{}
	id := uuid.New()
	svc.On("GetByID", mock.Anything, id).Return(&holding.Holding{ID: id, Name: "Вклад"}, nil)
	h := handlers.NewHoldingHandler(svc)

	c, rec := principalContext(http.MethodPost, "/api/v1/holdings",
		`{"id":"`+id.String()+`","name":"Другое","side":"asset","kind":"cash"}`, nil)
	require.NoError(t, h.CreateHolding(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	svc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHoldingHandler_CreateHolding_Validation(t *testing.T) {
	h := handlers.NewHoldingHandler(&mockHoldingService{})

	c, rec := principalContext(
		http.MethodPost,
		"/api/v1/holdings",
		`{"name":"Вклад","side":"equity","kind":"cash"}`,
		nil,
	)
	require.NoError(t, h.CreateHolding(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, "side", errorCode(t, rec.Body.Bytes()).Error.Details[0].Field)
}

func TestHoldingHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		field  string
	}{
		{"name exists", holding.ErrNameExists, http.StatusConflict, handlers.ErrCodeHoldingNameExists, ""},
		{"blank name", holding.ErrNameEmpty, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError, "name"},
		{"wrong kind", holding.ErrInvalidKind, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError, "kind"},
		{"not found", holding.ErrNotFound, http.StatusNotFound, handlers.ErrCodeHoldingNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockHoldingService{}
			svc.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.err)
			h := handlers.NewHoldingHandler(svc)

			c, rec := principalContext(http.MethodPut, "/", `{"kind":"loan"}`, nil)
			c.SetParamNames("id")
			c.SetParamValues(uuid.NewString())
			require.NoError(t, h.UpdateHolding(c))
			assert.Equal(t, tt.status, rec.Code)
			resp := errorCode(t, rec.Body.Bytes())
			assert.Equal(t, tt.code, resp.Error.Code)
			if tt.field != "" {
				assert.Equal(t, tt.field, resp.Error.Details[0].Field)
			}
		})
	}
}

func TestHoldingHandler_UpdateHolding_EmptyBody(t *testing.T) {
	h := handlers.NewHoldingHandler(&mockHoldingService{})

	c, rec := principalContext(http.MethodPut, "/", `{"side":"liability"}`, nil)
	c.SetParamNames("id")
	c.SetParamValues(uuid.NewString())
	require.NoError(t, h.UpdateHolding(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "одна side — это пустое обновление")
}

func valueContext(method, body, id, day string) (echo.Context, *httptest.ResponseRecorder) {
	c, rec := principalContext(method, "/", body, nil)
	c.SetParamNames("id", "date")
	c.SetParamValues(id, day)

	return c, rec
}

func TestHoldingHandler_PutHoldingValue(t *testing.T) {
	svc := &mockHoldingService{}
	id := uuid.New()
	day := date.New(2026, 9, 1)
	svc.On("PutValue", mock.Anything, id, day, money.Minor(0)).
		Return(&holding.Value{Date: day, ValueMinor: 0}, nil)
	h := handlers.NewHoldingHandler(svc)

	c, rec := valueContext(http.MethodPut, `{"value_minor":0}`, id.String(), "2026-09-01")
	require.NoError(t, h.PutHoldingValue(c))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"value_minor":0`)

	c, rec = valueContext(http.MethodPut, `{}`, id.String(), "2026-09-01")
	require.NoError(t, h.PutHoldingValue(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "без value_minor")

	c, rec = valueContext(http.MethodPut, `{"value_minor":1}`, id.String(), "2026-02-30")
	require.NoError(t, h.PutHoldingValue(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "несуществующая дата в пути")
	svc.AssertNumberOfCalls(t, "PutValue", 1)
}

func TestHoldingHandler_ValueErrorMapping(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
		field  string
	}{
		{"future", holding.ErrValueDateFuture, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError, "date"},
		{"range", holding.ErrValueOutOfRange, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError,
			"value_minor"},
		{"no holding", holding.ErrNotFound, http.StatusNotFound, handlers.ErrCodeHoldingNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockHoldingService{}
			svc.On("PutValue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tt.err)
			h := handlers.NewHoldingHandler(svc)

			c, rec := valueContext(http.MethodPut, `{"value_minor":1}`, uuid.NewString(), "2026-09-01")
			require.NoError(t, h.PutHoldingValue(c))
			assert.Equal(t, tt.status, rec.Code)
			resp := errorCode(t, rec.Body.Bytes())
			assert.Equal(t, tt.code, resp.Error.Code)
			if tt.field != "" {
				assert.Equal(t, tt.field, resp.Error.Details[0].Field)
			}
		})
	}
}

func TestHoldingHandler_DeleteHoldingValue_NotFound(t *testing.T) {
	svc := &mockHoldingService{}
	svc.On("DeleteValue", mock.Anything, mock.Anything, mock.Anything).Return(holding.ErrValueNotFound)
	h := handlers.NewHoldingHandler(svc)

	c, rec := valueContext(http.MethodDelete, "", uuid.NewString(), "2026-09-01")
	require.NoError(t, h.DeleteHoldingValue(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeHoldingValueNotFound, errorCode(t, rec.Body.Bytes()).Error.Code)
}

func TestHoldingHandler_ListHoldingValues_Pagination(t *testing.T) {
	svc := &mockHoldingService{}
	id := uuid.New()
	svc.On("ListValues", mock.Anything, id, 1, 2).
		Return([]*holding.Value{{Date: date.New(2026, 7, 1), ValueMinor: 5}}, 3, nil)
	h := handlers.NewHoldingHandler(svc)

	c, rec := principalContext(http.MethodGet, "/?limit=1&offset=2", "", nil)
	c.SetParamNames("id")
	c.SetParamValues(id.String())
	require.NoError(t, h.ListHoldingValues(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp handlers.APIResponse[[]handlers.HoldingValueResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.NotNil(t, resp.Meta.Pagination)
	assert.Equal(t, 3, resp.Meta.Pagination.Total)
}
