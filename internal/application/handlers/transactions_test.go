package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// stubTransactionService записывает пришедший запрос и отдаёт заготовленный ответ;
// остальной интерфейс не вызывается и остаётся во встроенном nil.
type stubTransactionService struct {
	services.TransactionService

	tx      *transaction.Transaction
	txs     []*transaction.Transaction
	total   int
	err     error
	created *dto.CreateTransactionDTO
	updated *dto.UpdateTransactionDTO
	filter  *dto.TransactionFilterDTO
	deleted *uuid.UUID
}

func (s *stubTransactionService) CreateTransaction(
	_ context.Context,
	req dto.CreateTransactionDTO,
) (*transaction.Transaction, error) {
	s.created = &req
	if s.err != nil {
		return nil, s.err
	}
	if s.tx != nil {
		return s.tx, nil
	}

	return &transaction.Transaction{
		ID:          uuid.New(),
		AmountMinor: req.AmountMinor,
		Type:        req.Type,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		Date:        req.Date,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (s *stubTransactionService) GetTransactionByID(
	_ context.Context,
	_ uuid.UUID,
) (*transaction.Transaction, error) {
	if s.tx == nil && s.err == nil {
		return nil, services.ErrTransactionNotFound
	}

	return s.tx, s.err
}

func (s *stubTransactionService) GetAllTransactions(
	_ context.Context,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	s.filter = &filter
	return s.txs, s.err
}

func (s *stubTransactionService) CountTransactions(_ context.Context, _ dto.TransactionFilterDTO) (int, error) {
	return s.total, s.err
}

func (s *stubTransactionService) UpdateTransaction(
	_ context.Context,
	_ uuid.UUID,
	req dto.UpdateTransactionDTO,
) (*transaction.Transaction, error) {
	s.updated = &req
	return s.tx, s.err
}

func (s *stubTransactionService) DeleteTransaction(_ context.Context, id uuid.UUID) error {
	s.deleted = &id
	return s.err
}

func setupTransactionHandler(service *stubTransactionService) *handlers.TransactionHandler {
	return handlers.NewTransactionHandler(service)
}

// createValidTransactionRequest creates a valid transaction request for testing
func createValidTransactionRequest() handlers.CreateTransactionRequest {
	return handlers.CreateTransactionRequest{
		AmountMinor: 10_050,
		Type:        "expense",
		Description: "Test transaction",
		CategoryID:  uuid.New(),
		Date:        date.Today(time.UTC),
		Tags:        []string{"test", "expense"},
	}
}

// withSessionUser кладёт в контекст владельца токена так же, как auth.RequireBearer
// на группе /api/v1. Без него API-хендлеры не знают автора записи.
func withSessionUser(c echo.Context, userID uuid.UUID) {
	c.Set(auth.ContextKey, &auth.Principal{
		UserID: userID,
		Role:   user.RoleAdmin,
		Email:  "session@example.com",
	})
}

func testTransaction(id uuid.UUID) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          id,
		AmountMinor: 15_000,
		Type:        transaction.TypeExpense,
		Description: "Test transaction",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		Date:        date.Today(time.UTC),
		Tags:        []string{"test"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func postTransaction(
	t *testing.T,
	handler *handlers.TransactionHandler,
	body []byte,
	userID *uuid.UUID,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	if userID != nil {
		withSessionUser(c, *userID)
	}

	require.NoError(t, handler.CreateTransaction(c))

	return rec
}

func TestTransactionHandler_CreateTransaction_Success(t *testing.T) {
	service := &stubTransactionService{}
	handler := setupTransactionHandler(service)

	req := createValidTransactionRequest()
	sessionUserID := uuid.New()
	body, err := json.Marshal(req)
	require.NoError(t, err)

	rec := postTransaction(t, handler, body, &sessionUserID)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, req.AmountMinor, response.Data.AmountMinor)
	assert.Equal(t, req.Type, response.Data.Type)
	assert.Equal(t, req.Description, response.Data.Description)
	assert.Equal(t, req.CategoryID, response.Data.CategoryID)
	assert.Equal(t, sessionUserID, response.Data.UserID)
	assert.Equal(t, req.Tags, response.Data.Tags)
}

// TestTransactionHandler_CreateTransaction_IgnoresBodyUserID закрывает вторую
// половину S-01: автор записи берётся из сессии, а не из тела запроса, поэтому
// подмена user_id в JSON не даёт писать от чужого имени.
func TestTransactionHandler_CreateTransaction_IgnoresBodyUserID(t *testing.T) {
	service := &stubTransactionService{}
	handler := setupTransactionHandler(service)

	sessionUserID := uuid.New()
	victimUserID := uuid.New()

	body, err := json.Marshal(map[string]any{
		"amount_minor": 10_050,
		"type":         "expense",
		"description":  "Impersonation attempt",
		"category_id":  uuid.New().String(),
		"user_id":      victimUserID.String(),
		"date":         date.Today(time.UTC),
	})
	require.NoError(t, err)

	rec := postTransaction(t, handler, body, &sessionUserID)
	require.Equal(t, http.StatusCreated, rec.Code, "тело ответа: %s", rec.Body.String())

	require.NotNil(t, service.created)
	assert.Equal(t, sessionUserID, service.created.UserID, "запись обязана быть создана от имени владельца сессии")
	assert.NotEqual(t, victimUserID, service.created.UserID, "user_id из тела запроса обязан игнорироваться")

	var response handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, sessionUserID, response.Data.UserID)
}

// TestTransactionHandler_CreateTransaction_NoSession — страховка на случай,
// если хендлер когда-нибудь окажется вне группы с RequireBearer: без сессии
// автора взять неоткуда, поэтому запись создавать нельзя.
func TestTransactionHandler_CreateTransaction_NoSession(t *testing.T) {
	service := &stubTransactionService{}
	handler := setupTransactionHandler(service)

	body, err := json.Marshal(createValidTransactionRequest())
	require.NoError(t, err)

	rec := postTransaction(t, handler, body, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	assert.Nil(t, service.created)
}

func TestTransactionHandler_CreateTransaction_InvalidRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    any
		expectedStatus int
		expectedCode   string
		expectedField  string
	}{
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   handlers.ErrCodeInvalidRequest,
		},
		{
			name: "Missing amount_minor",
			requestBody: map[string]any{
				"type":        "expense",
				"description": "Test",
				"category_id": uuid.New().String(),
				"date":        "2026-01-15",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
			expectedField:  "amount_minor",
		},
		{
			name: "Zero amount_minor",
			requestBody: map[string]any{
				"amount_minor": 0,
				"type":         "expense",
				"description":  "Test",
				"category_id":  uuid.New().String(),
				"date":         "2026-01-15",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
			expectedField:  "amount_minor",
		},
		{
			name: "Negative amount_minor",
			requestBody: map[string]any{
				"amount_minor": -10_000,
				"type":         "expense",
				"description":  "Test",
				"category_id":  uuid.New().String(),
				"date":         "2026-01-15",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
			expectedField:  "amount_minor",
		},
		{
			name: "Nonexistent calendar date",
			requestBody: map[string]any{
				"amount_minor": 10_000,
				"type":         "expense",
				"description":  "Test",
				"category_id":  uuid.New().String(),
				"date":         "2026-13-01",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
			expectedField:  "date",
		},
		{
			name: "Invalid type",
			requestBody: map[string]any{
				"amount_minor": 10_000,
				"type":         "invalid",
				"description":  "Test",
				"category_id":  uuid.New().String(),
				"date":         "2026-01-15",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
			expectedField:  "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &stubTransactionService{}
			handler := setupTransactionHandler(service)

			var body []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			userID := uuid.New()
			rec := postTransaction(t, handler, body, &userID)

			// Битый JSON — 400, непрошедшее валидацию тело — 422 с деталями по полям.
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.expectedCode, response.Error.Code)
			if tt.expectedField != "" {
				require.NotEmpty(t, response.Error.Details)
				assert.Equal(t, tt.expectedField, response.Error.Details[0].Field)
			}

			assert.Nil(t, service.created, "до сервиса запрос доходить не должен")
		})
	}
}

func TestTransactionHandler_CreateTransaction_ServiceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "database failure",
			err:            errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "CREATE_FAILED",
		},
		{
			name:           "unknown category",
			err:            services.ErrCategoryNotInFamily,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   handlers.ErrCodeValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := setupTransactionHandler(&stubTransactionService{err: tt.err})

			body, err := json.Marshal(createValidTransactionRequest())
			require.NoError(t, err)

			userID := uuid.New()
			rec := postTransaction(t, handler, body, &userID)
			assert.Equal(t, tt.expectedStatus, rec.Code, rec.Body.String())

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.expectedCode, response.Error.Code)
		})
	}
}

// TestTransactionHandler_CreateTransaction_ExistingClientID — запись с присланным id уже
// есть: отвечаем 200 и не пишем второй раз (A-07).
func TestTransactionHandler_CreateTransaction_ExistingClientID(t *testing.T) {
	clientID := uuid.New()
	existing := testTransaction(clientID)
	existing.AmountMinor = 12_345
	service := &stubTransactionService{tx: existing}
	handler := setupTransactionHandler(service)

	req := createValidTransactionRequest()
	req.ID = &clientID
	body, err := json.Marshal(req)
	require.NoError(t, err)

	userID := uuid.New()
	rec := postTransaction(t, handler, body, &userID)
	assert.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, clientID, response.Data.ID)
	assert.Equal(t, money.Minor(12_345), response.Data.AmountMinor)

	assert.Nil(t, service.created, "повторный POST не создаёт вторую запись")
}

func TestTransactionHandler_GetTransactions_Success(t *testing.T) {
	expectedTransactions := []*transaction.Transaction{testTransaction(uuid.New()), testTransaction(uuid.New())}
	expectedTransactions[1].AmountMinor = 20_000
	expectedTransactions[1].Type = transaction.TypeIncome

	service := &stubTransactionService{txs: expectedTransactions, total: len(expectedTransactions)}
	handler := setupTransactionHandler(service)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.GetTransactions(e.NewContext(httpReq, rec)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[[]handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	require.Len(t, response.Data, 2)
	assert.Equal(t, expectedTransactions[0].AmountMinor, response.Data[0].AmountMinor)
	assert.Equal(t, expectedTransactions[1].AmountMinor, response.Data[1].AmountMinor)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 2, response.Meta.Pagination.Total)
}

// TestTransactionHandler_GetTransactions_FilterErrorIs422 — отказ фильтра из сервиса остаётся
// ошибкой параметров, а не 500: клиент различает их по коду.
func TestTransactionHandler_GetTransactions_FilterErrorIs422(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: dto.ErrInvalidDateRange})

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.GetTransactions(e.NewContext(httpReq, rec)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
}

func TestTransactionHandler_GetTransactions_ServiceError(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: errors.New("database error")})

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.GetTransactions(e.NewContext(httpReq, rec)))
	assert.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "FETCH_FAILED", response.Error.Code)
}

func TestTransactionHandler_GetTransactions_WithFilters(t *testing.T) {
	service := &stubTransactionService{txs: []*transaction.Transaction{}}
	handler := setupTransactionHandler(service)

	userID := uuid.New()
	categoryID := uuid.New()
	dateFrom := date.Today(time.UTC).AddDays(-30)
	dateTo := date.Today(time.UTC)

	query := url.Values{}
	query.Set("user_id", userID.String())
	query.Set("category_id", categoryID.String())
	query.Set("type", "expense")
	query.Set("date_from", dateFrom.String())
	query.Set("date_to", dateTo.String())
	query.Set("limit", "25")
	query.Set("offset", "10")

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions?"+query.Encode(), nil)
	rec := httptest.NewRecorder()

	require.NoError(t, handler.GetTransactions(e.NewContext(httpReq, rec)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.NotNil(t, service.filter)
	assert.Equal(t, &userID, service.filter.UserID)
	assert.Equal(t, &categoryID, service.filter.CategoryID)
	require.NotNil(t, service.filter.Type)
	assert.Equal(t, transaction.TypeExpense, *service.filter.Type)
	assert.Equal(t, &dateFrom, service.filter.DateFrom)
	assert.Equal(t, &dateTo, service.filter.DateTo)
	assert.Equal(t, 25, service.filter.Limit)
	assert.Equal(t, 10, service.filter.Offset)
}

func TestTransactionHandler_GetTransactions_InvalidQueryParams(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		expectedParam string
	}{
		{
			name:          "invalid user_id uuid",
			query:         "/transactions?user_id=not-a-uuid",
			expectedParam: "user_id",
		},
		{
			name:          "invalid date_from",
			query:         "/transactions?date_from=2026-13-01",
			expectedParam: "date_from",
		},
		{
			name:          "invalid amount_from_minor",
			query:         "/transactions?amount_from_minor=abc",
			expectedParam: "amount_from_minor",
		},
		{
			name:          "invalid limit number",
			query:         "/transactions?limit=0",
			expectedParam: "limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &stubTransactionService{}
			handler := setupTransactionHandler(service)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.query, nil)
			rec := httptest.NewRecorder()

			require.NoError(t, handler.GetTransactions(e.NewContext(httpReq, rec)))
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)

			require.Len(t, response.Error.Details, 1)
			assert.Equal(t, "INVALID_QUERY_PARAM", response.Error.Details[0].Code)
			assert.Equal(t, tt.expectedParam, response.Error.Details[0].Field)

			assert.Nil(t, service.filter, "до сервиса битый параметр доходить не должен")
		})
	}
}

func TestTransactionHandler_GetTransactionByID_Success(t *testing.T) {
	transactionID := uuid.New()
	expected := testTransaction(transactionID)
	handler := setupTransactionHandler(&stubTransactionService{tx: expected})

	rec := transactionByIDRequest(t, handler, transactionID.String())
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, expected.ID, response.Data.ID)
	assert.Equal(t, expected.AmountMinor, response.Data.AmountMinor)
	assert.Equal(t, string(expected.Type), response.Data.Type)
}

func TestTransactionHandler_GetTransactionByID_InvalidID(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{})

	rec := transactionByIDRequest(t, handler, "invalid")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INVALID_ID", response.Error.Code)
}

func TestTransactionHandler_GetTransactionByID_NotFound(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: services.ErrTransactionNotFound})

	rec := transactionByIDRequest(t, handler, uuid.New().String())
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "TRANSACTION_NOT_FOUND", response.Error.Code)
}

func TestTransactionHandler_UpdateTransaction_Success(t *testing.T) {
	transactionID := uuid.New()
	updated := testTransaction(transactionID)
	updated.AmountMinor = 20_000
	updated.Description = "Updated description"
	updated.Tags = []string{"updated", "test"}

	service := &stubTransactionService{tx: updated}
	handler := setupTransactionHandler(service)

	rec := updateTransactionRequest(t, handler, transactionID.String(),
		`{"amount_minor":20000,"description":"Updated description","tags":["updated","test"]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, transactionID, response.Data.ID)
	assert.Equal(t, money.Minor(20_000), response.Data.AmountMinor)
	assert.Equal(t, "Updated description", response.Data.Description)

	require.NotNil(t, service.updated)
	require.NotNil(t, service.updated.AmountMinor)
	assert.Equal(t, money.Minor(20_000), *service.updated.AmountMinor)
	assert.Equal(t, []string{"updated", "test"}, service.updated.Tags)
}

func TestTransactionHandler_UpdateTransaction_NotFound(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: services.ErrTransactionNotFound})

	rec := updateTransactionRequest(t, handler, uuid.New().String(), `{"amount_minor":20000}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestTransactionHandler_UpdateTransaction_InvalidAmountIs422(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: services.ErrInvalidTransactionAmount})

	rec := updateTransactionRequest(t, handler, uuid.New().String(), `{"amount_minor":20000}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
}

func TestTransactionHandler_DeleteTransaction_Success(t *testing.T) {
	transactionID := uuid.New()
	service := &stubTransactionService{}
	handler := setupTransactionHandler(service)

	rec := deleteTransactionRequest(t, handler, transactionID.String())
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, service.deleted)
	assert.Equal(t, transactionID, *service.deleted)
}

func TestTransactionHandler_DeleteTransaction_InvalidID(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{})

	rec := deleteTransactionRequest(t, handler, "invalid")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTransactionHandler_DeleteTransaction_NotFound(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: services.ErrTransactionNotFound})

	rec := deleteTransactionRequest(t, handler, uuid.New().String())
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTransactionHandler_UpdateTransaction_ServiceError(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: errors.New("database error")})

	rec := updateTransactionRequest(t, handler, uuid.New().String(), `{"amount_minor":20000}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "UPDATE_FAILED", response.Error.Code)
}

func TestTransactionHandler_DeleteTransaction_ServiceError(t *testing.T) {
	handler := setupTransactionHandler(&stubTransactionService{err: errors.New("database error")})

	rec := deleteTransactionRequest(t, handler, uuid.New().String())
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "DELETE_FAILED", response.Error.Code)
}

// TestNewTransactionHandler_NilServicePanics — хендлер без сервиса отвечать не может,
// и обнаружиться это должно при сборке приложения, а не на первом запросе.
func TestNewTransactionHandler_NilServicePanics(t *testing.T) {
	assert.Panics(t, func() {
		handlers.NewTransactionHandler(nil)
	})
}

// TestNewBudgetHandler_NilServicePanics — то же для бюджетов.
func TestNewBudgetHandler_NilServicePanics(t *testing.T) {
	assert.Panics(t, func() {
		handlers.NewBudgetHandler(&handlers.Repositories{}, nil)
	})
}

func transactionByIDRequest(
	t *testing.T,
	handler *handlers.TransactionHandler,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions/"+id, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)

	require.NoError(t, handler.GetTransactionByID(c))

	return rec
}

func updateTransactionRequest(
	t *testing.T,
	handler *handlers.TransactionHandler,
	id, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPut, "/transactions/"+id, bytes.NewBufferString(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)

	require.NoError(t, handler.UpdateTransaction(c))

	return rec
}

func deleteTransactionRequest(
	t *testing.T,
	handler *handlers.TransactionHandler,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodDelete, "/transactions/"+id, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(id)

	require.NoError(t, handler.DeleteTransaction(c))

	return rec
}
