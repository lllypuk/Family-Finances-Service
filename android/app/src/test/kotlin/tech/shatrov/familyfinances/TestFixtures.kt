package tech.shatrov.familyfinances

import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import tech.shatrov.familyfinances.core.api.BudgetProgress
import tech.shatrov.familyfinances.core.api.CategoryShare
import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.PeriodTotals
import tech.shatrov.familyfinances.core.api.RecentTransactionItem
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.auth.SessionToken
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import tech.shatrov.familyfinances.core.api.auth.TokenVaultException
import tech.shatrov.familyfinances.ui.recognize.ImportJournalStore
import java.nio.file.Files
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

/** Robolectric 4.16 знает SDK не выше 36, а модуль собирается под `compileSdk = 37`. */
internal const val ROBOLECTRIC_SDK = 36

/** Хранилище для тестов: Keystore в Robolectric не поднимается. */
internal class FakeTokenVault(
    private var stored: SessionToken? = null,
    private val failOnWrite: Boolean = false,
) : TokenVault {
    override fun read(): SessionToken? = stored

    override fun write(token: SessionToken) {
        if (failOnWrite) throw TokenVaultException("тестовый отказ хранилища", null)
        stored = token
    }

    override fun clear() {
        stored = null
    }

    override fun clearIf(token: String): Boolean {
        if (stored?.token != token) return false
        stored = null
        return true
    }
}

/** Журналы импорта во временном каталоге — для графа без Android-контекста. */
internal fun tempJournals(): ImportJournalStore = ImportJournalStore(Files.createTempDirectory("import").toFile())

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

internal const val ADMIN_ID = "11111111-1111-1111-1111-111111111111"
internal const val MEMBER_ID = "33333333-3333-3333-3333-333333333333"
internal const val GROCERIES_ID = "44444444-4444-4444-4444-444444444444"
internal const val SALARY_ID = "55555555-5555-5555-5555-555555555555"

/** Сессия для экранов: бутстрап в тестах модели не гоняем, роль и валюта задаются прямо. */
internal fun testSession(role: Role = Role.admin): Session = Session(
    user = User(
        id = UUID.fromString(if (role == Role.admin) ADMIN_ID else MEMBER_ID),
        email = "admin@test.com",
        firstName = "Админ",
        lastName = "Тест",
        role = role,
        isActive = true,
        createdAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
    ),
    family = Family(
        id = UUID.fromString("22222222-2222-2222-2222-222222222222"),
        name = "Тестовая семья",
        currency = "RUB",
        timezone = "Europe/Moscow",
        createdAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
    ),
)

internal const val CATEGORIES_OK = """
{"data":[
{"id":"$GROCERIES_ID","name":"Продукты","type":"expense","color":"#ff0000","icon":"cart",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
{"id":"$SALARY_ID","name":"Зарплата","type":"income","color":"#00ff00","icon":"wallet",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-6","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

internal const val MILK_ID = "44444444-4444-4444-4444-444444444445"

/** Подкатегория «Молочное» под «Продуктами»: вложенность видна на первом же списке. */
internal const val CATEGORIES_NESTED = """
{"data":[
{"id":"$GROCERIES_ID","name":"Продукты","type":"expense","color":"#ff0000","icon":"cart",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
{"id":"$MILK_ID","name":"Молочное","type":"expense","color":"#ff0001","icon":"milk",
"parent_id":"$GROCERIES_ID",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
{"id":"$SALARY_ID","name":"Зарплата","type":"income","color":"#00ff00","icon":"wallet",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-6","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":3}}}
"""

internal const val CATEGORY_OK = """
{"data":{"id":"$GROCERIES_ID","name":"Продукты","type":"expense","color":"#ff0000","icon":"cart",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-13","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

internal const val SESSION_CURRENT_ID = "99999999-9999-9999-9999-999999999991"
internal const val SESSION_OTHER_ID = "99999999-9999-9999-9999-999999999992"

/** Текущая сессия с именем устройства и старая безымянная — обе ветки строки списка. */
internal const val SESSIONS_OK = """
{"data":[
{"id":"$SESSION_CURRENT_ID","created_at":"2026-09-10T07:30:00Z","last_used_at":"2026-09-11T09:00:00Z",
"expires_at":"2026-10-10T07:30:00Z","current":true,"device_name":"Pixel 8"},
{"id":"$SESSION_OTHER_ID","created_at":"2026-08-01T12:00:00Z","last_used_at":"2026-09-01T12:00:00Z",
"expires_at":"2026-10-01T12:00:00Z","current":false}],
"meta":{"request_id":"r-40","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

/** Тот же список, но сервер знает о пяти сессиях: страница одна, и это видно на экране. */
internal val SESSIONS_TRUNCATED = SESSIONS_OK.replace(""""total":2""", """"total":5""")

/** Два файла: свежий мегабайтный и старый килобайтный — обе ветки `formatBytes`. */
internal const val BACKUPS_OK = """
{"data":[
{"name":"backup_20260911_100000123.db","size_bytes":12876906,"created_at":"2026-09-11T10:00:00Z"},
{"name":"backup_20260910_100000123.db","size_bytes":2048,"created_at":"2026-09-10T07:30:00Z"}],
"meta":{"request_id":"r-50","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

/** Тот же список, но сервер знает о сорока файлах: страница одна, и это видно на экране. */
internal val BACKUPS_TRUNCATED = BACKUPS_OK.replace(""""total":2""", """"total":40""")

internal const val BACKUP_CREATED = """
{"data":{"name":"backup_20260911_120000000.db","size_bytes":12876906,
"created_at":"2026-09-11T12:00:00Z"},
"meta":{"request_id":"r-51","timestamp":"2026-09-11T12:00:00Z","version":"v0.1.0"}}
"""

internal const val USERS_OK = """
{"data":[
{"id":"$ADMIN_ID","email":"admin@test.com","first_name":"Админ","last_name":"Тест","role":"admin",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
{"id":"$MEMBER_ID","email":"member@test.com","first_name":"Член","last_name":"Семьи","role":"member",
"is_active":true,"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-7","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

/** Ответы `PATCH` по участнику: роль поднята и запись выключена — обе ветки формы. */
internal val USER_ADMIN_OK = MEMBER_OK.replace(""""role":"member"""", """"role":"admin"""")

internal val USER_INACTIVE_OK = MEMBER_OK.replace(""""is_active":true""", """"is_active":false""")

/** Тот же список, но сервер знает о трёх пользователях: страница одна, и это видно на экране. */
internal val USERS_TRUNCATED = USERS_OK.replace(""""total":2""", """"total":3""")

/** Две операции за 7 сентября и одна за 6-е: группировка по дате видна на первой же странице. */
internal const val TRANSACTIONS_PAGE_1 = """
{"data":[
{"id":"88888888-8888-8888-8888-888888888881","amount_minor":150000,"type":"expense",
"description":"Кофе","category_id":"$GROCERIES_ID","user_id":"$ADMIN_ID","date":"2026-09-07",
"tags":[],"created_at":"2026-09-07T09:00:00Z","updated_at":"2026-09-07T09:00:00Z"},
{"id":"88888888-8888-8888-8888-888888888882","amount_minor":320000,"type":"expense",
"description":"Хлеб","category_id":"$GROCERIES_ID","user_id":"$MEMBER_ID","date":"2026-09-07",
"tags":[],"created_at":"2026-09-07T08:00:00Z","updated_at":"2026-09-07T08:00:00Z"}],
"meta":{"request_id":"r-8","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":50,"offset":0,"total":3}}}
"""

internal const val TRANSACTIONS_PAGE_2 = """
{"data":[
{"id":"88888888-8888-8888-8888-888888888883","amount_minor":15000000,"type":"income",
"description":"Зарплата","category_id":"$SALARY_ID","user_id":"$ADMIN_ID","date":"2026-09-06",
"tags":[],"created_at":"2026-09-06T08:00:00Z","updated_at":"2026-09-06T08:00:00Z"}],
"meta":{"request_id":"r-9","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":50,"offset":2,"total":3}}}
"""

internal const val TRANSACTIONS_EMPTY = """
{"data":[],
"meta":{"request_id":"r-10","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":50,"offset":0,"total":0}}}
"""

internal const val COFFEE_ID = "88888888-8888-8888-8888-888888888881"

internal const val TRANSACTION_OK = """
{"data":{"id":"$COFFEE_ID","amount_minor":150000,"type":"expense","description":"Кофе",
"category_id":"$GROCERIES_ID","user_id":"$ADMIN_ID","date":"2026-09-07","tags":[],
"created_at":"2026-09-07T09:00:00Z","updated_at":"2026-09-07T09:00:00Z"},
"meta":{"request_id":"r-11","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** 422 с деталями: поле названо так же, как в теле запроса. */
internal const val VALIDATION_ERROR = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"amount_minor","message":"должно быть больше нуля","code":"gt"}]},
"meta":{"request_id":"r-12","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** Роль сняли с другого телефона: админ-роут отвечает так любому подразделу настроек. */
internal const val FORBIDDEN_ERROR = """
{"error":{"code":"FORBIDDEN","message":"admin role required"},
"meta":{"request_id":"r-60","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

internal const val INTERNAL_ERROR = """{"error":{"code":"INTERNAL","message":"всё сломалось"}}"""

internal const val FOOD_BUDGET_ID = "99999999-9999-9999-9999-999999999991"
internal const val ALL_BUDGET_ID = "99999999-9999-9999-9999-999999999992"

internal const val BUDGET_OK = """
{"data":{"id":"$FOOD_BUDGET_ID","name":"Еда","amount_minor":5000000,"spent_minor":3000000,
"remaining_minor":2000000,"utilization":60.0,"period":"monthly","start_date":"2026-09-01",
"end_date":"2026-09-30","is_active":true,"recurring":false,"category_id":"$GROCERIES_ID",
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
"meta":{"request_id":"r-14","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0"}}
"""

/** Перерасходованный бюджет с категорией и общий на все категории: обе строки списка разом. */
internal const val BUDGETS_LIST = """
{"data":[
{"id":"$FOOD_BUDGET_ID","name":"Еда","amount_minor":5000000,"spent_minor":7500000,
"remaining_minor":-2500000,"utilization":150.0,"period":"monthly","start_date":"2026-09-01",
"end_date":"2026-09-30","is_active":true,"recurring":false,"category_id":"$GROCERIES_ID",
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"},
{"id":"$ALL_BUDGET_ID","name":"Всё","amount_minor":10000000,"spent_minor":2000000,
"remaining_minor":8000000,"utilization":20.0,"period":"yearly","start_date":"2026-01-01",
"end_date":"2026-12-31","is_active":true,"recurring":false,
"created_at":"2026-09-07T10:00:00Z","updated_at":"2026-09-07T10:00:00Z"}],
"meta":{"request_id":"r-15","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

internal const val BUDGETS_EMPTY = """
{"data":[],
"meta":{"request_id":"r-16","timestamp":"2026-09-07T10:00:00Z","version":"v0.1.0",
"pagination":{"limit":200,"offset":0,"total":0}}}
"""

/** Сводка для Compose-тестов главной: `STATS_OK` — тот же ответ, но в виде JSON для MockWebServer. */
@Suppress("LongParameterList")
internal fun statsSummary(
    from: LocalDate = LocalDate.parse("2026-09-01"),
    to: LocalDate = LocalDate.parse("2026-09-15"),
    incomeMinor: Long = 15_000_00,
    expensesMinor: Long = 4_231_05,
    transactionCount: Int = 12,
    hasPreviousData: Boolean = true,
    incomeDelta: Double = 0.25,
    expensesDelta: Double = -0.15,
    expenseCategories: List<CategoryShare> = emptyList(),
    budgets: List<BudgetProgress> = emptyList(),
    recent: List<RecentTransactionItem> = emptyList(),
    transactionsTotal: Int = 42,
): StatsSummary = StatsSummary(
    from = from,
    to = to,
    current = PeriodTotals(
        from = from,
        to = to,
        incomeMinor = incomeMinor,
        expensesMinor = expensesMinor,
        netMinor = incomeMinor - expensesMinor,
        transactionCount = transactionCount,
    ),
    previous = PeriodTotals(
        from = from.minusMonths(1),
        to = to.minusMonths(1),
        incomeMinor = 0,
        expensesMinor = 0,
        netMinor = 0,
        transactionCount = 0,
    ),
    hasPreviousData = hasPreviousData,
    incomeDelta = incomeDelta,
    expensesDelta = expensesDelta,
    expenseCategories = expenseCategories,
    incomeCategories = emptyList(),
    budgets = budgets,
    recent = recent,
    transactionsTotal = transactionsTotal,
)

internal const val CARD_ACCOUNT_ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
internal const val OLD_ACCOUNT_ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"

/** Активная карта и перевыпущенная, ушедшая в архив. */
internal const val ACCOUNTS_OK = """
{"data":[
{"id":"$CARD_ACCOUNT_ID","name":"Тинькофф","is_archived":false,
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
{"id":"$OLD_ACCOUNT_ID","name":"Старая карта","is_archived":true,
"created_at":"2026-09-01T10:00:00Z","updated_at":"2026-09-10T10:00:00Z"}],
"meta":{"request_id":"r-70","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

/** [TRANSACTION_OK], привязанная к счёту [accountId]. */
internal fun transactionOnAccount(accountId: String): String =
    TRANSACTION_OK.replace("\"tags\":[]", "\"account_id\":\"$accountId\",\"tags\":[]")

internal const val ACCOUNTS_EMPTY = """
{"data":[],"meta":{"request_id":"r-71","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0",
"pagination":{"limit":200,"offset":0,"total":0}}}
"""

internal const val ACCOUNT_OK = """
{"data":{"id":"$CARD_ACCOUNT_ID","name":"Тинькофф","is_archived":false,
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
"meta":{"request_id":"r-72","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val ACCOUNT_NAME_EXISTS_ERROR = """
{"error":{"code":"ACCOUNT_NAME_EXISTS","message":"account name already exists"},
"meta":{"request_id":"r-73","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val ACCOUNT_IN_USE_ERROR = """
{"error":{"code":"ACCOUNT_IN_USE","message":"account is in use"},
"meta":{"request_id":"r-74","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val CASH_ACCOUNT_ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3"

/**
 * Сверка августа: карта сошлась, наличные не сверены, архивная карта с нулевой сверкой — «N из M»
 * её не считает, иначе вышло бы «2 из 3».
 */
internal const val RECONCILIATION_OK = """
{"data":{"month":"2026-08","unassigned_minor":125000,"accounts":[
{"account":{"id":"$CARD_ACCOUNT_ID","name":"Тинькофф","is_archived":false,
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
"recorded_minor":4310000,"bank_expense_minor":4310000,"diff_minor":0,"note":"выписка",
"updated_at":"2026-09-01T10:00:00Z"},
{"account":{"id":"$CASH_ACCOUNT_ID","name":"Наличные","is_archived":false,
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
"recorded_minor":325000,"bank_expense_minor":null,"diff_minor":null,"note":null,"updated_at":null},
{"account":{"id":"$OLD_ACCOUNT_ID","name":"Старая карта","is_archived":true,
"created_at":"2026-09-01T10:00:00Z","updated_at":"2026-09-10T10:00:00Z"},
"recorded_minor":0,"bank_expense_minor":0,"diff_minor":0,"note":"",
"updated_at":"2026-09-01T10:00:00Z"}]},
"meta":{"request_id":"r-80","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val RECONCILIATION_EMPTY = """
{"data":{"month":"2026-08","unassigned_minor":0,"accounts":[]},
"meta":{"request_id":"r-81","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val RECONCILIATION_PUT_OK = """
{"data":{"account_id":"$CASH_ACCOUNT_ID","month":"2026-08","bank_expense_minor":350000,"note":"",
"updated_at":"2026-09-18T10:00:00Z"},
"meta":{"request_id":"r-82","timestamp":"2026-09-18T10:00:00Z","version":"v0.6.0"}}
"""

internal const val FLAT_ID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1"
internal const val MORTGAGE_ID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2"
internal const val CRYPTO_ID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb3"
internal const val OLD_CAR_ID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb4"

/**
 * Квартира и ипотека со снимками, актив незнакомого клиенту вида без снимка и проданная машина в
 * архиве — её нулевой снимок в капитале остаётся. План есть у квартиры; остальные — в форме сервера
 * до `v0.8.0`, без полей плана.
 */
internal const val HOLDINGS_OK = """
{"data":[
{"id":"$FLAT_ID","name":"Квартира","side":"asset","kind":"property","is_archived":false,
"current":{"date":"2026-01-12","value_minor":1100000000},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z",
"monthly_income_minor":4500000,"monthly_expense_minor":830000,"plan_updated_at":"2026-03-05T10:00:00Z"},
{"id":"$MORTGAGE_ID","name":"Ипотека","side":"liability","kind":"mortgage","is_archived":false,
"current":{"date":"2026-09-01","value_minor":640000000},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
{"id":"$CRYPTO_ID","name":"Биткоин","side":"asset","kind":"crypto","is_archived":false,"current":null,
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
{"id":"$OLD_CAR_ID","name":"Машина","side":"asset","kind":"vehicle","is_archived":true,
"current":{"date":"2026-05-01","value_minor":0},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"}],
"meta":{"request_id":"r-90","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":200,"offset":0,"total":4}}}
"""

/** Только выплаты у действующей ипотеки; план проданной машины в итог не входит. */
internal const val HOLDINGS_EXPENSE_PLAN = """
{"data":[
{"id":"$MORTGAGE_ID","name":"Ипотека","side":"liability","kind":"mortgage","is_archived":false,
"current":{"date":"2026-09-01","value_minor":640000000},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z",
"monthly_income_minor":0,"monthly_expense_minor":7430000,"plan_updated_at":"2026-09-01T10:00:00Z"},
{"id":"$OLD_CAR_ID","name":"Машина","side":"asset","kind":"vehicle","is_archived":true,
"current":{"date":"2026-05-01","value_minor":0},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z",
"monthly_income_minor":0,"monthly_expense_minor":1500000,"plan_updated_at":"2026-01-10T10:00:00Z"}],
"meta":{"request_id":"r-90","timestamp":"2026-09-18T10:00:00Z","version":"v0.8.0",
"pagination":{"limit":200,"offset":0,"total":2}}}
"""

internal const val HOLDING_PLAN_ERROR = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля",
"details":[{"field":"monthly_income_minor","message":"слишком большое число","code":"lte"}]},
"meta":{"request_id":"r-98","timestamp":"2026-09-18T10:00:00Z","version":"v0.8.0"}}
"""

internal const val HOLDINGS_EMPTY = """
{"data":[],"meta":{"request_id":"r-91","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":200,"offset":0,"total":0}}}
"""

/** Ряд по умолчанию: итог шапки — последняя корзина, а не сумма строк списка. */
internal const val NET_WORTH_OK = """
{"data":{"from":"2025-10-01","to":"2026-09-18","months":[
{"month":"2026-08","assets_minor":1100000000,"liabilities_minor":650000000,"net_minor":450000000},
{"month":"2026-09","assets_minor":1100000000,"liabilities_minor":640000000,"net_minor":460000000}]},
"meta":{"request_id":"r-92","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0"}}
"""

internal const val HOLDING_OK = """
{"data":{"id":"$FLAT_ID","name":"Квартира","side":"asset","kind":"property","is_archived":false,
"current":{"date":"2026-01-12","value_minor":1100000000},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"},
"meta":{"request_id":"r-93","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0"}}
"""

internal const val HOLDING_VALUE_OK = """
{"data":{"date":"2026-09-18","value_minor":0,"updated_at":"2026-09-18T10:00:00Z"},
"meta":{"request_id":"r-94","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0"}}
"""

internal const val HOLDING_NAME_EXISTS_ERROR = """
{"error":{"code":"HOLDING_NAME_EXISTS","message":"holding name already exists"},
"meta":{"request_id":"r-95","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0"}}
"""

/** Первая страница истории квартиры: два снимка из трёх, третий — на второй. */
internal const val FLAT_VALUES_PAGE_1 = """
{"data":[
{"date":"2026-01-12","value_minor":1100000000,"updated_at":"2026-01-12T10:00:00Z"},
{"date":"2025-01-10","value_minor":1000000000,"updated_at":"2025-01-10T10:00:00Z"}],
"meta":{"request_id":"r-96","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":2,"offset":0,"total":3}}}
"""

internal const val FLAT_VALUES_PAGE_2 = """
{"data":[
{"date":"2024-01-15","value_minor":950000000,"updated_at":"2024-01-15T10:00:00Z"}],
"meta":{"request_id":"r-97","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":2,"offset":2,"total":3}}}
"""

/** История после удаления январского снимка 2026: `current` квартиры откатился на 2025 год. */
internal const val HOLDINGS_AFTER_VALUE_DELETE = """
{"data":[
{"id":"$FLAT_ID","name":"Квартира","side":"asset","kind":"property","is_archived":false,
"current":{"date":"2025-01-10","value_minor":1000000000},
"created_at":"2026-09-18T10:00:00Z","updated_at":"2026-09-18T10:00:00Z"}],
"meta":{"request_id":"r-98","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":200,"offset":0,"total":1}}}
"""

internal const val FLAT_VALUES_AFTER_DELETE = """
{"data":[
{"date":"2025-01-10","value_minor":1000000000,"updated_at":"2025-01-10T10:00:00Z"},
{"date":"2024-01-15","value_minor":950000000,"updated_at":"2024-01-15T10:00:00Z"}],
"meta":{"request_id":"r-99","timestamp":"2026-09-18T10:00:00Z","version":"v0.7.0",
"pagination":{"limit":50,"offset":0,"total":2}}}
"""

/** Ряд по умолчанию на 19.09.2026: 12 корзин с октября 2025. */
internal const val STATS_MONTHLY_OK = """
{"data":{"from":"2025-10-01","to":"2026-09-19","months":[
{"month":"2025-10","income_minor":0,"expenses_minor":4000000,"net_minor":-4000000,"transaction_count":5},
{"month":"2025-11","income_minor":15000000,"expenses_minor":4100000,"net_minor":10900000,"transaction_count":6},
{"month":"2025-12","income_minor":15000000,"expenses_minor":4200000,"net_minor":10800000,"transaction_count":7},
{"month":"2026-01","income_minor":0,"expenses_minor":4300000,"net_minor":-4300000,"transaction_count":8},
{"month":"2026-02","income_minor":15000000,"expenses_minor":0,"net_minor":15000000,"transaction_count":9},
{"month":"2026-03","income_minor":15000000,"expenses_minor":4500000,"net_minor":10500000,"transaction_count":10},
{"month":"2026-04","income_minor":0,"expenses_minor":4600000,"net_minor":-4600000,"transaction_count":11},
{"month":"2026-05","income_minor":15000000,"expenses_minor":4700000,"net_minor":10300000,"transaction_count":12},
{"month":"2026-06","income_minor":15000000,"expenses_minor":4800000,"net_minor":10200000,"transaction_count":13},
{"month":"2026-07","income_minor":0,"expenses_minor":4900000,"net_minor":-4900000,"transaction_count":14},
{"month":"2026-08","income_minor":15000000,"expenses_minor":5000000,"net_minor":10000000,"transaction_count":15},
{"month":"2026-09","income_minor":15000000,"expenses_minor":5100000,"net_minor":9900000,"transaction_count":16}]},
"meta":{"request_id":"r-40","timestamp":"2026-09-19T10:00:00Z","version":"v0.8.0"}}
"""

/** «3 месяца» на 19.09.2026: перед периодом операций не было, дельты сервер шлёт нулями. */
internal const val STATS_SUMMARY_RANGE_OK = """
{"data":{"from":"2026-07-01","to":"2026-09-19",
"current":{"from":"2026-07-01","to":"2026-09-19","income_minor":45000000,"expenses_minor":13000000,
"net_minor":32000000,"transaction_count":30},
"previous":{"from":"2026-04-11","to":"2026-06-30","income_minor":0,"expenses_minor":0,
"net_minor":0,"transaction_count":0},
"has_previous_data":false,"income_delta":0,"expenses_delta":0,
"expense_categories":[{"category_id":"88888888-8888-8888-8888-888888888888","name":"Ипотека",
"amount_minor":3000000,"transaction_count":3,"share":0.2307},
{"category_id":"44444444-4444-4444-4444-444444444444","name":"Продукты",
"amount_minor":10000000,"transaction_count":27,"share":0.7693}],
"income_categories":[{"category_id":"55555555-5555-5555-5555-555555555555","name":"Зарплата",
"amount_minor":45000000,"transaction_count":3,"share":1.0}],
"budgets":[],"recent":[],"transactions_total":42},
"meta":{"request_id":"r-41","timestamp":"2026-09-19T10:00:00Z","version":"v0.8.0"}}
"""
