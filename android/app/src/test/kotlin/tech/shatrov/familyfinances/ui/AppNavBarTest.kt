package tech.shatrov.familyfinances.ui

import android.content.Context
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
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
import org.robolectric.annotation.GraphicsMode
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.theme.AppTheme

/** Пять вкладок — предел панели: на самом узком телефоне подпись не переносится и не обрезается. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK], qualifiers = "w360dp-h640dp")
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class AppNavBarTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    @Test
    fun labelsFitOnNarrowPhone() {
        composeRule.setContent {
            AppTheme { AppNavBar(selected = AppTab.NET_WORTH, onSelect = {}) }
        }

        // Метка `didOverflowWidth` тут бесполезна: подпись меряется по содержимому и взводит её всегда.
        val bounds = AppTab.entries.map { tab ->
            val label = res.getString(tab.label)
            val node = composeRule.onNodeWithText(label, useUnmergedTree = true).fetchSemanticsNode()
            val layouts = mutableListOf<TextLayoutResult>()
            node.config[SemanticsActions.GetTextLayoutResult].action?.invoke(layouts)
            val layout = layouts.single()
            assertEquals(label, 1, layout.lineCount)
            assertFalse(label, layout.isLineEllipsized(0))
            node.boundsInRoot
        }
        val width = res.configuration.screenWidthDp * res.displayMetrics.density
        assertTrue(bounds.first().left >= 0f && bounds.last().right <= width)
        assertTrue(bounds.zipWithNext().all { (left, right) -> left.right <= right.left })
    }
}
