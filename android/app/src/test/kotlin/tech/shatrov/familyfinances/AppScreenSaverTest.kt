package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.SaverScope
import org.junit.Assert.assertEquals
import org.junit.Test
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import java.util.UUID

/** Сохранение экрана: ключ — строка, поэтому расхождение `save`/`restore` ловится только тестом. */
class AppScreenSaverTest {
    private fun roundTrip(screen: AppScreen): AppScreen? {
        val key = with(AppScreenSaver) { SaverScope { true }.save(screen) }
        return AppScreenSaver.restore(requireNotNull(key) { "экран не сохранился" })
    }

    @Test
    fun rootScreensSurviveRoundTrip() {
        for (screen in listOf(AppScreen.Home, AppScreen.Transactions, AppScreen.Categories, AppScreen.Budgets)) {
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
}
