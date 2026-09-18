package tech.shatrov.familyfinances.ui.home

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
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
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatPercent
import java.time.LocalDate
import java.time.YearMonth

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
    fun totalsShowSignedNetWithIncomeAndExpensesAndDeltas() {
        val summary = statsSummary(incomeDelta = 0.25, expensesDelta = -0.15)
        show(ready(summary))

        composeRule.onNodeWithText(res.getString(R.string.home_net)).assertIsDisplayed()
        composeRule
            .onNodeWithText(formatMoney(summary.current.netMinor, "RUB", signed = true))
            .assertIsDisplayed()
        composeRule.onNodeWithText(formatMoney(summary.current.incomeMinor, "RUB")).assertIsDisplayed()
        composeRule.onNodeWithText(formatMoney(summary.current.expensesMinor, "RUB")).assertIsDisplayed()
        composeRule.onNodeWithText(formatPercent(0.25, signed = true)).assertIsDisplayed()
    }

    @Test
    fun deltasAreHiddenWithoutPreviousData() {
        show(ready(statsSummary(hasPreviousData = false, incomeDelta = 0.25, expensesDelta = -0.15)))

        composeRule.onNodeWithText(res.getString(R.string.home_net)).assertIsDisplayed()
        composeRule.onAllNodesWithText(formatPercent(0.25, signed = true)).assertCountEquals(0)
        composeRule.onAllNodesWithText(formatPercent(-0.15, signed = true)).assertCountEquals(0)
    }

    @Test
    fun emptyStateOffersToAddTransaction() {
        var added = false
        show(ready(statsSummary(transactionsTotal = 0)), onAddTransaction = { added = true })

        composeRule.onNodeWithText(res.getString(R.string.home_add_transaction)).performClick()

        assertTrue(added)
    }

    // Без счетов карточка ведёт в «Счета», со счетами — на сверку своего месяца.
    @Test
    fun reconciliationCardLeadsToAccountsOrMonth() {
        var accounts = 0
        var opened: YearMonth? = null
        var card by mutableStateOf<ReconciliationCard>(ReconciliationCard.NoAccounts)
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = ready(statsSummary()),
                    onRetry = {},
                    onAddTransaction = {},
                    onSettings = {},
                    card = card,
                    onReconciliation = { opened = it },
                    onAccounts = { accounts++ },
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.home_reconciliation_no_accounts)).performClick()
        assertEquals(1, accounts)

        card = ReconciliationCard.Ready(YearMonth.of(2026, 8), matched = 1, total = 3)
        composeRule.onNodeWithText(res.getString(R.string.home_reconciliation_matched, 1, 3)).performClick()
        assertEquals(YearMonth.of(2026, 8), opened)
        composeRule.onNodeWithText(res.getString(R.string.home_reconciliation, "август")).assertIsDisplayed()
    }
}
