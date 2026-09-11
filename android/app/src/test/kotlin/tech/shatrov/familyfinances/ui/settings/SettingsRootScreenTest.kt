package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
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
import tech.shatrov.familyfinances.Session
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.testSession
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError

/** Корень настроек: что видит каждая роль и что выход спрашивает подтверждение. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class SettingsRootScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        session: Session = testSession(),
        error: UiError? = null,
        onOpen: (SettingsPage) -> Unit = {},
        onSignOut: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                SettingsRootScreen(
                    session = session,
                    refreshing = false,
                    error = error,
                    onOpen = onOpen,
                    onRetry = {},
                    onBack = {},
                    onSignOut = onSignOut,
                )
            }
        }
    }

    @Test
    fun memberDoesNotSeeAdminSections() {
        show(testSession(Role.member))

        composeRule.onNodeWithText(res.getString(R.string.settings_profile)).assertExists()
        composeRule.onNodeWithText(res.getString(R.string.settings_role_member)).assertExists()
        for (hidden in listOf(R.string.settings_users, R.string.settings_family, R.string.settings_backups)) {
            composeRule.onNodeWithText(res.getString(hidden)).assertDoesNotExist()
        }
    }

    @Test
    fun adminSeesAdminSections() {
        var opened: SettingsPage? = null
        show(onOpen = { opened = it })

        for (shown in listOf(R.string.settings_users, R.string.settings_family, R.string.settings_backups)) {
            composeRule.onNodeWithText(res.getString(shown)).assertExists()
        }
        composeRule.onNodeWithText(res.getString(R.string.settings_users)).performScrollTo().performClick()
        assertTrue(opened is SettingsPage.Users)
    }

    @Test
    fun signOutAsksForConfirmation() {
        var signedOut = false
        show(onSignOut = { signedOut = true })

        composeRule.onNodeWithText(res.getString(R.string.sign_out)).performScrollTo().performClick()
        assertFalse(signedOut)

        composeRule.onNodeWithText(res.getString(R.string.settings_sign_out_confirm)).assertExists()
        val confirm = hasText(res.getString(R.string.sign_out)) and hasAnyAncestor(isDialog())
        composeRule.onNode(confirm).performClick()
        assertTrue(signedOut)
    }

    // Отказ перечитки показывает причину, но карточки остаются от прежней сессии.
    @Test
    fun failedRefreshKeepsCards() {
        show(error = UiError.Network)

        composeRule.onNodeWithText(res.getString(R.string.error_network)).assertExists()
        composeRule.onNodeWithText(testSession().family.name).assertExists()
    }
}
