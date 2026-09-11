package tech.shatrov.familyfinances.ui.budgets

import org.junit.Assert.assertEquals
import org.junit.Test

/** Пороги цвета строки: `utilization` у бюджета — проценты 0…100, обе границы включительно. */
class BudgetLevelTest {
    @Test
    fun belowEightyPercentIsOk() {
        assertEquals(BudgetLevel.OK, levelOf(0.0))
        assertEquals(BudgetLevel.OK, levelOf(79.9))
    }

    @Test
    fun exactlyEightyPercentIsNearLimit() {
        assertEquals(BudgetLevel.NEAR, levelOf(80.0))
    }

    /** Потрачен ровно лимит — перерасход: сводка красит ту же запись так же (`>= 100`). */
    @Test
    fun exactlyFullIsOver() {
        assertEquals(BudgetLevel.OVER, levelOf(100.0))
        assertEquals(BudgetLevel.NEAR, levelOf(99.9))
    }

    @Test
    fun aboveFullIsOver() {
        assertEquals(BudgetLevel.OVER, levelOf(100.1))
    }
}
