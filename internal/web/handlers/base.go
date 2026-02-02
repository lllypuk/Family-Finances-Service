package handlers

import (
	"errors"
	"net/http"
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
)

var (
	// ErrNoSession is returned when no session is found
	ErrNoSession = errors.New("no session found")
)

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
func (h *BaseHandler) getFlashMessages(_ echo.Context) []Message {
	// Временная заглушка - в реальной реализации будет получать из сессии
	return []Message{}
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

// redirectWithError performs a redirect with an error message
func (h *BaseHandler) redirectWithError(c echo.Context, url, _ string) error {
	// TODO: Add flash message support for error messages
	// For now, just redirect
	return c.Redirect(http.StatusSeeOther, url)
}

// redirectWithSuccess performs a redirect with a success message
func (h *BaseHandler) redirectWithSuccess(c echo.Context, url, _ string) error {
	// TODO: Add flash message support for success messages
	// For now, just redirect
	return c.Redirect(http.StatusSeeOther, url)
}
