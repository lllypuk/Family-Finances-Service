package tech.shatrov.familyfinances

import androidx.compose.ui.test.junit4.v2.createAndroidComposeRule
import androidx.core.view.WindowCompat
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.theme.ThemeMode

/** На старте XML и `SideEffect` совпадают; расходятся они только при переключении в открытом окне. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK], qualifiers = "+notnight")
class ThemeSwitchTest {
    @get:Rule
    val composeRule = createAndroidComposeRule<MainActivity>()

    @Test
    fun switchInOpenAppRecolorsSystemBars() {
        val theme = (composeRule.activity.application as FamilyFinancesApp).graph.theme
        val window = composeRule.activity.window
        val bars = WindowCompat.getInsetsController(window, window.decorView)

        for ((mode, light) in listOf(ThemeMode.Dark to false, ThemeMode.Light to true, ThemeMode.System to true)) {
            composeRule.runOnIdle { theme.write(mode) }
            composeRule.waitForIdle()
            assertEquals("$mode", light, bars.isAppearanceLightStatusBars)
            assertEquals("$mode", light, bars.isAppearanceLightNavigationBars)
        }
    }
}
