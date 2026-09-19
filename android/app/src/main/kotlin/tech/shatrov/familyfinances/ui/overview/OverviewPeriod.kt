package tech.shatrov.familyfinances.ui.overview

import java.time.LocalDate
import java.time.YearMonth

/** Границы периода включительно, как у `summary(from, to)`. */
data class DateRange(
    val from: LocalDate,
    val to: LocalDate,
)

/** Период «Обзора»: четыре чипа и месяц, выбранный тапом по столбику. */
sealed interface OverviewPeriod {
    /** [today] — сегодня в зоне семьи. */
    fun bounds(today: LocalDate): DateRange

    /** Без `:` — ключ встаёт полем в `Saver` списка операций. */
    fun saveKey(): String

    data object ThisMonth : OverviewPeriod {
        override fun bounds(today: LocalDate) = DateRange(today.withDayOfMonth(1), today)

        override fun saveKey() = "THIS_MONTH"
    }

    data object PrevMonth : OverviewPeriod {
        override fun bounds(today: LocalDate): DateRange {
            val month = YearMonth.from(today).minusMonths(1)
            return DateRange(month.atDay(1), month.atEndOfMonth())
        }

        override fun saveKey() = "PREV_MONTH"
    }

    data object ThreeMonths : OverviewPeriod {
        override fun bounds(today: LocalDate) = DateRange(YearMonth.from(today).minusMonths(2).atDay(1), today)

        override fun saveKey() = "THREE_MONTHS"
    }

    /** Совпадает с умолчанием `GET /stats/monthly`: сумма столбиков равна итогу чипа. */
    data object Year : OverviewPeriod {
        override fun bounds(today: LocalDate) = DateRange(YearMonth.from(today).minusMonths(11).atDay(1), today)

        override fun saveKey() = "YEAR"
    }

    /** Текущий месяц обрезается сегодняшним днём, как [ThisMonth]. */
    data class Month(val month: YearMonth) : OverviewPeriod {
        override fun bounds(today: LocalDate): DateRange {
            val end = month.atEndOfMonth()
            return DateRange(month.atDay(1), if (end.isAfter(today)) today else end)
        }

        override fun saveKey() = "$MONTH_PREFIX$month"
    }

    companion object {
        private const val MONTH_PREFIX = "MONTH-"

        val chips: List<OverviewPeriod> = listOf(ThisMonth, PrevMonth, ThreeMonths, Year)

        /** Обратное к [saveKey]; незнакомый ключ — исключение, `restore` ловит его сам. */
        fun fromSaveKey(key: String): OverviewPeriod {
            chips.firstOrNull { it.saveKey() == key }?.let { return it }
            require(key.startsWith(MONTH_PREFIX)) { "unknown overview period: $key" }
            return Month(YearMonth.parse(key.removePrefix(MONTH_PREFIX)))
        }
    }
}
