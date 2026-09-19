package tech.shatrov.familyfinances.ui.home

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
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
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.RECONCILIATION_EMPTY
import tech.shatrov.familyfinances.RECONCILIATION_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.STATS_EMPTY
import tech.shatrov.familyfinances.STATS_OK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.RecognizeResult
import tech.shatrov.familyfinances.core.api.RecognizedTransaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.tempJournals
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.recognize.ImportJournal
import tech.shatrov.familyfinances.ui.recognize.ImportJournalStore
import tech.shatrov.familyfinances.ui.recognize.JournalImage
import java.time.LocalDate
import java.time.YearMonth
import java.time.ZoneId
import java.util.UUID
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger

/** Задержка ответа, ушедшего до полуночи: перезапущенный запрос должен успеть раньше него. */
private const val STALE_SUMMARY_DELAY_MS = 300L

/** Главная: один запрос сводки, валюта из сессии и состояния «пусто» и «повторить». */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HomeViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: HomeViewModel

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

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun settle(): HomeUiState = model.state.first { it != HomeUiState.Loading }

    /** Карточка приходит вторым запросом, уже после сводки. */
    private suspend fun settleCard(): ReconciliationCard = model.card.first { it != ReconciliationCard.Loading }

    private fun enqueueSummary() {
        server.enqueueJson(200, STATS_OK)
        server.enqueueJson(200, RECONCILIATION_OK)
    }

    /** Дата модели: тест переставляет её, чтобы проверить переход через полночь. */
    private var today = LocalDate.parse("2026-09-07")

    private val journals: ImportJournalStore = tempJournals()

    private fun createModel() {
        model = HomeViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            "RUB",
            ZoneId.of("Europe/Moscow"),
            { today },
            journals,
            UnconfinedTestDispatcher(),
        )
    }

    private fun journal(
        id: UUID = UUID.randomUUID(),
        recognizing: Boolean = false,
        withResult: Boolean = true,
    ): ImportJournal {
        val item = RecognizedTransaction(
            source = 0,
            amountMinor = 45_000,
            currency = null,
            type = TransactionType.expense,
            date = LocalDate.of(2026, 9, 7),
            dateAssumed = false,
            description = "Шаверма",
            categoryId = null,
            similar = emptyList(),
        )
        val result = if (withResult) RecognizeResult(listOf(item, item), incomplete = false, model = "m") else null
        return ImportJournal(
            importId = id.toString(),
            updatedAt = System.currentTimeMillis(),
            images = listOf(JournalImage("content://a", "0.jpg"), JournalImage("content://b", "1.jpg")),
            dropped = 0,
            recognizing = recognizing,
            result = result,
            accountId = null,
            rows = emptyList(),
            savedCount = 1,
        )
    }

    @Test
    fun noJournalNoImportCard() = runTest {
        enqueueSummary()

        createModel()
        settle()

        assertNull(model.import.value)
    }

    @Test
    fun journalWithResultReportsRowsAndSaved() = runTest {
        enqueueSummary()
        val id = UUID.randomUUID()
        journals.write(journal(id))

        createModel()

        val draft = model.import.value!!
        assertEquals(id, draft.importId)
        assertTrue(draft.recognized)
        assertEquals(2, draft.rows)
        assertEquals(1, draft.saved)
    }

    // Ответа нет, распознавание прервано: плашка говорит о скриншотах, а не о строках.
    @Test
    fun interruptedRecognitionIsItsOwnPhase() = runTest {
        enqueueSummary()
        journals.write(journal(recognizing = true, withResult = false))

        createModel()

        val draft = model.import.value!!
        assertFalse(draft.recognized)
        assertEquals(2, draft.images)
    }

    @Test
    fun deletingRemovesCardAndJournal() = runTest {
        enqueueSummary()
        val id = UUID.randomUUID()
        journals.write(journal(id))
        createModel()

        model.deleteImport()

        assertNull(model.import.value)
        assertNull(journals.read(id))
        model.loadImport()
        assertNull(model.import.value)
    }

    // Журнал пишет экран распознавания, пока главная в стеке: возврат перечитывает его.
    @Test
    fun loadImportPicksUpJournalWrittenLater() = runTest {
        enqueueSummary()
        createModel()
        assertNull(model.import.value)

        journals.write(journal())
        model.loadImport()

        assertTrue(model.import.value != null)
    }

    @Test
    fun summaryArrivesWithCurrencyFromSession() = runTest {
        enqueueSummary()

        createModel()
        val state = settle() as HomeUiState.Ready

        assertEquals("RUB", state.currency)
        assertEquals(15000000L, state.summary.current.incomeMinor)
        assertEquals(1, state.summary.budgets.size)
        assertFalse(state.isEmpty)
    }

    // Границы периода не шлём: текущий месяц сервер считает в часовом поясе семьи, а не телефона.
    @Test
    fun periodIsLeftToTheServer() = runTest {
        enqueueSummary()

        createModel()
        settle()

        val request = server.takeRequest()
        assertEquals("/api/v1/stats/summary", request.url.encodedPath)
        assertNull(request.url.query)
    }

    @Test
    fun familyWithoutTransactionsIsEmpty() = runTest {
        server.enqueueJson(200, STATS_EMPTY)

        createModel()

        assertTrue((settle() as HomeUiState.Ready).isEmpty)
    }

    @Test
    fun networkFailureIsRetried() = runTest {
        server.close()

        createModel()
        assertEquals(HomeUiState.Failure(UiError.Network), settle())

        server = MockWebServer()
        server.start()
        enqueueSummary()
        // Повтор идёт в новый сокет, поэтому модель пересобирается вместе с адресом.
        createModel()

        assertTrue(settle() is HomeUiState.Ready)
    }

    @Test
    fun retryAfterServerErrorLoadsSummary() = runTest {
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")

        createModel()
        assertEquals(HomeUiState.Failure(UiError.Server("всё сломалось")), settle())

        enqueueSummary()
        model.refresh()

        assertTrue(settle() is HomeUiState.Ready)
    }

    // Сводка посчитана по «сегодня» семьи: без сверки на возврате главная показывала бы
    // прошлый месяц до перезапуска процесса.
    @Test
    fun revalidateAfterMidnightReloadsSummary() = runTest {
        enqueueSummary()
        createModel()
        settleCard()

        today = LocalDate.parse("2026-10-01")
        server.enqueueJson(200, STATS_EMPTY)
        model.revalidate()

        assertTrue((settle() as HomeUiState.Ready).isEmpty)
        assertEquals(3, server.requestCount)
    }

    // Тот же день лишнего запроса не делает: сводка уже за него.
    @Test
    fun revalidateWithinSameDayKeepsSummary() = runTest {
        enqueueSummary()
        createModel()
        settleCard()

        model.revalidate()

        assertTrue(model.state.value is HomeUiState.Ready)
        assertEquals(2, server.requestCount)
    }

    // Возврат из фона под идущим запросом: он ушёл до полуночи, и без сверки с его датой
    // сводка за прошлые сутки встала бы на экран, а второго колбэка жизненного цикла не будет.
    @Test
    fun revalidateRestartsRequestStartedBeforeMidnight() = runTest {
        val served = AtomicInteger()
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = if (served.getAndIncrement() == 0) {
                jsonResponse(STATS_OK, STALE_SUMMARY_DELAY_MS)
            } else {
                jsonResponse(STATS_EMPTY)
            }
        }
        createModel()
        // Ответ сервер держит, но сам запрос уже ушёл: иначе сверять на возврате нечего.
        server.takeRequest()

        today = LocalDate.parse("2026-10-01")
        model.revalidate()

        assertTrue((settle() as HomeUiState.Ready).isEmpty)
        assertEquals(2, server.requestCount)
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

    // Сверяют закончившийся месяц: до 10-го числа карточка ещё на прошлом.
    @Test
    fun cardShowsPreviousMonthBeforeTenth() = runTest {
        today = LocalDate.parse("2026-09-09")
        enqueueSummary()

        createModel()

        assertEquals(ReconciliationCard.Ready(YearMonth.of(2026, 8), matched = 1, total = 2), settleCard())
        server.takeRequest()
        assertEquals("month=2026-08", server.takeRequest().url.query)
    }

    @Test
    fun cardShowsCurrentMonthFromTenth() = runTest {
        today = LocalDate.parse("2026-09-10")
        enqueueSummary()

        createModel()
        settleCard()

        server.takeRequest()
        assertEquals("month=2026-09", server.takeRequest().url.query)
    }

    @Test
    fun cardWithoutAccountsLeadsToAccounts() = runTest {
        server.enqueueJson(200, STATS_OK)
        server.enqueueJson(200, RECONCILIATION_EMPTY)

        createModel()

        assertEquals(ReconciliationCard.NoAccounts, settleCard())
    }

    // Отказ карточки главную не роняет: сводка остаётся, карточки просто нет.
    @Test
    fun cardFailureKeepsSummary() = runTest {
        server.enqueueJson(200, STATS_OK)
        server.enqueueJson(500, INTERNAL_ERROR)

        createModel()

        assertEquals(ReconciliationCard.Hidden, settleCard())
        assertTrue(model.state.value is HomeUiState.Ready)
    }

    @Test
    fun emptyFamilyAsksNoCard() = runTest {
        server.enqueueJson(200, STATS_EMPTY)

        createModel()

        assertEquals(ReconciliationCard.Hidden, settleCard())
        assertEquals(1, server.requestCount)
    }

    @Test
    fun unreadableBodyIsMalformed() = runTest {
        server.enqueueJson(200, "<html>прокси съел ответ</html>")

        createModel()

        assertEquals(HomeUiState.Failure(UiError.Malformed), settle())
    }
}
