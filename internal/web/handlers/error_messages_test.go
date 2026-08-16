package handlers_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// Текст ошибки сервиса наружу не отдаётся.
//
// Регрессия: TransactionHandler.Create подставлял err.Error() (и UserID) в
// сообщение echo.HTTPError, а getBudgetServiceErrorMessage /
// getReportServiceErrorMessage дописывали err.Error() в КАЖДУЮ ветку, включая
// default. customHTTPErrorHandler берёт he.Message как есть и рисует его на
// странице ошибки, поэтому обёрнутая ошибка репозитория показывала клиенту
// имена таблиц и колонок SQLite и текст ограничений.

// leakyRepoErrorText — то, что реально прилетает из сервиса поверх ошибки SQLite.
const leakyRepoErrorText = "failed to update budget: " +
	"UNIQUE constraint failed: budgets.category_id, budgets.start_date"

// httpErrorMessage достаёт текст сообщения из ошибки хендлера.
func httpErrorMessage(t *testing.T, err error) string {
	t.Helper()

	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)

	return fmt.Sprintf("%v", httpErr.Message)
}

func TestTransactionHandler_Create_ServiceError_DoesNotLeakDetails(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	sessionUserID := uuid.New()
	mockTxService.On("CreateTransaction", mock.Anything, mock.Anything).Return(nil, errors.New(leakyRepoErrorText))

	formData := url.Values{
		"amount":      {"100"},
		"type":        {"income"},
		"category_id": {uuid.New().String()},
		"date":        {"2024-01-15"},
		"description": {"Test"},
	}

	c, _ := newTestContext(http.MethodPost, "/transactions", formData.Encode())
	withSession(c, sessionUserID, user.RoleAdmin)

	err := handler.Create(c)

	require.Error(t, err)
	message := httpErrorMessage(t, err)
	assert.NotContains(t, message, "UNIQUE constraint", "текст ошибки БД уходит клиенту")
	assert.NotContains(t, message, "budgets.category_id")
	assert.NotContains(t, message, sessionUserID.String(), "UserID уходит клиенту")
}

// getTransactionServiceErrorMessage дописывал err.Error() в каждую ветку,
// включая default, а handleDelete кладёт это сообщение в echo.HTTPError и в
// components/alert — то есть текст ошибки SQLite доходил до браузера на
// DELETE /transactions/:id и DELETE /htmx/transactions/:id.
func TestTransactionHandler_Delete_ServiceError_DoesNotLeakDetails(t *testing.T) {
	handler, mockTxService, _, _ := setupTransactionHandler()

	transactionID := uuid.New()
	existing := createTestTransaction(time.Now(), 100.00, transaction.TypeExpense, uuid.New())
	mockTxService.On("GetTransactionByID", mock.Anything, transactionID).Return(existing, nil)
	mockTxService.On("DeleteTransaction", mock.Anything, transactionID).Return(errors.New(leakyRepoErrorText))

	c, _ := newTestContext(http.MethodDelete, "/transactions/"+transactionID.String(), "")
	c.SetParamNames("id")
	c.SetParamValues(transactionID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Delete(c)

	require.Error(t, err)
	message := httpErrorMessage(t, err)
	assert.NotContains(t, message, "UNIQUE constraint", "текст ошибки БД уходит клиенту")
	assert.NotContains(t, message, "budgets.category_id")
}

func TestBudgetHandler_Delete_ServiceError_DoesNotLeakDetails(t *testing.T) {
	handler, mockBudgetService, _, _ := setupBudgetHandler()

	budgetID := uuid.New()
	existingBudget := createTestBudgetWithSpent("To Delete", 1000, 0, nil, true)
	existingBudget.ID = budgetID

	mockBudgetService.On("GetBudgetByID", mock.Anything, budgetID).Return(existingBudget, nil)
	mockBudgetService.On("DeleteBudget", mock.Anything, budgetID).Return(errors.New(leakyRepoErrorText))

	c, _ := newTestContext(http.MethodDelete, "/budgets/"+budgetID.String(), "")
	c.SetParamNames("id")
	c.SetParamValues(budgetID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Delete(c)

	require.Error(t, err)
	message := httpErrorMessage(t, err)
	assert.NotContains(t, message, "UNIQUE constraint", "текст ошибки БД уходит клиенту")
	assert.NotContains(t, message, "budgets.category_id")
}

func TestReportHandler_Delete_ServiceError_DoesNotLeakDetails(t *testing.T) {
	handler, mockReportService, mockUserService := setupReportHandlerWithUser()

	reportID := uuid.New()
	mockReportService.On("GetReportByID", mock.Anything, reportID).
		Return(&report.Report{
			ID:     reportID,
			Name:   "Февраль",
			Type:   report.TypeExpenses,
			Period: report.PeriodMonthly,
			UserID: uuid.New(),
		}, nil)
	mockReportService.On("DeleteReport", mock.Anything, reportID).Return(errors.New(leakyRepoErrorText))
	mockUserService.On("GetUserByID", mock.Anything, mock.Anything).
		Return(&user.User{ID: uuid.New(), FirstName: "Test", LastName: "User"}, nil).Maybe()

	c, _ := newTestContext(http.MethodDelete, "/reports/"+reportID.String(), "")
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())
	withSession(c, uuid.New(), user.RoleAdmin)

	err := handler.Delete(c)

	require.Error(t, err)
	message := httpErrorMessage(t, err)
	assert.NotContains(t, message, "UNIQUE constraint", "текст ошибки БД уходит клиенту")
	assert.NotContains(t, message, "budgets.category_id")
}
