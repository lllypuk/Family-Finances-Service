package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.Saver
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.ui.overview.OverviewPeriod
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import tech.shatrov.familyfinances.ui.settings.restoreSettingsPage
import tech.shatrov.familyfinances.ui.settings.saveKey
import tech.shatrov.familyfinances.ui.transactions.TransactionFilters
import tech.shatrov.familyfinances.ui.transactions.TransactionPeriod
import tech.shatrov.familyfinances.ui.transactions.TransactionPrefill
import java.net.URLDecoder
import java.net.URLEncoder
import java.time.LocalDate
import java.time.YearMonth
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

    data class Overview(val period: OverviewPeriod = OverviewPeriod.ThisMonth) : AppScreen

    /**
     * Список операций. [filters] — фильтр, с которым его открыли снаружи (расшифровка сверки или
     * «Обзора»), [reconciliation] и [overview] — куда ведёт «назад»; у вкладки все `null`.
     */
    data class Transactions(
        val filters: TransactionFilters? = null,
        val reconciliation: YearMonth? = null,
        val overview: OverviewPeriod? = null,
    ) : AppScreen {
        val drilled: Boolean get() = reconciliation != null || overview != null
    }

    data object Categories : AppScreen

    /**
     * Форма операции; `id` = `null` — новая, тело правки перечитывается с сервера.
     * `draft` — клиентский UUID создаваемой записи: он же ключ модели, поэтому следующий заход
     * на форму получает чистую, а повтор после обрыва — ту же и не создаёт вторую запись.
     * `back` — экран, куда форма возвращает; пишется в бандл вместе с фильтром расшифровки,
     * иначе после поворота форма вернула бы на вкладку, а модель списка — фильтр сверки.
     * `prefill` — поля новой операции, которые открывший экран заполнил за пользователя.
     */
    data class TransactionEdit(
        val id: UUID?,
        val draft: UUID = UUID.randomUUID(),
        val back: AppScreen = Transactions(),
        val prefill: TransactionPrefill? = null,
    ) : AppScreen

    data object Budgets : AppScreen

    /** Форма бюджета; `id` и `draft` работают так же, как у [TransactionEdit]. */
    data class BudgetEdit(
        val id: UUID?,
        val draft: UUID = UUID.randomUUID(),
    ) : AppScreen

    data object NetWorth : AppScreen

    /** Форма позиции капитала; `id` и `draft` — как у [TransactionEdit], [side] выбран заранее для новой. */
    data class HoldingEdit(
        val id: UUID?,
        val draft: UUID = UUID.randomUUID(),
        val side: HoldingSide = HoldingSide.asset,
    ) : AppScreen

    /** История снимков позиции [id]. */
    data class HoldingHistory(val id: UUID) : AppScreen

    /** Распознавание импорта [importId] из [ImportStore]; после смерти процесса — из его журнала. */
    data class Recognize(val importId: UUID) : AppScreen

    /** Сверка счетов за [month]. */
    data class Reconciliation(val month: YearMonth) : AppScreen

    /** Настройки целиком: страницы внутри переключает свой хост, а не этот `when`. */
    data class Settings(val page: SettingsPage = SettingsPage.Root()) : AppScreen
}

private const val KEY_LOADING = "loading"
private const val KEY_LOGIN = "login"
private const val KEY_HOME = "home"
private const val KEY_OVERVIEW = "overview"
private const val KEY_TRANSACTIONS = "transactions"
private const val KEY_TRANSACTION_EDIT = "transaction-edit"
private const val KEY_CATEGORIES = "categories"
private const val KEY_BUDGETS = "budgets"
private const val KEY_BUDGET_EDIT = "budget-edit"
private const val KEY_NET_WORTH = "net-worth"
private const val KEY_HOLDING_EDIT = "holding-edit"
private const val KEY_HOLDING_HISTORY = "holding-history"
private const val KEY_SETTINGS = "settings"
private const val KEY_RECOGNIZE = "recognize"
private const val KEY_RECONCILIATION = "reconciliation"

// Ключ другой длины — из прошлой версии: `restore` уводит его в загрузку, а не читает поля не по местам.
private const val TRANSACTIONS_KEY_FIELDS = 10

/** Экран переживает поворот; всё остальное восстанавливается из хранилища токена. */
val AppScreenSaver: Saver<AppScreen, String> = Saver(
    save = { screen -> saveScreen(screen) },
    // Ключ мог прийти из бандла прошлой версии: разбор чужого формата — не крэш, а загрузка.
    restore = { key ->
        runCatching { restoreScreen(key) }.getOrNull() ?: AppScreen.Loading
    },
)

private fun saveScreen(screen: AppScreen): String = when (screen) {
    AppScreen.Loading -> KEY_LOADING

    AppScreen.Login -> KEY_LOGIN

    AppScreen.Home -> KEY_HOME

    is AppScreen.Overview -> "$KEY_OVERVIEW:${screen.period.saveKey()}"

    is AppScreen.Transactions -> screen.saveKey()

    AppScreen.Categories -> KEY_CATEGORIES

    AppScreen.Budgets -> KEY_BUDGETS

    is AppScreen.TransactionEdit ->
        "$KEY_TRANSACTION_EDIT:${screen.id ?: ""}:${screen.draft}:${screen.prefill?.saveKey().orEmpty()}:" +
            saveScreen(screen.back)

    is AppScreen.BudgetEdit -> "$KEY_BUDGET_EDIT:${screen.id ?: ""}:${screen.draft}"

    AppScreen.NetWorth -> KEY_NET_WORTH

    is AppScreen.HoldingEdit -> "$KEY_HOLDING_EDIT:${screen.id ?: ""}:${screen.draft}:${screen.side.value}"

    is AppScreen.HoldingHistory -> "$KEY_HOLDING_HISTORY:${screen.id}"

    is AppScreen.Settings -> "$KEY_SETTINGS:${screen.page.saveKey()}"

    is AppScreen.Recognize -> "$KEY_RECOGNIZE:${screen.importId}"

    is AppScreen.Reconciliation -> "$KEY_RECONCILIATION:${screen.month}"
}

private fun restoreScreen(key: String): AppScreen? = when {
    key == KEY_LOGIN -> AppScreen.Login

    key == KEY_HOME -> AppScreen.Home

    key.startsWith("$KEY_OVERVIEW:") ->
        AppScreen.Overview(OverviewPeriod.fromSaveKey(key.removePrefix("$KEY_OVERVIEW:")))

    key == KEY_TRANSACTIONS -> AppScreen.Transactions()

    key.startsWith("$KEY_TRANSACTIONS:") -> restoreTransactions(key.removePrefix("$KEY_TRANSACTIONS:"))

    key == KEY_CATEGORIES -> AppScreen.Categories

    key == KEY_BUDGETS -> AppScreen.Budgets

    key.startsWith("$KEY_TRANSACTION_EDIT:") -> {
        val parts = key.removePrefix("$KEY_TRANSACTION_EDIT:").split(':', limit = 3)
        val rest = parts.getOrNull(2)
        // Ключ до плана 20: сразу `back`, всегда список операций. `back` с двоеточиями идёт последним.
        val (prefill, back) = when {
            rest == null -> null to null
            rest.startsWith(KEY_TRANSACTIONS) -> null to rest
            else -> rest.split(':', limit = 2).let { (p, b) -> p.takeIf { it.isNotEmpty() } to b }
        }
        AppScreen.TransactionEdit(
            id = parts[0].takeIf { it.isNotEmpty() }?.let(UUID::fromString),
            draft = UUID.fromString(parts[1]),
            back = back?.let(::restoreScreen) ?: AppScreen.Transactions(),
            prefill = prefill?.let(::restorePrefill),
        )
    }

    key.startsWith("$KEY_BUDGET_EDIT:") -> {
        val (target, draft) = key.removePrefix("$KEY_BUDGET_EDIT:").split(':')
        AppScreen.BudgetEdit(
            id = target.takeIf { it.isNotEmpty() }?.let(UUID::fromString),
            draft = UUID.fromString(draft),
        )
    }

    key == KEY_NET_WORTH -> AppScreen.NetWorth

    key.startsWith("$KEY_HOLDING_EDIT:") -> {
        val (target, draft, side) = key.removePrefix("$KEY_HOLDING_EDIT:").split(':')
        AppScreen.HoldingEdit(
            id = target.takeIf { it.isNotEmpty() }?.let(UUID::fromString),
            draft = UUID.fromString(draft),
            side = requireNotNull(HoldingSide.decode(side)),
        )
    }

    key.startsWith("$KEY_HOLDING_HISTORY:") ->
        AppScreen.HoldingHistory(UUID.fromString(key.removePrefix("$KEY_HOLDING_HISTORY:")))

    key.startsWith("$KEY_SETTINGS:") -> {
        val (page, target, visit) = key.removePrefix("$KEY_SETTINGS:").split(':')
        restoreSettingsPage(page, target, visit)?.let(AppScreen::Settings) ?: AppScreen.Loading
    }

    key.startsWith("$KEY_RECOGNIZE:") -> AppScreen.Recognize(UUID.fromString(key.removePrefix("$KEY_RECOGNIZE:")))

    key.startsWith("$KEY_RECONCILIATION:") ->
        AppScreen.Reconciliation(YearMonth.parse(key.removePrefix("$KEY_RECONCILIATION:")))

    else -> null
}

// Описание кодируется: в нём может оказаться двоеточие, разделитель ключа.
private fun TransactionPrefill.saveKey(): String = listOf(
    amountMinor.toString(),
    type.name,
    date.toString(),
    URLEncoder.encode(description, "UTF-8"),
).joinToString(",")

private fun restorePrefill(value: String): TransactionPrefill {
    val (amount, type, date, description) = value.split(',')
    return TransactionPrefill(
        amountMinor = amount.toLong(),
        type = TransactionType.valueOf(type),
        date = LocalDate.parse(date),
        description = URLDecoder.decode(description, "UTF-8"),
    )
}

private fun AppScreen.Transactions.saveKey(): String {
    val f = filters ?: return KEY_TRANSACTIONS
    return listOf(
        f.period.name,
        f.month?.toString().orEmpty(),
        f.type?.name.orEmpty(),
        f.categoryIds.joinToString(","),
        f.accountId?.toString().orEmpty(),
        f.unassigned.toString(),
        f.from?.toString().orEmpty(),
        f.to?.toString().orEmpty(),
        reconciliation?.toString().orEmpty(),
        overview?.saveKey().orEmpty(),
    ).joinToString(":", prefix = "$KEY_TRANSACTIONS:")
}

private fun restoreTransactions(value: String): AppScreen.Transactions {
    val parts = value.split(':')
    require(parts.size == TRANSACTIONS_KEY_FIELDS)
    return AppScreen.Transactions(
        filters = TransactionFilters(
            period = TransactionPeriod.valueOf(parts[0]),
            month = parts[1].takeIf { it.isNotEmpty() }?.let(YearMonth::parse),
            type = parts[2].takeIf { it.isNotEmpty() }?.let(TransactionType::valueOf),
            categoryIds = parts[3].split(',').filter { it.isNotEmpty() }.mapTo(LinkedHashSet(), UUID::fromString),
            accountId = parts[4].takeIf { it.isNotEmpty() }?.let(UUID::fromString),
            unassigned = parts[5].toBooleanStrict(),
            from = parts[6].takeIf { it.isNotEmpty() }?.let(LocalDate::parse),
            to = parts[7].takeIf { it.isNotEmpty() }?.let(LocalDate::parse),
        ),
        reconciliation = parts[8].takeIf { it.isNotEmpty() }?.let(YearMonth::parse),
        overview = parts[9].takeIf { it.isNotEmpty() }?.let(OverviewPeriod::fromSaveKey),
    )
}
