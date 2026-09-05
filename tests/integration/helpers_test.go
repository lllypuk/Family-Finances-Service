package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"family-budget-service/internal/testhelpers"
)

// doGET выполняет GET через полный стек тестового сервера и возвращает ответ.
// auth == nil означает анонимный запрос.
func doGET(
	t *testing.T,
	ts *testhelpers.TestServer,
	auth *testhelpers.AuthSession,
	path string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if auth != nil {
		auth.Apply(req)
	}
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)

	return rec
}

// doAuthedGET выполняет GET с токеном и возвращает код ответа.
func doAuthedGET(t *testing.T, ts *testhelpers.TestServer, auth *testhelpers.AuthSession, path string) int {
	t.Helper()

	return doGET(t, ts, auth, path).Code
}
