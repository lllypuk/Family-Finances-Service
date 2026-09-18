package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

func doAccountRequest(
	t *testing.T,
	ts *testhelpers.TestServer,
	sess *testhelpers.AuthSession,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sess != nil {
		sess.Apply(req)
	}
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	return rec
}

func decodeAccount(t *testing.T, rec *httptest.ResponseRecorder) handlers.AccountResponse {
	t.Helper()

	var resp handlers.APIResponse[handlers.AccountResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Data
}

func listAccountNames(t *testing.T, ts *testhelpers.TestServer, path string) []string {
	t.Helper()

	rec := doGET(t, ts, ts.Auth(t), path)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp handlers.APIResponse[[]handlers.AccountResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Meta.Pagination)
	assert.Equal(t, len(resp.Data), resp.Meta.Pagination.Total)

	result := make([]string, 0, len(resp.Data))
	for _, a := range resp.Data {
		result = append(result, a.Name)
	}

	return result
}

func errorCodeOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var resp handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Error.Code
}

func TestAccountsAPI_CRUD(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":" Тинькофф "}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	created := decodeAccount(t, rec)
	assert.Equal(t, "Тинькофф", created.Name)
	assert.False(t, created.IsArchived)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/accounts/"+created.ID.String(),
		`{"name":"Т-Банк"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Т-Банк", decodeAccount(t, rec).Name)

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, "/api/v1/accounts/"+created.ID.String(), "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, listAccountNames(t, ts, "/api/v1/accounts?archived=true"))

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, "/api/v1/accounts/"+created.ID.String(), "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeAccountNotFound, errorCodeOf(t, rec))
}

func TestAccountsAPI_ArchivedHiddenByDefault(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":"Сбер"}`)
	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":"Старая карта"}`)
	old := decodeAccount(t, rec)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/accounts/"+old.ID.String(),
		`{"is_archived":true}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, decodeAccount(t, rec).IsArchived)

	assert.Equal(t, []string{"Сбер"}, listAccountNames(t, ts, "/api/v1/accounts"))
	assert.Equal(t, []string{"Сбер", "Старая карта"}, listAccountNames(t, ts, "/api/v1/accounts?archived=true"))
}

func TestAccountsAPI_Validation(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	created := decodeAccount(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts",
		`{"name":"Сбер"}`))

	rec := doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/accounts/"+created.ID.String(), `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":"   "}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":"`+strings.Repeat("я", 51)+`"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/accounts/"+uuid.NewString(), `{"name":"x"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAccountsAPI_IdempotentCreate(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	id := uuid.NewString()

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"id":"`+id+`","name":"Сбер"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"id":"`+id+`","name":"Другое"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Сбер", decodeAccount(t, rec).Name)
	assert.Equal(t, []string{"Сбер"}, listAccountNames(t, ts, "/api/v1/accounts"))
}

func TestAccountsAPI_DuplicateName(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	archived := decodeAccount(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts",
		`{"name":"Карта"}`))
	doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/accounts/"+archived.ID.String(), `{"is_archived":true}`)

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/accounts", `{"name":" карта"}`)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, handlers.ErrCodeAccountNameExists, errorCodeOf(t, rec))
}

func TestAccountsAPI_Roles(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	_, member := ts.AuthAs(t, user.RoleMember)

	rec := doAccountRequest(t, ts, member, http.MethodPost, "/api/v1/accounts", `{"name":"Сбер"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	id := decodeAccount(t, rec).ID.String()

	rec = doAccountRequest(t, ts, member, http.MethodPut, "/api/v1/accounts/"+id, `{"is_archived":true}`)
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = doAccountRequest(t, ts, member, http.MethodDelete, "/api/v1/accounts/"+id, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = doAccountRequest(t, ts, nil, http.MethodGet, "/api/v1/accounts", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
