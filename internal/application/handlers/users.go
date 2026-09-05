package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
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
		validator:    newAPIValidator(),
	}
}

// handleServiceError converts service errors to HTTP responses.
// Every branch goes through respondError, so the envelope (code/message/details
// plus request id, timestamp and API version) is built in exactly one place.
func (h *UserHandler) handleServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrValidationFailed):
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	case errors.Is(err, services.ErrEmailAlreadyExists):
		return respondError(c, http.StatusConflict, "EMAIL_EXISTS", "Email already exists")
	case errors.Is(err, services.ErrUserNotFound):
		return respondError(c, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	case errors.Is(err, services.ErrFamilyNotFound):
		return respondError(c, http.StatusBadRequest, ErrCodeFamilyNotFound, ErrMessageFamilyNotFound)
	case errors.Is(err, services.ErrUnauthorized):
		return respondError(c, http.StatusForbidden, ErrCodeForbidden, ErrMessageForbidden)
	case errors.Is(err, services.ErrCannotDeleteSelf):
		return respondError(c, http.StatusBadRequest, ErrCodeCannotDeleteSelf, ErrMessageCannotDeleteSelf)
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
		return h.handleServiceError(c, err)
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
		return h.handleServiceError(c, err)
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
	response := toUserResponse(updatedUser)

	return respondAPI(c, http.StatusOK, response)
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	// Удаляем от имени владельца сессии: запрет самоудаления — правило сервиса
	// (userService.DeleteUser), сюда возвращается ErrCannotDeleteSelf.
	sessionData, sessionErr := middleware.GetUserFromContext(c)
	if sessionErr != nil {
		return respondUnauthorized(c)
	}

	// Call service
	err = h.userService.DeleteUser(c.Request().Context(), id, sessionData.UserID)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// toUserResponse раскладывает доменного пользователя в API-форму.
func toUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
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
		return h.handleServiceError(c, err)
	}

	response := make([]UserResponse, 0, page.Limit)
	for _, u := range pageSlice(users, page) {
		response = append(response, toUserResponse(u))
	}

	return respondList(c, response, page, len(users))
}

// PatchUser меняет роль пользователя. Понижение последнего администратора
// отбивается сервисом (ErrLastAdmin -> 409).
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
	if req.Role == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldRole, Message: "required", Code: ErrCodeValidationError})
	}

	if roleErr := h.userService.ChangeUserRole(c.Request().Context(), id, user.Role(*req.Role)); roleErr != nil {
		return h.handleServiceError(c, roleErr)
	}

	updatedUser, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toUserResponse(updatedUser))
}
