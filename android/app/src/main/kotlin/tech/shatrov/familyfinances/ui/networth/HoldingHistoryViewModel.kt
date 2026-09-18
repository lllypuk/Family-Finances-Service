package tech.shatrov.familyfinances.ui.networth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Holding
import tech.shatrov.familyfinances.core.api.HoldingValue
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId
import java.util.UUID

private const val HISTORY_PAGE_SIZE = 50

sealed interface HoldingHistoryUiState {
    data object Loading : HoldingHistoryUiState

    data class Failure(val error: UiError) : HoldingHistoryUiState

    /** Позицию удалили с другого телефона: истории нет, экран уходит. */
    data object Gone : HoldingHistoryUiState

    data class Ready(
        val holding: Holding,
        val values: List<HoldingValue>,
        val hasMore: Boolean,
        val loadingMore: Boolean = false,
        val moreError: UiError? = null,
    ) : HoldingHistoryUiState
}

/** История снимков позиции, новые сверху; правка и удаление перечитывают и позицию — у неё сменился `current`. */
class HoldingHistoryViewModel(
    private val api: ApiGraph,
    private val holdingId: UUID,
    zone: ZoneId = ZoneId.systemDefault(),
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<HoldingHistoryUiState>(HoldingHistoryUiState.Loading)
    private val values = ValueEditor(api, viewModelScope, ::refresh)

    val state: StateFlow<HoldingHistoryUiState> = mutable.asStateFlow()

    /** `null` — лист снимка закрыт. */
    val editor: StateFlow<ValueEditUiState?> = values.state

    private var holding: Holding? = null
    private var loaded = emptyList<HoldingValue>()
    private var total = 0
    private var exhausted = false
    private var job: Job? = null

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = HoldingHistoryUiState.Loading
        load(fromStart = true)
    }

    /** Догрузка следующей страницы; вызовы во время запроса и после конца списка игнорируются. */
    fun loadMore() {
        val ready = mutable.value as? HoldingHistoryUiState.Ready ?: return
        if (!ready.hasMore || job?.isActive == true) return
        mutable.value = ready.copy(loadingMore = true, moreError = null)
        load(fromStart = false)
    }

    fun onOpenValue(value: HoldingValue) {
        val current = holding ?: return
        values.open(
            ValueEditUiState(
                holdingId = holdingId,
                name = current.name,
                today = today(),
                amount = formatAmountInput(value.valueMinor),
                date = value.date,
                existing = true,
            ),
        )
    }

    fun onAmountChange(amount: String) = values.onAmountChange(amount)

    fun onDateChange(date: LocalDate) = values.onDateChange(date)

    fun onDismissValue() = values.onDismiss()

    fun onSaveValue() = values.onSave()

    fun onDeleteValue() = values.onDelete()

    // Загрузка в один поток: перечитывание после правки во время догрузки иначе дописало бы
    // страницу прошлого списка к новому.
    private fun load(fromStart: Boolean) {
        job?.cancel()
        job = viewModelScope.launch {
            try {
                if (fromStart) {
                    val found = api.client
                        .unwrap { api.holdings.listHoldings(limit = HOLDING_LIMIT, archived = true) }
                        .`data`
                        .firstOrNull { it.id == holdingId }
                    if (found == null) {
                        mutable.value = HoldingHistoryUiState.Gone
                        return@launch
                    }
                    holding = found
                }
                val page = api.client.unwrap {
                    api.holdings.listHoldingValues(
                        holdingId,
                        limit = HISTORY_PAGE_SIZE,
                        offset = if (fromStart) 0 else loaded.size,
                    )
                }
                // distinctBy: снимок, записанный между страницами, сдвигает окно.
                val merged = if (fromStart) page.`data` else (loaded + page.`data`).distinctBy { it.date }
                // Страница без новых строк — конец списка, иначе подвал крутился бы вечно.
                exhausted = !fromStart && merged.size == loaded.size
                loaded = merged
                total = page.meta.pagination.total
                mutable.value = HoldingHistoryUiState.Ready(
                    holding = requireNotNull(holding),
                    values = loaded,
                    hasMore = !exhausted && loaded.size < total,
                )
            } catch (failure: ApiFailure) {
                val ready = mutable.value as? HoldingHistoryUiState.Ready
                mutable.value = if (fromStart || ready == null) {
                    HoldingHistoryUiState.Failure(failure.toUiError())
                } else {
                    ready.copy(loadingMore = false, moreError = failure.toUiError())
                }
            }
        }
    }
}
