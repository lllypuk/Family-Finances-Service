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
    fun alignedStartSnapsToTheCalendarPeriod() {
        val day = LocalDate.parse("2026-09-17")
        assertEquals(LocalDate.parse("2026-09-01"), alignedStart(BudgetPeriod.monthly, day))
        assertEquals(LocalDate.parse("2026-01-01"), alignedStart(BudgetPeriod.yearly, day))
        // Неделя выравнивания не имеет: серия шагает по +7 дней от любого начала.
        assertEquals(day, alignedStart(BudgetPeriod.weekly, day))
        assertEquals(day, alignedStart(BudgetPeriod.custom, day))
    }

    /** Выровненное начало плюс пресет конца — ровно календарный период, как требует сервер. */
    @Test
    fun alignedStartWithPresetEndCoversTheWholePeriod() {
        val start = alignedStart(BudgetPeriod.monthly, LocalDate.parse("2026-02-17"))
        assertEquals(LocalDate.parse("2026-02-28"), endOf(BudgetPeriod.monthly, start))

        val year = alignedStart(BudgetPeriod.yearly, LocalDate.parse("2026-09-17"))
        assertEquals(LocalDate.parse("2026-12-31"), endOf(BudgetPeriod.yearly, year))
    }

    @Test
    fun customLeavesTheEndAlone() {
        assertNull(endOf(BudgetPeriod.custom, LocalDate.parse("2026-09-07")))
    }
}
