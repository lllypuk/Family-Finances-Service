package tech.shatrov.familyfinances.ui.categories

import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.theme.Dimens

/** Robolectric нужен ради `android.icu`: на чистой JVM его нет. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class CategoryAvatarTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun emojiIconIsTheGlyph() {
        assertEquals("🛒", categoryGlyph("🛒", "Продукты"))
        assertEquals("🛒", categoryGlyph(" 🛒🚗", "Продукты"))
    }

    @Test
    fun multiCharClustersStayWhole() {
        assertEquals("👨‍👩‍👧", categoryGlyph("👨‍👩‍👧", "Семья"))
        assertEquals("1️⃣", categoryGlyph("1️⃣", "Первое"))
        assertEquals("👍🏽", categoryGlyph("👍🏽x", "Хорошо"))
    }

    @Test
    fun chipLabelPrefixesOnlyEmoji() {
        assertEquals("🛒 Продукты", categoryChipLabel("🛒", "Продукты"))
        assertEquals("Продукты", categoryChipLabel("food", "Продукты"))
        assertEquals("Продукты", categoryChipLabel("", "Продукты"))
    }

    @Test
    fun wordsAndBlankFallBackToNameInitial() {
        listOf("default", "food", "", "  ").forEach {
            assertEquals(it, "П", categoryGlyph(it, "Продукты"))
        }
        assertEquals("Ё", categoryGlyph("", "ёлка"))
        assertEquals("?", categoryGlyph("", ""))
    }

    @Test
    fun malformedColorDoesNotCrash() {
        composeRule.setContent {
            AppTheme { CategoryAvatar("food", "not-a-color", "Продукты", Dimens.AVATAR_M) }
        }

        composeRule.onNodeWithContentDescription("Продукты").assertIsDisplayed()
    }
}
