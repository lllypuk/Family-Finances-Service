package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Ошибки, не дошедшие до хендлера (404 роутера, 405, отказ middleware), тоже обязаны
// приходить в общем envelope: раньше клиент API получал на них HTML-страницу
// ошибки или строку plain text и не мог прочитать error.code (A-08).
func TestAPIErrorEnvelope_ErrorsOutsideHandlers(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	cases := []struct {
		name     string
		method   string
		target   string
		withAuth bool
		status   int
		code     string
	}{
		{"UnknownRoute", http.MethodGet, "/api/v1/does-not-exist", true, http.StatusNotFound, handlers.ErrCodeNotFound},
		{"UnknownRouteOutsideAPI", http.MethodGet, "/login", false, http.StatusNotFound, handlers.ErrCodeNotFound},
		// 405 недостижим нигде: catch-all группы перехватывает любой метод внутри /api/v1,
		// корневой RouteNotFound("/*") — снаружи.
		{"WrongMethod", http.MethodPost, "/health", false, http.StatusNotFound, handlers.ErrCodeNotFound},
		{
			name: "WriteWithoutToken", method: http.MethodPost, target: "/api/v1/transactions", withAuth: false,
			status: http.StatusUnauthorized, code: handlers.ErrCodeUnauthorized,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(testCase.method, testCase.target, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			if testCase.withAuth {
				session.Apply(req)
			}
			rec := httptest.NewRecorder()

			testServer.Server.ServeHTTP(rec, req)

			require.Equal(t, testCase.status, rec.Code, "тело: %s", rec.Body.String())
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, testCase.code, response.Error.Code)
			assert.NotEmpty(t, response.Error.Message)
			assert.NotEmpty(t, response.Meta.Version)
		})
	}
}

// snapshotCase — запрос и ответ, который обязан пережить смену HTTP-стека без правок;
// route — метка метрик (TestMetrics_RouteLabels).
type snapshotCase struct {
	name    string
	method  string
	target  string
	who     string // "", "admin" или "member"
	body    string
	status  int
	code    string
	route   string
	headers map[string]string // ожидаемые значения, "" — заголовок обязан отсутствовать
}

const (
	whoAdmin  = "admin"
	whoMember = "member"
)

func snapshotCases() []snapshotCase {
	const (
		anyRoute = "/*"
		apiRoute = "/api/v1/*"
	)
	notFound := handlers.ErrCodeNotFound
	unauth := handlers.ErrCodeUnauthorized
	forbidden := handlers.ErrCodeForbidden
	return []snapshotCase{
		{name: "UnknownInAPIAdmin", method: http.MethodGet, target: "/api/v1/nope", who: whoAdmin,
			status: 404, code: notFound, route: apiRoute},
		{name: "UnknownInAPINoToken", method: http.MethodGet, target: "/api/v1/nope",
			status: 401, code: unauth, route: apiRoute},
		{name: "UnknownOutsideAPIAdmin", method: http.MethodGet, target: "/nope", who: whoAdmin,
			status: 404, code: notFound, route: anyRoute},
		{name: "UnknownOutsideAPINoToken", method: http.MethodGet, target: "/nope",
			status: 404, code: notFound, route: anyRoute},
		{name: "WrongMethodHealth", method: http.MethodPost, target: "/health",
			status: 404, code: notFound, route: anyRoute},
		{name: "WrongMethodLogin", method: http.MethodGet, target: "/api/v1/auth/login",
			status: 401, code: unauth, route: apiRoute},
		{name: "WrongMethodTransactionsAdmin", method: http.MethodPatch, target: "/api/v1/transactions", who: whoAdmin,
			status: 404, code: notFound, route: apiRoute},
		{name: "WrongMethodTransactionsNoToken", method: http.MethodPatch, target: "/api/v1/transactions",
			status: 401, code: unauth, route: apiRoute},
		// Промах по методу на существующем пути уходит в fallback родителя (/api/v1/*),
		// а не подгруппы: роль не проверяется — member получает 404, а не 403.
		{name: "WrongMethodUsersRootMember", method: http.MethodDelete, target: "/api/v1/users", who: whoMember,
			status: 404, code: notFound, route: apiRoute},
		{name: "WrongMethodUsersRootAdmin", method: http.MethodDelete, target: "/api/v1/users", who: whoAdmin,
			status: 404, code: notFound, route: apiRoute},
		{name: "WrongMethodUsersDeepNoToken", method: http.MethodPost, target: "/api/v1/users/x/password",
			status: 401, code: unauth, route: "/api/v1/users/*"},
		{
			name:   "WrongMethodUsersDeepMember",
			method: http.MethodPost,
			target: "/api/v1/users/x/password",
			who:    whoMember,
			status: 403,
			code:   forbidden,
			route:  "/api/v1/users/*",
		},
		{name: "WrongMethodUsersDeepAdmin", method: http.MethodPost, target: "/api/v1/users/x/password", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/users/*"},
		{name: "MemberOnUsers", method: http.MethodGet, target: "/api/v1/users", who: whoMember,
			status: 403, code: forbidden, route: "/api/v1/users"},
		{name: "HeadHealth", method: http.MethodHead, target: "/health",
			status: 404, code: notFound, route: anyRoute},
		{name: "HeadAPINoToken", method: http.MethodHead, target: "/api/v1/transactions",
			status: 401, route: apiRoute},
		{name: "OptionsHealth", method: http.MethodOptions, target: "/health",
			status: 404, code: notFound, route: anyRoute},
		{name: "OptionsAPIAdmin", method: http.MethodOptions, target: "/api/v1/transactions", who: whoAdmin,
			status: 404, code: notFound, route: apiRoute},
		{name: "OptionsAPINoToken", method: http.MethodOptions, target: "/api/v1/transactions",
			status: 401, code: unauth, route: apiRoute},
		{name: "TrailingSlashList", method: http.MethodGet, target: "/api/v1/transactions/", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/transactions/*"},
		{name: "TrailingSlashHealth", method: http.MethodGet, target: "/health/",
			status: 404, code: notFound, route: anyRoute},
		{name: "DoubleSlash", method: http.MethodGet, target: "//api/v1/transactions", who: whoAdmin,
			status: 404, code: notFound, route: anyRoute},
		{name: "EncodedSlashInID", method: http.MethodGet, target: "/api/v1/transactions/a%2Fb", who: whoAdmin,
			status: 400, code: handlers.ErrCodeInvalidID, route: "/api/v1/transactions/:id"},
		{
			name:   "EncodedSlashInBackupName",
			method: http.MethodGet,
			target: "/api/v1/backups/a%2Fb/download",
			who:    whoAdmin,
			status: 400,
			code:   handlers.ErrCodeInvalidBackupName,
			route:  "/api/v1/backups/:name/download",
		},
		{name: "TerminalParamWithSlash", method: http.MethodGet, target: "/api/v1/transactions/a/b", who: whoAdmin,
			status: 400, code: handlers.ErrCodeInvalidID, route: "/api/v1/transactions/:id"},
		{name: "EmptyInteriorParam", method: http.MethodPut, target: "/api/v1/users//password", who: whoAdmin,
			status: 400, code: handlers.ErrCodeInvalidID, route: "/api/v1/users/:id/password"},
		{name: "NonLeafParamGetPassword", method: http.MethodGet, target: "/api/v1/users/x/password", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/users/*"},
		{
			name:   "NonLeafParamDeletePassword",
			method: http.MethodDelete,
			target: "/api/v1/users/x/password",
			who:    whoAdmin,
			status: 404,
			code:   notFound,
			route:  "/api/v1/users/*",
		},
		{name: "NonLeafParamUsersTwoSegments", method: http.MethodGet, target: "/api/v1/users/a/b", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/users/*"},
		{name: "NonLeafParamPasswordTail", method: http.MethodPut, target: "/api/v1/users/x/password/y", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/users/*"},
		{
			name:   "NonLeafParamAccountsBalances",
			method: http.MethodGet,
			target: "/api/v1/accounts/a/balances",
			who:    whoAdmin,
			status: 404,
			code:   notFound,
			route:  "/api/v1/accounts/*",
		},
		{
			name:   "NonLeafParamAccountsThreeSegments",
			method: http.MethodPut,
			target: "/api/v1/accounts/a/b/c",
			who:    whoAdmin,
			status: 404,
			code:   notFound,
			route:  "/api/v1/accounts/*",
		},
		{name: "NonLeafParamBackupsTwoSegments", method: http.MethodGet, target: "/api/v1/backups/a/b", who: whoAdmin,
			status: 404, code: notFound, route: "/api/v1/backups/*"},
		{name: "NonLeafParamBackupDeleteDownload", method: http.MethodDelete, target: "/api/v1/backups/a/download",
			who: whoAdmin, status: 404, code: notFound, route: "/api/v1/backups/*"},
		{
			name:   "LeafParamSessionWithSlash",
			method: http.MethodDelete,
			target: "/api/v1/auth/sessions/a/b",
			who:    whoAdmin,
			status: 400,
			code:   handlers.ErrCodeInvalidID,
			route:  "/api/v1/auth/sessions/:id",
		},
		{name: "PercentEncodedUUIDDigit", method: http.MethodGet,
			target: "/api/v1/transactions/%3123e4567-e89b-12d3-a456-426614174000", who: whoAdmin,
			status: 400, code: handlers.ErrCodeInvalidID, route: "/api/v1/transactions/:id"},
		{name: "BodyEmpty", method: http.MethodPost, target: "/api/v1/transactions", who: whoAdmin, body: "",
			status: 422, code: handlers.ErrCodeValidationError, route: "/api/v1/transactions"},
		{name: "BodyNull", method: http.MethodPost, target: "/api/v1/transactions", who: whoAdmin, body: "null",
			status: 422, code: handlers.ErrCodeValidationError, route: "/api/v1/transactions"},
		{name: "BodyExtraFields", method: http.MethodPost, target: "/api/v1/transactions", who: whoAdmin,
			body: `{"zzz":1}`, status: 422, code: handlers.ErrCodeValidationError, route: "/api/v1/transactions"},
		{name: "BodyGarbage", method: http.MethodPost, target: "/api/v1/transactions", who: whoAdmin, body: "{{{",
			status: 400, code: handlers.ErrCodeInvalidRequest, route: "/api/v1/transactions"},
		{
			name:   "BodyTrailingGarbage",
			method: http.MethodPost,
			target: "/api/v1/transactions",
			who:    whoAdmin,
			body:   `{"zzz":1}garbage`,
			status: 422,
			code:   handlers.ErrCodeValidationError,
			route:  "/api/v1/transactions",
		},
		{name: "StaticMethodMissGetFallsToParam", method: http.MethodGet, target: "/api/v1/transactions/recognize",
			who: whoAdmin, status: 400, code: handlers.ErrCodeInvalidID, route: "/api/v1/transactions/:id"},
		{name: "StaticMethodMissDeleteFallsToParam", method: http.MethodDelete,
			target: "/api/v1/transactions/bulk-delete", who: whoAdmin, status: 400, code: handlers.ErrCodeInvalidID,
			route: "/api/v1/transactions/:id"},
		{name: "Panic", method: http.MethodGet, target: testhelpers.PanicPath,
			status: 500, code: handlers.ErrCodeInternal, route: testhelpers.PanicPath},
	}
}

func (c snapshotCase) do(t *testing.T, ts *testhelpers.TestServer) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(c.method, c.target, strings.NewReader(c.body))
	req.Header.Set("Content-Type", "application/json")
	switch c.who {
	case whoAdmin:
		ts.Auth(t).Apply(req)
	case whoMember:
		_, sess := ts.AuthAs(t, user.RoleMember)
		sess.Apply(req)
	}
	rec := httptest.NewRecorder()
	ts.Server.ServeHTTP(rec, req)
	return rec
}

// Снимок поведения стека: ожидания не меняются при замене роутера (план 24).
func TestAPISnapshot_StatusCodeAndHeaders(t *testing.T) {
	for _, c := range snapshotCases() {
		t.Run(c.name, func(t *testing.T) {
			ts := testhelpers.SetupHTTPServer(t)
			ts.AddPanicRoute()

			rec := c.do(t, ts)

			require.Equal(t, c.status, rec.Code, "тело: %s", rec.Body.String())
			assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
			if c.code == "" {
				return
			}
			assert.Equal(t, c.code, errorCode(t, rec))
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
			for name, want := range c.headers {
				assert.Equal(t, want, rec.Header().Get(name), name)
			}
		})
	}
}

func TestAPISnapshot_DownloadHeaders(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)

	rec := bearerRequest(ts, http.MethodPost, "/api/v1/backups", admin.Token, nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created handlers.APIResponse[handlers.BackupResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = bearerRequest(ts, http.MethodGet, "/api/v1/backups/"+created.Data.Name+"/download", admin.Token, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `attachment; filename="`+created.Data.Name+`"`, rec.Header().Get("Content-Disposition"))
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestAPISnapshot_LoginRateLimitedRetryAfter(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	member := createBearerMember(t, ts)
	body := map[string]string{"email": member.Email, "password": "definitely-not-the-password", "device_name": "x"}

	var rec *httptest.ResponseRecorder
	for range 21 {
		rec = bearerRequest(ts, http.MethodPost, "/api/v1/auth/login", "", body)
	}

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, handlers.ErrCodeRateLimited, errorCode(t, rec))
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"))
}

func TestAPISnapshot_BodyContentType(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	sess := ts.Auth(t)

	for _, ct := range []string{"", "text/plain", "application/jsonp", "Application/JSON"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", strings.NewReader(`{}`))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		sess.Apply(req)
		rec := httptest.NewRecorder()

		ts.Server.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "Content-Type %q: %s", ct, rec.Body.String())
		var resp handlers.ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, handlers.ErrCodeInvalidRequest, resp.Error.Code)
	}
}
