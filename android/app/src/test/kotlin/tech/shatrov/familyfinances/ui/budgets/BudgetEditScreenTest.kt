package tech.shatrov.familyfinances.ui.budgets

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.filterToOne
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
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
import tech.shatrov.familyfinances.core.api.Budget
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.format.formatDay
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

private val START: LocalDate = LocalDate.parse("2026-09-01")
private val END: LocalDate = LocalDate.parse("2026-09-30")
private const val WEEK_TAIL = 6L

@Composable
private fun Screen(
    state: BudgetEditUiState,
    onPeriodChange: (BudgetPeriod) -> Unit,
    onDelete: () -> Unit,
) {
    BudgetEditScreen(
        state = state,
        onNameChange = {},
        onAmountChange = {},
        onPeriodChange = onPeriodChange,
        onCategoryChange = {},
        onStartChange = {},
        onEndChange = {},
        onSubmit = {},
        onDelete = onDelete,
        onRetry = {},
        onBack = {},
    )
}

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BudgetEditScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    @Test
    fun weeklyPresetMovesTheEndDate() {
        showLive(form())

        composeRule.onNodeWithText(res.getString(R.string.budget_period_weekly)).performClick()

        composeRule.onNodeWithText(endLabel(START.plusDays(WEEK_TAIL))).performScrollTo().assertIsDisplayed()
    }

    @Test
    fun unchangedEditFormCannotBeSaved() {
        show(edit())

        composeRule.onNodeWithText(res.getString(R.string.budget_save)).performScrollTo().assertIsNotEnabled()
    }

    @Test
    fun editedNameEnablesSaving() {
        show(edit().copy(name = "Еда и дом"))

        composeRule.onNodeWithText(res.getString(R.string.budget_save)).performScrollTo().assertIsEnabled()
    }

    @Test
    fun validationErrorIsShownUnderTheField() {
        show(form(fieldErrors = mapOf(BudgetField.NAME to "название занято")))

        composeRule.onNodeWithText("название занято").assertIsDisplayed()
    }

    @Test
    fun deletingAsksForConfirmation() {
        var deleted = false
        show(edit(), onDelete = { deleted = true })

        composeRule.onNodeWithText(res.getString(R.string.budget_delete)).performScrollTo().performClick()
        assertFalse(deleted)

        composeRule.onAllNodesWithText(res.getString(R.string.budget_delete))
            .filterToOne(hasAnyAncestor(isDialog()))
            .performClick()

        assertTrue(deleted)
    }

    @Test
    fun creationHasNoDeleteButton() {
        show(form())

        composeRule.onNodeWithText(res.getString(R.string.budget_delete)).assertDoesNotExist()
    }

    @Test
    fun periodIsReadOnlyWhenEditing() {
        show(edit())

        composeRule.onNodeWithText(res.getString(R.string.budget_period_weekly)).assertDoesNotExist()
        composeRule.onNodeWithText(res.getString(R.string.budget_period_monthly)).assertIsDisplayed()
    }

    private fun endLabel(date: LocalDate) = "${res.getString(R.string.budget_end)}: ${formatDay(date)}"

    private fun show(
        state: BudgetEditUiState,
        onDelete: () -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme { Screen(state, onPeriodChange = {}, onDelete = onDelete) }
        }
    }

    /** Пресет конца живёт в модели, поэтому экран для него запускается поверх живого состояния. */
    private fun showLive(initial: BudgetEditUiState) {
        composeRule.setContent {
            var state by remember { mutableStateOf(initial) }
            AppTheme {
                Screen(
                    state = state,
                    onPeriodChange = { period ->
                        state = state.copy(period = period, end = endOf(period, state.start) ?: state.end)
                    },
                    onDelete = {},
                )
            }
        }
    }

    private fun form(fieldErrors: Map<String, String> = emptyMap()) = BudgetEditUiState(
        name = "Еда",
        amount = "50000",
        period = BudgetPeriod.monthly,
        start = START,
        end = END,
        loading = false,
        ready = true,
        fieldErrors = fieldErrors,
    )

    private fun edit(): BudgetEditUiState {
        val stamp = OffsetDateTime.parse("2026-09-07T10:00:00Z")
        val budget = Budget(
            id = UUID.fromString("99999999-9999-9999-9999-999999999991"),
            name = "Еда",
            amountMinor = 5_000_000,
            spentMinor = 3_000_000,
            remainingMinor = 2_000_000,
            utilization = 60.0,
            period = BudgetPeriod.monthly,
            startDate = START,
            endDate = END,
            isActive = true,
            createdAt = stamp,
            updatedAt = stamp,
            categoryId = null,
        )
        return form().copy(loaded = budget, editing = true)
    }
}
