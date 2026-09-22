package tech.shatrov.familyfinances.ui.transactions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.Session
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.util.UUID

/** Страница списка: `limit` сервера — 200, но экран пролистывается, а не грузится целиком. */
private const val PAGE_SIZE = 50

/** Справочники берём одной страницей: категорий, счетов и пользователей у семьи считанные единицы. */
private const val REFERENCE_LIMIT = 200

/** Строка списка: имена подставлены здесь, чтобы разметка не искала их по словарям. */
data class TransactionRow(
    val transaction: Transaction,
    val categoryName: String?,
    val authorName: String?,
    val isMine: Boolean,
)

/** Границы периода, по которым посчитан загруженный список: обе даты берутся из одного `today()`. */
private data class DateWindow(
    val from: LocalDate?,
    val to: LocalDate?,
)

data class DayGroup(
    val date: LocalDate,
    val rows: List<TransactionRow>,
)

sealed interface TransactionsUiState {
    data object Loading : TransactionsUiState

    data class Failure(val error: UiError) : TransactionsUiState

    data class Ready(
        val groups: List<DayGroup>,
        val currency: String,
        val hasMore: Boolean,
        val loadingMore: Boolean = false,
        val moreError: UiError? = null,
    ) : TransactionsUiState {
        val isEmpty: Boolean get() = groups.isEmpty()
    }
}

/**
 * Список транзакций: фильтры уходят в query, следующая страница — по `meta.pagination`.
 * Отказ на первой странице заменяет экран, отказ на догрузке оставляет загруженное на месте.
 */
class TransactionsViewModel(
    private val api: ApiGraph,
    private val session: Session,
    // Дата берётся на каждый запрос, а не один раз: модель живёт всю сессию приложения, и
    // после полуночи «этот месяц» иначе остался бы прошлым.
    private val today: () -> LocalDate = { LocalDate.now(session.zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<TransactionsUiState>(TransactionsUiState.Loading)

    val state: StateFlow<TransactionsUiState> = mutable.asStateFlow()

    // Фильтры и справочник категорий — отдельно от состояния списка: ряды фильтров рисуются и
    // на загрузке, и на отказе, а в sealed-состояние с `data object Loading` их не втащить.
    private val mutableFilters = MutableStateFlow(TransactionFilters())

    val filters: StateFlow<TransactionFilters> = mutableFilters.asStateFlow()

    private val mutableCategories = MutableStateFlow(emptyList<Category>())

    val categories: StateFlow<List<Category>> = mutableCategories.asStateFlow()

    private val mutableAccounts = MutableStateFlow(emptyList<Account>())

    /** С архивными: выписку по перевыпущенной карте тоже бывает нужно найти. */
    val accounts: StateFlow<List<Account>> = mutableAccounts.asStateFlow()

    private var loaded = emptyList<Transaction>()
    private var loadedWindow: DateWindow? = null
    private var pendingWindow: DateWindow? = null
    private var total = 0
    private var exhausted = false
    private var authors = emptyMap<UUID, String>()
    private var referencesLoaded = false
    private var job: Job? = null

    init {
        refresh()
    }

    /** Возврат на экран и повтор после отказа: справочники перечитываются вместе со списком. */
    fun refresh() {
        mutable.value = TransactionsUiState.Loading
        load(fromStart = true, reloadReferences = true)
    }

    fun onFiltersChange(next: TransactionFilters) {
        mutableFilters.value = next
        mutable.value = TransactionsUiState.Loading
        load(fromStart = true, reloadReferences = false)
    }

    /** Фильтр, с которым экран открыли снаружи; тот же самый повторно список не перечитывает. */
    fun applyFilters(next: TransactionFilters) {
        if (mutableFilters.value != next) onFiltersChange(next)
    }

    /**
     * Заход на экран и возврат из фона. Дата пересчитывается только в запросе, а кончившиеся
     * страницы запросов больше не делают: без этой проверки «этот месяц» остался бы прошлым.
     * Идущий запрос сверяется по своему окну, а не по загруженному: полночь под ним иначе
     * осталась бы незамеченной, и его ответ встал бы на экран как «этот месяц».
     */
    fun revalidate() {
        val current = (if (job?.isActive == true) pendingWindow else loadedWindow) ?: return
        if (window(today()) == current) return
        mutable.value = TransactionsUiState.Loading
        load(fromStart = true)
    }

    /** Догрузка следующей страницы; вызовы во время запроса и после отказа игнорируются. */
    fun loadMore() {
        val ready = mutable.value as? TransactionsUiState.Ready ?: return
        if (!ready.hasMore || job?.isActive == true) return
        mutable.value = ready.copy(loadingMore = true, moreError = null)
        load(fromStart = false)
    }

    // Загрузка в один поток: смена фильтра во время догрузки иначе дописала бы к новому
    // списку страницу прошлого запроса, и в нём оказались бы две строки с одним id — на таком
    // ключе LazyColumn падает.
    private fun load(
        fromStart: Boolean,
        reloadReferences: Boolean = false,
    ) {
        job?.cancel()
        val bounds = window(today())
        pendingWindow = bounds
        job = viewModelScope.launch {
            // Полночь между страницами сдвигает границы «этого месяца», а offset посчитан
            // по прошлому окну: такую догрузку начинаем с нуля, иначе страницы двух
            // периодов слипаются, а начало нового теряется. Окно считается до запроса: по
            // отказу список прошлого месяца нельзя оставить на экране как «этот».
            val start = fromStart || bounds != loadedWindow
            try {
                // Категория, заведённая на соседнем экране, иначе не попала бы ни в строку,
                // ни в фильтр до перезапуска процесса: модель живёт всю сессию.
                if (reloadReferences || !referencesLoaded) {
                    loadReferences()
                }
                val page = api.client.unwrap { requestPage(bounds, offset = if (start) 0 else loaded.size) }
                // distinctBy: сосед мог вставить запись между страницами и сдвинуть окно.
                val merged = if (start) page.`data` else (loaded + page.`data`).distinctBy { it.id }
                // Страница, не добавившая ни строки, — конец списка, даже если total больше:
                // выброшенный дубликат навсегда оставил бы loaded.size меньше total, а подвал
                // перезапускается только по изменению числа строк — экран завис бы на спиннере.
                exhausted = !start && merged.size == loaded.size
                loaded = merged
                loadedWindow = bounds
                total = page.meta.pagination.total
                mutable.value = ready()
            } catch (failure: ApiFailure) {
                mutable.value = failed(start, failure.toUiError())
            }
        }
    }

    private suspend fun requestPage(
        bounds: DateWindow,
        offset: Int,
    ) = api.transactions.listTransactions(
        limit = PAGE_SIZE,
        offset = offset,
        categoryId = categoryQuery(),
        accountId = mutableFilters.value.accountId,
        unassigned = mutableFilters.value.unassigned.takeIf { it },
        type = mutableFilters.value.type,
        dateFrom = bounds.from,
        dateTo = bounds.to,
    )

    // Порядок справочника, а не выбора: одинаковый набор — один и тот же запрос. Удалённые — в конце.
    private fun categoryQuery(): List<UUID>? {
        val ids = mutableFilters.value.categoryIds.takeIf { it.isNotEmpty() } ?: return null
        val order = mutableCategories.value.withIndex().associate { (index, category) -> category.id to index }
        return ids.sortedBy { order[it] ?: Int.MAX_VALUE }
    }

    private fun window(day: LocalDate): DateWindow {
        val current = mutableFilters.value
        return DateWindow(current.dateFrom(day), current.dateTo(day))
    }

    // `/users` открыт только админу, поэтому у member список авторов остаётся пустым:
    // своя запись всё равно подписана «Вы», а чужая в семье из двух человек однозначна.
    private suspend fun loadReferences() {
        // Флаг снимается до запросов и ставится после обоих: смена фильтра между ними отменяет
        // корутину, а по непустым категориям загрузка авторов иначе считалась бы сделанной.
        referencesLoaded = false
        mutableCategories.value =
            api.client.unwrap { api.categories.listCategories(limit = REFERENCE_LIMIT) }.`data`
        mutableAccounts.value =
            api.client.unwrap { api.accounts.listAccounts(limit = REFERENCE_LIMIT, archived = true) }.`data`
        if (session.isAdmin) {
            authors = api.client
                .unwrap { api.users.listUsers(limit = REFERENCE_LIMIT) }
                .`data`
                .associate { it.id to it.firstName }
        }
        referencesLoaded = true
    }

    private fun ready(): TransactionsUiState.Ready {
        val names = mutableCategories.value.associate { it.id to it.name }
        return TransactionsUiState.Ready(
            groups = loaded
                .groupBy { it.date }
                .map { (date, transactions) ->
                    DayGroup(date, transactions.map { row(it, names) })
                },
            currency = session.currency,
            hasMore = !exhausted && loaded.size < total,
        )
    }

    private fun row(
        transaction: Transaction,
        names: Map<UUID, String>,
    ): TransactionRow = TransactionRow(
        transaction = transaction,
        categoryName = names[transaction.categoryId],
        authorName = authors[transaction.userId],
        isMine = transaction.userId == session.user.id,
    )

    private fun failed(
        fromStart: Boolean,
        error: UiError,
    ): TransactionsUiState {
        val current = mutable.value
        return if (fromStart || current !is TransactionsUiState.Ready) {
            TransactionsUiState.Failure(error)
        } else {
            current.copy(loadingMore = false, moreError = error)
        }
    }
}
