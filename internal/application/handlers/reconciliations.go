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
	fieldMonth        = "month"
	fieldBalanceMinor = "balance_minor"
	monthFormatReason = "must be a month in YYYY-MM format"
	monthFutureReason = "must not be after the current month"
)

// ReconciliationHandler — остатки счетов и GET /stats/reconciliation.
type ReconciliationHandler struct {
	reconciliations services.ReconciliationService
	validator       *validator.Validate
}

func NewReconciliationHandler(reconciliations services.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{reconciliations: reconciliations, validator: newAPIValidator()}
}

func (h *ReconciliationHandler) PutAccountBalance(c echo.Context) error {
	accountID, month, err := parseBalancePath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	var req AccountBalanceRequest
	if err = c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	b, err := h.reconciliations.PutBalance(c.Request().Context(), accountID, month, *req.BalanceMinor)
	if err != nil {
		return respondBalanceError(c, err)
	}

	return respondAPI(c, http.StatusOK, AccountBalanceResponse{
		AccountID:    b.AccountID,
		Month:        b.Month,
		BalanceMinor: b.BalanceMinor,
		UpdatedAt:    b.UpdatedAt,
	})
}

func (h *ReconciliationHandler) DeleteAccountBalance(c echo.Context) error {
	accountID, month, err := parseBalancePath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	if err = h.reconciliations.DeleteBalance(c.Request().Context(), accountID, month); err != nil {
		return respondBalanceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetReconciliationStats — остатки против операций за ?month=; без него — текущий месяц в поясе семьи.
func (h *ReconciliationHandler) GetReconciliationStats(c echo.Context) error {
	var month *date.Date
	raw := c.QueryParam(fieldMonth)
	if raw != "" {
		first, _, err := date.ParseMonth(raw)
		if err != nil {
			return ignoreWritten(writeInvalidQueryParam(c, fieldMonth, raw, monthFormatReason))
		}
		month = &first
	}

	stats, err := h.reconciliations.Summary(c.Request().Context(), month)
	switch {
	case errors.Is(err, services.ErrReconciliationMonthInFuture):
		return ignoreWritten(writeInvalidQueryParam(c, fieldMonth, raw, monthFutureReason))
	case err != nil:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondAPI(c, http.StatusOK, stats)
}

// parseBalancePath пишет ответ сам и возвращает errResponseAlreadyWritten.
func parseBalancePath(c echo.Context) (uuid.UUID, date.Date, error) {
	accountID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID))
	}

	month, _, err := date.ParseMonth(c.Param(fieldMonth))
	if err != nil {
		return uuid.Nil, date.Date{}, written(respondMonthError(c, monthFormatReason))
	}

	return accountID, month, nil
}

func respondMonthError(c echo.Context, reason string) error {
	return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: fieldMonth, Message: reason, Code: ErrCodeValidationError})
}

// written — ошибка записи ответа либо sentinel, если ответ ушёл.
func written(writeErr error) error {
	if writeErr != nil {
		return writeErr
	}

	return errResponseAlreadyWritten
}

func respondBalanceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, account.ErrNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeAccountNotFound, ErrMessageAccountNotFound)
	case errors.Is(err, reconciliation.ErrBalanceNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeBalanceNotFound, ErrMessageBalanceNotFound)
	case errors.Is(err, services.ErrReconciliationMonthInFuture):
		return respondMonthError(c, monthFutureReason)
	case errors.Is(err, reconciliation.ErrBalanceOutOfRange):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldBalanceMinor, Message: err.Error(), Code: ErrCodeValidationError})
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}
