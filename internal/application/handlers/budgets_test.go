package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// stubBudgetService отдаёт заранее заданный ответ или ошибку; остальной интерфейс
// не вызывается и остаётся во встроенном nil.
type stubBudgetService struct {
	services.BudgetService

	budget   *budget.Budget
	budgets  []*budget.Budget
	total    int
	err      error
	created  *dto.CreateBudgetDTO
	updated  *dto.UpdateBudgetDTO
	activeOn *date.Date
	deleted  *uuid.UUID
}

func (s *stubBudgetService) CreateBudget(_ context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error) {
	s.created = &req
	return s.budget, s.err
}

func (s *stubBudgetService) GetBudgetByID(_ context.Context, _ uuid.UUID) (*budget.Budget, error) {
	return s.budget, s.err
}

func (s *stubBudgetService) GetBudgetsPage(
	_ context.Context,
	_ dto.BudgetFilterDTO,
) ([]*budget.Budget, int, error) {
	return s.budgets, s.total, s.err
}

func (s *stubBudgetService) GetActiveBudgets(_ context.Context, on date.Date) ([]*budget.Budget, error) {
	s.activeOn = &on
	return s.budgets, s.err
}

func (s *stubBudgetService) UpdateBudget(
	_ context.Context,
	_ uuid.UUID,
	req dto.UpdateBudgetDTO,
) (*budget.Budget, error) {
	s.updated = &req
	return s.budget, s.err
}

func (s *stubBudgetService) DeleteBudget(_ context.Context, id uuid.UUID) error {
	s.deleted = &id
	return s.err
}

// stubFamilyRepository — семья нужна хендлеру бюджетов только ради часового пояса.
type stubFamilyRepository struct {
	handlers.FamilyRepository

	timezone string
}

func (s stubFamilyRepository) Get(_ context.Context) (*user.Family, error) {
	return &user.Family{ID: uuid.New(), Name: "Test", Currency: "RUB", Timezone: s.timezone}, nil
}

func setupBudgetHandler(service *stubBudgetService) *handlers.BudgetHandler {
	return handlers.NewBudgetHandler(
		&handlers.Repositories{Family: stubFamilyRepository{timezone: "Asia/Tokyo"}},
		service,
	)
}

func testBudget(id uuid.UUID) *budget.Budget {
	categoryID := uuid.New()
	start := date.New(2026, time.January, 1)
	end := date.New(2026, time.January, 31)

	return &budget.Budget{
		ID:          id,
		Name:        "Food",
		AmountMinor: 100_000,
		SpentMinor:  27_525,
		Period:      budget.PeriodMonthly,
		CategoryID:  &categoryID,
		StartDate:   start,
		EndDate:     end,
		IsActive:    true,
		CreatedAt:   start.In(time.UTC),
		UpdatedAt:   start.In(time.UTC),
	}
}

func TestBudgetHandler_CreateBudget_Success(t *testing.T) {
	created := testBudget(uuid.New())
	service := &stubBudgetService{budget: created}
	handler := setupBudgetHandler(service)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/budgets", strings.NewReader(
		`{"name":"Food","amount_minor":100000,"period":"monthly","start_date":"2026-01-01","end_date":"2026-01-31"}`,
	))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.CreateBudget(e.NewContext(req, rec)))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, created.ID, response.Data.ID)
	assert.Equal(t, money.Minor(27_525), response.Data.SpentMinor)
	assert.Equal(t, money.Minor(72_475), response.Data.RemainingMinor)

	require.NotNil(t, service.created)
	assert.Equal(t, budget.PeriodMonthly, service.created.Period)
}

// TestBudgetHandler_GetBudgets_ActiveOnlyUsesFamilyTimezone — активные бюджеты выбираются
// на сегодня в поясе семьи, а не сервера (A-06).
func TestBudgetHandler_GetBudgets_ActiveOnlyUsesFamilyTimezone(t *testing.T) {
	service := &stubBudgetService{budgets: []*budget.Budget{testBudget(uuid.New())}}
	handler := setupBudgetHandler(service)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets?active_only=true", nil)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.GetBudgets(e.NewContext(req, rec)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[[]handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 1, response.Meta.Pagination.Total)

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	require.NotNil(t, service.activeOn)
	assert.Equal(t, date.Today(tokyo), *service.activeOn)
}

func TestBudgetHandler_GetBudgetByID_NotFound(t *testing.T) {
	handler := setupBudgetHandler(&stubBudgetService{err: services.ErrBudgetNotFoundService})

	budgetID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/budgets/"+budgetID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	require.NoError(t, handler.GetBudgetByID(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "BUDGET_NOT_FOUND", response.Error.Code)
}

func TestBudgetHandler_UpdateBudget_BusinessConflicts(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "overlap", err: services.ErrBudgetOverlapExists, code: handlers.ErrCodeBudgetOverlap},
		{name: "name_exists", err: services.ErrBudgetNameExists, code: handlers.ErrCodeBudgetNameExists},
		{name: "below_spent", err: services.ErrBudgetAlreadyExceeded, code: handlers.ErrCodeBudgetBelowSpent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupBudgetHandler(&stubBudgetService{err: tt.err})

			rec := updateBudgetRequest(t, handler, `{"amount_minor":1000}`)
			assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.code, response.Error.Code)
			assert.Empty(t, response.Error.Details)
		})
	}
}

func TestBudgetHandler_UpdateBudget_Success(t *testing.T) {
	updated := testBudget(uuid.New())
	service := &stubBudgetService{budget: updated}
	handler := setupBudgetHandler(service)

	rec := updateBudgetRequest(t, handler, `{"amount_minor":100000}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, updated.ID, response.Data.ID)

	require.NotNil(t, service.updated)
	require.NotNil(t, service.updated.AmountMinor)
	assert.Equal(t, money.Minor(100_000), *service.updated.AmountMinor)
}

// TestBudgetHandler_UpdateBudget_CalculationFailedIs500 — сорванный расчёт расхода это сбой
// инфраструктуры: бизнес-кода у него нет, и 409 клиент трактовал бы как «поправь тело».
func TestBudgetHandler_UpdateBudget_CalculationFailedIs500(t *testing.T) {
	handler := setupBudgetHandler(&stubBudgetService{err: services.ErrBudgetCalculationFailed})

	rec := updateBudgetRequest(t, handler, `{"amount_minor":1000}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
}

// TestBudgetHandler_UpdateBudget_AmountTooLargeStays422 — отказ формы остаётся валидацией.
func TestBudgetHandler_UpdateBudget_AmountTooLargeStays422(t *testing.T) {
	handler := setupBudgetHandler(&stubBudgetService{err: services.ErrBudgetAmountTooLarge})

	rec := updateBudgetRequest(t, handler, `{"amount_minor":1000}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
}

func TestBudgetHandler_DeleteBudget_Success(t *testing.T) {
	service := &stubBudgetService{}
	handler := setupBudgetHandler(service)

	budgetID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/budgets/"+budgetID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	require.NoError(t, handler.DeleteBudget(c))
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, service.deleted)
	assert.Equal(t, budgetID, *service.deleted)
}

func TestBudgetHandler_DeleteBudget_NotFound(t *testing.T) {
	handler := setupBudgetHandler(&stubBudgetService{err: services.ErrBudgetNotFoundService})

	budgetID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/budgets/"+budgetID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	require.NoError(t, handler.DeleteBudget(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func updateBudgetRequest(t *testing.T, handler *handlers.BudgetHandler, body string) *httptest.ResponseRecorder {
	t.Helper()

	budgetID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/budgets/"+budgetID.String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())

	require.NoError(t, handler.UpdateBudget(c))

	return rec
}
