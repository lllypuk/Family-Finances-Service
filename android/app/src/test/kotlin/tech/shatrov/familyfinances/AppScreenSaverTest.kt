package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.SaverScope
import org.junit.Assert.assertEquals
import org.junit.Test
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import tech.shatrov.familyfinances.ui.transactions.TransactionFilters
import tech.shatrov.familyfinances.ui.transactions.TransactionPeriod
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
        for (screen in listOf(AppScreen.Home, AppScreen.Transactions(), AppScreen.Categories, AppScreen.Budgets)) {
            assertEquals(screen, roundTrip(screen))
        }
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
            AppScreen.Transactions(TransactionFilters.reconciliation(month, UUID.fromString(COFFEE_ID)), month),
            AppScreen.Transactions(TransactionFilters.reconciliation(month, null), month),
            AppScreen.Transactions(TransactionFilters(period = TransactionPeriod.THIS_MONTH), month),
        )
        for (screen in screens) {
            assertEquals(screen, roundTrip(screen))
        }
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
        )
        for (key in keys) {
            assertEquals(AppScreen.Loading, AppScreenSaver.restore(key))
        }
    }
}
