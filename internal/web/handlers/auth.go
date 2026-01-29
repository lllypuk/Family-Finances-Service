package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/infrastructure/validation"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	"family-budget-service/internal/web/models"
)

// AuthHandler обрабатывает запросы аутентификации
type AuthHandler struct {
	repos    *handlers.Repositories
	services *services.Services
}

// NewAuthHandler создает новый обработчик аутентификации
func NewAuthHandler(repos *handlers.Repositories, services *services.Services) *AuthHandler {
	return &AuthHandler{
		repos:    repos,
		services: services,
	}
}

// LoginPage отображает страницу входа
func (h *AuthHandler) LoginPage(c echo.Context) error {
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]any{
		"CSRFToken": csrfToken,
		"Title":     "Sign In",
		"IsLogin":   true,
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

	// Дополнительная валидация email на уровне репозитория для предотвращения инъекций
	if err := validation.ValidateEmail(form.Email); err != nil {
		return h.loginError(c, "Invalid email format", map[string]string{
			"email": "Please enter a valid email address",
		})
	}

	// Поиск пользователя по email
	foundUser, err := h.repos.User.GetByEmail(c.Request().Context(), form.Email)
	if err != nil {
		return h.loginError(c, "Invalid email or password", nil)
	}

	// Проверка пароля
	if passwordErr := bcrypt.CompareHashAndPassword(
		[]byte(foundUser.Password),
		[]byte(form.Password),
	); passwordErr != nil {
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
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create session")
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

// SetupPage отображает страницу первоначальной настройки
func (h *AuthHandler) SetupPage(c echo.Context) error {
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return err
	}

	data := map[string]any{
		"CSRFToken": csrfToken,
		"Title":     "Первоначальная настройка",
		"IsSetup":   true,
	}

	return c.Render(http.StatusOK, "setup.html", data)
}

// Setup обрабатывает первоначальную настройку семьи
func (h *AuthHandler) Setup(c echo.Context) error {
	var form models.SetupForm
	if err := c.Bind(&form); err != nil {
		return h.setupError(c, "Invalid form data", nil)
	}

	// Валидация
	if err := c.Validate(&form); err != nil {
		return h.setupError(c, "Please check your input", models.GetValidationErrors(err))
	}

	// Дополнительная валидация email
	if err := validation.ValidateEmail(form.Email); err != nil {
		return h.setupError(c, "Invalid email format", map[string]string{
			"email": "Please enter a valid email address",
		})
	}

	// Вызываем сервис для создания семьи и первого пользователя
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: form.FamilyName,
		Currency:   form.Currency,
		Email:      form.Email,
		FirstName:  form.FirstName,
		LastName:   form.LastName,
		Password:   form.Password,
	}

	_, err := h.services.Family.SetupFamily(c.Request().Context(), setupDTO)
	if err != nil {
		return h.setupError(c, "Failed to create family: "+err.Error(), nil)
	}

	// Если это HTMX запрос
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/login")
}

// Logout обрабатывает выход из системы
func (h *AuthHandler) Logout(c echo.Context) error {
	if err := middleware.ClearSession(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to logout")
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

	data := map[string]any{
		"CSRFToken":   csrfToken,
		"Title":       "Sign In",
		"Error":       message,
		"FieldErrors": fieldErrors,
		"Email":       c.FormValue("email"), // Сохраняем введенный email
		"IsLogin":     true,
	}

	// Если это HTMX запрос, возвращаем только форму
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return c.Render(http.StatusUnprocessableEntity, "login_form.html", data)
	}

	return c.Render(http.StatusUnprocessableEntity, "login.html", data)
}

// setupError возвращает ошибку настройки
func (h *AuthHandler) setupError(c echo.Context, message string, fieldErrors map[string]string) error {
	csrfToken, _ := middleware.GetCSRFToken(c)

	data := map[string]any{
		"CSRFToken":   csrfToken,
		"Title":       "Первоначальная настройка",
		"Error":       message,
		"FieldErrors": fieldErrors,
		"FirstName":   c.FormValue("first_name"),
		"LastName":    c.FormValue("last_name"),
		"Email":       c.FormValue("email"),
		"FamilyName":  c.FormValue("family_name"),
		"Currency":    c.FormValue("currency"),
		"IsSetup":     true,
	}

	// Если это HTMX запрос, возвращаем только форму
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return c.Render(http.StatusUnprocessableEntity, "setup_form.html", data)
	}

	return c.Render(http.StatusUnprocessableEntity, "setup.html", data)
}
