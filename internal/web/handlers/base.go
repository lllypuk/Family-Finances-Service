package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/handlers"
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
}

// NewBaseHandler создает новый базовый обработчик
func NewBaseHandler(repositories *handlers.Repositories) *BaseHandler {
	return &BaseHandler{
		repositories: repositories,
	}
}

// Дублируем необходимые типы чтобы избежать циклического импорта

// SessionData содержит данные пользовательской сессии
type SessionData struct {
	UserID    uuid.UUID `json:"user_id"`
	FamilyID  uuid.UUID `json:"family_id"`
	Role      user.Role `json:"role"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PageData содержит общие данные для всех страниц
type PageData struct {
	Title       string       `json:"title"`
	CurrentUser *user.User   `json:"current_user"`
	Family      *user.Family `json:"family"`
	Errors      FormErrors   `json:"errors"`
	Messages    []Message    `json:"messages"`
	CSRFToken   string       `json:"csrf_token"`
}

// Message представляет сообщение для пользователя
type Message struct {
	Type    string `json:"type"` // success, error, warning, info
	Text    string `json:"text"`
	Timeout int    `json:"timeout"` // время отображения в секундах
}

// FormErrors представляет ошибки валидации форм
type FormErrors map[string]string

// renderPage отображает страницу с данными
func (h *BaseHandler) renderPage(c echo.Context, template string, data any) error {
	return c.Render(http.StatusOK, template, data)
}

// renderPartial отображает частичный шаблон (для HTMX)
func (h *BaseHandler) renderPartial(c echo.Context, template string, data any) error {
	return c.Render(http.StatusOK, template, data)
}

// handleError обрабатывает ошибки и отображает страницу ошибки
//
//nolint:unused // Will be used in future authentication and form handlers
func (h *BaseHandler) handleError(c echo.Context, err error, message string) error {
	// Логируем ошибку
	c.Logger().Error(err)

	// Если это HTMX запрос, возвращаем ошибку в заголовке
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Trigger", "error")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": message,
		})
	}

	// Обычная страница ошибки
	pageData := &PageData{
		Title: "Ошибка",
		Messages: []Message{
			{
				Type: "error",
				Text: message,
			},
		},
	}

	return h.renderPage(c, "error", pageData)
}

// redirect выполняет перенаправление
//
//nolint:unused // Will be used in future authentication and form handlers
func (h *BaseHandler) redirect(c echo.Context, url string) error {
	// Если это HTMX запрос, используем HX-Redirect
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", url)
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, url)
}

// getCurrentSession получает текущую сессию пользователя
//
//nolint:unused // Will be used in future authentication implementation
func (h *BaseHandler) getCurrentSession(_ echo.Context) (*SessionData, error) {
	// TODO: Реализовать получение сессии из middleware
	// Пока возвращаем ошибку
	return nil, ErrNoSession
}

// setFlashMessage устанавливает сообщение для отображения после redirect
//
//nolint:unused // Will be used in future authentication and form handlers
func (h *BaseHandler) setFlashMessage(_ echo.Context, _, _ string) {
	// TODO: Реализовать flash messages через сессии
}

// getFlashMessages получает и очищает flash messages
//
//nolint:unused // Will be used in future authentication and form handlers
func (h *BaseHandler) getFlashMessages(_ echo.Context) []Message {
	// TODO: Реализовать получение flash messages
	return []Message{}
}

// validateForm валидирует структуру формы и возвращает ошибки
//
//nolint:unused // Will be used in future form handlers
func (h *BaseHandler) validateForm(c echo.Context, form any) FormErrors {
	errors := make(FormErrors)

	// Привязываем данные формы
	if err := c.Bind(form); err != nil {
		errors["form"] = "Неверный формат данных"
		return errors
	}

	// TODO: Добавить валидацию с помощью validator

	return errors
}

// isHTMXRequest проверяет, является ли запрос HTMX запросом
//
//nolint:unused // Will be used in future HTMX handlers
func (h *BaseHandler) isHTMXRequest(c echo.Context) bool {
	return c.Request().Header.Get("Hx-Request") == HTMXRequestHeader
}
