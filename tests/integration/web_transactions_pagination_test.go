package integration_test

import (
	"context"
	"html"
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

// Регрессия: шаблоны пагинации транзакций собирали хвост ссылки через
// `{{range $key, $value := .Filters}}`, но `.Filters` — это структура
// webModels.TransactionFilters, а не карта. text/template на таком range падает
// («range can't iterate over {...}»), и с буферизованным TemplateRenderer.Render
// это уже не обрезанный 200, а честный 500 на любом `GET /transactions?page=2`
// и на HTMX-партиале components/transaction_table.
//
// Затронутые места: pages/transactions/index.html:333,342 и
// components/transaction_table.html:98,110. Починка — метод
// TransactionFilters.QueryParams(), отдающий шаблону карту непустых фильтров.
//
// Тест ставит ровно те условия, при которых блок пагинации вообще рендерится:
// список непустой и HasPrev истинно, то есть page=2 при page_size=1 и двух
// записях в БД.
func TestWebTransactions_PaginatedPagesRender(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	createPaginationFixtures(t, testServer)

	cases := []struct {
		name string
		path string
	}{
		{
			name: "page",
			path: "/transactions?page=2&page_size=1&type=expense",
		},
		{
			name: "htmx_filter_partial",
			path: "/htmx/transactions/filter?page=2&page_size=1&type=expense",
		},
		{
			name: "htmx_list_partial",
			path: "/htmx/transactions/list?page=2&page_size=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			auth.Apply(req)
			rec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code,
				"GET %s отдал %d, тело: %s", tc.path, rec.Code, rec.Body.String())
			assert.NotContains(t, rec.Body.String(), "range can't iterate over",
				"шаблон %s всё ещё итерируется по структуре фильтров", tc.path)
		})
	}
}

// prevPageLinkRe вытаскивает href ссылки «Предыдущая» из блока пагинации.
var prevPageLinkRe = regexp.MustCompile(`href="\?page=[^"]*"`)

// TestWebTransactions_PaginationLinksCarryFilters проверяет, что активные
// фильтры действительно попадают в ссылки пагинации: если бы QueryParams()
// отдавал пустую карту, страница бы отрендерилась, но переход на соседнюю
// страницу сбрасывал бы фильтр.
func TestWebTransactions_PaginationLinksCarryFilters(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)
	createPaginationFixtures(t, testServer)

	req := httptest.NewRequest(http.MethodGet, "/transactions?page=2&page_size=1&type=expense", nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()

	testServer.Server.Echo().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

	prevLink := prevPageLinkRe.FindString(rec.Body.String())
	require.NotEmpty(t, prevLink, "на странице нет ссылки «Предыдущая»")
	assert.Contains(t, prevLink, "type=expense",
		"ссылка «Предыдущая» потеряла активный фильтр type: %s", prevLink)
}

var (
	// nextPageHrefRe вытаскивает href ссылки «Следующая» со страницы /transactions.
	nextPageHrefRe = regexp.MustCompile(`(?s)href="(\?page=[^"]*)"[^>]*>\s*Следующая`)
	// nextPageHxGetRe вытаскивает hx-get кнопки «Следующая» из components/transaction_table.
	nextPageHxGetRe = regexp.MustCompile(`(?s)hx-get="(/htmx/transactions/list\?[^"]*)"[^>]*>\s*Следующая`)
)

// TestWebTransactions_NextPageLinkReachesSecondPage — регрессия на «мёртвую»
// пагинацию: calculatePagination получал len(транзакций текущей страницы) как
// общее количество, хотя репозиторий уже применил LIMIT. TotalPages всегда был
// 1, HasNext всегда false — ссылка «Следующая» не рендерилась вовсе, и семья с
// количеством транзакций больше размера страницы не могла из UI попасть на
// вторую страницу.
//
// Тест ходит именно по отрендеренной ссылке, а не по URL, набранному руками:
// прежний тест запрашивал `?page=2` напрямую и потому дефект не ловил.
func TestWebTransactions_NextPageLinkReachesSecondPage(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)

	expenseCategory := createTestCategoryFor(t, testServer, category.TypeExpense)
	createDatedTransactions(t, testServer, expenseCategory.ID, transaction.TypeExpense,
		"Расход первой страницы A", "Расход первой страницы B", "Расход второй страницы C")

	firstPage := fetchPage(t, testServer, auth, "/transactions?page_size=2")
	assert.Contains(t, firstPage, "из 3", "блок пагинации не показывает реальное общее количество")

	match := nextPageHrefRe.FindStringSubmatch(firstPage)
	require.NotNil(t, match, "на первой странице нет ссылки «Следующая», тело: %s", firstPage)

	nextHref := html.UnescapeString(match[1])
	assert.Contains(t, nextHref, "page=2", "ссылка «Следующая» ведёт не на вторую страницу: %s", nextHref)

	secondPage := fetchPage(t, testServer, auth, "/transactions"+nextHref)
	assert.Contains(t, secondPage, "Расход второй страницы C", "вторая страница не отдала оставшуюся транзакцию")
	assert.NotContains(t, secondPage, "Расход первой страницы A", "вторая страница повторяет первую")
}

// TestWebTransactions_HTMXPaginationKeepsFilterAndTable — регрессия на HTMX-пагинацию:
// кнопки «Следующая»/«Предыдущая» свапают ответ /htmx/transactions/list в
// <section id="transactions-list">, но обработчик читал из запроса только
// page/page_size (терял фильтры из ссылки) и отдавал голые <tr>
// (components/transaction_rows), которые парсер HTML выбрасывал — таблица
// исчезала целиком.
func TestWebTransactions_HTMXPaginationKeepsFilterAndTable(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	auth := webAuth(t, testServer)

	expenseCategory := createTestCategoryFor(t, testServer, category.TypeExpense)
	createDatedTransactions(t, testServer, expenseCategory.ID, transaction.TypeExpense,
		"Расход первой страницы A", "Расход первой страницы B", "Расход второй страницы C")

	incomeCategory := createTestCategoryFor(t, testServer, category.TypeIncome)
	createDatedTransactions(t, testServer, incomeCategory.ID, transaction.TypeIncome, "Доход вне фильтра")

	firstPage := fetchPage(t, testServer, auth, "/htmx/transactions/filter?type=expense&page_size=2")
	match := nextPageHxGetRe.FindStringSubmatch(firstPage)
	require.NotNil(t, match, "в HTMX-фрагменте нет кнопки «Следующая», тело: %s", firstPage)

	nextURL := html.UnescapeString(match[1])
	require.Contains(t, nextURL, "type=expense", "кнопка «Следующая» потеряла активный фильтр: %s", nextURL)

	secondPage := fetchPage(t, testServer, auth, nextURL)
	assert.Contains(t, secondPage, "<table", "фрагмент пагинации пришёл без таблицы — HTMX-своп стёр бы список")
	assert.Contains(t, secondPage, "Расход второй страницы C", "вторая страница не отдала оставшуюся транзакцию")
	assert.NotContains(t, secondPage, "Доход вне фильтра", "переход на вторую страницу сбросил фильтр type=expense")
}

// createTestCategoryFor создаёт категорию нужного типа для семьи тестового сервера.
func createTestCategoryFor(t *testing.T, ts *testhelpers.TestServer, categoryType category.Type) *category.Category {
	t.Helper()

	testCategory := testhelpers.CreateTestCategory(ts.AuthFamily.ID, categoryType)
	require.NoError(t, ts.Repos.Category.Create(context.Background(), testCategory))

	return testCategory
}

// createDatedTransactions создаёт транзакции с указанными описаниями и убывающими
// датами, чтобы порядок выдачи (date DESC) совпадал с порядком аргументов.
func createDatedTransactions(
	t *testing.T,
	ts *testhelpers.TestServer,
	categoryID uuid.UUID,
	transactionType transaction.Type,
	descriptions ...string,
) {
	t.Helper()

	ctx := context.Background()
	for i, description := range descriptions {
		testTransaction := testhelpers.CreateTestTransaction(
			ts.AuthFamily.ID, ts.AuthUser.ID, categoryID, transactionType,
		)
		testTransaction.Description = description
		testTransaction.Date = time.Now().AddDate(0, 0, -i)
		require.NoError(t, ts.Repos.Transaction.Create(ctx, testTransaction))
	}
}

// createPaginationFixtures кладёт в БД две транзакции одной категории — этого
// достаточно, чтобы страница `page=2&page_size=1` была непустой и с HasPrev.
func createPaginationFixtures(t *testing.T, ts *testhelpers.TestServer) {
	t.Helper()

	ctx := context.Background()
	familyID := ts.AuthFamily.ID

	testCategory := testhelpers.CreateTestCategory(familyID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(ctx, testCategory))

	for range 2 {
		testTransaction := testhelpers.CreateTestTransaction(
			familyID, ts.AuthUser.ID, testCategory.ID, transaction.TypeExpense,
		)
		require.NoError(t, ts.Repos.Transaction.Create(ctx, testTransaction))
	}
}
