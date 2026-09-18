package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

func decodeHolding(t *testing.T, rec *httptest.ResponseRecorder) handlers.HoldingResponse {
	t.Helper()

	var resp handlers.APIResponse[handlers.HoldingResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Data
}

func listHoldingNames(t *testing.T, ts *testhelpers.TestServer, path string) []string {
	t.Helper()

	rec := doGET(t, ts, ts.Auth(t), path)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp handlers.APIResponse[[]handlers.HoldingResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Meta.Pagination)
	assert.Equal(t, len(resp.Data), resp.Meta.Pagination.Total)

	result := make([]string, 0, len(resp.Data))
	for _, h := range resp.Data {
		result = append(result, h.Name)
	}

	return result
}

func TestHoldingsAPI_CRUD(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":" Квартира ","side":"asset","kind":"property"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"current":null`)
	created := decodeHolding(t, rec)
	assert.Equal(t, "Квартира", created.Name)
	assert.Equal(t, "asset", created.Side)
	assert.Nil(t, created.Current)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/holdings/"+created.ID.String(),
		`{"name":"Дача","kind":"other","side":"liability"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	updated := decodeHolding(t, rec)
	assert.Equal(t, "Дача", updated.Name)
	assert.Equal(t, "other", updated.Kind)
	assert.Equal(t, "asset", updated.Side, "side в PUT игнорируется")

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, "/api/v1/holdings/"+created.ID.String(), "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, listHoldingNames(t, ts, "/api/v1/holdings?archived=true"))

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, "/api/v1/holdings/"+created.ID.String(), "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeHoldingNotFound, errorCodeOf(t, rec))
}

func TestHoldingsAPI_ArchivedHiddenByDefault(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Ипотека","side":"liability","kind":"mortgage"}`)
	old := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Машина","side":"asset","kind":"vehicle"}`))

	rec := doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/holdings/"+old.ID.String(), `{"is_archived":true}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, decodeHolding(t, rec).IsArchived)

	assert.Equal(t, []string{"Ипотека"}, listHoldingNames(t, ts, "/api/v1/holdings"))
	assert.Equal(t, []string{"Ипотека", "Машина"}, listHoldingNames(t, ts, "/api/v1/holdings?archived=true"))
}

func TestHoldingsAPI_IdempotentCreate(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	id := uuid.NewString()

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"id":"`+id+`","name":"Вклад","side":"asset","kind":"deposit"}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"id":"`+id+`","name":"Другое","side":"liability","kind":"loan"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Вклад", decodeHolding(t, rec).Name)
	assert.Equal(t, []string{"Вклад"}, listHoldingNames(t, ts, "/api/v1/holdings"))
}

func TestHoldingsAPI_Validation(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	rec := doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Квартира","side":"asset","kind":"mortgage"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	created := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Квартира","side":"asset","kind":"property"}`))
	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/holdings/"+created.ID.String(), `{"kind":"loan"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/holdings/"+created.ID.String(), `{"name":"  "}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "пустое имя в PUT — 422, а не CHECK базы")

	rec = doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":" квартира","side":"liability","kind":"loan"}`)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, handlers.ErrCodeHoldingNameExists, errorCodeOf(t, rec))
}

func TestHoldingsAPI_Roles(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	_, member := ts.AuthAs(t, user.RoleMember)

	rec := doAccountRequest(t, ts, member, http.MethodPost, "/api/v1/holdings",
		`{"name":"Вклад","side":"asset","kind":"deposit"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	id := decodeHolding(t, rec).ID.String()

	rec = doAccountRequest(t, ts, member, http.MethodPut, "/api/v1/holdings/"+id, `{"is_archived":true}`)
	assert.Equal(t, http.StatusOK, rec.Code)

	day := date.Today(ts.AuthFamily.Location()).String()
	rec = doAccountRequest(t, ts, member, http.MethodPut, "/api/v1/holdings/"+id+"/values/"+day, `{"value_minor":1}`)
	assert.Equal(t, http.StatusOK, rec.Code)
	rec = doGET(t, ts, member, "/api/v1/holdings/"+id+"/values")
	assert.Equal(t, http.StatusOK, rec.Code)
	rec = doAccountRequest(t, ts, member, http.MethodDelete, "/api/v1/holdings/"+id+"/values/"+day, "")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = doAccountRequest(t, ts, member, http.MethodDelete, "/api/v1/holdings/"+id, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = doAccountRequest(t, ts, nil, http.MethodGet, "/api/v1/holdings", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	rec = doAccountRequest(t, ts, nil, http.MethodGet, "/api/v1/holdings/"+id+"/values", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func holdingValuesPath(id uuid.UUID, day string) string {
	path := "/api/v1/holdings/" + id.String() + "/values"
	if day != "" {
		path += "/" + day
	}
	return path
}

func TestHoldingsAPI_Values(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	h := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Вклад","side":"asset","kind":"deposit"}`))
	today := date.Today(ts.AuthFamily.Location())
	earlier := today.AddDays(-10)

	rec := doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, today.String()),
		`{"value_minor":500}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, today.String()),
		`{"value_minor":700}`)
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, earlier.String()),
		`{"value_minor":0}`)
	require.Equal(t, http.StatusOK, rec.Code, "0 принимается")

	rec = doGET(t, ts, admin, holdingValuesPath(h.ID, ""))
	require.Equal(t, http.StatusOK, rec.Code)
	var history handlers.APIResponse[[]handlers.HoldingValueResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &history))
	require.Len(t, history.Data, 2, "повторный PUT на дату — одна строка")
	assert.Equal(t, 2, history.Meta.Pagination.Total)
	assert.Equal(t, today, history.Data[0].Date)
	assert.Equal(t, money.Minor(700), history.Data[0].ValueMinor)

	var list handlers.APIResponse[[]handlers.HoldingResponse]
	require.NoError(t, json.Unmarshal(doGET(t, ts, admin, "/api/v1/holdings").Body.Bytes(), &list))
	require.NotNil(t, list.Data[0].Current)
	assert.Equal(t, today, list.Data[0].Current.Date, "current — по дате, а не последний записанный")
	assert.Equal(t, money.Minor(700), list.Data[0].Current.ValueMinor)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, today.AddDays(1).String()),
		`{"value_minor":1}`)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "завтра")
	rec = doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, "yesterday"), `{"value_minor":1}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, holdingValuesPath(h.ID, earlier.String()), "")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	rec = doAccountRequest(t, ts, admin, http.MethodDelete, holdingValuesPath(h.ID, earlier.String()), "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, handlers.ErrCodeHoldingValueNotFound, errorCodeOf(t, rec))
	rec = doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(uuid.New(), today.String()),
		`{"value_minor":1}`)
	assert.Equal(t, handlers.ErrCodeHoldingNotFound, errorCodeOf(t, rec))

	rec = doAccountRequest(t, ts, admin, http.MethodDelete, "/api/v1/holdings/"+h.ID.String(), "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	rec = doGET(t, ts, admin, holdingValuesPath(h.ID, ""))
	assert.Equal(t, http.StatusNotFound, rec.Code, "позиция ушла вместе со снимками")
}

func TestHoldingsAPI_ZeroValueLocksCurrency(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	h := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Вклад","side":"asset","kind":"deposit"}`))
	today := date.Today(ts.AuthFamily.Location())

	rec := doAccountRequest(t, ts, admin, http.MethodPut, holdingValuesPath(h.ID, today.String()),
		`{"value_minor":0}`)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doAccountRequest(t, ts, admin, http.MethodPut, "/api/v1/family", `{"currency":"USD"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeCurrencyLocked, errorCodeOf(t, rec))
}
