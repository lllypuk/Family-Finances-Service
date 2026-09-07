package tech.shatrov.familyfinances.ui.login

import android.content.Context
import androidx.test.core.app.ApplicationProvider
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
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.LOGIN_OK
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson

/**
 * Вход: токен доезжает до хранилища, а отказы различаются по виду — их чинят по-разному.
 * Robolectric нужен только ради ресурсов: тексты ошибок живут в `strings.xml`.
 */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class LoginViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var vault: FakeTokenVault
    private lateinit var model: LoginViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        vault = FakeTokenVault()
        model = LoginViewModel(ApiGraph(server.url("/").toString(), vault))
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun fillCredentials() {
        model.onEmailChange("admin@test.com")
        model.onPasswordChange("Admin1234!")
    }

    /** Ответ приходит с сетевого потока, поэтому итог ждём по состоянию, а не по планировщику. */
    private suspend fun submitAndSettle(): LoginUiState {
        model.onSubmit()
        return model.state.first { !it.submitting }
    }

    @Test
    fun emptyFormCannotBeSubmitted() {
        assertFalse(model.state.value.canSubmit)

        model.onEmailChange("admin@test.com")

        assertFalse(model.state.value.canSubmit)
    }

    @Test
    fun successStoresTokenAndSignsIn() = runTest {
        server.enqueueJson(200, LOGIN_OK)
        fillCredentials()

        val state = submitAndSettle()

        assertTrue(state.signedIn)
        assertEquals("t-new", vault.read()?.token)
        // Пароль в состоянии не остаётся: экран уже сменился, а он живёт до смерти процесса.
        assertEquals("", state.password)
        assertEquals("/api/v1/auth/login", server.takeRequest().url.encodedPath)
    }

    @Test
    fun invalidCredentialsAreReported() = runTest {
        server.enqueueJson(401, """{"error":{"code":"INVALID_CREDENTIALS","message":"неверно"}}""")
        fillCredentials()

        val state = submitAndSettle()

        assertEquals(LoginError.InvalidCredentials, state.error)
        assertFalse(state.signedIn)
        assertNull(vault.read())
    }

    @Test
    fun rateLimitCarriesWait() = runTest {
        server.enqueueJson(
            429,
            """{"error":{"code":"RATE_LIMITED","message":"слишком много попыток"}}""",
            retryAfter = "60",
        )
        fillCredentials()

        val state = submitAndSettle()

        assertEquals(LoginError.RateLimited(60), state.error)
    }

    @Test
    fun setupRequiredIsItsOwnState() = runTest {
        server.enqueueJson(409, """{"error":{"code":"SETUP_REQUIRED","message":"не настроен"}}""")
        fillCredentials()

        assertEquals(LoginError.SetupRequired, submitAndSettle().error)
    }

    @Test
    fun serverErrorKeepsServerMessage() = runTest {
        server.enqueueJson(500, """{"error":{"code":"INTERNAL","message":"всё сломалось"}}""")
        fillCredentials()

        assertEquals(LoginError.Server("всё сломалось"), submitAndSettle().error)
    }

    @Test
    fun networkFailureIsReported() = runTest {
        server.close()
        fillCredentials()

        assertEquals(LoginError.Network, submitAndSettle().error)
    }

    @Test
    fun editingClearsPreviousError() = runTest {
        server.enqueueJson(401, """{"error":{"code":"INVALID_CREDENTIALS","message":"неверно"}}""")
        fillCredentials()
        submitAndSettle()

        model.onPasswordChange("Admin4321!")

        assertNull(model.state.value.error)
    }

    /** Три причины — три разных текста: иначе пользователь чинит не то. */
    @Test
    fun errorsReadDifferently() {
        val res = ApplicationProvider.getApplicationContext<Context>().resources
        val texts = listOf(
            LoginError.InvalidCredentials,
            LoginError.RateLimited(60),
            LoginError.RateLimited(null),
            LoginError.SetupRequired,
            LoginError.Network,
            LoginError.Malformed,
        ).map { it.message(res) }

        assertEquals(texts.size, texts.toSet().size)
        assertTrue(texts.none { it.isBlank() })
        assertTrue(LoginError.RateLimited(60).message(res).contains("60"))
        assertNotEquals(LoginError.RateLimited(60).message(res), LoginError.RateLimited(null).message(res))
    }
}
