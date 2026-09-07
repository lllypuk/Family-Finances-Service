package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

class ApiSerializersTest {
    private val json = Json { serializersModule = apiSerializersModule }

    @Test
    fun decodesContextualDateAndId() {
        val transaction = json.decodeFromString(
            Transaction.serializer(),
            """
            {
              "id": "11111111-1111-1111-1111-111111111111",
              "amount_minor": 12345,
              "type": "expense",
              "description": "кофе",
              "category_id": "22222222-2222-2222-2222-222222222222",
              "user_id": "33333333-3333-3333-3333-333333333333",
              "date": "2026-09-07",
              "tags": [],
              "created_at": "2026-09-07T10:00:00Z",
              "updated_at": "2026-09-07T10:00:00Z"
            }
            """.trimIndent(),
        )

        assertEquals(UUID.fromString("11111111-1111-1111-1111-111111111111"), transaction.id)
        assertEquals(LocalDate.of(2026, 9, 7), transaction.date)
        assertEquals(OffsetDateTime.parse("2026-09-07T10:00:00Z"), transaction.createdAt)
        assertEquals(TransactionType.expense, transaction.type)
        assertEquals(12345L, transaction.amountMinor)
    }

    @Test
    fun encodesDateWithoutTimeOrZone() {
        val transaction = Transaction(
            id = UUID.fromString("11111111-1111-1111-1111-111111111111"),
            amountMinor = 1,
            type = TransactionType.income,
            description = "",
            categoryId = UUID.fromString("22222222-2222-2222-2222-222222222222"),
            userId = UUID.fromString("33333333-3333-3333-3333-333333333333"),
            date = LocalDate.of(2026, 1, 2),
            tags = emptyList(),
            createdAt = OffsetDateTime.parse("2026-01-02T03:04:05Z"),
            updatedAt = OffsetDateTime.parse("2026-01-02T03:04:05Z"),
        )

        val encoded = json.encodeToString(Transaction.serializer(), transaction)

        assertEquals(transaction, json.decodeFromString(Transaction.serializer(), encoded))
        assertEquals(true, encoded.contains("\"date\":\"2026-01-02\""))
    }
}
