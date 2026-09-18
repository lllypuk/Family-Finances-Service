package tech.shatrov.familyfinances.ui.reconciliation

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ReconciliationStats
import tech.shatrov.familyfinances.theme.AppTheme
import java.time.YearMonth
import java.util.UUID

private val AUGUST = YearMonth.of(2026, 8)

/** Сверка: пустое состояние ведёт в «Счета», лист цифры банка, удаление сверки через подтверждение. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ReconciliationScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun bank(exists: Boolean) = BankEditUiState(
        accountId = UUID.fromString(CARD_ACCOUNT_ID),
        accountName = "Тинькофф",
        month = AUGUST,
        amount = if (exists) "43100" else "",
        exists = exists,
    )

    @Test
    fun emptyStateLeadsToAccounts() {
        var added = 0
        composeRule.setContent {
            AppTheme {
                ReconciliationScreen(
                    month = AUGUST,
                    state = ReconciliationUiState.Ready(AUGUST, ReconciliationStats("2026-08", 0, emptyList())),
                    editor = null,
                    currency = "RUB",
                    onBack = {},
                    onMonthChange = {},
                    onRetry = {},
                    onOpenTransactions = {},
                    onOpenBank = {},
                    onAddAccount = { added++ },
                    onAmountChange = {},
                    onNoteChange = {},
                    onSave = {},
                    onDelete = {},
                    onDismissBank = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_empty)).assertExists()
        composeRule
            .onNode(hasText(res.getString(R.string.reconciliation_add_account)) and hasClickAction())
            .performClick()
        assertEquals(1, added)
    }

    @Test
    fun bankAmountIsTypedAndSaved() {
        var state by mutableStateOf(bank(exists = false))
        var saved = 0
        composeRule.setContent {
            AppTheme {
                BankSheetContent(
                    state = state,
                    currency = "RUB",
                    onAmountChange = { state = state.copy(amount = it) },
                    onNoteChange = {},
                    onSave = { saved++ },
                    onDelete = {},
                )
            }
        }

        val save = composeRule.onNodeWithText(res.getString(R.string.reconciliation_save))
        save.assertIsNotEnabled()
        // Удалять нечего, пока сверки нет.
        composeRule.onNodeWithText(res.getString(R.string.reconciliation_delete)).assertDoesNotExist()

        composeRule
            .onNode(hasSetTextAction() and hasText(res.getString(R.string.reconciliation_bank_amount)))
            .performTextInput("1900,5")
        save.assertIsEnabled().performClick()

        assertEquals("1900,5", state.amount)
        assertEquals(1, saved)
    }

    @Test
    fun deleteWaitsForConfirmation() {
        var deleted = false
        composeRule.setContent {
            AppTheme {
                BankSheetContent(
                    state = bank(exists = true),
                    currency = "RUB",
                    onAmountChange = {},
                    onNoteChange = {},
                    onSave = {},
                    onDelete = { deleted = true },
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_delete)).performClick()
        assertFalse(deleted)

        composeRule.onNodeWithText(res.getString(R.string.reconciliation_delete_confirm)).assertExists()
        composeRule
            .onNode(hasText(res.getString(R.string.reconciliation_delete)) and hasAnyAncestor(isDialog()))
            .performClick()
        assertTrue(deleted)
    }
}
