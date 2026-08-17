package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
	"family-budget-service/internal/web/middleware"
)

// Регрессия на находку S-02 (docs/specs/002-security-audit.md#s-02): CSRF-токен
// не перевыпускался при логине, поэтому токен, полученный анонимно (и, значит,
// потенциально подсунутый атакующим), продолжал действовать после входа.
//
// Задача 7 плана docs/plans/20260816-deployment-blockers.md. Проверка идёт на
// полном стеке — настоящие шаблоны, SessionStore, CSRFProtection и маршруты, —
// потому что перевыпуск легко сделать так, что вход через UI сломается:
// ClearSession выставляет MaxAge = -1 на том же объекте сессии, и без его
// восстановления клиент получил бы cookie на удаление.

// integrationLoginPassword — пароль тестового пользователя (bcrypt-хеш кладётся в БД).
const integrationLoginPassword = "Admin123!"

func TestLogin_RegeneratesCSRFToken(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t) // семья, иначе RequireSetup уводит всё на /setup

	email := createPasswordUser(t, testServer)
	anon := newAnonymousSession(t, testServer)
	authCookie := loginViaForm(t, testServer, anon, email)

	require.Positive(t, authCookie.MaxAge,
		"cookie сессии после входа обязана жить, иначе вход через UI сломан")

	// Токен снимается до подтестов: иначе `go test -run …/NewTokenAccepted`
	// падал бы на пустом newToken — подтесты не должны зависеть от порядка.
	newToken := csrfTokenFromPage(t, testServer, authCookie, "/admin/users")

	t.Run("FormCarriesNewToken", func(t *testing.T) {
		assert.NotEqual(t, anon.token, newToken,
			"после входа форма обязана нести новый CSRF-токен")
	})

	t.Run("AnonymousTokenRejected", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, csrfProbe(t, testServer, authCookie, anon.token),
			"токен, полученный до входа, обязан отвергаться")
	})

	t.Run("NewTokenAccepted", func(t *testing.T) {
		assert.Equal(t, http.StatusFound, csrfProbe(t, testServer, authCookie, newToken),
			"новый токен обязан приниматься — иначе вход через UI сломан")
	})
}

func TestLogout_ClearsCSRFToken(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	email := createPasswordUser(t, testServer)
	anon := newAnonymousSession(t, testServer)
	authCookie := loginViaForm(t, testServer, anon, email)
	authToken := csrfTokenFromPage(t, testServer, authCookie, "/admin/users")

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(authCookie)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)

	cleared := lastSessionCookie(t, rec)
	assert.Negative(t, cleared.MaxAge, "выход обязан удалять cookie сессии")

	assert.Equal(t, http.StatusForbidden, csrfProbe(t, testServer, cleared, authToken),
		"после выхода прежний CSRF-токен не должен подходить к сессии")
}

// createPasswordUser заводит пользователя с настоящим bcrypt-хешем: CreateTestUser
// кладёт в поле пароля заглушку, через которую реальный POST /login не пройдёт.
func createPasswordUser(t *testing.T, ts *testhelpers.TestServer) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(integrationLoginPassword), bcrypt.MinCost)
	require.NoError(t, err)

	newUser := testhelpers.CreateTestUser(ts.AuthFamily.ID)
	newUser.Email = "login-rotation@example.com"
	newUser.Password = string(hash)
	require.NoError(t, ts.Repos.User.Create(context.Background(), newUser))

	return newUser.Email
}

// loginViaForm проходит настоящую форму входа анонимным CSRF-токеном и
// возвращает cookie аутентифицированной сессии.
func loginViaForm(t *testing.T, ts *testhelpers.TestServer, anon *anonymousSession, email string) *http.Cookie {
	t.Helper()

	form := url.Values{}
	form.Set("email", email)
	form.Set("password", integrationLoginPassword)
	form.Set("_token", anon.token)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(anon.cookie)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code, "вход не удался: %s", rec.Body.String())

	return lastSessionCookie(t, rec)
}

// csrfTokenFromPage забирает токен из скрытого поля формы на отрендеренной странице.
func csrfTokenFromPage(t *testing.T, ts *testhelpers.TestServer, cookie *http.Cookie, path string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "страница %s недоступна", path)

	match := csrfFormTokenRe.FindStringSubmatch(rec.Body.String())
	require.Len(t, match, 2, "CSRF-токен не найден в форме на %s", path)

	return match[1]
}

// csrfProbe шлёт POST /login с указанным токеном. Глобальный CSRFProtection
// отрабатывает раньше маршрутного RedirectIfAuthenticated, поэтому негодный
// токен даёт 403, а годный — 302 на главную.
func csrfProbe(t *testing.T, ts *testhelpers.TestServer, cookie *http.Cookie, token string) int {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set(middleware.CSRFHeaderKey, token)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)

	return rec.Code
}

// lastSessionCookie возвращает последнюю cookie сессии из ответа: за один запрос
// сессия сохраняется несколько раз, а клиент применяет Set-Cookie по порядку.
func lastSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	var found *http.Cookie
	for _, cookie := range (&http.Response{Header: rec.Header()}).Cookies() {
		if cookie.Name == middleware.SessionName {
			found = cookie
		}
	}
	require.NotNil(t, found, "сессионная cookie не выдана")

	return found
}

// TestInviteRegister_RegeneratesSession — та же находка S-02 на второй точке
// входа: регистрация по приглашению создавала сессию поверх анонимной, не
// сбрасывая её CSRF-токен. В однофамильной модели приглашение — единственный
// способ появления новых участников, то есть путь каждого не-администратора.
func TestInviteRegister_RegeneratesSession(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	admin := testServer.Auth(t)
	_ = admin

	const invitedEmail = "invited-member@example.com"

	invite, err := testServer.Services.Invite.CreateInvite(
		context.Background(),
		testServer.AuthUser.ID,
		dto.CreateInviteDTO{Email: invitedEmail, Role: string(user.RoleMember)},
	)
	require.NoError(t, err)

	anon := newAnonymousSession(t, testServer)

	form := url.Values{}
	form.Set("email", invitedEmail)
	form.Set("name", "Invited Member")
	form.Set("password", integrationLoginPassword)
	form.Set("_token", anon.token)

	req := httptest.NewRequest(http.MethodPost, "/invite/"+invite.Token, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(anon.cookie)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code, "регистрация по приглашению не удалась: %s", rec.Body.String())

	memberCookie := lastSessionCookie(t, rec)
	require.Positive(t, memberCookie.MaxAge, "cookie сессии после приглашения обязана жить")

	assert.Equal(t, http.StatusForbidden, csrfProbe(t, testServer, memberCookie, anon.token),
		"токен анонимной сессии обязан перестать действовать после регистрации по приглашению")

	newToken := csrfTokenFromPage(t, testServer, memberCookie, "/transactions")
	assert.NotEqual(t, anon.token, newToken, "после регистрации форма обязана нести новый CSRF-токен")
	assert.Equal(t, http.StatusFound, csrfProbe(t, testServer, memberCookie, newToken),
		"новый токен обязан приниматься")
}
