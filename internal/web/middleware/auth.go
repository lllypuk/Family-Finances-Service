package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

const HTMXRequestValue = "true"

// sessionRevalidationErrorMessage — то, что видит клиент, когда перепроверку
// сессии не удалось выполнить. Детали ошибки уходят только в лог.
const sessionRevalidationErrorMessage = "Service temporarily unavailable"

// Аналоги этих middleware для группы /api/v1 (401/403 в виде JSON, без
// редиректа на /login) живут в internal/application/handlers/api_auth.go:
// там же лежит общий формат ошибок API, дублировать который здесь незачем.

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
				return redirectToLogin(c)
			}

			// Сохраняем данные пользователя в контексте
			c.Set("user", sessionData)
			return next(c)
		}
	}
}

// redirectToLogin отправляет неаутентифицированного клиента на страницу входа.
// HTMX-запросу редирект нужен заголовком, иначе он подставит страницу входа
// внутрь фрагмента.
func redirectToLogin(c echo.Context) error {
	if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
		c.Response().Header().Set("Hx-Redirect", "/login")
		return c.NoContent(http.StatusUnauthorized)
	}
	return c.Redirect(http.StatusFound, "/login")
}

// SessionUserLookup — минимальный контракт для перепроверки владельца сессии.
// Ровно один метод, чтобы middleware не тянул за собой пакет services.
type SessionUserLookup interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

// ErrSessionUserGone — владельца сессии больше нет (удалён или cookie ни на
// кого не указывает). Единственный случай, когда cookie нужно гасить.
var ErrSessionUserGone = errors.New("session user no longer exists")

// RevalidateSessionUser перечитывает владельца сессии из БД.
//
// Хранилище сессий cookie-based: id и роль приходят из подписанной cookie и до
// сих пор никем не перепроверялись, поэтому удалённый или пониженный в правах
// пользователь сохранял доступ все 24 часа SessionTimeout, и отозвать его было
// нечем. Здесь сессия сверяется с текущим состоянием БД: пользователя нет —
// доступа нет, роль изменилась — дальше по цепочке идёт актуальная роль.
//
// Ошибка ErrSessionUserGone означает «сессия недействительна, cookie гасить»;
// любая другая ошибка — сбой инфраструктуры, и сессию трогать нельзя: пул
// SQLite открыт в одно соединение, так что первый же SQLITE_BUSY или таймаут
// контекста разлогинил бы работающего пользователя навсегда.
func RevalidateSessionUser(c echo.Context, lookup SessionUserLookup) (*SessionData, error) {
	sessionData, err := GetUserFromContext(c)
	if err != nil {
		return nil, ErrSessionUserGone
	}

	record, lookupErr := lookup.GetUserByID(c.Request().Context(), sessionData.UserID)
	if lookupErr != nil {
		if errors.Is(lookupErr, user.ErrNotFound) {
			return nil, ErrSessionUserGone
		}
		return nil, fmt.Errorf("revalidate session user: %w", lookupErr)
	}
	if record == nil {
		return nil, ErrSessionUserGone
	}

	fresh := *sessionData
	fresh.Role = record.Role
	fresh.Email = record.Email

	return &fresh, nil
}

// RequireActiveUser middleware перепроверяет владельца сессии в БД.
// Ставится сразу после RequireAuth: тот кладёт SessionData в контекст, а этот
// заменяет её на актуальную либо разлогинивает.
func RequireActiveUser(lookup SessionUserLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fresh, err := RevalidateSessionUser(c, lookup)
			if err != nil {
				if errors.Is(err, ErrSessionUserGone) {
					// Cookie больше ничего не значит — гасим её, иначе браузер
					// будет слать её до истечения SessionTimeout.
					_ = ClearSession(c)
					return redirectToLogin(c)
				}

				// БД недоступна — это ошибка сервера, а не отзыв доступа:
				// сессию оставляем как есть, пользователь повторит запрос.
				c.Logger().Errorf("session revalidation failed on %s %s: %v",
					c.Request().Method, c.Request().URL.Path, err)

				return echo.NewHTTPError(http.StatusInternalServerError, sessionRevalidationErrorMessage)
			}

			c.Set("user", fresh)
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
