package tech.shatrov.familyfinances.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId

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
    zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<HomeUiState>(HomeUiState.Loading)

    val state: StateFlow<HomeUiState> = mutable.asStateFlow()

    private var requestedOn: LocalDate? = null
    private var job: Job? = null

    init {
        refresh()
    }

    /**
     * Заход на экран и возврат из фона: сводка посчитана по «сегодня» семьи, и после полуночи
     * она уже за прошлый день, а на границе месяца — за прошлый месяц.
     * Идущий запрос сверяется по своей дате, а не по показанной сводке: полночь под ним иначе
     * осталась бы незамеченной, его ответ встал бы на экран за прошлые сутки, а второго
     * колбэка жизненного цикла не будет.
     */
    fun revalidate() {
        val ready = mutable.value as? HomeUiState.Ready
        val current = if (job?.isActive == true) requestedOn else ready?.summary?.to
        if (current == null || current == today()) return
        refresh()
    }

    fun refresh() {
        job?.cancel()
        requestedOn = today()
        mutable.value = HomeUiState.Loading
        job = viewModelScope.launch {
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
