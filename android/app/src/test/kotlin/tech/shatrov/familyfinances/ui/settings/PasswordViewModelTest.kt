package tech.shatrov.familyfinances.ui.settings

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.coroutines.withTimeoutOrNull
import mockwebserver3.MockWebServer
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
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError

/** Неверный текущий пароль: сессия жива, разлогинивать нельзя. */
private const val WRONG_CURRENT = """
{"error":{"code":"INVALID_CREDENTIALS","message":"invalid credentials"},
"meta":{"request_id":"r-32","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

/** Десять байт из пяти букв: политика сервера считает байты UTF-8. */
private const val TEN_BYTES = "ёёёёё"
private const val NINE_BYTES = "ёёёёa"

/** Ожидание события «сессия кончилась»: виртуальное время `runTest` не тратит секунд. */
private const val EVENT_WAIT_MS = 200L

@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class PasswordViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var api: ApiGraph
    private lateinit var model: PasswordViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        api = ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken()))
        model = PasswordViewModel(api)
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun fill(next: String = TEN_BYTES) {
        model.onCurrentChange("Admin1234!")
        model.onNewChange(next)
        model.onRepeatChange(next)
    }

    private suspend fun settled(): PasswordUiState = model.state.first { !it.submitting }

    @Test
    fun lengthIsCountedInBytes() {
        fill(TEN_BYTES)
        assertTrue(model.state.value.canSubmit)

        fill(NINE_BYTES)
        assertFalse(model.state.value.canSubmit)
        assertTrue(model.state.value.lengthInvalid)
    }

    @Test
    fun repeatMustMatch() {
        model.onCurrentChange("Admin1234!")
        model.onNewChange(TEN_BYTES)
        model.onRepeatChange("ёёёёю")

        assertFalse(model.state.value.canSubmit)
        assertTrue(model.state.value.mismatch)
    }

    @Test
    fun successClearsTheForm() = runTest {
        fill()

        server.enqueueJson(204, "")
        model.onSubmit()
        val state = model.state.first { it.changed }

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/me/password", request.url.encodedPath)
        assertEquals(
            """{"current_password":"Admin1234!","new_password":"$TEN_BYTES"}""",
            request.body?.utf8().orEmpty(),
        )
        assertEquals("", state.current)
        assertEquals("", state.next)
        assertEquals("", state.repeat)
    }

    /** `401` про текущий пароль не кончает сессию: токен на месте, события нет. */
    @Test
    fun wrongCurrentPasswordKeepsTheSession() = runTest {
        fill()

        server.enqueueJson(401, WRONG_CURRENT)
        model.onSubmit()
        val state = settled()

        assertTrue(state.currentInvalid)
        assertNull(state.error)
        assertNotNull(api.tokens.read())
        assertNull(withTimeoutOrNull(EVENT_WAIT_MS) { api.sessionExpired.first() })
    }

    /** `5xx` приходит после записи хеша или до неё — различить нечем, повторять нельзя. */
    @Test
    fun serverErrorLeavesTheResultUnknown() = runTest {
        fill()

        server.enqueueJson(500, INTERNAL_ERROR)
        model.onSubmit()

        assertEquals(UiError.Resource(R.string.settings_password_unknown), settled().error)
    }
}
