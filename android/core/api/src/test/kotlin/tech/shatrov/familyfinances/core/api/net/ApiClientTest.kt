package tech.shatrov.familyfinances.core.api.net

import kotlinx.coroutines.test.runTest
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.LoginRequest
import tech.shatrov.familyfinances.core.api.auth.InMemoryTokenVault
import java.time.LocalDate
import java.time.OffsetDateTime

private const val LOGIN_OK = """
{"data":{"token":"t-1","expires_at":"2027-03-06T10:00:00Z",
"user":{"id":"11111111-1111-1111-1111-111111111111","email":"admin@test.com","first_name":"Админ",
"last_name":"Тест","role":"admin","is_active":true,"created_at":"2026-09-07T10:00:00Z",
"updated_at":"2026-09-07T10:00:00Z"}},
"meta":{"request_id":"r-1","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

private const val TRANSACTIONS_OK = """
{"data":[{"id":"44444444-4444-4444-4444-444444444444","amount_minor":25000,"type":"expense",
"description":"продукты","category_id":"22222222-2222-2222-2222-222222222222",
"user_id":"11111111-1111-1111-1111-111111111111","date":"2026-09-07","tags":[],
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-2","timestamp":"2026-09-07T10:00:00Z",
"pagination":{"limit":50,"offset":0,"total":1}}}
"""

class ApiClientTest {
    private lateinit var server: MockWebServer
    private lateinit var graph: ApiGraph

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
        graph = ApiGraph(server.url("/").toString(), InMemoryTokenVault())
    }

    @After
    fun stop() {
        server.close()
    }

    private fun enqueue(
        code: Int,
        body: String,
        retryAfter: String? = null,
    ) {
        val response = MockResponse.Builder()
            .code(code)
            .body(body)
            .setHeader("Content-Type", "application/json")
        if (retryAfter != null) {
            response.setHeader("Retry-After", retryAfter)
        }
        server.enqueue(response.build())
    }

    private suspend fun login() = graph.client
        .unwrap { graph.auth.login(LoginRequest(email = "admin@test.com", password = "Admin1234!")) }
        .`data`

    private suspend fun expectFailure(block: suspend () -> Unit): ApiFailure = try {
        block()
        throw AssertionError("ожидался ApiFailure")
    } catch (failure: ApiFailure) {
        failure
    }

    @Test
    fun unwrapsEnvelopeOnSuccess() = runTest {
        enqueue(200, LOGIN_OK)

        val data = login()

        assertEquals("t-1", data.token)
        assertEquals(OffsetDateTime.parse("2027-03-06T10:00:00Z"), data.expiresAt)
        assertEquals("admin@test.com", data.user.email)
        assertEquals("/api/v1/auth/login", server.takeRequest().url.encodedPath)
    }

    // Ловит незарегистрированный сериализатор: `meta.timestamp` есть в каждом ответе, а
    // календарная дата — только у транзакции.
    @Test
    fun parsesCalendarDateAndMetaTimestamp() = runTest {
        enqueue(200, TRANSACTIONS_OK)

        val data = graph.client.unwrap { graph.transactions.listTransactions() }.`data`

        assertEquals(LocalDate.of(2026, 9, 7), data.single().date)
    }

    @Test
    fun emptyBodyOperationSucceeds() = runTest {
        server.enqueue(MockResponse.Builder().code(204).build())

        graph.client.send { graph.auth.logout() }

        assertEquals("/api/v1/auth/logout", server.takeRequest().url.encodedPath)
    }

    @Test
    fun unauthorizedBecomesApiFailure() = runTest {
        enqueue(401, """{"error":{"code":"UNAUTHORIZED","message":"токен истёк"}}""")

        val failure = expectFailure { login() } as ApiFailure.Api

        assertTrue(failure.isUnauthorized)
        assertEquals(ApiErrorCode.UNAUTHORIZED, failure.code)
    }

    @Test
    fun forbiddenKeepsStatusAndCode() = runTest {
        enqueue(403, """{"error":{"code":"FORBIDDEN","message":"нужна роль admin"}}""")

        val failure = expectFailure { login() } as ApiFailure.Api

        assertEquals(403, failure.status)
        assertEquals(ApiErrorCode.FORBIDDEN, failure.code)
    }

    @Test
    fun notFoundKeepsStatus() = runTest {
        enqueue(404, """{"error":{"code":"NOT_FOUND","message":"нет такого пути"}}""")

        assertEquals(404, (expectFailure { login() } as ApiFailure.Api).status)
    }

    @Test
    fun conflictReportsSetupRequired() = runTest {
        enqueue(409, """{"error":{"code":"SETUP_REQUIRED","message":"сервис не настроен"}}""")

        assertTrue((expectFailure { login() } as ApiFailure.Api).isSetupRequired)
    }

    @Test
    fun validationErrorCarriesFieldDetails() = runTest {
        enqueue(
            422,
            """{"error":{"code":"VALIDATION_ERROR","message":"не прошло валидацию",
            "details":[{"field":"email","message":"обязательное поле","code":"required"}]}}""",
        )

        val failure = expectFailure { login() } as ApiFailure.Api

        assertEquals("email", failure.details.single().`field`)
    }

    @Test
    fun rateLimitedCarriesRetryAfter() = runTest {
        enqueue(
            429,
            """{"error":{"code":"RATE_LIMITED","message":"слишком много попыток"}}""",
            retryAfter = "60",
        )

        val failure = expectFailure { login() } as ApiFailure.Api

        assertEquals(60, failure.retryAfterSeconds)
    }

    @Test
    fun proxyErrorWithoutEnvelopeIsMalformed() = runTest {
        server.enqueue(MockResponse.Builder().code(502).body("<html>Bad Gateway</html>").build())

        val failure = expectFailure { login() } as ApiFailure.Malformed

        assertEquals(502, failure.status)
    }

    @Test
    fun brokenConnectionIsNetworkFailure() = runTest {
        server.close()

        val failure = expectFailure { login() }

        assertTrue(failure is ApiFailure.Network)
    }
}
