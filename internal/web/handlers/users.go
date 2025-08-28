package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	userRepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

// UserHandler обрабатывает запросы управления пользователями
type UserHandler struct {
	*BaseHandler
}

// NewUserHandler создает новый обработчик пользователей
func NewUserHandler(repos *handlers.Repositories) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler(repos),
	}
}

// Index отображает список пользователей семьи
func (h *UserHandler) Index(c echo.Context) error {
	session, err := middleware.GetSessionData(c)
	if err != nil {
		return h.redirect(c, "/login")
	}

	// Получаем всех пользователей семьи
	users, err := h.repositories.User.GetByFamilyID(c.Request().Context(), session.FamilyID)
	if err != nil {
		return h.handleError(c, err, "Failed to load users")
	}

	// Получаем семью для отображения названия
	family, err := h.repositories.Family.GetByID(c.Request().Context(), session.FamilyID)
	if err != nil {
		return h.handleError(c, err, "Failed to load family")
	}

	// Получаем текущего пользователя для проверки прав
	currentUser, err := h.repositories.User.GetByID(c.Request().Context(), session.UserID)
	if err != nil {
		return h.handleError(c, err, "Failed to load current user")
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

	// Проверяем права доступа - только админ может добавлять пользователей
	currentUser, err := h.repositories.User.GetByID(c.Request().Context(), session.UserID)
	if err != nil {
		return h.handleError(c, err, "Failed to load current user")
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

	// Проверяем права доступа
	currentUser, userErr := h.repositories.User.GetByID(c.Request().Context(), session.UserID)
	if userErr != nil {
		return h.handleError(c, userErr, "Failed to load current user")
	}

	if currentUser.Role != user.RoleAdmin {
		return c.String(http.StatusForbidden, "Only family admin can add new members")
	}

	var form models.CreateUserForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return h.userError(c, "Invalid form data", nil, &form)
	}

	// Валидация
	if validateErr := c.Validate(&form); validateErr != nil {
		return h.userError(c, "Please check your input", models.GetValidationErrors(validateErr), &form)
	}

	// Валидация email
	if emailErr := userRepo.ValidateEmail(form.Email); emailErr != nil {
		return h.userError(c, "Invalid email format", map[string]string{
			"email": "Please enter a valid email address",
		}, &form)
	}

	// Проверяем, что пользователь с таким email не существует
	existingUser, _ := h.repositories.User.GetByEmail(c.Request().Context(), form.Email)
	if existingUser != nil {
		return h.userError(c, "", map[string]string{
			"email": "User with this email already exists",
		}, &form)
	}

	// Валидация роли
	role := user.Role(form.Role)
	if role != user.RoleAdmin && role != user.RoleMember && role != user.RoleChild {
		return h.userError(c, "", map[string]string{
			"role": "Invalid role selected",
		}, &form)
	}

	// Хешируем пароль
	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		return c.String(http.StatusInternalServerError, "Failed to process password")
	}

	// Создаем пользователя
	newUser := user.NewUser(form.Email, form.FirstName, form.LastName, session.FamilyID, role)
	newUser.Password = string(hashedPassword)

	if createErr := h.repositories.User.Create(c.Request().Context(), newUser); createErr != nil {
		return h.handleError(c, createErr, "Failed to create user")
	}

	// Если это HTMX запрос
	if h.isHTMXRequest(c) {
		c.Response().Header().Set("Hx-Redirect", "/users")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/users")
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
