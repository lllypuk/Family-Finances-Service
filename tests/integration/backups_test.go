package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// doBackupRequest выполняет запрос к /api/v1/backups с сессией.
func doBackupRequest(
	t *testing.T,
	ts *testhelpers.TestServer,
	auth *testhelpers.AuthSession,
	method, path string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	return rec
}

// TestBackupsAPI_FullCycle — создание, список, скачивание и удаление бэкапа.
// Каталог задан через BACKUP_DIR тестового хелпера (t.TempDir()).
func TestBackupsAPI_FullCycle(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)

	rec := doBackupRequest(t, testServer, adminAuth, http.MethodPost, "/api/v1/backups")
	require.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())

	var created handlers.APIResponse[handlers.BackupResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	name := created.Data.Name
	require.NotEmpty(t, name)
	assert.Positive(t, created.Data.SizeBytes)

	rec = doBackupRequest(t, testServer, adminAuth, http.MethodGet, "/api/v1/backups")
	require.Equal(t, http.StatusOK, rec.Code)

	var listed handlers.APIResponse[[]handlers.BackupResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	names := make([]string, 0, len(listed.Data))
	for _, backup := range listed.Data {
		names = append(names, backup.Name)
	}
	assert.Contains(t, names, name)

	rec = doBackupRequest(t, testServer, adminAuth, http.MethodGet, "/api/v1/backups/"+name+"/download")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/octet-stream", rec.Header().Get(echo.HeaderContentType))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), name)
	assert.NotEmpty(t, rec.Body.Bytes())

	rec = doBackupRequest(t, testServer, adminAuth, http.MethodDelete, "/api/v1/backups/"+name)
	require.Equal(t, http.StatusNoContent, rec.Code)

	rec = doBackupRequest(t, testServer, adminAuth, http.MethodGet, "/api/v1/backups/"+name+"/download")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBackupsAPI_InvalidName(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)

	rec := doBackupRequest(t, testServer, adminAuth, http.MethodDelete, "/api/v1/backups/notes.txt")
	require.Equal(t, http.StatusBadRequest, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeInvalidBackupName, response.Error.Code)
}

func TestBackupsAPI_NotFound(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)

	rec := doBackupRequest(t, testServer, adminAuth,
		http.MethodDelete, "/api/v1/backups/backup_20200101_000000000.db")
	require.Equal(t, http.StatusNotFound, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBackupNotFound, response.Error.Code)
}

// TestBackupsAPI_MemberForbidden — бэкапы только для admin, как и в веб-админке.
func TestBackupsAPI_MemberForbidden(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)
	_, memberAuth := testServer.AuthAs(t, user.RoleMember)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/backups"},
		{http.MethodPost, "/api/v1/backups"},
		{http.MethodGet, "/api/v1/backups/backup_20200101_000000000.db/download"},
		{http.MethodDelete, "/api/v1/backups/backup_20200101_000000000.db"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doBackupRequest(t, testServer, memberAuth, tc.method, tc.path)
			assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
		})
	}
}
