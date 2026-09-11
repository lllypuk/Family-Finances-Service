package tech.shatrov.familyfinances.ui.settings

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
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.FORBIDDEN_ERROR
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.MEMBER_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.util.UUID

/** Установка пароля админом: `PUT` по чужому пути и неизвестный результат после `5xx`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class UserPasswordViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: UserPasswordViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        model = UserPasswordViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            UUID.fromString(MEMBER_ID),
        )
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun fill(password: String) {
        model.onNewChange(password)
        model.onRepeatChange(password)
    }

    private suspend fun settled(): UserPasswordUiState = model.state.first { !it.submitting }

    /** Роль сняли, пока страница была открыта: страницу закроет хост, а текст свой. */
    @Test
    fun forbiddenClosesThePage() = runTest {
        fill("Password12")
        server.enqueueJson(403, FORBIDDEN_ERROR)
        model.onSubmit()
        val state = settled()

        assertTrue(state.forbidden)
        assertEquals(UiError.Resource(R.string.settings_forbidden), state.error)
    }

    @Test
    fun mismatchAndShortPasswordBlockSending() {
        model.onNewChange("Password12")
        model.onRepeatChange("Password13")
        assertTrue(model.state.value.mismatch)
        assertFalse(model.state.value.canSubmit)

        fill("short")
        assertTrue(model.state.value.lengthInvalid)
        assertFalse(model.state.value.canSubmit)
    }

    @Test
    fun passwordGoesToTheTargetUser() = runTest {
        fill("Password12")

        server.enqueueJson(204, "")
        model.onSubmit()

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/users/$MEMBER_ID/password", request.url.encodedPath)
        assertEquals("""{"new_password":"Password12"}""", request.body?.utf8().orEmpty())
        assertTrue(settled().done)
    }

    @Test
    fun serverErrorLeavesTheResultUnknown() = runTest {
        fill("Password12")

        server.enqueueJson(500, INTERNAL_ERROR)
        model.onSubmit()
        val state = settled()

        assertEquals(UiError.Resource(R.string.settings_user_password_unknown), state.error)
        assertFalse(state.done)
    }
}
