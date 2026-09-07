package tech.shatrov.familyfinances.ui.transactions

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.AppTheme
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class TransactionEditScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: TransactionEditUiState,
        onTypeChange: (TransactionType) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                TransactionEditScreen(
                    state = state,
                    onAmountChange = {},
                    onTypeChange = onTypeChange,
                    onCategoryChange = {},
                    onDateChange = {},
                    onDescriptionChange = {},
                    onSubmit = {},
                    onDelete = {},
                    onBack = {},
                )
            }
        }
    }

    @Test
    fun emptyAmountBlocksSaving() {
        show(form(amount = ""))

        composeRule.onNodeWithText(res.getString(R.string.transaction_save)).assertIsNotEnabled()
    }

    @Test
    fun filledFormCanBeSaved() {
        show(form())

        composeRule.onNodeWithText(res.getString(R.string.transaction_save)).assertIsEnabled()
    }

    @Test
    fun validationErrorIsShownUnderTheField() {
        show(form(fieldErrors = mapOf(TransactionField.AMOUNT to "должно быть больше нуля")))

        composeRule.onNodeWithText("должно быть больше нуля").assertIsDisplayed()
    }

    @Test
    fun tappingTypeReachesCallback() {
        var type: TransactionType? = null
        show(form(), onTypeChange = { type = it })

        composeRule.onNodeWithText(res.getString(R.string.type_income)).performClick()

        assertEquals(TransactionType.income, type)
    }

    private fun form(
        amount: String = "1500",
        fieldErrors: Map<String, String> = emptyMap(),
    ) = TransactionEditUiState(
        amount = amount,
        categoryId = UUID.fromString(GROCERIES_ID),
        date = LocalDate.parse("2026-09-07"),
        description = "Кофе",
        categories = listOf(groceries),
        loading = false,
        fieldErrors = fieldErrors,
    )

    private val groceries = Category(
        id = UUID.fromString(GROCERIES_ID),
        name = "Продукты",
        type = CategoryType.expense,
        color = "#ff0000",
        icon = "cart",
        isActive = true,
        createdAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-07T10:00:00Z"),
    )
}
