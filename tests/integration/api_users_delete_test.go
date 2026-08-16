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

// DELETE /api/v1/users/:id не имел ни запрета на самоудаление (в вебе он есть:
// AdminHandler.DeleteUser), ни защиты последнего администратора. В
// однофамильной модели это невосстановимо: семья остаётся, RequireSetup гоняет
// /setup → /login, открытой регистрации нет, а инвайт выпускает только админ.
// RequireAPIActiveUser отзывает сессию удалённого пользователя на следующем же
// запросе, так что оператор теряет консоль немедленно.
func TestAPIUsers_DeleteSelfRejected(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	self := testServer.AuthUser

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+self.ID.String(), nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "тело: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "CANNOT_DELETE_SELF")

	stored, err := testServer.Repos.User.GetByID(context.Background(), self.ID)
	require.NoError(t, err, "пользователь всё-таки удалён")
	assert.Equal(t, self.ID, stored.ID)
}

// TestAPIUsers_DeleteLastAdminRejected — вторая половина защиты: последний
// администратор не удаляется даже чужой сессией. Проверка живёт в
// userService.DeleteUser, поэтому одинаково закрывает и API, и веб.
func TestAPIUsers_DeleteLastAdminRejected(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	admin := testServer.AuthUser

	// Понижаем всех остальных до member: в семье остаётся ровно один админ.
	others, err := testServer.Repos.User.GetAll(context.Background())
	require.NoError(t, err)
	for _, other := range others {
		if other.ID == admin.ID {
			continue
		}
		other.Role = user.RoleMember
		require.NoError(t, testServer.Repos.User.Update(context.Background(), other))
	}

	// Сервис вызывается напрямую: через API тот же запрос упёрся бы в запрет
	// самоудаления раньше, а проверить надо именно защиту последнего админа.
	err = testServer.Services.User.DeleteUser(context.Background(), admin.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "last admin")

	stored, getErr := testServer.Repos.User.GetByID(context.Background(), admin.ID)
	require.NoError(t, getErr, "последний администратор всё-таки удалён")
	assert.Equal(t, admin.ID, stored.ID)

	// А удаление обычного участника той же сессией по-прежнему проходит.
	member, _ := testServer.AuthAs(t, user.RoleMember)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+member.ID.String(), nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code, "тело: %s", rec.Body.String())
}
