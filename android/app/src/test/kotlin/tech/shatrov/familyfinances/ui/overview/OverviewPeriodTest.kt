package tech.shatrov.familyfinances.ui.overview

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test
import java.time.LocalDate
import java.time.YearMonth

/** Границы чипов по таблице плана: 1-е, 31-е и 29 февраля — края, где месяц считается неверно чаще всего. */
class OverviewPeriodTest {
    private fun range(
        from: String,
        to: String,
    ) = DateRange(LocalDate.parse(from), LocalDate.parse(to))

    @Test
    fun thisMonthRunsFromFirstToToday() {
        assertEquals(range("2026-09-01", "2026-09-01"), OverviewPeriod.ThisMonth.bounds(LocalDate.of(2026, 9, 1)))
        assertEquals(range("2026-12-01", "2026-12-31"), OverviewPeriod.ThisMonth.bounds(LocalDate.of(2026, 12, 31)))
        assertEquals(range("2028-02-01", "2028-02-29"), OverviewPeriod.ThisMonth.bounds(LocalDate.of(2028, 2, 29)))
    }

    @Test
    fun prevMonthIsWholeMonth() {
        assertEquals(range("2026-12-01", "2026-12-31"), OverviewPeriod.PrevMonth.bounds(LocalDate.of(2027, 1, 1)))
        assertEquals(range("2028-02-01", "2028-02-29"), OverviewPeriod.PrevMonth.bounds(LocalDate.of(2028, 3, 31)))
        assertEquals(range("2028-01-01", "2028-01-31"), OverviewPeriod.PrevMonth.bounds(LocalDate.of(2028, 2, 29)))
    }

    @Test
    fun threeMonthsStartsTwoMonthsBack() {
        assertEquals(range("2026-11-01", "2027-01-01"), OverviewPeriod.ThreeMonths.bounds(LocalDate.of(2027, 1, 1)))
        assertEquals(range("2028-01-01", "2028-03-31"), OverviewPeriod.ThreeMonths.bounds(LocalDate.of(2028, 3, 31)))
        assertEquals(range("2027-12-01", "2028-02-29"), OverviewPeriod.ThreeMonths.bounds(LocalDate.of(2028, 2, 29)))
    }

    @Test
    fun yearStartsElevenMonthsBack() {
        assertEquals(range("2026-02-01", "2027-01-01"), OverviewPeriod.Year.bounds(LocalDate.of(2027, 1, 1)))
        assertEquals(range("2027-04-01", "2028-03-31"), OverviewPeriod.Year.bounds(LocalDate.of(2028, 3, 31)))
        assertEquals(range("2027-03-01", "2028-02-29"), OverviewPeriod.Year.bounds(LocalDate.of(2028, 2, 29)))
    }

    @Test
    fun pickedMonthIsWholeUnlessCurrent() {
        val today = LocalDate.of(2028, 3, 31)
        assertEquals(range("2028-02-01", "2028-02-29"), OverviewPeriod.Month(YearMonth.of(2028, 2)).bounds(today))
        assertEquals(
            range("2028-02-01", "2028-02-10"),
            OverviewPeriod.Month(YearMonth.of(2028, 2)).bounds(LocalDate.of(2028, 2, 10)),
        )
        assertEquals(range("2028-03-01", "2028-03-31"), OverviewPeriod.Month(YearMonth.of(2028, 3)).bounds(today))
    }

    @Test
    fun saveKeyRoundTripsWithoutColons() {
        val periods = OverviewPeriod.chips + OverviewPeriod.Month(YearMonth.of(2026, 9))
        for (period in periods) {
            assertFalse(period.saveKey(), period.saveKey().contains(':'))
            assertEquals(period, OverviewPeriod.fromSaveKey(period.saveKey()))
        }
        assertEquals("MONTH-2026-09", OverviewPeriod.Month(YearMonth.of(2026, 9)).saveKey())
    }

    @Test(expected = IllegalArgumentException::class)
    fun unknownSaveKeyThrows() {
        OverviewPeriod.fromSaveKey("WEEK")
    }
}
