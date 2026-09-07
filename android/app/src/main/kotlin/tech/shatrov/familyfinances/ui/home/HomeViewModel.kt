package tech.shatrov.familyfinances.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

sealed interface HomeUiState {
    data object Loading : HomeUiState

    data class Failure(val error: UiError) : HomeUiState

    data class Ready(
        val summary: StatsSummary,
        val currency: String,
    ) : HomeUiState {
        /** Ни одной операции за всё время: карточки с нулями рисовать незачем. */
        val isEmpty: Boolean get() = summary.transactionsTotal == 0
    }
}

/** Главная — один `GET /stats/summary`: складывать несколько ответов на клиенте не приходится. */
class HomeViewModel(
    private val api: ApiGraph,
    private val currency: String,
) : ViewModel() {
    private val mutable = MutableStateFlow<HomeUiState>(HomeUiState.Loading)

    val state: StateFlow<HomeUiState> = mutable.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = HomeUiState.Loading
        viewModelScope.launch {
            mutable.value = try {
                // Границы периода не задаём: текущий месяц сервер считает в часовом поясе семьи
                // (A-06), а телефон может стоять в другом.
                val summary = api.client.unwrap { api.stats.getStatsSummary() }.`data`
                HomeUiState.Ready(summary, currency)
            } catch (failure: ApiFailure) {
                HomeUiState.Failure(failure.toUiError())
            }
        }
    }
}
