package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type TransactionHandler struct {
	transactionService services.TransactionService
	validator          *validator.Validate
}

var errResponseAlreadyWritten = errors.New("response already written")

// NewTransactionHandler — сервис обязателен: бизнес-правила операции живут только в нём.
func NewTransactionHandler(transactionService services.TransactionService) *TransactionHandler {
	if transactionService == nil {
		panic("handlers: NewTransactionHandler requires a non-nil services.TransactionService")
	}

	return &TransactionHandler{
		transactionService: transactionService,
		validator:          newAPIValidator(),
	}
}

func (h *TransactionHandler) CreateTransaction(c echo.Context) error {
	// Автор записи — владелец токена, которого RequireBearer кладёт в контекст.
	// Единственный допустимый источник: тело запроса им быть не может, иначе
	// аутентифицированный клиент пишет от чужого имени (S-01,
	// docs/specs/002-security-audit.md). Проверяем до разбора тела: без сессии
	// транзакцию всё равно не от кого создавать.
	principal, principalErr := auth.FromContext(c)
	if principalErr != nil {
		return respondUnauthorized(c)
	}
	userID := principal.UserID

	var req CreateTransactionRequest
	if err := c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if handled, err := respondClientID(c, req.ID, h.findTransaction, h.buildTransactionResponse); handled {
		return err
	}

	createdTx, err := h.transactionService.CreateTransaction(c.Request().Context(), dto.CreateTransactionDTO{
		ID:          req.ID,
		AmountMinor: req.AmountMinor,
		Type:        transaction.Type(req.Type),
		Description: req.Description,
		CategoryID:  req.CategoryID,
		AccountID:   req.AccountID,
		UserID:      userID,
		Date:        req.Date,
		Tags:        req.Tags,
	})
	if err != nil {
		return h.handleCreateTransactionServiceError(c, err)
	}

	return respondAPI(c, http.StatusCreated, h.buildTransactionResponse(createdTx))
}

// findTransaction ищет запись по клиентскому id; ошибка сервиса здесь означает
// «не найдено» — создание всё равно упрётся в неё повторно.
func (h *TransactionHandler) findTransaction(c echo.Context, id uuid.UUID) (*transaction.Transaction, bool) {
	tx, err := h.transactionService.GetTransactionByID(c.Request().Context(), id)

	return tx, err == nil
}

// respondAccountInvalid — 422 на счёт, который нельзя привязать к операции.
func respondAccountInvalid(c echo.Context) error {
	return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: fieldAccountID, Message: "must be an existing active account", Code: ErrCodeValidationError})
}

func (h *TransactionHandler) handleCreateTransactionServiceError(c echo.Context, err error) error {
	if errors.Is(err, services.ErrTransactionAccountInvalid) {
		return respondAccountInvalid(c)
	}

	message := "Failed to create transaction"

	switch {
	case strings.Contains(err.Error(), "category not found"),
		strings.Contains(err.Error(), "user not found"),
		errors.Is(err, services.ErrCategoryNotInFamily),
		errors.Is(err, services.ErrUserNotInFamily):
		message = ErrMessageInvalidCategoryRef
	case errors.Is(err, services.ErrInsufficientBudget):
		message = "Transaction would exceed budget limit"
	}

	switch {
	case errors.Is(err, services.ErrInsufficientBudget),
		errors.Is(err, services.ErrInvalidTransactionAmount),
		errors.Is(err, services.ErrInvalidTransactionType),
		errors.Is(err, services.ErrTransactionAmountTooLarge),
		errors.Is(err, services.ErrTransactionDateOutOfRange),
		errors.Is(err, services.ErrCategoryNotInFamily),
		errors.Is(err, services.ErrUserNotInFamily),
		strings.Contains(err.Error(), "validation failed"),
		strings.Contains(err.Error(), "user not found"),
		strings.Contains(err.Error(), "category not found"):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, message,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return respondError(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create transaction")
	}
}

// buildTransactionResponse — tags в контракте обязательный массив, поэтому nil становится [].
func (h *TransactionHandler) buildTransactionResponse(tx *transaction.Transaction) TransactionResponse {
	tags := tx.Tags
	if tags == nil {
		tags = []string{}
	}

	return TransactionResponse{
		ID:          tx.ID,
		AmountMinor: tx.AmountMinor,
		Type:        string(tx.Type),
		Description: tx.Description,
		CategoryID:  tx.CategoryID,
		AccountID:   tx.AccountID,
		UserID:      tx.UserID,
		Date:        tx.Date,
		Tags:        tags,
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

	serviceFilter := h.buildTransactionServiceFilter(filters)

	transactions, err := h.transactionService.GetAllTransactions(c.Request().Context(), serviceFilter)
	if err != nil {
		if errors.Is(err, dto.ErrInvalidDateRange) ||
			errors.Is(err, dto.ErrInvalidAmountRange) ||
			strings.Contains(err.Error(), "validation failed") {
			return respondError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeValidationError,
				"Invalid transaction filters",
				bodyDetail(ErrCodeValidationError, err.Error()),
			)
		}
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch transactions")
	}

	total, err := h.transactionService.CountTransactions(c.Request().Context(), serviceFilter)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch transactions")
	}

	return respondList(
		c,
		h.buildTransactionListResponse(transactions),
		pageParams{Limit: filters.Limit, Offset: filters.Offset},
		total,
	)
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
			return writeInvalidQueryParam(c, "user_id", userIDParam, "must be a valid UUID")
		}
		filters.UserID = &userID
	}

	for _, raw := range c.QueryParams()["category_id"] {
		if raw == "" {
			continue
		}
		categoryID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return writeInvalidQueryParam(c, "category_id", raw, "must be a valid UUID")
		}
		if !slices.Contains(filters.CategoryIDs, categoryID) {
			filters.CategoryIDs = append(filters.CategoryIDs, categoryID)
		}
	}

	if err := h.parseAccountFilters(c, filters); err != nil {
		return err
	}

	if typeParam := c.QueryParam("type"); typeParam != "" {
		filters.Type = &typeParam
	}

	if err := h.parseDateFilters(c, filters); err != nil {
		return err
	}

	if err := h.parseAmountFilters(c, filters); err != nil {
		return err
	}

	if descriptionParam := c.QueryParam("description"); descriptionParam != "" {
		filters.Description = &descriptionParam
	}

	return nil
}

// parseAccountFilters — account_id и unassigned=true вместе дали бы заведомо пустую выборку, поэтому 422.
func (h *TransactionHandler) parseAccountFilters(c echo.Context, filters *TransactionFilterParams) error {
	if raw := c.QueryParam(fieldAccountID); raw != "" {
		accountID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return writeInvalidQueryParam(c, fieldAccountID, raw, "must be a valid UUID")
		}
		filters.AccountID = &accountID
	}

	if raw := c.QueryParam(fieldUnassigned); raw != "" {
		unassigned, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return writeInvalidQueryParam(c, fieldUnassigned, raw, "must be a boolean")
		}
		filters.Unassigned = unassigned
	}

	if filters.AccountID != nil && filters.Unassigned {
		return writeInvalidQueryParam(c, fieldUnassigned, "true", "must not be combined with account_id")
	}

	return nil
}

// parseDateFilters — обе границы календарные и включительные.
func (h *TransactionHandler) parseDateFilters(c echo.Context, filters *TransactionFilterParams) error {
	for _, bound := range []struct {
		param string
		dst   **date.Date
	}{
		{"date_from", &filters.DateFrom},
		{"date_to", &filters.DateTo},
	} {
		raw := c.QueryParam(bound.param)
		if raw == "" {
			continue
		}
		parsed, parseErr := date.Parse(raw)
		if parseErr != nil {
			return writeInvalidQueryParam(c, bound.param, raw, "must be a date in YYYY-MM-DD format")
		}
		*bound.dst = &parsed
	}

	return nil
}

func (h *TransactionHandler) parseAmountFilters(c echo.Context, filters *TransactionFilterParams) error {
	for _, bound := range []struct {
		param string
		dst   **money.Minor
	}{
		{"amount_from_minor", &filters.AmountFromMinor},
		{"amount_to_minor", &filters.AmountToMinor},
	} {
		raw := c.QueryParam(bound.param)
		if raw == "" {
			continue
		}
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			return writeInvalidQueryParam(c, bound.param, raw, "must be an integer amount in minor units")
		}
		amount := money.Minor(parsed)
		*bound.dst = &amount
	}

	return nil
}

func (h *TransactionHandler) parsePaginationParams(c echo.Context, filters *TransactionFilterParams) error {
	page, err := parsePagination(c)
	if err != nil {
		return err
	}

	filters.Limit = page.Limit
	filters.Offset = page.Offset

	return nil
}

func (h *TransactionHandler) validateTransactionFilters(c echo.Context, filters TransactionFilterParams) error {
	err := h.validator.Struct(filters)
	if err != nil {
		return respondValidationErrors(c, err)
	}
	return nil
}

func (h *TransactionHandler) buildTransactionServiceFilter(filters TransactionFilterParams) dto.TransactionFilterDTO {
	filter := dto.NewTransactionFilterDTO()
	filter.UserID = filters.UserID
	filter.CategoryIDs = filters.CategoryIDs
	filter.AccountID = filters.AccountID
	filter.Unassigned = filters.Unassigned
	if filters.Type != nil {
		t := transaction.Type(*filters.Type)
		filter.Type = &t
	}
	filter.DateFrom = filters.DateFrom
	filter.DateTo = filters.DateTo
	filter.AmountFromMinor = filters.AmountFromMinor
	filter.AmountToMinor = filters.AmountToMinor
	filter.Description = filters.Description
	filter.Limit = filters.Limit
	filter.Offset = filters.Offset
	return filter
}

func (h *TransactionHandler) buildTransactionListResponse(
	transactions []*transaction.Transaction,
) []TransactionResponse {
	response := make([]TransactionResponse, 0, len(transactions))
	for _, tx := range transactions {
		response = append(response, h.buildTransactionResponse(tx))
	}
	return response
}

func (h *TransactionHandler) GetTransactionByID(c echo.Context) error {
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
		return respondBindError(c, bindErr)
	}
	if validationErr := h.validator.Struct(req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}
	if req.isEmpty() {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}
	clearAccount := req.ClearAccount != nil && *req.ClearAccount
	if clearAccount && req.AccountID != nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldClearAcct, Message: "must not be combined with account_id",
				Code: ErrCodeValidationError})
	}

	serviceReq := dto.UpdateTransactionDTO{
		Description:  req.Description,
		CategoryID:   req.CategoryID,
		AccountID:    req.AccountID,
		ClearAccount: clearAccount,
		Tags:         req.Tags,
	}
	serviceReq.AmountMinor = req.AmountMinor
	serviceReq.Date = req.Date
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

func (h *TransactionHandler) DeleteTransaction(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.transactionService.DeleteTransaction(c.Request().Context(), id)
	}, "Transaction")
}

// BulkDeleteTransactions удаляет несколько транзакций за один запрос; неизвестные id
// не ошибка — ответ несёт число фактически удалённых записей.
func (h *TransactionHandler) BulkDeleteTransactions(c echo.Context) error {
	var req BulkDeleteRequest
	if err := c.Bind(&req); err != nil {
		return respondBindError(c, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	deleted, err := h.transactionService.BulkDelete(c.Request().Context(), req.IDs)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete transactions")
	}

	return respondAPI(c, http.StatusOK, BulkDeleteResponse{Deleted: deleted})
}

func (h *TransactionHandler) handleUpdateTransactionServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrTransactionNotFound):
		return HandleNotFoundError(c, "Transaction")
	case errors.Is(err, services.ErrTransactionAccountInvalid):
		return respondAccountInvalid(c)
	case errors.Is(err, services.ErrInsufficientBudget),
		errors.Is(err, services.ErrInvalidTransactionAmount),
		errors.Is(err, services.ErrInvalidTransactionType),
		errors.Is(err, services.ErrTransactionAmountTooLarge),
		errors.Is(err, services.ErrTransactionDateOutOfRange),
		errors.Is(err, dto.ErrInvalidDateRange),
		errors.Is(err, dto.ErrInvalidAmountRange),
		strings.Contains(err.Error(), "validation failed"),
		strings.Contains(err.Error(), "category not found"):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageInvalidTransaction,
			bodyDetail(ErrCodeValidationError, err.Error()))
	default:
		return respondError(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update transaction")
	}
}
