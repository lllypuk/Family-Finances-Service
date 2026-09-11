package tech.shatrov.familyfinances.ui.budgets

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
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
import tech.shatrov.familyfinances.BUDGET_OK
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.FOOD_BUDGET_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.LocalDate
import java.util.UUID

/** Перерасход: сумма меньше потраченного, и повтор её в `PUT` сервер отверг бы. */
private const val OVERSPENT_BUDGET_OK = """
{"data":{"id":"$FOOD_BUDGET_ID","name":"Еда","amount_minor":5000000,"spent_minor":7500000,
"remaining_minor":-2500000,"utilization":150.0,"period":"monthly","start_date":"2026-09-01",
"end_date":"2026-09-30","is_active":true,"category_id":"$GROCERIES_ID",
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-18","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** Бизнес-отказ: поля формы в деталях нет, различить причину нельзя. */
private const val REJECTED_ERROR = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"body","message":"budget period overlaps","code":"conflict"}]},
"meta":{"request_id":"r-19","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

private const val NAME_ERROR = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"name","message":"слишком короткое","code":"min"}]},
"meta":{"request_id":"r-20","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** Форма бюджета: клиентский UUID в `POST`, `PUT` из diff и 422 двух видов. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BudgetEditViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: BudgetEditViewModel

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

    private fun createModel(budgetId: UUID? = null) {
        model = BudgetEditViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            budgetId,
            DRAFT_ID,
            LocalDate.parse("2026-09-15"),
        )
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun loaded(): BudgetEditUiState = model.state.first { !it.loading }

    private suspend fun settled(): BudgetEditUiState = model.state.first { !it.loading && !it.submitting }

    /** Правка существующего бюджета: справочник, затем сам бюджет. */
    private suspend fun editing(budget: String = BUDGET_OK): BudgetEditUiState {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, budget)
        createModel(UUID.fromString(FOOD_BUDGET_ID))
        return loaded()
    }

    /** Пропускает справочник и предыдущие попытки: интересен всегда последний запрос. */
    private fun lastRequest(count: Int): RecordedRequest = (1..count).map { server.takeRequest() }.last()

    @Test
    fun newFormStartsOnAMonthFromToday() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)

        createModel()
        val state = loaded()

        assertFalse(state.editing)
        assertFalse(state.canSubmit)
        assertEquals(LocalDate.parse("2026-09-15"), state.start)
        assertEquals(LocalDate.parse("2026-10-14"), state.end)
        // Бюджет считается по расходам: доходную категорию форма не предлагает.
        assertEquals(listOf("Продукты"), state.categories.map { it.name })
    }

    @Test
    fun periodChangeMovesTheEnd() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()

        model.onPeriodChange(BudgetPeriod.weekly)
        assertEquals(LocalDate.parse("2026-09-21"), model.state.value.end)

        // «Произвольный» конец не трогает: его ставят руками.
        model.onEndChange(LocalDate.parse("2026-09-30"))
        model.onPeriodChange(BudgetPeriod.custom)
        assertEquals(LocalDate.parse("2026-09-30"), model.state.value.end)
    }

    @Test
    fun createSendsDraftIdAndNoCategoryForAllCategories() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        model.onNameChange("Еда")
        model.onAmountChange("50000")

        server.enqueueJson(201, BUDGET_OK)
        model.onSubmit()
        assertTrue(settled().done)

        val request = lastRequest(2)
        assertEquals("POST", request.method)
        assertEquals("/api/v1/budgets", request.url.encodedPath)
        val body = request.text()
        assertTrue(body, body.contains("\"id\":\"$DRAFT_ID\""))
        assertTrue(body, body.contains("\"amount_minor\":5000000"))
        assertTrue(body, body.contains("\"period\":\"monthly\""))
        assertTrue(body, body.contains("\"start_date\":\"2026-09-15\""))
        // «Все категории» — поля в теле нет, а не `null`: иначе сервер получил бы явный null.
        assertFalse(body, body.contains("category_id"))
    }

    @Test
    fun editSendsOnlyChangedFields() = runTest {
        val state = editing()

        assertTrue(state.editing)
        assertEquals("Еда", state.name)
        assertEquals("50000", state.amount)
        assertEquals(UUID.fromString(GROCERIES_ID), state.categoryId)
        // Ничего не изменено — отправлять нечего.
        assertFalse(state.canSubmit)

        model.onNameChange("Продукты")
        assertTrue(model.state.value.canSubmit)

        server.enqueueJson(200, BUDGET_OK)
        model.onSubmit()
        assertTrue(settled().done)

        val request = lastRequest(3)
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/budgets/$FOOD_BUDGET_ID", request.url.encodedPath)
        assertEquals("""{"name":"Продукты"}""", request.text())
    }

    // Сервер сверяет присланную сумму с уже потраченным: неизменённая не уходит и 422 не даёт.
    @Test
    fun overspentBudgetIsRenamedWithoutItsAmount() = runTest {
        editing(OVERSPENT_BUDGET_OK)
        model.onNameChange("Продукты")

        server.enqueueJson(200, OVERSPENT_BUDGET_OK)
        model.onSubmit()
        assertTrue(settled().done)

        val body = lastRequest(3).text()
        assertFalse(body, body.contains("amount_minor"))
    }

    @Test
    fun rejectionWithoutAFieldGetsItsOwnText() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        model.onNameChange("Еда")
        model.onAmountChange("50000")

        server.enqueueJson(422, REJECTED_ERROR)
        model.onSubmit()
        val state = settled()

        assertFalse(state.done)
        assertEquals(UiError.Resource(R.string.budget_error_rejected), state.error)
        assertTrue(state.fieldErrors.isEmpty())
    }

    @Test
    fun fieldDetailsLandUnderFields() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        model.onNameChange("Еда")
        model.onAmountChange("50000")

        server.enqueueJson(422, NAME_ERROR)
        model.onSubmit()
        val state = settled()

        assertEquals(mapOf(BudgetField.NAME to "слишком короткое"), state.fieldErrors)
        // Всё легло под поля, поэтому общего сообщения над кнопкой нет.
        assertNull(state.error)

        model.onNameChange("Продукты")
        assertTrue(model.state.value.fieldErrors.isEmpty())
    }

    @Test
    fun deleteRemovesBudget() = runTest {
        editing()

        server.enqueueJson(204, "")
        model.onDelete()
        assertTrue(settled().done)

        val request = lastRequest(3)
        assertEquals("DELETE", request.method)
        assertEquals("/api/v1/budgets/$FOOD_BUDGET_ID", request.url.encodedPath)
    }

    /** Тело записанного запроса: у `RecordedRequest` оно необязательное. */
    private fun RecordedRequest.text(): String = body?.utf8().orEmpty()

    private companion object {
        val DRAFT_ID: UUID = UUID.fromString("99999999-9999-9999-9999-999999999999")
    }
}
