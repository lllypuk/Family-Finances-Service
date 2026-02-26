package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/transaction"
)

// MockTransactionRepository is a mock implementation of transaction repository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTotalByDateRange(
	ctx context.Context,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, startDate, endDate, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepository) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepository) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepository) GetAll(ctx context.Context, limit, offset int) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetTotalsByCategory(
	ctx context.Context,
	familyID uuid.UUID,
	period string,
) (map[uuid.UUID]float64, error) {
	args := m.Called(ctx, familyID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]float64), args.Error(1)
}

// setupTransactionHandler creates a new transaction handler with mock repositories
func setupTransactionHandler() (*handlers.TransactionHandler, *MockTransactionRepository) {
	mockRepo := &MockTransactionRepository{}
	repositories := &handlers.Repositories{
		Transaction: mockRepo,
	}
	handler := handlers.NewTransactionHandler(repositories)
	return handler, mockRepo
}

// createValidTransactionRequest creates a valid transaction request for testing
func createValidTransactionRequest() handlers.CreateTransactionRequest {
	return handlers.CreateTransactionRequest{
		Amount:      100.50,
		Type:        "expense",
		Description: "Test transaction",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		Date:        time.Now(),
		Tags:        []string{"test", "expense"},
	}
}

func TestTransactionHandler_CreateTransaction_Success(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	req := createValidTransactionRequest()
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response handlers.APIResponse[handlers.TransactionResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.InDelta(t, req.Amount, response.Data.Amount, 0.01)
	assert.Equal(t, req.Type, response.Data.Type)
	assert.Equal(t, req.Description, response.Data.Description)
	assert.Equal(t, req.CategoryID, response.Data.CategoryID)
	assert.Equal(t, req.UserID, response.Data.UserID)
	assert.Equal(t, req.Tags, response.Data.Tags)

	mockRepo.AssertExpectations(t)
}

func TestTransactionHandler_CreateTransaction_InvalidRequest(t *testing.T) {
	handler, _ := setupTransactionHandler()

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
			name: "Missing amount",
			requestBody: map[string]any{
				"type":        "expense",
				"description": "Test",
				"category_id": uuid.New().String(),
				"user_id":     uuid.New().String(),
				"family_id":   uuid.New().String(),
				"date":        time.Now(),
			},
			expectedMsg: "",
		},
		{
			name: "Negative amount",
			requestBody: map[string]any{
				"amount":      -100.0,
				"type":        "expense",
				"description": "Test",
				"category_id": uuid.New().String(),
				"user_id":     uuid.New().String(),
				"family_id":   uuid.New().String(),
				"date":        time.Now(),
			},
			expectedMsg: "",
		},
		{
			name: "Invalid type",
			requestBody: map[string]any{
				"amount":      100.0,
				"type":        "invalid",
				"description": "Test",
				"category_id": uuid.New().String(),
				"user_id":     uuid.New().String(),
				"family_id":   uuid.New().String(),
				"date":        time.Now(),
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
			httpReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			// Act
			err = handler.CreateTransaction(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestTransactionHandler_CreateTransaction_RepositoryError(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	req := createValidTransactionRequest()
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).
		Return(errors.New("database error"))

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "CREATE_FAILED", response.Error.Code)
}

func TestTransactionHandler_GetTransactions_Success(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	familyID := uuid.New()
	expectedTransactions := []*transaction.Transaction{
		{
			ID:          uuid.New(),
			Amount:      100.0,
			Type:        transaction.TypeExpense,
			Description: "Test transaction 1",
			CategoryID:  uuid.New(),
			UserID:      uuid.New(),
			Date:        time.Now(),
			Tags:        []string{"test"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Amount:      200.0,
			Type:        transaction.TypeIncome,
			Description: "Test transaction 2",
			CategoryID:  uuid.New(),
			UserID:      uuid.New(),
			Date:        time.Now(),
			Tags:        []string{"test"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	mockRepo.On("GetByFilter", mock.Anything, mock.AnythingOfType("transaction.Filter")).
		Return(expectedTransactions, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions?family_id=%s", familyID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetTransactions(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.TransactionResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Data, 2)
	assert.InDelta(t, expectedTransactions[0].Amount, response.Data[0].Amount, 0.01)
	assert.InDelta(t, expectedTransactions[1].Amount, response.Data[1].Amount, 0.01)

	mockRepo.AssertExpectations(t)
}

// TestTransactionHandler_GetTransactions_MissingFamilyID is deprecated in single-family model
/*
func TestTransactionHandler_GetTransactions_MissingFamilyID(t *testing.T) {
	handler, _ := setupTransactionHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetTransactions(c)

	// Assert - the handler should return an error, but JSON response should be set
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "MISSING_FAMILY_ID", response.Error.Code)
}
*/

// TestTransactionHandler_GetTransactions_InvalidFamilyID is deprecated in single-family model
/*
func TestTransactionHandler_GetTransactions_InvalidFamilyID(t *testing.T) {
	handler, _ := setupTransactionHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions?family_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetTransactions(c)

	// Assert - the handler should return an error, but JSON response should be set
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_FAMILY_ID", response.Error.Code)
}
*/

func TestTransactionHandler_GetTransactions_WithFilters(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	dateFrom := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	dateTo := time.Now().Format(time.RFC3339)

	mockRepo.On("GetByFilter", mock.Anything, mock.AnythingOfType("transaction.Filter")).
		Return([]*transaction.Transaction{}, nil)

	e := echo.New()
	query := url.Values{}
	query.Set("family_id", familyID.String())
	query.Set("user_id", userID.String())
	query.Set("category_id", categoryID.String())
	query.Set("type", "expense")
	query.Set("date_from", dateFrom)
	query.Set("date_to", dateTo)
	query.Set("limit", "25")
	query.Set("offset", "10")
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions?"+query.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetTransactions(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
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
			query:         "/transactions?date_from=2026-01-01",
			expectedParam: "date_from",
		},
		{
			name:          "invalid amount_from",
			query:         "/transactions?amount_from=abc",
			expectedParam: "amount_from",
		},
		{
			name:          "invalid limit number",
			query:         "/transactions?limit=0",
			expectedParam: "limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockRepo := setupTransactionHandler()

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err := handler.GetTransactions(c)

			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, "INVALID_QUERY_PARAM", response.Error.Code)

			details, ok := response.Error.Details.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.expectedParam, details["param"])

			mockRepo.AssertNotCalled(t, "GetByFilter", mock.Anything, mock.Anything)
		})
	}
}

func TestTransactionHandler_GetTransactionByID_Success(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	transactionID := uuid.New()
	expectedTransaction := &transaction.Transaction{
		ID:          transactionID,
		Amount:      150.0,
		Type:        transaction.TypeExpense,
		Description: "Test transaction",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		Date:        time.Now(),
		Tags:        []string{"test"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("GetByID", mock.Anything, transactionID).Return(expectedTransaction, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions/"+transactionID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())

	// Act
	err := handler.GetTransactionByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.TransactionResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedTransaction.ID, response.Data.ID)
	assert.InDelta(t, expectedTransaction.Amount, response.Data.Amount, 0.01)
	assert.Equal(t, string(expectedTransaction.Type), response.Data.Type)

	mockRepo.AssertExpectations(t)
}

func TestTransactionHandler_GetTransactionByID_InvalidID(t *testing.T) {
	handler, _ := setupTransactionHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.GetTransactionByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_ID", response.Error.Code)
}

func TestTransactionHandler_GetTransactionByID_NotFound(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	transactionID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, transactionID).Return(nil, errors.New("not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/transactions/"+transactionID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())

	// Act
	err := handler.GetTransactionByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "TRANSACTION_NOT_FOUND", response.Error.Code)

	mockRepo.AssertExpectations(t)
}

func TestTransactionHandler_UpdateTransaction_Success(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	transactionID := uuid.New()
	existingTransaction := &transaction.Transaction{
		ID:          transactionID,
		Amount:      100.0,
		Type:        transaction.TypeExpense,
		Description: "Old description",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		Date:        time.Now(),
		Tags:        []string{"old"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	updateReq := handlers.UpdateTransactionRequest{
		Amount:      new(200.0),
		Description: new("Updated description"),
		Tags:        []string{"updated", "test"},
	}

	mockRepo.On("GetByID", mock.Anything, transactionID).Return(existingTransaction, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(tx *transaction.Transaction) bool {
		return tx.ID == transactionID &&
			tx.Amount == 200.0 &&
			tx.Description == "Updated description" &&
			len(tx.Tags) == 2 &&
			tx.Tags[0] == "updated" &&
			tx.Tags[1] == "test"
	})).Return(nil)

	// Prepare HTTP request
	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPatch, "/transactions/"+transactionID.String(), bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())

	// Act
	err = handler.UpdateTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.TransactionResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, transactionID, response.Data.ID)
	assert.InDelta(t, 200.0, response.Data.Amount, 0.01)
	assert.Equal(t, "Updated description", response.Data.Description)

	mockRepo.AssertExpectations(t)
}

func TestTransactionHandler_DeleteTransaction_Success(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	transactionID := uuid.New()
	mockRepo.On("Delete", mock.Anything, transactionID).Return(nil)

	e := echo.New()
	httpReq := httptest.NewRequest(
		http.MethodDelete,
		"/transactions/"+transactionID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())

	// Act
	err := handler.DeleteTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	mockRepo.AssertExpectations(t)
}

func TestTransactionHandler_DeleteTransaction_InvalidID(t *testing.T) {
	handler, _ := setupTransactionHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodDelete, "/transactions/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.DeleteTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTransactionHandler_DeleteTransaction_RepositoryError(t *testing.T) {
	handler, mockRepo := setupTransactionHandler()

	// Arrange
	transactionID := uuid.New()
	mockRepo.On("Delete", mock.Anything, transactionID).Return(errors.New("database error"))

	e := echo.New()
	httpReq := httptest.NewRequest(
		http.MethodDelete,
		"/transactions/"+transactionID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())

	// Act
	err := handler.DeleteTransaction(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "DELETE_FAILED", response.Error.Code)

	mockRepo.AssertExpectations(t)
}

// Helper function for creating float64 pointers
//
//go:fix inline
func floatPtr(f float64) *float64 {
	return new(f)
}

// Benchmark tests for performance validation
func BenchmarkTransactionHandler_CreateTransaction(b *testing.B) {
	handler, mockRepo := setupTransactionHandler()

	// Setup mock to return nil for all calls
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*transaction.Transaction")).Return(nil)

	req := createValidTransactionRequest()
	body, _ := json.Marshal(req)

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(body))
		httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.CreateTransaction(c)
	}
}

func BenchmarkTransactionHandler_GetTransactions(b *testing.B) {
	handler, mockRepo := setupTransactionHandler()

	// Setup mock to return empty slice for all calls
	mockRepo.On("GetByFilter", mock.Anything, mock.AnythingOfType("transaction.Filter")).
		Return([]*transaction.Transaction{}, nil)

	familyID := uuid.New()

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions?family_id=%s", familyID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.GetTransactions(c)
	}
}
