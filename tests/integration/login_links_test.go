package integration_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// Регрессионный тест на находку U-05 (docs/specs/003-ui-ux-audit.md#u-05):
// страница входа предлагала «Создать семейный аккаунт» по адресу /register,
// которого в роутере нет — открытой регистрации в однофамильной модели не
// существует, новые участники приходят по инвайту (/invite/:token).
//
// Тест не зашивает список «плохих» ссылок, а сверяет каждую локальную ссылку
// страницы с реально зарегистрированными GET-маршрутами Echo: так же поймается
// любая следующая мёртвая ссылка, а не только /register.
//
// Задача 16 плана docs/plans/20260816-deployment-blockers.md.

// localRefRe вытаскивает значения href/src, указывающие внутрь приложения.
var localRefRe = regexp.MustCompile(`(?:href|src)="(/[^"]*)"`)

func TestLoginPage_NoDeadLinks(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	// RequireSetup редиректит на /setup, пока семьи нет; Auth её заводит.
	testServer.Auth(t)

	body := fetchAnonymousPage(t, testServer, "/login")

	assert.NotContains(t, body, `href="/register"`,
		"на странице входа осталась ссылка на несуществующий маршрут /register")

	routes := registeredGETRoutes(testServer.Server.Echo())
	refs := localRefRe.FindAllStringSubmatch(body, -1)
	require.NotEmpty(t, refs, "на странице входа не нашлось ни одной локальной ссылки")

	for _, ref := range refs {
		path := stripQuery(ref[1])
		assert.True(t, matchesAnyRoute(path, routes),
			"страница /login ссылается на %q, но такого GET-маршрута нет", path)
	}
}

// fetchAnonymousPage запрашивает страницу без сессии и возвращает тело ответа.
func fetchAnonymousPage(t *testing.T, ts *testhelpers.TestServer, path string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "GET %s отдал %d, тело: %s", path, rec.Code, rec.Body.String())

	return rec.Body.String()
}

// registeredGETRoutes возвращает шаблоны всех GET-маршрутов, скомпилированные
// в регулярные выражения: `:param` — один сегмент, `*` — любой хвост.
func registeredGETRoutes(e *echo.Echo) []*regexp.Regexp {
	return registeredRoutes(e)[http.MethodGet]
}

// registeredRoutes раскладывает маршруты Echo по HTTP-методам: hx-post и
// hx-delete должны сверяться со своим методом, иначе кнопка, ведущая на
// зарегистрированный GET, считалась бы живой.
func registeredRoutes(e *echo.Echo) map[string][]*regexp.Regexp {
	routes := make(map[string][]*regexp.Regexp)

	for _, route := range e.Routes() {
		pattern := "^"
		for _, segment := range strings.Split(strings.Trim(route.Path, "/"), "/") {
			switch {
			case segment == "":
				continue
			case strings.HasPrefix(segment, ":"):
				pattern += "/[^/]+"
			case strings.HasSuffix(segment, "*"):
				pattern += "/" + regexp.QuoteMeta(strings.TrimSuffix(segment, "*")) + ".*"
			default:
				pattern += "/" + regexp.QuoteMeta(segment)
			}
		}
		pattern += "/?$"

		routes[route.Method] = append(routes[route.Method], regexp.MustCompile(pattern))
	}

	return routes
}

// matchesAnyRoute сообщает, обслуживается ли путь хоть одним GET-маршрутом.
func matchesAnyRoute(path string, routes []*regexp.Regexp) bool {
	for _, route := range routes {
		if route.MatchString(path) {
			return true
		}
	}

	return false
}

// stripQuery отбрасывает query-строку и якорь — маршрутизации они не касаются.
func stripQuery(ref string) string {
	if idx := strings.IndexAny(ref, "?#"); idx >= 0 {
		return ref[:idx]
	}

	return ref
}
