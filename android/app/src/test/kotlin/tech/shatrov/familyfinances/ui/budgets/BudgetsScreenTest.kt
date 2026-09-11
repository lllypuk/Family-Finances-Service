package tech.shatrov.familyfinances.ui.budgets

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Budget
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatPeriod
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BudgetsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: BudgetsUiState,
        filter: BudgetFilter = BudgetFilter.TODAY,
        onFilterChange: (BudgetFilter) -> Unit = {},
        onOpen: (UUID) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                BudgetsScreen(
                    state = state,
                    filter = filter,
                    currency = "RUB",
                    onRetry = {},
                    onFilterChange = onFilterChange,
                    onCreate = {},
                    onOpen = onOpen,
                )
            }
        }
    }

    @Test
    fun emptyTodayPointsAtOtherFilter() {
        show(BudgetsUiState.Ready(emptyList()), BudgetFilter.TODAY)

        composeRule.onNodeWithText(res.getString(R.string.budgets_empty_today)).assertIsDisplayed()
    }

    @Test
    fun emptyAllJustSaysSo() {
        show(BudgetsUiState.Ready(emptyList()), BudgetFilter.ALL)

        composeRule.onNodeWithText(res.getString(R.string.budgets_empty)).assertIsDisplayed()
    }

    @Test
    fun rowShowsNameCategoryPeriodAndAmounts() {
        show(BudgetsUiState.Ready(listOf(row(categoryName = "Продукты"))))

        val period = formatPeriod(LocalDate.parse("2026-09-01"), LocalDate.parse("2026-09-30"))
        composeRule.onNodeWithText("Еда").assertIsDisplayed()
        composeRule.onNodeWithText("Продукты · $period").assertIsDisplayed()
        val progress = res.getString(
            R.string.budget_progress,
            formatMoney(3_000_000, "RUB"),
            formatMoney(5_000_000, "RUB"),
        )
        composeRule.onNodeWithText(progress).assertIsDisplayed()
    }

    @Test
    fun budgetWithoutCategorySaysAllCategories() {
        show(BudgetsUiState.Ready(listOf(row(categoryName = null))), BudgetFilter.ALL)

        val period = formatPeriod(LocalDate.parse("2026-09-01"), LocalDate.parse("2026-09-30"))
        composeRule
            .onNodeWithText("${res.getString(R.string.budgets_all_categories)} · $period")
            .assertIsDisplayed()
    }

    @Test
    fun tappingFilterReachesCallback() {
        var filter: BudgetFilter? = null
        show(BudgetsUiState.Ready(listOf(row())), onFilterChange = { filter = it })

        composeRule.onNodeWithText(res.getString(R.string.budgets_filter_all)).performClick()

        assertEquals(BudgetFilter.ALL, filter)
    }

    @Test
    fun tappingRowOpensBudget() {
        var opened: UUID? = null
        val budget = row()
        show(BudgetsUiState.Ready(listOf(budget)), onOpen = { opened = it })

        composeRule.onNodeWithText("Еда").performClick()

        assertEquals(budget.budget.id, opened)
    }

    /** Категория удалена: справочник её не отдал, и выдавать бюджет за общий нельзя. */
    @Test
    fun budgetOfAnUnknownCategoryShowsADash() {
        show(BudgetsUiState.Ready(listOf(row(categoryName = null, categoryId = UUID.randomUUID()))))

        val period = formatPeriod(LocalDate.parse("2026-09-01"), LocalDate.parse("2026-09-30"))
        composeRule.onNodeWithText("— · $period").assertIsDisplayed()
    }

    @Test
    fun failureOffersRetry() {
        show(BudgetsUiState.Failure(UiError.Network))

        composeRule.onNodeWithText(res.getString(R.string.retry)).assertIsDisplayed()
    }

    /** Упавший запрос «на сегодня» иначе не переключить: «Повторить» повторяет ровно его. */
    @Test
    fun filterStaysReachableOnFailure() {
        var picked: BudgetFilter? = null
        show(BudgetsUiState.Failure(UiError.Network), onFilterChange = { picked = it })

        composeRule.onNodeWithText(res.getString(R.string.budgets_filter_all)).performClick()

        assertEquals(BudgetFilter.ALL, picked)
    }

    @Test
    fun filterStaysReachableWhileLoading() {
        show(BudgetsUiState.Loading)

        composeRule.onNodeWithText(res.getString(R.string.budgets_filter_today)).assertIsDisplayed()
    }

    private fun row(
        categoryName: String? = "Продукты",
        utilization: Double = 60.0,
        categoryId: UUID? = if (categoryName == null) null else UUID.randomUUID(),
    ): BudgetRow {
        val stamp = OffsetDateTime.parse("2026-09-07T10:00:00Z")
        return BudgetRow(
            budget = Budget(
                id = UUID.fromString("99999999-9999-9999-9999-999999999991"),
                name = "Еда",
                amountMinor = 5_000_000,
                spentMinor = 3_000_000,
                remainingMinor = 2_000_000,
                utilization = utilization,
                period = BudgetPeriod.monthly,
                startDate = LocalDate.parse("2026-09-01"),
                endDate = LocalDate.parse("2026-09-30"),
                isActive = true,
                createdAt = stamp,
                updatedAt = stamp,
                categoryId = categoryId,
            ),
            categoryName = categoryName,
            level = levelOf(utilization),
        )
    }
}
