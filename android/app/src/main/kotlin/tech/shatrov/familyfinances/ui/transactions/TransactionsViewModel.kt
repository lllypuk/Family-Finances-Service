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
    private val today: LocalDate = LocalDate.now(),
) : ViewModel() {
    private val mutable = MutableStateFlow<TransactionsUiState>(TransactionsUiState.Loading)

    val state: StateFlow<TransactionsUiState> = mutable.asStateFlow()

    private var filters = TransactionFilters()
    private var loaded = emptyList<Transaction>()
    private var total = 0
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
        job = viewModelScope.launch {
            try {
                // Категория, заведённая на соседнем экране, иначе не попала бы ни в строку,
                // ни в фильтр до перезапуска процесса: модель живёт всю сессию.
                if (reloadReferences || categories.isEmpty()) {
                    loadReferences()
                }
                val page = api.client.unwrap { requestPage(fromStart) }
                // distinctBy: сосед мог вставить запись между страницами и сдвинуть окно.
                loaded = if (fromStart) page.`data` else (loaded + page.`data`).distinctBy { it.id }
                total = page.meta.pagination.total
                mutable.value = ready()
            } catch (failure: ApiFailure) {
                mutable.value = failed(fromStart, failure.toUiError())
            }
        }
    }

    private suspend fun requestPage(fromStart: Boolean) = api.transactions.listTransactions(
        limit = PAGE_SIZE,
        offset = if (fromStart) 0 else loaded.size,
        categoryId = filters.categoryId,
        type = filters.type,
        dateFrom = filters.dateFrom(today),
        dateTo = filters.dateTo(today),
    )

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
            hasMore = loaded.size < total,
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
