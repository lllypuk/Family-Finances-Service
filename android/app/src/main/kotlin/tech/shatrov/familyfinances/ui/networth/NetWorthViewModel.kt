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
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.NetWorthMonth
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId

/** Строка списка: вид уже разобран, незнакомый стал [HoldingKind.OTHER]. */
data class HoldingRow(
    val holding: Holding,
    val kind: HoldingKind,
)

sealed interface NetWorthUiState {
    data object Loading : NetWorthUiState

    data class Failure(val error: UiError) : NetWorthUiState

    /**
     * [months] — ряд по умолчанию (`to` = сегодня), [latest] — его последняя корзина: итог берётся из неё,
     * а не суммой строк, потому что архивные позиции в капитале остаются.
     */
    data class Ready(
        val assets: List<HoldingRow>,
        val liabilities: List<HoldingRow>,
        val archived: List<HoldingRow>,
        val months: List<NetWorthMonth>,
    ) : NetWorthUiState {
        val latest: NetWorthMonth? get() = months.lastOrNull()

        val isEmpty: Boolean get() = assets.isEmpty() && liabilities.isEmpty() && archived.isEmpty()
    }
}

/** Капитал: позиции с архивом одной страницей и ряд `net-worth` для шапки; снимок — `PUT` по дате. */
class NetWorthViewModel(
    private val api: ApiGraph,
    zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<NetWorthUiState>(NetWorthUiState.Loading)
    private val values = ValueEditor(api, viewModelScope, ::refresh)

    val state: StateFlow<NetWorthUiState> = mutable.asStateFlow()

    /** `null` — лист снимка закрыт. */
    val editor: StateFlow<ValueEditUiState?> = values.state

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
                    months = series.months,
                )
            } catch (failure: ApiFailure) {
                NetWorthUiState.Failure(failure.toUiError())
            }
        }
    }

    /** Сумма подставляется из `current`: ежемесячный снимок чаще правка прошлого, чем новое число. */
    fun onOpenValue(holding: Holding) {
        values.open(
            ValueEditUiState(
                holdingId = holding.id,
                name = holding.name,
                today = today(),
                amount = holding.current?.valueMinor?.let(::formatAmountInput).orEmpty(),
            ),
        )
    }

    fun onAmountChange(amount: String) = values.onAmountChange(amount)

    fun onDateChange(date: LocalDate) = values.onDateChange(date)

    fun onDismissValue() = values.onDismiss()

    fun onSaveValue() = values.onSave()
}
