package tech.shatrov.familyfinances.ui.categories

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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.CATEGORIES_NESTED
import tech.shatrov.familyfinances.CATEGORY_OK
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.MILK_ID
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.util.UUID

/** Категории: список с вложенностью, создание, правка и удаление, скрытое от `member`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class CategoriesViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: CategoriesViewModel

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

    private fun createModel(isAdmin: Boolean = true) {
        model = CategoriesViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())), isAdmin)
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun ready(): CategoriesUiState.Ready =
        model.state.first { it !is CategoriesUiState.Loading } as CategoriesUiState.Ready

    private suspend fun settled(): CategoryEditUiState? = model.editor.first { it?.submitting != true }

    /** Пропускает список и предыдущие попытки: интересен всегда последний запрос. */
    private fun lastRequest(count: Int): RecordedRequest = (1..count).map { server.takeRequest() }.last()

    @Test
    fun listSplitsTypesAndNestsChildren() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)

        createModel()
        val state = ready()

        assertEquals(listOf("Продукты"), state.expense.map { it.category.name })
        assertEquals(listOf("Молочное"), state.expense.single().children.map { it.name })
        assertEquals(listOf("Зарплата"), state.income.map { it.category.name })
        assertTrue(state.income.single().children.isEmpty())
    }

    @Test
    fun createSendsDraftIdTypeAndParent() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        ready()

        model.onAdd()
        model.onNameChange("Кафе")
        model.onIconChange("cup")
        model.onColorChange(CategoryPalette.last())
        // Родителем предлагаются только корневые того же типа: подкатегория в список не попадает.
        assertEquals(listOf("Продукты"), model.editor.value?.parents?.map { it.name })
        model.onParentChange(UUID.fromString(GROCERIES_ID))
        val draft = model.editor.value?.draft

        server.enqueueJson(201, CATEGORY_OK)
        server.enqueueJson(200, CATEGORIES_NESTED)
        model.onSubmit()
        assertNull(settled())

        val request = lastRequest(2)
        assertEquals("POST", request.method)
        assertEquals("/api/v1/categories", request.url.encodedPath)
        val body = request.text()
        assertTrue(body, body.contains("\"id\":\"$draft\""))
        assertTrue(body, body.contains("\"name\":\"Кафе\""))
        assertTrue(body, body.contains("\"type\":\"expense\""))
        assertTrue(body, body.contains("\"color\":\"${CategoryPalette.last()}\""))
        assertTrue(body, body.contains("\"parent_id\":\"$GROCERIES_ID\""))
    }

    /** Смена типа снимает родителя: под доходным типом расходных корней нет. */
    @Test
    fun typeChangeResetsParent() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        ready()

        model.onAdd()
        model.onParentChange(UUID.fromString(GROCERIES_ID))
        model.onTypeChange(CategoryType.income)

        val editor = model.editor.value
        assertNotNull(editor)
        assertNull(editor?.parentId)
        assertEquals(listOf("Зарплата"), editor?.parents?.map { it.name })
    }

    @Test
    fun editSendsPutWithoutTypeAndParent() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        val state = ready()

        model.onOpen(state.expense.single().category)
        assertEquals("Продукты", model.editor.value?.name)
        assertTrue(model.editor.value?.editing == true)
        model.onNameChange("Еда")

        server.enqueueJson(200, CATEGORY_OK)
        server.enqueueJson(200, CATEGORIES_NESTED)
        model.onSubmit()
        assertNull(settled())

        val request = lastRequest(2)
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/categories/$GROCERIES_ID", request.url.encodedPath)
        val body = request.text()
        assertTrue(body, body.contains("\"name\":\"Еда\""))
        // Тип и родитель после создания не меняются, поэтому в теле правки их нет.
        assertFalse(body, body.contains("\"type\""))
        assertFalse(body, body.contains("\"parent_id\""))
    }

    @Test
    fun adminDeletesCategory() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        val state = ready()

        model.onOpen(state.expense.single().children.single())
        assertTrue(model.editor.value?.canDelete == true)

        server.enqueueJson(204, "")
        server.enqueueJson(200, CATEGORIES_NESTED)
        model.onDelete()
        assertNull(settled())

        val request = lastRequest(2)
        assertEquals("DELETE", request.method)
        assertEquals("/api/v1/categories/$MILK_ID", request.url.encodedPath)
    }

    // У member кнопки удаления нет вовсе; вызов, если он всё же случится, до сети не доходит.
    @Test
    fun memberHasNoDelete() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel(isAdmin = false)
        val state = ready()

        model.onOpen(state.expense.single().category)
        assertFalse(model.editor.value?.canDelete == true)

        model.onDelete()
        assertNotNull(model.editor.value)
        assertEquals(1, server.requestCount)
    }

    // 422 ложится под поля формы: общего текста при этом нет, чинить нужно именно поле.
    @Test
    fun validationErrorLandsUnderField() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        ready()
        model.onAdd()
        model.onNameChange("Еда")

        server.enqueueJson(
            422,
            """{"error":{"code":"VALIDATION_ERROR","message":"проверьте поля",
            "details":[{"field":"name","message":"уже занято","code":"conflict"}]}}""",
        )
        model.onSubmit()
        val form = requireNotNull(settled())

        assertEquals("уже занято", form.fieldErrors[CategoryField.NAME])
        assertNull(form.error)

        // Правка поля гасит ошибку под ним: она была про прошлую попытку.
        model.onNameChange("Еда и напитки")
        assertTrue(requireNotNull(model.editor.value).fieldErrors.isEmpty())
    }

    // Отказ не под полем остаётся общим текстом, иначе он пропал бы вовсе.
    @Test
    fun serverFailureKeepsFormOpen() = runTest {
        server.enqueueJson(200, CATEGORIES_NESTED)
        createModel()
        ready()
        model.onAdd()
        model.onNameChange("Еда")

        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")
        model.onSubmit()
        val form = requireNotNull(settled())

        assertEquals(UiError.Server("всё сломалось"), form.error)
        assertTrue(form.fieldErrors.isEmpty())
    }

    /** Тело записанного запроса: у `RecordedRequest` оно необязательное. */
    private fun RecordedRequest.text(): String = body?.utf8().orEmpty()
}
