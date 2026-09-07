package tech.shatrov.familyfinances.ui.transactions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
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
    private var inFlight = false

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = TransactionsUiState.Loading
        load(fromStart = true)
    }

    fun onFiltersChange(next: TransactionFilters) {
        filters = next
        refresh()
    }

    /** Догрузка следующей страницы; вызовы во время запроса и после отказа игнорируются. */
    fun loadMore() {
        val ready = mutable.value as? TransactionsUiState.Ready ?: return
        if (!ready.hasMore || inFlight) return
        mutable.value = ready.copy(loadingMore = true, moreError = null)
        load(fromStart = false)
    }

    /** Повтор после отказа на догрузке: состояние отказа снимается, иначе `loadMore` встанет. */
    fun retryMore() {
        val ready = mutable.value as? TransactionsUiState.Ready ?: return
        mutable.value = ready.copy(moreError = null)
        loadMore()
    }

    private fun load(fromStart: Boolean) {
        inFlight = true
        viewModelScope.launch {
            try {
                if (categories.isEmpty()) {
                    loadReferences()
                }
                val page = api.client.unwrap({ requestPage(fromStart) }, { it })
                loaded = if (fromStart) page.`data` else loaded + page.`data`
                total = page.meta.pagination.total
                mutable.value = ready()
            } catch (failure: ApiFailure) {
                mutable.value = failed(fromStart, failure.toUiError())
            } finally {
                inFlight = false
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
        categories = api.client.unwrap({ api.categories.listCategories(limit = REFERENCE_LIMIT) }, { it.`data` })
        if (session.isAdmin) {
            authors = api.client
                .unwrap({ api.users.listUsers(limit = REFERENCE_LIMIT) }, { it.`data` })
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
