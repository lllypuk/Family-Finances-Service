package tech.shatrov.familyfinances.ui.networth

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
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
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.theme.AppTheme
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HoldingEditScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    @Test
    fun bothPlanNumbersAreEnteredAndSaved() {
        var submitted: HoldingEditUiState? = null
        composeRule.setContent {
            var state by remember {
                mutableStateOf(HoldingEditUiState(id = null, draft = UUID.randomUUID(), side = HoldingSide.asset))
            }
            AppTheme {
                HoldingEditScreen(
                    state = state,
                    currency = "RUB",
                    onSideChange = {},
                    onNameChange = { state = state.copy(name = it) },
                    onKindChange = {},
                    onIncomeChange = { state = state.copy(income = it) },
                    onExpenseChange = { state = state.copy(expense = it) },
                    onSubmit = { submitted = state },
                    onToggleArchive = {},
                    onDelete = {},
                    onRetry = {},
                    onBack = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.holding_name)).performTextInput("Квартира")
        composeRule.onNodeWithText(res.getString(R.string.holding_plan_income)).performScrollTo()
            .performTextInput("45000")
        composeRule.onNodeWithText(res.getString(R.string.holding_plan_expense)).performScrollTo()
            .performTextInput("8300,50")
        composeRule.onNodeWithText(res.getString(R.string.holding_save)).performScrollTo().assertIsEnabled()
            .performClick()

        assertEquals(4_500_000L, submitted?.incomeMinor)
        assertEquals(830_050L, submitted?.expenseMinor)
    }

    @Test
    fun unreadablePlanShowsErrorAndBlocksSaving() {
        composeRule.setContent {
            AppTheme {
                HoldingEditScreen(
                    state = HoldingEditUiState(
                        id = null,
                        draft = UUID.randomUUID(),
                        side = HoldingSide.liability,
                        name = "Ипотека",
                        expense = "8з",
                    ),
                    currency = "RUB",
                    onSideChange = {},
                    onNameChange = {},
                    onKindChange = {},
                    onIncomeChange = {},
                    onExpenseChange = {},
                    onSubmit = {},
                    onToggleArchive = {},
                    onDelete = {},
                    onRetry = {},
                    onBack = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.holding_plan_error_amount)).performScrollTo()
        composeRule.onNodeWithText(res.getString(R.string.holding_save)).performScrollTo().assertIsNotEnabled()
    }
}
