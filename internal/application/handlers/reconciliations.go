package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/services"
)

const (
	fieldMonth            = "month"
	fieldBankExpenseMinor = "bank_expense_minor"
	monthFormatReason     = "must be a month in YYYY-MM format"
)

// ReconciliationHandler — сверки счетов и GET /stats/reconciliation.
type ReconciliationHandler struct {
	reconciliations services.ReconciliationService
	validator       *validator.Validate
}

func NewReconciliationHandler(reconciliations services.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{reconciliations: reconciliations, validator: newAPIValidator()}
}

func (h *ReconciliationHandler) PutReconciliation(c echo.Context) error {
	accountID, month, err := parseReconciliationPath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	var req ReconciliationRequest
	if err = c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	rec, err := h.reconciliations.Put(c.Request().Context(), accountID, month, *req.BankExpenseMinor, req.Note)
	if err != nil {
		return respondReconciliationError(c, err)
	}

	return respondAPI(c, http.StatusOK, ReconciliationResponse{
		AccountID:        rec.AccountID,
		Month:            rec.Month,
		BankExpenseMinor: rec.BankExpenseMinor,
		Note:             rec.Note,
		UpdatedAt:        rec.UpdatedAt,
	})
}

func (h *ReconciliationHandler) DeleteReconciliation(c echo.Context) error {
	accountID, month, err := parseReconciliationPath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	if err = h.reconciliations.Delete(c.Request().Context(), accountID, month); err != nil {
		return respondReconciliationError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetReconciliationStats — записанное против банка за ?month=; без него — текущий месяц в поясе семьи.
func (h *ReconciliationHandler) GetReconciliationStats(c echo.Context) error {
	var month *date.Date
	if raw := c.QueryParam(fieldMonth); raw != "" {
		first, _, err := date.ParseMonth(raw)
		if err != nil {
			return ignoreWritten(writeInvalidQueryParam(c, fieldMonth, raw, monthFormatReason))
		}
		month = &first
	}

	stats, err := h.reconciliations.Summary(c.Request().Context(), month)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondAPI(c, http.StatusOK, stats)
}

// parseReconciliationPath пишет ответ сам и возвращает errResponseAlreadyWritten.
func parseReconciliationPath(c echo.Context) (uuid.UUID, date.Date, error) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID))
	}

	month, _, err := date.ParseMonth(c.Param(fieldMonth))
	if err != nil {
		return uuid.Nil, date.Date{}, written(respondError(c, http.StatusUnprocessableEntity,
			ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldMonth, Message: monthFormatReason, Code: ErrCodeValidationError}))
	}

	return accountID, month, nil
}

// written — ошибка записи ответа либо sentinel, если ответ ушёл.
func written(writeErr error) error {
	if writeErr != nil {
		return writeErr
	}

	return errResponseAlreadyWritten
}

func respondReconciliationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, account.ErrNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeAccountNotFound, ErrMessageAccountNotFound)
	case errors.Is(err, reconciliation.ErrNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeReconciliationNotFound, ErrMessageReconciliationNotFound)
	case errors.Is(err, reconciliation.ErrAmountOutOfRange):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldBankExpenseMinor, Message: err.Error(), Code: ErrCodeValidationError})
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}
