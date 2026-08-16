package handlers

import (
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/middleware"
)

// contextUserKey — ключ, под которым middleware веба и API кладут *SessionData
// в контекст Echo; middleware.GetUserFromContext читает его же.
const contextUserKey = "user"

// RequireAPIAuth middleware — аналог middleware.RequireAuth для группы /api/v1.
// Отличие принципиальное: программному клиенту не нужен редирект на /login,
// поэтому при отсутствии валидной сессии возвращается 401 и JSON-ошибка в том
// же формате, что отдают остальные ответы API. При валидной сессии SessionData
// кладётся в контекст под тем же ключом, что и у RequireAuth, поэтому
// middleware.GetUserFromContext работает и в API-хендлерах.
func RequireAPIAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionData, err := middleware.GetSessionData(c)
			if err != nil {
				return respondUnauthorized(c)
			}

			c.Set(contextUserKey, sessionData)
			return next(c)
		}
	}
}

// RequireAPIRole middleware — аналог middleware.RequireRole для группы /api/v1.
// Веб-вариант отдаёт HTML «Access denied», программному клиенту нужен JSON,
// поэтому здесь: нет сессии — 401, роль не подходит — 403, оба раза телом
// служит ErrorResponse. Ставится после RequireAPIAuth, который кладёт
// SessionData в контекст.
func RequireAPIRole(roles ...user.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sessionData, err := middleware.GetUserFromContext(c)
			if err != nil {
				return respondUnauthorized(c)
			}

			if !slices.Contains(roles, sessionData.Role) {
				return respondError(c, http.StatusForbidden, ErrCodeForbidden, ErrMessageForbidden)
			}

			return next(c)
		}
	}
}
