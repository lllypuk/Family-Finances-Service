package tech.shatrov.familyfinances.ui.login

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
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
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.theme.AppTheme

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class LoginScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: LoginUiState,
        onEmailChange: (String) -> Unit = {},
        onPasswordChange: (String) -> Unit = {},
        onSubmit: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                LoginScreen(
                    state = state,
                    onEmailChange = onEmailChange,
                    onPasswordChange = onPasswordChange,
                    onSubmit = onSubmit,
                )
            }
        }
    }

    @Test
    fun showsBothFields() {
        show(LoginUiState())

        composeRule.onNodeWithText(res.getString(R.string.login_email)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.login_password)).assertIsDisplayed()
    }

    @Test
    fun typingReachesCallbacks() {
        val typed = mutableListOf<String>()
        show(LoginUiState(), onEmailChange = { typed += it })

        composeRule.onNodeWithText(res.getString(R.string.login_email)).performTextInput("a@b.c")

        assertEquals(listOf("a@b.c"), typed)
    }

    @Test
    fun submitIsDisabledUntilBothFieldsFilled() {
        show(LoginUiState(email = "admin@test.com"))

        composeRule.onNodeWithText(res.getString(R.string.login_submit)).assertIsNotEnabled()
    }

    @Test
    fun submitReachesCallback() {
        var submits = 0
        show(LoginUiState(email = "admin@test.com", password = "Admin1234!"), onSubmit = { submits++ })

        composeRule.onNodeWithText(res.getString(R.string.login_submit)).assertIsEnabled().performClick()

        assertEquals(1, submits)
    }

    @Test
    fun errorIsShown() {
        show(LoginUiState(email = "admin@test.com", password = "x", error = LoginError.SetupRequired))

        composeRule.onNodeWithText(res.getString(R.string.login_error_setup_required)).assertIsDisplayed()
    }
}
