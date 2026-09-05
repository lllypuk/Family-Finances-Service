package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Учётные данные не должны переживать изменения в БД: JOIN users в FindByTokenHash
// плюс отзыв сессий при деактивации — роль и активность берутся из БД,
// а не из выданного токена.

// TestSessionRevalidation_DeactivatedUserLosesAccess — деактивированный
// пользователь теряет доступ к API по уже выданному токену.
func TestSessionRevalidation_DeactivatedUserLosesAccess(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t) // семья + админ

	member, memberAuth := testServer.AuthAs(t, user.RoleMember)

	// Пока пользователь активен — доступ работает.
	require.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/transactions"))

	require.NoError(
		t,
		testServer.Services.User.SetActive(context.Background(), member.ID, false, testServer.AuthUser.ID),
	)

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

	require.NoError(t, testServer.Repos.User.UpdateRole(context.Background(), member.ID, user.RoleChild))

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

	require.NoError(t, testServer.Repos.User.UpdateRole(context.Background(), member.ID, user.RoleAdmin))

	assert.Equal(t, http.StatusOK, doAuthedGET(t, testServer, memberAuth, "/api/v1/users"))
}
