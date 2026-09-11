package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.Saver
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import tech.shatrov.familyfinances.ui.settings.restoreSettingsPage
import tech.shatrov.familyfinances.ui.settings.saveKey
import java.util.UUID

/**
 * Экран приложения. Навигационной библиотеки нет намеренно: экранов горстка, переходов между
 * ними — единицы.
 */
sealed interface AppScreen {
    /** Токен есть, но роль и валюта ещё не приехали. */
    data object Loading : AppScreen

    data object Login : AppScreen

    data object Home : AppScreen

    data object Transactions : AppScreen

    data object Categories : AppScreen

    /**
     * Форма операции; `id` = `null` — новая, тело правки перечитывается с сервера.
     * `draft` — клиентский UUID создаваемой записи: он же ключ модели, поэтому следующий заход
     * на форму получает чистую, а повтор после обрыва — ту же и не создаёт вторую запись.
     */
    data class TransactionEdit(
        val id: UUID?,
        val draft: UUID = UUID.randomUUID(),
    ) : AppScreen

    data object Budgets : AppScreen

    /** Форма бюджета; `id` и `draft` работают так же, как у [TransactionEdit]. */
    data class BudgetEdit(
        val id: UUID?,
        val draft: UUID = UUID.randomUUID(),
    ) : AppScreen

    /** Настройки целиком: страницы внутри переключает свой хост, а не этот `when`. */
    data class Settings(val page: SettingsPage = SettingsPage.Root()) : AppScreen
}

private const val KEY_LOADING = "loading"
private const val KEY_LOGIN = "login"
private const val KEY_HOME = "home"
private const val KEY_TRANSACTIONS = "transactions"
private const val KEY_TRANSACTION_EDIT = "transaction-edit"
private const val KEY_CATEGORIES = "categories"
private const val KEY_BUDGETS = "budgets"
private const val KEY_BUDGET_EDIT = "budget-edit"
private const val KEY_SETTINGS = "settings"

/** Экран переживает поворот; всё остальное восстанавливается из хранилища токена. */
val AppScreenSaver: Saver<AppScreen, String> = Saver(
    save = { screen ->
        when (screen) {
            AppScreen.Loading -> KEY_LOADING
            AppScreen.Login -> KEY_LOGIN
            AppScreen.Home -> KEY_HOME
            AppScreen.Transactions -> KEY_TRANSACTIONS
            AppScreen.Categories -> KEY_CATEGORIES
            AppScreen.Budgets -> KEY_BUDGETS
            is AppScreen.TransactionEdit -> "$KEY_TRANSACTION_EDIT:${screen.id ?: ""}:${screen.draft}"
            is AppScreen.BudgetEdit -> "$KEY_BUDGET_EDIT:${screen.id ?: ""}:${screen.draft}"
            is AppScreen.Settings -> "$KEY_SETTINGS:${screen.page.saveKey()}"
        }
    },
    restore = { key ->
        when {
            key == KEY_LOGIN -> AppScreen.Login

            key == KEY_HOME -> AppScreen.Home

            key == KEY_TRANSACTIONS -> AppScreen.Transactions

            key == KEY_CATEGORIES -> AppScreen.Categories

            key == KEY_BUDGETS -> AppScreen.Budgets

            key.startsWith("$KEY_TRANSACTION_EDIT:") -> {
                val (target, draft) = key.removePrefix("$KEY_TRANSACTION_EDIT:").split(':')
                AppScreen.TransactionEdit(
                    id = target.takeIf { it.isNotEmpty() }?.let(UUID::fromString),
                    draft = UUID.fromString(draft),
                )
            }

            key.startsWith("$KEY_BUDGET_EDIT:") -> {
                val (target, draft) = key.removePrefix("$KEY_BUDGET_EDIT:").split(':')
                AppScreen.BudgetEdit(
                    id = target.takeIf { it.isNotEmpty() }?.let(UUID::fromString),
                    draft = UUID.fromString(draft),
                )
            }

            key.startsWith("$KEY_SETTINGS:") -> {
                val (page, target, visit) = key.removePrefix("$KEY_SETTINGS:").split(':')
                restoreSettingsPage(page, target, visit)?.let(AppScreen::Settings) ?: AppScreen.Loading
            }

            else -> AppScreen.Loading
        }
    },
)
