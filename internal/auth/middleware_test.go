package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/router"
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
				req.Header.Set("Authorization", tc.header)
			}
			got, ok := auth.BearerToken(req)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func serve(mw router.Middleware, authorization string) error {
	h := mw(func(http.ResponseWriter, *http.Request) error { return nil })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	return h(httptest.NewRecorder(), req)
}

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var he *router.HTTPError
	require.ErrorAs(t, err, &he)

	return he.Status
}

func TestRequireBearer_Outcomes(t *testing.T) {
	t.Run("valid token puts the principal into the context", func(t *testing.T) {
		a := newStubAuthenticator(user.RoleAdmin)
		var seen *auth.Principal
		h := auth.RequireBearer(a)(func(_ http.ResponseWriter, r *http.Request) error {
			seen, _ = auth.PrincipalFromContext(r.Context())

			return nil
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		require.NoError(t, h(httptest.NewRecorder(), req))
		assert.Equal(t, a.principal, seen)
	})

	t.Run("missing header is 401", func(t *testing.T) {
		err := serve(auth.RequireBearer(newStubAuthenticator(user.RoleAdmin)), "")
		assert.Equal(t, http.StatusUnauthorized, statusOf(t, err))
		require.ErrorIs(t, err, auth.ErrUnauthorized)
	})

	t.Run("unknown token is 401", func(t *testing.T) {
		err := serve(auth.RequireBearer(newStubAuthenticator(user.RoleAdmin)), "Bearer nope")
		assert.Equal(t, http.StatusUnauthorized, statusOf(t, err))
	})

	t.Run("storage failure is 500", func(t *testing.T) {
		a := newStubAuthenticator(user.RoleAdmin)
		a.failErr = errors.New("db down")
		err := serve(auth.RequireBearer(a), "Bearer "+validToken)
		assert.Equal(t, http.StatusInternalServerError, statusOf(t, err))
		assert.NotErrorIs(t, err, auth.ErrUnauthorized)
	})
}

func TestRequireRole_Outcomes(t *testing.T) {
	chain := func(role user.Role) router.Middleware {
		bearer := auth.RequireBearer(newStubAuthenticator(role))
		admin := auth.RequireRole(user.RoleAdmin)

		return func(next router.Handler) router.Handler { return bearer(admin(next)) }
	}

	err := serve(chain(user.RoleAdmin), "Bearer "+validToken)
	require.NoError(t, err)

	err = serve(chain(user.RoleMember), "Bearer "+validToken)
	assert.Equal(t, http.StatusForbidden, statusOf(t, err))

	h := auth.RequireRole(user.RoleAdmin)(func(http.ResponseWriter, *http.Request) error { return nil })
	err = h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusUnauthorized, statusOf(t, err))
}

func TestPrincipalFromContext_Empty(t *testing.T) {
	_, err := auth.PrincipalFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	require.ErrorIs(t, err, auth.ErrUnauthorized)
}
