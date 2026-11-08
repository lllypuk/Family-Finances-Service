package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

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
func respondUserServiceError(w http.ResponseWriter, err error) error {
	switch {
	case errors.Is(err, services.ErrValidationFailed):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, err.Error()))
	case errors.Is(err, services.ErrEmailAlreadyExists):
		return respondError(w, http.StatusConflict, ErrCodeEmailTaken, ErrMessageEmailTaken)
	case errors.Is(err, services.ErrUserNotFound):
		return handleNotFoundError(w, entityUser)
	case errors.Is(err, services.ErrFamilyNotFound):
		return respondError(w, http.StatusBadRequest, ErrCodeFamilyNotFound, ErrMessageFamilyNotFound)
	case errors.Is(err, services.ErrCannotDeactivateSelf):
		return respondError(w, http.StatusConflict, ErrCodeCannotDeactivateSelf, ErrMessageCannotDeactivate)
	case errors.Is(err, services.ErrLastAdmin):
		return respondError(w, http.StatusConflict, ErrCodeLastAdmin, ErrMessageLastAdmin)
	case errors.Is(err, services.ErrInvalidRole):
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: fieldRole, Message: "invalid role", Code: ErrCodeValidationError})
	default:
		return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	var req CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		return respondBindError(w, err)
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
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
	createdUser, err := h.userService.CreateUser(r.Context(), userDTO)
	if err != nil {
		return respondUserServiceError(w, err)
	}

	// Convert to API response
	response := toUserResponse(createdUser)

	return respondAPI(w, http.StatusCreated, response)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) error {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	// Call service
	foundUser, err := h.userService.GetUserByID(r.Context(), id)
	if err != nil {
		return respondUserServiceError(w, err)
	}

	// Convert to API response
	response := toUserResponse(foundUser)

	return respondAPI(w, http.StatusOK, response)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) error {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req UpdateUserRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}

	// Convert API request to DTO
	updateDTO := dto.UpdateUserDTO{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	}

	// Call service
	updatedUser, err := h.userService.UpdateUser(r.Context(), id, updateDTO)
	if err != nil {
		return respondUserServiceError(w, err)
	}

	// Convert to API response
	response := toUserResponse(updatedUser)

	return respondAPI(w, http.StatusOK, response)
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
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) error {
	page, pageErr := parsePagination(w, r)
	if pageErr != nil {
		return ignoreWritten(pageErr)
	}

	users, err := h.userService.GetUsers(r.Context())
	if err != nil {
		return respondUserServiceError(w, err)
	}

	response := make([]UserResponse, 0, page.Limit)
	for _, u := range pageSlice(users, page) {
		response = append(response, toUserResponse(u))
	}

	return respondList(w, response, page, len(users))
}

// PatchUser меняет роль и/или активность одной записью: при отказе не применено ни одно поле.
// Оба правила (последний админ, самодеактивация) приходят сюда как sentinel-ошибки -> 409.
func (h *UserHandler) PatchUser(w http.ResponseWriter, r *http.Request) error {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req PatchUserRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}

	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}
	if req.Role == nil && req.IsActive == nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	// Актор — владелец токена: запрет самодеактивации.
	principal, principalErr := auth.PrincipalFromContext(r.Context())
	if principalErr != nil {
		return respondUnauthorized(w)
	}

	ctx := r.Context()
	var role *user.Role
	if req.Role != nil {
		parsed := user.Role(*req.Role)
		role = &parsed
	}
	if patchErr := h.userService.PatchUser(ctx, id, role, req.IsActive, principal.UserID); patchErr != nil {
		return respondUserServiceError(w, patchErr)
	}

	updatedUser, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		return respondUserServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, toUserResponse(updatedUser))
}

// SetUserPassword задаёт пароль без текущего; все сессии пользователя отзываются.
func (h *UserHandler) SetUserPassword(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return respondError(w, http.StatusBadRequest, ErrCodeInvalidID, ErrMessageInvalidUserID)
	}

	var req SetPasswordRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}

	if err = h.authService.AdminSetPassword(r.Context(), id, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return handleNotFoundError(w, entityUser)
		case errors.Is(err, auth.ErrInvalidPassword):
			return respondPasswordPolicyError(w)
		default:
			return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
		}
	}

	return noContent(w)
}
