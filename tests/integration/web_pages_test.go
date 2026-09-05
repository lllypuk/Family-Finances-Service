package integration_test

import (
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
// Зелёная фаза для /categories и /budgets — задача 14, там же сняты их skip'и.
// Зелёная фаза для /reports — задача 15: контракт данных в reports.go плюс
// блок `{{if .CurrentUser}}` в pages/reports/index.html, которого там не было
// вовсе. Skip'ов в тесте не осталось.
//
// Заодно проверяется заголовок страницы (U-05,
// docs/specs/003-ui-ux-audit.md#u-05): интерфейс русскоязычный, английские
// «Transactions»/«Budgets»/«Categories»/«Reports» в <title> — регрессия.

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
	auth := webAuth(t, testServer)
	currentUser := testServer.AuthUser

	cases := []struct {
		name  string
		path  string
		title string
	}{
		{name: "transactions", path: "/transactions", title: "Транзакции"},
		{name: "categories", path: "/categories", title: "Категории"},
		{name: "budgets", path: "/budgets", title: "Бюджеты"},
		{name: "reports", path: "/reports", title: "Отчёты"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := fetchPage(t, testServer, auth, tc.path)

			assert.Contains(t, body, "<title>"+tc.title+" - Семейный бюджет</title>",
				"на странице %s заголовок не на русском (U-05)", tc.path)

			assertLoggedInNav(t, body, tc.path, currentUser.FirstName+" "+currentUser.LastName)
		})
	}
}

// logoutTokenRe достаёт значение скрытого поля _token из формы выхода.
var logoutTokenRe = regexp.MustCompile(`(?s)name="_token"[^>]*value="([^"]*)"`)

// assertLoggedInNav проверяет, что в шапке страницы действительно шапка
// вошедшего администратора.
//
// Проверять «блок <nav> непустой» бессмысленно: у страницы с вырезанным меню
// (ровно симптом U-02) блок <nav> тоже есть — там остаётся один логотип.
// Именно поэтому pages/users/new.html с захардкоженной шапкой без
// `{{if .CurrentUser}}`, без пользовательского меню и без формы выхода
// проходил регрессионный тест.
func assertLoggedInNav(t *testing.T, body, path, userName string) {
	t.Helper()

	nav := navBlock(t, body)

	for _, link := range navExpectedLinks {
		assert.Contains(t, nav, link,
			"на странице %s в шапке нет пункта меню %s", path, link)
	}

	assert.Contains(t, nav, userName,
		"на странице %s в шапке нет текущего пользователя — .CurrentUser не дошёл до шаблона", path)
	assert.Contains(t, nav, `action="/logout"`,
		"на странице %s в шапке нет формы выхода", path)

	match := logoutTokenRe.FindStringSubmatch(nav)
	require.NotNil(t, match, "в шапке страницы %s нет поля _token", path)
	assert.NotEmpty(t, match[1], "на странице %s форма выхода несёт пустой _token", path)
}

// navBlock возвращает разметку шапки страницы.
func navBlock(t *testing.T, body string) string {
	t.Helper()

	nav := navBlockRe.FindString(body)
	require.NotEmpty(t, nav, "на странице нет блока <nav>")

	return nav
}
