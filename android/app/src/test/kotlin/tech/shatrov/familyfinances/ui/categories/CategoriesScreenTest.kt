package tech.shatrov.familyfinances.ui.categories

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.theme.AppTheme
import java.time.OffsetDateTime
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class CategoriesScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: CategoriesUiState,
        onAdd: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                CategoriesScreen(state = state, onRetry = {}, onAdd = onAdd, onOpen = {})
            }
        }
    }

    @Test
    fun emptyStateOffersToAddCategory() {
        show(CategoriesUiState.Ready(emptyList(), emptyList()))

        composeRule.onNodeWithText(res.getString(R.string.categories_empty)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.categories_add_first)).assertIsDisplayed()
    }

    @Test
    fun emptyStateButtonReachesCallback() {
        var added = false
        show(CategoriesUiState.Ready(emptyList(), emptyList()), onAdd = { added = true })

        composeRule.onNodeWithText(res.getString(R.string.categories_add_first)).performClick()

        assertTrue(added)
    }

    @Test
    fun rowShowsAvatarNamedAfterCategory() {
        val at = OffsetDateTime.parse("2026-09-07T09:00:00Z")
        val groceries = Category(
            id = UUID.randomUUID(),
            name = "Продукты",
            type = CategoryType.expense,
            color = "#ff0000",
            icon = "🛒",
            isActive = true,
            createdAt = at,
            updatedAt = at,
        )
        show(CategoriesUiState.Ready(income = emptyList(), expense = listOf(CategoryNode(groceries, emptyList()))))

        composeRule.onNodeWithContentDescription("Продукты").assertIsDisplayed()
    }
}
