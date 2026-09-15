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

/**
 * Начало календарного периода: повторяющийся бюджет сервер принимает только выровненным,
 * иначе `422` с полем `start_date`. `weekly` выравнивать не по чему — серия шагает по +7 дней.
 */
fun alignedStart(
    period: BudgetPeriod,
    date: LocalDate,
): LocalDate = when (period) {
    BudgetPeriod.monthly -> date.withDayOfMonth(1)
    BudgetPeriod.yearly -> date.withDayOfYear(1)
    BudgetPeriod.weekly, BudgetPeriod.custom -> date
}
