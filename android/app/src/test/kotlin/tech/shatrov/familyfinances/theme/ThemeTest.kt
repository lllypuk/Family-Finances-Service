package tech.shatrov.familyfinances.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.junit4.v2.createComposeRule
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ThemeTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    @Config(qualifiers = "+night")
    fun system_mode_follows_night_qualifier_dark() {
        assertPalette(ThemeMode.System, DarkColors)
    }

    @Test
    @Config(qualifiers = "+notnight")
    fun system_mode_follows_night_qualifier_light() {
        assertPalette(ThemeMode.System, LightColors)
    }

    @Test
    @Config(qualifiers = "+night")
    fun light_mode_overrides_night_system() {
        assertPalette(ThemeMode.Light, LightColors)
    }

    @Test
    @Config(qualifiers = "+notnight")
    fun dark_mode_overrides_light_system() {
        assertPalette(ThemeMode.Dark, DarkColors)
    }

    private fun assertPalette(
        mode: ThemeMode,
        expected: AppColors,
    ) {
        lateinit var colors: AppColors
        lateinit var scheme: ColorScheme
        composeRule.setContent {
            AppTheme(mode) {
                colors = LocalAppColors.current
                scheme = MaterialTheme.colorScheme
            }
        }
        composeRule.waitForIdle()
        assertEquals(expected, colors)
        assertEquals(expected.action, scheme.primary)
        assertEquals(expected.action, scheme.surfaceTint)
        assertEquals(expected.canvas, scheme.background)
    }
}
