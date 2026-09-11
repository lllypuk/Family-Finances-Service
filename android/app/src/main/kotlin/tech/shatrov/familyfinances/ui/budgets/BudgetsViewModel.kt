package tech.shatrov.familyfinances.ui.budgets

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Budget
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.time.ZoneId
import java.util.UUID

/** Бюджеты и категории берутся одной страницей: тех и других у семьи считанные единицы. */
private const val BUDGET_LIMIT = 200

/** Порог «на исходе» — тот же, что у серверного `BudgetAlertNearLimit`, в процентах. */
private const val NEAR_LIMIT = 80.0

/** Перерасход — с включённой границей: сводка красит ту же запись по `utilization >= 100`. */
private const val FULL = 100.0

/** Строка списка по сырому `utilization`: у `Budget` флагов `is_*` нет, в отличие от сводки. */
enum class BudgetLevel { OK, NEAR, OVER }

enum class BudgetFilter { TODAY, ALL }

/**
 * Строка списка: имя категории подставлено здесь, чтобы разметка не искала его по словарю.
 * `categoryName == null` при непустом `budget.categoryId` — категории нет в справочнике
 * (удалена), и выдавать её за бюджет на все категории нельзя.
 */
data class BudgetRow(
    val budget: Budget,
    val categoryName: String?,
    val level: BudgetLevel,
) {
    val allCategories: Boolean get() = budget.categoryId == null
}

sealed interface BudgetsUiState {
    data object Loading : BudgetsUiState

    data class Failure(val error: UiError) : BudgetsUiState

    data class Ready(val rows: List<BudgetRow>) : BudgetsUiState {
        val isEmpty: Boolean get() = rows.isEmpty()
    }
}

/**
 * Бюджеты: одна страница списка с фильтром «на сегодня» / «все периоды» и именами категорий.
 * Отказ любого из двух запросов заменяет экран: строка без имени категории неотличима от общей.
 */
class BudgetsViewModel(
    private val api: ApiGraph,
    zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
) : ViewModel() {
    private val mutable = MutableStateFlow<BudgetsUiState>(BudgetsUiState.Loading)

    val state: StateFlow<BudgetsUiState> = mutable.asStateFlow()

    // Фильтр отдельно от состояния списка: чипы рисуются и на загрузке, и на отказе — иначе
    // упавший запрос «на сегодня» не переключить, и «Повторить» повторяет ровно его.
    private val mutableFilter = MutableStateFlow(BudgetFilter.TODAY)

    val filter: StateFlow<BudgetFilter> = mutableFilter.asStateFlow()

    private var requestedOn: LocalDate? = null
    private var job: Job? = null

    init {
        refresh()
    }

    fun onFilterChange(next: BudgetFilter) {
        if (next == mutableFilter.value) return
        mutableFilter.value = next
        refresh()
    }

    /**
     * Заход на экран и возврат из фона: «сегодня» для `active_only` считает сервер, и ответ,
     * полученный вчера, после полуночи показывает уже не те бюджеты. Идущий запрос сверяется
     * по своей дате: полночь под ним иначе осталась бы незамеченной, а второго колбэка
     * жизненного цикла не будет.
     */
    fun revalidate() {
        val current = requestedOn ?: return
        if (current == today()) return
        refresh()
    }

    fun refresh() {
        job?.cancel()
        requestedOn = today()
        mutable.value = BudgetsUiState.Loading
        job = viewModelScope.launch {
            mutable.value = try {
                val budgets = api.client
                    .unwrap {
                        api.budgets.listBudgets(
                            limit = BUDGET_LIMIT,
                            activeOnly = if (mutableFilter.value == BudgetFilter.TODAY) true else null,
                        )
                    }
                    .`data`
                val names = api.client
                    .unwrap { api.categories.listCategories(limit = BUDGET_LIMIT) }
                    .`data`
                    .associate { it.id to it.name }
                BudgetsUiState.Ready(budgets.map { row(it, names) })
            } catch (failure: ApiFailure) {
                BudgetsUiState.Failure(failure.toUiError())
            }
        }
    }

    private fun row(
        budget: Budget,
        names: Map<UUID, String>,
    ): BudgetRow = BudgetRow(
        budget = budget,
        categoryName = budget.categoryId?.let { names[it] },
        level = levelOf(budget.utilization),
    )
}

internal fun levelOf(utilization: Double): BudgetLevel = when {
    utilization >= FULL -> BudgetLevel.OVER
    utilization >= NEAR_LIMIT -> BudgetLevel.NEAR
    else -> BudgetLevel.OK
}
