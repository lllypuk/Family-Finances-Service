package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/services"
)

type HoldingHandler struct {
	holdings  services.HoldingService
	validator *validator.Validate
}

func NewHoldingHandler(holdings services.HoldingService) *HoldingHandler {
	return &HoldingHandler{holdings: holdings, validator: newAPIValidator()}
}

// ListHoldings — активные позиции; ?archived=true добавляет архивные.
func (h *HoldingHandler) ListHoldings(w http.ResponseWriter, r *http.Request) error {
	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}
	includeArchived, archivedErr := parseArchivedParam(w, r)
	if archivedErr != nil {
		return ignoreWritten(archivedErr)
	}

	all, err := h.holdings.List(r.Context(), includeArchived)
	if err != nil {
		return respondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch holdings")
	}

	items := make([]HoldingResponse, 0, page.Limit)
	for _, item := range pageSlice(all, page) {
		items = append(items, toHoldingResponse(item))
	}

	return respondList(w, items, page, len(all))
}

func (h *HoldingHandler) CreateHolding(w http.ResponseWriter, r *http.Request) error {
	var req CreateHoldingRequest
	if err := decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}

	if handled, err := respondClientID(w, r, req.ID, h.findHolding, toHoldingResponse); handled {
		return err
	}

	created, err := h.holdings.Create(r.Context(), req.ID, req.Name,
		holding.Side(req.Side), holding.Kind(req.Kind),
		holding.Plan{MonthlyIncomeMinor: req.MonthlyIncomeMinor, MonthlyExpenseMinor: req.MonthlyExpenseMinor})
	if err != nil {
		return respondHoldingError(w, err)
	}

	return respondAPI(w, http.StatusCreated, toHoldingResponse(created))
}

func (h *HoldingHandler) findHolding(r *http.Request, id uuid.UUID) (*holding.Holding, bool) {
	found, err := h.holdings.GetByID(r.Context(), id)

	return found, err == nil
}

func (h *HoldingHandler) UpdateHolding(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}

	var req UpdateHoldingRequest
	if err = decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}
	if req.Name == nil && req.Kind == nil && req.IsArchived == nil &&
		req.MonthlyIncomeMinor == nil && req.MonthlyExpenseMinor == nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	var kind *holding.Kind
	if req.Kind != nil {
		k := holding.Kind(*req.Kind)
		kind = &k
	}

	updated, err := h.holdings.Update(r.Context(), id, req.Name, kind, req.IsArchived,
		req.MonthlyIncomeMinor, req.MonthlyExpenseMinor)
	if err != nil {
		return respondHoldingError(w, err)
	}

	return respondAPI(w, http.StatusOK, toHoldingResponse(updated))
}

func (h *HoldingHandler) DeleteHolding(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}

	if err = h.holdings.Delete(r.Context(), id); err != nil {
		return respondHoldingError(w, err)
	}

	return noContent(w)
}

// ListHoldingValues — история снимков позиции, новые сверху.
func (h *HoldingHandler) ListHoldingValues(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}
	page, err := parsePagination(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	values, total, err := h.holdings.ListValues(r.Context(), id, page.Limit, page.Offset)
	if err != nil {
		return respondHoldingError(w, err)
	}

	items := make([]HoldingValueResponse, 0, len(values))
	for _, v := range values {
		items = append(items, toHoldingValueResponse(v))
	}

	return respondList(w, items, page, total)
}

func (h *HoldingHandler) PutHoldingValue(w http.ResponseWriter, r *http.Request) error {
	id, day, err := parseHoldingValuePath(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	var req HoldingValueRequest
	if err = decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}

	v, err := h.holdings.PutValue(r.Context(), id, day, *req.ValueMinor)
	if err != nil {
		return respondHoldingError(w, err)
	}

	return respondAPI(w, http.StatusOK, toHoldingValueResponse(v))
}

func (h *HoldingHandler) DeleteHoldingValue(w http.ResponseWriter, r *http.Request) error {
	id, day, err := parseHoldingValuePath(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	if err = h.holdings.DeleteValue(r.Context(), id, day); err != nil {
		return respondHoldingError(w, err)
	}

	return noContent(w)
}

// parseHoldingValuePath пишет 400 сам и возвращает errResponseAlreadyWritten.
func parseHoldingValuePath(w http.ResponseWriter, r *http.Request) (uuid.UUID, date.Date, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID))
	}

	day, err := date.Parse(r.PathValue(fieldDate))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(w, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidValueDate))
	}

	return id, day, nil
}

func respondHoldingError(w http.ResponseWriter, err error) error {
	var (
		field   string
		message = err.Error()
		planErr *holding.PlanError
	)
	switch {
	case errors.Is(err, holding.ErrNotFound):
		return respondError(w, http.StatusNotFound, ErrCodeHoldingNotFound, ErrMessageHoldingNotFound)
	case errors.Is(err, holding.ErrValueNotFound):
		return respondError(w, http.StatusNotFound, ErrCodeHoldingValueNotFound, ErrMessageHoldingValueNotFound)
	case errors.Is(err, holding.ErrNameExists):
		return respondError(w, http.StatusConflict, ErrCodeHoldingNameExists, ErrMessageHoldingNameExists)
	case errors.Is(err, holding.ErrNameEmpty), errors.Is(err, holding.ErrNameLong):
		field = fieldName
	case errors.Is(err, holding.ErrInvalidSide):
		field = fieldSide
	case errors.Is(err, holding.ErrInvalidKind):
		field = fieldKind
	case errors.Is(err, holding.ErrValueOutOfRange):
		field = fieldValueMinor
	case errors.Is(err, holding.ErrValueDateFuture):
		field = fieldDate
	case errors.As(err, &planErr):
		// На PUT ошибка приходит обёрнутой сервисом, а текст уходит клиенту в поле формы.
		field, message = planErr.Field, planErr.Error()
	default:
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: field, Message: message, Code: ErrCodeValidationError})
}

func toHoldingResponse(h *holding.Holding) HoldingResponse {
	resp := HoldingResponse{
		ID:         h.ID,
		Name:       h.Name,
		Side:       string(h.Side),
		Kind:       string(h.Kind),
		IsArchived: h.IsArchived,
		CreatedAt:  h.CreatedAt,
		UpdatedAt:  h.UpdatedAt,

		MonthlyIncomeMinor:  h.Plan.MonthlyIncomeMinor,
		MonthlyExpenseMinor: h.Plan.MonthlyExpenseMinor,
		PlanUpdatedAt:       h.Plan.UpdatedAt,
	}
	if h.Current != nil {
		resp.Current = &HoldingCurrentResponse{Date: h.Current.Date, ValueMinor: h.Current.ValueMinor}
	}

	return resp
}

func toHoldingValueResponse(v *holding.Value) HoldingValueResponse {
	return HoldingValueResponse{Date: v.Date, ValueMinor: v.ValueMinor, UpdatedAt: v.UpdatedAt}
}
