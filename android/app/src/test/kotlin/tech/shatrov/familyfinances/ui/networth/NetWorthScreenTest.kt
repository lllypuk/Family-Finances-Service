package tech.shatrov.familyfinances.ui.networth

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasSetTextAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.FLAT_ID
import tech.shatrov.familyfinances.MORTGAGE_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Holding
import tech.shatrov.familyfinances.core.api.HoldingCurrent
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.theme.AppTheme
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZoneOffset
import java.util.UUID

/** Капитал: пустое состояние ведёт в форму обеих сторон, лист снимка, необратимое — через подтверждение. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class NetWorthScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources
    private val today = LocalDate.of(2026, 9, 18)

    private val flat = Holding(
        id = UUID.fromString(FLAT_ID),
        name = "Квартира",
        side = HoldingSide.asset,
        kind = "property",
        isArchived = false,
        current = HoldingCurrent(LocalDate.of(2026, 1, 12), 1_100_000_000L),
        createdAt = OffsetDateTime.parse("2026-09-18T10:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-18T10:00:00Z"),
    )

    private fun editState() = HoldingEditUiState(
        id = flat.id,
        draft = UUID.randomUUID(),
        side = flat.side,
        name = flat.name,
        kind = flat.kind,
        saved = flat,
        canDelete = true,
    )

    @Test
    fun emptyStateOffersBothSides() {
        val added = mutableListOf<HoldingSide>()
        composeRule.setContent {
            AppTheme {
                NetWorthScreen(
                    state = NetWorthUiState.Ready(emptyList(), emptyList(), emptyList(), months = emptyList()),
                    currency = "RUB",
                    today = today,
                    zone = ZoneOffset.UTC,
                    onRetry = {},
                    onAdd = { added += it },
                    onOpenValue = {},
                    onEdit = {},
                )
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.net_worth_empty)).assertExists()
        composeRule.onNode(hasText(res.getString(R.string.net_worth_add_asset)) and hasClickAction()).performClick()
        composeRule.onNode(hasText(res.getString(R.string.net_worth_add_liability)) and hasClickAction()).performClick()
        assertEquals(listOf(HoldingSide.asset, HoldingSide.liability), added)
    }

    @Test
    fun planLineIsShownOnlyForHoldingWithPlan() {
        val planned = flat.copy(
            monthlyIncomeMinor = 4_500_000L,
            monthlyExpenseMinor = 830_000L,
            planUpdatedAt = OffsetDateTime.parse("2026-03-05T10:00:00Z"),
        )
        val mortgage = flat.copy(id = UUID.fromString(MORTGAGE_ID), name = "Ипотека", side = HoldingSide.liability)
        composeRule.setContent {
            AppTheme {
                NetWorthScreen(
                    state = NetWorthUiState.Ready(
                        assets = listOf(HoldingRow(planned, HoldingKind.PROPERTY)),
                        liabilities = listOf(HoldingRow(mortgage, HoldingKind.of("mortgage"))),
                        archived = emptyList(),
                        months = emptyList(),
                    ),
                    currency = "RUB",
                    today = today,
                    zone = ZoneOffset.UTC,
                    onRetry = {},
                    onAdd = {},
                    onOpenValue = {},
                    onEdit = {},
                )
            }
        }

        composeRule
            .onNodeWithText("+45\u00A0000,00\u00A0₽ · -8\u00A0300,00\u00A0₽ в мес · план от 03.2026")
            .assertExists()
        composeRule.onAllNodesWithText("в мес", substring = true).assertCountEquals(1)
    }

    @Test
    fun planLineWithoutDateSkipsSeparatorAndUsesFamilyZone() {
        val expenseOnly = flat.copy(monthlyExpenseMinor = 830_000L)
        val dated = flat.copy(
            id = UUID.fromString(MORTGAGE_ID),
            name = "Вклад",
            monthlyIncomeMinor = 4_500_000L,
            planUpdatedAt = OffsetDateTime.parse("2026-03-31T22:00:00Z"),
        )
        setScreen(
            NetWorthUiState.Ready(
                assets = listOf(HoldingRow(expenseOnly, HoldingKind.PROPERTY), HoldingRow(dated, HoldingKind.PROPERTY)),
                liabilities = emptyList(),
                archived = emptyList(),
                months = emptyList(),
            ),
            zone = ZoneId.of("Europe/Moscow"),
        )

        composeRule.onNodeWithText("-8\u00A0300,00\u00A0₽ в мес").assertExists()
        composeRule.onNodeWithText("+45\u00A0000,00\u00A0₽ в мес · план от 04.2026").assertExists()
    }

    @Test
    fun planTotalIsShownWithoutSeriesAndNamesCoverage() {
        setScreen(
            NetWorthUiState.Ready(
                assets = listOf(HoldingRow(flat, HoldingKind.PROPERTY)),
                liabilities = emptyList(),
                archived = emptyList(),
                months = emptyList(),
                plan = PlanTotal(incomeMinor = 4_500_000L, expenseMinor = 8_300_000L, planned = 1, active = 3),
            ),
        )

        composeRule.onNodeWithText(res.getString(R.string.net_worth_plan)).assertExists()
        composeRule
            .onNodeWithText("+45\u00A0000,00\u00A0₽ · -83\u00A0000,00\u00A0₽ = -38\u00A0000,00\u00A0₽")
            .assertExists()
        composeRule.onNodeWithText(res.getQuantityString(R.plurals.net_worth_plan_coverage, 3, 1, 3)).assertExists()
    }

    private fun setScreen(
        state: NetWorthUiState.Ready,
        zone: ZoneId = ZoneOffset.UTC,
    ) {
        composeRule.setContent {
            AppTheme {
                NetWorthScreen(
                    state = state,
                    currency = "RUB",
                    today = today,
                    zone = zone,
                    onRetry = {},
                    onAdd = {},
                    onOpenValue = {},
                    onEdit = {},
                )
            }
        }
    }

    @Test
    fun valueSheetTakesAmountAndShowsDate() {
        var state by mutableStateOf(ValueEditUiState(holdingId = flat.id, name = flat.name, today = today))
        var saved = 0
        composeRule.setContent {
            AppTheme {
                ValueSheetContent(
                    state = state,
                    currency = "RUB",
                    onAmountChange = { state = state.copy(amount = it) },
                    onDateChange = { state = state.copy(date = it) },
                    onSave = { saved++ },
                )
            }
        }

        val save = hasText(res.getString(R.string.holding_value_save)) and hasClickAction()
        composeRule.onNode(save).assertIsNotEnabled()
        composeRule.onNodeWithText(res.getString(R.string.holding_value_date, "18 сентября 2026")).assertExists()

        composeRule.onNode(hasSetTextAction()).performTextInput("850000,00")
        composeRule.onNode(save).assertIsEnabled().performClick()

        assertEquals(85_000_000L, state.amountMinor)
        assertEquals(1, saved)
    }

    @Test
    fun valueDeleteWaitsForConfirmation() {
        val state = ValueEditUiState(
            holdingId = flat.id,
            name = flat.name,
            today = today,
            amount = "11000000",
            date = LocalDate.of(2026, 1, 12),
            existing = true,
        )
        var deleted = false
        composeRule.setContent {
            AppTheme {
                ValueSheetContent(
                    state = state,
                    currency = "RUB",
                    onAmountChange = {},
                    onDateChange = {},
                    onSave = {},
                    onDelete = { deleted = true },
                )
            }
        }

        composeRule.onNode(hasText(res.getString(R.string.holding_value_date, "12 января 2026"))).assertIsNotEnabled()
        composeRule.onNodeWithText(res.getString(R.string.holding_value_delete)).performClick()
        assertFalse(deleted)

        composeRule.onNodeWithText(res.getString(R.string.holding_value_delete_confirm)).assertExists()
        composeRule.onNode(
            hasText(res.getString(R.string.holding_value_delete)) and hasAnyAncestor(isDialog()),
        ).performClick()
        assertTrue(deleted)
    }

    @Test
    fun deleteWaitsForConfirmation() {
        var deleted = false
        composeRule.setContent {
            AppTheme { EditScreen(editState(), onDelete = { deleted = true }) }
        }

        composeRule.onNodeWithText(res.getString(R.string.holding_delete)).performScrollTo().performClick()
        assertFalse(deleted)

        composeRule.onNodeWithText(res.getString(R.string.holding_delete_confirm)).assertExists()
        composeRule.onNode(hasText("Квартира") and hasAnyAncestor(isDialog())).assertExists()
        composeRule.onNode(
            hasText(res.getString(R.string.holding_delete)) and hasAnyAncestor(isDialog()),
        ).performClick()
        assertTrue(deleted)
    }

    @Test
    fun archiveWithValueOffersZeroSnapshot() {
        val archived = mutableListOf<Boolean>()
        composeRule.setContent {
            AppTheme { EditScreen(editState(), onToggleArchive = { archived += it }) }
        }

        composeRule.onNodeWithText(res.getString(R.string.holding_archive)).performScrollTo().performClick()
        assertTrue(archived.isEmpty())

        composeRule.onNodeWithText(res.getString(R.string.holding_archive_zero_confirm)).performClick()
        assertEquals(listOf(true), archived)
    }

    @Test
    fun sideIsChosenOnlyOnCreate() {
        composeRule.setContent {
            AppTheme { EditScreen(editState()) }
        }

        composeRule.onNodeWithText(res.getString(R.string.holding_side_liability)).assertDoesNotExist()
    }

    @Composable
    private fun EditScreen(
        state: HoldingEditUiState,
        onToggleArchive: (Boolean) -> Unit = {},
        onDelete: () -> Unit = {},
    ) {
        HoldingEditScreen(
            state = state,
            currency = "RUB",
            onSideChange = {},
            onNameChange = {},
            onKindChange = {},
            onIncomeChange = {},
            onExpenseChange = {},
            onSubmit = {},
            onToggleArchive = onToggleArchive,
            onDelete = onDelete,
            onRetry = {},
            onBack = {},
        )
    }
}
