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
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.ME_OK
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.ui.UiError

private const val EMAIL_TAKEN = """
{"error":{"code":"EMAIL_TAKEN","message":"email already taken"},
"meta":{"request_id":"r-30","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

private const val EMAIL_INVALID = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"email","message":"неверная почта","code":"email"}]},
"meta":{"request_id":"r-31","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

/** Профиль: `PUT /me` из одного поля и перевод `409 EMAIL_TAKEN`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ProfileViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: ProfileViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        model = ProfileViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            testSession().user,
        )
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private suspend fun settled(): ProfileUiState = model.state.first { !it.submitting }

    @Test
    fun unchangedFormHasNothingToSend() {
        assertFalse(model.state.value.canSubmit)

        model.onFirstNameChange("Админ")
        assertFalse(model.state.value.canSubmit)
    }

    @Test
    fun sendsOnlyTheChangedField() = runTest {
        model.onFirstNameChange("Саша")
        assertTrue(model.state.value.canSubmit)

        server.enqueueJson(200, ME_OK)
        model.onSubmit()
        val state = model.state.first { it.saved != null }

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/me", request.url.encodedPath)
        assertEquals("""{"first_name":"Саша"}""", request.body?.utf8().orEmpty())
        assertEquals("Админ", state.saved?.firstName)
    }

    @Test
    fun takenEmailIsTranslatedByCode() = runTest {
        model.onEmailChange("member@test.com")

        server.enqueueJson(409, EMAIL_TAKEN)
        model.onSubmit()

        assertEquals(UiError.Resource(R.string.settings_error_email_taken), settled().error)
    }

    @Test
    fun validationGoesUnderTheField() = runTest {
        model.onEmailChange("не почта")

        server.enqueueJson(422, EMAIL_INVALID)
        model.onSubmit()
        val state = settled()

        assertEquals("неверная почта", state.fieldErrors[ProfileField.EMAIL])
        assertEquals(null, state.error)

        // Правка поля гасит ошибку под ним.
        model.onEmailChange("admin@test.com")
        assertTrue(model.state.value.fieldErrors.isEmpty())
    }
}
