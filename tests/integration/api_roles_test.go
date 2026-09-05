package integration_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

// Ролевые проверки API — вторая половина находки S-01
// (docs/specs/002-security-audit.md#s-01, задача 6 плана
// docs/plans/20260816-deployment-blockers.md).
//
// До задачи 6 группа /api/v1 была закрыта только RequireAPIAuth, то есть любой
// аутентифицированный пользователь — включая роль child — мог удалить чужого
// пользователя или категорию. В удалённом вебе те же действия были закрыты
// RequireAdmin и RequireAdminOrMember (финансовые разделы).
//
// TDD, красная фаза (`go test ./tests/integration -run TestAPIRoles`):
//
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/delete_user/member   actual: 204
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/delete_user/child    actual: 204
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/create_user/member   actual: 201
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/create_user/child    actual: 201
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/update_user/member   actual: 200
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/update_user/child    actual: 200
//	--- FAIL: TestAPIRoles_DestructiveRoutesRequireAdmin/delete_category/*    actual: 204
//	--- FAIL: TestAPIRoles_ChildHasNoAccessToFinanceRoutes/*                  actual: 200/201

// apiRoleCase — один разрушающий маршрут API и код успеха для админа.
type apiRoleCase struct {
	name    string
	method  string
	path    func(apiFixtures) string
	body    func(t *testing.T, f apiFixtures) string
	adminOK int
}

func TestAPIRoles_DestructiveRoutesRequireAdmin(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	adminAuth := testServer.Auth(t)
	_, memberAuth := testServer.AuthAs(t, user.RoleMember)
	_, childAuth := testServer.AuthAs(t, user.RoleChild)

	cases := []apiRoleCase{
		{
			name:    "deactivate user",
			method:  http.MethodPatch,
			path:    func(f apiFixtures) string { return "/api/v1/users/" + f.userID.String() },
			body:    func(_ *testing.T, _ apiFixtures) string { return `{"is_active":false}` },
			adminOK: http.StatusOK,
		},
		{
			name:    "set user password",
			method:  http.MethodPut,
			path:    func(f apiFixtures) string { return "/api/v1/users/" + f.userID.String() + "/password" },
			body:    func(_ *testing.T, _ apiFixtures) string { return `{"new_password":"Sup3rSecret!!"}` },
			adminOK: http.StatusNoContent,
		},
		{
			name:   "create user",
			method: http.MethodPost,
			path:   func(apiFixtures) string { return "/api/v1/users" },
			body: func(t *testing.T, _ apiFixtures) string {
				return string(mustJSON(t, map[string]any{
					"email":      fmt.Sprintf("role.probe+%s@example.com", uuid.New()),
					"password":   "Sup3rSecret!",
					"first_name": "Role",
					"last_name":  "Probe",
					"role":       "member",
				}))
			},
			adminOK: http.StatusCreated,
		},
		{
			name:    "update user",
			method:  http.MethodPut,
			path:    func(f apiFixtures) string { return "/api/v1/users/" + f.userID.String() },
			body:    func(_ *testing.T, _ apiFixtures) string { return `{"first_name":"Renamed"}` },
			adminOK: http.StatusOK,
		},
		{
			name:    "delete category",
			method:  http.MethodDelete,
			path:    func(f apiFixtures) string { return "/api/v1/categories/" + f.freeCategoryID.String() },
			adminOK: http.StatusNoContent,
		},
		{
			name:    "patch user role",
			method:  http.MethodPatch,
			path:    func(f apiFixtures) string { return "/api/v1/users/" + f.userID.String() },
			body:    func(_ *testing.T, _ apiFixtures) string { return `{"role":"child"}` },
			adminOK: http.StatusOK,
		},
		// Чтение карточки пользователя закрыто наравне с остальными
		// операциями над /users: в вебе весь раздел под RequireAdmin,
		// и API не должен отдавать member/child чужие email и роли.
		{
			name:    "get user",
			method:  http.MethodGet,
			path:    func(f apiFixtures) string { return "/api/v1/users/" + f.userID.String() },
			adminOK: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roleCases := []struct {
				role string
				auth *testhelpers.AuthSession
				want int
			}{
				{role: "admin", auth: adminAuth, want: tc.adminOK},
				{role: "member", auth: memberAuth, want: http.StatusForbidden},
				{role: "child", auth: childAuth, want: http.StatusForbidden},
			}

			for _, rc := range roleCases {
				t.Run(rc.role, func(t *testing.T) {
					// Свежие фикстуры на каждый прогон: успешная деактивация админом
					// действительно меняет запись.
					fixtures := createAPIFixtures(t, testServer)

					rec := doRoleRequest(t, testServer, tc, fixtures, rc.auth)

					assert.Equal(t, rc.want, rec.Code,
						"%s %s под ролью %s, тело ответа: %s",
						tc.method, tc.path(fixtures), rc.role, rec.Body.String())
				})
			}
		})
	}
}

// TestAPIRoles_ChildHasNoAccessToFinanceRoutes — разделы транзакций, бюджетов,
// категорий и отчётов закрыты для роли child (RequireRole admin|member).
func TestAPIRoles_ChildHasNoAccessToFinanceRoutes(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	adminAuth := testServer.Auth(t)
	_, childAuth := testServer.AuthAs(t, user.RoleChild)
	fixtures := createAPIFixtures(t, testServer)

	financePaths := []string{
		"/api/v1/categories",
		"/api/v1/transactions",
		"/api/v1/budgets",
		"/api/v1/reports",
		"/api/v1/stats/summary",
	}

	for _, path := range financePaths {
		t.Run("child "+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			childAuth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusForbidden, rec.Code,
				"роль child не имеет доступа к %s в вебе, значит и в API, тело: %s", path, rec.Body.String())
		})

		t.Run("admin "+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			adminAuth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())
		})
	}

	// Массовое удаление — тот же financeAccess, что и остальные операции с транзакциями.
	t.Run("child bulk delete transactions", func(t *testing.T) {
		body := mustJSON(t, map[string]any{"ids": []string{uuid.New().String()}})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/bulk-delete", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		childAuth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
	})

	// Запись под ролью child закрыта так же, как чтение.
	t.Run("child create transaction", func(t *testing.T) {
		body := mustJSON(t, map[string]any{
			"amount":      10.0,
			"type":        "expense",
			"description": "child write attempt",
			"category_id": fixtures.freeCategoryID,
			"date":        "2026-01-01T00:00:00Z",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		childAuth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
	})
}

// TestAPIRoles_MemberHasAccessToFinanceRoutes — обратная сторона предыдущего
// теста: финансовые разделы открыты роли member так же, как в вебе
// (RequireAdminOrMember). Без этой проверки сужение financeAccess до одного
// admin не уронило бы ни одного теста.
func TestAPIRoles_MemberHasAccessToFinanceRoutes(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	testServer.Auth(t)
	_, memberAuth := testServer.AuthAs(t, user.RoleMember)
	fixtures := createAPIFixtures(t, testServer)

	financePaths := []string{
		"/api/v1/categories",
		"/api/v1/transactions",
		"/api/v1/budgets",
		"/api/v1/reports",
		"/api/v1/stats/summary",
	}

	for _, path := range financePaths {
		t.Run("member "+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			memberAuth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code,
				"роль member работает с %s в вебе, значит и в API, тело: %s", path, rec.Body.String())
		})
	}

	// Запись под ролью member тоже разрешена, причём автором становится сам
	// member: user_id из тела запроса игнорируется (S-01).
	t.Run("member create transaction", func(t *testing.T) {
		body := mustJSON(t, map[string]any{
			"amount":      12.5,
			"type":        "expense",
			"description": "member write",
			"category_id": fixtures.categoryID,
			"date":        "2026-01-01T00:00:00Z",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		memberAuth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())
	})

	t.Run("member bulk delete transactions", func(t *testing.T) {
		body := mustJSON(t, map[string]any{"ids": []string{uuid.New().String()}})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/bulk-delete", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		memberAuth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())
	})

	// Раздел пользователей роли member по-прежнему закрыт, даже на чтение.
	t.Run("member reads user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+fixtures.userID.String(), nil)
		memberAuth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
	})
}

// TestAPIRoles_GetUnknownUser_AdminReaches404 — ролевой guard на GET
// /api/v1/users/:id не должен подменять ответ хендлера: у админа запрос
// доходит до него и отдаёт честный 404 на несуществующем id.
func TestAPIRoles_GetUnknownUser_AdminReaches404(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+uuid.New().String(), nil)
	adminAuth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "тело: %s", rec.Body.String())
}

// TestAPIRoles_ForbiddenResponseIsJSON — программный клиент обязан получить
// JSON-ошибку, а не HTML-страницу «Access denied», которую отдаёт веб-вариант
// RequireRole.
func TestAPIRoles_ForbiddenResponseIsJSON(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	testServer.Auth(t)
	_, childAuth := testServer.AuthAs(t, user.RoleChild)
	fixtures := createAPIFixtures(t, testServer)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+fixtures.userID.String(),
		strings.NewReader(`{"role":"admin"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	childAuth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Body.String(), `"FORBIDDEN"`)
}

// doRoleRequest выполняет запрос кейса под указанной сессией.
func doRoleRequest(
	t *testing.T,
	ts *testhelpers.TestServer,
	tc apiRoleCase,
	fixtures apiFixtures,
	auth *testhelpers.AuthSession,
) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if tc.body == nil {
		req = httptest.NewRequest(tc.method, tc.path(fixtures), nil)
	} else {
		req = httptest.NewRequest(tc.method, tc.path(fixtures), bytes.NewBufferString(tc.body(t, fixtures)))
		req.Header.Set("Content-Type", "application/json")
	}

	auth.Apply(req)
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	return rec
}
