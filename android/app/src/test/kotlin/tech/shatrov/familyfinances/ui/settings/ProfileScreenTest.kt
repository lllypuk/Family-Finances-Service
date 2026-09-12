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
class ProfileScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(state: ProfileUiState) {
        composeRule.setContent {
            AppTheme {
                ProfileScreen(
                    state = state,
                    onEmailChange = {},
                    onFirstNameChange = {},
                    onLastNameChange = {},
                    onSubmit = {},
                    onBack = {},
                )
            }
        }
    }

    private fun form(): ProfileUiState = ProfileUiState(loaded = testSession().user)

    @Test
    fun savingIsBlockedUntilSomethingChanges() {
        show(form())

        composeRule.onNodeWithText(res.getString(R.string.settings_save)).performScrollTo().assertIsNotEnabled()
    }

    @Test
    fun changedFieldEnablesSaving() {
        show(form().copy(firstName = "Саша"))

        composeRule.onNodeWithText(res.getString(R.string.settings_save)).performScrollTo().assertIsEnabled()
    }

    @Test
    fun fieldErrorAndConflictAreBothShown() {
        show(
            form().copy(
                email = "member@test.com",
                fieldErrors = mapOf(ProfileField.EMAIL to "неверная почта"),
                error = UiError.Resource(R.string.settings_error_email_taken),
            ),
        )

        composeRule.onNodeWithText("неверная почта").performScrollTo().assertExists()
        composeRule.onNodeWithText(res.getString(R.string.settings_error_email_taken)).assertExists()
    }
}
