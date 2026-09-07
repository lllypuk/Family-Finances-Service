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

internal const val STATS_OK = """
{"data":{"from":"2026-09-01","to":"2026-09-07",
"current":{"from":"2026-09-01","to":"2026-09-07","income_minor":15000000,"expenses_minor":4231050,
"net_minor":10768950,"transaction_count":12},
"previous":{"from":"2026-08-01","to":"2026-08-07","income_minor":12000000,"expenses_minor":5000000,
"net_minor":7000000,"transaction_count":9},
"has_previous_data":true,"income_delta":0.25,"expenses_delta":-0.15,
"expense_categories":[{"category_id":"44444444-4444-4444-4444-444444444444","name":"Продукты",
"amount_minor":3000050,"transaction_count":7,"share":0.709}],
"income_categories":[{"category_id":"55555555-5555-5555-5555-555555555555","name":"Зарплата",
"amount_minor":15000000,"transaction_count":1,"share":1.0}],
"budgets":[{"id":"66666666-6666-6666-6666-666666666666","name":"Еда","amount_minor":5000000,
"spent_minor":3000050,"remaining_minor":1999950,"utilization":0.6,"period":"monthly",
"start_date":"2026-09-01","end_date":"2026-09-30","days_remaining":23,"is_active":true,
"is_over_budget":false,"is_near_limit":false,"category_name":"Продукты"}],
"recent":[{"id":"77777777-7777-7777-7777-777777777777","type":"expense","amount_minor":150000,
"description":"Кофе","date":"2026-09-07","created_at":"2026-09-07T09:00:00Z","category_name":"Кафе"}],
"transactions_total":42},
"meta":{"request_id":"r-4","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** Семья только что создана: сервер шлёт нули и пустые списки, а не отказ. */
internal const val STATS_EMPTY = """
{"data":{"from":"2026-09-01","to":"2026-09-07",
"current":{"from":"2026-09-01","to":"2026-09-07","income_minor":0,"expenses_minor":0,
"net_minor":0,"transaction_count":0},
"previous":{"from":"2026-08-01","to":"2026-08-07","income_minor":0,"expenses_minor":0,
"net_minor":0,"transaction_count":0},
"has_previous_data":false,"income_delta":0,"expenses_delta":0,
"expense_categories":[],"income_categories":[],"budgets":[],"recent":[],"transactions_total":0},
"meta":{"request_id":"r-5","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
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
