package handlers

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/services"
)

const apiVersion = "v1"

const (
	// defaultLimit — размер страницы, если limit не задан.
	defaultLimit = 50
	// maxLimit — потолок страницы на всех списках API (A-08).
	maxLimit = 200
)

// pageParams — разобранные limit/offset одного запроса-списка.
type pageParams struct {
	Limit  int
	Offset int
}

// newAPIValidator — валидатор всех API-хендлеров: имя поля в error.details берётся
// из json-тега, иначе клиент получал бы имя поля Go (StartDate вместо start_date).
func newAPIValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return field.Name
		}

		return name
	})

	return v
}

func apiResponseMeta(c echo.Context) ResponseMeta {
	return ResponseMeta{
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		Timestamp: time.Now(),
		Version:   apiVersion,
	}
}

func respondAPI[T any](c echo.Context, status int, data T) error {
	return c.JSON(status, APIResponse[T]{
		Data: data,
		Meta: apiResponseMeta(c),
	})
}

func respondError(
	c echo.Context,
	status int,
	code, message string,
	details ...ErrorDetail,
) error {
	return c.JSON(status, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: apiResponseMeta(c),
	})
}

// bodyDetail — деталь для ошибки, не привязанной к конкретному полю запроса.
func bodyDetail(code, message string) ErrorDetail {
	return ErrorDetail{Field: fieldBody, Message: message, Code: code}
}

// parsePagination разбирает limit/offset. При выходе за границы ответ 422 уже записан,
// а вызывающему возвращается errResponseAlreadyWritten — гасится через ignoreWritten.
func parsePagination(c echo.Context) (pageParams, error) {
	page := pageParams{Limit: defaultLimit, Offset: 0}

	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err != nil || limit < 1 || limit > maxLimit {
			return page, writeInvalidQueryParam(c, "limit", limitParam,
				"must be an integer between 1 and "+strconv.Itoa(maxLimit))
		}
		page.Limit = limit
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		offset, err := strconv.Atoi(offsetParam)
		if err != nil || offset < 0 {
			return page, writeInvalidQueryParam(c, "offset", offsetParam, "must be a non-negative integer")
		}
		page.Offset = offset
	}

	return page, nil
}

func writeInvalidQueryParam(c echo.Context, param, value, reason string) error {
	if err := respondError(
		c,
		http.StatusUnprocessableEntity,
		ErrCodeValidationError,
		ErrMessageValidationFailed,
		ErrorDetail{Field: param, Message: reason + `, got "` + value + `"`, Code: ErrCodeInvalidQueryParam},
	); err != nil {
		return err
	}

	return errResponseAlreadyWritten
}

// ignoreWritten гасит sentinel: ответ уже отправлен тем, кто вернул ошибку.
func ignoreWritten(err error) error {
	if errors.Is(err, errResponseAlreadyWritten) {
		return nil
	}

	return err
}

// respondList отдаёт список с meta.pagination; data всегда массив, а не null.
func respondList[T any](c echo.Context, items []T, page pageParams, total int) error {
	meta := apiResponseMeta(c)
	meta.Pagination = &PaginationMeta{Limit: page.Limit, Offset: page.Offset, Total: total}

	if items == nil {
		items = []T{}
	}

	return c.JSON(http.StatusOK, APIResponse[[]T]{Data: items, Meta: meta})
}

// pageSlice режет окно [offset, offset+limit) для списков, которые репозиторий отдаёт целиком.
func pageSlice[T any](items []T, page pageParams) []T {
	if page.Offset >= len(items) {
		return []T{}
	}

	end := page.Offset + page.Limit
	if end > len(items) {
		end = len(items)
	}

	return items[page.Offset:end]
}

// RespondAPIError пишет общий envelope для ошибки, не дошедшей до хендлера
// (404 роутера, 405, отбитый CSRF); пустой code выводится из статуса.
func RespondAPIError(c echo.Context, status int, code, message string) error {
	if code == "" {
		code = ErrorCodeForStatus(status)
	}

	return respondError(c, status, code, message)
}

// ErrorCodeForStatus — error.code, соответствующий HTTP-статусу.
func ErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusMethodNotAllowed:
		return ErrCodeMethodNotAllowed
	case http.StatusUnprocessableEntity:
		return ErrCodeValidationError
	default:
		return ErrCodeInternal
	}
}

// respondUnauthorized отдаёт 401 в общем формате ошибок API: так отвечает
// RequireAPIAuth и хендлеры, которым без сессии нечего делать.
func respondUnauthorized(c echo.Context) error {
	return respondError(c, http.StatusUnauthorized, ErrCodeUnauthorized, ErrMessageUnauthorized)
}

// buildValidationErrors раскладывает ошибки validator по полям; на нераспознанной
// ошибке возвращает одну деталь с её текстом, чтобы details никогда не был пустым.
func buildValidationErrors(err error) []ErrorDetail {
	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return []ErrorDetail{bodyDetail(ErrCodeValidationError, err.Error())}
	}

	details := make([]ErrorDetail, 0, len(fieldErrs))
	for _, fieldErr := range fieldErrs {
		details = append(details, ErrorDetail{
			Field:   fieldErr.Field(),
			Message: validationMessage(fieldErr),
			Code:    ErrCodeValidationError,
		})
	}

	return details
}

// validationMessage переводит правило валидатора в текст для клиента.
func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "must be one of: " + fieldErr.Param()
	case "email":
		return "must be a valid email"
	}

	if fieldErr.Param() != "" {
		return "must satisfy " + fieldErr.Tag() + "=" + fieldErr.Param()
	}

	return "must satisfy " + fieldErr.Tag()
}

// respondValidationErrors — единственный вход для 422: тело или параметры не прошли проверку.
func respondValidationErrors(c echo.Context, err error) error {
	return respondError(
		c,
		http.StatusUnprocessableEntity,
		ErrCodeValidationError,
		ErrMessageValidationFailed,
		buildValidationErrors(err)...,
	)
}

// isNotFoundError отличает «сущности нет» от настоящего сбоя: спека требует 404, а не 500.
func isNotFoundError(err error) bool {
	return errors.Is(err, services.ErrTransactionNotFound) ||
		errors.Is(err, services.ErrBudgetNotFoundService) ||
		errors.Is(err, services.ErrReportNotFound) ||
		errors.Is(err, services.ErrCategoryNotFound)
}

// DeleteEntityHelper provides common delete functionality for all handlers
func DeleteEntityHelper(c echo.Context, deleter func(uuid.UUID) error, entityType string) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(
			c,
			http.StatusBadRequest,
			ErrCodeInvalidID,
			"Invalid "+strings.ToLower(entityType)+" ID format",
		)
	}

	err = deleter(id)
	if err != nil {
		if isNotFoundError(err) {
			return HandleNotFoundError(c, entityType)
		}

		return respondError(
			c,
			http.StatusInternalServerError,
			"DELETE_FAILED",
			"Failed to delete "+strings.ToLower(entityType),
		)
	}

	return c.NoContent(http.StatusNoContent)
}

// ParseIDParam extracts and validates UUID from request parameter
func ParseIDParam(c echo.Context) (uuid.UUID, error) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		jsonErr := respondError(c, http.StatusBadRequest, ErrCodeInvalidID, "Invalid ID format")
		return uuid.Nil, jsonErr
	}
	return id, nil
}

// HandleNotFoundError returns standardized not found error response
func HandleNotFoundError(c echo.Context, entityType string) error {
	return respondError(
		c,
		http.StatusNotFound,
		strings.ToUpper(entityType)+"_NOT_FOUND",
		entityType+" not found",
	)
}

// HandleUpdateError returns standardized update error response
func HandleUpdateError(c echo.Context, entityType string) error {
	return respondError(
		c,
		http.StatusInternalServerError,
		"UPDATE_FAILED",
		"Failed to update "+strings.ToLower(entityType),
	)
}

// ReturnSuccessResponse returns standardized success response with data
func ReturnSuccessResponse[T any](c echo.Context, data T) error {
	return respondAPI(c, http.StatusOK, data)
}

// ParseIDParamWithError extracts and validates UUID from request parameter with custom error message
func ParseIDParamWithError(c echo.Context, entityType string) (uuid.UUID, error) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		// Return a sentinel error that the helper can detect and handle
		return uuid.Nil, &IDParseError{EntityType: entityType}
	}
	return id, nil
}

// IDParseError represents an ID parsing error
type IDParseError struct {
	EntityType string
}

func (e *IDParseError) Error() string {
	return "invalid " + strings.ToLower(e.EntityType) + " ID format"
}

// HandleIDParseError returns standardized ID parse error response
func HandleIDParseError(c echo.Context, entityType string) error {
	return respondError(
		c,
		http.StatusBadRequest,
		ErrCodeInvalidID,
		"Invalid "+strings.ToLower(entityType)+" ID format",
	)
}

// HandleBindError returns standardized bind error response
func HandleBindError(c echo.Context) error {
	return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest)
}

// UpdateEntityHelper provides common update functionality for all handlers
type UpdateEntityHelper[TRequest any, TEntity any, TResponse any] struct {
	ParseID       func(echo.Context, string) (uuid.UUID, error)
	BindRequest   func(echo.Context, *TRequest) error
	Validate      func(*TRequest) error
	GetExisting   func(echo.Context, uuid.UUID) (TEntity, error)
	UpdateFields  func(TEntity, *TRequest)
	SaveEntity    func(echo.Context, TEntity) error
	BuildResponse func(TEntity) TResponse
	EntityType    string
}

// UpdateEntityParams contains the parameters needed to create an UpdateEntityHelper
type UpdateEntityParams[TRequest any, TEntity any, TResponse any] struct {
	Validator     *validator.Validate
	GetByID       func(echo.Context, uuid.UUID) (TEntity, error)
	Update        func(echo.Context, TEntity) error
	UpdateFields  func(TEntity, *TRequest)
	BuildResponse func(TEntity) TResponse
	EntityType    string
}

// NewUpdateEntityHelper creates a configured UpdateEntityHelper with standard implementations
func NewUpdateEntityHelper[TRequest any, TEntity any, TResponse any](
	params UpdateEntityParams[TRequest, TEntity, TResponse],
) *UpdateEntityHelper[TRequest, TEntity, TResponse] {
	return &UpdateEntityHelper[TRequest, TEntity, TResponse]{
		ParseID: ParseIDParamWithError,
		BindRequest: func(c echo.Context, req *TRequest) error {
			return c.Bind(req)
		},
		Validate: func(req *TRequest) error {
			return params.Validator.Struct(req)
		},
		GetExisting:   params.GetByID,
		UpdateFields:  params.UpdateFields,
		SaveEntity:    params.Update,
		BuildResponse: params.BuildResponse,
		EntityType:    params.EntityType,
	}
}

// Execute runs the complete update flow
func (h *UpdateEntityHelper[TRequest, TEntity, TResponse]) Execute(c echo.Context) error {
	// Parse and validate ID
	id, err := h.ParseID(c, h.EntityType)
	if err != nil {
		// Check if it's an ID parse error and handle it
		var idParseErr *IDParseError
		if errors.As(err, &idParseErr) {
			return HandleIDParseError(c, h.EntityType)
		}
		return err
	}

	// Bind request
	var req TRequest
	if bindErr := h.BindRequest(c, &req); bindErr != nil {
		return HandleBindError(c)
	}

	// Validate request
	if validationErr := h.Validate(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	// Get existing entity
	existing, err := h.GetExisting(c, id)
	if err != nil {
		return HandleNotFoundError(c, h.EntityType)
	}

	// Update fields
	h.UpdateFields(existing, &req)

	// Save entity
	if saveErr := h.SaveEntity(c, existing); saveErr != nil {
		return HandleUpdateError(c, h.EntityType)
	}

	// Build and return response
	response := h.BuildResponse(existing)
	return ReturnSuccessResponse(c, response)
}
