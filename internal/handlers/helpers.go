package handlers

import (
	"net/http"
	"strings"
	"time"

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

	return c.JSON(http.StatusOK, APIResponse[interface{}]{
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
