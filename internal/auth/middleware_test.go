package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
)

const validToken = "valid-token"

// stubAuthenticator знает один токен; любой другой — ErrUnauthorized, а failErr имитирует сбой БД.
type stubAuthenticator struct {
	principal *auth.Principal
	failErr   error
	seen      []string
}

func (s *stubAuthenticator) Authenticate(_ context.Context, token string) (*auth.Principal, error) {
	s.seen = append(s.seen, token)
	if s.failErr != nil {
		return nil, s.failErr
	}
	if token != validToken {
		return nil, auth.ErrUnauthorized
	}
	return s.principal, nil
}

func newStubAuthenticator(role user.Role) *stubAuthenticator {
	return &stubAuthenticator{principal: &auth.Principal{
		SessionID: uuid.New(),
		UserID:    uuid.New(),
		Email:     "user@example.com",
		Role:      role,
	}}
}

func serve(handler echo.HandlerFunc, authorization string) (*httptest.ResponseRecorder, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	err := handler(e.NewContext(req, rec))
	return rec, err
}

func httpCode(t *testing.T, err error) int {
	t.Helper()
	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	return he.Code
}

func TestBearerToken(t *testing.T) {
	cases := map[string]struct {
		header string
		want   string
		ok     bool
	}{
		"missing":          {header: "", ok: false},
		"basic scheme":     {header: "Basic abc", ok: false},
		"empty token":      {header: "Bearer ", ok: false},
		"bearer":           {header: "Bearer abc", want: "abc", ok: true},
		"lowercase scheme": {header: "bearer abc", want: "abc", ok: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set(echo.HeaderAuthorization, tc.header)
			}
			got, ok := auth.BearerToken(req)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRequireBearer_NoHeader_Returns401WithoutLookup(t *testing.T) {
	a := newStubAuthenticator(user.RoleMember)
	nextCalled := false
	handler := auth.RequireBearer(a)(func(echo.Context) error {
		nextCalled = true
		return nil
	})

	_, err := serve(handler, "")

	assert.Equal(t, http.StatusUnauthorized, httpCode(t, err))
	require.ErrorIs(t, err, auth.ErrUnauthorized)
	assert.False(t, nextCalled)
	assert.Empty(t, a.seen, "без заголовка хранилище не опрашивается")
}

func TestRequireBearer_GarbageToken_Returns401(t *testing.T) {
	a := newStubAuthenticator(user.RoleMember)
	handler := auth.RequireBearer(a)(func(echo.Context) error { return nil })

	_, err := serve(handler, "Bearer not-a-real-token")

	assert.Equal(t, http.StatusUnauthorized, httpCode(t, err))
	assert.Equal(t, []string{"not-a-real-token"}, a.seen)
}

func TestRequireBearer_ValidToken_PutsPrincipalInContext(t *testing.T) {
	a := newStubAuthenticator(user.RoleAdmin)
	var seen *auth.Principal
	handler := auth.RequireBearer(a)(func(c echo.Context) error {
		p, err := auth.FromContext(c)
		if err != nil {
			return err
		}
		seen = p
		return c.NoContent(http.StatusOK)
	})

	rec, err := serve(handler, "Bearer "+validToken)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, a.principal, seen)
}

func TestRequireBearer_StorageFailure_Returns500(t *testing.T) {
	a := newStubAuthenticator(user.RoleAdmin)
	a.failErr = errors.New("database is locked")
	handler := auth.RequireBearer(a)(func(echo.Context) error { return nil })

	_, err := serve(handler, "Bearer "+validToken)

	assert.Equal(t, http.StatusInternalServerError, httpCode(t, err))
	assert.NotErrorIs(t, err, auth.ErrUnauthorized, "сбой БД не должен выглядеть как отзыв токена")
}

func TestFromContext_Empty_ReturnsUnauthorized(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())

	_, err := auth.FromContext(c)

	require.ErrorIs(t, err, auth.ErrUnauthorized)
}

func TestRequireRole(t *testing.T) {
	cases := []struct {
		name     string
		role     user.Role
		required []user.Role
		wantCode int
	}{
		{
			name:     "admin on admin route",
			role:     user.RoleAdmin,
			required: []user.Role{user.RoleAdmin},
			wantCode: http.StatusOK,
		},
		{
			name:     "member on admin route",
			role:     user.RoleMember,
			required: []user.Role{user.RoleAdmin},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "member on finance route",
			role:     user.RoleMember,
			required: []user.Role{user.RoleAdmin, user.RoleMember},
			wantCode: http.StatusOK,
		},
		{
			name:     "child on finance route",
			role:     user.RoleChild,
			required: []user.Role{user.RoleAdmin, user.RoleMember},
			wantCode: http.StatusForbidden,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newStubAuthenticator(tc.role)
			nextCalled := false
			handler := auth.RequireBearer(a)(auth.RequireRole(tc.required...)(func(c echo.Context) error {
				nextCalled = true
				return c.NoContent(http.StatusOK)
			}))

			rec, err := serve(handler, "Bearer "+validToken)

			if tc.wantCode == http.StatusOK {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.True(t, nextCalled)
				return
			}
			assert.Equal(t, tc.wantCode, httpCode(t, err))
			require.ErrorIs(t, err, auth.ErrForbidden)
			assert.False(t, nextCalled)
		})
	}
}

func TestRequireRole_WithoutPrincipal_Returns401(t *testing.T) {
	handler := auth.RequireRole(user.RoleAdmin)(func(echo.Context) error { return nil })

	_, err := serve(handler, "")

	assert.Equal(t, http.StatusUnauthorized, httpCode(t, err))
}
