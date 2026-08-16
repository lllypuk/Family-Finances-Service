package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

// Регрессионный тест на находку S-01 (docs/specs/002-security-audit.md#s-01):
// группа /api/v1 регистрируется без auth middleware, поэтому анонимный клиент
// получает полный CRUD над данными семьи.
//
// TDD, красная фаза (задача 2 плана docs/plans/20260816-deployment-blockers.md).
// Вывод первого прогона (`go test ./tests/integration -run
// TestAPIAuth_AnonymousRequestsRejected`, всюду expected 401) — дыра
// воспроизведена целиком, 15 упавших подтестов:
//
//	--- FAIL: .../GetWithoutSession/users                       actual: 200
//	--- FAIL: .../GetWithoutSession/categories                  actual: 200
//	--- FAIL: .../GetWithoutSession/transactions                actual: 200
//	--- FAIL: .../GetWithoutSession/budgets                     actual: 200
//	--- FAIL: .../GetWithoutSession/reports                     actual: 200
//	--- FAIL: .../WriteWithAnonymousCSRFToken/create_transaction actual: 201
//	--- FAIL: .../WriteWithAnonymousCSRFToken/create_category    actual: 201
//	--- FAIL: .../WriteWithAnonymousCSRFToken/create_budget      actual: 201
//	--- FAIL: .../WriteWithAnonymousCSRFToken/update_transaction actual: 200
//	--- FAIL: .../WriteWithAnonymousCSRFToken/delete_transaction actual: 204
//	--- FAIL: .../WriteWithAnonymousCSRFToken/delete_category    actual: 204
//	--- FAIL: .../WriteWithAnonymousCSRFToken/delete_budget      actual: 204
//	--- FAIL: .../WriteWithAnonymousCSRFToken/delete_report      actual: 204
//	--- FAIL: .../WriteWithAnonymousCSRFToken/delete_user        actual: 204
//	--- FAIL: .../AuditBypassScenario                            actual: 201
//
// AuditBypassScenario вернул созданную запись целиком, то есть анонимный клиент
// не просто прошёл, а записал транзакцию от имени чужого пользователя:
//
//	{"data":{"id":"c68a0cb6-…","amount":13.37,"type":"expense",
//	 "description":"anonymous injection","user_id":"184701d2-…"}, …}
//
// Единственный подтест, который проходил и в красной фазе, —
// WriteWithoutCSRFToken: 403 отдаёт глобальный CSRFProtection, а не авторизация.
//
// Тест снимается со `t.Skip` в задаче 4, которая вешает RequireAPIAuth на группу.
const apiAuthSkipReason = "S-01: RequireAPIAuth ещё не подключён к /api/v1 — снять skip в задаче 4 плана"

// csrfFormTokenRe вытаскивает CSRF-токен из скрытого поля формы входа
// (internal/web/templates/pages/login.html).
var csrfFormTokenRe = regexp.MustCompile(`name="_token"\s+value="([^"]+)"`)

// anonymousSession — сессия без пользователя: cookie, выданная на GET /login,
// и лежащий в ней CSRF-токен. Ровно тот набор, которым в аудите обходили защиту.
type anonymousSession struct {
	cookie *http.Cookie
	token  string
}

func (s *anonymousSession) apply(req *http.Request) {
	req.AddCookie(s.cookie)
	req.Header.Set("X-Csrf-Token", s.token)
}

// apiFixtures — идентификаторы существующих записей, чтобы анонимные
// GET/PUT/DELETE били по реальным данным, а не получали 404.
type apiFixtures struct {
	userID        uuid.UUID
	categoryID    uuid.UUID
	transactionID uuid.UUID
	budgetID      uuid.UUID
	reportID      uuid.UUID

	// freeCategoryID — категория без бюджета: BudgetService отвергает второй
	// пересекающийся бюджет на ту же категорию, а нам нужен честный 201.
	freeCategoryID uuid.UUID
}

func TestAPIAuth_AnonymousRequestsRejected(t *testing.T) {
	t.Skip(apiAuthSkipReason)

	testServer := testhelpers.SetupHTTPServer(t)

	// Семья и админ нужны только для того, чтобы RequireSetup не уводил всё на /setup.
	testServer.Auth(t)

	t.Run("GetWithoutSession", func(t *testing.T) {
		fixtures := createAPIFixtures(t, testServer)

		readCases := []struct {
			name string
			path string
		}{
			{name: "users", path: "/api/v1/users/" + fixtures.userID.String()},
			{name: "categories", path: "/api/v1/categories"},
			{name: "transactions", path: "/api/v1/transactions"},
			{name: "budgets", path: "/api/v1/budgets"},
			{name: "reports", path: "/api/v1/reports"},
		}

		for _, tc := range readCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, tc.path, nil)
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusUnauthorized, rec.Code,
					"анонимный GET %s обязан получить 401", tc.path)
			})
		}
	})

	t.Run("WriteWithAnonymousCSRFToken", func(t *testing.T) {
		for _, tc := range anonymousWriteCases(t, testServer) {
			t.Run(tc.name, func(t *testing.T) {
				anon := newAnonymousSession(t, testServer)

				req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
				req.Header.Set("Content-Type", "application/json")
				anon.apply(req)
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusUnauthorized, rec.Code,
					"анонимный %s %s с валидным CSRF-токеном обязан получить 401", tc.method, tc.path)
			})
		}
	})

	// Без токена отвечает глобальный CSRFProtection — он регистрируется через
	// e.Use и отрабатывает раньше group-middleware группы /api/v1, поэтому здесь
	// ожидается 403, а не 401 (см. Technical Details плана).
	t.Run("WriteWithoutCSRFToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	// Дословный сценарий обхода из аудита: анонимно взять CSRF-токен со страницы
	// входа и создать транзакцию от имени произвольного пользователя.
	t.Run("AuditBypassScenario", func(t *testing.T) {
		fixtures := createAPIFixtures(t, testServer)
		anon := newAnonymousSession(t, testServer)

		body := mustJSON(t, map[string]any{
			"amount":      13.37,
			"type":        "expense",
			"description": "anonymous injection",
			"category_id": fixtures.categoryID,
			"user_id":     fixtures.userID,
			"date":        time.Now(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		anon.apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code,
			"анонимный клиент создал транзакцию, тело ответа: %s", rec.Body.String())
	})
}

type apiWriteCase struct {
	name   string
	method string
	path   string
	body   string
}

// anonymousWriteCases готовит по отдельному набору записей на каждый кейс:
// пока дыра открыта, DELETE действительно удаляет, и общие фикстуры дали бы
// 404 вместо честного «запрос прошёл».
func anonymousWriteCases(t *testing.T, ts *testhelpers.TestServer) []apiWriteCase {
	t.Helper()

	createTx := createAPIFixtures(t, ts)
	createCat := createAPIFixtures(t, ts)
	createBudget := createAPIFixtures(t, ts)
	updateTx := createAPIFixtures(t, ts)
	deleteTx := createAPIFixtures(t, ts)
	deleteCat := createAPIFixtures(t, ts)
	deleteBudget := createAPIFixtures(t, ts)
	deleteReport := createAPIFixtures(t, ts)
	deleteUser := createAPIFixtures(t, ts)

	transactionBody := func(f apiFixtures) string {
		return string(mustJSON(t, map[string]any{
			"amount":      42.0,
			"type":        "expense",
			"description": "anonymous write",
			"category_id": f.categoryID,
			"user_id":     f.userID,
			"date":        time.Now(),
		}))
	}

	return []apiWriteCase{
		{
			name:   "create transaction",
			method: http.MethodPost,
			path:   "/api/v1/transactions",
			body:   transactionBody(createTx),
		},
		{
			name:   "create category",
			method: http.MethodPost,
			path:   "/api/v1/categories",
			body: string(mustJSON(t, map[string]any{
				"name":      "Anonymous category",
				"type":      "expense",
				"color":     "#ff0000",
				"icon":      "shopping",
				"parent_id": createCat.categoryID,
			})),
		},
		{
			name:   "create budget",
			method: http.MethodPost,
			path:   "/api/v1/budgets",
			body: string(mustJSON(t, map[string]any{
				"name":        "Anonymous budget",
				"amount":      500.0,
				"period":      "monthly",
				"category_id": createBudget.freeCategoryID,
				"start_date":  time.Now(),
				"end_date":    time.Now().AddDate(0, 1, 0),
			})),
		},
		{
			name:   "update transaction",
			method: http.MethodPut,
			path:   "/api/v1/transactions/" + updateTx.transactionID.String(),
			body:   `{"description":"hijacked by anonymous"}`,
		},
		{
			name:   "delete transaction",
			method: http.MethodDelete,
			path:   "/api/v1/transactions/" + deleteTx.transactionID.String(),
		},
		{
			name:   "delete category",
			method: http.MethodDelete,
			path:   "/api/v1/categories/" + deleteCat.categoryID.String(),
		},
		{
			name:   "delete budget",
			method: http.MethodDelete,
			path:   "/api/v1/budgets/" + deleteBudget.budgetID.String(),
		},
		{
			name:   "delete report",
			method: http.MethodDelete,
			path:   "/api/v1/reports/" + deleteReport.reportID.String(),
		},
		{
			name:   "delete user",
			method: http.MethodDelete,
			path:   "/api/v1/users/" + deleteUser.userID.String(),
		},
	}
}

// newAnonymousSession повторяет шаг аудита: GET /login отдаёт cookie сессии и
// CSRF-токен в форме, при этом пользователь в сессию не записывается.
func newAnonymousSession(t *testing.T, ts *testhelpers.TestServer) *anonymousSession {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "страница входа недоступна")

	cookies := (&http.Response{Header: rec.Header()}).Cookies()
	require.NotEmpty(t, cookies, "GET /login не выдал cookie сессии")

	match := csrfFormTokenRe.FindStringSubmatch(rec.Body.String())
	require.Len(t, match, 2, "CSRF-токен не найден в форме входа")

	return &anonymousSession{cookie: cookies[0], token: match[1]}
}

// createAPIFixtures создаёт по одной записи каждого типа в тестовой семье.
func createAPIFixtures(t *testing.T, ts *testhelpers.TestServer) apiFixtures {
	t.Helper()

	ctx := context.Background()
	familyID := ts.AuthFamily.ID

	testUser := testhelpers.CreateTestUser(familyID)
	require.NoError(t, ts.Repos.User.Create(ctx, testUser))

	testCategory := testhelpers.CreateTestCategory(familyID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(ctx, testCategory))

	testTransaction := testhelpers.CreateTestTransaction(
		familyID, testUser.ID, testCategory.ID, transaction.TypeExpense,
	)
	require.NoError(t, ts.Repos.Transaction.Create(ctx, testTransaction))

	testBudget := testhelpers.CreateTestBudget(familyID, testCategory.ID)
	require.NoError(t, ts.Repos.Budget.Create(ctx, testBudget))

	testReport := testhelpers.CreateTestReport(familyID, testUser.ID)
	require.NoError(t, ts.Repos.Report.Create(ctx, testReport))

	freeCategory := testhelpers.CreateTestCategory(familyID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(ctx, freeCategory))

	return apiFixtures{
		userID:         testUser.ID,
		categoryID:     testCategory.ID,
		transactionID:  testTransaction.ID,
		budgetID:       testBudget.ID,
		reportID:       testReport.ID,
		freeCategoryID: freeCategory.ID,
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err)

	return data
}
