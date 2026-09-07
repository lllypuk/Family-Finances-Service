package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ApiFailureTest {
    private val json = Json {
        ignoreUnknownKeys = true
        serializersModule = apiSerializersModule
    }

    private fun parse(
        status: Int,
        body: String?,
        retryAfter: String? = null,
    ) = parseApiFailure(status, body, retryAfter, json)

    @Test
    fun readsValidationDetails() {
        val failure = parse(
            422,
            """
            {"error":{"code":"VALIDATION_ERROR","message":"тело не прошло валидацию",
            "details":[{"field":"amount_minor","message":"должно быть больше нуля","code":"min"}]},
            "meta":{"request_id":"r-1","timestamp":"2026-09-07T10:00:00Z"}}
            """.trimIndent(),
        )

        val api = failure as ApiFailure.Api
        assertEquals(ApiErrorCode.VALIDATION_ERROR, api.code)
        assertEquals(1, api.details.size)
        assertEquals("amount_minor", api.details[0].`field`)
    }

    @Test
    fun readsRetryAfterFromHeader() {
        val failure = parse(
            429,
            """{"error":{"code":"RATE_LIMITED","message":"слишком много попыток"},"meta":{}}""",
            retryAfter = "42",
        )

        assertEquals(42, (failure as ApiFailure.Api).retryAfterSeconds)
    }

    @Test
    fun recognisesSetupRequired() {
        val failure = parse(409, """{"error":{"code":"SETUP_REQUIRED","message":"сервис не настроен"}}""")

        val api = failure as ApiFailure.Api
        assertTrue(api.isSetupRequired)
        assertNull(api.retryAfterSeconds)
        assertTrue(api.details.isEmpty())
    }

    @Test
    fun marksUnauthorized() {
        val failure = parse(401, """{"error":{"code":"UNAUTHORIZED","message":"токен истёк"}}""")

        assertTrue((failure as ApiFailure.Api).isUnauthorized)
    }

    @Test
    fun envelopeFromProxyIsMalformed() {
        val failure = parse(502, "<html><body>Bad Gateway</body></html>")

        assertEquals(502, (failure as ApiFailure.Malformed).status)
    }

    @Test
    fun emptyBodyIsMalformed() {
        assertEquals(500, (parse(500, "") as ApiFailure.Malformed).status)
        assertEquals(503, (parse(503, null) as ApiFailure.Malformed).status)
    }
}
