package tech.shatrov.familyfinances.ui.budgets

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
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.BUDGETS_EMPTY
import tech.shatrov.familyfinances.BUDGETS_LIST
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.LocalDate
import java.time.ZoneId

/** Бюджет на исходе: между порогом «близко» и перерасходом. */
private const val BUDGETS_NEAR = """
{"data":[
{"id":"99999999-9999-9999-9999-999999999993","name":"Кафе","amount_minor":1000000,
"spent_minor":850000,"remaining_minor":150000,"utilization":85.0,"period":"monthly",
"start_date":"2026-09-01","end_date":"2026-09-30","is_active":true,
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-17","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":1}}}
"""

/** Бюджеты: фильтр уходит в query, имена категорий приходят вторым запросом. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BudgetsViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: BudgetsViewModel

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
    private var today = LocalDate.parse("2026-09-07")

    private fun createModel() {
        model = BudgetsViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            ZoneId.of("Europe/Moscow"),
            { today },
        )
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun settle(): BudgetsUiState = model.state.first { it != BudgetsUiState.Loading }

    private fun enqueueList(budgets: String = BUDGETS_LIST) {
        server.enqueueJson(200, budgets)
        server.enqueueJson(200, CATEGORIES_OK)
    }

    /** Пропускает предыдущие запросы: интересен всегда последний список. */
    private fun requestUrl(count: Int) = (1..count).map { server.takeRequest() }.last().url

    @Test
    fun todayFilterAsksForActiveOnly() = runTest {
        enqueueList()

        createModel()
        val state = settle() as BudgetsUiState.Ready

        assertEquals(BudgetFilter.TODAY, state.filter)
        assertEquals("true", requestUrl(1).queryParameter("active_only"))
    }

    @Test
    fun allFilterDropsTheParameter() = runTest {
        enqueueList()
        createModel()
        settle()

        enqueueList()
        model.onFilterChange(BudgetFilter.ALL)
        val state = settle() as BudgetsUiState.Ready

        assertEquals(BudgetFilter.ALL, state.filter)
        assertNull(requestUrl(3).queryParameter("active_only"))
    }

    @Test
    fun rowsCarryCategoryNameAndLevel() = runTest {
        enqueueList()

        createModel()
        val rows = (settle() as BudgetsUiState.Ready).rows

        assertEquals(listOf("Еда", "Всё"), rows.map { it.budget.name })
        assertEquals("Продукты", rows.first().categoryName)
        assertEquals(GROCERIES_ID, rows.first().budget.categoryId.toString())
        // Бюджет на все категории: имени нет, и строка покажет «Все категории».
        assertNull(rows.last().categoryName)
        assertEquals(BudgetLevel.OVER, rows.first().level)
        assertEquals(BudgetLevel.OK, rows.last().level)
    }

    @Test
    fun eightyFivePercentIsNearLimit() = runTest {
        enqueueList(BUDGETS_NEAR)

        createModel()
        val rows = (settle() as BudgetsUiState.Ready).rows

        assertEquals(BudgetLevel.NEAR, rows.single().level)
    }

    @Test
    fun emptyListIsReadyNotFailure() = runTest {
        enqueueList(BUDGETS_EMPTY)

        createModel()

        assertEquals(BudgetsUiState.Ready(emptyList(), BudgetFilter.TODAY), settle())
    }

    // «Сегодня» считает сервер: после полуночи ответ уже про другой набор бюджетов.
    @Test
    fun revalidateAfterMidnightReloadsList() = runTest {
        enqueueList()
        createModel()
        settle()

        today = LocalDate.parse("2026-09-08")
        enqueueList(BUDGETS_EMPTY)
        model.revalidate()
        settle()

        assertEquals(4, server.requestCount)
    }

    @Test
    fun revalidateWithinSameDayKeepsList() = runTest {
        enqueueList()
        createModel()
        settle()

        model.revalidate()

        assertEquals(2, server.requestCount)
    }

    @Test
    fun networkFailureIsReported() = runTest {
        server.close()

        createModel()

        assertEquals(BudgetsUiState.Failure(UiError.Network), settle())
    }

    /** Список пришёл, а справочник — нет: строка без имени категории неотличима от общей. */
    @Test
    fun categoriesFailureFailsTheScreen() = runTest {
        server.enqueueJson(200, BUDGETS_LIST)
        server.enqueueJson(500, INTERNAL_ERROR)

        createModel()

        assertEquals(BudgetsUiState.Failure(UiError.Server("всё сломалось")), settle())
    }
}
