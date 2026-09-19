package tech.shatrov.familyfinances.ui.home

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onLast
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
import java.time.ZoneId
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HomeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: HomeUiState,
        onAddTransaction: () -> Unit = {},
        onOverview: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = state,
                    onRetry = {},
                    onAddTransaction = onAddTransaction,
                    onSettings = {},
                    onOverview = onOverview,
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

    @Test
    fun overviewRowOpensOverview() {
        var opened = false
        show(ready(statsSummary()), onOverview = { opened = true })

        composeRule.onNodeWithText(res.getString(R.string.overview_title)).performClick()

        assertTrue(opened)
    }

    // В пустом состоянии обзору нечего показать: вместо строки — кнопка добавить операцию.
    @Test
    fun emptyStateHasNoOverviewRow() {
        show(ready(statsSummary(transactionsTotal = 0)))

        composeRule.onAllNodesWithText(res.getString(R.string.overview_title)).assertCountEquals(0)
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

        card = ReconciliationCard.Incomplete(YearMonth.of(2026, 8))
        composeRule.onNodeWithText(res.getString(R.string.home_reconciliation_incomplete)).performClick()
        assertEquals(YearMonth.of(2026, 8), opened)
        composeRule.onNodeWithText(res.getString(R.string.home_reconciliation, "август")).assertIsDisplayed()

        card = ReconciliationCard.Ready(YearMonth.of(2026, 7), gapMinor = 0)
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_matched)).performClick()
        assertEquals(YearMonth.of(2026, 7), opened)

        card = ReconciliationCard.Ready(YearMonth.of(2026, 7), gapMinor = 10000)
        val gap = res.getString(R.string.reconciliation_gap, formatMoney(10000, "RUB", signed = true))
        composeRule.onNodeWithText(gap).assertIsDisplayed()
    }

    private val today = LocalDate.of(2026, 9, 19)

    private val draft = ImportDraft(
        importId = UUID.randomUUID(),
        recognized = true,
        rows = 5,
        saved = 2,
        images = 3,
        updatedAt = today.atTime(14, 20).atZone(ZoneId.of("Europe/Moscow")),
    )

    private fun showDraft(
        state: HomeUiState,
        onResume: (UUID) -> Unit = {},
        onDelete: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = state,
                    onRetry = {},
                    onAddTransaction = {},
                    onSettings = {},
                    importDraft = draft,
                    onResumeImport = onResume,
                    onDeleteImport = onDelete,
                    today = today,
                )
            }
        }
    }

    private fun draftDetails(): String {
        val at = res.getString(R.string.home_import_today, "14:20")
        return res.getQuantityString(R.plurals.home_import_rows, 5, 5, 2, at)
    }

    // Журнал локальный: плашка не ждёт сервера и не пропадает при его отказе.
    @Test
    fun importCardShowsOnLoadingFailureAndEmpty() {
        var state by mutableStateOf<HomeUiState>(HomeUiState.Loading)
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = state,
                    onRetry = {},
                    onAddTransaction = {},
                    onSettings = {},
                    importDraft = draft,
                    today = today,
                )
            }
        }
        composeRule.onNodeWithText(draftDetails()).assertIsDisplayed()

        state = HomeUiState.Failure(UiError.Network)
        composeRule.onNodeWithText(draftDetails()).assertIsDisplayed()

        state = ready(statsSummary(transactionsTotal = 0))
        composeRule.onNodeWithText(draftDetails()).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.home_add_transaction)).assertIsDisplayed()
    }

    @Test
    fun interruptedImportTalksAboutScreenshots() {
        composeRule.setContent {
            AppTheme {
                HomeScreen(
                    state = HomeUiState.Loading,
                    onRetry = {},
                    onAddTransaction = {},
                    onSettings = {},
                    importDraft = draft.copy(recognized = false),
                    today = today,
                )
            }
        }

        composeRule.onNodeWithText(res.getQuantityString(R.plurals.home_import_interrupted, 3, 3)).assertIsDisplayed()
    }

    @Test
    fun resumeOpensTheImport() {
        var resumed: UUID? = null
        showDraft(HomeUiState.Loading, onResume = { resumed = it })

        composeRule.onNodeWithText(res.getString(R.string.home_import_resume)).performClick()

        assertEquals(draft.importId, resumed)
    }

    @Test
    fun deleteAsksForConfirmation() {
        var deleted = 0
        showDraft(HomeUiState.Loading, onDelete = { deleted++ })

        composeRule.onNodeWithText(res.getString(R.string.home_import_delete)).performClick()
        composeRule.onNodeWithText(res.getString(R.string.home_import_delete_confirm)).assertIsDisplayed()
        assertEquals(0, deleted)

        composeRule.onNodeWithText(res.getString(R.string.cancel)).performClick()
        assertEquals(0, deleted)

        composeRule.onNodeWithText(res.getString(R.string.home_import_delete)).performClick()
        composeRule.onAllNodesWithText(res.getString(R.string.home_import_delete)).onLast().performClick()
        assertEquals(1, deleted)
    }
}
