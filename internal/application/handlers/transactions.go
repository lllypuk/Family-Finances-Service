package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/transaction"
)

type TransactionHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewTransactionHandler(repositories *Repositories) *TransactionHandler {
	return &TransactionHandler{
		repositories: repositories,
		validator:    validator.New(),
	}
}

func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	var req CreateTransactionRequest
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
		return h.handleValidationError(c, err)
	}

	newTransaction := h.buildTransaction(req)

	if err := h.repositories.Transaction.Create(c.Request().Context(), newTransaction); err != nil {
		// Check if it's a PostgreSQL foreign key constraint error
		if h.isForeignKeyConstraintError(err) {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: "Invalid category, user, or family ID",
				},
				Meta: ResponseMeta{
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
					Timestamp: time.Now(),
					Version:   "v1",
				},
			})
		}

		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create transaction",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	h.updateBudgetIfNeeded(c, newTransaction)

	response := h.buildTransactionResponse(newTransaction)
	return c.JSON(http.StatusCreated, APIResponse[TransactionResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *TransactionHandler) handleValidationError(c echo.Context, err error) error {
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

	return c.JSON(http.StatusBadRequest, APIResponse[any]{
		Data: nil,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
		Errors: validationErrors,
	})
}

func (h *TransactionHandler) buildTransaction(req CreateTransactionRequest) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      req.Amount,
		Type:        transaction.Type(req.Type),
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		FamilyID:    req.FamilyID,
		Date:        req.Date,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (h *TransactionHandler) updateBudgetIfNeeded(c echo.Context, tx *transaction.Transaction) {
	if tx.Type != transaction.TypeExpense {
		return
	}

	// Check if Budget repository is available (might be nil in tests)
	if h.repositories.Budget == nil {
		return
	}

	budgets, err := h.repositories.Budget.GetActiveBudgets(c.Request().Context(), tx.FamilyID)
	if err != nil {
		return
	}

	for _, b := range budgets {
		if b.CategoryID != nil && *b.CategoryID == tx.CategoryID {
			b.Spent += tx.Amount
			b.UpdatedAt = time.Now()
			_ = h.repositories.Budget.Update(c.Request().Context(), b)
			break
		}
	}
}

func (h *TransactionHandler) buildTransactionResponse(tx *transaction.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Type:        string(tx.Type),
		Description: tx.Description,
		CategoryID:  tx.CategoryID,
		UserID:      tx.UserID,
		FamilyID:    tx.FamilyID,
		Date:        tx.Date,
		Tags:        tx.Tags,
		CreatedAt:   tx.CreatedAt,
		UpdatedAt:   tx.UpdatedAt,
	}
}

func (h *TransactionHandler) GetTransactions(c echo.Context) error {
	filters, err := h.parseTransactionFilters(c)
	if err != nil {
		return err
	}

	err = h.validateTransactionFilters(c, filters)
	if err != nil {
		return err
	}

	repoFilter := h.buildRepositoryFilter(filters)

	transactions, err := h.repositories.Transaction.GetByFilter(c.Request().Context(), repoFilter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch transactions",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := h.buildTransactionListResponse(transactions)

	return c.JSON(http.StatusOK, APIResponse[[]TransactionResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *TransactionHandler) parseTransactionFilters(c echo.Context) (TransactionFilterParams, error) {
	var filters TransactionFilterParams

	// Обязательный параметр family_id
	familyIDParam := c.QueryParam("family_id")
	if familyIDParam == "" {
		_ = c.JSON(http.StatusBadRequest, ErrorResponse{
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
		return filters, errors.New("missing family_id")
	}

	familyID, err := uuid.Parse(familyIDParam)
	if err != nil {
		_ = c.JSON(http.StatusBadRequest, ErrorResponse{
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
		return filters, errors.New("invalid family_id")
	}
	filters.FamilyID = familyID

	h.parseOptionalFilters(c, &filters)
	h.parsePaginationParams(c, &filters)

	return filters, nil
}

func (h *TransactionHandler) parseOptionalFilters(c echo.Context, filters *TransactionFilterParams) {
	if userIDParam := c.QueryParam("user_id"); userIDParam != "" {
		if userID, parseErr := uuid.Parse(userIDParam); parseErr == nil {
			filters.UserID = &userID
		}
	}

	if categoryIDParam := c.QueryParam("category_id"); categoryIDParam != "" {
		if categoryID, parseErr := uuid.Parse(categoryIDParam); parseErr == nil {
			filters.CategoryID = &categoryID
		}
	}

	if typeParam := c.QueryParam("type"); typeParam != "" {
		filters.Type = &typeParam
	}

	if dateFromParam := c.QueryParam("date_from"); dateFromParam != "" {
		if dateFrom, parseErr := time.Parse(time.RFC3339, dateFromParam); parseErr == nil {
			filters.DateFrom = &dateFrom
		}
	}

	if dateToParam := c.QueryParam("date_to"); dateToParam != "" {
		if dateTo, parseErr := time.Parse(time.RFC3339, dateToParam); parseErr == nil {
			filters.DateTo = &dateTo
		}
	}

	if amountFromParam := c.QueryParam("amount_from"); amountFromParam != "" {
		if amountFrom, parseErr := strconv.ParseFloat(amountFromParam, 64); parseErr == nil {
			filters.AmountFrom = &amountFrom
		}
	}

	if amountToParam := c.QueryParam("amount_to"); amountToParam != "" {
		if amountTo, parseErr := strconv.ParseFloat(amountToParam, 64); parseErr == nil {
			filters.AmountTo = &amountTo
		}
	}

	if descriptionParam := c.QueryParam("description"); descriptionParam != "" {
		filters.Description = &descriptionParam
	}
}

func (h *TransactionHandler) parsePaginationParams(c echo.Context, filters *TransactionFilterParams) {
	filters.Limit = 50 // По умолчанию
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, parseErr := strconv.Atoi(limitParam); parseErr == nil && limit > 0 && limit <= 1000 {
			filters.Limit = limit
		}
	}

	filters.Offset = 0 // По умолчанию
	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, parseErr := strconv.Atoi(offsetParam); parseErr == nil && offset >= 0 {
			filters.Offset = offset
		}
	}
}

func (h *TransactionHandler) validateTransactionFilters(c echo.Context, filters TransactionFilterParams) error {
	err := h.validator.Struct(filters)
	if err != nil {
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

		return c.JSON(http.StatusBadRequest, APIResponse[any]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}
	return nil
}

func (h *TransactionHandler) buildRepositoryFilter(filters TransactionFilterParams) transaction.Filter {
	var typeFilter *transaction.Type
	if filters.Type != nil {
		t := transaction.Type(*filters.Type)
		typeFilter = &t
	}

	return transaction.Filter{
		FamilyID:   filters.FamilyID,
		UserID:     filters.UserID,
		CategoryID: filters.CategoryID,
		Type:       typeFilter,
		DateFrom:   filters.DateFrom,
		DateTo:     filters.DateTo,
		AmountFrom: filters.AmountFrom,
		AmountTo:   filters.AmountTo,
		Description: func() string {
			if filters.Description != nil {
				return *filters.Description
			}
			return ""
		}(),
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}
}

func (h *TransactionHandler) buildTransactionListResponse(
	transactions []*transaction.Transaction,
) []TransactionResponse {
	var response []TransactionResponse
	for _, tx := range transactions {
		response = append(response, TransactionResponse{
			ID:          tx.ID,
			Amount:      tx.Amount,
			Type:        string(tx.Type),
			Description: tx.Description,
			CategoryID:  tx.CategoryID,
			UserID:      tx.UserID,
			FamilyID:    tx.FamilyID,
			Date:        tx.Date,
			Tags:        tx.Tags,
			CreatedAt:   tx.CreatedAt,
			UpdatedAt:   tx.UpdatedAt,
		})
	}
	return response
}

func (h *TransactionHandler) GetTransactionByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid transaction ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	foundTransaction, err := h.repositories.Transaction.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "TRANSACTION_NOT_FOUND",
				Message: "Transaction not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := TransactionResponse{
		ID:          foundTransaction.ID,
		Amount:      foundTransaction.Amount,
		Type:        string(foundTransaction.Type),
		Description: foundTransaction.Description,
		CategoryID:  foundTransaction.CategoryID,
		UserID:      foundTransaction.UserID,
		FamilyID:    foundTransaction.FamilyID,
		Date:        foundTransaction.Date,
		Tags:        foundTransaction.Tags,
		CreatedAt:   foundTransaction.CreatedAt,
		UpdatedAt:   foundTransaction.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[TransactionResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *TransactionHandler) UpdateTransaction(c echo.Context) error {
	helper := NewUpdateEntityHelper(
		UpdateEntityParams[UpdateTransactionRequest, *transaction.Transaction, TransactionResponse]{
			Validator: h.validator,
			GetByID: func(c echo.Context, id uuid.UUID) (*transaction.Transaction, error) {
				return h.repositories.Transaction.GetByID(c.Request().Context(), id)
			},
			Update: func(c echo.Context, entity *transaction.Transaction) error {
				return h.repositories.Transaction.Update(c.Request().Context(), entity)
			},
			UpdateFields:  h.updateTransactionFields,
			BuildResponse: h.buildTransactionResponse,
			EntityType:    "transaction",
		})

	return helper.Execute(c)
}

func (h *TransactionHandler) updateTransactionFields(tx *transaction.Transaction, req *UpdateTransactionRequest) {
	if req.Amount != nil {
		tx.Amount = *req.Amount
	}
	if req.Type != nil {
		tx.Type = transaction.Type(*req.Type)
	}
	if req.Description != nil {
		tx.Description = *req.Description
	}
	if req.CategoryID != nil {
		tx.CategoryID = *req.CategoryID
	}
	if req.Date != nil {
		tx.Date = *req.Date
	}
	if req.Tags != nil {
		tx.Tags = req.Tags
	}
	tx.UpdatedAt = time.Now()
}

// isForeignKeyConstraintError checks if the error is a PostgreSQL foreign key constraint violation
func (h *TransactionHandler) isForeignKeyConstraintError(err error) bool {
	errStr := err.Error()
	// PostgreSQL foreign key constraint error messages contain these patterns
	return strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "violates foreign key constraint") ||
		strings.Contains(errStr, "SQLSTATE 23503")
}

func (h *TransactionHandler) DeleteTransaction(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		// Get family ID from query parameter
		familyIDParam := c.QueryParam("family_id")
		if familyIDParam == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "family_id query parameter is required")
		}

		familyID, err := uuid.Parse(familyIDParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid family ID format")
		}

		return h.repositories.Transaction.Delete(c.Request().Context(), id, familyID)
	}, "Transaction")
}
