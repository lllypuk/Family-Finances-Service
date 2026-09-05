package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

var ErrFamilyNotFound = errors.New("family not found")

type UserHandler struct {
	userService services.UserService
	authService AuthService
	validator   *validator.Validate
}

func NewUserHandler(userService services.UserService, authService AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
		validator:   newAPIValidator(),
	}
}

// respondUserServiceError — ошибки UserService в envelope; общая для /users и /me.
// Every branch goes through respondError, so the envelope (code/message/details
// plus request id, timestamp and API version) is built in exactly one place.
func respondUserServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrValidationFailed):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	case errors.Is(err, services.ErrEmailAlreadyExists):
		return respondError(c, http.StatusConflict, ErrCodeEmailTaken, ErrMessageEmailTaken)
	case errors.Is(err, services.ErrUserNotFound):
		return HandleNotFoundError(c, entityUser)
	case errors.Is(err, services.ErrFamilyNotFound):
		return respondError(c, http.StatusBadRequest, ErrCodeFamilyNotFound, ErrMessageFamilyNotFound)
	case errors.Is(err, services.ErrCannotDeactivateSelf):
		return respondError(c, http.StatusConflict, ErrCodeCannotDeactivateSelf, ErrMessageCannotDeactivate)
	case errors.Is(err, services.ErrLastAdmin):
		return respondError(c, http.StatusConflict, ErrCodeLastAdmin, ErrMessageLastAdmin)
	case errors.Is(err, services.ErrInvalidRole):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldRole, Message: "invalid role", Code: ErrCodeValidationError})
	default:
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func (h *UserHandler) CreateUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, err.Error()))
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	// Convert API request to DTO
	userDTO := dto.CreateUserDTO{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      user.Role(req.Role),
	}

	// Call service
	createdUser, err := h.userService.CreateUser(c.Request().Context(), userDTO)
	if err != nil {
		return respondUserServiceError(c, err)
	}

	// Convert to API response
	response := toUserResponse(createdUser)

	return respondAPI(c, http.StatusCreated, response)
}

func (h *UserHandler) GetUserByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	// Call service
	foundUser, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return respondUserServiceError(c, err)
	}

	// Convert to API response
	response := toUserResponse(foundUser)

	return respondAPI(c, http.StatusOK, response)
}

func (h *UserHandler) UpdateUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req UpdateUserRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
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
		return respondUserServiceError(c, err)
	}

	// Convert to API response
	response := toUserResponse(updatedUser)

	return respondAPI(c, http.StatusOK, response)
}

// toUserResponse раскладывает доменного пользователя в API-форму.
func toUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// GetUsers отдаёт всех пользователей семьи; группа /users закрыта adminOnly.
func (h *UserHandler) GetUsers(c echo.Context) error {
	page, pageErr := parsePagination(c)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	users, err := h.userService.GetUsers(c.Request().Context())
	if err != nil {
		return respondUserServiceError(c, err)
	}

	response := make([]UserResponse, 0, page.Limit)
	for _, u := range pageSlice(users, page) {
		response = append(response, toUserResponse(u))
	}

	return respondList(c, response, page, len(users))
}

// PatchUser меняет роль и/или активность. Оба правила (последний админ, самодеактивация)
// живут в сервисе и приходят сюда как sentinel-ошибки -> 409.
func (h *UserHandler) PatchUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req PatchUserRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}
	if req.Role == nil && req.IsActive == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, "at least one field is required"))
	}

	// Актор — владелец токена: запрет самодеактивации.
	principal, principalErr := auth.FromContext(c)
	if principalErr != nil {
		return respondUnauthorized(c)
	}

	ctx := c.Request().Context()
	if req.Role != nil {
		if roleErr := h.userService.ChangeUserRole(ctx, id, user.Role(*req.Role)); roleErr != nil {
			return respondUserServiceError(c, roleErr)
		}
	}
	if req.IsActive != nil {
		if activeErr := h.userService.SetActive(ctx, id, *req.IsActive, principal.UserID); activeErr != nil {
			return respondUserServiceError(c, activeErr)
		}
	}

	updatedUser, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		return respondUserServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toUserResponse(updatedUser))
}

// SetUserPassword задаёт пароль без текущего; все сессии пользователя отзываются.
func (h *UserHandler) SetUserPassword(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req SetPasswordRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return HandleBindError(c)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	if err = h.authService.AdminSetPassword(c.Request().Context(), id, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return HandleNotFoundError(c, entityUser)
		case errors.Is(err, auth.ErrInvalidPassword):
			return respondError(
				c,
				http.StatusUnprocessableEntity,
				ErrCodeValidationError,
				ErrMessageValidationFailed,
				ErrorDetail{
					Field:   fieldNewPassword,
					Message: auth.ErrInvalidPassword.Error(),
					Code:    ErrCodeValidationError,
				},
			)
		default:
			return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
