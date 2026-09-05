package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
// Задача 4 повесила RequireAPIAuth на группу /api/v1, skip снят — тест обязан
// проходить. Зелёный прогон означает, что S-01 закрыт для анонимного клиента.

// apiV1Prefix — префикс группы API, по которому маршруты отбираются из роутера.
const apiV1Prefix = "/api/v1"

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
	testServer := testhelpers.SetupHTTPServer(t)

	// Семья и админ нужны только для того, чтобы RequireSetup не уводил всё на /setup.
	testServer.Auth(t)

	// Список маршрутов не поддерживается руками: он берётся из роутера Echo,
	// поэтому маршрут, зарегистрированный мимо защищённой группы, немедленно
	// уронит тест, а не тихо останется без проверки.
	t.Run("EveryAPIRouteWithoutSession", func(t *testing.T) {
		fixtures := createAPIFixtures(t, testServer)
		routes := apiV1Routes(testServer.Server.Echo(), fixtures)
		require.NotEmpty(t, routes, "в роутере не нашлось ни одного маршрута /api/v1")

		for _, route := range routes {
			t.Run(route.method+" "+route.path, func(t *testing.T) {
				// Анонимная сессия несёт валидный CSRF-токен: без него запись
				// отвергает CSRFProtection и до авторизации дело не доходит.
				anon := newAnonymousSession(t, testServer)

				req := httptest.NewRequest(route.method, route.path, bytes.NewBufferString("{}"))
				req.Header.Set("Content-Type", "application/json")
				anon.apply(req)
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusUnauthorized, rec.Code,
					"анонимный %s %s обязан получить 401, тело: %s", route.method, route.path, rec.Body.String())
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

// TestAPIAuth_AuthenticatedRequestsAllowed — обратная сторона задачи 4:
// RequireAPIAuth не должен ломать легитимный доступ. Заодно проверяется, что
// middleware повешено именно на группу /api/v1: /health и веб-маршруты не задеты.
func TestAPIAuth_AuthenticatedRequestsAllowed(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	fixtures := createAPIFixtures(t, testServer)

	t.Run("GetWithSession", func(t *testing.T) {
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
				auth.Apply(req)
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusOK, rec.Code,
					"аутентифицированный GET %s обязан получить 200, тело: %s", tc.path, rec.Body.String())
			})
		}
	})

	t.Run("WriteWithSession", func(t *testing.T) {
		body := mustJSON(t, map[string]any{
			"amount":      21.0,
			"type":        "expense",
			"description": "authenticated write",
			"category_id": fixtures.categoryID,
			"user_id":     fixtures.userID,
			"date":        time.Now(),
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		auth.Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code, "тело ответа: %s", rec.Body.String())
	})

	// /health зарегистрирован вне группы /api/v1 и обязан остаться публичным:
	// на него смотрит внешний мониторинг и healthcheck контейнера.
	t.Run("HealthStaysPublic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	// Веб-маршруты живут своей аутентификацией: у анонимного клиента /login
	// открывается, а защищённая страница уводит редиректом, а не отдаёт 401 JSON.
	t.Run("WebRoutesUntouched", func(t *testing.T) {
		loginReq := httptest.NewRequest(http.MethodGet, "/login", nil)
		loginRec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(loginRec, loginReq)
		assert.Equal(t, http.StatusOK, loginRec.Code)

		dashReq := httptest.NewRequest(http.MethodGet, "/", nil)
		dashRec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(dashRec, dashReq)
		assert.Equal(t, http.StatusFound, dashRec.Code)
		assert.Equal(t, "/login", dashRec.Header().Get("Location"))
	})
}

// TestAPIAuth_BodyUserIDIgnored — вторая половина S-01 на полном стеке:
// аутентифицированный клиент подставляет в тело чужой user_id, а запись всё
// равно создаётся от имени владельца сессии. Поле user_id из
// handlers.CreateTransactionRequest удалено (задача 5), поэтому лишний ключ в
// JSON просто игнорируется при биндинге.
func TestAPIAuth_BodyUserIDIgnored(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	fixtures := createAPIFixtures(t, testServer)

	body := mustJSON(t, map[string]any{
		"amount":      99.0,
		"type":        "expense",
		"description": "impersonation attempt",
		"category_id": fixtures.freeCategoryID,
		"user_id":     fixtures.userID,
		"date":        time.Now(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	auth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "тело ответа: %s", rec.Body.String())

	var response struct {
		Data struct {
			ID     uuid.UUID `json:"id"`
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	assert.Equal(t, testServer.AuthUser.ID, response.Data.UserID,
		"автор записи обязан совпадать с владельцем сессии")
	assert.NotEqual(t, fixtures.userID, response.Data.UserID,
		"user_id из тела запроса обязан игнорироваться")

	// Проверяем не только ответ, но и то, что легло в БД.
	stored, err := testServer.Repos.Transaction.GetByID(context.Background(), response.Data.ID)
	require.NoError(t, err)
	assert.Equal(t, testServer.AuthUser.ID, stored.UserID)
}

// apiRoute — конкретный запрос, собранный из шаблона маршрута Echo.
type apiRoute struct {
	method string
	path   string
}

// apiV1Routes перечисляет все зарегистрированные маршруты группы /api/v1,
// подставляя вместо `:param` идентификаторы существующих записей (техника взята
// из tests/integration/login_links_test.go, только в обратную сторону: там
// шаблон превращается в регулярное выражение, здесь — в конкретный путь).
// Wildcard-маршруты (`/*`), которые Echo сам вешает на группу с middleware,
// пропускаются: за ними стоит NotFoundHandler, а не хендлер приложения.
func apiV1Routes(e *echo.Echo, fixtures apiFixtures) []apiRoute {
	seen := make(map[apiRoute]struct{})
	routes := make([]apiRoute, 0)

	for _, route := range e.Routes() {
		if !strings.HasPrefix(route.Path, apiV1Prefix) || strings.Contains(route.Path, "*") {
			continue
		}
		// Псевдо-метод Echo для «маршрут не найден» — не запрос приложения.
		if route.Method == echo.RouteNotFound {
			continue
		}
		// Единственный публичный маршрут API: логин выдаёт токен анонимному клиенту.
		if route.Method == http.MethodPost && route.Path == apiV1Prefix+"/auth/login" {
			continue
		}

		concrete := apiRoute{method: route.Method, path: concreteAPIPath(route.Path, fixtures)}
		if _, duplicate := seen[concrete]; duplicate {
			continue
		}

		seen[concrete] = struct{}{}
		routes = append(routes, concrete)
	}

	return routes
}

// concreteAPIPath подставляет в шаблон маршрута реальные идентификаторы: тогда
// пропущенная авторизация проявится как 200/204 на существующей записи, а не
// как 404 на выдуманной.
func concreteAPIPath(pattern string, fixtures apiFixtures) string {
	resourceIDs := map[string]uuid.UUID{
		"users":        fixtures.userID,
		"categories":   fixtures.categoryID,
		"transactions": fixtures.transactionID,
		"budgets":      fixtures.budgetID,
		"reports":      fixtures.reportID,
	}

	segments := strings.Split(pattern, "/")
	resource := ""

	for i, segment := range segments {
		if !strings.HasPrefix(segment, ":") {
			resource = segment
			continue
		}

		id, ok := resourceIDs[resource]
		if !ok {
			id = uuid.New()
		}
		segments[i] = id.String()
	}

	return strings.Join(segments, "/")
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
