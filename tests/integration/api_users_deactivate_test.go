package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/testhelpers"
)

// adminJSON — запрос с bearer-токеном администратора и JSON-телом.
func adminJSON(
	t *testing.T, ts *testhelpers.TestServer, sess *testhelpers.AuthSession, method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	sess.Apply(req)
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)
	return rec
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) handlers.UserResponse {
	t.Helper()

	var response handlers.APIResponse[handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), rec.Body.String())
	return response.Data
}

// Деактивация (A-04): токен деактивированного — 401 на следующем же запросе, логин — 401,
// в списке он остаётся с is_active=false; повторная активация возвращает вход.
func TestAPIUsers_Deactivate_RevokesAccess(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	member := createBearerMember(t, ts)
	token := loginBearer(t, ts, member.Email, bearerPassword, "Pixel 8").Token
	path := "/api/v1/users/" + member.ID.String()

	require.Equal(t, http.StatusOK,
		bearerRequest(ts, http.MethodGet, "/api/v1/transactions", token, nil).Code)

	rec := adminJSON(t, ts, admin, http.MethodPatch, path, `{"is_active":false}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.False(t, decodeUser(t, rec).IsActive)

	rec = bearerRequest(ts, http.MethodGet, "/api/v1/transactions", token, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "токен деактивированного пользователя жив")

	rec = bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": member.Email, "password": bearerPassword,
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_CREDENTIALS", errorCode(t, rec))

	rec = adminJSON(t, ts, admin, http.MethodGet, "/api/v1/users", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var list handlers.APIResponse[[]handlers.UserResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	seen := false
	for _, u := range list.Data {
		if u.ID == member.ID {
			seen = true
			assert.False(t, u.IsActive)
		}
	}
	assert.True(t, seen, "GET /users скрыл неактивного пользователя")

	rec = adminJSON(t, ts, admin, http.MethodPatch, path, `{"is_active":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.True(t, decodeUser(t, rec).IsActive)
	loginBearer(t, ts, member.Email, bearerPassword, "Pixel 8")
}

// Самодеактивация: администратор мгновенно потерял бы доступ, а в однофамильной
// модели вернуть его некому — 409 CANNOT_DEACTIVATE_SELF, запись не тронута.
func TestAPIUsers_DeactivateSelfRejected(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	self := ts.AuthUser

	rec := adminJSON(t, ts, admin, http.MethodPatch, "/api/v1/users/"+self.ID.String(), `{"is_active":false}`)

	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, "CANNOT_DEACTIVATE_SELF", errorCode(t, rec))

	stored, err := ts.Repos.User.GetByID(context.Background(), self.ID)
	require.NoError(t, err)
	assert.True(t, stored.IsActive)
}

// Последний активный администратор не деактивируется даже чужой сессией:
// неактивные админы в счёт не идут. Проверка живёт в userService.SetActive.
func TestAPIUsers_DeactivateLastAdminRejected(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	firstSess := ts.Auth(t)
	first := ts.AuthUser
	second, secondSess := ts.AuthAs(t, user.RoleAdmin)

	rec := adminJSON(t, ts, secondSess, http.MethodPatch, "/api/v1/users/"+first.ID.String(), `{"is_active":false}`)
	require.Equal(t, http.StatusOK, rec.Code, "второй админ остаётся, первого можно выключить: %s", rec.Body.String())

	assert.Equal(t, http.StatusUnauthorized, doAuthedGET(t, ts, firstSess, "/api/v1/transactions"),
		"токен деактивированного администратора всё ещё открывает API")

	err := ts.Services.User.SetActive(context.Background(), second.ID, false, uuid.New())
	require.ErrorIs(t, err, services.ErrLastAdmin)

	stored, getErr := ts.Repos.User.GetByID(context.Background(), second.ID)
	require.NoError(t, getErr)
	assert.True(t, stored.IsActive)

	rec = adminJSON(t, ts, secondSess, http.MethodPatch, "/api/v1/users/"+first.ID.String(), `{"is_active":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// Роль и активность в одном PATCH; пустое тело — 422.
func TestAPIUsers_PatchValidation(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	member, _ := ts.AuthAs(t, user.RoleMember)
	path := "/api/v1/users/" + member.ID.String()

	rec := adminJSON(t, ts, admin, http.MethodPatch, path, `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	rec = adminJSON(t, ts, admin, http.MethodPatch, path, `{"role":"child","is_active":false}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	got := decodeUser(t, rec)
	assert.Equal(t, "child", got.Role)
	assert.False(t, got.IsActive)
}

// PUT /users/{id}/password: без текущего пароля, все сессии пользователя отзываются.
func TestAPIUsers_SetPassword(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	member := createBearerMember(t, ts)
	token := loginBearer(t, ts, member.Email, bearerPassword, "Pixel 8").Token
	path := "/api/v1/users/" + member.ID.String() + "/password"

	rec := adminJSON(t, ts, admin, http.MethodPut, path, `{"new_password":"short"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	rec = adminJSON(t, ts, admin, http.MethodPut, "/api/v1/users/"+uuid.NewString()+"/password",
		`{"new_password":"`+bearerNewPassword+`"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	rec = adminJSON(t, ts, admin, http.MethodPut, path, `{"new_password":"`+bearerNewPassword+`"}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

	assert.Equal(t, http.StatusUnauthorized,
		bearerRequest(ts, http.MethodGet, "/api/v1/transactions", token, nil).Code,
		"старая сессия пережила сброс пароля")

	rec = bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email": member.Email, "password": bearerPassword,
	})
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "старый пароль всё ещё подходит")

	loginBearer(t, ts, member.Email, bearerNewPassword, "Pixel 8")
}
