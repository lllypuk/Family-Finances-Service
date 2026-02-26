package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type TransactionHandler struct {
	repositories       *Repositories
	transactionService services.TransactionService
	validator          *validator.Validate
	logger             *slog.Logger
}

var errResponseAlreadyWritten = errors.New("response already written")

func NewTransactionHandler(
	repositories *Repositories,
	transactionServices ...services.TransactionService,
) *TransactionHandler {
	var transactionService services.TransactionService
	if len(transactionServices) > 0 {
		transactionService = transactionServices[0]
	}

	return &TransactionHandler{
		repositories:       repositories,
		transactionService: transactionService,
		validator:          validator.New(),
		logger:             slog.Default(),
	}
}

func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	var req CreateTransactionRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
	}

	if err := h.validator.Struct(req); err != nil {
		return HandleValidationError(c, err)
	}

	if h.transactionService != nil {
		return h.createTransactionViaService(c, req)
	}

	newTransaction := h.buildTransaction(req)

	if err := h.repositories.Transaction.Create(c.Request().Context(), newTransaction); err != nil {
		// Check if it's a foreign key constraint error
		if h.isForeignKeyConstraintError(err) {
			return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid category, user, or family ID")
		}

		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create transaction")
	}

	h.updateBudgetIfNeeded(c, newTransaction)

	response := h.buildTransactionResponse(newTransaction)
	return respondAPI(c, http.StatusCreated, response)
}

func (h *TransactionHandler) createTransactionViaService(c echo.Context, req CreateTransactionRequest) error {
	createdTx, err := h.transactionService.CreateTransaction(c.Request().Context(), dto.CreateTransactionDTO{
		Amount:      req.Amount,
		Type:        transaction.Type(req.Type),
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		Date:        req.Date,
		Tags:        req.Tags,
	})
	if err != nil {
		return h.handleCreateTransactionServiceError(c, err)
	}

	response := h.buildTransactionResponse(createdTx)
	return respondAPI(c, http.StatusCreated, response)
}

func (h *TransactionHandler) handleCreateTransactionServiceError(c echo.Context, err error) error {
	message := "Failed to create transaction"

	switch {
	case strings.Contains(err.Error(), "category not found"),
		strings.Contains(err.Error(), "user not found"),
		errors.Is(err, services.ErrCategoryNotInFamily),
		errors.Is(err, services.ErrUserNotInFamily):
		message = "Invalid category, user, or family ID"
	case errors.Is(err, services.ErrInsufficientBudget):
		message = "Transaction would exceed budget limit"
	}

	switch {
	case errors.Is(err, services.ErrInsufficientBudget),
		errors.Is(err, services.ErrInvalidTransactionAmount),
		errors.Is(err, services.ErrInvalidTransactionType),
		errors.Is(err, services.ErrCategoryNotInFamily),
		errors.Is(err, services.ErrUserNotInFamily),
		strings.Contains(err.Error(), "validation failed"),
		strings.Contains(err.Error(), "user not found"),
		strings.Contains(err.Error(), "category not found"):
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", message, err.Error())
	default:
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create transaction")
	}
}

func (h *TransactionHandler) buildTransaction(req CreateTransactionRequest) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      req.Amount,
		Type:        transaction.Type(req.Type),
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
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

	budgets, err := h.repositories.Budget.GetActiveBudgets(c.Request().Context())
	if err != nil {
		if h.logger != nil {
			h.logger.WarnContext(
				c.Request().Context(),
				"failed to load active budgets after transaction create",
				slog.String("transaction_id", tx.ID.String()),
				slog.String("category_id", tx.CategoryID.String()),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	for _, b := range budgets {
		if b.CategoryID != nil && *b.CategoryID == tx.CategoryID {
			b.Spent += tx.Amount
			b.UpdatedAt = time.Now()
			if updateErr := h.repositories.Budget.Update(c.Request().Context(), b); updateErr != nil {
				if h.logger != nil {
					h.logger.WarnContext(
						c.Request().Context(),
						"failed to update budget spent after transaction create",
						slog.String("transaction_id", tx.ID.String()),
						slog.String("budget_id", b.ID.String()),
						slog.String("error", updateErr.Error()),
					)
				}
			}
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
		Date:        tx.Date,
		Tags:        tx.Tags,
		CreatedAt:   tx.CreatedAt,
		UpdatedAt:   tx.UpdatedAt,
	}
}

func (h *TransactionHandler) GetTransactions(c echo.Context) error {
	filters, err := h.parseTransactionFilters(c)
	if err != nil {
		if errors.Is(err, errResponseAlreadyWritten) {
			return nil
		}
		return err
	}

	err = h.validateTransactionFilters(c, filters)
	if err != nil {
		return err
	}

	if h.transactionService != nil {
		return h.getTransactionsViaService(c, filters)
	}

	repoFilter := h.buildRepositoryFilter(filters)

	transactions, err := h.repositories.Transaction.GetByFilter(c.Request().Context(), repoFilter)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch transactions")
	}

	response := h.buildTransactionListResponse(transactions)

	return respondAPI(c, http.StatusOK, response)
}

func (h *TransactionHandler) getTransactionsViaService(c echo.Context, filters TransactionFilterParams) error {
	transactions, err := h.transactionService.GetAllTransactions(
		c.Request().Context(),
		h.buildTransactionServiceFilter(filters),
	)
	if err != nil {
		if errors.Is(err, dto.ErrInvalidDateRange) ||
			errors.Is(err, dto.ErrInvalidAmountRange) ||
			strings.Contains(err.Error(), "validation failed") {
			return respondError(
				c,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				"Invalid transaction filters",
				err.Error(),
			)
		}
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch transactions")
	}

	return respondAPI(c, http.StatusOK, h.buildTransactionListResponse(transactions))
}

func (h *TransactionHandler) parseTransactionFilters(c echo.Context) (TransactionFilterParams, error) {
	var filters TransactionFilterParams

	// In single-family model, FamilyID is not needed in filters
	// Repository will handle it internally

	if err := h.parseOptionalFilters(c, &filters); err != nil {
		return TransactionFilterParams{}, err
	}
	if err := h.parsePaginationParams(c, &filters); err != nil {
		return TransactionFilterParams{}, err
	}

	return filters, nil
}

func (h *TransactionHandler) parseOptionalFilters(c echo.Context, filters *TransactionFilterParams) error {
	if userIDParam := c.QueryParam("user_id"); userIDParam != "" {
		userID, parseErr := uuid.Parse(userIDParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "user_id", userIDParam, "must be a valid UUID")
		}
		filters.UserID = &userID
	}

	if categoryIDParam := c.QueryParam("category_id"); categoryIDParam != "" {
		categoryID, parseErr := uuid.Parse(categoryIDParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "category_id", categoryIDParam, "must be a valid UUID")
		}
		filters.CategoryID = &categoryID
	}

	if typeParam := c.QueryParam("type"); typeParam != "" {
		filters.Type = &typeParam
	}

	if dateFromParam := c.QueryParam("date_from"); dateFromParam != "" {
		dateFrom, parseErr := time.Parse(time.RFC3339, dateFromParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "date_from", dateFromParam, "must be RFC3339 datetime")
		}
		filters.DateFrom = &dateFrom
	}

	if dateToParam := c.QueryParam("date_to"); dateToParam != "" {
		dateTo, parseErr := time.Parse(time.RFC3339, dateToParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "date_to", dateToParam, "must be RFC3339 datetime")
		}
		filters.DateTo = &dateTo
	}

	if amountFromParam := c.QueryParam("amount_from"); amountFromParam != "" {
		amountFrom, parseErr := strconv.ParseFloat(amountFromParam, 64)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "amount_from", amountFromParam, "must be a valid number")
		}
		filters.AmountFrom = &amountFrom
	}

	if amountToParam := c.QueryParam("amount_to"); amountToParam != "" {
		amountTo, parseErr := strconv.ParseFloat(amountToParam, 64)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "amount_to", amountToParam, "must be a valid number")
		}
		filters.AmountTo = &amountTo
	}

	if descriptionParam := c.QueryParam("description"); descriptionParam != "" {
		filters.Description = &descriptionParam
	}

	return nil
}

func (h *TransactionHandler) parsePaginationParams(c echo.Context, filters *TransactionFilterParams) error {
	filters.Limit = 50 // По умолчанию
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, parseErr := strconv.Atoi(limitParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "limit", limitParam, "must be an integer between 1 and 1000")
		}
		if limit <= 0 || limit > 1000 {
			return h.writeInvalidQueryParamError(c, "limit", limitParam, "must be an integer between 1 and 1000")
		}
		filters.Limit = limit
	}

	filters.Offset = 0 // По умолчанию
	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		offset, parseErr := strconv.Atoi(offsetParam)
		if parseErr != nil {
			return h.writeInvalidQueryParamError(c, "offset", offsetParam, "must be a non-negative integer")
		}
		if offset < 0 {
			return h.writeInvalidQueryParamError(c, "offset", offsetParam, "must be a non-negative integer")
		}
		filters.Offset = offset
	}

	return nil
}

func (h *TransactionHandler) invalidQueryParamError(
	c echo.Context,
	param, value, reason string,
) error {
	return respondError(c, http.StatusBadRequest, "INVALID_QUERY_PARAM", "Invalid query parameter", map[string]string{
		"param":  param,
		"value":  value,
		"reason": reason,
	})
}

func (h *TransactionHandler) writeInvalidQueryParamError(
	c echo.Context,
	param, value, reason string,
) error {
	if err := h.invalidQueryParamError(c, param, value, reason); err != nil {
		return err
	}
	return errResponseAlreadyWritten
}

func (h *TransactionHandler) validateTransactionFilters(c echo.Context, filters TransactionFilterParams) error {
	err := h.validator.Struct(filters)
	if err != nil {
		return HandleValidationError(c, err)
	}
	return nil
}

func (h *TransactionHandler) buildTransactionServiceFilter(filters TransactionFilterParams) dto.TransactionFilterDTO {
	filter := dto.NewTransactionFilterDTO()
	filter.UserID = filters.UserID
	filter.CategoryID = filters.CategoryID
	if filters.Type != nil {
		t := transaction.Type(*filters.Type)
		filter.Type = &t
	}
	filter.DateFrom = filters.DateFrom
	filter.DateTo = filters.DateTo
	filter.AmountFrom = filters.AmountFrom
	filter.AmountTo = filters.AmountTo
	filter.Description = filters.Description
	filter.Limit = filters.Limit
	filter.Offset = filters.Offset
	return filter
}

func (h *TransactionHandler) buildRepositoryFilter(filters TransactionFilterParams) transaction.Filter {
	var typeFilter *transaction.Type
	if filters.Type != nil {
		t := transaction.Type(*filters.Type)
		typeFilter = &t
	}

	return transaction.Filter{
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
			Date:        tx.Date,
			Tags:        tx.Tags,
			CreatedAt:   tx.CreatedAt,
			UpdatedAt:   tx.UpdatedAt,
		})
	}
	return response
}

func (h *TransactionHandler) GetTransactionByID(c echo.Context) error {
	if h.transactionService != nil {
		return h.getTransactionByIDViaService(c)
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return HandleIDParseError(c, "transaction")
	}

	foundTransaction, err := h.repositories.Transaction.GetByID(c.Request().Context(), id)
	if err != nil {
		return HandleNotFoundError(c, "Transaction")
	}

	response := TransactionResponse{
		ID:          foundTransaction.ID,
		Amount:      foundTransaction.Amount,
		Type:        string(foundTransaction.Type),
		Description: foundTransaction.Description,
		CategoryID:  foundTransaction.CategoryID,
		UserID:      foundTransaction.UserID,
		Date:        foundTransaction.Date,
		Tags:        foundTransaction.Tags,
		CreatedAt:   foundTransaction.CreatedAt,
		UpdatedAt:   foundTransaction.UpdatedAt,
	}

	return respondAPI(c, http.StatusOK, response)
}

func (h *TransactionHandler) getTransactionByIDViaService(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "transaction")
	if err != nil {
		var idParseErr *IDParseError
		if errors.As(err, &idParseErr) {
			return HandleIDParseError(c, "transaction")
		}
		return err
	}

	foundTransaction, err := h.transactionService.GetTransactionByID(c.Request().Context(), id)
	if err != nil {
		return HandleNotFoundError(c, "Transaction")
	}

	return respondAPI(c, http.StatusOK, h.buildTransactionResponse(foundTransaction))
}

func (h *TransactionHandler) UpdateTransaction(c echo.Context) error {
	if h.transactionService != nil {
		return h.updateTransactionViaService(c)
	}

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

func (h *TransactionHandler) updateTransactionViaService(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "transaction")
	if err != nil {
		var idParseErr *IDParseError
		if errors.As(err, &idParseErr) {
			return HandleIDParseError(c, "transaction")
		}
		return err
	}

	var req UpdateTransactionRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return HandleBindError(c)
	}
	if validationErr := h.validator.Struct(req); validationErr != nil {
		return HandleValidationError(c, validationErr)
	}

	serviceReq := dto.UpdateTransactionDTO{
		Amount:      req.Amount,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Date:        req.Date,
		Tags:        req.Tags,
	}
	if req.Type != nil {
		txType := transaction.Type(*req.Type)
		serviceReq.Type = &txType
	}

	updatedTx, err := h.transactionService.UpdateTransaction(c.Request().Context(), id, serviceReq)
	if err != nil {
		return h.handleUpdateTransactionServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, h.buildTransactionResponse(updatedTx))
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

// isForeignKeyConstraintError checks if the error is a foreign key constraint violation
func (h *TransactionHandler) isForeignKeyConstraintError(err error) bool {
	errStr := err.Error()
	// Database foreign key constraint error messages contain these patterns
	return strings.Contains(errStr, "foreign key constraint") ||
		strings.Contains(errStr, "violates foreign key constraint") ||
		strings.Contains(errStr, "FOREIGN KEY constraint failed")
}

func (h *TransactionHandler) DeleteTransaction(c echo.Context) error {
	if h.transactionService != nil {
		return DeleteEntityHelper(c, func(id uuid.UUID) error {
			return h.transactionService.DeleteTransaction(c.Request().Context(), id)
		}, "Transaction")
	}

	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		// In single-family model, repository handles family ID internally
		return h.repositories.Transaction.Delete(c.Request().Context(), id)
	}, "Transaction")
}

func (h *TransactionHandler) handleUpdateTransactionServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrTransactionNotFound):
		return HandleNotFoundError(c, "Transaction")
	case errors.Is(err, services.ErrInsufficientBudget),
		errors.Is(err, services.ErrInvalidTransactionAmount),
		errors.Is(err, services.ErrInvalidTransactionType),
		errors.Is(err, dto.ErrInvalidDateRange),
		errors.Is(err, dto.ErrInvalidAmountRange),
		strings.Contains(err.Error(), "validation failed"),
		strings.Contains(err.Error(), "category not found"):
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid transaction data", err.Error())
	default:
		return respondError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update transaction")
	}
}
