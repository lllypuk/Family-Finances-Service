package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performScrollTo
import androidx.test.core.app.ApplicationProvider
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.theme.AppTheme

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class PasswordScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(state: PasswordUiState) {
        composeRule.setContent {
            AppTheme {
                PasswordScreen(
                    state = state,
                    onCurrentChange = {},
                    onNewChange = {},
                    onRepeatChange = {},
                    onSubmit = {},
                    onBack = {},
                )
            }
        }
    }

    @Test
    fun mismatchIsNamedAndBlocksSaving() {
        show(PasswordUiState(current = "Admin1234!", next = "ёёёёё", repeat = "ёёёёю"))

        composeRule.onNodeWithText(res.getString(R.string.settings_password_mismatch)).performScrollTo().assertExists()
        composeRule.onNodeWithText(res.getString(R.string.settings_save)).performScrollTo().assertIsNotEnabled()
    }

    @Test
    fun wrongCurrentPasswordIsShownUnderTheField() {
        show(PasswordUiState(currentInvalid = true))

        composeRule.onNodeWithText(res.getString(R.string.settings_password_wrong_current))
            .performScrollTo()
            .assertExists()
    }

    @Test
    fun successIsConfirmedOnTheScreen() {
        show(PasswordUiState(changed = true))

        composeRule.onNodeWithText(res.getString(R.string.settings_password_changed)).performScrollTo().assertExists()
    }
}
