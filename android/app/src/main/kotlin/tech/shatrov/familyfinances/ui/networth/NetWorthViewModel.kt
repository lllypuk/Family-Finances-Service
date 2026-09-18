package tech.shatrov.familyfinances.ui.networth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Holding
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.HoldingValueRequest
import tech.shatrov.familyfinances.core.api.NetWorthMonth
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId
import java.util.UUID

/** Строка списка: вид уже разобран, незнакомый стал [HoldingKind.OTHER]. */
data class HoldingRow(
    val holding: Holding,
    val kind: HoldingKind,
)

sealed interface NetWorthUiState {
    data object Loading : NetWorthUiState

    data class Failure(val error: UiError) : NetWorthUiState

    /**
     * [latest] — последняя корзина ряда по умолчанию (`to` = сегодня): итог берётся из неё, а не
     * суммой строк, потому что архивные позиции в капитале остаются.
     */
    data class Ready(
        val assets: List<HoldingRow>,
        val liabilities: List<HoldingRow>,
        val archived: List<HoldingRow>,
        val latest: NetWorthMonth?,
    ) : NetWorthUiState {
        val isEmpty: Boolean get() = assets.isEmpty() && liabilities.isEmpty() && archived.isEmpty()
    }
}

/** Лист снимка позиции [holdingId]: сумма и дата, дата не позже [today] семьи. */
data class ValueEditUiState(
    val holdingId: UUID,
    val name: String,
    val today: LocalDate,
    val amount: String = "",
    val date: LocalDate = today,
    val submitting: Boolean = false,
    val error: UiError? = null,
) {
    val amountMinor: Long? get() = parseAmountMinor(amount)

    val canSubmit: Boolean get() = amountMinor != null && !submitting
}

/** Капитал: позиции с архивом одной страницей и ряд `net-worth` для шапки; снимок — `PUT` по дате. */
class NetWorthViewModel(
    private val api: ApiGraph,
    zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<NetWorthUiState>(NetWorthUiState.Loading)
    private val mutableEditor = MutableStateFlow<ValueEditUiState?>(null)

    val state: StateFlow<NetWorthUiState> = mutable.asStateFlow()

    /** `null` — лист снимка закрыт. */
    val editor: StateFlow<ValueEditUiState?> = mutableEditor.asStateFlow()

    private var requestedOn: LocalDate? = null
    private var job: Job? = null

    init {
        refresh()
    }

    /** Возврат из фона: `current` и последняя корзина отсечены по «сегодня», которое могло смениться. */
    fun revalidate() {
        val current = requestedOn ?: return
        if (current == today()) return
        refresh()
    }

    fun refresh() {
        job?.cancel()
        requestedOn = today()
        mutable.value = NetWorthUiState.Loading
        job = viewModelScope.launch {
            mutable.value = try {
                val holdings = api.client
                    .unwrap { api.holdings.listHoldings(limit = HOLDING_LIMIT, archived = true) }
                    .`data`
                val series = api.client.unwrap { api.stats.getNetWorthStats() }.`data`
                val rows = holdings.map { HoldingRow(it, HoldingKind.of(it.kind)) }
                val (archived, active) = rows.partition { it.holding.isArchived }
                NetWorthUiState.Ready(
                    assets = active.filter { it.holding.side == HoldingSide.asset },
                    liabilities = active.filter { it.holding.side == HoldingSide.liability },
                    archived = archived,
                    latest = series.months.lastOrNull(),
                )
            } catch (failure: ApiFailure) {
                NetWorthUiState.Failure(failure.toUiError())
            }
        }
    }

    /** Сумма подставляется из `current`: ежемесячный снимок чаще правка прошлого, чем новое число. */
    fun onOpenValue(holding: Holding) {
        mutableEditor.value = ValueEditUiState(
            holdingId = holding.id,
            name = holding.name,
            today = today(),
            amount = holding.current?.valueMinor?.let(::formatAmountInput).orEmpty(),
        )
    }

    fun onAmountChange(amount: String) {
        mutableEditor.update { it?.copy(amount = amount, error = null) }
    }

    fun onDateChange(date: LocalDate) {
        mutableEditor.update { it?.copy(date = minOf(date, it.today), error = null) }
    }

    fun onDismissValue() {
        if (mutableEditor.value?.submitting == true) return
        mutableEditor.value = null
    }

    fun onSaveValue() {
        val current = mutableEditor.value ?: return
        val amount = current.amountMinor?.takeIf { !current.submitting } ?: return
        mutableEditor.value = current.copy(submitting = true, error = null)
        viewModelScope.launch {
            try {
                api.client.unwrap {
                    api.holdings.putHoldingValue(current.holdingId, current.date, HoldingValueRequest(amount))
                }
                mutableEditor.value = null
                refresh()
            } catch (failure: ApiFailure) {
                // Позицию удалили с другого телефона: листу нечего сохранять, устарел список.
                if ((failure as? ApiFailure.Api)?.isNotFound == true) {
                    mutableEditor.value = null
                    refresh()
                } else {
                    mutableEditor.update { it?.copy(submitting = false, error = failure.toUiError()) }
                }
            }
        }
    }
}
