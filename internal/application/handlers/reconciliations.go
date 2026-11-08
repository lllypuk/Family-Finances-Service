package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

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

func (h *ReconciliationHandler) PutAccountBalance(w http.ResponseWriter, r *http.Request) error {
	accountID, month, err := parseBalancePath(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	var req AccountBalanceRequest
	if err = decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}

	b, err := h.reconciliations.PutBalance(r.Context(), accountID, month, *req.BalanceMinor)
	if err != nil {
		return respondBalanceError(w, err)
	}

	return respondAPI(w, http.StatusOK, AccountBalanceResponse{
		AccountID:    b.AccountID,
		Month:        b.Month,
		BalanceMinor: b.BalanceMinor,
		UpdatedAt:    b.UpdatedAt,
	})
}

func (h *ReconciliationHandler) DeleteAccountBalance(w http.ResponseWriter, r *http.Request) error {
	accountID, month, err := parseBalancePath(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	if err = h.reconciliations.DeleteBalance(r.Context(), accountID, month); err != nil {
		return respondBalanceError(w, err)
	}

	return noContent(w)
}

// GetReconciliationStats — остатки против операций за ?month=; без него — текущий месяц в поясе семьи.
func (h *ReconciliationHandler) GetReconciliationStats(w http.ResponseWriter, r *http.Request) error {
	var month *date.Date
	raw := r.URL.Query().Get(fieldMonth)
	if raw != "" {
		first, _, err := date.ParseMonth(raw)
		if err != nil {
			return ignoreWritten(writeInvalidQueryParam(w, fieldMonth, raw, monthFormatReason))
		}
		month = &first
	}

	stats, err := h.reconciliations.Summary(r.Context(), month)
	switch {
	case errors.Is(err, services.ErrReconciliationMonthInFuture):
		return ignoreWritten(writeInvalidQueryParam(w, fieldMonth, raw, monthFutureReason))
	case err != nil:
		return RespondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondAPI(w, http.StatusOK, stats)
}

// parseBalancePath пишет ответ сам и возвращает errResponseAlreadyWritten.
func parseBalancePath(w http.ResponseWriter, r *http.Request) (uuid.UUID, date.Date, error) {
	accountID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			RespondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidAccountID))
	}

	month, _, err := date.ParseMonth(r.PathValue(fieldMonth))
	if err != nil {
		return uuid.Nil, date.Date{}, written(respondMonthError(w, monthFormatReason))
	}

	return accountID, month, nil
}

func respondMonthError(w http.ResponseWriter, reason string) error {
	return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: fieldMonth, Message: reason, Code: ErrCodeValidationError})
}

// written — ошибка записи ответа либо sentinel, если ответ ушёл.
func written(writeErr error) error {
	if writeErr != nil {
		return writeErr
	}

	return errResponseAlreadyWritten
}

func respondBalanceError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, account.ErrNotFound):
		return RespondError(w, http.StatusNotFound, ErrCodeAccountNotFound, ErrMessageAccountNotFound)
	case errors.Is(err, reconciliation.ErrBalanceNotFound):
		return RespondError(w, http.StatusNotFound, ErrCodeBalanceNotFound, ErrMessageBalanceNotFound)
	case errors.Is(err, services.ErrReconciliationMonthInFuture):
		return respondMonthError(w, monthFutureReason)
	case errors.Is(err, reconciliation.ErrBalanceOutOfRange):
		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldBalanceMinor, Message: err.Error(), Code: ErrCodeValidationError})
	default:
		return RespondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}
