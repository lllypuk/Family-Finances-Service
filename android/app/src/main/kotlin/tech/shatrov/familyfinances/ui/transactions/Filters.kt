package tech.shatrov.familyfinances.ui.transactions

import tech.shatrov.familyfinances.core.api.TransactionType
import java.time.LocalDate
import java.time.temporal.TemporalAdjusters
import java.util.UUID

enum class TransactionPeriod {
    ALL,
    THIS_MONTH,
    PREV_MONTH,
}

/**
 * Что просим у сервера: фильтрация целиком в query, потому что на клиенте лежит только
 * загруженная часть списка. Границы периода считаются по дате телефона — часовой пояс семьи
 * отличается от него не больше чем на день, а лишнего запроса это не стоит.
 */
data class TransactionFilters(
    val period: TransactionPeriod = TransactionPeriod.ALL,
    val type: TransactionType? = null,
    val categoryId: UUID? = null,
) {
    fun dateFrom(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.withDayOfMonth(1)
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).withDayOfMonth(1)
    }

    fun dateTo(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.with(TemporalAdjusters.lastDayOfMonth())
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).with(TemporalAdjusters.lastDayOfMonth())
    }
}
