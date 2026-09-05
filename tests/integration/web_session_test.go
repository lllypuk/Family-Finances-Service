package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Cookie-сессии для веб-маршрутов. Файл живёт до удаления веб-слоя (план 03,
// задача 8): testhelpers выдаёт только bearer-токены, а HTML-страницы и CSRF
// всё ещё держатся на cookie.

// csrfFormTokenRe вытаскивает CSRF-токен из скрытого поля формы входа
// (internal/web/templates/pages/login.html).
var csrfFormTokenRe = regexp.MustCompile(`name="_token"\s+value="([^"]+)"`)

// anonymousSession — сессия без пользователя: cookie, выданная на GET /login,
// и лежащий в ней CSRF-токен.
type anonymousSession struct {
	cookie *http.Cookie
	token  string
}

// newAnonymousSession повторяет шаг аудита S-01: GET /login отдаёт cookie сессии и
// CSRF-токен в форме, при этом пользователь в сессию не записывается.
func newAnonymousSession(t *testing.T, ts *testhelpers.TestServer) *anonymousSession {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "страница входа недоступна")

	cookies := (&http.Response{Header: rec.Header()}).Cookies()
	require.NotEmpty(t, cookies, "GET /login не выдал cookie сессии")

	match := csrfFormTokenRe.FindStringSubmatch(rec.Body.String())
	require.Len(t, match, 2, "CSRF-токен не найден в форме входа")

	return &anonymousSession{cookie: cookies[0], token: match[1]}
}

// webSession — cookie аутентифицированного пользователя и CSRF-токен той же сессии.
type webSession struct {
	cookie *http.Cookie
	token  string
}

// Apply добавляет к запросу cookie сессии и CSRF-токен в заголовке.
func (s *webSession) Apply(req *http.Request) {
	req.AddCookie(s.cookie)
	req.Header.Set("X-Csrf-Token", s.token)
}

// webLoginAs проходит настоящую форму входа за указанного пользователя.
// Пароль пользователя перезаписывается известным хешем: фабрики кладут заглушку.
func webLoginAs(t *testing.T, ts *testhelpers.TestServer, u *user.User) *webSession {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(integrationLoginPassword), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, ts.Repos.User.UpdatePassword(context.Background(), u.ID, string(hash)))

	cookie := loginViaForm(t, ts, newAnonymousSession(t, ts), u.Email)

	// Дашборд открыт любой роли, поэтому токен берётся с него.
	return &webSession{cookie: cookie, token: csrfTokenFromPage(t, ts, cookie, "/")}
}

// webAuth — cookie-сессия администратора тестовой семьи (создаёт семью и админа, как Auth).
func webAuth(t *testing.T, ts *testhelpers.TestServer) *webSession {
	t.Helper()

	ts.Auth(t)

	return webLoginAs(t, ts, ts.AuthUser)
}
