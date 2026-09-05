package handlers

import (
	"errors"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/middleware"
)

// RequireAPIAuth — аутентификация группы /api/v1: bearer-токен, а без заголовка Authorization —
// cookie веб-сессии. Оба пути кладут *middleware.SessionData под middleware.ContextUserKey,
// чтобы хендлеры не различали их до удаления веб-слоя (план 03, задача 8).
// Присутствующий, но невалидный bearer — 401 без отката на cookie.
func RequireAPIAuth(authenticator auth.Authenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if _, ok := auth.BearerToken(c.Request()); ok {
				principal, err := auth.AuthenticateRequest(c, authenticator)
				if err != nil {
					if errors.Is(err, auth.ErrUnauthorized) {
						return respondUnauthorized(c)
					}
					c.Logger().Errorf("bearer authentication failed on %s %q: %q",
						c.Request().Method, c.Request().URL.Path, err.Error())
					return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
				}

				c.Set(auth.ContextKey, principal)
				c.Set(middleware.ContextUserKey, &middleware.SessionData{
					UserID: principal.UserID,
					Role:   principal.Role,
					Email:  principal.Email,
				})
				return next(c)
			}

			sessionData, err := middleware.GetSessionData(c)
			if err != nil {
				return respondUnauthorized(c)
			}

			c.Set(middleware.ContextUserKey, sessionData)
			return next(c)
		}
	}
}

// RequireAPIActiveUser перечитывает владельца сессии из БД: удалённый пользователь получает 401,
// а роль для RequireAPIRole берётся актуальная. Ставится сразу после RequireAPIAuth.
func RequireAPIActiveUser(lookup middleware.SessionUserLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fresh, err := middleware.RevalidateSessionUser(c, lookup)
			if err != nil {
				if !errors.Is(err, middleware.ErrSessionUserGone) {
					// Сбой БД, а не отзыв доступа: 401 заставил бы клиента выбросить рабочую сессию.
					return respondError(c, http.StatusInternalServerError,
						ErrCodeInternal, ErrMessageInternal)
				}

				return respondUnauthorized(c)
			}

			c.Set(middleware.ContextUserKey, fresh)
			return next(c)
		}
	}
}

// RequireAPIRole — нет сессии 401, роль не подходит 403; ставится после RequireAPIAuth.
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
