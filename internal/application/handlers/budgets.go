package handlers

import (
	"errors"
	"net/http"
	"strings"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type BudgetHandler struct {
	repositories  *Repositories
	validator     *validator.Validate
	budgetService services.BudgetService
}

// NewBudgetHandler — сервис обязателен: бизнес-правила бюджета живут только в нём.
func NewBudgetHandler(repositories *Repositories, budgetService services.BudgetService) *BudgetHandler {
	if budgetService == nil {
		panic("handlers: NewBudgetHandler requires a non-nil services.BudgetService")
	}

	return &BudgetHandler{
		repositories:  repositories,
		validator:     newAPIValidator(),
		budgetService: budgetService,
	}
}

func (h *BudgetHandler) CreateBudget(w http.ResponseWriter, r *http.Request) error {
	var req CreateBudgetRequest
	if err := decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(w, err)
	}

	if handled, err := respondClientID(w, r, req.ID, h.findBudget, h.buildBudgetResponse); handled {
		return err
	}

	createdBudget, err := h.budgetService.CreateBudget(r.Context(), dto.CreateBudgetDTO{
		ID:          req.ID,
		Name:        req.Name,
		AmountMinor: req.AmountMinor,
		Period:      budget.Period(req.Period),
		CategoryID:  req.CategoryID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Recurring:   req.Recurring,
	})
	if err != nil {
		return h.handleBudgetServiceError(w, err, "create")
	}

	return respondAPI(w, http.StatusCreated, h.buildBudgetResponse(createdBudget))
}

// findBudget ищет бюджет по клиентскому id; ошибка означает «не найден».
func (h *BudgetHandler) findBudget(r *http.Request, id uuid.UUID) (*budget.Budget, bool) {
	b, err := h.budgetService.GetBudgetByID(r.Context(), id)

	return b, err == nil
}

func (h *BudgetHandler) GetBudgets(w http.ResponseWriter, r *http.Request) error {
	page, err := parsePagination(w, r)
	if err != nil {
		return ignoreWritten(err)
	}

	var budgets []*budget.Budget
	var total int

	if r.URL.Query().Get("active_only") == "true" {
		var active []*budget.Budget
		active, err = h.budgetService.GetActiveBudgets(
			r.Context(),
			familyToday(r.Context(), h.repositories.Family),
		)
		total = len(active)
		budgets = pageSlice(active, page)
	} else {
		filter := dto.NewBudgetFilterDTO()
		filter.Limit = page.Limit
		filter.Offset = page.Offset
		today := familyToday(r.Context(), h.repositories.Family)
		filter.Today = &today
		budgets, total, err = h.budgetService.GetBudgetsPage(r.Context(), filter)
	}
	if err != nil {
		return h.handleBudgetServiceError(w, err, "fetch")
	}

	response := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		response = append(response, h.buildBudgetResponse(b))
	}

	return respondList(w, response, page, total)
}

func (h *BudgetHandler) GetBudgetByID(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return handleIDParseError(w, "budget")
	}

	foundBudget, err := h.budgetService.GetBudgetByID(r.Context(), id)
	if err != nil {
		return h.handleBudgetServiceError(w, err, "get_by_id")
	}

	return respondAPI(w, http.StatusOK, h.buildBudgetResponse(foundBudget))
}

func (h *BudgetHandler) UpdateBudget(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return handleIDParseError(w, "budget")
	}

	var req UpdateBudgetRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}
	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}
	if req.Name == nil && req.AmountMinor == nil && req.StartDate == nil && req.EndDate == nil &&
		req.Recurring == nil {
		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	serviceReq := dto.UpdateBudgetDTO{
		Name:        req.Name,
		AmountMinor: req.AmountMinor,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Recurring:   req.Recurring,
	}

	updatedBudget, err := h.budgetService.UpdateBudget(r.Context(), id, serviceReq)
	if err != nil {
		return h.handleBudgetServiceError(w, err, "update")
	}

	return respondAPI(w, http.StatusOK, h.buildBudgetResponse(updatedBudget))
}

func (h *BudgetHandler) DeleteBudget(w http.ResponseWriter, r *http.Request) error {
	return deleteEntity(w, r, func(id uuid.UUID) error {
		return h.budgetService.DeleteBudget(r.Context(), id)
	}, "Budget")
}

func (h *BudgetHandler) buildBudgetResponse(b *budget.Budget) BudgetResponse {
	return BudgetResponse{
		ID:             b.ID,
		Name:           b.Name,
		AmountMinor:    b.AmountMinor,
		SpentMinor:     b.SpentMinor,
		RemainingMinor: b.GetRemainingAmount(),
		Utilization:    b.GetSpentPercentage(),
		Period:         string(b.Period),
		CategoryID:     b.CategoryID,
		StartDate:      b.StartDate,
		EndDate:        b.EndDate,
		IsActive:       b.IsActive,
		Recurring:      b.Recurring,
		SeriesID:       b.SeriesID,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// respondRecurringError — отказы вокруг серии приходят из домена и указывают на поле формы.
func respondRecurringError(w http.ResponseWriter, field string, err error) error {
	return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: field, Message: err.Error(), Code: ErrCodeValidationError})
}

func (h *BudgetHandler) handleBudgetServiceError(w http.ResponseWriter, err error, operation string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrBudgetNotFoundService), errors.Is(err, services.ErrBudgetNotFound):
		return handleNotFoundError(w, "Budget")
	case errors.Is(err, services.ErrBudgetOverlapExists):
		return RespondError(w, http.StatusConflict, ErrCodeBudgetOverlap, ErrMessageBudgetOverlap)
	case errors.Is(err, services.ErrBudgetNameExists):
		return RespondError(w, http.StatusConflict, ErrCodeBudgetNameExists, ErrMessageBudgetNameExists)
	case errors.Is(err, services.ErrBudgetIDExists):
		return RespondError(w, http.StatusConflict, ErrCodeBudgetIDExists, ErrMessageBudgetIDExists)
	case errors.Is(err, services.ErrBudgetNotTail):
		return RespondError(w, http.StatusConflict, ErrCodeBudgetNotTail, ErrMessageBudgetNotTail)
	case errors.Is(err, budget.ErrRecurringCustom):
		return respondRecurringError(w, fieldRecurring, err)
	case errors.Is(err, budget.ErrRecurringNotAligned), errors.Is(err, budget.ErrSeriesDatesFixed):
		return respondRecurringError(w, fieldStartDate, err)
	case errors.Is(err, services.ErrBudgetAmountTooLarge),
		errors.Is(err, dto.ErrInvalidBudgetPeriod),
		errors.Is(err, dto.ErrInvalidBudgetAmount),
		errors.Is(err, dto.ErrInvalidDateRange),
		errors.Is(err, dto.ErrInvalidAmountRange),
		strings.Contains(err.Error(), "validation failed"):
		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		switch operation {
		case "create":
			return RespondError(w, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create budget")
		case "update":
			return RespondError(w, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update budget")
		default:
			return RespondError(w, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch budgets")
		}
	}
}
