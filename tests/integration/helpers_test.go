package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// doGET выполняет GET через полный стек тестового сервера и возвращает ответ.
// auth == nil означает анонимный запрос.
//
// Общая точка входа для всех HTTP-хелперов пакета: раньше каждый файл нёс
// собственную копию этих семи строк.
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

// fetchPage запрашивает HTML-страницу с сессией и возвращает тело ответа.
func fetchPage(t *testing.T, ts *testhelpers.TestServer, auth *testhelpers.AuthSession, path string) string {
	t.Helper()

	return requireOKBody(t, doGET(t, ts, auth, path), path)
}

// fetchAnonymousPage запрашивает страницу без сессии и возвращает тело ответа.
func fetchAnonymousPage(t *testing.T, ts *testhelpers.TestServer, path string) string {
	t.Helper()

	return requireOKBody(t, doGET(t, ts, nil, path), path)
}

// doAuthedGET выполняет GET с сессией и возвращает код ответа.
func doAuthedGET(t *testing.T, ts *testhelpers.TestServer, auth *testhelpers.AuthSession, path string) int {
	t.Helper()

	return doGET(t, ts, auth, path).Code
}

// requireOKBody требует 200 и отдаёт тело ответа.
func requireOKBody(t *testing.T, rec *httptest.ResponseRecorder, path string) string {
	t.Helper()

	require.Equal(t, http.StatusOK, rec.Code, "GET %s отдал %d, тело: %s", path, rec.Code, rec.Body.String())

	return rec.Body.String()
}
