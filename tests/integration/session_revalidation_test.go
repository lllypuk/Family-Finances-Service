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

// Хранилище сессий cookie-based: id пользователя и его роль приходят из
// подписанной cookie и раньше никем не перепроверялись. Поэтому удалённый
// администратором участник продолжал работать до конца SessionTimeout (24
// часа) — и в вебе, и в /api/v1, — а понижение роли не действовало вовсе.
// Отозвать доступ было нечем: серверного session id не существует.
//
// Красная фаза (до RequireActiveUser/RequireAPIActiveUser): все проверки ниже
// получали 200 вместо 302/401/403.

// TestSessionRevalidation_DeletedUserLosesAccess — cookie удалённого
// пользователя больше не открывает ни веб-страницу, ни API.
func TestSessionRevalidation_DeletedUserLosesAccess(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t) // семья + админ

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	// Пока пользователь есть — доступ работает.
	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/transactions"))
	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"))

	require.NoError(t, testServer.Repos.User.Delete(context.Background(), member.ID))

	assert.Equal(t, http.StatusFound, doAuthedGET(t, testServer, memberAuth, "/transactions"),
		"веб-страница осталась доступна по cookie удалённого пользователя")
	assert.Equal(t, http.StatusUnauthorized, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"),
		"API осталось доступно по cookie удалённого пользователя")
}

// TestSessionRevalidation_RoleDowngradeTakesEffect — роль берётся из БД, а не
// из cookie: понижение до child закрывает финансовые разделы немедленно.
func TestSessionRevalidation_RoleDowngradeTakesEffect(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"))

	member.Role = user.RoleChild
	require.NoError(t, testServer.Repos.User.Update(context.Background(), member))

	assert.Equal(t, http.StatusForbidden, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"),
		"роль всё ещё читается из подписанной cookie, а не из БД")
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

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/htmx/transactions/list"))

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
			memberAuth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code,
				"роль на /htmx читается из подписанной cookie, а не из БД")
		})
	}
}
