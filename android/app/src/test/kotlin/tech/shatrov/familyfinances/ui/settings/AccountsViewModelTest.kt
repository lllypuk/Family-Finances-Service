package tech.shatrov.familyfinances.ui.settings

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
import tech.shatrov.familyfinances.ACCOUNTS_OK
import tech.shatrov.familyfinances.ACCOUNT_IN_USE_ERROR
import tech.shatrov.familyfinances.ACCOUNT_NAME_EXISTS_ERROR
import tech.shatrov.familyfinances.ACCOUNT_OK
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.FORBIDDEN_ERROR
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError

/** Счета: список с архивом, создание с клиентским `id`, переименование, архив, удаление и их `409`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class AccountsViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: AccountsViewModel

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
        model = AccountsViewModel(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())), isAdmin)
    }

    private suspend fun ready(): AccountsUiState.Ready =
        model.state.first { it !is AccountsUiState.Loading } as AccountsUiState.Ready

    private suspend fun settled(): AccountEditUiState? = model.editor.first { it?.submitting != true }

    /** Пропускает список и предыдущие попытки: интересен всегда последний запрос. */
    private fun lastRequest(count: Int): RecordedRequest = (1..count).map { server.takeRequest() }.last()

    /**
     * Мутация после списка: форма закрыта, перезагрузка списка дочитана. Без ожидания перезагрузки
     * её `GET` приходит во время `server.close()` и на нагруженном раннере роняет его по таймауту.
     */
    private suspend fun mutated(): RecordedRequest {
        assertNull(settled())
        val request = lastRequest(2)
        server.takeRequest()
        ready()
        return request
    }

    private suspend fun openCard() {
        server.enqueueJson(200, ACCOUNTS_OK)
        createModel()
        model.onOpen(ready().active.single())
    }

    @Test
    fun listAsksForArchiveAndSplitsIt() = runTest {
        server.enqueueJson(200, ACCOUNTS_OK)

        createModel()
        val state = ready()

        assertEquals(listOf("Тинькофф"), state.active.map { it.name })
        assertEquals(listOf("Старая карта"), state.archived.map { it.name })
        assertEquals("true", server.takeRequest().url.queryParameter("archived"))
    }

    @Test
    fun loadFailureOffersRetry() = runTest {
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"boom"}}""")
        createModel()

        val failure = model.state.first { it !is AccountsUiState.Loading }
        assertTrue(failure is AccountsUiState.Failure)
        assertFalse((failure as AccountsUiState.Failure).forbidden)
    }

    @Test
    fun createSendsDraftIdAndTrimmedName() = runTest {
        server.enqueueJson(200, ACCOUNTS_OK)
        createModel()
        ready()

        model.onAdd()
        model.onNameChange("  Наличные ")
        val draft = model.editor.value?.draft

        server.enqueueJson(201, ACCOUNT_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        model.onSubmit()

        val request = mutated()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/accounts", request.url.encodedPath)
        val body = request.body?.utf8().orEmpty()
        assertTrue(body, body.contains("\"id\":\"$draft\""))
        assertTrue(body, body.contains("\"name\":\"Наличные\""))
    }

    @Test
    fun nameTakenStaysInFormWithOwnText() = runTest {
        server.enqueueJson(200, ACCOUNTS_OK)
        createModel()
        ready()

        model.onAdd()
        model.onNameChange("Старая карта")
        server.enqueueJson(409, ACCOUNT_NAME_EXISTS_ERROR)
        model.onSubmit()

        val editor = settled()
        assertEquals(UiError.Resource(R.string.settings_error_account_name_exists), editor?.error)
        assertEquals("Старая карта", editor?.name)
    }

    @Test
    fun renameSendsOnlyName() = runTest {
        openCard()
        // Без изменений сохранять нечего: `minProperties: 1`.
        assertFalse(model.editor.value?.canSubmit == true)
        model.onNameChange("Т-Банк")

        server.enqueueJson(200, ACCOUNT_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        model.onSubmit()

        val request = mutated()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/accounts/$CARD_ACCOUNT_ID", request.url.encodedPath)
        assertEquals("""{"name":"Т-Банк"}""", request.body?.utf8())
    }

    @Test
    fun archiveSendsOnlyFlag() = runTest {
        openCard()

        server.enqueueJson(200, ACCOUNT_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        model.onToggleArchive()

        assertEquals("""{"is_archived":true}""", mutated().body?.utf8())
    }

    @Test
    fun deleteInUseExplainsArchive() = runTest {
        openCard()

        server.enqueueJson(409, ACCOUNT_IN_USE_ERROR)
        model.onDelete()

        assertEquals(UiError.Resource(R.string.settings_error_account_in_use), settled()?.error)
        assertEquals("DELETE", lastRequest(2).method)
    }

    @Test
    fun memberCannotDelete() = runTest {
        server.enqueueJson(200, ACCOUNTS_OK)
        createModel(isAdmin = false)
        model.onOpen(ready().active.single())

        assertFalse(model.editor.value?.canDelete == true)
        model.onDelete()
        assertFalse(model.editor.value?.submitting == true)
    }

    /** Админ-роль сняли с другого телефона: удаление отвечает `403`, хост закрывает страницу. */
    @Test
    fun forbiddenClosesPage() = runTest {
        openCard()

        server.enqueueJson(403, FORBIDDEN_ERROR)
        model.onDelete()
        settled()

        assertNull(model.editor.value)
        assertTrue((model.state.value as? AccountsUiState.Failure)?.forbidden == true)
    }
}
