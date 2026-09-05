package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

// Регрессионный тест на рассинхрон «шаблон читает поле — хендлер его не отдаёт».
// tests/integration/web_pages_test.go покрывает только четыре страницы-списка,
// поэтому формы и карточки сущностей падали в проде незамеченными: у структуры
// (в отличие от map) обращение к отсутствующему полю — ошибка исполнения
// шаблона, то есть 500.
//
// Красная фаза (ревью фазы 1) — падали:
//
//	/categories/{id}/edit -> 500 (can't evaluate field CategoryID)
//	/budgets/{id}         -> 500 при перерасходе (FormattedOverspent)
//	/budgets/alerts       -> 500 (.Settings.WarningThreshold и ещё 6 ключей)
//
// Тест рендерит каждую страницу настоящими шаблонами и проверяет только код
// ответа и шапку — содержательные проверки живут в web_pages_test.go.
func TestWebPages_AllHTMLRoutesRender(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	fixtures := seedRenderFixtures(t, testServer)

	cases := []struct {
		name string
		path string
	}{
		{name: "dashboard", path: "/"},
		{name: "transactions_new", path: "/transactions/new"},
		{name: "transactions_edit", path: "/transactions/" + fixtures.transactionID.String() + "/edit"},
		{name: "categories_new", path: "/categories/new"},
		{name: "categories_show", path: "/categories/" + fixtures.categoryID.String()},
		{name: "categories_edit", path: "/categories/" + fixtures.categoryID.String() + "/edit"},
		{name: "budgets_new", path: "/budgets/new"},
		{name: "budgets_show", path: "/budgets/" + fixtures.budgetID.String()},
		{name: "budgets_show_overspent", path: "/budgets/" + fixtures.overspentBudgetID.String()},
		{name: "budgets_edit", path: "/budgets/" + fixtures.budgetID.String() + "/edit"},
		{name: "budgets_alerts", path: "/budgets/alerts"},
		{name: "reports_new", path: "/reports/new"},
		{name: "reports_show", path: "/reports/" + fixtures.reportID.String()},
		{name: "users", path: "/users"},
		{name: "users_new", path: "/users/new"},
		{name: "admin_users", path: "/admin/users"},
		{name: "admin_backup", path: "/admin/backup"},
	}

	currentUser := testServer.AuthUser

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fetchPage(t, testServer, auth, tc.path)
			// Не «есть блок <nav>», а именно шапка вошедшего пользователя:
			// проверка на существование <nav> проходила и на странице с
			// вырезанным меню (U-02 на /users/new).
			assertLoggedInNav(t, body, tc.path, currentUser.FirstName+" "+currentUser.LastName)
		})
	}
}

// TestWebPages_HTMXPagesCarryCSRFMeta — страницы с hx-delete/hx-post вне формы
// обязаны нести <meta name="csrf-token">: static/js/app.js вешает заголовок
// X-Csrf-Token на HTMX-запросы только из этого тега, а вне формы поле _token
// не отправляется. Без тега удаление категории, транзакции и пользователя
// отвечало 403 «CSRF token validation failed» в живом браузере.
func TestWebPages_HTMXPagesCarryCSRFMeta(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	fixtures := seedRenderFixtures(t, testServer)

	metaRe := regexp.MustCompile(`<meta name="csrf-token" content="([^"]*)"`)

	paths := []string{
		"/",
		"/transactions",
		"/transactions/new",
		"/transactions/" + fixtures.transactionID.String() + "/edit",
		"/categories",
		"/categories/new",
		"/categories/" + fixtures.categoryID.String() + "/edit",
		"/budgets",
		"/budgets/alerts",
		"/reports",
		"/reports/new",
		"/users",
		"/users/new",
		"/admin/users",
		"/admin/backup",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body := fetchPage(t, testServer, auth, path)

			match := metaRe.FindStringSubmatch(body)
			require.NotNil(t, match, "на странице %s нет <meta name=\"csrf-token\">", path)
			assert.NotEmpty(t, match[1], "на странице %s csrf-token пустой", path)
		})
	}
}

// hxRefRe вытаскивает адрес из hx-get/hx-post/hx-put/hx-patch/hx-delete.
var hxRefRe = regexp.MustCompile(`hx-(get|post|put|patch|delete)="(/[^"]*)"`)

// hxMethods переводит суффикс hx-атрибута в HTTP-метод.
var hxMethods = map[string]string{
	"get":    http.MethodGet,
	"post":   http.MethodPost,
	"put":    http.MethodPut,
	"patch":  http.MethodPatch,
	"delete": http.MethodDelete,
}

// TestWebPages_NoDeadLinks — та же проверка, что и для /login
// (login_links_test.go), но по всем страницам аутентифицированного
// пользователя: шапка вела на /profile и /family/settings, которых в роутере
// нет, а формы — на несуществующие обработчики.
//
// Помимо href/src проверяются hx-атрибуты: сквозной обход только по ссылкам
// пропустил `hx-post="/budgets/:id/archive"` на странице алертов,
// `hx-get="/categories/:id/progress"` на карточке категории и
// `hx-get="/htmx/dashboard/category-insights/export"` в блоке аналитики — три
// управляющих элемента на незарегистрированных маршрутах. Сверка идёт по
// методу, иначе кнопка, ведущая на существующий GET, считалась бы живой.
func TestWebPages_NoDeadLinks(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	fixtures := seedRenderFixtures(t, testServer)

	routes := registeredRoutes(testServer.Server.Echo())
	paths := []string{
		"/",
		"/transactions",
		"/transactions/new",
		"/transactions/" + fixtures.transactionID.String() + "/edit",
		"/categories",
		"/categories/new",
		"/categories/" + fixtures.categoryID.String(),
		"/categories/" + fixtures.categoryID.String() + "/edit",
		"/budgets",
		"/budgets/new",
		"/budgets/" + fixtures.budgetID.String(),
		"/budgets/" + fixtures.budgetID.String() + "/edit",
		"/budgets/alerts",
		"/reports",
		"/reports/new",
		"/users",
		"/users/new",
		"/admin/users",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body := fetchPage(t, testServer, auth, path)

			for _, ref := range localRefRe.FindAllStringSubmatch(body, -1) {
				link := stripQuery(ref[1])
				assert.True(t, matchesAnyRoute(link, routes[http.MethodGet]),
					"страница %s ссылается на %q, но такого GET-маршрута нет", path, link)
			}

			for _, ref := range hxRefRe.FindAllStringSubmatch(body, -1) {
				method := hxMethods[ref[1]]
				target := stripQuery(ref[2])
				assert.True(t, matchesAnyRoute(target, routes[method]),
					"страница %s вешает hx-%s на %q, но маршрута %s %s нет",
					path, ref[1], target, method, target)
			}
		})
	}
}

// TestWebPages_LogoutFormCarriesCSRFToken — форма выхода в шапке обязана нести
// непустой _token: иначе единственный способ выйти из системы отвергается
// CSRFProtection с 403 (так было на дашборде).
func TestWebPages_LogoutFormCarriesCSRFToken(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)

	paths := []string{
		"/", "/transactions", "/categories", "/budgets", "/reports",
		// Эти три страницы несли собственную копию шапки: на /users/new она
		// была вовсе без пользовательского меню и без формы выхода (U-02).
		"/users", "/users/new", "/admin/users", "/admin/backup",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			nav := navBlock(t, fetchPage(t, testServer, auth, path))

			match := logoutTokenRe.FindStringSubmatch(nav)
			require.NotNil(t, match, "в шапке страницы %s нет поля _token", path)
			assert.NotEmpty(t, match[1], "на странице %s форма выхода несёт пустой _token", path)
		})
	}
}

// renderFixtures — идентификаторы сущностей, на которых рендерятся карточки и
// формы редактирования.
type renderFixtures struct {
	categoryID        uuid.UUID
	transactionID     uuid.UUID
	budgetID          uuid.UUID
	overspentBudgetID uuid.UUID
	reportID          uuid.UUID
}

// seedRenderFixtures наполняет БД минимальным набором данных для страниц выше.
func seedRenderFixtures(t *testing.T, ts *testhelpers.TestServer) renderFixtures {
	t.Helper()

	ctx := context.Background()
	familyID := ts.AuthFamily.ID
	userID := ts.AuthUser.ID

	expenseCategory := testhelpers.CreateTestCategory(familyID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(ctx, expenseCategory))

	overspentCategory := testhelpers.CreateTestCategory(familyID, category.TypeExpense)
	overspentCategory.Name = "Overspent Category"
	require.NoError(t, ts.Repos.Category.Create(ctx, overspentCategory))

	tx := testhelpers.CreateTestTransaction(familyID, userID, expenseCategory.ID, transaction.TypeExpense)
	require.NoError(t, ts.Repos.Transaction.Create(ctx, tx))

	budgetEntity := testhelpers.CreateTestBudget(familyID, expenseCategory.ID)
	require.NoError(t, ts.Repos.Budget.Create(ctx, budgetEntity))

	// Бюджет с перерасходом: ветка `{{if .Budget.IsOverBudget}}` в
	// pages/budgets/show.html иначе не исполняется.
	overspent := testhelpers.CreateTestBudget(familyID, overspentCategory.ID)
	overspent.Name = "Overspent Budget"
	overspent.Spent = overspent.Amount * 2
	require.NoError(t, ts.Repos.Budget.Create(ctx, overspent))

	reportEntity := testhelpers.CreateTestReport(familyID, userID)
	require.NoError(t, ts.Repos.Report.Create(ctx, reportEntity))

	return renderFixtures{
		categoryID:        expenseCategory.ID,
		transactionID:     tx.ID,
		budgetID:          budgetEntity.ID,
		overspentBudgetID: overspent.ID,
		reportID:          reportEntity.ID,
	}
}

// TestWebPages_HTMXPartialsRender — HTMX-эндпоинты отдают частичные шаблоны,
// и их имена так же расходились с реальными define-блоками
// (components/category_list и components/category_select не существовали вовсе).
func TestWebPages_HTMXPartialsRender(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	fixtures := seedRenderFixtures(t, testServer)

	paths := []string{
		"/htmx/dashboard/stats",
		"/htmx/dashboard/filter",
		"/htmx/dashboard/category-insights",
		"/htmx/transactions/recent",
		"/htmx/transactions/filter",
		"/htmx/transactions/list",
		"/htmx/budgets/overview",
		"/htmx/budgets/" + fixtures.budgetID.String() + "/progress",
		"/htmx/categories/search",
		"/htmx/categories/search?name=Test",
		"/htmx/categories/select",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Hx-Request", "true")
			auth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code, "GET %s отдал %d: %s", path, rec.Code, rec.Body.String())
		})
	}
}

// TestWebPages_FormValidationErrorsRerender — перерисовка формы после неудачной
// валидации. Хендлеры рендерят её отдельным путём (renderXxxFormWithErrors), и
// именно там шаблон edit.html читает поля, которых в структуре ошибок не было.
func TestWebPages_FormValidationErrorsRerender(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	fixtures := seedRenderFixtures(t, testServer)

	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "transaction_edit",
			path: "/transactions/" + fixtures.transactionID.String(),
			body: "amount=&type=expense&description=&category_id=&date=",
		},
		{
			name: "category_edit",
			path: "/categories/" + fixtures.categoryID.String(),
			body: "name=&type=expense",
		},
		{
			name: "budget_edit",
			path: "/budgets/" + fixtures.budgetID.String(),
			body: "name=&amount=&period=&start_date=&end_date=",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, tc.path, strings.NewReader(tc.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			auth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			assert.Less(t, rec.Code, http.StatusInternalServerError,
				"PUT %s с невалидными данными отдал %d, тело: %s", tc.path, rec.Code, rec.Body.String())
		})
	}
}
