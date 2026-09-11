package tech.shatrov.familyfinances.ui.budgets

import tech.shatrov.familyfinances.core.api.BudgetPeriod
import java.time.LocalDate

/** Неделя — начало и ещё шесть дней: обе границы включительно. */
private const val WEEK_TAIL = 6L

/**
 * Конец периода по его началу — пресет формы, а не правило сервера: `period` к датам там не
 * привязан. `custom` конец не подставляет.
 */
fun endOf(
    period: BudgetPeriod,
    start: LocalDate,
): LocalDate? = when (period) {
    BudgetPeriod.weekly -> start.plusDays(WEEK_TAIL)
    BudgetPeriod.monthly -> start.plusMonths(1).minusDays(1)
    BudgetPeriod.yearly -> start.plusYears(1).minusDays(1)
    BudgetPeriod.custom -> null
}
