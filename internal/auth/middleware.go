package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"family-budget-service/internal/application/router"
	"family-budget-service/internal/domain/user"
)

const bearerScheme = "bearer"

// Authenticator проверяет bearer-токен; реализация — *Service.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

// bearerToken — токен из Authorization: Bearer <token>; false, если заголовок не bearer или пуст.
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

type principalKey struct{}

// WithPrincipal кладёт владельца токена в контекст запроса.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFromContext — владелец токена, положенный RequireBearer; без него ErrUnauthorized.
func PrincipalFromContext(ctx context.Context) (*Principal, error) {
	p, ok := ctx.Value(principalKey{}).(*Principal)
	if !ok || p == nil {
		return nil, ErrUnauthorized
	}

	return p, nil
}

// RequireBearer — 401 без валидного токена, иначе *Principal в контексте; Err у 401 — ErrUnauthorized, у сбоя хранилища — 500.
func RequireBearer(a Authenticator) router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			token, ok := bearerToken(r)
			if !ok {
				return &router.HTTPError{Status: http.StatusUnauthorized, Err: ErrUnauthorized}
			}
			p, err := a.Authenticate(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrUnauthorized) {
					return &router.HTTPError{Status: http.StatusUnauthorized, Err: err}
				}

				return &router.HTTPError{
					Status: http.StatusInternalServerError,
					Err:    fmt.Errorf("authenticate request: %w", err),
				}
			}

			return next(w, r.WithContext(WithPrincipal(r.Context(), p)))
		}
	}
}

// RequireRole — 403, если роль владельца токена не входит в roles; ставится после RequireBearer.
func RequireRole(roles ...user.Role) router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			p, err := PrincipalFromContext(r.Context())
			if err != nil {
				return &router.HTTPError{Status: http.StatusUnauthorized, Err: err}
			}
			if !slices.Contains(roles, p.Role) {
				return &router.HTTPError{Status: http.StatusForbidden}
			}

			return next(w, r)
		}
	}
}
