package tech.shatrov.familyfinances.core.api.auth

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.runTest
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.LoginRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import java.time.OffsetDateTime

private const val UNAUTHORIZED = """{"error":{"code":"UNAUTHORIZED","message":"токен истёк"}}"""
private const val INVALID_CREDENTIALS =
    """{"error":{"code":"INVALID_CREDENTIALS","message":"неверный email или пароль"}}"""

@OptIn(ExperimentalCoroutinesApi::class)
class TokenInterceptorTest {
    private lateinit var server: MockWebServer
    private lateinit var vault: InMemoryTokenVault
    private lateinit var graph: ApiGraph

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
        vault = InMemoryTokenVault(SessionToken("t-1", OffsetDateTime.parse("2027-03-06T10:00:00Z")))
        graph = ApiGraph(server.url("/").toString(), vault)
    }

    @After
    fun stop() {
        server.close()
    }

    private fun enqueue(
        code: Int,
        body: String,
    ) {
        server.enqueue(
            MockResponse.Builder()
                .code(code)
                .body(body)
                .setHeader("Content-Type", "application/json")
                .build(),
        )
    }

    private suspend fun callMe() = graph.client.unwrap { graph.me.getCurrentUser() }.`data`

    @Test
    fun addsBearerHeaderFromVault() = runTest {
        server.enqueue(MockResponse.Builder().code(204).build())

        graph.client.send { graph.auth.logout() }

        assertEquals("Bearer t-1", server.takeRequest().headers["Authorization"])
    }

    @Test
    fun unauthorizedClearsVaultAndRaisesEvent() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        enqueue(401, UNAUTHORIZED)

        val failure = runCatching { callMe() }.exceptionOrNull()

        assertTrue(failure is ApiFailure.Api)
        assertNull(vault.read())
        assertEquals(1, expired.size)
    }

    // Бутстрап мог отвалиться по сети и увести на вход с живым токеном в хранилище: опечатка
    // в пароле не должна стирать его и выбрасывать соседа из работающей сессии.
    @Test
    fun loginFailureKeepsLiveTokenAndRaisesNoEvent() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        enqueue(401, INVALID_CREDENTIALS)

        val failure = runCatching {
            graph.client
                .unwrap { graph.auth.login(LoginRequest(email = "admin@test.com", password = "Admin1234!")) }
                .`data`
        }.exceptionOrNull()

        assertTrue(failure is ApiFailure.Api)
        assertEquals("t-1", vault.read()?.token)
        assertNull(server.takeRequest().headers["Authorization"])
        assertTrue(expired.isEmpty())
    }
}
