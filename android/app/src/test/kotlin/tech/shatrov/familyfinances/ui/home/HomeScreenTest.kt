package tech.shatrov.familyfinances.ui.home

import android.content.Context
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.statsSummary
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatPercent
import java.time.LocalDate

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HomeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: HomeUiState,
        onAddTransaction: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = state,
                    onRetry = {},
                    onAddTransaction = onAddTransaction,
                    onSettings = {},
                )
            }
        }
    }

    private fun ready(summary: StatsSummary) = HomeUiState.Ready(summary, "RUB")

    @Test
    fun headerShowsMonthAndLastDayOfPeriod() {
        val summary = statsSummary(from = LocalDate.parse("2026-09-01"), to = LocalDate.parse("2026-09-15"))
        show(ready(summary))

        composeRule.onNodeWithText(formatMonth(summary.from)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.home_through, formatDay(summary.to))).assertIsDisplayed()
    }

    @Test
    fun headerFallsBackToTitleWhileLoading() {
        show(HomeUiState.Loading)

        composeRule.onNodeWithText(res.getString(R.string.home_title)).assertIsDisplayed()
    }

    @Test
    fun headerFallsBackToTitleOnFailure() {
        show(HomeUiState.Failure(UiError.Network))

        composeRule.onNodeWithText(res.getString(R.string.home_title)).assertIsDisplayed()
    }

    @Test
    fun totalsShowNetAndDeltas() {
        show(ready(statsSummary(incomeDelta = 0.25, expensesDelta = -0.15)))

        composeRule.onNodeWithText(res.getString(R.string.home_net)).assertIsDisplayed()
        composeRule.onNodeWithText(formatPercent(0.25, signed = true)).assertIsDisplayed()
    }

    @Test
    fun deltasAreHiddenWithoutPreviousData() {
        show(ready(statsSummary(hasPreviousData = false, incomeDelta = 0.0, expensesDelta = 0.0)))

        composeRule.onNodeWithText(res.getString(R.string.home_net)).assertIsDisplayed()
        composeRule.onAllNodesWithText(formatPercent(0.0, signed = true)).assertCountEquals(0)
    }

    @Test
    fun emptyStateOffersToAddTransaction() {
        var added = false
        show(ready(statsSummary(transactionsTotal = 0)), onAddTransaction = { added = true })

        composeRule.onNodeWithText(res.getString(R.string.home_add_transaction)).performClick()

        assertTrue(added)
    }
}
