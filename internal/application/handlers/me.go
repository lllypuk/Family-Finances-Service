package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// MeHandler — профиль владельца токена; доступен любой роли.
type MeHandler struct {
	userService services.UserService
	authService AuthService
	validator   *validator.Validate
}

func NewMeHandler(userService services.UserService, authService AuthService) *MeHandler {
	return &MeHandler{
		userService: userService,
		authService: authService,
		validator:   newAPIValidator(),
	}
}

// GetMe отдаёт текущего пользователя из БД, а не из токена: имя и updated_at в principal не лежат.
func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	u, err := h.userService.GetUserByID(r.Context(), principal.UserID)
	if err != nil {
		return respondUserServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, toUserResponse(u))
}

// UpdateMe меняет имя и email; роль через этот роут не меняется.
func (h *MeHandler) UpdateMe(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	var req UpdateUserRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}
	if req.FirstName == nil && req.LastName == nil && req.Email == nil {
		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, ErrMessageNoFields))
	}

	updated, err := h.userService.UpdateUser(r.Context(), principal.UserID, dto.UpdateUserDTO{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})
	if err != nil {
		return respondUserServiceError(w, err)
	}

	return respondAPI(w, http.StatusOK, toUserResponse(updated))
}

// ChangePassword — по текущему паролю; отзывает все сессии, кроме той, чьим токеном сделан запрос.
func (h *MeHandler) ChangePassword(w http.ResponseWriter, r *http.Request) error {
	principal, err := auth.PrincipalFromContext(r.Context())
	if err != nil {
		return respondUnauthorized(w)
	}

	var req ChangePasswordRequest
	if bindErr := decodeJSON(r, &req); bindErr != nil {
		return respondBindError(w, bindErr)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(w, validationErr)
	}

	err = h.authService.ChangePassword(
		r.Context(), principal.UserID, req.CurrentPassword, req.NewPassword, principal.SessionID,
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return respondError(w, http.StatusUnauthorized, ErrCodeInvalidCredentials, ErrMessageInvalidCredentials)
		case errors.Is(err, auth.ErrInvalidPassword):
			return respondPasswordPolicyError(w)
		default:
			return respondError(w, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
		}
	}

	return noContent(w)
}
