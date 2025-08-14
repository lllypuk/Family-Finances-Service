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

// DeleteEntityHelper provides common delete functionality for all handlers
func DeleteEntityHelper(c echo.Context, deleter func(uuid.UUID) error, entityType string) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid " + strings.ToLower(entityType) + " ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	err = deleter(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    strings.ToUpper(entityType) + "_NOT_FOUND",
				Message: entityType + " not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	return c.JSON(http.StatusOK, APIResponse[map[string]string]{
		Data: map[string]string{"message": entityType + " deleted successfully"},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

// ParseIDParam extracts and validates UUID from request parameter
func ParseIDParam(c echo.Context) (uuid.UUID, error) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		jsonErr := c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
		return uuid.Nil, jsonErr
	}
	return id, nil
}

// HandleNotFoundError returns standardized not found error response
func HandleNotFoundError(c echo.Context, entityType string) error {
	return c.JSON(http.StatusNotFound, ErrorResponse{
		Error: ErrorDetail{
			Code:    strings.ToUpper(entityType) + "_NOT_FOUND",
			Message: entityType + " not found",
		},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

// HandleUpdateError returns standardized update error response
func HandleUpdateError(c echo.Context, entityType string) error {
	return c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorDetail{
			Code:    "UPDATE_FAILED",
			Message: "Failed to update " + strings.ToLower(entityType),
		},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

// ReturnSuccessResponse returns standardized success response with data
func ReturnSuccessResponse[T any](c echo.Context, data T) error {
	return c.JSON(http.StatusOK, APIResponse[T]{
		Data: data,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
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
	return c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: ErrorDetail{
			Code:    "INVALID_ID",
			Message: "Invalid " + strings.ToLower(entityType) + " ID format",
		},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

// HandleValidationError processes validation errors and returns standardized error response
func HandleValidationError(c echo.Context, err error) error {
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

// HandleBindError returns standardized bind error response
func HandleBindError(c echo.Context) error {
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
