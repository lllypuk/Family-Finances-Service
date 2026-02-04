package handlers_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appHandlers "family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/handlers"
)

// setupTransactionHandler creates a test transaction handler with mocks
func setupTransactionHandler() (*handlers.TransactionHandler, *MockTransactionService, *MockCategoryService, *MockUserService) {
	mockTransactionService := new(MockTransactionService)
	mockCategoryService := new(MockCategoryService)
	mockUserService := new(MockUserService)

	servicesStruct := &services.Services{
		Transaction: mockTransactionService,
		Category:    mockCategoryService,
		User:        mockUserService,
	}

	handler := handlers.NewTransactionHandler(
		&appHandlers.Repositories{},
		servicesStruct,
	)

	return handler, mockTransactionService, mockCategoryService, mockUserService
}

// TestTransactionHandler_Index tests the main index page
func TestTransactionHandler_Index(t *testing.T) {
	handler, mockTxService, mockCatService, mockUserService := setupTransactionHandler()

	// Setup mocks
	transactions := []*transaction.Transaction{
		createTestTransaction(time.Now(), 100.50, transaction.TypeExpense, uuid.New()),
		createTestTransaction(time.Now(), 200.00, transaction.TypeIncome, uuid.New()),
	}
	mockTxService.On("GetAllTransactions", mock.Anything, mock.Anything).Return(transactions, nil)

	categories := []*category.Category{
		createTestCategory("Food", category.TypeExpense),
	}
	mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
	mockCatService.On("GetCategoryByID", mock.Anything, mock.Anything).Return(categories[0], nil).Maybe()

	testUser := &user.User{ID: uuid.New(), Email: "test@example.com", FirstName: "Test"}
	mockUserService.On("GetUserByID", mock.Anything, mock.Anything).Return(testUser, nil).Maybe()

	c, rec := newTestContext(http.MethodGet, "/transactions", "")
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Index(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTransactionHandler_New tests the new transaction form
func TestTransactionHandler_New(t *testing.T) {
	handler, _, mockCatService, _ := setupTransactionHandler()

	categories := []*category.Category{createTestCategory("Food", category.TypeExpense)}
	mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

	c, rec := newTestContext(http.MethodGet, "/transactions/new", "")
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.New(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTransactionHandler_Create_ValidIncome tests creating a valid income transaction
func TestTransactionHandler_Create_ValidIncome(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	tx := createTestTransaction(time.Now(), 1000.50, transaction.TypeIncome, uuid.New())
	mockTxService.On("CreateTransaction", mock.Anything, mock.Anything).Return(tx, nil)

	formData := url.Values{
		"amount":      {"1000.50"},
		"type":        {"income"},
		"category_id": {uuid.New().String()},
		"date":        {"2024-01-15"},
		"description": {"Salary"},
	}

	c, rec := newTestContext(http.MethodPost, "/transactions", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Create(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/transactions", rec.Header().Get("Location"))
}

// TestTransactionHandler_Create_ValidExpense tests creating a valid expense transaction
func TestTransactionHandler_Create_ValidExpense(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	tx := createTestTransaction(time.Now(), 50.00, transaction.TypeExpense, uuid.New())
	mockTxService.On("CreateTransaction", mock.Anything, mock.Anything).Return(tx, nil)

	formData := url.Values{
		"amount":      {"50.00"},
		"type":        {"expense"},
		"category_id": {uuid.New().String()},
		"date":        {"2024-01-16"},
		"description": {"Groceries"},
	}

	c, rec := newTestContext(http.MethodPost, "/transactions", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Create(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
}

// TestTransactionHandler_Create_ServiceError tests service error handling
func TestTransactionHandler_Create_ServiceError(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	mockTxService.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	formData := url.Values{
		"amount":      {"100"},
		"type":        {"income"},
		"category_id": {uuid.New().String()},
		"date":        {"2024-01-15"},
		"description": {"Test"},
	}

	c, _ := newTestContext(http.MethodPost, "/transactions", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Create(c)

	require.Error(t, err)
}

// TestTransactionHandler_Edit tests the edit form
func TestTransactionHandler_Edit(t *testing.T) {
	handler, mockTxService, mockCatService, _ := setupTransactionHandler()

	transactionID := uuid.New()
	tx := createTestTransaction(time.Now(), 100.50, transaction.TypeExpense, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, transactionID).Return(tx, nil)

	categories := []*category.Category{createTestCategory("Food", category.TypeExpense)}
	mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)

	c, rec := newTestContext(http.MethodGet, "/transactions/"+transactionID.String()+"/edit", "")
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Edit(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTransactionHandler_Update tests updating a transaction
func TestTransactionHandler_Update(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	transactionID := uuid.New()
	tx := createTestTransaction(time.Now(), 1000.00, transaction.TypeIncome, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, transactionID).Return(tx, nil)

	updatedTx := createTestTransaction(time.Now(), 2000.00, transaction.TypeIncome, uuid.New())
	mockTxService.On("UpdateTransaction", mock.Anything, transactionID, mock.Anything).Return(updatedTx, nil)

	formData := url.Values{
		"amount":      {"2000.00"},
		"type":        {"income"},
		"category_id": {uuid.New().String()},
		"date":        {"2024-01-15"},
		"description": {"Updated"},
	}

	c, rec := newTestContext(http.MethodPost, "/transactions/"+transactionID.String(), formData.Encode())
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Update(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
}

// TestTransactionHandler_Delete tests deleting a transaction
func TestTransactionHandler_Delete(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	transactionID := uuid.New()
	tx := createTestTransaction(time.Now(), 100.00, transaction.TypeExpense, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, transactionID).Return(tx, nil)
	mockTxService.On("DeleteTransaction", mock.Anything, transactionID).Return(nil)

	c, rec := newTestContext(http.MethodDelete, "/transactions/"+transactionID.String(), "")
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Delete(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
}

// TestTransactionHandler_BulkDelete_AllSuccess tests successful bulk deletion
func TestTransactionHandler_BulkDelete_AllSuccess(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	// Three successful deletions
	tx := createTestTransaction(time.Now(), 100.00, transaction.TypeExpense, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, mock.Anything).Return(tx, nil).Times(3)
	mockTxService.On("DeleteTransaction", mock.Anything, mock.Anything).Return(nil).Times(3)

	formData := url.Values{}
	for range 3 {
		formData.Add("transaction_ids", uuid.New().String())
	}

	c, rec := newTestContext(http.MethodPost, "/transactions/bulk-delete", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.BulkDelete(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
}

// TestTransactionHandler_BulkDelete_PartialFailure tests partial bulk deletion failure
func TestTransactionHandler_BulkDelete_PartialFailure(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	// Two succeed, one fails
	tx := createTestTransaction(time.Now(), 100.00, transaction.TypeExpense, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, mock.Anything).Return(tx, nil).Times(2)
	mockTxService.On("DeleteTransaction", mock.Anything, mock.Anything).Return(nil).Times(2)
	mockTxService.On("GetTransactionByID", mock.Anything, mock.Anything).Return(nil, errors.New("not found")).Once()

	formData := url.Values{}
	for range 3 {
		formData.Add("transaction_ids", uuid.New().String())
	}

	c, rec := newTestContext(http.MethodPost, "/transactions/bulk-delete", formData.Encode())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.BulkDelete(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
}

// TestTransactionHandler_Filter_HTMX tests HTMX filter endpoint
func TestTransactionHandler_Filter_HTMX(t *testing.T) {
	handler, mockTxService, mockCatService, mockUserService := setupTransactionHandler()

	transactions := []*transaction.Transaction{
		createTestTransaction(time.Now(), 100.50, transaction.TypeExpense, uuid.New()),
	}
	mockTxService.On("GetAllTransactions", mock.Anything, mock.Anything).Return(transactions, nil)

	categories := []*category.Category{createTestCategory("Food", category.TypeExpense)}
	mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
	mockCatService.On("GetCategoryByID", mock.Anything, mock.Anything).Return(categories[0], nil)

	testUser := &user.User{ID: uuid.New(), Email: "test@example.com", FirstName: "Test"}
	mockUserService.On("GetUserByID", mock.Anything, mock.Anything).Return(testUser, nil)

	c, rec := newTestContext(http.MethodGet, "/transactions/filter?type=expense", "")
	withSession(c, uuid.New(), user.RoleAdmin)
	withHTMX(c)

	err := handler.Filter(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTransactionHandler_List_HTMX tests HTMX list endpoint
func TestTransactionHandler_List_HTMX(t *testing.T) {
	handler, mockTxService, mockCatService, mockUserService := setupTransactionHandler()

	transactions := []*transaction.Transaction{
		createTestTransaction(time.Now(), 100.50, transaction.TypeExpense, uuid.New()),
	}
	mockTxService.On("GetAllTransactions", mock.Anything, mock.Anything).Return(transactions, nil)

	categories := []*category.Category{createTestCategory("Food", category.TypeExpense)}
	mockCatService.On("GetCategories", mock.Anything, mock.Anything).Return(categories, nil)
	mockCatService.On("GetCategoryByID", mock.Anything, mock.Anything).Return(categories[0], nil)

	testUser := &user.User{ID: uuid.New(), Email: "test@example.com", FirstName: "Test"}
	mockUserService.On("GetUserByID", mock.Anything, mock.Anything).Return(testUser, nil)

	c, rec := newTestContext(http.MethodGet, "/transactions/list?page=1", "")
	withSession(c, uuid.New(), user.RoleAdmin)
	withHTMX(c)

	err := handler.List(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
