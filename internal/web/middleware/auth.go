package middleware

import (
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

const HTMXRequestValue = "true"

// Формат ошибки для API повторяет ErrorResponse/ErrorDetail/ResponseMeta из
// internal/application/handlers (types.go, helpers.go, errors.go). Структуры
// продублированы намеренно: middleware тянет только echo и domain/user, и
// импорт пакета хендлеров развернул бы направление зависимостей
// (web → application) ради трёх полей.
const (
	// apiErrCodeUnauthorized — код ошибки для запроса без валидной сессии.
	apiErrCodeUnauthorized = "UNAUTHORIZED"
	// apiErrMessageUnauthorized — сообщение, парное к apiErrCodeUnauthorized.
	apiErrMessageUnauthorized = "Authentication required"
	// apiErrCodeForbidden — код ошибки для сессии с недостаточной ролью.
	apiErrCodeForbidden = "FORBIDDEN"
	// apiErrMessageForbidden — сообщение, парное к apiErrCodeForbidden.
	apiErrMessageForbidden = "Insufficient permissions"
	// apiVersion совпадает с handlers.apiVersion — версия API в meta ответа.
	apiVersion = "v1"
)

// apiErrorDetail — тело поля error в ответе API.
type apiErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiErrorMeta — тело поля meta в ответе API.
type apiErrorMeta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// apiErrorResponse — JSON-ошибка API в том же виде, что отдают хендлеры.
type apiErrorResponse struct {
	Error apiErrorDetail `json:"error"`
	Meta  apiErrorMeta   `json:"meta"`
}

// RequireAuth middleware проверяет, что пользователь аутентифицирован
func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Поддержка моков для тестирования
			if mockData := c.Get("mock_session_data"); mockData != nil {
				c.Set("user", mockData)
				return next(c)
			}
			if mockError := c.Get("mock_session_error"); mockError != nil {
				if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
					c.Response().Header().Set("Hx-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				_ = c.Redirect(http.StatusFound, "/login")
				return echo.NewHTTPError(http.StatusFound, "redirect to login")
			}

			// Проверяем аутентификацию
			sessionData, err := GetSessionData(c)
			if err != nil {
				// Если это HTMX запрос, возвращаем специальный заголовок
				if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
					c.Response().Header().Set("Hx-Redirect", "/login")
					return c.NoContent(http.StatusUnauthorized)
				}
				// Для обычных запросов - редирект на страницу входа
				return c.Redirect(http.StatusFound, "/login")
			}

			// Сохраняем данные пользователя в контексте
			c.Set("user", sessionData)
			return next(c)
		}
	}
}

// RequireAPIAuth middleware — аналог RequireAuth для группы /api/v1.
// Отличие принципиальное: программному клиенту не нужен редирект на /login,
// поэтому при отсутствии валидной сессии возвращается 401 и JSON-ошибка в том
// же формате, что отдают API-хендлеры. При валидной сессии SessionData кладётся
// в контекст под ключом "user" — так же, как это делает RequireAuth,
// поэтому GetUserFromContext работает и в API-хендлерах.
func RequireAPIAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Поддержка моков для тестирования (как в RequireAuth)
			if mockData := c.Get("mock_session_data"); mockData != nil {
				c.Set("user", mockData)
				return next(c)
			}
			if mockError := c.Get("mock_session_error"); mockError != nil {
				return respondAPIUnauthorized(c)
			}

			sessionData, err := GetSessionData(c)
			if err != nil {
				return respondAPIUnauthorized(c)
			}

			c.Set("user", sessionData)
			return next(c)
		}
	}
}

// RequireAPIRole middleware — аналог RequireRole для группы /api/v1.
// Веб-вариант отдаёт HTML «Access denied», программному клиенту нужен JSON,
// поэтому здесь: нет сессии — 401, роль не подходит — 403, оба раза телом
// служит apiErrorResponse. Ставится после RequireAPIAuth, который кладёт
// SessionData в контекст.
func RequireAPIRole(roles ...user.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionData, err := GetUserFromContext(c)
			if err != nil {
				return respondAPIUnauthorized(c)
			}

			if !hasRequiredRole(sessionData.Role, roles) {
				return respondAPIForbidden(c)
			}

			return next(c)
		}
	}
}

// RequireAPIAdmin middleware — API-ярлык для админов (аналог RequireAdmin).
func RequireAPIAdmin() echo.MiddlewareFunc {
	return RequireAPIRole(user.RoleAdmin)
}

// RequireAPIAdminOrMember middleware — API-аналог RequireAdminOrMember:
// финансовые разделы, закрытые в вебе от роли child.
func RequireAPIAdminOrMember() echo.MiddlewareFunc {
	return RequireAPIRole(user.RoleAdmin, user.RoleMember)
}

// respondAPIUnauthorized отдаёт 401 с JSON-телом в формате API-хендлеров.
func respondAPIUnauthorized(c echo.Context) error {
	return respondAPIError(c, http.StatusUnauthorized, apiErrCodeUnauthorized, apiErrMessageUnauthorized)
}

// respondAPIForbidden отдаёт 403 с JSON-телом в формате API-хендлеров.
func respondAPIForbidden(c echo.Context) error {
	return respondAPIError(c, http.StatusForbidden, apiErrCodeForbidden, apiErrMessageForbidden)
}

// respondAPIError собирает JSON-ошибку API.
func respondAPIError(c echo.Context, status int, code, message string) error {
	return c.JSON(status, apiErrorResponse{
		Error: apiErrorDetail{
			Code:    code,
			Message: message,
		},
		Meta: apiErrorMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   apiVersion,
		},
	})
}

// RequireRole middleware проверяет, что пользователь имеет нужную роль
func RequireRole(roles ...user.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userData, err := validateUserAccess(c)
			if err != nil {
				return err
			}

			if !hasRequiredRole(userData.Role, roles) {
				return handleForbiddenAccess(c)
			}

			return next(c)
		}
	}
}

// RequireAdmin middleware - ярлык для админов
func RequireAdmin() echo.MiddlewareFunc {
	return RequireRole(user.RoleAdmin)
}

// RequireAdminOrMember middleware - для админов и обычных членов семьи
func RequireAdminOrMember() echo.MiddlewareFunc {
	return RequireRole(user.RoleAdmin, user.RoleMember)
}

// GetUserFromContext извлекает данные пользователя из контекста
func GetUserFromContext(c echo.Context) (*SessionData, error) {
	userData, ok := c.Get("user").(*SessionData)
	if !ok {
		return nil, echo.ErrUnauthorized
	}
	return userData, nil
}

// RedirectIfAuthenticated middleware перенаправляет аутентифицированных пользователей
// Полезно для страниц входа/регистрации
func RedirectIfAuthenticated(redirectTo string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if IsAuthenticated(c) {
				return c.Redirect(http.StatusFound, redirectTo)
			}
			return next(c)
		}
	}
}

// validateUserAccess проверяет доступ пользователя
func validateUserAccess(c echo.Context) (*SessionData, error) {
	userData, ok := c.Get("user").(*SessionData)
	if !ok {
		if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
			c.Response().Header().Set("Hx-Redirect", "/login")
			_ = c.NoContent(http.StatusUnauthorized)
			return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		_ = c.Redirect(http.StatusFound, "/login")
		return nil, echo.NewHTTPError(http.StatusFound, "redirect to login")
	}
	return userData, nil
}

// hasRequiredRole проверяет наличие требуемой роли
func hasRequiredRole(userRole user.Role, requiredRoles []user.Role) bool {
	return slices.Contains(requiredRoles, userRole)
}

// handleForbiddenAccess обрабатывает случай недостатка прав
func handleForbiddenAccess(c echo.Context) error {
	if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Insufficient permissions",
		})
	}
	return c.String(http.StatusForbidden, "Access denied")
}
