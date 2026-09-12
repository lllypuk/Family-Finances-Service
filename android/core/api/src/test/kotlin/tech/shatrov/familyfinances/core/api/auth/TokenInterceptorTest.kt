package tech.shatrov.familyfinances.core.api.auth

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.ChangePasswordRequest
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

    private suspend fun changePassword() = graph.client.send {
        graph.me.changePassword(ChangePasswordRequest(currentPassword = "Admin1234!", newPassword = "Admin4321!"))
    }

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

    // Ответ запроса прошлой сессии приходит после нового входа: он не должен ни стирать свежий
    // токен, ни уводить с экрана.
    @Test
    fun unauthorizedKeepsTokenIssuedAfterRequest() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        // Новый вход происходит, пока запрос в полёте: токен ложится в хранилище до того,
        // как интерцептор увидит 401 старого.
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                vault.write(SessionToken("t-2", OffsetDateTime.parse("2027-03-06T10:00:00Z")))
                return MockResponse.Builder()
                    .code(401)
                    .body(UNAUTHORIZED)
                    .setHeader("Content-Type", "application/json")
                    .build()
            }
        }

        runCatching { callMe() }

        assertEquals("t-2", vault.read()?.token)
        assertTrue(expired.isEmpty())
    }

    // Запрос ушёл без токена (Keystore не дал прочитать), а пока он был в полёте, прошёл вход:
    // его 401 — про кончившуюся прошлую сессию, и уводить с экрана новую он не должен.
    @Test
    fun unauthorizedWithoutTokenKeepsSessionOpenedMeanwhile() = runTest {
        vault.clear()
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                vault.write(SessionToken("t-2", OffsetDateTime.parse("2027-03-06T10:00:00Z")))
                return MockResponse.Builder()
                    .code(401)
                    .body(UNAUTHORIZED)
                    .setHeader("Content-Type", "application/json")
                    .build()
            }
        }

        runCatching { callMe() }

        assertEquals("t-2", vault.read()?.token)
        assertTrue(expired.isEmpty())
    }

    // Подписчика в момент отказа не было (экран пересоздавался), а пока событие ждало в канале,
    // пользователь вошёл заново: 401 прошлой сессии не должен увести с экрана свежую.
    @Test
    fun bufferedEventIsDroppedAfterNewLogin() = runTest {
        enqueue(401, UNAUTHORIZED)
        runCatching { callMe() }
        assertNull(vault.read())
        vault.write(SessionToken("t-2", OffsetDateTime.parse("2027-03-06T10:00:00Z")))

        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        advanceUntilIdle()

        assertTrue(expired.isEmpty())
    }

    // Keystore потерял ключ: запрос уходит без заголовка и получает 401 — сессия всё равно
    // кончилась, иначе экран под ней остаётся с вечно падающими повторами.
    @Test
    fun unauthorizedWithoutTokenRaisesEvent() = runTest {
        vault.clear()
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        enqueue(401, UNAUTHORIZED)

        runCatching { callMe() }

        assertEquals(1, expired.size)
    }

    // Неверный текущий пароль: сервер отвечает тем же `401`, но сессия жива — стирать токен
    // и уводить на вход из-за опечатки нельзя.
    @Test
    fun wrongCurrentPasswordKeepsSession() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        enqueue(401, INVALID_CREDENTIALS)

        val failure = runCatching { changePassword() }.exceptionOrNull()

        assertTrue(failure is ApiFailure.Api)
        assertEquals("t-1", vault.read()?.token)
        assertEquals("Bearer t-1", server.takeRequest().headers["Authorization"])
        assertTrue(expired.isEmpty())
    }

    // Тот же путь с истёкшим токеном: это конец сессии, как везде.
    @Test
    fun unauthorizedOnPasswordPathClearsVault() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        enqueue(401, UNAUTHORIZED)

        runCatching { changePassword() }

        assertNull(vault.read())
        assertEquals(1, expired.size)
    }

    // Тело не разобралось (прокси отдал свою страницу): код неизвестен, значит сессия кончилась.
    @Test
    fun unreadableBodyOnPasswordPathClearsVault() = runTest {
        val expired = mutableListOf<Unit>()
        backgroundScope.launch(UnconfinedTestDispatcher(testScheduler)) {
            graph.sessionExpired.toList(expired)
        }
        server.enqueue(MockResponse.Builder().code(401).build())

        runCatching { changePassword() }

        assertNull(vault.read())
        assertEquals(1, expired.size)
    }
}
