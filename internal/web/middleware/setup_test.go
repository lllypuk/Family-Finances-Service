package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/web/middleware"
)

// stubSetupChecker — заглушка SetupChecker со счётчиком обращений к «БД».
type stubSetupChecker struct {
	complete bool
	err      error
	calls    int
}

func (s *stubSetupChecker) IsSetupComplete(_ context.Context) (bool, error) {
	s.calls++
	if s.err != nil {
		return false, s.err
	}
	return s.complete, nil
}

// newSetupEcho собирает Echo с теми же маршрутами, что и веб-сервер: статика
// регистрируется шаблоном `/static*`, из-за чего c.Path() возвращает не URL запроса.
func newSetupEcho(checker middleware.SetupChecker) *echo.Echo {
	e := echo.New()
	e.Use(middleware.RequireSetup(checker))

	ok := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	e.GET("/static*", ok)
	e.GET("/health", ok)
	e.GET("/setup", ok)
	e.GET("/login", ok)
	e.GET("/dashboard", ok)

	return e
}

func doSetupRequest(e *echo.Echo, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return rec
}

func TestRequireSetup_IncompleteSetup_ExemptPaths(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		expectedCode int
	}{
		{name: "static css", target: "/static/css/pico.min.css", expectedCode: http.StatusOK},
		{name: "static root", target: "/static", expectedCode: http.StatusOK},
		{name: "health", target: "/health", expectedCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &stubSetupChecker{complete: false}
			rec := doSetupRequest(newSetupEcho(checker), tt.target)

			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Empty(t, rec.Header().Get(echo.HeaderLocation), "исключённый путь не должен редиректиться")
			assert.Zero(t, checker.calls, "для исключённого пути в БД ходить не нужно")
		})
	}
}

func TestRequireSetup_IncompleteSetup_FaviconNotRedirected(t *testing.T) {
	// /favicon.ico не зарегистрирован: c.Path() для него пуст, поэтому проверять
	// исключение надо по URL запроса.
	checker := &stubSetupChecker{complete: false}
	rec := doSetupRequest(newSetupEcho(checker), "/favicon.ico")

	assert.NotEqual(t, http.StatusFound, rec.Code)
	assert.Empty(t, rec.Header().Get(echo.HeaderLocation))
	assert.Zero(t, checker.calls)
}

func TestRequireSetup_IncompleteSetup_RedirectsToSetup(t *testing.T) {
	checker := &stubSetupChecker{complete: false}
	rec := doSetupRequest(newSetupEcho(checker), "/dashboard")

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/setup", rec.Header().Get(echo.HeaderLocation))
	assert.Equal(t, 1, checker.calls)
}

func TestRequireSetup_IncompleteSetup_SetupPageAccessible(t *testing.T) {
	checker := &stubSetupChecker{complete: false}
	rec := doSetupRequest(newSetupEcho(checker), "/setup")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get(echo.HeaderLocation))
}

func TestRequireSetup_CompleteSetup_SetupRedirectsToLogin(t *testing.T) {
	checker := &stubSetupChecker{complete: true}
	rec := doSetupRequest(newSetupEcho(checker), "/setup")

	require.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get(echo.HeaderLocation))
}

func TestRequireSetup_CompleteSetup_OtherPathsPass(t *testing.T) {
	checker := &stubSetupChecker{complete: true}
	rec := doSetupRequest(newSetupEcho(checker), "/dashboard")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get(echo.HeaderLocation))
}

func TestRequireSetup_CheckerError_GracefulDegradation(t *testing.T) {
	checker := &stubSetupChecker{err: errors.New("db is down")}
	rec := doSetupRequest(newSetupEcho(checker), "/dashboard")

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get(echo.HeaderLocation))
}
