package tech.shatrov.familyfinances.ui.overview

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.CategoryShare
import tech.shatrov.familyfinances.core.api.MonthTotals
import tech.shatrov.familyfinances.core.api.PeriodTotals
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId

sealed interface MonthlyState {
    data object Loading : MonthlyState

    data class Failure(val error: UiError) : MonthlyState

    data class Ready(val months: List<MonthTotals>) : MonthlyState
}

sealed interface SummaryState {
    data object Loading : SummaryState

    data class Failure(val error: UiError) : SummaryState

    /** Дельты — доли 0…1 к отрезку той же длины перед периодом; `null`, когда там не было операций. */
    data class Ready(
        val totals: PeriodTotals,
        val incomeDelta: Double?,
        val expensesDelta: Double?,
        val expenseCategories: List<CategoryShare>,
        val incomeCategories: List<CategoryShare>,
    ) : SummaryState {
        val isEmpty: Boolean get() = totals.transactionCount == 0
    }
}

/** [range] — те же даты, что ушли в `summary`: расшифровка категории берёт их, а не считает заново. */
data class OverviewUiState(
    val period: OverviewPeriod,
    val range: DateRange,
    val currency: String,
    val monthly: MonthlyState = MonthlyState.Loading,
    val summary: SummaryState = SummaryState.Loading,
)

/**
 * «Обзор»: ряд `monthly` за 12 месяцев и `summary` выбранного периода. Ряд не зависит от периода и
 * запрашивается без границ — их считает сервер, и сумма столбиков сходится с чипом «Год».
 */
class OverviewViewModel(
    private val api: ApiGraph,
    currency: String,
    initial: OverviewPeriod = OverviewPeriod.ThisMonth,
    zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private var requestedOn: LocalDate = today()

    private val mutable = MutableStateFlow(OverviewUiState(initial, initial.bounds(requestedOn), currency))

    val state: StateFlow<OverviewUiState> = mutable.asStateFlow()

    private var monthlyJob: Job? = null
    private var summaryJob: Job? = null

    init {
        loadMonthly()
        loadSummary()
    }

    /** Возврат из фона: «сегодня» семьи сдвигает и ряд, и границы периода. */
    fun revalidate() {
        if (requestedOn != today()) refresh()
    }

    fun refresh() {
        requestedOn = today()
        mutable.update { it.copy(range = it.period.bounds(requestedOn)) }
        loadMonthly()
        loadSummary()
    }

    fun select(period: OverviewPeriod) {
        if (period == mutable.value.period) return
        mutable.update { it.copy(period = period) }
        if (requestedOn != today()) {
            refresh()
            return
        }
        mutable.update { it.copy(range = period.bounds(requestedOn)) }
        loadSummary()
    }

    fun retryMonthly() = loadMonthly()

    fun retrySummary() = loadSummary()

    private fun loadMonthly() {
        monthlyJob?.cancel()
        mutable.update { it.copy(monthly = MonthlyState.Loading) }
        monthlyJob = viewModelScope.launch {
            val monthly = try {
                MonthlyState.Ready(api.client.unwrap { api.stats.getStatsMonthly() }.`data`.months)
            } catch (failure: ApiFailure) {
                MonthlyState.Failure(failure.toUiError())
            }
            mutable.update { it.copy(monthly = monthly) }
        }
    }

    /** Прежний запрос отменяется: при быстрой смене чипов его ответ пришёл бы поверх последнего. */
    private fun loadSummary() {
        summaryJob?.cancel()
        val range = mutable.value.range
        mutable.update { it.copy(summary = SummaryState.Loading) }
        summaryJob = viewModelScope.launch {
            val summary = try {
                api.client.unwrap { api.stats.getStatsSummary(range.from, range.to) }.`data`.toState()
            } catch (failure: ApiFailure) {
                SummaryState.Failure(failure.toUiError())
            }
            mutable.update { it.copy(summary = summary) }
        }
    }
}

private fun StatsSummary.toState() = SummaryState.Ready(
    totals = current,
    incomeDelta = incomeDelta.takeIf { hasPreviousData },
    expensesDelta = expensesDelta.takeIf { hasPreviousData },
    expenseCategories = expenseCategories.sortedByDescending { it.amountMinor },
    incomeCategories = incomeCategories.sortedByDescending { it.amountMinor },
)
