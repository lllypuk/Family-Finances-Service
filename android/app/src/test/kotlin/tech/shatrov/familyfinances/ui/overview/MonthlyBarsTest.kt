package tech.shatrov.familyfinances.ui.overview

import org.junit.Assert.assertEquals
import org.junit.Test
import tech.shatrov.familyfinances.core.api.MonthTotals

private const val DELTA = 0.0001f

/** Оба ряда в одном масштабе: столбик дохода и расхода сравнимы по высоте. */
class MonthlyBarsTest {
    @Test
    fun sharedScaleByMaxOfBothSeries() {
        val geometry = barGeometry(listOf(month("2026-08", 100, 50), month("2026-09", 25, 200)))

        assertEquals(listOf(0.5f, 0.125f), geometry.income, DELTA)
        assertEquals(listOf(0.25f, 1f), geometry.expenses, DELTA)
    }

    @Test
    fun zeroMonthHasZeroHeight() {
        val geometry = barGeometry(listOf(month("2026-08", 0, 0), month("2026-09", 300, 0)))

        assertEquals(listOf(0f, 1f), geometry.income, DELTA)
        assertEquals(listOf(0f, 0f), geometry.expenses, DELTA)
    }

    @Test
    fun allZeroSeriesDoesNotDivideByZero() {
        val geometry = barGeometry(listOf(month("2026-08", 0, 0), month("2026-09", 0, 0)))

        assertEquals(listOf(0f, 0f), geometry.income, DELTA)
        assertEquals(listOf(0f, 0f), geometry.expenses, DELTA)
    }

    @Test
    fun emptySeriesIsEmpty() {
        val geometry = barGeometry(emptyList())

        assertEquals(emptyList<Float>(), geometry.income)
    }

    private fun assertEquals(
        expected: List<Float>,
        actual: List<Float>,
        delta: Float,
    ) {
        assertEquals(expected.size, actual.size)
        expected.zip(actual).forEach { (e, a) -> assertEquals(e, a, delta) }
    }
}

internal fun month(
    month: String,
    incomeMinor: Long,
    expensesMinor: Long,
) = MonthTotals(month, incomeMinor, expensesMinor, incomeMinor - expensesMinor, 1)
