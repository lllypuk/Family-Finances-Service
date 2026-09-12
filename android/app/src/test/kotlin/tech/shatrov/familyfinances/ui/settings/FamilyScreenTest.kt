package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.assertIsEnabled
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
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class FamilyScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(state: FamilyUiState) {
        composeRule.setContent {
            AppTheme {
                FamilyScreen(
                    state = state,
                    onNameChange = {},
                    onCurrencyChange = {},
                    onTimezoneChange = {},
                    onSubmit = {},
                    onBack = {},
                )
            }
        }
    }

    private fun form(): FamilyUiState = FamilyUiState(loaded = testSession().family)

    @Test
    fun savingIsBlockedUntilSomethingChanges() {
        show(form())

        composeRule.onNodeWithText(res.getString(R.string.settings_save)).performScrollTo().assertIsNotEnabled()
    }

    @Test
    fun changedFieldEnablesSaving() {
        show(form().copy(timezone = "Asia/Novosibirsk"))

        composeRule.onNodeWithText(res.getString(R.string.settings_save)).performScrollTo().assertIsEnabled()
    }

    @Test
    fun fieldErrorAndConflictAreBothShown() {
        show(
            form().copy(
                currency = "EUR",
                timezone = "Europe/Атлантида",
                fieldErrors = mapOf(FamilyField.TIMEZONE to "неизвестный часовой пояс"),
                error = UiError.Resource(R.string.settings_error_currency_locked),
            ),
        )

        composeRule.onNodeWithText("неизвестный часовой пояс").performScrollTo().assertExists()
        composeRule.onNodeWithText(res.getString(R.string.settings_error_currency_locked)).assertExists()
    }
}
