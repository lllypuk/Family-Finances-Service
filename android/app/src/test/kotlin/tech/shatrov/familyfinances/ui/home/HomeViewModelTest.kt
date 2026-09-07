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
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.STATS_EMPTY
import tech.shatrov.familyfinances.STATS_OK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.LocalDate
import java.time.ZoneId
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

    /** Дата модели: тест переставляет её, чтобы проверить переход через полночь. */
    private var today = LocalDate.parse("2026-09-07")

    private fun createModel() {
        model = HomeViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            "RUB",
            ZoneId.of("Europe/Moscow"),
            { today },
        )
    }

    @Test
    fun summaryArrivesWithCurrencyFromSession() = runTest {
        server.enqueueJson(200, STATS_OK)

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
        server.enqueueJson(200, STATS_OK)

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
        server.enqueueJson(200, STATS_OK)
        // Повтор идёт в новый сокет, поэтому модель пересобирается вместе с адресом.
        createModel()

        assertTrue(settle() is HomeUiState.Ready)
    }

    @Test
    fun retryAfterServerErrorLoadsSummary() = runTest {
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")

        createModel()
        assertEquals(HomeUiState.Failure(UiError.Server("всё сломалось")), settle())

        server.enqueueJson(200, STATS_OK)
        model.refresh()

        assertTrue(settle() is HomeUiState.Ready)
    }

    // Сводка посчитана по «сегодня» семьи: без сверки на возврате главная показывала бы
    // прошлый месяц до перезапуска процесса.
    @Test
    fun revalidateAfterMidnightReloadsSummary() = runTest {
        server.enqueueJson(200, STATS_OK)
        createModel()
        settle()

        today = LocalDate.parse("2026-10-01")
        server.enqueueJson(200, STATS_EMPTY)
        model.revalidate()

        assertTrue((settle() as HomeUiState.Ready).isEmpty)
        assertEquals(2, server.requestCount)
    }

    // Тот же день лишнего запроса не делает: сводка уже за него.
    @Test
    fun revalidateWithinSameDayKeepsSummary() = runTest {
        server.enqueueJson(200, STATS_OK)
        createModel()
        settle()

        model.revalidate()

        assertTrue(model.state.value is HomeUiState.Ready)
        assertEquals(1, server.requestCount)
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

    @Test
    fun unreadableBodyIsMalformed() = runTest {
        server.enqueueJson(200, "<html>прокси съел ответ</html>")

        createModel()

        assertEquals(HomeUiState.Failure(UiError.Malformed), settle())
    }
}
