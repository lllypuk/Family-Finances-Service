package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.assertIsNotSelected
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.text.TextLayoutResult
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
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
import tech.shatrov.familyfinances.theme.ThemeMode
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
        themeMode: ThemeMode = ThemeMode.System,
        onThemeMode: (ThemeMode) -> Unit = {},
        onOpen: (SettingsPage) -> Unit = {},
        onSignOut: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                SettingsRootScreen(
                    session = session,
                    refreshing = false,
                    error = error,
                    themeMode = themeMode,
                    onThemeMode = onThemeMode,
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

    @Test
    fun themeSegmentFollowsMode() {
        var picked: ThemeMode? = null
        show(themeMode = ThemeMode.Dark, onThemeMode = { picked = it })

        composeRule.onNodeWithText(res.getString(R.string.settings_theme_dark)).assertIsSelected()
        composeRule.onNodeWithText(res.getString(R.string.settings_theme_system)).assertIsNotSelected()

        composeRule.onNodeWithText(res.getString(R.string.settings_theme_light)).performClick()
        assertEquals(ThemeMode.Light, picked)
    }

    // Выбранный сегмент теряет место на галочку — подпись должна влезть в одну строку при любом выборе.
    @Test
    @Config(qualifiers = "+w360dp-h640dp")
    fun themeLabelsFitOnNarrowPhone() {
        var mode by mutableStateOf(ThemeMode.System)
        composeRule.setContent {
            AppTheme {
                SettingsRootScreen(
                    session = testSession(),
                    refreshing = false,
                    error = null,
                    themeMode = mode,
                    onThemeMode = {},
                    onOpen = {},
                    onRetry = {},
                    onBack = {},
                    onSignOut = {},
                )
            }
        }

        val labels = listOf(
            R.string.settings_theme_system,
            R.string.settings_theme_light,
            R.string.settings_theme_dark,
        ).map(res::getString)
        for (selected in ThemeMode.entries) {
            mode = selected
            composeRule.waitForIdle()
            for (label in labels) {
                val node = composeRule.onNodeWithText(label, useUnmergedTree = true).fetchSemanticsNode()
                val layouts = mutableListOf<TextLayoutResult>()
                node.config[SemanticsActions.GetTextLayoutResult].action?.invoke(layouts)
                val layout = layouts.single()
                assertEquals("$label/$selected", 1, layout.lineCount)
                assertFalse("$label/$selected", layout.isLineEllipsized(0))
            }
        }
    }
}
