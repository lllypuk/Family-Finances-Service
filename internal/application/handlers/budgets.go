package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"family-budget-service/internal/domain/budget"
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
		validator:     validator.New(),
		budgetService: budgetService,
	}
}

func (h *BudgetHandler) CreateBudget(c echo.Context) error {
	var req CreateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
	}

	if err := h.validator.Struct(req); err != nil {
		return HandleValidationError(c, err)
	}

	if h.budgetService != nil {
		return h.createBudgetViaService(c, req)
	}

	// Создаем новый бюджет
	newBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       req.Name,
		Amount:     req.Amount,
		Spent:      0.0, // Начальная потраченная сумма
		Period:     budget.Period(req.Period),
		CategoryID: req.CategoryID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.repositories.Budget.Create(c.Request().Context(), newBudget); err != nil {
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create budget")
	}

	response := BudgetResponse{
		ID:         newBudget.ID,
		Name:       newBudget.Name,
		Amount:     newBudget.Amount,
		Spent:      newBudget.Spent,
		Remaining:  newBudget.Amount - newBudget.Spent,
		Period:     string(newBudget.Period),
		CategoryID: newBudget.CategoryID,
		StartDate:  newBudget.StartDate,
		EndDate:    newBudget.EndDate,
		IsActive:   newBudget.IsActive,
		CreatedAt:  newBudget.CreatedAt,
		UpdatedAt:  newBudget.UpdatedAt,
	}

	return respondAPI(c, http.StatusCreated, response)
}

func (h *BudgetHandler) GetBudgets(c echo.Context) error {
	if h.budgetService != nil {
		return h.getBudgetsViaService(c)
	}

	// Получаем параметры запроса
	activeOnlyParam := c.QueryParam("active_only")

	var budgets []*budget.Budget
	var err error

	// Если запрашиваются только активные бюджеты
	if activeOnlyParam == "true" {
		budgets, err = h.repositories.Budget.GetActiveBudgets(c.Request().Context())
	} else {
		budgets, err = h.repositories.Budget.GetAll(c.Request().Context())
	}

	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch budgets")
	}

	var response []BudgetResponse
	for _, b := range budgets {
		response = append(response, BudgetResponse{
			ID:         b.ID,
			Name:       b.Name,
			Amount:     b.Amount,
			Spent:      b.Spent,
			Remaining:  b.Amount - b.Spent,
			Period:     string(b.Period),
			CategoryID: b.CategoryID,
			StartDate:  b.StartDate,
			EndDate:    b.EndDate,
			IsActive:   b.IsActive,
			CreatedAt:  b.CreatedAt,
			UpdatedAt:  b.UpdatedAt,
		})
	}

	return respondAPI(c, http.StatusOK, response)
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
	var spent float64
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
			spent = foundBudget.Spent
		}
	}

	response := BudgetResponse{
		ID:         foundBudget.ID,
		Name:       foundBudget.Name,
		Amount:     foundBudget.Amount,
		Spent:      spent,
		Remaining:  foundBudget.Amount - spent,
		Period:     string(foundBudget.Period),
		CategoryID: foundBudget.CategoryID,
		StartDate:  foundBudget.StartDate,
		EndDate:    foundBudget.EndDate,
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
		budget.Amount = *req.Amount
	}
	if req.StartDate != nil {
		budget.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		budget.EndDate = *req.EndDate
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
		Amount:     b.Amount,
		Spent:      b.Spent,
		Remaining:  b.Amount - b.Spent,
		Period:     string(b.Period),
		CategoryID: b.CategoryID,
		StartDate:  b.StartDate,
		EndDate:    b.EndDate,
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
		Name:       req.Name,
		Amount:     req.Amount,
		Period:     budget.Period(req.Period),
		CategoryID: req.CategoryID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	})
	if err != nil {
		return h.handleBudgetServiceError(c, err, "create")
	}

	return respondAPI(c, http.StatusCreated, h.buildBudgetResponse(createdBudget))
}

func (h *BudgetHandler) getBudgetsViaService(c echo.Context) error {
	activeOnlyParam := c.QueryParam("active_only")

	var budgets []*budget.Budget
	var err error

	if activeOnlyParam == "true" {
		budgets, err = h.budgetService.GetActiveBudgets(c.Request().Context(), time.Now())
	} else {
		budgets, err = h.getAllBudgetsViaService(c)
	}
	if err != nil {
		return h.handleBudgetServiceError(c, err, "fetch")
	}

	response := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		response = append(response, h.buildBudgetResponse(b))
	}

	return respondAPI(c, http.StatusOK, response)
}

func (h *BudgetHandler) getAllBudgetsViaService(c echo.Context) ([]*budget.Budget, error) {
	filter := dto.NewBudgetFilterDTO()
	filter.Limit = 100
	filter.Offset = 0

	var allBudgets []*budget.Budget
	for {
		page, err := h.budgetService.GetAllBudgets(c.Request().Context(), filter)
		if err != nil {
			return nil, err
		}

		allBudgets = append(allBudgets, page...)
		if len(page) < filter.Limit {
			break
		}
		filter.Offset += filter.Limit
	}

	return allBudgets, nil
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
		return HandleValidationError(c, validationErr)
	}

	updatedBudget, err := h.budgetService.UpdateBudget(c.Request().Context(), id, dto.UpdateBudgetDTO{
		Name:      req.Name,
		Amount:    req.Amount,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		IsActive:  req.IsActive,
	})
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
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid budget data", err.Error())
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
