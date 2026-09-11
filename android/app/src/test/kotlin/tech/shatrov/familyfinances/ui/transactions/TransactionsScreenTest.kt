package tech.shatrov.familyfinances.ui.transactions

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
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
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class TransactionsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: TransactionsUiState,
        onFiltersChange: (TransactionFilters) -> Unit = {},
        onOpen: (UUID) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                TransactionsScreen(
                    state = state,
                    onRetry = {},
                    onFiltersChange = onFiltersChange,
                    onLoadMore = {},
                    onCreate = {},
                    onOpen = onOpen,
                )
            }
        }
    }

    @Test
    fun emptyListSaysSo() {
        show(ready(emptyList()))

        composeRule.onNodeWithText(res.getString(R.string.transactions_empty)).assertIsDisplayed()
    }

    @Test
    fun rowShowsDayCategoryAuthorAndAmount() {
        show(ready(listOf(row(description = "Кофе", categoryName = "Продукты", authorName = "Член"))))

        composeRule.onNodeWithText(formatDay(LocalDate.parse("2026-09-07"))).assertIsDisplayed()
        composeRule.onNodeWithText("Кофе").assertIsDisplayed()
        composeRule.onNodeWithText("Продукты · Член").assertIsDisplayed()
        composeRule.onNodeWithText("-1 500,00 ₽").assertIsDisplayed()
    }

    @Test
    fun ownRowIsSignedByPronoun() {
        show(ready(listOf(row(authorName = "Админ", isMine = true))))

        composeRule
            .onNodeWithText("Продукты · ${res.getString(R.string.transactions_author_me)}")
            .assertIsDisplayed()
    }

    @Test
    fun tappingFilterReachesCallback() {
        var filters: TransactionFilters? = null
        show(ready(listOf(row())), onFiltersChange = { filters = it })

        composeRule.onNodeWithText(res.getString(R.string.filter_expense)).performClick()

        assertEquals(TransactionType.expense, filters?.type)
    }

    @Test
    fun tappingRowOpensTransaction() {
        var opened: UUID? = null
        show(ready(listOf(row())), onOpen = { opened = it })

        composeRule.onNodeWithText("Кофе").performClick()

        assertEquals(UUID.fromString("88888888-8888-8888-8888-888888888881"), opened)
    }

    private fun ready(rows: List<TransactionRow>) = TransactionsUiState.Ready(
        groups = if (rows.isEmpty()) emptyList() else listOf(DayGroup(LocalDate.parse("2026-09-07"), rows)),
        currency = "RUB",
        filters = TransactionFilters(),
        categories = emptyList(),
        hasMore = false,
    )

    private fun row(
        description: String = "Кофе",
        categoryName: String? = "Продукты",
        authorName: String? = null,
        isMine: Boolean = false,
    ) = TransactionRow(
        transaction = Transaction(
            id = UUID.fromString("88888888-8888-8888-8888-888888888881"),
            amountMinor = 150000,
            type = TransactionType.expense,
            description = description,
            categoryId = UUID.fromString("44444444-4444-4444-4444-444444444444"),
            userId = UUID.fromString("11111111-1111-1111-1111-111111111111"),
            date = LocalDate.parse("2026-09-07"),
            tags = emptyList(),
            createdAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
            updatedAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
        ),
        categoryName = categoryName,
        authorName = authorName,
        isMine = isMine,
    )
}
