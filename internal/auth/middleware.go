package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/user"
)

// ContextKey — ключ *Principal в контексте Echo; читается через FromContext.
const ContextKey = "principal"

const bearerScheme = "bearer"

// ErrForbidden — роль владельца токена не допущена к маршруту.
var ErrForbidden = errors.New("forbidden")

// Authenticator проверяет bearer-токен; реализация — *Service.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

// BearerToken — токен из Authorization: Bearer <token>; false, если заголовок не bearer или пуст.
func BearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get(echo.HeaderAuthorization), " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

// AuthenticateRequest проверяет bearer-токен запроса.
// Без заголовка или с неизвестным токеном — ErrUnauthorized; прочие ошибки — сбой хранилища.
func AuthenticateRequest(c echo.Context, a Authenticator) (*Principal, error) {
	token, ok := BearerToken(c.Request())
	if !ok {
		return nil, ErrUnauthorized
	}
	return a.Authenticate(c.Request().Context(), token)
}

// FromContext — владелец токена, положенный RequireBearer; без него ErrUnauthorized.
func FromContext(c echo.Context) (*Principal, error) {
	p, ok := c.Get(ContextKey).(*Principal)
	if !ok || p == nil {
		return nil, ErrUnauthorized
	}
	return p, nil
}

// RequireBearer — 401 без валидного токена, иначе *Principal в контексте.
// Сбой хранилища — 500, а не 401: клиент не должен выбрасывать рабочий токен.
func RequireBearer(a Authenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p, err := AuthenticateRequest(c, a)
			if err != nil {
				if errors.Is(err, ErrUnauthorized) {
					return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(err)
				}
				return echo.NewHTTPError(http.StatusInternalServerError).
					SetInternal(fmt.Errorf("authenticate request: %w", err))
			}
			c.Set(ContextKey, p)
			return next(c)
		}
	}
}

// RequireRole — 403, если роль владельца токена не входит в roles; ставится после RequireBearer.
func RequireRole(roles ...user.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p, err := FromContext(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(err)
			}
			if !slices.Contains(roles, p.Role) {
				return echo.NewHTTPError(http.StatusForbidden).SetInternal(ErrForbidden)
			}
			return next(c)
		}
	}
}
