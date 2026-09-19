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

    /** Явный диапазон [TransactionFilters.from]…[TransactionFilters.to]: расшифровка «Обзора» с датами его `summary`. */
    RANGE,
}

/**
 * Что просим у сервера: фильтрация целиком в query, потому что на клиенте лежит только
 * загруженная часть списка. Границы периода считаются от `today` в зоне семьи.
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
    val from: LocalDate? = null,
    val to: LocalDate? = null,
) {
    fun dateFrom(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.withDayOfMonth(1)
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).withDayOfMonth(1)
        TransactionPeriod.MONTH -> month?.atDay(1)
        TransactionPeriod.RANGE -> from
    }

    fun dateTo(today: LocalDate): LocalDate? = when (period) {
        TransactionPeriod.ALL -> null
        TransactionPeriod.THIS_MONTH -> today.with(TemporalAdjusters.lastDayOfMonth())
        TransactionPeriod.PREV_MONTH -> today.minusMonths(1).with(TemporalAdjusters.lastDayOfMonth())
        TransactionPeriod.MONTH -> month?.atEndOfMonth()
        TransactionPeriod.RANGE -> to
    }

    fun withPeriod(next: TransactionPeriod): TransactionFilters =
        copy(period = next, month = null, from = null, to = null)

    fun withAccount(id: UUID?): TransactionFilters = copy(accountId = id, unassigned = false)

    companion object {
        /** «Операции месяца» сверки: её приход и расход считаются по всем операциям месяца, без счёта. */
        fun reconciliation(month: YearMonth): TransactionFilters =
            TransactionFilters(period = TransactionPeriod.MONTH, month = month)

        /** Расшифровка строки «Обзора»: те же даты, что ушли в `summary`, иначе суммы не сойдутся. */
        fun overview(
            type: TransactionType,
            categoryId: UUID,
            from: LocalDate,
            to: LocalDate,
        ): TransactionFilters = TransactionFilters(
            period = TransactionPeriod.RANGE,
            type = type,
            categoryId = categoryId,
            from = from,
            to = to,
        )
    }
}
