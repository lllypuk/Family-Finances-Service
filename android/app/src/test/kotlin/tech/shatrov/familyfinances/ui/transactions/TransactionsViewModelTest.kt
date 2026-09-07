package tech.shatrov.familyfinances.ui.transactions

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.coroutines.withContext
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.TRANSACTIONS_EMPTY
import tech.shatrov.familyfinances.TRANSACTIONS_PAGE_1
import tech.shatrov.familyfinances.TRANSACTIONS_PAGE_2
import tech.shatrov.familyfinances.USERS_OK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.ui.UiError
import java.time.LocalDate
import java.util.UUID
import java.util.concurrent.TimeUnit

/** Задержка ответа в тесте гонки: столько сервер держит страницу прошлого запроса. */
private const val STALE_PAGE_DELAY_MS = 300L

/** Список транзакций: страницы, фильтры в query и имена из справочников. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class TransactionsViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: TransactionsViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    /** Дата модели: тест переставляет её, чтобы проверить переход через полночь. */
    private var today = LocalDate.parse("2026-09-15")

    private fun createModel(role: Role = Role.admin) {
        model = TransactionsViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            testSession(role),
            { today },
        )
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun settle(): TransactionsUiState = model.state.first { it != TransactionsUiState.Loading }

    private suspend fun settleMore(): TransactionsUiState.Ready =
        model.state.first { it is TransactionsUiState.Ready && !it.loadingMore } as TransactionsUiState.Ready

    private fun enqueueFirstPage(body: String = TRANSACTIONS_PAGE_1) {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, USERS_OK)
        server.enqueueJson(200, body)
    }

    private fun jsonResponse(
        body: String,
        delayMs: Long = 0,
    ): MockResponse = MockResponse.Builder()
        .code(200)
        .body(body)
        .setHeader("Content-Type", "application/json")
        .bodyDelay(delayMs, TimeUnit.MILLISECONDS)
        .build()

    /** Пропускает справочники и предыдущие страницы: интересен всегда последний запрос. */
    private fun lastRequestUrl(count: Int) = (1..count).map { server.takeRequest() }.last().url

    @Test
    fun firstPageIsGroupedByDateWithNames() = runTest {
        enqueueFirstPage()

        createModel()
        val state = settle() as TransactionsUiState.Ready

        assertEquals(1, state.groups.size)
        assertEquals(LocalDate.parse("2026-09-07"), state.groups[0].date)
        val rows = state.groups[0].rows
        assertEquals(listOf("Продукты", "Продукты"), rows.map { it.categoryName })
        assertTrue(rows[0].isMine)
        assertEquals("Член", rows[1].authorName)
        assertTrue(state.hasMore)
        assertEquals("RUB", state.currency)

        val url = lastRequestUrl(3)
        assertEquals("/api/v1/transactions", url.encodedPath)
        assertEquals("0", url.queryParameter("offset"))
    }

    @Test
    fun nextPageIsAppended() = runTest {
        enqueueFirstPage()
        createModel()
        settle()

        server.enqueueJson(200, TRANSACTIONS_PAGE_2)
        model.loadMore()
        val state = settleMore()

        assertEquals(2, state.groups.size)
        assertEquals(LocalDate.parse("2026-09-06"), state.groups[1].date)
        assertFalse(state.hasMore)
        // Справочники за второй страницей не перезапрашиваются: их всего один запрос на экран.
        assertEquals("2", lastRequestUrl(4).queryParameter("offset"))
    }

    // Сосед вставил запись перед окном: страница приходит той же, и total её уже считает.
    @Test
    fun pageWithoutNewRowsStopsPaging() = runTest {
        enqueueFirstPage()
        createModel()
        settle()

        server.enqueueJson(200, TRANSACTIONS_PAGE_1)
        model.loadMore()
        val state = settleMore()

        assertEquals(2, state.groups[0].rows.size)
        assertFalse(state.hasMore)
    }

    @Test
    fun emptyAnswerIsNotAFailure() = runTest {
        enqueueFirstPage(TRANSACTIONS_EMPTY)

        createModel()
        val state = settle() as TransactionsUiState.Ready

        assertTrue(state.isEmpty)
        assertFalse(state.hasMore)
    }

    @Test
    fun failureOnSecondPageKeepsLoadedRows() = runTest {
        enqueueFirstPage()
        createModel()
        settle()

        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")
        model.loadMore()
        val state = settleMore()

        assertEquals(UiError.Server("всё сломалось"), state.moreError)
        assertEquals(2, state.groups[0].rows.size)
        assertTrue(state.hasMore)
    }

    @Test
    fun failureOnFirstPageReplacesScreen() = runTest {
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")

        createModel()

        assertEquals(TransactionsUiState.Failure(UiError.Server("всё сломалось")), settle())
    }

    // Полночь между страницами: границы «этого месяца» уехали на новый месяц, а offset
    // прошлого окна пропустил бы его начало и склеил два периода в один список.
    @Test
    fun monthRolloverRestartsPagingFromZero() = runTest {
        today = LocalDate.parse("2026-09-30")
        enqueueFirstPage()
        createModel()
        settle()
        server.enqueueJson(200, TRANSACTIONS_PAGE_1)
        model.onFiltersChange(TransactionFilters(period = TransactionPeriod.THIS_MONTH))
        settle()

        today = LocalDate.parse("2026-10-01")
        server.enqueueJson(200, TRANSACTIONS_PAGE_2)
        model.loadMore()
        val state = settleMore()

        val url = lastRequestUrl(5)
        assertEquals("0", url.queryParameter("offset"))
        assertEquals("2026-10-01", url.queryParameter("date_from"))
        assertEquals("2026-10-31", url.queryParameter("date_to"))
        // Страница нового месяца заменяет список, а не дописывается к сентябрьскому.
        assertEquals(1, state.groups.size)
        assertEquals(LocalDate.parse("2026-09-06"), state.groups[0].date)
    }

    // Страницы кончились, и loadMore выходит до расчёта даты: без проверки на заходе экран
    // показывал бы сентябрь под «этим месяцем» до перезапуска процесса.
    @Test
    fun monthRolloverIsCaughtWhenPagingIsExhausted() = runTest {
        today = LocalDate.parse("2026-09-30")
        enqueueFirstPage()
        createModel()
        settle()
        server.enqueueJson(200, TRANSACTIONS_EMPTY)
        model.onFiltersChange(TransactionFilters(period = TransactionPeriod.THIS_MONTH))
        assertFalse((settle() as TransactionsUiState.Ready).hasMore)

        today = LocalDate.parse("2026-10-01")
        server.enqueueJson(200, TRANSACTIONS_PAGE_1)
        model.revalidate()
        settle()

        val url = lastRequestUrl(5)
        assertEquals("0", url.queryParameter("offset"))
        assertEquals("2026-10-01", url.queryParameter("date_from"))
        assertEquals("2026-10-31", url.queryParameter("date_to"))
    }

    // Возврат из фона под идущим запросом: его окно посчитано до полуночи, и без сверки
    // с ним ответ встал бы на экран сентябрём под видом «этого месяца».
    @Test
    fun revalidateRestartsRequestStartedBeforeMidnight() = runTest {
        // Ответы раздаются по запросу: октябрьский приходит, пока сервер держит сентябрьский.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = when {
                request.url.encodedPath.endsWith("/categories") -> jsonResponse(CATEGORIES_OK)

                request.url.encodedPath.endsWith("/users") -> jsonResponse(USERS_OK)

                request.url.queryParameter("date_from") == "2026-10-01" -> jsonResponse(TRANSACTIONS_EMPTY)

                request.url.queryParameter("offset") == "2" ->
                    jsonResponse(TRANSACTIONS_PAGE_1, delayMs = STALE_PAGE_DELAY_MS)

                else -> jsonResponse(TRANSACTIONS_PAGE_1)
            }
        }
        today = LocalDate.parse("2026-09-30")
        createModel()
        settle()
        model.onFiltersChange(TransactionFilters(period = TransactionPeriod.THIS_MONTH))
        settle()

        model.loadMore()
        today = LocalDate.parse("2026-10-01")
        model.revalidate()

        assertTrue((settle() as TransactionsUiState.Ready).isEmpty)
    }

    // Тот же заход в пределах месяца лишнего запроса не делает: список уже за это окно.
    @Test
    fun revalidateWithinSameMonthKeepsList() = runTest {
        enqueueFirstPage()
        createModel()
        settle()
        server.enqueueJson(200, TRANSACTIONS_EMPTY)
        model.onFiltersChange(TransactionFilters(period = TransactionPeriod.THIS_MONTH))
        settle()

        model.revalidate()

        assertTrue(model.state.value is TransactionsUiState.Ready)
        assertEquals(4, server.requestCount)
    }

    // Отказ запроса нового месяца: сентябрьский список нельзя оставить на экране как «этот
    // месяц», поэтому такая догрузка падает целым экраном, а не подвалом.
    @Test
    fun monthRolloverFailureReplacesScreen() = runTest {
        today = LocalDate.parse("2026-09-30")
        enqueueFirstPage()
        createModel()
        settle()
        server.enqueueJson(200, TRANSACTIONS_PAGE_1)
        model.onFiltersChange(TransactionFilters(period = TransactionPeriod.THIS_MONTH))
        settle()

        today = LocalDate.parse("2026-10-01")
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")
        model.loadMore()

        assertEquals(
            TransactionsUiState.Failure(UiError.Server("всё сломалось")),
            model.state.first { it is TransactionsUiState.Failure },
        )
    }

    @Test
    fun filtersGoToQuery() = runTest {
        enqueueFirstPage()
        createModel()
        settle()

        server.enqueueJson(200, TRANSACTIONS_PAGE_1)
        model.onFiltersChange(
            TransactionFilters(
                period = TransactionPeriod.THIS_MONTH,
                type = TransactionType.expense,
                categoryId = UUID.fromString(GROCERIES_ID),
            ),
        )
        settle()

        val url = lastRequestUrl(4)
        assertEquals("expense", url.queryParameter("type"))
        assertEquals(GROCERIES_ID, url.queryParameter("category_id"))
        assertEquals("2026-09-01", url.queryParameter("date_from"))
        assertEquals("2026-09-30", url.queryParameter("date_to"))
    }

    // Смена фильтра во время догрузки: страница прошлого запроса не должна дописаться к новому
    // списку — в нём оказались бы две строки с одним id, и LazyColumn упал бы на ключе.
    @Test
    fun filterChangeDropsPageOfPreviousQuery() = runTest {
        // Ответы раздаются по запросу, а не очередью: порядок прихода здесь и есть предмет теста.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = when {
                request.url.encodedPath.endsWith("/categories") -> jsonResponse(CATEGORIES_OK)

                request.url.encodedPath.endsWith("/users") -> jsonResponse(USERS_OK)

                request.url.queryParameter("type") == "income" -> jsonResponse(TRANSACTIONS_PAGE_2)

                // Догрузка старого фильтра: сервер держит её, пока фильтр уже сменился.
                request.url.queryParameter("offset") == "2" ->
                    jsonResponse(TRANSACTIONS_PAGE_1, delayMs = STALE_PAGE_DELAY_MS)

                else -> jsonResponse(TRANSACTIONS_PAGE_1)
            }
        }
        createModel()
        settle()

        model.loadMore()
        model.onFiltersChange(TransactionFilters(type = TransactionType.income))
        settle()

        // Ожидание реальное: задержку держит сервер, а не планировщик runTest.
        withContext(Dispatchers.IO) { Thread.sleep(STALE_PAGE_DELAY_MS * 3) }

        val rows = (model.state.value as TransactionsUiState.Ready).groups.flatMap { it.rows }
        assertEquals(listOf("Зарплата"), rows.map { it.transaction.description })
    }

    // `/users` открыт только админу: у member список авторов остаётся пустым, а не роняет экран.
    @Test
    fun memberDoesNotAskForUsers() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, TRANSACTIONS_PAGE_1)

        createModel(Role.member)
        val state = settle() as TransactionsUiState.Ready

        assertNull(state.groups[0].rows[0].authorName)
        assertTrue(state.groups[0].rows[1].isMine)
        assertEquals("/api/v1/transactions", lastRequestUrl(2).encodedPath)
    }
}
