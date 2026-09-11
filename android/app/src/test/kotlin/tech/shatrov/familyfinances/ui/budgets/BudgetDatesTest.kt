package tech.shatrov.familyfinances.ui.budgets

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import java.time.LocalDate

/** Пресет конца периода: обе границы включительно, короткий месяц конец не удлиняет. */
class BudgetDatesTest {
    @Test
    fun weeklyEndsSixDaysLater() {
        assertEquals(
            LocalDate.parse("2026-09-13"),
            endOf(BudgetPeriod.weekly, LocalDate.parse("2026-09-07")),
        )
    }

    @Test
    fun monthlyKeepsTheDayBeforeTheNextMonthsStart() {
        assertEquals(
            LocalDate.parse("2026-10-29"),
            endOf(BudgetPeriod.monthly, LocalDate.parse("2026-09-30")),
        )
    }

    @Test
    fun yearlyEndsInTheNextYear() {
        assertEquals(
            LocalDate.parse("2027-12-30"),
            endOf(BudgetPeriod.yearly, LocalDate.parse("2026-12-31")),
        )
    }

    @Test
    fun customLeavesTheEndAlone() {
        assertNull(endOf(BudgetPeriod.custom, LocalDate.parse("2026-09-07")))
    }
}
