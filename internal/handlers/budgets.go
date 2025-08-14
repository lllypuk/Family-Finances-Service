package handlers

import (
	"errors"
	"net/http"
	"time"

	"family-budget-service/internal/domain/budget"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type BudgetHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewBudgetHandler(repositories *Repositories) *BudgetHandler {
	return &BudgetHandler{
		repositories: repositories,
		validator:    validator.New(),
	}
}

func (h *BudgetHandler) CreateBudget(c echo.Context) error {
	var req CreateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErrors []ValidationError
		for _, err := range func() validator.ValidationErrors {
			var target validator.ValidationErrors
			_ = errors.As(err, &target)
			return target
		}() {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Message: err.Tag(),
				Code:    "VALIDATION_ERROR",
			})
		}

		return c.JSON(http.StatusBadRequest, APIResponse[interface{}]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Создаем новый бюджет
	newBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       req.Name,
		Amount:     req.Amount,
		Spent:      0.0, // Начальная потраченная сумма
		Period:     budget.BudgetPeriod(req.Period),
		CategoryID: req.CategoryID,
		FamilyID:   req.FamilyID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.repositories.Budget.Create(c.Request().Context(), newBudget); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create budget",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := BudgetResponse{
		ID:         newBudget.ID,
		Name:       newBudget.Name,
		Amount:     newBudget.Amount,
		Spent:      newBudget.Spent,
		Period:     string(newBudget.Period),
		CategoryID: newBudget.CategoryID,
		FamilyID:   newBudget.FamilyID,
		StartDate:  newBudget.StartDate,
		EndDate:    newBudget.EndDate,
		IsActive:   newBudget.IsActive,
		CreatedAt:  newBudget.CreatedAt,
		UpdatedAt:  newBudget.UpdatedAt,
	}

	return c.JSON(http.StatusCreated, APIResponse[BudgetResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *BudgetHandler) GetBudgets(c echo.Context) error {
	// Получаем параметры запроса
	familyIDParam := c.QueryParam("family_id")
	activeOnlyParam := c.QueryParam("active_only")

	if familyIDParam == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "MISSING_FAMILY_ID",
				Message: "family_id query parameter is required",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	familyID, err := uuid.Parse(familyIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_FAMILY_ID",
				Message: "Invalid family ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var budgets []*budget.Budget

	// Если запрашиваются только активные бюджеты
	if activeOnlyParam == "true" {
		budgets, err = h.repositories.Budget.GetActiveBudgets(c.Request().Context(), familyID)
	} else {
		budgets, err = h.repositories.Budget.GetByFamilyID(c.Request().Context(), familyID)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch budgets",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var response []BudgetResponse
	for _, b := range budgets {
		response = append(response, BudgetResponse{
			ID:         b.ID,
			Name:       b.Name,
			Amount:     b.Amount,
			Spent:      b.Spent,
			Period:     string(b.Period),
			CategoryID: b.CategoryID,
			FamilyID:   b.FamilyID,
			StartDate:  b.StartDate,
			EndDate:    b.EndDate,
			IsActive:   b.IsActive,
			CreatedAt:  b.CreatedAt,
			UpdatedAt:  b.UpdatedAt,
		})
	}

	return c.JSON(http.StatusOK, APIResponse[[]BudgetResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *BudgetHandler) GetBudgetByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid budget ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	foundBudget, err := h.repositories.Budget.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "BUDGET_NOT_FOUND",
				Message: "Budget not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := BudgetResponse{
		ID:         foundBudget.ID,
		Name:       foundBudget.Name,
		Amount:     foundBudget.Amount,
		Spent:      foundBudget.Spent,
		Period:     string(foundBudget.Period),
		CategoryID: foundBudget.CategoryID,
		FamilyID:   foundBudget.FamilyID,
		StartDate:  foundBudget.StartDate,
		EndDate:    foundBudget.EndDate,
		IsActive:   foundBudget.IsActive,
		CreatedAt:  foundBudget.CreatedAt,
		UpdatedAt:  foundBudget.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[BudgetResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *BudgetHandler) UpdateBudget(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid budget ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var req UpdateBudgetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErrors []ValidationError
		for _, err := range func() validator.ValidationErrors {
			var target validator.ValidationErrors
			_ = errors.As(err, &target)
			return target
		}() {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Message: err.Tag(),
				Code:    "VALIDATION_ERROR",
			})
		}

		return c.JSON(http.StatusBadRequest, APIResponse[interface{}]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Получаем существующий бюджет
	existingBudget, err := h.repositories.Budget.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "BUDGET_NOT_FOUND",
				Message: "Budget not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Обновляем поля
	if req.Name != nil {
		existingBudget.Name = *req.Name
	}
	if req.Amount != nil {
		existingBudget.Amount = *req.Amount
	}
	if req.StartDate != nil {
		existingBudget.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		existingBudget.EndDate = *req.EndDate
	}
	if req.IsActive != nil {
		existingBudget.IsActive = *req.IsActive
	}
	existingBudget.UpdatedAt = time.Now()

	if err := h.repositories.Budget.Update(c.Request().Context(), existingBudget); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "UPDATE_FAILED",
				Message: "Failed to update budget",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := BudgetResponse{
		ID:         existingBudget.ID,
		Name:       existingBudget.Name,
		Amount:     existingBudget.Amount,
		Spent:      existingBudget.Spent,
		Period:     string(existingBudget.Period),
		CategoryID: existingBudget.CategoryID,
		FamilyID:   existingBudget.FamilyID,
		StartDate:  existingBudget.StartDate,
		EndDate:    existingBudget.EndDate,
		IsActive:   existingBudget.IsActive,
		CreatedAt:  existingBudget.CreatedAt,
		UpdatedAt:  existingBudget.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[BudgetResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *BudgetHandler) DeleteBudget(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.repositories.Budget.Delete(c.Request().Context(), id)
	}, "Budget")
}
