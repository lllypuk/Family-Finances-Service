package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

var ErrFamilyNotFound = errors.New("family not found")

type UserHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewUserHandler(repositories *Repositories) *UserHandler {
	return &UserHandler{
		repositories: repositories,
		validator:    validator.New(),
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

	// Проверяем существование семьи
	if err := h.validateFamilyExists(c, req.FamilyID); err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrorDetail{
					Code:    "FAMILY_NOT_FOUND",
					Message: "Family not found",
					Details: fmt.Sprintf("Family with ID %s does not exist", req.FamilyID),
				},
				Meta: ResponseMeta{
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
					Timestamp: time.Now(),
					Version:   "v1",
				},
			})
		}
		return err
	}

	return h.createUserEntity(c, req)
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

	foundUser, err := h.repositories.User.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "USER_NOT_FOUND",
				Message: "User not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

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
	helper := NewUpdateEntityHelper(
		UpdateEntityParams[UpdateUserRequest, *user.User, UserResponse]{
			Validator: h.validator,
			GetByID: func(c echo.Context, id uuid.UUID) (*user.User, error) {
				return h.repositories.User.GetByID(c.Request().Context(), id)
			},
			Update: func(c echo.Context, entity *user.User) error {
				return h.repositories.User.Update(c.Request().Context(), entity)
			},
			UpdateFields:  h.updateUserFields,
			BuildResponse: h.buildUserResponse,
			EntityType:    "user",
		})

	return helper.Execute(c)
}

func (h *UserHandler) updateUserFields(user *user.User, req *UpdateUserRequest) {
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	user.UpdatedAt = time.Now()
}

func (h *UserHandler) buildUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		FamilyID:  u.FamilyID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.repositories.User.Delete(c.Request().Context(), id)
	}, "User")
}

func (h *UserHandler) validateFamilyExists(c echo.Context, familyID uuid.UUID) error {
	family, err := h.repositories.Family.GetByID(c.Request().Context(), familyID)
	if err != nil {
		return ErrFamilyNotFound
	}
	if family == nil {
		return ErrFamilyNotFound
	}
	return nil
}

func (h *UserHandler) createUserEntity(c echo.Context, req CreateUserRequest) error {
	// Создаем нового пользователя
	newUser := &user.User{
		ID:        uuid.New(),
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		FamilyID:  req.FamilyID,
		Role:      user.Role(req.Role),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Хешируем пароль (в будущем добавим bcrypt)

	if err := h.repositories.User.Create(c.Request().Context(), newUser); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create user",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := UserResponse{
		ID:        newUser.ID,
		Email:     newUser.Email,
		FirstName: newUser.FirstName,
		LastName:  newUser.LastName,
		Role:      string(newUser.Role),
		FamilyID:  newUser.FamilyID,
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
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
