package tech.shatrov.familyfinances.ui.networth

import org.junit.Assert.assertEquals
import org.junit.Test

/** Шкала графика всегда включает ноль: отрицательный капитал рисуется под осью, а не сдвигает её. */
class NetWorthChartTest {
    @Test
    fun negativeValuesPutZeroInside() {
        val geometry = chartGeometry(listOf(-100L, 300L))

        assertEquals(0.75f, geometry.zeroY, DELTA)
        assertEquals(listOf(0f, 1f), geometry.points.map { it.x })
        assertEquals(listOf(1f, 0f), geometry.points.map { it.y })
    }

    @Test
    fun positiveValuesSitOnZeroAtBottom() {
        val geometry = chartGeometry(listOf(0L, 50L, 100L))

        assertEquals(1f, geometry.zeroY, DELTA)
        assertEquals(listOf(1f, 0.5f, 0f), geometry.points.map { it.y })
    }

    @Test
    fun singleBucketIsCentredPoint() {
        val geometry = chartGeometry(listOf(-500L))

        assertEquals(0.5f, geometry.points.single().x, DELTA)
        assertEquals(1f, geometry.points.single().y, DELTA)
        assertEquals(0f, geometry.zeroY, DELTA)
    }

    @Test
    fun flatZeroSeriesDoesNotDivideByZero() {
        val geometry = chartGeometry(listOf(0L, 0L))

        assertEquals(listOf(0f, 0f), geometry.points.map { it.y })
    }

    private companion object {
        const val DELTA = 1e-6f
    }
}
