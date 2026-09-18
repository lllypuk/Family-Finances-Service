package tech.shatrov.familyfinances.ui.transactions

import tech.shatrov.familyfinances.core.api.TransactionType
import java.time.LocalDate
import java.time.YearMonth
import java.time.temporal.TemporalAdjusters
import java.util.UUID

enum class TransactionPeriod {
    ALL,
    THIS_MONTH,
    PREV_MONTH,

    /** Явный месяц [TransactionFilters.month]: переход со сверки, где месяц выбран не по часам телефона. */
    MONTH,
}

/**
 * Что просим у сервера: фильтрация целиком в query, потому что на клиенте лежит только
 * загруженная часть списка. Границы периода считаются по дате телефона — часовой пояс семьи
 * отличается от него не больше чем на день, а лишнего запроса это не стоит.
 *
 * [unassigned] и [accountId] взаимоисключающие: вместе сервер отвечает `422`.
 */
data class TransactionFilters(
    val period: TransactionPeriod = TransactionPeriod.ALL,
    val type: TransactionType? = null,
    val categoryId: UUID? = null,
    val accountId: UUID? = null,
    val month: YearMonth? = null,
    val unassigned: Boolean = false,
) {
    fun dateFrom(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.withDayOfMonth(1)
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).withDayOfMonth(1)
        TransactionPeriod.MONTH -> month?.atDay(1)
    }

    fun dateTo(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.with(TemporalAdjusters.lastDayOfMonth())
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).with(TemporalAdjusters.lastDayOfMonth())
        TransactionPeriod.MONTH -> month?.atEndOfMonth()
    }

    fun withPeriod(next: TransactionPeriod): TransactionFilters = copy(period = next, month = null)

    fun withAccount(id: UUID?): TransactionFilters = copy(accountId = id, unassigned = false)

    companion object {
        /** Расшифровка строки сверки: те же условия, по которым сервер считает `recorded_minor`. */
        fun reconciliation(
            month: YearMonth,
            accountId: UUID?,
        ): TransactionFilters = TransactionFilters(
            period = TransactionPeriod.MONTH,
            type = TransactionType.expense,
            accountId = accountId,
            month = month,
            unassigned = accountId == null,
        )
    }
}
