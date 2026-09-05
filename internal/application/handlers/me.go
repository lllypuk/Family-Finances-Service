package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

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
func (h *MeHandler) GetMe(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	u, err := h.userService.GetUserByID(c.Request().Context(), principal.UserID)
	if err != nil {
		return respondUserServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toUserResponse(u))
}

// UpdateMe меняет имя и email; роль через этот роут не меняется.
func (h *MeHandler) UpdateMe(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	var req UpdateUserRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, bindErr.Error()))
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}
	if req.FirstName == nil && req.LastName == nil && req.Email == nil {
		return respondError(c, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			bodyDetail(ErrCodeValidationError, "at least one field is required"))
	}

	updated, err := h.userService.UpdateUser(c.Request().Context(), principal.UserID, dto.UpdateUserDTO{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})
	if err != nil {
		return respondUserServiceError(c, err)
	}

	return respondAPI(c, http.StatusOK, toUserResponse(updated))
}

// ChangePassword — по текущему паролю; отзывает все сессии, кроме той, чьим токеном сделан запрос.
func (h *MeHandler) ChangePassword(c echo.Context) error {
	principal, err := auth.FromContext(c)
	if err != nil {
		return respondUnauthorized(c)
	}

	var req ChangePasswordRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return HandleBindError(c)
	}
	if validationErr := h.validator.Struct(&req); validationErr != nil {
		return respondValidationErrors(c, validationErr)
	}

	err = h.authService.ChangePassword(
		c.Request().Context(), principal.UserID, req.CurrentPassword, req.NewPassword, principal.SessionID,
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			return respondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, ErrMessageInvalidCredentials)
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
