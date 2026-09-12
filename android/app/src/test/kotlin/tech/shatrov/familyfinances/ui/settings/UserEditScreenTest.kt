package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onLast
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.theme.AppTheme

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class UserEditScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private var roleToggled = false

    private fun show(
        state: UserEditUiState,
        passwordSet: Boolean = false,
    ) {
        composeRule.setContent {
            AppTheme {
                UserEditScreen(
                    state = state,
                    passwordSet = passwordSet,
                    onEmailChange = {},
                    onFirstNameChange = {},
                    onLastNameChange = {},
                    onPasswordChange = {},
                    onRoleChange = {},
                    onSubmit = {},
                    onToggleRole = { roleToggled = true },
                    onToggleActive = {},
                    onSetPassword = {},
                    onRetry = {},
                    onBack = {},
                )
            }
        }
    }

    private fun form(
        user: User,
        self: Boolean,
    ): UserEditUiState = UserEditUiState(
        id = user.id,
        self = self,
        loaded = user,
        email = user.email,
        firstName = user.firstName,
        lastName = user.lastName,
        role = user.role,
    )

    @Test
    fun ownRecordHasNoDeactivationAndNoPassword() {
        show(form(testSession().user, self = true))

        composeRule.onNodeWithText(res.getString(R.string.settings_user_deactivate)).assertDoesNotExist()
        composeRule.onNodeWithText(res.getString(R.string.settings_user_set_password)).assertDoesNotExist()
        // Роль своей записи менять можно: понижение себя сервер разрешает, пока админ не последний.
        composeRule.onNodeWithText(res.getString(R.string.settings_user_make_member)).assertExists()
    }

    @Test
    fun fieldErrorIsShownUnderTheField() {
        show(form(testSession().user, self = true).copy(fieldErrors = mapOf(UserField.EMAIL to "неверная почта")))

        composeRule.onNodeWithText("неверная почта").performScrollTo().assertExists()
    }

    @Test
    fun roleChangeWaitsForConfirmation() {
        show(form(testSession(Role.member).user, self = false))

        val label = res.getString(R.string.settings_user_make_admin)
        composeRule.onNodeWithText(label).performScrollTo().performClick()
        composeRule.onNodeWithText(res.getString(R.string.settings_user_role_confirm)).assertExists()
        assertFalse(roleToggled)

        composeRule.onAllNodesWithText(label).onLast().performClick()
        assertTrue(roleToggled)
    }
}
