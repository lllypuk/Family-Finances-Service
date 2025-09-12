package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

// UserHandler обрабатывает запросы управления пользователями
type UserHandler struct {
	*BaseHandler
}

// NewUserHandler создает новый обработчик пользователей
func NewUserHandler(repos *handlers.Repositories, services *services.Services) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler(repos, services),
	}
}

// Index отображает список пользователей семьи
func (h *UserHandler) Index(c echo.Context) error {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return h.redirect(c, "/login")
	}

	// Получаем всех пользователей семьи через сервис
	users, err := h.services.User.GetUsersByFamily(c.Request().Context(), session.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load users")
	}

	// Получаем семью для отображения названия через сервис
	family, err := h.services.Family.GetFamilyByID(c.Request().Context(), session.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load family")
	}

	// Получаем текущего пользователя для проверки прав через сервис
	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load current user")
	}

	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]any{
		"Title":       "Family Members",
		"Users":       users,
		"Family":      family,
		"CurrentUser": currentUser,
		"CSRFToken":   csrfToken,
		"CanManage":   currentUser.Role == user.RoleAdmin,
	}

	return c.Render(http.StatusOK, "users/index.html", data)
}

// New отображает форму добавления нового пользователя
func (h *UserHandler) New(c echo.Context) error {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return h.redirect(c, "/login")
	}

	// Проверяем права доступа через сервис - только админ может добавлять пользователей
	currentUser, err := h.services.User.GetUserByID(c.Request().Context(), session.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load current user")
	}

	if currentUser.Role != user.RoleAdmin {
		return c.String(http.StatusForbidden, "Only family admin can add new members")
	}

	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]any{
		"Title":     "Add Family Member",
		"CSRFToken": csrfToken,
		"Roles": []map[string]any{
			{"Value": string(user.RoleAdmin), "Label": "Admin"},
			{"Value": string(user.RoleMember), "Label": "Member"},
			{"Value": string(user.RoleChild), "Label": "Child"},
		},
	}

	return c.Render(http.StatusOK, "users/new.html", data)
}

// Create создает нового пользователя в семье
func (h *UserHandler) Create(c echo.Context) error {
	session, sessionErr := middleware.GetSessionData(c)
	if sessionErr != nil {
		return h.redirect(c, "/login")
	}

	var form models.CreateUserForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return h.userError(c, "Invalid form data", nil, &form)
	}

	// Валидация формы
	if validateErr := c.Validate(&form); validateErr != nil {
		return h.userError(c, "Please check your input", models.GetValidationErrors(validateErr), &form)
	}

	// Конвертируем форму в DTO
	webReq := dto.CreateUserWebRequest{
		Email:     form.Email,
		Password:  form.Password,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Role:      form.Role,
	}
	userDTO := dto.FromCreateUserWebRequest(webReq, session.FamilyID)

	// Вызываем сервис
	createdUser, err := h.services.User.CreateUser(c.Request().Context(), userDTO)
	if err != nil {
		return h.handleServiceError(c, err, &form)
	}

	_ = createdUser // Use the created user if needed for response

	// Если это HTMX запрос
	if h.isHTMXRequest(c) {
		c.Response().Header().Set("Hx-Redirect", "/users")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/users")
}

// handleServiceError handles service errors and converts them to web form errors
func (h *UserHandler) handleServiceError(c echo.Context, err error, form *models.CreateUserForm) error {
	switch {
	case errors.Is(err, services.ErrValidationFailed):
		return h.userError(c, "Please check your input", map[string]string{
			"form": err.Error(),
		}, form)
	case errors.Is(err, services.ErrEmailAlreadyExists):
		return h.userError(c, "", map[string]string{
			"email": "User with this email already exists",
		}, form)
	case errors.Is(err, services.ErrFamilyNotFound):
		return h.userError(c, "Family not found", nil, form)
	case errors.Is(err, services.ErrUnauthorized):
		return c.String(http.StatusForbidden, "Only family admin can add new members")
	case errors.Is(err, services.ErrInvalidRole):
		return h.userError(c, "", map[string]string{
			"role": "Invalid role selected",
		}, form)
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user")
	}
}

// userError возвращает ошибку для форм пользователей
func (h *UserHandler) userError(c echo.Context, message string, fieldErrors map[string]string,
	form *models.CreateUserForm) error {
	csrfToken, _ := middleware.GetCSRFToken(c)

	data := map[string]any{
		"Title":       "Add Family Member",
		"Error":       message,
		"FieldErrors": fieldErrors,
		"CSRFToken":   csrfToken,
		"Roles": []map[string]any{
			{"Value": string(user.RoleAdmin), "Label": "Admin"},
			{"Value": string(user.RoleMember), "Label": "Member"},
			{"Value": string(user.RoleChild), "Label": "Child"},
		},
	}

	// Сохраняем введенные данные при ошибке
	if form != nil {
		data["FirstName"] = form.FirstName
		data["LastName"] = form.LastName
		data["Email"] = form.Email
		data["Role"] = form.Role
	}

	// Если это HTMX запрос, возвращаем только форму
	if h.isHTMXRequest(c) {
		return c.Render(http.StatusUnprocessableEntity, "users/new_form.html", data)
	}

	return c.Render(http.StatusUnprocessableEntity, "users/new.html", data)
}
