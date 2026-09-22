package tech.shatrov.familyfinances.ui.overview

import android.content.Context
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.SALARY_ID
import tech.shatrov.familyfinances.core.api.CategoryShare
import tech.shatrov.familyfinances.core.api.PeriodTotals
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatPercent
import java.time.LocalDate
import java.time.YearMonth
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
// Высокий экран: категории стоят под итогами и графиком, ленивый список ниже края их не создаёт.
@Config(sdk = [ROBOLECTRIC_SDK], qualifiers = "w411dp-h1600dp")
class OverviewScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources
    private val range = DateRange(LocalDate.of(2026, 9, 1), LocalDate.of(2026, 9, 19))
    private val months = listOf(month("2026-08", 150_000_00, 50_000_00), month("2026-09", 150_000_00, 51_000_00))

    private val selected = mutableListOf<OverviewPeriod>()
    private val opened = mutableListOf<Pair<TransactionType, UUID>>()

    private fun show(
        summary: SummaryState = ready(),
        monthly: MonthlyState = MonthlyState.Ready(months),
        period: OverviewPeriod = OverviewPeriod.ThisMonth,
    ) {
        composeRule.setContent {
            AppTheme {
                OverviewScreen(
                    state = OverviewUiState(period, range, "RUB", monthly, summary),
                    onBack = {},
                    onSelect = { selected += it },
                    onRetryMonthly = {},
                    onRetrySummary = {},
                    onOpenCategory = { type, id -> opened += type to id },
                )
            }
        }
    }

    private fun ready(
        count: Int = 12,
        incomeDelta: Double? = 0.04,
    ) = SummaryState.Ready(
        totals = PeriodTotals(range.from, range.to, 312_000_00, 227_700_00, 84_300_00, count),
        incomeDelta = incomeDelta,
        expensesDelta = incomeDelta?.let { -0.11 },
        expenseCategories = if (count ==
            0
        ) {
            emptyList()
        } else {
            listOf(share(GROCERIES_ID, "Продукты", 68_400_00, 0.3, icon = "🛒"))
        },
        incomeCategories = if (count == 0) emptyList() else listOf(share(SALARY_ID, "Зарплата", 290_000_00, 0.93)),
    )

    private fun share(
        id: String,
        name: String,
        amount: Long,
        share: Double,
        icon: String? = null,
    ) = CategoryShare(UUID.fromString(id), name, amount, 1, share, color = icon?.let { "#4caf50" }, icon = icon)

    private fun bar(month: String) = res.getString(
        R.string.overview_bar_description,
        formatMonth(YearMonth.parse(month).atDay(1)),
        formatMoney(150_000_00, "RUB"),
        formatMoney(if (month == "2026-08") 50_000_00 else 51_000_00, "RUB"),
    )

    @Test
    fun chipSelectsPeriod() {
        show()

        composeRule.onNodeWithText(res.getString(R.string.overview_this_month)).assertIsSelected()
        composeRule.onNodeWithText(res.getString(R.string.overview_three_months)).performClick()

        assertEquals(listOf<OverviewPeriod>(OverviewPeriod.ThreeMonths), selected)
    }

    @Test
    fun monthFromBarIsStaticChip() {
        show(period = OverviewPeriod.Month(YearMonth.of(2026, 8)))

        composeRule.onNodeWithText(formatMonth(LocalDate.of(2026, 8, 1))).assertIsSelected()
    }

    @Test
    fun categoriesShowAvatarWithAndWithoutIcon() {
        show()

        composeRule.onNodeWithContentDescription("Продукты", useUnmergedTree = true).assertExists()
        composeRule.onNodeWithContentDescription("Зарплата", useUnmergedTree = true).assertExists()
    }

    @Test
    fun categoryOpensItsTransactions() {
        show()

        composeRule.onNodeWithText("Продукты").performClick()
        composeRule.onNodeWithText("Зарплата").performClick()

        assertEquals(
            listOf(
                TransactionType.expense to UUID.fromString(GROCERIES_ID),
                TransactionType.income to UUID.fromString(SALARY_ID),
            ),
            opened,
        )
    }

    @Test
    fun barSelectsItsMonth() {
        show()

        composeRule.onNodeWithContentDescription(bar("2026-08")).performClick()

        assertEquals(listOf<OverviewPeriod>(OverviewPeriod.Month(YearMonth.of(2026, 8))), selected)
    }

    @Test
    fun totalsShowDeltaWithCaption() {
        show()

        composeRule.onNodeWithText(formatMoney(84_300_00, "RUB", signed = true)).assertIsDisplayed()
        composeRule.onNodeWithText(formatPercent(0.04, signed = true)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.overview_delta_caption)).assertIsDisplayed()
    }

    @Test
    fun noPreviousDataHidesDeltaCaption() {
        show(summary = ready(incomeDelta = null))

        composeRule.onAllNodesWithText(res.getString(R.string.overview_delta_caption)).assertCountEquals(0)
    }

    @Test
    fun emptyPeriodKeepsChart() {
        show(summary = ready(count = 0))

        composeRule.onNodeWithText(res.getString(R.string.overview_empty)).assertIsDisplayed()
        composeRule.onNodeWithContentDescription(bar("2026-09")).assertIsDisplayed()
        composeRule.onAllNodesWithText(res.getString(R.string.overview_expense_categories)).assertCountEquals(0)
    }

    @Test
    fun summaryFailureDoesNotHideChart() {
        show(summary = SummaryState.Failure(UiError.Network))

        composeRule.onNodeWithText(res.getString(R.string.retry)).assertIsDisplayed()
        composeRule.onNodeWithContentDescription(bar("2026-09")).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.overview_year)).assertIsDisplayed()
    }
}
