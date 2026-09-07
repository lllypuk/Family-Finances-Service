package tech.shatrov.familyfinances.ui.transactions

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import mockwebserver3.MockWebServer
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

    private fun createModel(role: Role = Role.admin) {
        model = TransactionsViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            testSession(role),
            LocalDate.parse("2026-09-15"),
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
