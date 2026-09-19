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
import tech.shatrov.familyfinances.BALANCE_PUT_OK
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.CASH_ACCOUNT_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.RECONCILIATION_COMPLETE
import tech.shatrov.familyfinances.RECONCILIATION_EMPTY
import tech.shatrov.familyfinances.RECONCILIATION_NO_OPENING
import tech.shatrov.familyfinances.RECONCILIATION_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.transactions.TransactionPrefill
import java.time.LocalDate
import java.time.YearMonth
import java.util.UUID

private val AUGUST = YearMonth.of(2026, 8)
private val TODAY = LocalDate.of(2026, 9, 18)

private const val BALANCE_NOT_FOUND = """
{"error":{"code":"BALANCE_NOT_FOUND","message":"balance not found"},
"meta":{"request_id":"r-1","timestamp":"2026-09-18T10:00:00Z","version":"v0.9.0"}}
"""

/** Сверка остатков: итог по краям, знак разницы, запись и очистка остатка, отказы. */
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
        model = newModel()
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun newModel() = ReconciliationViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())))

    private suspend fun settle(): ReconciliationUiState = model.state.first { it != ReconciliationUiState.Loading }

    private suspend fun loadAugust(body: String = RECONCILIATION_OK): ReconciliationUiState.Ready {
        server.enqueueJson(200, body)
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
        val request = server.takeRequest()
        assertEquals("/api/v1/stats/reconciliation", request.url.encodedPath)
        assertEquals("month=2026-08", request.url.query)
    }

    @Test
    fun missingClosingCountsAccountsWithoutBalance() = runTest {
        assertEquals(ReconciliationTotal.MissingClosing(1), loadAugust().total)
    }

    @Test
    fun missingOpeningPointsToPreviousMonth() = runTest {
        assertEquals(
            ReconciliationTotal.MissingOpening(YearMonth.of(2026, 7)),
            loadAugust(RECONCILIATION_NO_OPENING).total,
        )
    }

    @Test
    fun completeCarriesGap() = runTest {
        assertEquals(
            ReconciliationTotal.Complete(openingMinor = 5100000, closingMinor = 5610000, gapMinor = 10000),
            loadAugust(RECONCILIATION_COMPLETE).total,
        )
    }

    @Test
    fun gapSignNamesWhatIsMissing() {
        assertEquals(GapVerdict.MATCHED, gapVerdict(0))
        assertEquals(GapVerdict.MISSING_INCOME, gapVerdict(10000))
        assertEquals(GapVerdict.MISSING_EXPENSE, gapVerdict(-1))
    }

    // `gap > 0` — не записан приход: корректировка приходом; дата — конец прошлого месяца.
    @Test
    fun positiveGapIsClosedByIncomeAtMonthEnd() {
        assertEquals(
            TransactionPrefill(10000, TransactionType.income, LocalDate.of(2026, 8, 31), "Корректировка"),
            gapCorrection(AUGUST, 10000, TODAY, "Корректировка"),
        )
    }

    // Текущий месяц ещё не кончился: конец месяца — будущая дата, которую сервер отвергнет.
    @Test
    fun negativeGapInCurrentMonthIsExpenseToday() {
        assertEquals(
            TransactionPrefill(2550, TransactionType.expense, TODAY, "Корректировка"),
            gapCorrection(YearMonth.from(TODAY), -2550, TODAY, "Корректировка"),
        )
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
        model = newModel()
        server.enqueueJson(200, RECONCILIATION_OK)
        model.load(AUGUST)

        assertTrue(settle() is ReconciliationUiState.Ready)
    }

    @Test
    fun saveSendsNegativeBalanceAndReloads() = runTest {
        loadAugust()
        model.onOpenBalance(row(CASH_ACCOUNT_ID))
        assertFalse(model.editor.value!!.exists)

        model.onAmountChange("3 500,00")
        model.onToggleSign()
        assertEquals("-3 500,00", model.editor.value!!.amount)
        server.enqueueJson(200, BALANCE_PUT_OK)
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onSave()

        model.editor.first { it == null }
        settle()
        server.takeRequest()
        val put = server.takeRequest()
        assertEquals("PUT", put.method)
        assertEquals("/api/v1/accounts/$CASH_ACCOUNT_ID/balances/2026-08", put.url.encodedPath)
        assertEquals("""{"balance_minor":-350000}""", put.body?.utf8())
        assertEquals("/api/v1/stats/reconciliation", server.takeRequest().url.encodedPath)
    }

    // Счёт, заведённый в августе, в июле не показан: его старт пишется отсюда, а экран остаётся на августе.
    @Test
    fun openingIsSavedForPreviousMonthAndCurrentMonthReloads() = runTest {
        loadAugust()
        model.onOpenOpening(row(CASH_ACCOUNT_ID))
        val editor = model.editor.value!!
        assertEquals(YearMonth.of(2026, 7), editor.month)
        assertFalse(editor.exists)

        model.onAmountChange("100")
        server.enqueueJson(200, BALANCE_PUT_OK)
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onSave()

        model.editor.first { it == null }
        assertEquals(AUGUST, (settle() as ReconciliationUiState.Ready).month)
        server.takeRequest()
        assertEquals("/api/v1/accounts/$CASH_ACCOUNT_ID/balances/2026-07", server.takeRequest().url.encodedPath)
        assertEquals("month=2026-08", server.takeRequest().url.query)
    }

    // Отказ оставляет диалог с введённым: остаток не надо набирать заново.
    @Test
    fun saveFailureKeepsDialog() = runTest {
        loadAugust()
        model.onOpenBalance(row(CASH_ACCOUNT_ID))
        model.onAmountChange("100")
        server.enqueueJson(500, INTERNAL_ERROR)
        model.onSave()

        val editor = model.editor.first { it?.submitting == false && it.error != null }!!
        assertEquals("100", editor.amount)
        assertEquals(UiError.Server("всё сломалось"), editor.error)
    }

    @Test
    fun existingBalanceIsPrefilledAndCleared() = runTest {
        loadAugust()
        model.onOpenBalance(row(CARD_ACCOUNT_ID))
        val editor = model.editor.value!!
        assertTrue(editor.exists)
        assertEquals("-4500", editor.amount)
        assertEquals(-450000L, editor.amountMinor)

        server.enqueueJson(204, "")
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onClear()

        assertNull(model.editor.first { it == null })
        settle()
        server.takeRequest()
        val delete = server.takeRequest()
        assertEquals("DELETE", delete.method)
        assertEquals("/api/v1/accounts/$CARD_ACCOUNT_ID/balances/2026-08", delete.url.encodedPath)
    }

    // Остаток уже удалили с другого телефона: 404 закрывает диалог, как успех.
    @Test
    fun clearOfMissingBalanceClosesDialog() = runTest {
        loadAugust()
        model.onOpenBalance(row(CARD_ACCOUNT_ID))
        server.enqueueJson(404, BALANCE_NOT_FOUND)
        server.enqueueJson(200, RECONCILIATION_OK)
        model.onClear()

        assertNull(model.editor.first { it == null })
        settle()
        server.takeRequest()
        assertEquals("DELETE", server.takeRequest().method)
        assertEquals("/api/v1/stats/reconciliation", server.takeRequest().url.encodedPath)
    }

    @Test
    fun unparsableAmountCannotBeSaved() = runTest {
        loadAugust()
        model.onOpenBalance(row(CASH_ACCOUNT_ID))
        model.onAmountChange("сто")

        assertFalse(model.editor.value!!.canSubmit)
        model.onSave()
        assertEquals(1, server.requestCount)
    }
}
