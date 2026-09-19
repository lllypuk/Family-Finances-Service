package tech.shatrov.familyfinances.ui.reconciliation

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.ReconciliationRow
import tech.shatrov.familyfinances.core.api.ReconciliationStats
import tech.shatrov.familyfinances.theme.AppTheme
import java.time.OffsetDateTime
import java.time.YearMonth
import java.util.UUID

private val AUGUST = YearMonth.of(2026, 8)

/** Сверка: пустое состояние ведёт в «Счета», диалог остатка со знаком, «Очистить» без подтверждения. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ReconciliationScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun balance(exists: Boolean) = BalanceEditUiState(
        accountId = UUID.fromString(CARD_ACCOUNT_ID),
        accountName = "Тинькофф",
        month = AUGUST,
        amount = if (exists) "43100" else "",
        exists = exists,
    )

    private fun setScreen(
        state: ReconciliationUiState,
        editor: BalanceEditUiState?,
        onAmountChange: (String) -> Unit = {},
        onToggleSign: () -> Unit = {},
        onSave: () -> Unit = {},
        onClear: () -> Unit = {},
        onAddAccount: () -> Unit = {},
        onOpenTransactions: () -> Unit = {},
        onCloseGap: (Long) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                ReconciliationScreen(
                    month = AUGUST,
                    today = AUGUST.plusMonths(1),
                    state = state,
                    editor = editor,
                    currency = "RUB",
                    onBack = {},
                    onMonthChange = {},
                    onRetry = {},
                    onOpenBalance = {},
                    onOpenOpening = {},
                    onAddAccount = onAddAccount,
                    onOpenTransactions = onOpenTransactions,
                    onCloseGap = onCloseGap,
                    onAmountChange = onAmountChange,
                    onToggleSign = onToggleSign,
                    onSave = onSave,
                    onClear = onClear,
                    onDismissBalance = {},
                )
            }
        }
    }

    @Test
    fun emptyStateLeadsToAccounts() {
        var added = 0
        val empty = ReconciliationStats("2026-08", null, null, 0, 0, null, false, emptyList())
        setScreen(ReconciliationUiState.Ready(AUGUST, empty), editor = null, onAddAccount = { added++ })

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_empty)).assertExists()
        composeRule
            .onNode(hasText(res.getString(R.string.reconciliation_add_account)) and hasClickAction())
            .performClick()
        assertEquals(1, added)
    }

    private fun complete(gap: Long): ReconciliationUiState.Ready {
        val at = OffsetDateTime.parse("2026-09-01T10:00:00Z")
        val card = Account(UUID.fromString(CARD_ACCOUNT_ID), "Тинькофф", false, at, at)
        val row = ReconciliationRow(card, 5_000_000, 5_000_000 + gap, at)
        return ReconciliationUiState.Ready(
            AUGUST,
            ReconciliationStats("2026-08", 5_000_000, 5_000_000 + gap, 0, 0, gap, true, listOf(row)),
        )
    }

    @Test
    fun gapIsClosedWithItsSign() {
        var closed: Long? = null
        var opened = 0
        setScreen(complete(-2550), editor = null, onOpenTransactions = { opened++ }, onCloseGap = { closed = it })

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_close_gap)).performClick()
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_transactions)).performClick()

        assertEquals(-2550L, closed)
        assertEquals(1, opened)
    }

    @Test
    fun matchedMonthHasNothingToClose() {
        setScreen(complete(0), editor = null)

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_matched)).assertExists()
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_close_gap)).assertDoesNotExist()
    }

    @Test
    fun negativeBalanceIsTypedAndSaved() {
        var editor by mutableStateOf(balance(exists = false))
        var saved: Long? = null
        composeRule.setContent {
            AppTheme {
                ReconciliationScreen(
                    month = AUGUST,
                    today = AUGUST.plusMonths(1),
                    state = ReconciliationUiState.Loading,
                    editor = editor,
                    currency = "RUB",
                    onBack = {},
                    onMonthChange = {},
                    onRetry = {},
                    onOpenBalance = {},
                    onOpenOpening = {},
                    onAddAccount = {},
                    onOpenTransactions = {},
                    onCloseGap = {},
                    onAmountChange = { editor = editor.copy(amount = it) },
                    onToggleSign = { editor = editor.copy(amount = "-${editor.amount}") },
                    onSave = { saved = editor.amountMinor },
                    onClear = {},
                    onDismissBalance = {},
                )
            }
        }

        composeRule
            .onNode(hasSetTextAction() and hasText(res.getString(R.string.reconciliation_balance_amount)))
            .performTextInput("1900,5")
        composeRule.onNodeWithContentDescription(res.getString(R.string.reconciliation_toggle_sign)).performClick()
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_save)).performClick()

        assertEquals(-190050L, saved)
    }

    @Test
    fun saveIsDisabledWithoutAmountAndClearIsHidden() {
        setScreen(ReconciliationUiState.Loading, balance(exists = false))

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_save)).assertIsNotEnabled()
        // Очищать нечего, пока остатка нет.
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_clear)).assertDoesNotExist()
    }

    @Test
    fun saveAndClearReachTheModel() {
        var saved = 0
        var cleared = 0
        setScreen(ReconciliationUiState.Loading, balance(exists = true), onSave = { saved++ }, onClear = { cleared++ })

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_save)).assertIsEnabled().performClick()
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_clear)).performClick()

        assertEquals(1, saved)
        assertEquals(1, cleared)
    }
}
