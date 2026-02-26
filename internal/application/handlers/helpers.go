package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const apiVersion = "v1"

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
	details ...any,
) error {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		Meta: apiResponseMeta(c),
	}
	if len(details) > 0 {
		resp.Error.Details = details[0]
	}

	return c.JSON(status, resp)
}

func buildValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError
	for _, fieldErr := range func() validator.ValidationErrors {
		var target validator.ValidationErrors
		_ = errors.As(err, &target)
		return target
	}() {
		validationErrors = append(validationErrors, ValidationError{
			Field:   fieldErr.Field(),
			Message: fieldErr.Tag(),
			Code:    "VALIDATION_ERROR",
		})
	}

	return validationErrors
}

func respondValidationErrors(c echo.Context, validationErrors []ValidationError) error {
	return c.JSON(http.StatusBadRequest, APIResponse[any]{
		Data:   nil,
		Meta:   apiResponseMeta(c),
		Errors: validationErrors,
	})
}

// DeleteEntityHelper provides common delete functionality for all handlers
func DeleteEntityHelper(c echo.Context, deleter func(uuid.UUID) error, entityType string) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(
			c,
			http.StatusBadRequest,
			"INVALID_ID",
			"Invalid "+strings.ToLower(entityType)+" ID format",
		)
	}

	err = deleter(id)
	if err != nil {
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
		jsonErr := respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid ID format")
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
		"INVALID_ID",
		"Invalid "+strings.ToLower(entityType)+" ID format",
	)
}

// HandleValidationError processes validation errors and returns standardized error response
func HandleValidationError(c echo.Context, err error) error {
	return respondValidationErrors(c, buildValidationErrors(err))
}

// HandleBindError returns standardized bind error response
func HandleBindError(c echo.Context) error {
	return respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
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
		return HandleValidationError(c, validationErr)
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
