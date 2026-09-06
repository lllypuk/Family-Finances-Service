package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
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

func NewBudgetHandler(
	repositories *Repositories,
	budgetServices ...services.BudgetService,
) *BudgetHandler {
	var budgetService services.BudgetService
	if len(budgetServices) > 0 {
		budgetService = budgetServices[0]
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
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, err.Error()))
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if h.budgetService != nil {
		return h.createBudgetViaService(c, req)
	}

	// Создаем новый бюджет
	newBudget := &budget.Budget{
		ID:          uuid.New(),
		Name:        req.Name,
		AmountMinor: money.FromFloat(req.Amount),
		SpentMinor:  0, // Начальная потраченная сумма
		Period:      budget.Period(req.Period),
		CategoryID:  req.CategoryID,
		StartDate:   date.FromTime(req.StartDate),
		EndDate:     date.FromTime(req.EndDate),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repositories.Budget.Create(c.Request().Context(), newBudget); err != nil {
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create budget")
	}

	response := BudgetResponse{
		ID:         newBudget.ID,
		Name:       newBudget.Name,
		Amount:     newBudget.AmountMinor.Float(),
		Spent:      newBudget.SpentMinor.Float(),
		Remaining:  newBudget.GetRemainingAmount().Float(),
		Period:     string(newBudget.Period),
		CategoryID: newBudget.CategoryID,
		StartDate:  newBudget.StartDate.In(time.UTC),
		EndDate:    newBudget.EndDate.In(time.UTC),
		IsActive:   newBudget.IsActive,
		CreatedAt:  newBudget.CreatedAt,
		UpdatedAt:  newBudget.UpdatedAt,
	}

	return respondAPI(c, http.StatusCreated, response)
}

func (h *BudgetHandler) GetBudgets(c echo.Context) error {
	page, err := parsePagination(c)
	if err != nil {
		return ignoreWritten(err)
	}

	if h.budgetService != nil {
		return h.getBudgetsViaService(c, page)
	}

	var budgets []*budget.Budget
	if c.QueryParam("active_only") == "true" {
		budgets, err = h.repositories.Budget.GetActiveBudgets(c.Request().Context())
	} else {
		budgets, err = h.repositories.Budget.GetAll(c.Request().Context())
	}

	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch budgets")
	}

	total := len(budgets)
	response := make([]BudgetResponse, 0, page.Limit)
	for _, b := range pageSlice(budgets, page) {
		response = append(response, h.buildBudgetResponse(b))
	}

	return respondList(c, response, page, total)
}

func (h *BudgetHandler) GetBudgetByID(c echo.Context) error {
	if h.budgetService != nil {
		return h.getBudgetByIDViaService(c)
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return HandleIDParseError(c, "budget")
	}

	foundBudget, err := h.repositories.Budget.GetByID(c.Request().Context(), id)
	if err != nil {
		return HandleNotFoundError(c, "Budget")
	}

	// Вычисляем сумму расходов по бюджету (по категории и семье)
	var spent money.Minor
	if foundBudget.CategoryID != nil {
		// Получаем сумму расходов по категории бюджета в пределах периода бюджета
		spent, err = h.repositories.Transaction.GetTotalByCategoryAndDateRange(
			c.Request().Context(),
			*foundBudget.CategoryID,
			foundBudget.StartDate,
			foundBudget.EndDate,
			transaction.TypeExpense,
		)
		if err != nil {
			spent = 0
		}
	} else {
		// Если категория не указана, считаем все расходы семьи в пределах периода бюджета
		spent, err = h.repositories.Transaction.GetTotalByDateRange(
			c.Request().Context(),
			foundBudget.StartDate,
			foundBudget.EndDate,
			transaction.TypeExpense,
		)
		if err != nil {
			spent = foundBudget.SpentMinor
		}
	}

	response := BudgetResponse{
		ID:         foundBudget.ID,
		Name:       foundBudget.Name,
		Amount:     foundBudget.AmountMinor.Float(),
		Spent:      spent.Float(),
		Remaining:  (foundBudget.AmountMinor - spent).Float(),
		Period:     string(foundBudget.Period),
		CategoryID: foundBudget.CategoryID,
		StartDate:  foundBudget.StartDate.In(time.UTC),
		EndDate:    foundBudget.EndDate.In(time.UTC),
		IsActive:   foundBudget.IsActive,
		CreatedAt:  foundBudget.CreatedAt,
		UpdatedAt:  foundBudget.UpdatedAt,
	}

	return respondAPI(c, http.StatusOK, response)
}

func (h *BudgetHandler) UpdateBudget(c echo.Context) error {
	if h.budgetService != nil {
		return h.updateBudgetViaService(c)
	}

	helper := NewUpdateEntityHelper(
		UpdateEntityParams[UpdateBudgetRequest, *budget.Budget, BudgetResponse]{
			Validator: h.validator,
			GetByID: func(c echo.Context, id uuid.UUID) (*budget.Budget, error) {
				return h.repositories.Budget.GetByID(c.Request().Context(), id)
			},
			Update: func(c echo.Context, entity *budget.Budget) error {
				return h.repositories.Budget.Update(c.Request().Context(), entity)
			},
			UpdateFields:  h.updateBudgetFields,
			BuildResponse: h.buildBudgetResponse,
			EntityType:    "budget",
		})

	return helper.Execute(c)
}

func (h *BudgetHandler) updateBudgetFields(budget *budget.Budget, req *UpdateBudgetRequest) {
	if req.Name != nil {
		budget.Name = *req.Name
	}
	if req.Amount != nil {
		budget.AmountMinor = money.FromFloat(*req.Amount)
	}
	if req.StartDate != nil {
		budget.StartDate = date.FromTime(*req.StartDate)
	}
	if req.EndDate != nil {
		budget.EndDate = date.FromTime(*req.EndDate)
	}
	if req.IsActive != nil {
		budget.IsActive = *req.IsActive
	}
	budget.UpdatedAt = time.Now()
}

func (h *BudgetHandler) buildBudgetResponse(b *budget.Budget) BudgetResponse {
	return BudgetResponse{
		ID:         b.ID,
		Name:       b.Name,
		Amount:     b.AmountMinor.Float(),
		Spent:      b.SpentMinor.Float(),
		Remaining:  b.GetRemainingAmount().Float(),
		Period:     string(b.Period),
		CategoryID: b.CategoryID,
		StartDate:  b.StartDate.In(time.UTC),
		EndDate:    b.EndDate.In(time.UTC),
		IsActive:   b.IsActive,
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
	}
}

func (h *BudgetHandler) DeleteBudget(c echo.Context) error {
	if h.budgetService != nil {
		return DeleteEntityHelper(c, func(id uuid.UUID) error {
			return h.budgetService.DeleteBudget(c.Request().Context(), id)
		}, "Budget")
	}

	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		// In single-family model, repository will handle family ID internally
		return h.repositories.Budget.Delete(c.Request().Context(), id)
	}, "Budget")
}

func (h *BudgetHandler) createBudgetViaService(c echo.Context, req CreateBudgetRequest) error {
	createdBudget, err := h.budgetService.CreateBudget(c.Request().Context(), dto.CreateBudgetDTO{
		Name:        req.Name,
		AmountMinor: money.FromFloat(req.Amount),
		Period:      budget.Period(req.Period),
		CategoryID:  req.CategoryID,
		StartDate:   date.FromTime(req.StartDate),
		EndDate:     date.FromTime(req.EndDate),
	})
	if err != nil {
		return h.handleBudgetServiceError(c, err, "create")
	}

	return respondAPI(c, http.StatusCreated, h.buildBudgetResponse(createdBudget))
}

func (h *BudgetHandler) getBudgetsViaService(c echo.Context, page pageParams) error {
	var budgets []*budget.Budget
	var total int
	var err error

	if c.QueryParam("active_only") == "true" {
		var active []*budget.Budget
		active, err = h.budgetService.GetActiveBudgets(c.Request().Context(), date.Today(time.UTC))
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

func (h *BudgetHandler) getBudgetByIDViaService(c echo.Context) error {
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

func (h *BudgetHandler) updateBudgetViaService(c echo.Context) error {
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
		return HandleBindError(c)
	}
	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	serviceReq := dto.UpdateBudgetDTO{Name: req.Name, IsActive: req.IsActive}
	if req.Amount != nil {
		amount := money.FromFloat(*req.Amount)
		serviceReq.AmountMinor = &amount
	}
	if req.StartDate != nil {
		start := date.FromTime(*req.StartDate)
		serviceReq.StartDate = &start
	}
	if req.EndDate != nil {
		end := date.FromTime(*req.EndDate)
		serviceReq.EndDate = &end
	}

	updatedBudget, err := h.budgetService.UpdateBudget(c.Request().Context(), id, serviceReq)
	if err != nil {
		return h.handleBudgetServiceError(c, err, "update")
	}

	return respondAPI(c, http.StatusOK, h.buildBudgetResponse(updatedBudget))
}

func (h *BudgetHandler) handleBudgetServiceError(c echo.Context, err error, operation string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrBudgetNotFoundService), errors.Is(err, services.ErrBudgetNotFound):
		return HandleNotFoundError(c, "Budget")
	case operation == "create" &&
		(errors.Is(err, dto.ErrInvalidBudgetPeriod) || errors.Is(err, dto.ErrInvalidDateRange)):
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create budget")
	case errors.Is(err, services.ErrBudgetOverlapExists),
		errors.Is(err, services.ErrBudgetAlreadyExceeded),
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
