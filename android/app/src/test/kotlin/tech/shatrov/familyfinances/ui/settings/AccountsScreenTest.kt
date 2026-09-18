package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
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
import tech.shatrov.familyfinances.theme.AppTheme
import java.util.UUID

/** Счета: пустое состояние ведёт в форму, удаление счёта — только через подтверждение с его именем. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class AccountsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    @Test
    fun emptyStateOffersAdd() {
        var added = 0
        composeRule.setContent {
            AppTheme {
                AccountsScreen(
                    state = AccountsUiState.Ready(active = emptyList(), archived = emptyList()),
                    onRetry = {},
                    onAdd = { added++ },
                    onOpen = {},
                    onBack = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.settings_accounts_empty)).assertExists()
        composeRule.onNode(hasText(res.getString(R.string.settings_account_add)) and hasClickAction()).performClick()
        assertEquals(1, added)
    }

    @Test
    fun deleteWaitsForConfirmation() {
        var deleted = false
        composeRule.setContent {
            AppTheme {
                AccountEditScreen(
                    state = AccountEditUiState(
                        id = UUID.fromString(CARD_ACCOUNT_ID),
                        name = "Тинькофф",
                        savedName = "Тинькофф",
                        canDelete = true,
                    ),
                    onNameChange = {},
                    onSubmit = {},
                    onToggleArchive = {},
                    onDelete = { deleted = true },
                    onBack = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.settings_account_delete)).performClick()
        assertFalse(deleted)

        composeRule.onNodeWithText(res.getString(R.string.settings_account_delete_confirm)).assertExists()
        composeRule.onNode(hasText("Тинькофф") and hasAnyAncestor(isDialog())).assertExists()
        val confirm = hasText(res.getString(R.string.settings_account_delete)) and hasAnyAncestor(isDialog())
        composeRule.onNode(confirm).performClick()
        assertTrue(deleted)
    }
}
