package tech.shatrov.familyfinances.ui.overview

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
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.STATS_MONTHLY_OK
import tech.shatrov.familyfinances.STATS_OK
import tech.shatrov.familyfinances.STATS_SUMMARY_RANGE_OK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.liveToken
import java.time.LocalDate
import java.time.YearMonth
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.TimeUnit

private const val STALE_SUMMARY_DELAY_MS = 300L

/** «Обзор»: ряд один на экран, `summary` — на каждую смену периода, ответ всегда последнего. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class OverviewViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: OverviewViewModel
    private var today = LocalDate.of(2026, 9, 19)
    private val requests = CopyOnWriteArrayList<RecordedRequest>()

    @Volatile private var summaryFails = false

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        // Ответы раздаются по запросу: ряд и сводка уходят параллельно, очередь их перепутала бы.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                requests += request
                return when {
                    request.url.encodedPath.endsWith("/monthly") -> json(STATS_MONTHLY_OK)

                    summaryFails -> json(INTERNAL_ERROR, code = 500)

                    request.url.queryParameter("from") == "2026-07-01" -> json(STATS_SUMMARY_RANGE_OK)

                    // «Прошлый месяц» сервер держит: его ответ приходит после ответа следующего чипа.
                    request.url.queryParameter("from") == "2026-08-01" ->
                        json(STATS_OK, delayMs = STALE_SUMMARY_DELAY_MS)

                    else -> json(STATS_OK)
                }
            }
        }
        server.start()
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun json(
        body: String,
        code: Int = 200,
        delayMs: Long = 0,
    ): MockResponse = MockResponse.Builder()
        .code(code)
        .body(body)
        .setHeader("Content-Type", "application/json")
        .bodyDelay(delayMs, TimeUnit.MILLISECONDS)
        .build()

    private fun createModel(initial: OverviewPeriod = OverviewPeriod.ThisMonth) {
        model = OverviewViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            currency = "RUB",
            initial = initial,
            today = { today },
        )
    }

    private suspend fun settle(): OverviewUiState =
        model.state.first { it.monthly !is MonthlyState.Loading && it.summary !is SummaryState.Loading }

    private fun paths(suffix: String) = requests.filter { it.url.encodedPath.endsWith(suffix) }

    @Test
    fun loadAsksMonthlyWithoutBoundsAndSummaryForPeriod() = runTest {
        createModel()
        val state = settle()

        val monthly = paths("/monthly").single()
        assertNull(monthly.url.queryParameter("from"))
        assertNull(monthly.url.queryParameter("to"))
        val summary = paths("/summary").single()
        assertEquals("2026-09-01", summary.url.queryParameter("from"))
        assertEquals("2026-09-19", summary.url.queryParameter("to"))
        assertEquals(DateRange(LocalDate.of(2026, 9, 1), today), state.range)
        assertEquals(12, (state.monthly as MonthlyState.Ready).months.size)
        val ready = state.summary as SummaryState.Ready
        assertEquals(0.25, ready.incomeDelta!!, 0.0)
        assertEquals(-0.15, ready.expensesDelta!!, 0.0)
        assertEquals("Продукты", ready.expenseCategories.single().name)
        assertEquals("Зарплата", ready.incomeCategories.single().name)
    }

    // Чип «Год» совпадает с рядом по умолчанию: сумма столбиков сходится с его итогом.
    @Test
    fun yearMatchesFirstAndLastBucketOfMonthly() = runTest {
        createModel(OverviewPeriod.Year)
        val state = settle()

        val months = (state.monthly as MonthlyState.Ready).months
        assertEquals(YearMonth.parse(months.first().month).atDay(1), state.range.from)
        assertEquals(YearMonth.parse(months.last().month), YearMonth.from(state.range.to))
        assertEquals(today, state.range.to)
    }

    @Test
    fun periodChangeSendsOneSummaryAndKeepsMonthly() = runTest {
        createModel()
        settle()

        model.select(OverviewPeriod.ThreeMonths)
        val state = settle()

        assertEquals(1, paths("/monthly").size)
        val summaries = paths("/summary")
        assertEquals(2, summaries.size)
        assertEquals("2026-07-01", summaries.last().url.queryParameter("from"))
        assertEquals("2026-09-19", summaries.last().url.queryParameter("to"))
        val ready = state.summary as SummaryState.Ready
        // Перед периодом операций не было: дельта не «0 %», а её нет.
        assertNull(ready.incomeDelta)
        assertNull(ready.expensesDelta)
        assertEquals(listOf("Продукты", "Ипотека"), ready.expenseCategories.map { it.name })
    }

    @Test
    fun reselectingSamePeriodSendsNothing() = runTest {
        createModel()
        settle()

        model.select(OverviewPeriod.ThisMonth)

        assertEquals(1, paths("/summary").size)
    }

    @Test
    fun fastDoubleChangeKeepsAnswerOfLast() = runTest {
        createModel()
        settle()

        model.select(OverviewPeriod.PrevMonth)
        model.select(OverviewPeriod.ThreeMonths)
        settle()
        // Ожидание реальное: задержку держит сервер, а не планировщик runTest.
        withContext(Dispatchers.IO) { Thread.sleep(STALE_SUMMARY_DELAY_MS * 3) }

        val state = model.state.value
        assertEquals(OverviewPeriod.ThreeMonths, state.period)
        assertEquals(LocalDate.of(2026, 7, 1), (state.summary as SummaryState.Ready).totals.from)
    }

    @Test
    fun summaryFailureKeepsSeries() = runTest {
        createModel()
        settle()

        summaryFails = true
        model.select(OverviewPeriod.Year)
        val failed = settle()

        assertTrue(failed.summary is SummaryState.Failure)
        assertEquals(12, (failed.monthly as MonthlyState.Ready).months.size)

        summaryFails = false
        model.retrySummary()
        assertTrue(settle().summary is SummaryState.Ready)
        assertEquals(1, paths("/monthly").size)
    }

    @Test
    fun revalidateReloadsOnlyAfterDayChange() = runTest {
        createModel()
        settle()

        model.revalidate()
        assertEquals(2, requests.size)

        today = LocalDate.of(2026, 10, 1)
        model.revalidate()
        val state = settle()

        assertEquals(2, paths("/monthly").size)
        assertEquals(DateRange(LocalDate.of(2026, 10, 1), today), state.range)
        assertEquals("2026-10-01", paths("/summary").last().url.queryParameter("from"))
    }
}
