package tech.shatrov.familyfinances.ui.budgets

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Budget
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.CreateBudgetRequest
import tech.shatrov.familyfinances.core.api.UpdateBudgetRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.util.UUID

/** Справочник категорий берётся одной страницей: их у семьи считанные единицы. */
private const val CATEGORY_LIMIT = 200

/** Нижняя граница контракта: `name` от двух символов. */
private const val MIN_NAME = 2

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object BudgetField {
    const val NAME = "name"
    const val AMOUNT = "amount_minor"
    const val PERIOD = "period"
    const val CATEGORY = "category_id"
    const val START = "start_date"
    const val END = "end_date"
}

private val formFields = setOf(
    BudgetField.NAME,
    BudgetField.AMOUNT,
    BudgetField.PERIOD,
    BudgetField.CATEGORY,
    BudgetField.START,
    BudgetField.END,
)

data class BudgetEditUiState(
    val name: String = "",
    val amount: String = "",
    val period: BudgetPeriod = BudgetPeriod.monthly,
    val categoryId: UUID? = null,
    val start: LocalDate = LocalDate.now(),
    val end: LocalDate = LocalDate.now(),
    val categories: List<Category> = emptyList(),
    val loaded: Budget? = null,
    val editing: Boolean = false,
    // Справочник прочитан: до этого ошибка формы — про загрузку, и «Повторить» имеет смысл.
    val ready: Boolean = false,
    val loading: Boolean = true,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    val done: Boolean = false,
) {
    val amountMinor: Long?
        get() = parseAmountMinor(amount)

    /**
     * Тело `PUT` из одних изменённых полей: `null`, когда менять нечего (`minProperties: 1`),
     * и заодно потому, что повтор неизменённой суммы перерасходованного бюджета сервер
     * отвергает — он сверяет её с уже посчитанным расходом.
     */
    val changes: UpdateBudgetRequest?
        get() {
            val existing = loaded ?: return null
            val request = UpdateBudgetRequest(
                name = name.trim().takeIf { it != existing.name },
                amountMinor = amountMinor?.takeIf { it != existing.amountMinor },
                startDate = start.takeIf { it != existing.startDate },
                endDate = end.takeIf { it != existing.endDate },
            )
            return request.takeIf { it != UpdateBudgetRequest() }
        }

    val canSubmit: Boolean
        get() = name.trim().length >= MIN_NAME &&
            (amountMinor ?: 0L) > 0L &&
            end > start &&
            !loading &&
            !submitting &&
            (!editing || changes != null)
}

/**
 * Форма бюджета: создание, правка и удаление. `draftId` уходит в `POST` клиентским UUID,
 * поэтому повтор после обрыва не создаёт второй бюджет. Период и категория при правке не
 * меняются: их нет в `UpdateBudgetRequest`.
 */
class BudgetEditViewModel(
    private val api: ApiGraph,
    private val budgetId: UUID? = null,
    private val draftId: UUID = UUID.randomUUID(),
    today: LocalDate = LocalDate.now(),
) : ViewModel() {
    private val mutable = MutableStateFlow(
        BudgetEditUiState(
            start = today,
            end = endOf(BudgetPeriod.monthly, today) ?: today,
            editing = budgetId != null,
        ),
    )

    val state: StateFlow<BudgetEditUiState> = mutable.asStateFlow()

    init {
        load()
    }

    fun load() {
        mutable.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            try {
                val categories = api.client
                    .unwrap { api.categories.listCategories(limit = CATEGORY_LIMIT) }
                    .`data`
                    .filter { it.type == CategoryType.expense }
                val existing = budgetId?.let { id -> api.client.unwrap { api.budgets.getBudget(id) }.`data` }
                mutable.update { it.filled(categories, existing) }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(loading = false, error = failure.toUiError()) }
            }
        }
    }

    fun onNameChange(name: String) {
        mutable.update { it.copy(name = name).cleared(BudgetField.NAME) }
    }

    fun onAmountChange(amount: String) {
        mutable.update { it.copy(amount = amount).cleared(BudgetField.AMOUNT) }
    }

    /** Пресет конца — только при создании: правка конца по периоду сдвинула бы чужие даты. */
    fun onPeriodChange(period: BudgetPeriod) {
        mutable.update { current ->
            current
                .copy(period = period, end = current.presetEnd(period, current.start))
                .cleared(BudgetField.PERIOD)
        }
    }

    fun onStartChange(start: LocalDate) {
        mutable.update { current ->
            current
                .copy(start = start, end = current.presetEnd(current.period, start))
                .cleared(BudgetField.START)
        }
    }

    /** Ручной конец период не переключает: «произвольный» выбирают, а не получают молча. */
    fun onEndChange(end: LocalDate) {
        mutable.update { it.copy(end = end).cleared(BudgetField.END) }
    }

    fun onCategoryChange(categoryId: UUID?) {
        mutable.update { it.copy(categoryId = categoryId).cleared(BudgetField.CATEGORY) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        val amountMinor = current.amountMinor ?: return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                save(current, amountMinor)
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    fun onDelete() {
        val id = budgetId ?: return
        mutable.update { it.copy(submitting = true, error = null) }
        viewModelScope.launch {
            try {
                api.client.send { api.budgets.deleteBudget(id) }
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    private suspend fun save(
        current: BudgetEditUiState,
        amountMinor: Long,
    ) {
        if (budgetId == null) {
            api.client.unwrap {
                api.budgets.createBudget(
                    CreateBudgetRequest(
                        name = current.name.trim(),
                        amountMinor = amountMinor,
                        period = current.period,
                        startDate = current.start,
                        endDate = current.end,
                        id = draftId,
                        categoryId = current.categoryId,
                    ),
                )
            }
        } else {
            val changes = current.changes ?: return
            api.client.unwrap { api.budgets.updateBudget(budgetId, changes) }
        }
    }
}

private fun BudgetEditUiState.presetEnd(
    period: BudgetPeriod,
    start: LocalDate,
): LocalDate = if (editing) end else endOf(period, start) ?: end

private fun BudgetEditUiState.filled(
    categories: List<Category>,
    existing: Budget?,
): BudgetEditUiState = if (existing == null) {
    copy(loading = false, ready = true, categories = categories)
} else {
    copy(
        loading = false,
        ready = true,
        categories = categories,
        loaded = existing,
        name = existing.name,
        amount = formatAmountInput(existing.amountMinor),
        period = existing.period,
        categoryId = existing.categoryId,
        start = existing.startDate,
        end = existing.endDate,
    )
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun BudgetEditUiState.cleared(field: String): BudgetEditUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

// Детали 422 ложатся под поля; отказ с `field: "body"` сервер не даёт различить — своим текстом.
private fun BudgetEditUiState.failed(failure: ApiFailure): BudgetEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in formFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        error = when {
            details.size > underFields.size -> UiError.Resource(R.string.budget_error_rejected)
            details.isNotEmpty() -> null
            else -> failure.toUiError()
        },
    )
}
