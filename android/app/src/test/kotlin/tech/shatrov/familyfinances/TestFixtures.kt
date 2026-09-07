package tech.shatrov.familyfinances

import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import tech.shatrov.familyfinances.core.api.auth.SessionToken
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import java.time.OffsetDateTime

/** Robolectric 4.16 знает SDK не выше 36, а модуль собирается под `compileSdk = 37`. */
internal const val ROBOLECTRIC_SDK = 36

/** Хранилище для тестов: Keystore в Robolectric не поднимается. */
internal class FakeTokenVault(private var stored: SessionToken? = null) : TokenVault {
    override fun read(): SessionToken? = stored

    override fun write(token: SessionToken) {
        stored = token
    }

    override fun clear() {
        stored = null
    }
}

internal fun liveToken(token: String = "t-1"): SessionToken = SessionToken(token, OffsetDateTime.now().plusDays(30))

internal const val ME_OK = """
{"data":{"id":"11111111-1111-1111-1111-111111111111","email":"admin@test.com","first_name":"Админ",
"last_name":"Тест","role":"admin","is_active":true,"created_at":"2026-09-07T10:00:00Z",
"updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-1","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

internal const val MEMBER_OK = """
{"data":{"id":"33333333-3333-3333-3333-333333333333","email":"member@test.com","first_name":"Член",
"last_name":"Семьи","role":"member","is_active":true,"created_at":"2026-09-07T10:00:00Z",
"updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-1","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

internal const val FAMILY_OK = """
{"data":{"id":"22222222-2222-2222-2222-222222222222","name":"Тестовая семья","currency":"RUB",
"timezone":"Europe/Moscow","created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-2","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

internal const val LOGIN_OK = """
{"data":{"token":"t-new","expires_at":"2027-03-06T10:00:00Z",
"user":{"id":"11111111-1111-1111-1111-111111111111","email":"admin@test.com","first_name":"Админ",
"last_name":"Тест","role":"admin","is_active":true,"created_at":"2026-09-07T10:00:00Z",
"updated_at":"2026-09-07T10:00:00Z"}},
"meta":{"request_id":"r-3","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

internal fun MockWebServer.enqueueJson(
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
    enqueue(response.build())
}
