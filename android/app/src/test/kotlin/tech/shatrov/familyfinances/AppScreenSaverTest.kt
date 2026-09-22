package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.SaverScope
import org.junit.Assert.assertEquals
import org.junit.Test
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.ui.overview.OverviewPeriod
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import tech.shatrov.familyfinances.ui.transactions.TransactionFilters
import tech.shatrov.familyfinances.ui.transactions.TransactionPeriod
import tech.shatrov.familyfinances.ui.transactions.TransactionPrefill
import java.time.LocalDate
import java.time.YearMonth
import java.util.UUID

/** Сохранение экрана: ключ — строка, поэтому расхождение `save`/`restore` ловится только тестом. */
class AppScreenSaverTest {
    private fun roundTrip(screen: AppScreen): AppScreen? {
        val key = with(AppScreenSaver) { SaverScope { true }.save(screen) }
        return AppScreenSaver.restore(requireNotNull(key) { "экран не сохранился" })
    }

    @Test
    fun rootScreensSurviveRoundTrip() {
        for (screen in listOf(
            AppScreen.Home,
            AppScreen.Overview(),
            AppScreen.Transactions(),
            AppScreen.Categories,
            AppScreen.Budgets,
            AppScreen.NetWorth,
        )) {
            assertEquals(screen, roundTrip(screen))
        }
    }

    @Test
    fun holdingEditKeepsSideOfNewHolding() {
        val screen = AppScreen.HoldingEdit(
            id = null,
            draft = UUID.fromString(FOOD_BUDGET_ID),
            side = HoldingSide.liability,
        )
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun holdingEditKeepsBothIds() {
        val screen = AppScreen.HoldingEdit(id = UUID.fromString(FLAT_ID), draft = UUID.fromString(FOOD_BUDGET_ID))
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun holdingHistoryKeepsId() {
        val screen = AppScreen.HoldingHistory(UUID.fromString(FLAT_ID))
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun budgetEditKeepsDraftWithoutId() {
        val screen = AppScreen.BudgetEdit(id = null, draft = UUID.fromString(FOOD_BUDGET_ID))
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun budgetEditKeepsBothIds() {
        val screen = AppScreen.BudgetEdit(
            id = UUID.fromString(ALL_BUDGET_ID),
            draft = UUID.fromString(FOOD_BUDGET_ID),
        )
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun transactionEditKeepsBothIds() {
        val screen = AppScreen.TransactionEdit(
            id = UUID.fromString(COFFEE_ID),
            draft = UUID.fromString(FOOD_BUDGET_ID),
        )
        assertEquals(screen, roundTrip(screen))
    }

    // Без `back` форма после поворота вернула бы на вкладку, а модель списка — фильтр сверки.
    @Test
    fun transactionEditKeepsReconciliationBack() {
        val month = YearMonth.of(2026, 8)
        val screen = AppScreen.TransactionEdit(
            id = UUID.fromString(COFFEE_ID),
            draft = UUID.fromString(FOOD_BUDGET_ID),
            back = AppScreen.Transactions(
                TransactionFilters.reconciliation(month),
                month,
            ),
        )
        assertEquals(screen, roundTrip(screen))
    }

    // «Закрыть разницу»: после поворота форма не теряет предзаполнение и возвращает в сверку.
    @Test
    fun transactionEditKeepsPrefillAndReconciliationBack() {
        val screen = AppScreen.TransactionEdit(
            id = null,
            draft = UUID.fromString(FOOD_BUDGET_ID),
            back = AppScreen.Reconciliation(YearMonth.of(2026, 8)),
            prefill = TransactionPrefill(
                amountMinor = 150_050,
                type = TransactionType.income,
                date = LocalDate.of(2026, 8, 31),
                description = "Корректировка: сверки, 100%",
            ),
        )
        assertEquals(screen, roundTrip(screen))
    }

    // Ключ версии 0.11: сразу за черновиком `back`, без поля предзаполнения.
    @Test
    fun transactionEditFromPreviousVersionKeepsBack() {
        val month = YearMonth.of(2026, 8)
        val restored = AppScreenSaver.restore(
            "transaction-edit:$COFFEE_ID:$FOOD_BUDGET_ID:transactions:MONTH:2026-08:expense:::true:::2026-08:",
        )
        val back = AppScreen.Transactions(
            TransactionFilters(
                period = TransactionPeriod.MONTH,
                type = TransactionType.expense,
                month = month,
                unassigned = true,
            ),
            month,
        )
        assertEquals(
            AppScreen.TransactionEdit(UUID.fromString(COFFEE_ID), UUID.fromString(FOOD_BUDGET_ID), back),
            restored,
        )
    }

    @Test
    fun transactionEditFromOldBundleReturnsToTab() {
        val restored = AppScreenSaver.restore("transaction-edit::$FOOD_BUDGET_ID")
        assertEquals(AppScreen.TransactionEdit(id = null, draft = UUID.fromString(FOOD_BUDGET_ID)), restored)
    }

    @Test
    fun recognizeKeepsImportId() {
        val screen = AppScreen.Recognize(UUID.fromString(COFFEE_ID))
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun reconciliationKeepsMonth() {
        val screen = AppScreen.Reconciliation(YearMonth.of(2026, 8))
        assertEquals(screen, roundTrip(screen))
    }

    // Расшифровка сверки переживает поворот вместе с фильтром и месяцем, куда ведёт «назад».
    @Test
    fun transactionsKeepReconciliationFilters() {
        val month = YearMonth.of(2026, 8)
        val screens = listOf(
            AppScreen.Transactions(TransactionFilters.reconciliation(month), month),
            AppScreen.Transactions(TransactionFilters(period = TransactionPeriod.THIS_MONTH), month),
        )
        for (screen in screens) {
            assertEquals(screen, roundTrip(screen))
        }
    }

    // Без категории после поворота расшифровка «Обзора» показала бы все операции диапазона,
    // без периода «назад» вернуло бы в «Обзор» на другом чипе.
    @Test
    fun transactionsKeepOverviewRangeCategoryAndPeriod() {
        val screen = AppScreen.Transactions(
            TransactionFilters.overview(
                TransactionType.expense,
                UUID.fromString(COFFEE_ID),
                LocalDate.of(2026, 7, 1),
                LocalDate.of(2026, 9, 19),
            ),
            overview = OverviewPeriod.ThreeMonths,
        )
        assertEquals(screen, roundTrip(screen))
    }

    @Test
    fun transactionsKeepCategorySet() {
        val two = linkedSetOf(UUID.fromString(COFFEE_ID), UUID.fromString(GROCERIES_ID))
        for (ids in listOf(two, emptySet())) {
            val screen = AppScreen.Transactions(TransactionFilters(categoryIds = ids))
            val restored = roundTrip(screen) as AppScreen.Transactions
            assertEquals(screen, restored)
            assertEquals(ids.toList(), restored.filters!!.categoryIds.toList())
        }
    }

    // Бандл 0.13.0: в поле категории один uuid.
    @Test
    fun transactionsFromBundleWithSingleCategoryLoad() {
        val restored = AppScreenSaver.restore("transactions:ALL:::$GROCERIES_ID::false::::")

        assertEquals(
            AppScreen.Transactions(TransactionFilters(categoryIds = setOf(UUID.fromString(GROCERIES_ID)))),
            restored,
        )
    }

    @Test
    fun overviewKeepsEveryPeriod() {
        for (period in OverviewPeriod.chips + OverviewPeriod.Month(YearMonth.of(2026, 8))) {
            val screen = AppScreen.Overview(period)
            assertEquals(screen, roundTrip(screen))
        }
    }

    @Test
    fun transactionsFromOldBundleLoad() {
        assertEquals(AppScreen.Loading, AppScreenSaver.restore("transactions:MONTH:2026-08:expense::true:2026-08"))
        assertEquals(AppScreen.Loading, AppScreenSaver.restore("transactions:ALL:::::false:::"))
    }

    @Test
    fun settingsPagesSurviveRoundTrip() {
        val visit = UUID.fromString(FOOD_BUDGET_ID)
        val target = UUID.fromString(COFFEE_ID)
        val pages = listOf(
            SettingsPage.Root(visit),
            SettingsPage.Profile(visit),
            SettingsPage.Password(visit),
            SettingsPage.Sessions(visit),
            SettingsPage.Users(visit),
            SettingsPage.UserEdit(null, visit),
            SettingsPage.UserEdit(target, visit),
            SettingsPage.UserPassword(target, visit),
            SettingsPage.Family(visit),
            SettingsPage.Backups(visit),
            SettingsPage.Accounts(visit),
        )

        for (page in pages) {
            val screen = AppScreen.Settings(page)
            assertEquals(screen, roundTrip(screen))
        }
    }

    @Test
    fun unknownKeyFallsBackToLoading() {
        assertEquals(AppScreen.Loading, AppScreenSaver.restore("что-то не то"))
    }

    /** Бандл прошлой версии несёт ключ другого формата: разбор его не роняет приложение. */
    @Test
    fun malformedKeyFallsBackToLoading() {
        val keys = listOf(
            "settings:root",
            "settings:root::не-uuid",
            "transaction-edit:",
            "budget-edit:x:y",
            "transactions:MONTH:2026-08",
            "reconciliation:август",
            "overview:WEEK",
        )
        for (key in keys) {
            assertEquals(AppScreen.Loading, AppScreenSaver.restore(key))
        }
    }
}
