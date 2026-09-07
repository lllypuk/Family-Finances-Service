package tech.shatrov.familyfinances.ui.transactions

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
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.COFFEE_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.TRANSACTION_OK
import tech.shatrov.familyfinances.VALIDATION_ERROR
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import java.time.LocalDate
import java.util.UUID

/** Форма транзакции: создание с клиентским UUID, правка, удаление и 422 под полями. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class TransactionEditViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: TransactionEditViewModel

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

    private fun createModel(transactionId: UUID? = null) {
        model = TransactionEditViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            transactionId,
            DRAFT_ID,
            LocalDate.parse("2026-09-15"),
        )
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun loaded(): TransactionEditUiState = model.state.first { !it.loading }

    private suspend fun settled(): TransactionEditUiState = model.state.first { !it.loading && !it.submitting }

    private fun fill() {
        model.onAmountChange("1500,50")
        model.onCategoryChange(UUID.fromString(GROCERIES_ID))
        model.onDescriptionChange(" Кофе ")
    }

    /** Пропускает справочник и предыдущие попытки: интересен всегда последний запрос. */
    private fun lastRequest(count: Int): RecordedRequest = (1..count).map { server.takeRequest() }.last()

    @Test
    fun emptyFormCannotBeSubmitted() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)

        createModel()
        val state = loaded()

        assertFalse(state.canSubmit)
        // Категории показываются под выбранный тип: расходная форма не предлагает «Зарплату».
        assertEquals(listOf("Продукты"), state.visibleCategories.map { it.name })
    }

    @Test
    fun createSendsDraftIdAmountAndDate() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        fill()

        server.enqueueJson(201, TRANSACTION_OK)
        model.onSubmit()
        val state = settled()

        assertTrue(state.done)
        val request = lastRequest(2)
        assertEquals("POST", request.method)
        assertEquals("/api/v1/transactions", request.url.encodedPath)
        val body = request.text()
        assertTrue(body, body.contains("\"id\":\"$DRAFT_ID\""))
        assertTrue(body, body.contains("\"amount_minor\":150050"))
        assertTrue(body, body.contains("\"date\":\"2026-09-15\""))
        assertTrue(body, body.contains("\"description\":\"Кофе\""))
    }

    // Повтор после обрыва уходит с тем же id: сервер отвечает существующей записью, а не создаёт вторую.
    @Test
    fun retryRepeatsTheSameDraftId() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        fill()

        server.enqueueJson(500, INTERNAL_ERROR)
        model.onSubmit()
        val failed = settled()
        assertFalse(failed.done)

        server.enqueueJson(201, TRANSACTION_OK)
        model.onSubmit()
        assertTrue(settled().done)

        val requests = (1..3).map { server.takeRequest() }
        assertTrue(requests[1].text().contains("\"id\":\"$DRAFT_ID\""))
        assertTrue(requests[2].text().contains("\"id\":\"$DRAFT_ID\""))
    }

    @Test
    fun editPrefillsFormAndSendsPut() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, TRANSACTION_OK)

        createModel(UUID.fromString(COFFEE_ID))
        val state = loaded()

        assertTrue(state.editing)
        assertEquals("1500", state.amount)
        assertEquals(TransactionType.expense, state.type)
        assertEquals(UUID.fromString(GROCERIES_ID), state.categoryId)
        assertEquals(LocalDate.parse("2026-09-07"), state.date)

        model.onAmountChange("2000")
        server.enqueueJson(200, TRANSACTION_OK)
        model.onSubmit()
        assertTrue(settled().done)

        val request = lastRequest(3)
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/transactions/$COFFEE_ID", request.url.encodedPath)
        val body = request.text()
        assertTrue(body, body.contains("\"amount_minor\":200000"))
        // Клиентский id в теле правки не нужен: запись адресована путём.
        assertFalse(body, body.contains("\"id\""))
    }

    @Test
    fun deleteRemovesTransaction() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, TRANSACTION_OK)
        createModel(UUID.fromString(COFFEE_ID))
        loaded()

        server.enqueueJson(204, "")
        model.onDelete()
        assertTrue(settled().done)

        val request = lastRequest(3)
        assertEquals("DELETE", request.method)
        assertEquals("/api/v1/transactions/$COFFEE_ID", request.url.encodedPath)
    }

    @Test
    fun validationDetailsLandUnderFields() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        createModel()
        loaded()
        fill()

        server.enqueueJson(422, VALIDATION_ERROR)
        model.onSubmit()
        val state = settled()

        assertFalse(state.done)
        assertEquals(mapOf(TransactionField.AMOUNT to "должно быть больше нуля"), state.fieldErrors)
        // Всё легло под поля, поэтому общего сообщения над кнопкой нет.
        assertNull(state.error)

        model.onAmountChange("20")
        assertTrue(model.state.value.fieldErrors.isEmpty())
    }

    /** Тело записанного запроса: у `RecordedRequest` оно необязательное. */
    private fun RecordedRequest.text(): String = body?.utf8().orEmpty()

    private companion object {
        val DRAFT_ID: UUID = UUID.fromString("99999999-9999-9999-9999-999999999999")
    }
}
