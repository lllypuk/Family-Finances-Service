package integration_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// Регрессионный тест на находку U-02 (docs/specs/003-ui-ux-audit.md#u-02):
// на страницах-списках навигация пропадает, в шапке остаётся один логотип.
// Причина — рассогласование контракта данных: хендлеры отдают
// `map[string]any{"PageData": …}`, а шаблон проверяет `{{if .CurrentUser}}`
// в корне контекста. Go-шаблоны считают отсутствующий ключ map нулевым
// значением, поэтому блок меню молча выпадает без единой ошибки.
//
// Тест намеренно бьёт по реальному серверу с настоящими шаблонами
// (testhelpers.SetupHTTPServer, задача 1) и проверяет данные, которые собрал
// **хендлер**, а не собранные вручную в самом тесте — иначе он прошёл бы при
// любом контракте и регрессию не поймал бы.
//
// TDD, красная фаза (задача 12 плана docs/plans/20260816-deployment-blockers.md).
// Вывод первого прогона (`go test ./tests/integration -run
// TestWebPages_NavigationRendered`) — падают все четыре страницы:
//
//	--- FAIL: TestWebPages_NavigationRendered/transactions
//	--- FAIL: TestWebPages_NavigationRendered/categories
//	--- FAIL: TestWebPages_NavigationRendered/budgets
//	--- FAIL: TestWebPages_NavigationRendered/reports
//
// На /transactions, /categories и /budgets шапка схлопывается до логотипа —
// второй <ul> пуст, потому что `{{if .CurrentUser}}` ложно:
//
//	<nav class="container-fluid">
//	    <ul><li><a href="/" class="brand"><strong>💰 Семейный бюджет</strong></a></li></ul>
//	    <ul>
//	    </ul>
//	</nav>
//
// не содержит ни href="/transactions", ни href="/categories", ни href="/budgets",
// ни href="/reports", ни "John Doe", ни action="/logout".
//
// На /reports иначе: pages/reports/index.html рисует пункты меню безусловно,
// поэтому ссылки на месте, но пользовательской части шапки (имя + форма выхода)
// в шаблоне нет вовсе:
//
//	… does not contain "John Doe"
//	… does not contain "action=\"/logout\""
//
// То есть /reports закрывается не только контрактом данных (задача 15), но и
// правкой самого шаблона — блок `{{if .CurrentUser}}…{{end}}` там надо завести
// по образцу pages/transactions/index.html.
//
// Подтесты закрыты t.Skip, чтобы `make test` оставался зелёным до задач 13-15;
// каждая из них обязана снять свой skip (см. чек-листы в плане).
//
// Зелёная фаза для /transactions — задача 13: хендлеры транзакций отдают
// встроенный *PageData вместо `map[string]any{"PageData": …}`, skip снят.
// Оставшиеся skip'и (categories, budgets, reports) снимают задачи 14 и 15.

const (
	webPagesSkipCategories = "U-02: /categories — контракт данных хендлера, skip снимает задача 14"
	webPagesSkipBudgets    = "U-02: /budgets — контракт данных хендлера, skip снимает задача 14"
	webPagesSkipReports    = "U-02: /reports — контракт данных и шаблон шапки, skip снимает задача 15"
)

// navBlockRe вырезает первый блок <nav>…</nav> — шапку страницы.
var navBlockRe = regexp.MustCompile(`(?s)<nav[^>]*>.*?</nav>`)

// navExpectedLinks — пункты меню, которые обязаны быть в шапке любой страницы
// аутентифицированного пользователя с ролью admin.
var navExpectedLinks = []string{
	`href="/"`,
	`href="/transactions"`,
	`href="/categories"`,
	`href="/budgets"`,
	`href="/reports"`,
}

func TestWebPages_NavigationRendered(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	currentUser := testServer.AuthUser

	cases := []struct {
		name       string
		path       string
		skipReason string
	}{
		{name: "transactions", path: "/transactions"},
		{name: "categories", path: "/categories", skipReason: webPagesSkipCategories},
		{name: "budgets", path: "/budgets", skipReason: webPagesSkipBudgets},
		{name: "reports", path: "/reports", skipReason: webPagesSkipReports},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
			}

			nav := navBlock(t, fetchPage(t, testServer, auth, tc.path))

			for _, link := range navExpectedLinks {
				assert.Contains(t, nav, link,
					"на странице %s в шапке нет пункта меню %s", tc.path, link)
			}

			assert.Contains(t, nav, currentUser.FirstName+" "+currentUser.LastName,
				"на странице %s в шапке нет текущего пользователя — .CurrentUser не дошёл до шаблона", tc.path)
			assert.Contains(t, nav, `action="/logout"`,
				"на странице %s в шапке нет формы выхода", tc.path)
		})
	}
}

// fetchPage запрашивает HTML-страницу с сессией и возвращает тело ответа.
func fetchPage(t *testing.T, ts *testhelpers.TestServer, auth *testhelpers.AuthSession, path string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()

	ts.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "GET %s отдал %d, тело: %s", path, rec.Code, rec.Body.String())

	return rec.Body.String()
}

// navBlock возвращает разметку шапки страницы.
func navBlock(t *testing.T, body string) string {
	t.Helper()

	nav := navBlockRe.FindString(body)
	require.NotEmpty(t, nav, "на странице нет блока <nav>")

	return nav
}
