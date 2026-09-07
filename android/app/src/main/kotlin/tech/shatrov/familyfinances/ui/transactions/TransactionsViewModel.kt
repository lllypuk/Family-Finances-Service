package tech.shatrov.familyfinances.ui.transactions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.Session
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

/** Справочники берём одной страницей: категорий и пользователей у семьи считанные единицы. */
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
        val filters: TransactionFilters,
        val categories: List<Category>,
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

    private var filters = TransactionFilters()
    private var loaded = emptyList<Transaction>()
    private var loadedWindow: DateWindow? = null
    private var pendingWindow: DateWindow? = null
    private var total = 0
    private var exhausted = false
    private var categories = emptyList<Category>()
    private var authors = emptyMap<UUID, String>()
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
        filters = next
        mutable.value = TransactionsUiState.Loading
        load(fromStart = true, reloadReferences = false)
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
                if (reloadReferences || categories.isEmpty()) {
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
        categoryId = filters.categoryId,
        type = filters.type,
        dateFrom = bounds.from,
        dateTo = bounds.to,
    )

    private fun window(day: LocalDate) = DateWindow(filters.dateFrom(day), filters.dateTo(day))

    // `/users` открыт только админу, поэтому у member список авторов остаётся пустым:
    // своя запись всё равно подписана «Вы», а чужая в семье из двух человек однозначна.
    private suspend fun loadReferences() {
        categories = api.client.unwrap { api.categories.listCategories(limit = REFERENCE_LIMIT) }.`data`
        if (session.isAdmin) {
            authors = api.client
                .unwrap { api.users.listUsers(limit = REFERENCE_LIMIT) }
                .`data`
                .associate { it.id to it.firstName }
        }
    }

    private fun ready(): TransactionsUiState.Ready {
        val names = categories.associate { it.id to it.name }
        return TransactionsUiState.Ready(
            groups = loaded
                .groupBy { it.date }
                .map { (date, transactions) ->
                    DayGroup(date, transactions.map { row(it, names) })
                },
            currency = session.currency,
            filters = filters,
            categories = categories,
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
