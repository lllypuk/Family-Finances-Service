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
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.FLAT_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.HOLDINGS_AFTER_VALUE_DELETE
import tech.shatrov.familyfinances.HOLDINGS_EXPENSE_PLAN
import tech.shatrov.familyfinances.HOLDINGS_OK
import tech.shatrov.familyfinances.HOLDING_VALUE_OK
import tech.shatrov.familyfinances.NET_WORTH_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.VALIDATION_ERROR
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import java.time.LocalDate

/** Капитал: группы и архив из одного списка, итог из ряда, снимок — `PUT` по дате не позже сегодня. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class NetWorthViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: NetWorthViewModel
    private var today = LocalDate.of(2026, 9, 18)

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

    private suspend fun loaded(holdings: String = HOLDINGS_OK): NetWorthUiState.Ready {
        server.enqueueJson(200, holdings)
        server.enqueueJson(200, NET_WORTH_OK)
        model = NetWorthViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())), today = { today })
        return model.state.first { it !is NetWorthUiState.Loading } as NetWorthUiState.Ready
    }

    @Test
    fun loadSplitsSidesAndArchiveAndTakesTotalFromLastBucket() = runTest {
        val state = loaded()

        assertEquals(listOf("Квартира", "Биткоин"), state.assets.map { it.holding.name })
        assertEquals(listOf("Ипотека"), state.liabilities.map { it.holding.name })
        assertEquals(listOf("Машина"), state.archived.map { it.holding.name })
        assertEquals(460000000L, state.latest?.netMinor)

        val list = server.takeRequest()
        assertEquals("/api/v1/holdings", list.url.encodedPath)
        assertEquals("true", list.url.queryParameter("archived"))
        val series = server.takeRequest()
        assertEquals("/api/v1/stats/net-worth", series.url.encodedPath)
        // `to` не передаётся: сервер берёт сегодня семьи, и последняя корзина равна сумме `current`.
        assertNull(series.url.queryParameter("to"))
    }

    @Test
    fun unknownKindIsShownAsOther() = runTest {
        val state = loaded()

        assertEquals(HoldingKind.OTHER, state.assets.single { it.holding.kind == "crypto" }.kind)
        assertEquals(HoldingKind.PROPERTY, state.assets.first().kind)
    }

    @Test
    fun valueIsPutOnPickedDateAndReloads() = runTest {
        val flat = loaded().assets.first().holding
        repeat(2) { server.takeRequest() }

        model.onOpenValue(flat)
        assertEquals("11000000", model.editor.value?.amount)
        assertEquals(today, model.editor.value?.date)

        model.onAmountChange("850000,50")
        model.onDateChange(LocalDate.of(2026, 9, 1))
        server.enqueueJson(200, HOLDING_VALUE_OK)
        server.enqueueJson(200, HOLDINGS_OK)
        server.enqueueJson(200, NET_WORTH_OK)
        model.onSaveValue()

        assertNull(model.editor.first { it?.submitting != true })
        val put = server.takeRequest()
        assertEquals("PUT", put.method)
        assertEquals("/api/v1/holdings/$FLAT_ID/values/2026-09-01", put.url.encodedPath)
        assertEquals("""{"value_minor":85000050}""", put.body?.utf8())
        assertEquals("/api/v1/holdings", server.takeRequest().url.encodedPath)
    }

    @Test
    fun futureDateIsCappedAtToday() = runTest {
        val flat = loaded().assets.first().holding
        model.onOpenValue(flat)

        model.onDateChange(today.plusDays(1))

        assertEquals(today, model.editor.value?.date)
    }

    @Test
    fun rejectedValueKeepsSheetOpenWithError() = runTest {
        val flat = loaded().assets.first().holding
        model.onOpenValue(flat)
        server.enqueueJson(422, VALIDATION_ERROR)

        model.onSaveValue()

        val editor = model.editor.first { it?.submitting != true }
        assertTrue(editor?.error != null)
    }

    @Test
    fun revalidateReloadsOnlyAfterMidnight() = runTest {
        loaded()
        repeat(2) { server.takeRequest() }

        model.revalidate()
        assertEquals(2, server.requestCount)

        today = today.plusDays(1)
        server.enqueueJson(200, HOLDINGS_OK)
        server.enqueueJson(200, NET_WORTH_OK)
        model.revalidate()
        model.state.first { it is NetWorthUiState.Ready }
        assertEquals(4, server.requestCount)
    }

    @Test
    fun planTotalCountsActiveHoldingsAndHowManyHavePlan() = runTest {
        // У ипотеки и биткоина полей плана нет — ответ сервера до `v0.8.0` разбирается как нули.
        assertEquals(PlanTotal(4_500_000L, 830_000L, planned = 1, active = 3), loaded().plan)
    }

    @Test
    fun planTotalSkipsArchivedAndGoesNegativeOnExpensesOnly() = runTest {
        val plan = loaded(HOLDINGS_EXPENSE_PLAN).plan

        assertEquals(PlanTotal(0L, 7_430_000L, planned = 1, active = 1), plan)
        assertEquals(-7_430_000L, plan?.netMinor)
    }

    @Test
    fun noPlansMeansNoPlanTotal() = runTest {
        assertNull(loaded(HOLDINGS_AFTER_VALUE_DELETE).plan)
    }

    @Test
    fun planTotalIsHiddenWhenListDoesNotFitOnePage() = runTest {
        assertNull(loaded(HOLDINGS_OK.replace("\"total\":4", "\"total\":201")).plan)
    }
}
