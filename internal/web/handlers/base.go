package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/middleware"
)

const (
	// HTMXRequestHeader is the header value for HTMX requests
	HTMXRequestHeader = "true"
	// Flash message cookie names
	flashCookieName     = "flash_message"
	flashTypeCookieName = "flash_type"
	// Flash message cookie expiration in seconds
	flashCookieMaxAge = 10 // 10 seconds - enough time for redirect

	// formValidationMessage — общее сообщение над формой, вернувшейся с
	// ошибками валидации.
	formValidationMessage = "Проверьте правильность заполнения формы"
)

// Заголовки страниц. Интерфейс русскоязычный, английские заголовки — находка
// U-05 (docs/specs/003-ui-ux-audit.md#u-05). Заголовок вида «Редактирование …»
// работает ещё и признаком шаблона в renderXxxFormWithErrors, поэтому все
// заголовки живут константами, а не литералами по месту.
const (
	titleLogin           = "Вход"
	titleInvite          = "Регистрация по приглашению"
	titleDashboard       = "Главная"
	titleTransactions    = "Транзакции"
	titleNewTransaction  = "Новая транзакция"
	titleEditTransaction = "Редактирование транзакции"

	titleCategories   = "Категории"
	titleNewCategory  = "Новая категория"
	titleEditCategory = "Редактирование категории"

	titleBudgets      = "Бюджеты"
	titleNewBudget    = "Новый бюджет"
	titleEditBudget   = "Редактирование бюджета"
	titleBudgetAlerts = "Оповещения по бюджетам"
	titleBudgetPrefix = "Бюджет: "

	titleReports      = "Отчёты"
	titleNewReport    = "Новый отчёт"
	titleReportPrefix = "Отчёт: "
)

var (
	// ErrNoSession is returned when no session is found
	ErrNoSession = errors.New("no session found")

	// cookieSecureForProduction is set by SetCookieSecureForProduction in
	// production-like environments, so flash cookies are emitted with the
	// Secure flag. Defaults to false for local development over plain HTTP.
	cookieSecureForProduction = false //nolint:gochecknoglobals // env flag, set once at startup
)

// SetCookieSecureForProduction toggles the Secure flag for flash cookies.
// Call once at startup from web.NewWebServer; safe to leave unset in dev.
func SetCookieSecureForProduction(secure bool) {
	cookieSecureForProduction = secure
}

// BaseHandler содержит общие методы для всех веб-обработчиков
type BaseHandler struct {
	repositories *handlers.Repositories
	services     *services.Services
}

// NewBaseHandler создает новый базовый обработчик
func NewBaseHandler(repositories *handlers.Repositories, services *services.Services) *BaseHandler {
	return &BaseHandler{
		repositories: repositories,
		services:     services,
	}
}

// SessionData содержит данные пользовательской сессии
type SessionData struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      user.Role `json:"role"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PageData содержит общие данные для всех страниц
type PageData struct {
	Title       string            `json:"title"`
	CurrentUser *SessionData      `json:"current_user,omitempty"`
	Family      *FamilyInfo       `json:"family,omitempty"`
	Errors      map[string]string `json:"errors,omitempty"`
	Messages    []Message         `json:"messages,omitempty"`
	CSRFToken   string            `json:"csrf_token,omitempty"`
}

// FamilyInfo содержит базовую информацию о семье
type FamilyInfo struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Currency string    `json:"currency"`
}

// Message содержит flash сообщение
type Message struct {
	Type string `json:"type"` // "success", "error", "warning", "info"
	Text string `json:"text"`
}

// getFlashMessages получает flash сообщения из сессии
func (h *BaseHandler) getFlashMessages(c echo.Context) []Message {
	msgType, message := GetFlashMessage(c)
	if message == "" {
		return []Message{}
	}
	return []Message{
		{
			Type: msgType,
			Text: message,
		},
	}
}

// buildPageData собирает общие данные страницы: заголовок, flash-сообщения,
// CSRF-токен и текущего пользователя.
//
// Имя и фамилия в сессии не хранятся (см. middleware.SessionData), поэтому они
// дочитываются через UserService — как это делает DashboardHandler.
// Если сессии нет или пользователь не читается, CurrentUser остаётся nil:
// шаблоны шапки завязаны на `{{if .CurrentUser}}` и просто не покажут меню.
func (h *BaseHandler) buildPageData(c echo.Context, title string) *PageData {
	pageData := &PageData{
		Title:    title,
		Messages: h.getFlashMessages(c),
	}

	if csrfToken, err := middleware.GetCSRFToken(c); err == nil {
		pageData.CSRFToken = csrfToken
	}

	currentUser := h.sessionPageUser(c)
	if currentUser == nil {
		return pageData
	}

	currentUserRecord, userErr := h.services.User.GetUserByID(c.Request().Context(), currentUser.UserID)
	if userErr == nil && currentUserRecord != nil {
		currentUser.FirstName = currentUserRecord.FirstName
		currentUser.LastName = currentUserRecord.LastName
	}
	pageData.CurrentUser = currentUser

	return pageData
}

// sessionPageUser собирает CurrentUser для шапки страницы из одной только
// сессии, без обращения к БД: имени и фамилии там нет, поэтому шаблон шапки
// подписывает меню email'ом (components/nav.html). Нужен там, где страница
// перерисовывается после ошибки валидации и лишний запрос неоправдан.
func (h *BaseHandler) sessionPageUser(c echo.Context) *SessionData {
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return nil
	}

	return &SessionData{
		UserID: sessionData.UserID,
		Role:   sessionData.Role,
		Email:  sessionData.Email,
	}
}

// formPageData — buildPageData для формы, которую перерисовывают после
// неудачной валидации: к общим данным страницы добавляются ошибки полей и
// одно общее сообщение.
func (h *BaseHandler) formPageData(c echo.Context, title string, errors map[string]string) *PageData {
	pageData := h.buildPageData(c, title)
	pageData.Errors = errors
	pageData.Messages = []Message{
		{Type: "error", Text: formValidationMessage},
	}

	return pageData
}

// renderPage рендерит полную страницу
func (h *BaseHandler) renderPage(c echo.Context, templateName string, data any) error {
	return c.Render(http.StatusOK, templateName, data)
}

// renderPartial рендерит частичный шаблон (для HTMX)
func (h *BaseHandler) renderPartial(c echo.Context, templateName string, data any) error {
	return c.Render(http.StatusOK, templateName, data)
}

// redirect выполняет редирект
func (h *BaseHandler) redirect(c echo.Context, url string) error {
	if h.IsHTMXRequest(c) {
		c.Response().Header().Set("Hx-Redirect", url)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, url)
}

// IsHTMXRequest проверяет, является ли запрос HTMX запросом
func (h *BaseHandler) IsHTMXRequest(c echo.Context) bool {
	return c.Request().Header.Get("Hx-Request") == HTMXRequestHeader
}

// DeleteEntityParams содержит параметры для общего метода удаления
type DeleteEntityParams struct {
	EntityName       string                                                  // Название сущности для сообщений об ошибках
	IDParamName      string                                                  // Имя параметра ID в URL (по умолчанию "id")
	GetEntityFunc    func(ctx echo.Context, entityID uuid.UUID) (any, error) // Функция получения сущности
	DeleteEntityFunc func(ctx echo.Context, entityID uuid.UUID) error        // Функция удаления сущности
	GetErrorMsgFunc  func(err error) string                                  // Функция получения сообщения об ошибке
	RedirectURL      string                                                  // URL для редиректа после успешного удаления
}

// handleDelete общий метод для удаления сущностей
func (h *BaseHandler) handleDelete(c echo.Context, params DeleteEntityParams) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID
	paramName := params.IDParamName
	if paramName == "" {
		paramName = "id"
	}
	id := c.Param(paramName)
	entityID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid "+params.EntityName+" ID")
	}

	// Получаем сущность для проверки существования
	_, err = params.GetEntityFunc(c, entityID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, params.EntityName+" not found")
	}

	// Single family model - no family access check needed

	// Удаляем сущность
	err = params.DeleteEntityFunc(c, entityID)
	if err != nil {
		errorMsg := params.GetErrorMsgFunc(err)

		if h.IsHTMXRequest(c) {
			return h.renderPartial(c, "components/alert", map[string]any{
				"Type":    "error",
				"Message": errorMsg,
			})
		}

		return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
	}

	if h.IsHTMXRequest(c) {
		// Для HTMX возвращаем пустой ответ для удаления строки
		return c.NoContent(http.StatusOK)
	}

	return h.redirect(c, params.RedirectURL)
}

// htmxError returns an HTMX-compatible error response
func (h *BaseHandler) htmxError(c echo.Context, message string) error {
	return c.String(http.StatusBadRequest, message)
}

// setFlashMessage sets a flash message in a cookie
func setFlashMessage(c echo.Context, msgType, message string) {
	// #nosec G124 -- Secure flag is configured at startup via SetCookieSecureForProduction;
	// HttpOnly and SameSite are always set.
	c.SetCookie(&http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   flashCookieMaxAge,
		HttpOnly: true,
		Secure:   cookieSecureForProduction,
		SameSite: http.SameSiteStrictMode,
	})
	// #nosec G124 -- see comment above.
	c.SetCookie(&http.Cookie{
		Name:     flashTypeCookieName,
		Value:    msgType,
		Path:     "/",
		MaxAge:   flashCookieMaxAge,
		HttpOnly: true,
		Secure:   cookieSecureForProduction,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetFlashMessage reads and clears the flash message
func GetFlashMessage(c echo.Context) (string, string) {
	msgCookie, err := c.Cookie(flashCookieName)
	if err != nil {
		return "", ""
	}
	typeCookie, _ := c.Cookie(flashTypeCookieName)

	// Clear cookies
	// #nosec G124 -- Secure flag is configured at startup via SetCookieSecureForProduction.
	c.SetCookie(&http.Cookie{
		Name:     flashCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecureForProduction,
		SameSite: http.SameSiteStrictMode,
	})
	// #nosec G124 -- see comment above.
	c.SetCookie(&http.Cookie{
		Name:     flashTypeCookieName,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecureForProduction,
		SameSite: http.SameSiteStrictMode,
	})

	message, _ := url.QueryUnescape(msgCookie.Value)
	msgType := "info"
	if typeCookie != nil {
		msgType = typeCookie.Value
	}

	return msgType, message
}

// redirectWithError performs a redirect with an error message
func (h *BaseHandler) redirectWithError(c echo.Context, redirectURL, message string) error {
	setFlashMessage(c, "error", message)
	if h.IsHTMXRequest(c) {
		c.Response().Header().Set("Hx-Redirect", redirectURL)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, redirectURL)
}

// redirectWithSuccess performs a redirect with a success message
func (h *BaseHandler) redirectWithSuccess(c echo.Context, redirectURL, message string) error {
	setFlashMessage(c, "success", message)
	if h.IsHTMXRequest(c) {
		c.Response().Header().Set("Hx-Redirect", redirectURL)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, redirectURL)
}
