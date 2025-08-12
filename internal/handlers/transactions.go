package handlers

import (
	"errors"
	"net/http"
	"strconv"
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

	// Создаем новую транзакцию
	newTransaction := &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      req.Amount,
		Type:        transaction.TransactionType(req.Type),
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		FamilyID:    req.FamilyID,
		Date:        req.Date,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.repositories.Transaction.Create(c.Request().Context(), newTransaction); err != nil {
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

	response := TransactionResponse{
		ID:          newTransaction.ID,
		Amount:      newTransaction.Amount,
		Type:        string(newTransaction.Type),
		Description: newTransaction.Description,
		CategoryID:  newTransaction.CategoryID,
		UserID:      newTransaction.UserID,
		FamilyID:    newTransaction.FamilyID,
		Date:        newTransaction.Date,
		Tags:        newTransaction.Tags,
		CreatedAt:   newTransaction.CreatedAt,
		UpdatedAt:   newTransaction.UpdatedAt,
	}

	return c.JSON(http.StatusCreated, APIResponse[TransactionResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *TransactionHandler) GetTransactions(c echo.Context) error {
	// Парсим параметры фильтрации
	var filters TransactionFilterParams

	// Обязательный параметр family_id
	familyIDParam := c.QueryParam("family_id")
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
	filters.FamilyID = familyID

	// Опциональные параметры
	if userIDParam := c.QueryParam("user_id"); userIDParam != "" {
		if userID, err := uuid.Parse(userIDParam); err == nil {
			filters.UserID = &userID
		}
	}

	if categoryIDParam := c.QueryParam("category_id"); categoryIDParam != "" {
		if categoryID, err := uuid.Parse(categoryIDParam); err == nil {
			filters.CategoryID = &categoryID
		}
	}

	if typeParam := c.QueryParam("type"); typeParam != "" {
		filters.Type = &typeParam
	}

	if dateFromParam := c.QueryParam("date_from"); dateFromParam != "" {
		if dateFrom, err := time.Parse(time.RFC3339, dateFromParam); err == nil {
			filters.DateFrom = &dateFrom
		}
	}

	if dateToParam := c.QueryParam("date_to"); dateToParam != "" {
		if dateTo, err := time.Parse(time.RFC3339, dateToParam); err == nil {
			filters.DateTo = &dateTo
		}
	}

	if amountFromParam := c.QueryParam("amount_from"); amountFromParam != "" {
		if amountFrom, err := strconv.ParseFloat(amountFromParam, 64); err == nil {
			filters.AmountFrom = &amountFrom
		}
	}

	if amountToParam := c.QueryParam("amount_to"); amountToParam != "" {
		if amountTo, err := strconv.ParseFloat(amountToParam, 64); err == nil {
			filters.AmountTo = &amountTo
		}
	}

	if descriptionParam := c.QueryParam("description"); descriptionParam != "" {
		filters.Description = &descriptionParam
	}

	// Пагинация
	filters.Limit = 50 // По умолчанию
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}

	filters.Offset = 0 // По умолчанию
	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Валидация фильтров
	if err := h.validator.Struct(filters); err != nil {
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

	// Создаем фильтр для репозитория
	var typeFilter *transaction.TransactionType
	if filters.Type != nil {
		t := transaction.TransactionType(*filters.Type)
		typeFilter = &t
	}

	repoFilter := transaction.TransactionFilter{
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

	return c.JSON(http.StatusOK, APIResponse[[]TransactionResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
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

	var req UpdateTransactionRequest
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

	// Получаем существующую транзакцию
	existingTransaction, err := h.repositories.Transaction.GetByID(c.Request().Context(), id)
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

	// Обновляем поля
	if req.Amount != nil {
		existingTransaction.Amount = *req.Amount
	}
	if req.Type != nil {
		existingTransaction.Type = transaction.TransactionType(*req.Type)
	}
	if req.Description != nil {
		existingTransaction.Description = *req.Description
	}
	if req.CategoryID != nil {
		existingTransaction.CategoryID = *req.CategoryID
	}
	if req.Date != nil {
		existingTransaction.Date = *req.Date
	}
	if req.Tags != nil {
		existingTransaction.Tags = req.Tags
	}
	existingTransaction.UpdatedAt = time.Now()

	if err := h.repositories.Transaction.Update(c.Request().Context(), existingTransaction); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "UPDATE_FAILED",
				Message: "Failed to update transaction",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := TransactionResponse{
		ID:          existingTransaction.ID,
		Amount:      existingTransaction.Amount,
		Type:        string(existingTransaction.Type),
		Description: existingTransaction.Description,
		CategoryID:  existingTransaction.CategoryID,
		UserID:      existingTransaction.UserID,
		FamilyID:    existingTransaction.FamilyID,
		Date:        existingTransaction.Date,
		Tags:        existingTransaction.Tags,
		CreatedAt:   existingTransaction.CreatedAt,
		UpdatedAt:   existingTransaction.UpdatedAt,
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

func (h *TransactionHandler) DeleteTransaction(c echo.Context) error {
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

	if err := h.repositories.Transaction.Delete(c.Request().Context(), id); err != nil {
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

	return c.JSON(http.StatusOK, APIResponse[interface{}]{
		Data: map[string]string{"message": "Transaction deleted successfully"},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}
