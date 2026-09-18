package tech.shatrov.familyfinances.ui.reconciliation

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
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.CASH_ACCOUNT_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.RECONCILIATION_EMPTY
import tech.shatrov.familyfinances.RECONCILIATION_OK
import tech.shatrov.familyfinances.RECONCILIATION_PUT_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.YearMonth
import java.util.UUID

private val AUGUST = YearMonth.of(2026, 8)

/** Сверка: загрузка месяца, запись и удаление цифры банка, отказы. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ReconciliationViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: ReconciliationViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        model = ReconciliationViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())))
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private suspend fun settle(): ReconciliationUiState = model.state.first { it != ReconciliationUiState.Loading }

    private suspend fun loadAugust(): ReconciliationUiState.Ready {
        server.enqueueJson(200, RECONCILIATION_OK)
        model.load(AUGUST)
        return settle() as ReconciliationUiState.Ready
    }

    private fun row(id: String) = (model.state.value as ReconciliationUiState.Ready)
        .stats
        .accounts
        .first { it.account.id == UUID.fromString(id) }

    @Test
    fun loadAsksForMonth() = runTest {
        val state = loadAugust()

        assertEquals(AUGUST, state.month)
        assertEquals(3, state.stats.accounts.size)
        assertEquals(125000L, state.stats.unassignedMinor)
        val request = server.takeRequest()
        assertEquals("/api/v1/stats/reconciliation", request.url.encodedPath)
        assertEquals("month=2026-08", request.url.query)
    }

    @Test
    fun monthChangeReloads() = runTest {
        loadAugust()

        server.enqueueJson(200, RECONCILIATION_EMPTY)
        model.load(AUGUST.plusMonths(1))

        assertTrue((settle() as ReconciliationUiState.Ready).isEmpty)
        server.takeRequest()
        assertEquals("month=2026-09", server.takeRequest().url.query)
    }

    @Test
    fun networkFailureIsRetried() = runTest {
        server.close()
        model.load(AUGUST)
        assertEquals(ReconciliationUiState.Failure(UiError.Network), settle())

        server = MockWebServer()
        server.start()
        model = ReconciliationViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())))
        server.enqueueJson(200, RECONCILIATION_OK)
        model.load(AUGUST)

        assertTrue(settle() is ReconciliationUiState.Ready)
    }

    @Test
    fun saveSendsBankFigureAndReloads() = runTest {
        loadAugust()
        model.onOpenBank(row(CASH_ACCOUNT_ID))
        assertFalse(model.editor.value!!.exists)

        model.onAmountChange("3 500,00")
        model.onNoteChange("  чек  ")
        server.enqueueJson(200, RECONCILIATION_PUT_OK)
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onSave()

        model.editor.first { it == null }
        settle()
        server.takeRequest()
        val put = server.takeRequest()
        assertEquals("PUT", put.method)
        assertEquals("/api/v1/accounts/$CASH_ACCOUNT_ID/reconciliations/2026-08", put.url.encodedPath)
        assertEquals("""{"bank_expense_minor":350000,"note":"чек"}""", put.body?.utf8())
        assertEquals("/api/v1/stats/reconciliation", server.takeRequest().url.encodedPath)
    }

    // Отказ оставляет лист с введённым: цифру банка не надо набирать заново.
    @Test
    fun saveFailureKeepsSheet() = runTest {
        loadAugust()
        model.onOpenBank(row(CASH_ACCOUNT_ID))
        model.onAmountChange("100")
        server.enqueueJson(500, INTERNAL_ERROR)
        model.onSave()

        val editor = model.editor.first { it?.submitting == false && it.error != null }!!
        assertEquals("100", editor.amount)
        assertEquals(UiError.Server("всё сломалось"), editor.error)
    }

    @Test
    fun existingReconciliationIsPrefilledAndDeleted() = runTest {
        loadAugust()
        model.onOpenBank(row(CARD_ACCOUNT_ID))
        val editor = model.editor.value!!
        assertTrue(editor.exists)
        assertEquals("43100", editor.amount)
        assertEquals("выписка", editor.note)

        server.enqueueJson(204, "")
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onDelete()

        assertNull(model.editor.first { it == null })
        settle()
        server.takeRequest()
        val delete = server.takeRequest()
        assertEquals("DELETE", delete.method)
        assertEquals("/api/v1/accounts/$CARD_ACCOUNT_ID/reconciliations/2026-08", delete.url.encodedPath)
    }

    @Test
    fun unparsableAmountCannotBeSaved() = runTest {
        loadAugust()
        model.onOpenBank(row(CASH_ACCOUNT_ID))
        model.onAmountChange("сто")

        assertFalse(model.editor.value!!.canSubmit)
        model.onSave()
        assertEquals(1, server.requestCount)
    }
}
