package handlers

import (
	"errors"
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/middleware"
)

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

			c.Set(middleware.ContextUserKey, sessionData)
			return next(c)
		}
	}
}

// RequireAPIActiveUser middleware — аналог middleware.RequireActiveUser для
// группы /api/v1: владелец сессии перечитывается из БД, удалённый пользователь
// получает 401, а роль для RequireAPIRole берётся актуальная, а не та, что
// лежит в подписанной cookie. Ставится сразу после RequireAPIAuth.
func RequireAPIActiveUser(lookup middleware.SessionUserLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			fresh, err := middleware.RevalidateSessionUser(c, lookup)
			if err != nil {
				if !errors.Is(err, middleware.ErrSessionUserGone) {
					// Сбой БД, а не отзыв доступа: 401 заставил бы клиента
					// выбросить рабочую сессию и перелогиниться.
					// Детали уже записаны в middleware.RevalidateSessionUser.
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
