package tech.shatrov.familyfinances.ui.recognize

import android.content.Context
import android.net.Uri
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.assertIsEnabled
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.getUnclippedBoundsInRoot
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.COFFEE_ID
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.SimilarTransaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError
import java.io.File
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

private val TODAY: LocalDate = LocalDate.parse("2026-09-16")

// Высокий экран: строки кандидатов крупные, и ленивый список иначе не собрал бы нижние группы.
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK], qualifiers = "w411dp-h2400dp")
class RecognizeScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: RecognizeUiState,
        onSave: () -> Unit = {},
        onRetryRow: (UUID) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                RecognizeScreen(
                    state = state,
                    currency = "RUB",
                    waiting = false,
                    today = TODAY,
                    onIncludedChange = { _, _ -> },
                    onDateChange = { _, _ -> },
                    onCategoryChange = { _, _ -> },
                    onDescriptionChange = { _, _ -> },
                    onSave = onSave,
                    onRetry = {},
                    onRetryRow = onRetryRow,
                    onBack = {},
                )
            }
        }
    }

    @Test
    fun rowsAreGroupedUnderTheirImages() {
        show(review(row("Шавуха", image = 0), row("Зарплата", image = 1), row("Без картинки", image = null)))

        val first = top(res.getString(R.string.recognize_image, 1))
        val second = top(res.getString(R.string.recognize_image, 2))
        val unattached = top(res.getString(R.string.recognize_unattached))
        assertTrue(first < top("Шавуха") && top("Шавуха") < second)
        assertTrue(second < top("Зарплата") && top("Зарплата") < unattached)
        assertTrue(unattached < top("Без картинки"))
    }

    @Test
    fun saveCountsOnlyIncludedAndSavableRows() {
        var saved = false
        show(
            review(
                row("Шавуха"),
                row("Кофе", included = false),
                row("Без даты", date = null),
                row("Год не тот", dateAssumed = true),
                row("Без категории", categoryId = null),
            ),
            onSave = { saved = true },
        )

        composeRule.onNodeWithText(res.getString(R.string.recognize_save, 1)).assertIsEnabled().performClick()
        assertTrue(saved)
    }

    @Test
    fun missingDateAssumedYearAndMissingCategoryBlockSaving() {
        show(
            review(
                row("Без даты", date = null),
                row("Год не тот", date = LocalDate.parse("2026-09-10"), dateAssumed = true),
                row("Без категории", categoryId = null),
            ),
        )

        composeRule.onNodeWithText(res.getString(R.string.recognize_pick_date)).assertIsDisplayed()
        composeRule
            .onNodeWithText(res.getString(R.string.recognize_date_assumed, "10 сентября 2026"))
            .assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.recognize_pick_category)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.recognize_save, 0)).assertIsNotEnabled()
    }

    @Test
    fun similarBadgeAndIncompleteNoteAreShown() {
        val similar = SimilarTransaction(UUID.fromString(COFFEE_ID), LocalDate.parse("2026-09-14"), "шавуха")
        show(review(row("Шавуха", similarTo = listOf(similar))).copy(incomplete = true))

        composeRule.onNodeWithText(
            res.getString(R.string.recognize_similar, "шавуха", "14 сентября"),
        ).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.recognize_incomplete)).assertIsDisplayed()
        // Похожая — не дубль: строка остаётся к сохранению.
        composeRule.onNodeWithText(res.getString(R.string.recognize_save, 1)).assertIsEnabled()
    }

    @Test
    fun foreignCurrencyIsExplained() {
        show(review(row("Такси", currency = "USD", currencyMismatch = true)))

        composeRule.onNodeWithText(
            res.getString(R.string.recognize_currency_mismatch, "USD", "RUB"),
        ).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.recognize_save, 0)).assertIsNotEnabled()
    }

    @Test
    fun failedRowOffersRetryOfItself() {
        var retried: UUID? = null
        val failed = row("Шавуха", status = RowStatus.Failed(UiError.Network))
        show(review(failed, row("Кофе", status = RowStatus.Saved)), onRetryRow = { retried = it })

        composeRule.onNodeWithText(res.getString(R.string.recognize_saved)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.error_network)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.retry)).performClick()
        assertEquals(failed.draft, retried)
    }

    @Test
    fun emptyResultIsNotAnError() {
        show(RecognizeUiState(phase = RecognizePhase.Review(emptyList()), images = images()))

        composeRule.onNodeWithText(res.getString(R.string.recognize_empty)).assertIsDisplayed()
        composeRule.onAllNodesWithText(res.getString(R.string.retry)).assertCountEquals(0)
    }

    @Test
    fun unavailableFailureCanBeRetried() {
        show(
            RecognizeUiState(
                phase = RecognizePhase.Failure(
                    UiError.Resource(R.string.recognize_error_unavailable),
                    retryable = true,
                ),
            ),
        )

        composeRule.onNodeWithText(res.getString(R.string.recognize_error_unavailable)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.retry)).assertIsDisplayed()
    }

    private fun top(text: String) = composeRule.onNodeWithText(text).getUnclippedBoundsInRoot().top

    private fun images(): List<ImportImage> = listOf(0, 1).map {
        ImportImage.Ready(Uri.parse("content://shots/$it"), File("missing-$it.jpg"))
    }

    private fun review(vararg rows: RecognizedRow) = RecognizeUiState(
        phase = RecognizePhase.Review(rows.toList()),
        images = images(),
        categories = listOf(category()),
    )

    private fun row(
        description: String,
        image: Int? = 0,
        date: LocalDate? = TODAY,
        dateAssumed: Boolean = false,
        categoryId: UUID? = UUID.fromString(GROCERIES_ID),
        included: Boolean = true,
        currency: String? = null,
        currencyMismatch: Boolean = false,
        similarTo: List<SimilarTransaction> = emptyList(),
        status: RowStatus = RowStatus.Pending,
    ) = RecognizedRow(
        draft = UUID.randomUUID(),
        image = image,
        amountMinor = 30000,
        currency = currency,
        currencyMismatch = currencyMismatch,
        type = TransactionType.expense,
        date = date,
        dateAssumed = dateAssumed,
        description = description,
        categoryId = categoryId,
        similarTo = similarTo,
        included = included,
        status = status,
    )

    private fun category() = Category(
        id = UUID.fromString(GROCERIES_ID),
        name = "Продукты",
        type = CategoryType.expense,
        color = "#ff0000",
        icon = "cart",
        isActive = true,
        createdAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
    )
}
