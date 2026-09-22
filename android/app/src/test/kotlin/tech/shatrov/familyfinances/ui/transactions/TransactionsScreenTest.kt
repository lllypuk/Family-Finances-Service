package tech.shatrov.familyfinances.ui.transactions

import android.content.Context
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.recognize.ImportLaunchers
import tech.shatrov.familyfinances.ui.recognize.ImportSourceSheetContent
import java.time.LocalDate
import java.time.OffsetDateTime
import java.util.UUID

private val CATEGORY_ID: UUID = UUID.fromString(GROCERIES_ID)

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class TransactionsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: TransactionsUiState,
        filters: TransactionFilters = TransactionFilters(),
        categories: List<Category> = emptyList(),
        accounts: List<Account> = emptyList(),
        onFiltersChange: (TransactionFilters) -> Unit = {},
        onCreate: () -> Unit = {},
        onOpen: (UUID) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                TransactionsScreen(
                    state = state,
                    filters = filters,
                    categories = categories,
                    accounts = accounts,
                    onRetry = {},
                    onFiltersChange = onFiltersChange,
                    onLoadMore = {},
                    onCreate = onCreate,
                    onOpen = onOpen,
                    importLaunchers = ImportLaunchers(gallery = {}, camera = {}),
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
    fun emptyListWithoutFiltersOffersToAddTransaction() {
        var created = false
        show(ready(emptyList()), onCreate = { created = true })

        composeRule.onNodeWithText(res.getString(R.string.transactions_add_first)).performClick()

        assertTrue(created)
    }

    @Test
    fun emptyListWithFiltersOffersToResetThem() {
        var filters: TransactionFilters? = null
        show(
            ready(emptyList()),
            filters = TransactionFilters(type = TransactionType.income),
            onFiltersChange = { filters = it },
        )

        composeRule.onNodeWithText(res.getString(R.string.transactions_reset_filters)).performClick()

        assertEquals(TransactionFilters(), filters)
    }

    @Test
    fun rowShowsDayCategoryAuthorAndAmount() {
        show(ready(listOf(row(description = "Кофе", categoryName = "Продукты", authorName = "Член"))))

        composeRule.onNodeWithText(formatDay(LocalDate.parse("2026-09-07"))).assertIsDisplayed()
        composeRule.onNodeWithText("Кофе").assertIsDisplayed()
        composeRule.onNodeWithText("Продукты · Член").assertIsDisplayed()
        composeRule.onNodeWithContentDescription("Продукты", useUnmergedTree = true).assertIsDisplayed()
        composeRule.onNodeWithText("-1 500,00 ₽").assertIsDisplayed()
    }

    @Test
    fun rowWithoutCategoryShowsPlaceholderAvatar() {
        show(ready(listOf(row(categoryName = null))))

        composeRule
            .onNodeWithContentDescription(res.getString(R.string.transactions_no_category), useUnmergedTree = true)
            .assertIsDisplayed()
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

    // Фильтры вне `when`: упавший запрос иначе не переключить — «Повторить» повторяет ровно его.
    @Test
    fun filtersStayOnFailure() {
        show(TransactionsUiState.Failure(UiError.Server("всё сломалось")))

        composeRule.onNodeWithText(res.getString(R.string.filter_this_month)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.filter_all_categories)).assertIsDisplayed()
    }

    @Test
    fun filtersStayWhileLoading() {
        show(TransactionsUiState.Loading)

        composeRule.onNodeWithText(res.getString(R.string.filter_this_month)).assertIsDisplayed()
        composeRule.onNodeWithText(res.getString(R.string.filter_all_categories)).assertIsDisplayed()
    }

    @Test
    fun categoryChipShowsSelectedName() {
        show(
            ready(listOf(row(categoryName = null))),
            filters = TransactionFilters(categoryIds = setOf(CATEGORY_ID)),
            categories = listOf(category()),
        )

        composeRule.onNodeWithText("Продукты").assertIsDisplayed()
    }

    @Test
    fun categoryChipShowsDashWhenSelectedCategoryIsGone() {
        show(
            ready(emptyList()),
            filters = TransactionFilters(categoryIds = setOf(CATEGORY_ID)),
            categories = emptyList(),
        )

        composeRule.onNodeWithText("—").assertIsDisplayed()
        composeRule.onAllNodesWithText(res.getString(R.string.filter_all_categories)).assertCountEquals(0)
    }

    // Тело листа отдельно от `ModalBottomSheet`: тот в Robolectric ненадёжен.
    @Test
    fun categorySheetSelectionReachesCallback() {
        var picked: UUID? = null
        var called = false
        composeRule.setContent {
            AppTheme {
                CategorySheetContent(listOf(category()), selected = null) {
                    picked = it
                    called = true
                }
            }
        }

        composeRule.onNodeWithText("Продукты").performClick()

        assertTrue(called)
        assertEquals(CATEGORY_ID, picked)
    }

    @Test
    fun categorySheetShowsParentOfSubcategory() {
        val child = category().copy(id = UUID.randomUUID(), name = "Прочее", parentId = CATEGORY_ID)
        composeRule.setContent {
            AppTheme { CategorySheetContent(listOf(category(), child), selected = null) {} }
        }

        composeRule.onNodeWithText("Продукты / Прочее").assertIsDisplayed()
        composeRule.onNodeWithContentDescription("Прочее", useUnmergedTree = true).assertIsDisplayed()
    }

    // Первое имя — по справочнику, не по порядку отметок: так же категории уходят в запрос.
    @Test
    fun categoryChipShowsFirstNameAndCountOfOthers() {
        val home = category().copy(id = UUID.randomUUID(), name = "Дом")
        val cafe = category().copy(id = UUID.randomUUID(), name = "Кафе")
        show(
            ready(emptyList()),
            filters = TransactionFilters(categoryIds = setOf(cafe.id, CATEGORY_ID, home.id)),
            categories = listOf(category(), home, cafe),
        )

        composeRule.onNodeWithText("Продукты +2").assertIsDisplayed()
    }

    @Test
    fun multiSheetReturnsBothCheckedOnDone() {
        val home = category().copy(id = UUID.randomUUID(), name = "Дом")
        var done: Set<UUID>? = null
        composeRule.setContent {
            AppTheme { CategorySheetContent(listOf(category(), home), selectedIds = emptySet()) { done = it } }
        }

        composeRule.onNodeWithText("Продукты").performClick()
        composeRule.onNodeWithText("Дом").performClick()
        assertNull(done)
        composeRule.onNodeWithText(res.getString(R.string.filter_done)).performClick()

        assertEquals(setOf(CATEGORY_ID, home.id), done)
    }

    @Test
    fun multiSheetUncheckDropsCategory() {
        val home = category().copy(id = UUID.randomUUID(), name = "Дом")
        var done: Set<UUID>? = null
        composeRule.setContent {
            AppTheme {
                CategorySheetContent(listOf(category(), home), selectedIds = setOf(CATEGORY_ID, home.id)) { done = it }
            }
        }

        composeRule.onNodeWithText("Дом").performClick()
        composeRule.onNodeWithText(res.getString(R.string.filter_done)).performClick()

        assertEquals(setOf(CATEGORY_ID), done)
    }

    @Test
    fun multiSheetAllClearsSelection() {
        var done: Set<UUID>? = null
        composeRule.setContent {
            AppTheme { CategorySheetContent(listOf(category()), selectedIds = setOf(CATEGORY_ID)) { done = it } }
        }

        composeRule.onNodeWithText(res.getString(R.string.filter_all_categories)).performClick()
        composeRule.onNodeWithText(res.getString(R.string.filter_done)).performClick()

        assertEquals(emptySet<UUID>(), done)
    }

    @Test
    fun importSheetStartsChosenSource() {
        var gallery = 0
        var camera = 0
        composeRule.setContent {
            AppTheme { ImportSourceSheetContent(onGallery = { gallery++ }, onCamera = { camera++ }) }
        }

        composeRule.onNodeWithText(res.getString(R.string.recognize_from_gallery)).performClick()
        composeRule.onNodeWithText(res.getString(R.string.recognize_take_photo)).performClick()

        assertEquals(1, gallery)
        assertEquals(1, camera)
    }

    @Test
    fun scanIconIsInHeader() {
        show(ready(emptyList()))

        composeRule.onNodeWithContentDescription(res.getString(R.string.recognize_scan)).assertIsDisplayed()
    }

    private fun category() = Category(
        id = CATEGORY_ID,
        name = "Продукты",
        type = CategoryType.expense,
        color = "#ff0000",
        icon = "cart",
        isActive = true,
        createdAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
        updatedAt = OffsetDateTime.parse("2026-09-07T09:00:00Z"),
    )

    private fun ready(rows: List<TransactionRow>) = TransactionsUiState.Ready(
        groups = if (rows.isEmpty()) emptyList() else listOf(DayGroup(LocalDate.parse("2026-09-07"), rows)),
        currency = "RUB",
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
        category = categoryName?.let { category().copy(name = it) },
        authorName = authorName,
        isMine = isMine,
    )

    @Test
    fun categorySheetManageReachesCallback() {
        var managed = 0
        var picked = false
        composeRule.setContent {
            AppTheme {
                CategorySheetContent(listOf(category()), selected = null, onManage = { managed++ }) { picked = true }
            }
        }

        composeRule.onNodeWithText(res.getString(R.string.categories_manage)).performClick()

        assertEquals(1, managed)
        assertFalse(picked)
    }

    // Переход в справочник отметки не применяет: «Готово» не нажата.
    @Test
    fun multiSheetManageDoesNotApplyDraft() {
        var managed = 0
        var done: Set<UUID>? = null
        composeRule.setContent {
            AppTheme {
                CategorySheetContent(listOf(category()), selectedIds = emptySet(), onManage = { managed++ }) {
                    done = it
                }
            }
        }

        composeRule.onNodeWithText("Продукты").performClick()
        composeRule.onNodeWithText(res.getString(R.string.categories_manage)).performClick()

        assertEquals(1, managed)
        assertNull(done)
    }
}
