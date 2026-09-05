package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Учётные данные не должны переживать изменения в БД. Для cookie-сессий это
// обеспечивает RequireActiveUser (пользователь перечитывается на каждом
// запросе), для bearer — JOIN users в FindByTokenHash плюс отзыв сессий при
// деактивации: роль и активность берутся из БД, а не из выданного токена.

// TestSessionRevalidation_DeactivatedUserLosesAccess — деактивированный
// пользователь теряет и cookie-доступ к вебу, и bearer-доступ к API.
func TestSessionRevalidation_DeactivatedUserLosesAccess(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t) // семья + админ

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)
	memberWeb := webLoginAs(t, testServer, member)

	// Пока пользователь активен — доступ работает.
	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberWeb, "/transactions"))
	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"))

	require.NoError(
		t,
		testServer.Services.User.SetActive(context.Background(), member.ID, false, testServer.AuthUser.ID),
	)

	assert.Equal(t, http.StatusFound, doAuthedGET(t, testServer, memberWeb, "/transactions"),
		"веб-страница осталась доступна по cookie деактивированного пользователя")
	assert.Equal(t, http.StatusUnauthorized, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"),
		"API осталось доступно по токену деактивированного пользователя")
}

// TestSessionRevalidation_RoleDowngradeTakesEffect — роль берётся из БД, а не
// из токена: понижение до child закрывает финансовые разделы на следующем запросе.
func TestSessionRevalidation_RoleDowngradeTakesEffect(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"))

	member.Role = user.RoleChild
	require.NoError(t, testServer.Repos.User.Update(context.Background(), member))

	assert.Equal(t, http.StatusForbidden, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"),
		"роль всё ещё читается из выданного токена, а не из БД")
}

// TestSessionRevalidation_RoleUpgradeTakesEffect — повышение роли тоже видно
// сразу: member не допущен к /users, admin — допущен, токен тот же.
func TestSessionRevalidation_RoleUpgradeTakesEffect(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	require.Equal(t, http.StatusForbidden, doAuthedGET(t, testServer, memberAuth, "/api/v1/users"))

	member.Role = user.RoleAdmin
	require.NoError(t, testServer.Repos.User.Update(context.Background(), member))

	assert.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/users"))
}

// TestSessionRevalidation_RoleDowngradeAppliesToHTMXRoutes — та же проверка для
// группы /htmx.
//
// Регрессия: группа объявлялась как protected.Group("/htmx", RequireAuth()).
// Echo склеивает middleware родительской группы с middleware дочерней, поэтому
// цепочка получалась RequireAuth -> RequireActiveUser -> RequireAuth ->
// RequireAdminOrMember: третье звено клало в контекст SessionData из подписанной
// cookie поверх свежей, прочитанной из БД, и ролевая проверка видела старую
// роль. Понижённый до child пользователь сохранял доступ ко всем HTMX-ручкам
// (включая удаление транзакций) до конца SessionTimeout.
func TestSessionRevalidation_RoleDowngradeAppliesToHTMXRoutes(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	member, _ := testServer.AuthAs(t, user.RoleMember)
	memberWeb := webLoginAs(t, testServer, member)

	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberWeb, "/htmx/transactions/list"))

	member.Role = user.RoleChild
	require.NoError(t, testServer.Repos.User.Update(context.Background(), member))

	htmxRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/htmx/transactions/list"},
		{http.MethodGet, "/htmx/transactions/filter"},
		{http.MethodGet, "/htmx/categories/search"},
		{http.MethodDelete, "/htmx/transactions/bulk-delete"},
	}

	for _, route := range htmxRoutes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			memberWeb.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code,
				"роль на /htmx читается из подписанной cookie, а не из БД")
		})
	}
}
