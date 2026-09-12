package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.theme.AppTheme

private val ROW = BackupRow(
    name = "backup_20260911_100000123.db",
    size = "12,3 МБ",
    created = "11 сентября 2026, 13:00",
)

/** Бэкапы: удаление необратимо, поэтому идёт через подтверждение с именем файла. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BackupsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: BackupsUiState = BackupsUiState.Ready(rows = listOf(ROW), total = 1),
        onDelete: (String) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                BackupsScreen(state = state, onRetry = {}, onCreate = {}, onDelete = onDelete, onBack = {})
            }
        }
    }

    @Test
    fun deleteWaitsForConfirmation() {
        var deleted: String? = null
        show(onDelete = { deleted = it })

        composeRule.onNodeWithText(res.getString(R.string.settings_backup_delete)).performClick()
        assertNull(deleted)

        composeRule.onNodeWithText(res.getString(R.string.settings_backup_delete_confirm)).assertExists()
        // Имя файла в диалоге: строки списка отличаются только им.
        composeRule.onNode(hasText(ROW.name) and hasAnyAncestor(isDialog())).assertExists()
        val confirm = hasText(res.getString(R.string.settings_backup_delete)) and hasAnyAncestor(isDialog())
        composeRule.onNode(confirm).performClick()
        assertEquals(ROW.name, deleted)
    }

    @Test
    fun failureOffersRetry() {
        show(BackupsUiState.Failure(tech.shatrov.familyfinances.ui.UiError.Network))

        composeRule.onNodeWithText(res.getString(R.string.error_network)).assertExists()
        composeRule.onNodeWithText(res.getString(R.string.retry)).assertExists()
    }
}
