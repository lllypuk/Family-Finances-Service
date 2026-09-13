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
	"github.com/labstack/echo/v4"
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

func (h *BudgetHandler) CreateBudget(c echo.Context) error {
	var req CreateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if handled, err := respondClientID(c, req.ID, h.findBudget, h.buildBudgetResponse); handled {
		return err
	}

	createdBudget, err := h.budgetService.CreateBudget(c.Request().Context(), dto.CreateBudgetDTO{
		ID:          req.ID,
		Name:        req.Name,
		AmountMinor: req.AmountMinor,
		Period:      budget.Period(req.Period),
		CategoryID:  req.CategoryID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if err != nil {
		return h.handleBudgetServiceError(c, err, "create")
	}

	return respondAPI(c, http.StatusCreated, h.buildBudgetResponse(createdBudget))
}

// findBudget ищет бюджет по клиентскому id; ошибка означает «не найден».
func (h *BudgetHandler) findBudget(c echo.Context, id uuid.UUID) (*budget.Budget, bool) {
	b, err := h.budgetService.GetBudgetByID(c.Request().Context(), id)

	return b, err == nil
}

func (h *BudgetHandler) GetBudgets(c echo.Context) error {
	page, err := parsePagination(c)
	if err != nil {
		return ignoreWritten(err)
	}

	var budgets []*budget.Budget
	var total int

	if c.QueryParam("active_only") == "true" {
		var active []*budget.Budget
		active, err = h.budgetService.GetActiveBudgets(
			c.Request().Context(),
			familyToday(c.Request().Context(), h.repositories.Family),
		)
		total = len(active)
		budgets = pageSlice(active, page)
	} else {
		filter := dto.NewBudgetFilterDTO()
		filter.Limit = page.Limit
		filter.Offset = page.Offset
		budgets, total, err = h.budgetService.GetBudgetsPage(c.Request().Context(), filter)
	}
	if err != nil {
		return h.handleBudgetServiceError(c, err, "fetch")
	}

	response := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		response = append(response, h.buildBudgetResponse(b))
	}

	return respondList(c, response, page, total)
}

func (h *BudgetHandler) GetBudgetByID(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "budget")
	if err != nil {
		iDParseError := &IDParseError{}
		if errors.As(err, &iDParseError) {
			return HandleIDParseError(c, "budget")
		}
		return err
	}

	foundBudget, err := h.budgetService.GetBudgetByID(c.Request().Context(), id)
	if err != nil {
		return h.handleBudgetServiceError(c, err, "get_by_id")
	}

	return respondAPI(c, http.StatusOK, h.buildBudgetResponse(foundBudget))
}

func (h *BudgetHandler) UpdateBudget(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "budget")
	if err != nil {
		iDParseError := &IDParseError{}
		if errors.As(err, &iDParseError) {
			return HandleIDParseError(c, "budget")
		}
		return err
	}

	var req UpdateBudgetRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondBindError(c, bindErr)
	}
	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}
	if req.Name == nil && req.AmountMinor == nil && req.StartDate == nil && req.EndDate == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	serviceReq := dto.UpdateBudgetDTO{
		Name:        req.Name,
		AmountMinor: req.AmountMinor,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	updatedBudget, err := h.budgetService.UpdateBudget(c.Request().Context(), id, serviceReq)
	if err != nil {
		return h.handleBudgetServiceError(c, err, "update")
	}

	return respondAPI(c, http.StatusOK, h.buildBudgetResponse(updatedBudget))
}

func (h *BudgetHandler) DeleteBudget(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.budgetService.DeleteBudget(c.Request().Context(), id)
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
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

func (h *BudgetHandler) handleBudgetServiceError(c echo.Context, err error, operation string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrBudgetNotFoundService), errors.Is(err, services.ErrBudgetNotFound):
		return HandleNotFoundError(c, "Budget")
	case errors.Is(err, services.ErrBudgetOverlapExists):
		return respondError(c, http.StatusConflict, ErrCodeBudgetOverlap, ErrMessageBudgetOverlap)
	case errors.Is(err, services.ErrBudgetNameExists):
		return respondError(c, http.StatusConflict, ErrCodeBudgetNameExists, ErrMessageBudgetNameExists)
	case errors.Is(err, services.ErrBudgetIDExists):
		return respondError(c, http.StatusConflict, ErrCodeBudgetIDExists, ErrMessageBudgetIDExists)
	case errors.Is(err, services.ErrBudgetAlreadyExceeded):
		return respondError(c, http.StatusConflict, ErrCodeBudgetBelowSpent, ErrMessageBudgetBelowSpent)
	case errors.Is(err, services.ErrBudgetAmountTooLarge),
		errors.Is(err, dto.ErrInvalidBudgetPeriod),
		errors.Is(err, dto.ErrInvalidBudgetAmount),
		errors.Is(err, dto.ErrInvalidDateRange),
		errors.Is(err, dto.ErrInvalidAmountRange),
		strings.Contains(err.Error(), "validation failed"):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		switch operation {
		case "create":
			return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create budget")
		case "update":
			return respondError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update budget")
		default:
			return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch budgets")
		}
	}
}
