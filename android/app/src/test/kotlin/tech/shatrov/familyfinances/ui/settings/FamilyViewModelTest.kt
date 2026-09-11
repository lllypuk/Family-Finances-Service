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
import tech.shatrov.familyfinances.FAMILY_OK
import tech.shatrov.familyfinances.FORBIDDEN_ERROR
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.ui.UiError

private const val CURRENCY_LOCKED = """
{"error":{"code":"CURRENCY_LOCKED","message":"currency is locked"},
"meta":{"request_id":"r-40","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

private const val TIMEZONE_INVALID = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"timezone","message":"неизвестный часовой пояс","code":"timezone"}]},
"meta":{"request_id":"r-41","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

/** Семья: `PUT /family` из одного поля, перевод `409 CURRENCY_LOCKED` и `422` под полем. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class FamilyViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: FamilyViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        model = FamilyViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            testSession().family,
        )
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private suspend fun settled(): FamilyUiState = model.state.first { !it.submitting }

    /** Роль сняли между заходом и «Сохранить»: текст свой, а страницу закроет хост. */
    @Test
    fun forbiddenClosesThePage() = runTest {
        model.onNameChange("Шатровы")
        server.enqueueJson(403, FORBIDDEN_ERROR)
        model.onSubmit()
        val state = settled()

        assertTrue(state.forbidden)
        assertEquals(UiError.Resource(R.string.settings_forbidden), state.error)
    }

    @Test
    fun unchangedFormHasNothingToSend() {
        assertFalse(model.state.value.canSubmit)

        model.onNameChange("Тестовая семья")
        assertFalse(model.state.value.canSubmit)
    }

    @Test
    fun sendsOnlyTheChangedField() = runTest {
        model.onNameChange("Шатровы")
        assertTrue(model.state.value.canSubmit)

        server.enqueueJson(200, FAMILY_OK)
        model.onSubmit()
        val state = model.state.first { it.saved != null }

        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("/api/v1/family", request.url.encodedPath)
        assertEquals("""{"name":"Шатровы"}""", request.body?.utf8().orEmpty())
        assertEquals("Тестовая семья", state.saved?.name)
    }

    @Test
    fun lockedCurrencyIsTranslatedByCode() = runTest {
        model.onCurrencyChange("EUR")

        server.enqueueJson(409, CURRENCY_LOCKED)
        model.onSubmit()

        assertEquals(UiError.Resource(R.string.settings_error_currency_locked), settled().error)
    }

    @Test
    fun unknownTimezoneGoesUnderTheField() = runTest {
        model.onTimezoneChange("Europe/Атлантида")

        server.enqueueJson(422, TIMEZONE_INVALID)
        model.onSubmit()
        val state = settled()

        assertEquals("неизвестный часовой пояс", state.fieldErrors[FamilyField.TIMEZONE])
        assertEquals(null, state.error)

        model.onTimezoneChange("Europe/Moscow")
        assertTrue(model.state.value.fieldErrors.isEmpty())
    }
}
