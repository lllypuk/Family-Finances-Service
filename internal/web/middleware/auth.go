package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

const HTMXRequestValue = "true"

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
				return echo.NewHTTPError(http.StatusFound, "redirect to login")
			}

			// Сохраняем данные пользователя в контексте
			c.Set("user", sessionData)
			return next(c)
		}
	}
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
		return nil, echo.NewHTTPError(http.StatusFound, "redirect to login")
	}
	return userData, nil
}

// hasRequiredRole проверяет наличие требуемой роли
func hasRequiredRole(userRole user.Role, requiredRoles []user.Role) bool {
	for _, role := range requiredRoles {
		if userRole == role {
			return true
		}
	}
	return false
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
