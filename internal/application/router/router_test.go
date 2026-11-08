package router_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/router"
)

func text(s string) router.Handler {
	return func(w http.ResponseWriter, _ *http.Request) error {
		_, err := w.Write([]byte(s))

		return err
	}
}

func gate(name string, log *[]string, deny int) router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			*log = append(*log, name)
			if deny != 0 {
				return &router.HTTPError{Status: deny}
			}

			return next(w, r)
		}
	}
}

func onError(w http.ResponseWriter, _ *http.Request, err error) {
	var he *router.HTTPError
	status := http.StatusInternalServerError
	if errors.As(err, &he) {
		status = he.Status
	}
	if !router.Committed(w) {
		w.WriteHeader(status)
	}
}

type result struct {
	status int
	body   string
	label  string
	log    []string
}

func newStand(deny int) (*router.Router, *[]string) {
	log := &[]string{}
	rt := router.New(onError)
	rt.Register(http.MethodGet, "/health", text("health"))
	api := rt.Group("/api/v1", gate("auth", log, 0))
	api.Register(http.MethodGet, "/transactions", text("list"))
	api.Register(http.MethodGet, "/transactions/:id", text("one"))
	api.Register(http.MethodGet, "/transactions/export", text("export"))
	api.Register(http.MethodPost, "/transactions/bulk", text("bulk"))
	api.Register(http.MethodGet, "/backups/:name/download", func(_ http.ResponseWriter, r *http.Request) error {
		return &router.HTTPError{Status: http.StatusTeapot, Message: r.PathValue("name")}
	})
	users := api.Group("/users", gate("role", log, deny))
	users.Register(http.MethodGet, "", text("users"))
	users.Register(http.MethodPut, "/:id/password", text("pw"))
	api.Group("/stats", gate("stats", log, 0))

	return rt, log
}

func do(rt *router.Router, log *[]string, method, target string) result {
	*log = nil
	req := httptest.NewRequest(method, target, strings.NewReader(""))
	ctx, holder := router.WithLabelHolder(req.Context())
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req.WithContext(ctx))

	return result{status: rec.Code, body: rec.Body.String(), label: holder.Route(), log: *log}
}

func TestRouter_Dispatch_MatchedRoute(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodGet, "/api/v1/transactions")
	assert.Equal(t, "list", res.body)
	assert.Equal(t, "/api/v1/transactions", res.label)
	assert.Equal(t, []string{"auth"}, res.log)

	res = do(rt, log, http.MethodGet, "/api/v1/transactions/abc")
	assert.Equal(t, "one", res.body)
	assert.Equal(t, "/api/v1/transactions/:id", res.label)
}

func TestRouter_Dispatch_StaticBeatsParam(t *testing.T) {
	rt, log := newStand(0)

	assert.Equal(t, "export", do(rt, log, http.MethodGet, "/api/v1/transactions/export").body)
}

func TestRouter_Dispatch_StaticMethodMissFallsToParam(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodPost, "/api/v1/transactions/bulk")
	assert.Equal(t, "bulk", res.body)

	res = do(rt, log, http.MethodGet, "/api/v1/transactions/bulk")
	assert.Equal(t, "one", res.body)
	assert.Equal(t, "/api/v1/transactions/:id", res.label)

	res = do(rt, log, http.MethodPut, "/api/v1/transactions/bulk")
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "/api/v1/*", res.label)
}

func TestRouter_Dispatch_ParamDecodedOnlyWithoutRawPath(t *testing.T) {
	rt := router.New(onError)
	var got string
	rt.Register(http.MethodGet, "/x/:id", func(_ http.ResponseWriter, r *http.Request) error {
		got = r.PathValue("id")

		return nil
	})

	for target, want := range map[string]string{"/x/a%2Fb": "a%2Fb", "/x/%61bc": "%61bc", "/x/a%20b": "a b"} {
		got = ""
		rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
		assert.Equal(t, want, got, target)
	}
}

func TestRouter_Dispatch_EncodedSlashIsOneParam(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodGet, "/api/v1/backups/a%2Fb/download")
	assert.Equal(t, http.StatusTeapot, res.status)
	assert.Equal(t, "/api/v1/backups/:name/download", res.label)
}

func TestRouter_Dispatch_FallbackLabels(t *testing.T) {
	tests := []struct {
		name, method, target, label string
		status                      int
		log                         []string
	}{
		{"UnknownInGroup", http.MethodGet, "/api/v1/nope", "/api/v1/*", 404, []string{"auth"}},
		{"UnknownOutside", http.MethodGet, "/nope", "/*", 404, nil},
		{"WrongMethodRoot", http.MethodPost, "/health", "/*", 404, nil},
		{"HeadHealth", http.MethodHead, "/health", "/*", 404, nil},
		{"WrongMethodSiblingGroup", http.MethodPatch, "/api/v1/transactions", "/api/v1/*", 404, []string{"auth"}},
		{"WrongMethodSubgroupRoot", http.MethodDelete, "/api/v1/users", "/api/v1/*", 404, []string{"auth"}},
		{
			"WrongMethodDeepInSubgroup", http.MethodPost, "/api/v1/users/x/password", "/api/v1/users/*", 403,
			[]string{"auth", "role"},
		},
		{"GroupRootWithoutRoute", http.MethodGet, "/api/v1/stats", "/api/v1/stats", 404, []string{"auth", "stats"}},
		{"TrailingSlash", http.MethodGet, "/api/v1/transactions/", "/api/v1/*", 404, []string{"auth"}},
		{"TrailingSlashHealth", http.MethodGet, "/health/", "/*", 404, nil},
		{"DoubleSlash", http.MethodGet, "//api/v1/transactions", "/*", 404, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deny := 0
			if tt.status == 403 {
				deny = 403
			}
			rt, log := newStand(deny)

			res := do(rt, log, tt.method, tt.target)

			assert.Equal(t, tt.status, res.status)
			assert.Equal(t, tt.label, res.label)
			assert.Equal(t, tt.log, res.log)
		})
	}
}

func TestRouter_Dispatch_GroupFallbackRunsGroupMiddleware(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodGet, "/api/v1/users/x/y/z")

	assert.Equal(t, "/api/v1/users/*", res.label)
	assert.Equal(t, []string{"auth", "role"}, res.log)
}

func TestRouter_Routes_RegistrationOrder(t *testing.T) {
	rt, _ := newStand(0)

	routes := rt.Routes()

	require.GreaterOrEqual(t, len(routes), 3)
	assert.Equal(t, router.Route{Method: http.MethodGet, Path: "/health"}, routes[0])
	assert.Contains(t, routes, router.Route{Method: http.MethodPut, Path: "/api/v1/users/:id/password"})
}

func TestRouter_Register_DuplicatePanics(t *testing.T) {
	rt := router.New(onError)
	rt.Register(http.MethodGet, "/a", text("a"))

	assert.Panics(t, func() { rt.Register(http.MethodGet, "/a", text("b")) })
}

func TestRouter_Committed_ErrorAfterWrite(t *testing.T) {
	rt := router.New(func(w http.ResponseWriter, _ *http.Request, _ error) {
		if router.Committed(w) {
			w.Header().Set("X-Committed", "yes")
		}
	})
	rt.Register(http.MethodGet, "/x", func(w http.ResponseWriter, _ *http.Request) error {
		_, _ = w.Write([]byte("partial"))

		return errors.New("late")
	})
	rec := httptest.NewRecorder()

	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, "yes", rec.Header().Get("X-Committed"))
}

func TestRouter_Dispatch_SpecificityIndependentOfRegistrationOrder(t *testing.T) {
	orders := [][]string{
		{"/a/:x/c", "/a/b/:y", "/a/b/c", "/a/:x/:y"},
		{"/a/:x/:y", "/a/b/c", "/a/b/:y", "/a/:x/c"},
		{"/a/b/:y", "/a/:x/:y", "/a/:x/c", "/a/b/c"},
	}
	want := map[string]string{
		"/a/b/c": "/a/b/c",
		"/a/z/c": "/a/:x/c",
		"/a/b/z": "/a/b/:y",
		"/a/z/z": "/a/:x/:y",
	}
	for _, order := range orders {
		rt := router.New(onError)
		for _, tpl := range order {
			rt.Register(http.MethodGet, tpl, text(tpl))
		}
		log := &[]string{}
		for path, tpl := range want {
			assert.Equal(t, tpl, do(rt, log, http.MethodGet, path).label, "%v %s", order, path)
		}
	}
}

func TestRouter_Dispatch_TerminalParamTakesSlashes(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodGet, "/api/v1/transactions/a/b/c")
	assert.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, "/api/v1/transactions/:id", res.label)
}

func TestRouter_Dispatch_InteriorParamMayBeEmpty(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodPut, "/api/v1/users//password")
	assert.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, "/api/v1/users/:id/password", res.label)

	res = do(rt, log, http.MethodPut, "/api/v1/users/a/b/password")
	assert.Equal(t, "/api/v1/users/*", res.label)
}

func TestRouter_Dispatch_NonLeafTerminalParamDoesNotTakeSlashes(t *testing.T) {
	rt, log := newStand(0)

	res := do(rt, log, http.MethodGet, "/api/v1/users/x/password")
	assert.Equal(t, http.StatusNotFound, res.status)
	assert.Equal(t, "/api/v1/users/*", res.label)
}

func TestRouter_Dispatch_LongerStaticBeatsGreedyParam(t *testing.T) {
	rt := router.New(onError)
	rt.Register(http.MethodPut, "/a/:id", text("short"))
	rt.Register(http.MethodPut, "/a/:id/b/:m", text("long"))

	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/a/1/b/2", nil))
	assert.Equal(t, "long", rec.Body.String())
}
