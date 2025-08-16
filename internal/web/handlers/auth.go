package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/handlers"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	repos *handlers.Repositories
}

// NewAuthHandler создает новый обработчик аутентификации
func NewAuthHandler(repos *handlers.Repositories) *AuthHandler {
	return &AuthHandler{
		repos: repos,
	}
}

// LoginPage отображает страницу входа
func (h *AuthHandler) LoginPage(c echo.Context) error {
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"CSRFToken": csrfToken,
		"Title":     "Sign In",
	}

	return c.Render(http.StatusOK, "login.html", data)
}

// Login обрабатывает вход в систему
func (h *AuthHandler) Login(c echo.Context) error {
	var form models.LoginForm
	if err := c.Bind(&form); err != nil {
		return h.loginError(c, "Invalid form data", nil)
	}

	// Валидация
	if err := c.Validate(&form); err != nil {
		return h.loginError(c, "Please check your input", models.GetValidationErrors(err))
	}

	// Поиск пользователя по email
	foundUser, err := h.repos.User.GetByEmail(c.Request().Context(), form.Email)
	if err != nil {
		return h.loginError(c, "Invalid email or password", nil)
	}

	// Проверка пароля
	if passwordErr := bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(form.Password)); passwordErr != nil {
		return h.loginError(c, "Invalid email or password", nil)
	}

	// Создание сессии
	sessionData := &middleware.SessionData{
		UserID:   foundUser.ID,
		FamilyID: foundUser.FamilyID,
		Role:     foundUser.Role,
		Email:    foundUser.Email,
	}

	if sessionErr := middleware.SetSessionData(c, sessionData); sessionErr != nil {
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}

	// Определяем куда перенаправить после входа
	redirectTo := c.QueryParam("redirect")
	if redirectTo == "" {
		redirectTo = "/"
	} else {
		// Replace backslashes with forward slashes to normalize
		redirectTo = strings.ReplaceAll(redirectTo, "\\", "/")
		parsed, parsErr := url.Parse(redirectTo)
		// Only allow local redirects (no host, no scheme)
		if parsErr != nil || parsed.IsAbs() || parsed.Host != "" {
			redirectTo = "/"
		}
	}

	// Если это HTMX запрос, возвращаем redirect header
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", redirectTo)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, redirectTo)
}

// RegisterPage отображает страницу регистрации
func (h *AuthHandler) RegisterPage(c echo.Context) error {
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"CSRFToken": csrfToken,
		"Title":     "Create Family Account",
	}

	return c.Render(http.StatusOK, "register.html", data)
}

// Register обрабатывает регистрацию новой семьи
func (h *AuthHandler) Register(c echo.Context) error {
	var form models.RegisterForm
	if err := c.Bind(&form); err != nil {
		return h.registerError(c, "Invalid form data", nil)
	}

	// Валидация
	if err := c.Validate(&form); err != nil {
		return h.registerError(c, "Please check your input", models.GetValidationErrors(err))
	}

	// Проверяем, что пользователь с таким email не существует
	existingUser, _ := h.repos.User.GetByEmail(c.Request().Context(), form.Email)
	if existingUser != nil {
		return h.registerError(c, "", map[string]string{
			"email": "User with this email already exists",
		})
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to process password")
	}

	// Создаем семью
	family := user.NewFamily(form.FamilyName, form.Currency)
	if familyErr := h.repos.Family.Create(c.Request().Context(), family); familyErr != nil {
		return c.String(http.StatusInternalServerError, "Failed to create family")
	}

	// Создаем пользователя (первый пользователь в семье - всегда админ)
	newUser := user.NewUser(form.Email, form.FirstName, form.LastName, family.ID, user.RoleAdmin)
	newUser.Password = string(hashedPassword)

	if userErr := h.repos.User.Create(c.Request().Context(), newUser); userErr != nil {
		return c.String(http.StatusInternalServerError, "Failed to create user")
	}

	// Создание сессии для нового пользователя
	sessionData := &middleware.SessionData{
		UserID:   newUser.ID,
		FamilyID: newUser.FamilyID,
		Role:     newUser.Role,
		Email:    newUser.Email,
	}

	if sessionErr := middleware.SetSessionData(c, sessionData); sessionErr != nil {
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}

	// Если это HTMX запрос
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", "/")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/")
}

// Logout обрабатывает выход из системы
func (h *AuthHandler) Logout(c echo.Context) error {
	if err := middleware.ClearSession(c); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to logout")
	}

	// Если это HTMX запрос
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/login")
}

// loginError возвращает ошибку входа
func (h *AuthHandler) loginError(c echo.Context, message string, fieldErrors map[string]string) error {
	csrfToken, _ := middleware.GetCSRFToken(c)

	data := map[string]interface{}{
		"CSRFToken":   csrfToken,
		"Title":       "Sign In",
		"Error":       message,
		"FieldErrors": fieldErrors,
		"Email":       c.FormValue("email"), // Сохраняем введенный email
	}

	// Если это HTMX запрос, возвращаем только форму
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return c.Render(http.StatusUnprocessableEntity, "login_form.html", data)
	}

	return c.Render(http.StatusUnprocessableEntity, "login.html", data)
}

// registerError возвращает ошибку регистрации
func (h *AuthHandler) registerError(c echo.Context, message string, fieldErrors map[string]string) error {
	csrfToken, _ := middleware.GetCSRFToken(c)

	data := map[string]interface{}{
		"CSRFToken":   csrfToken,
		"Title":       "Create Family Account",
		"Error":       message,
		"FieldErrors": fieldErrors,
		"FirstName":   c.FormValue("first_name"),
		"LastName":    c.FormValue("last_name"),
		"Email":       c.FormValue("email"),
		"FamilyName":  c.FormValue("family_name"),
		"Currency":    c.FormValue("currency"),
	}

	// Если это HTMX запрос, возвращаем только форму
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return c.Render(http.StatusUnprocessableEntity, "register_form.html", data)
	}

	return c.Render(http.StatusUnprocessableEntity, "register.html", data)
}
