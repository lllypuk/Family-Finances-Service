package tech.shatrov.familyfinances.ui.networth

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
import tech.shatrov.familyfinances.FLAT_ID
import tech.shatrov.familyfinances.FLAT_VALUES_AFTER_DELETE
import tech.shatrov.familyfinances.FLAT_VALUES_PAGE_1
import tech.shatrov.familyfinances.FLAT_VALUES_PAGE_2
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.HOLDINGS_AFTER_VALUE_DELETE
import tech.shatrov.familyfinances.HOLDINGS_EMPTY
import tech.shatrov.familyfinances.HOLDINGS_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import java.time.LocalDate
import java.util.UUID

/** История позиции: страницы по `meta.pagination`, правка и удаление снимка перечитывают `current`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HoldingHistoryViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: HoldingHistoryViewModel
    private val today = LocalDate.of(2026, 9, 18)

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

    private fun create() {
        model = HoldingHistoryViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            UUID.fromString(FLAT_ID),
            today = { today },
        )
    }

    private suspend fun loaded(): HoldingHistoryUiState.Ready {
        server.enqueueJson(200, HOLDINGS_OK)
        server.enqueueJson(200, FLAT_VALUES_PAGE_1)
        create()
        return model.state.first { it !is HoldingHistoryUiState.Loading } as HoldingHistoryUiState.Ready
    }

    @Test
    fun secondPageIsAppendedAtOffset() = runTest {
        val first = loaded()
        assertEquals("Квартира", first.holding.name)
        assertEquals(2, first.values.size)
        assertTrue(first.hasMore)
        repeat(2) { server.takeRequest() }

        server.enqueueJson(200, FLAT_VALUES_PAGE_2)
        model.loadMore()

        val second = model.state.first { it is HoldingHistoryUiState.Ready && !it.loadingMore }
            as HoldingHistoryUiState.Ready
        assertEquals(
            listOf("2026-01-12", "2025-01-10", "2024-01-15"),
            second.values.map { it.date.toString() },
        )
        assertFalse(second.hasMore)
        val page = server.takeRequest()
        assertEquals("/api/v1/holdings/$FLAT_ID/values", page.url.encodedPath)
        assertEquals("2", page.url.queryParameter("offset"))

        model.loadMore()
        assertEquals(3, server.requestCount)
    }

    @Test
    fun deletingValueReloadsCurrent() = runTest {
        val ready = loaded()
        repeat(2) { server.takeRequest() }

        model.onOpenValue(ready.values.first())
        val editor = requireNotNull(model.editor.value)
        assertTrue(editor.existing)
        assertEquals(LocalDate.of(2026, 1, 12), editor.date)

        server.enqueueJson(204, "")
        server.enqueueJson(200, HOLDINGS_AFTER_VALUE_DELETE)
        server.enqueueJson(200, FLAT_VALUES_AFTER_DELETE)
        model.onDeleteValue()

        assertNull(model.editor.first { it?.submitting != true })
        val after = model.state.first {
            it is HoldingHistoryUiState.Ready && it.values.size == 2 && it.values.first().date.year == 2025
        } as HoldingHistoryUiState.Ready
        assertEquals(LocalDate.of(2025, 1, 10), after.holding.current?.date)
        assertEquals(1_000_000_000L, after.holding.current?.valueMinor)
        val delete = server.takeRequest()
        assertEquals("DELETE", delete.method)
        assertEquals("/api/v1/holdings/$FLAT_ID/values/2026-01-12", delete.url.encodedPath)
    }

    @Test
    fun existingValueKeepsItsDate() = runTest {
        val ready = loaded()
        model.onOpenValue(ready.values.first())

        model.onDateChange(LocalDate.of(2026, 9, 1))

        assertEquals(LocalDate.of(2026, 1, 12), model.editor.value?.date)
    }

    @Test
    fun missingHoldingIsGone() = runTest {
        server.enqueueJson(200, HOLDINGS_EMPTY)
        create()

        assertEquals(HoldingHistoryUiState.Gone, model.state.first { it !is HoldingHistoryUiState.Loading })
    }
}
