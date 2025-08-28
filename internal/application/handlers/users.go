package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

var ErrFamilyNotFound = errors.New("family not found")

type UserHandler struct {
	repositories *Repositories
	userService  services.UserService
	validator    *validator.Validate
}

func NewUserHandler(repositories *Repositories, userService services.UserService) *UserHandler {
	return &UserHandler{
		repositories: repositories,
		userService:  userService,
		validator:    validator.New(),
	}
}

// handleServiceError converts service errors to HTTP responses
func (h *UserHandler) handleServiceError(c echo.Context, err error) error {
	requestID := c.Response().Header().Get(echo.HeaderXRequestID)
	timestamp := time.Now()

	switch {
	case errors.Is(err, services.ErrValidationFailed):
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Validation failed",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	case errors.Is(err, services.ErrEmailAlreadyExists):
		return c.JSON(http.StatusConflict, ErrorResponse{
			Error: ErrorDetail{
				Code:    "EMAIL_EXISTS",
				Message: "Email already exists",
				Details: "A user with this email address already exists",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	case errors.Is(err, services.ErrUserNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
				Details: "The requested user does not exist",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	case errors.Is(err, services.ErrFamilyNotFound):
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FAMILY_NOT_FOUND",
				Message: "Family not found",
				Details: "The specified family does not exist",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	case errors.Is(err, services.ErrUnauthorized):
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "Unauthorized access",
				Details: "You don't have permission to access this resource",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	case errors.Is(err, services.ErrInvalidRole):
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ROLE",
				Message: "Invalid role",
				Details: "The specified role is not valid",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	default:
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Internal server error",
				Details: "An unexpected error occurred",
			},
			Meta: ResponseMeta{
				RequestID: requestID,
				Timestamp: timestamp,
				Version:   "v1",
			},
		})
	}
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	var req CreateUserRequest
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

	// Convert API request to DTO
	userDTO := dto.CreateUserDTO{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      user.Role(req.Role),
		FamilyID:  req.FamilyID,
	}

	// Call service
	createdUser, err := h.userService.CreateUser(c.Request().Context(), userDTO)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	// Convert to API response
	response := UserResponse{
		ID:        createdUser.ID,
		Email:     createdUser.Email,
		FirstName: createdUser.FirstName,
		LastName:  createdUser.LastName,
		Role:      string(createdUser.Role),
		FamilyID:  createdUser.FamilyID,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}

	return c.JSON(http.StatusCreated, APIResponse[UserResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *UserHandler) GetUserByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid user ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Call service
	foundUser, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	// Convert to API response
	response := UserResponse{
		ID:        foundUser.ID,
		Email:     foundUser.Email,
		FirstName: foundUser.FirstName,
		LastName:  foundUser.LastName,
		Role:      string(foundUser.Role),
		FamilyID:  foundUser.FamilyID,
		CreatedAt: foundUser.CreatedAt,
		UpdatedAt: foundUser.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[UserResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *UserHandler) UpdateUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid user ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var req UpdateUserRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: bindErr.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Convert API request to DTO
	updateDTO := dto.UpdateUserDTO{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	}

	// Call service
	updatedUser, err := h.userService.UpdateUser(c.Request().Context(), id, updateDTO)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	// Convert to API response
	response := UserResponse{
		ID:        updatedUser.ID,
		Email:     updatedUser.Email,
		FirstName: updatedUser.FirstName,
		LastName:  updatedUser.LastName,
		Role:      string(updatedUser.Role),
		FamilyID:  updatedUser.FamilyID,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[UserResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid user ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	// Call service
	err = h.userService.DeleteUser(c.Request().Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}
