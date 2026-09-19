package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

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
func (h *HoldingHandler) ListHoldings(c echo.Context) error {
	page, pageErr := parsePagination(c)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}
	includeArchived, archivedErr := parseArchivedParam(c)
	if archivedErr != nil {
		return ignoreWritten(archivedErr)
	}

	all, err := h.holdings.List(c.Request().Context(), includeArchived)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch holdings")
	}

	items := make([]HoldingResponse, 0, page.Limit)
	for _, item := range pageSlice(all, page) {
		items = append(items, toHoldingResponse(item))
	}

	return respondList(c, items, page, len(all))
}

func (h *HoldingHandler) CreateHolding(c echo.Context) error {
	var req CreateHoldingRequest
	if err := c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if handled, err := respondClientID(c, req.ID, h.findHolding, toHoldingResponse); handled {
		return err
	}

	created, err := h.holdings.Create(c.Request().Context(), req.ID, req.Name,
		holding.Side(req.Side), holding.Kind(req.Kind),
		holding.Plan{MonthlyIncomeMinor: req.MonthlyIncomeMinor, MonthlyExpenseMinor: req.MonthlyExpenseMinor})
	if err != nil {
		return respondHoldingError(c, err)
	}

	return respondAPI(c, http.StatusCreated, toHoldingResponse(created))
}

func (h *HoldingHandler) findHolding(c echo.Context, id uuid.UUID) (*holding.Holding, bool) {
	found, err := h.holdings.GetByID(c.Request().Context(), id)

	return found, err == nil
}

func (h *HoldingHandler) UpdateHolding(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}

	var req UpdateHoldingRequest
	if err = c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}
	if req.Name == nil && req.Kind == nil && req.IsArchived == nil &&
		req.MonthlyIncomeMinor == nil && req.MonthlyExpenseMinor == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	var kind *holding.Kind
	if req.Kind != nil {
		k := holding.Kind(*req.Kind)
		kind = &k
	}

	updated, err := h.holdings.Update(c.Request().Context(), id, req.Name, kind, req.IsArchived,
		req.MonthlyIncomeMinor, req.MonthlyExpenseMinor)
	if err != nil {
		return respondHoldingError(c, err)
	}

	return respondAPI(c, http.StatusOK, toHoldingResponse(updated))
}

func (h *HoldingHandler) DeleteHolding(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}

	if err = h.holdings.Delete(c.Request().Context(), id); err != nil {
		return respondHoldingError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ListHoldingValues — история снимков позиции, новые сверху.
func (h *HoldingHandler) ListHoldingValues(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID)
	}
	page, err := parsePagination(c)
	if err != nil {
		return ignoreWritten(err)
	}

	values, total, err := h.holdings.ListValues(c.Request().Context(), id, page.Limit, page.Offset)
	if err != nil {
		return respondHoldingError(c, err)
	}

	items := make([]HoldingValueResponse, 0, len(values))
	for _, v := range values {
		items = append(items, toHoldingValueResponse(v))
	}

	return respondList(c, items, page, total)
}

func (h *HoldingHandler) PutHoldingValue(c echo.Context) error {
	id, day, err := parseHoldingValuePath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	var req HoldingValueRequest
	if err = c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}
	if err = h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	v, err := h.holdings.PutValue(c.Request().Context(), id, day, *req.ValueMinor)
	if err != nil {
		return respondHoldingError(c, err)
	}

	return respondAPI(c, http.StatusOK, toHoldingValueResponse(v))
}

func (h *HoldingHandler) DeleteHoldingValue(c echo.Context) error {
	id, day, err := parseHoldingValuePath(c)
	if err != nil {
		return ignoreWritten(err)
	}

	if err = h.holdings.DeleteValue(c.Request().Context(), id, day); err != nil {
		return respondHoldingError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// parseHoldingValuePath пишет 400 сам и возвращает errResponseAlreadyWritten.
func parseHoldingValuePath(c echo.Context) (uuid.UUID, date.Date, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidHoldingID))
	}

	day, err := date.Parse(c.Param(fieldDate))
	if err != nil {
		return uuid.Nil, date.Date{}, written(
			respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidValueDate))
	}

	return id, day, nil
}

func respondHoldingError(c echo.Context, err error) error {
	var (
		field   string
		message = err.Error()
		planErr *holding.PlanError
	)
	switch {
	case errors.Is(err, holding.ErrNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeHoldingNotFound, ErrMessageHoldingNotFound)
	case errors.Is(err, holding.ErrValueNotFound):
		return respondError(c, http.StatusNotFound, ErrCodeHoldingValueNotFound, ErrMessageHoldingValueNotFound)
	case errors.Is(err, holding.ErrNameExists):
		return respondError(c, http.StatusConflict, ErrCodeHoldingNameExists, ErrMessageHoldingNameExists)
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
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
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
