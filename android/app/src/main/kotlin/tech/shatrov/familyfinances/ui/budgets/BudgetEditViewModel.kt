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

/** `201` — бюджет создан этим запросом; `200` — сервер отдал созданный прошлой попыткой. */
private const val HTTP_CREATED = 201

private const val HTTP_CONFLICT = 409

/** Серия ушла вперёд: форма держит прошлый инстанс, хвост теперь другая запись. */
private const val NOT_TAIL = "BUDGET_NOT_TAIL"

/** Коды `409` бюджета: серверный текст английский, на форме нужен свой. */
private val budgetConflicts: Map<String, Int> = mapOf(
    "BUDGET_OVERLAP" to R.string.budget_error_overlap,
    "BUDGET_NAME_EXISTS" to R.string.budget_error_name_exists,
    "BUDGET_BELOW_SPENT" to R.string.budget_error_below_spent,
    "BUDGET_ID_EXISTS" to R.string.budget_error_id_taken,
    NOT_TAIL to R.string.budget_error_not_tail,
)

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object BudgetField {
    const val NAME = "name"
    const val AMOUNT = "amount_minor"
    const val PERIOD = "period"
    const val CATEGORY = "category_id"
    const val START = "start_date"
    const val END = "end_date"
    const val RECURRING = "recurring"
}

private val formFields = setOf(
    BudgetField.NAME,
    BudgetField.AMOUNT,
    BudgetField.PERIOD,
    BudgetField.CATEGORY,
    BudgetField.START,
    BudgetField.END,
    BudgetField.RECURRING,
)

data class BudgetEditUiState(
    val name: String = "",
    val amount: String = "",
    val period: BudgetPeriod = BudgetPeriod.monthly,
    val categoryId: UUID? = null,
    val start: LocalDate = LocalDate.now(),
    val end: LocalDate = LocalDate.now(),
    val recurring: Boolean = false,
    val categories: List<Category> = emptyList(),
    val loaded: Budget? = null,
    val editing: Boolean = false,
    // Справочник прочитан: до этого ошибка формы — про загрузку, и «Повторить» имеет смысл.
    val ready: Boolean = false,
    val loading: Boolean = true,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    // Сервер запись принял, даже если форма осталась открытой: список и сводка устарели.
    val saved: Boolean = false,
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
                recurring = recurring.takeIf { it != existing.recurring },
            )
            return request.takeIf { it != UpdateBudgetRequest() }
        }

    /** `custom` — период без длины: шагать серии нечем, сервер отвечает `422`. */
    val canRecur: Boolean
        get() = period != BudgetPeriod.custom

    /** Член серии: его даты неизменяемы — перенос прошлого инстанса за хвост сломал бы серию. */
    val inSeries: Boolean
        get() = loaded?.seriesId != null

    // Границы повторяющегося бюджета — календарные, и подставляет их форма; у `weekly`
    // выравнивания нет, поэтому начало остаётся за пользователем.
    val startLocked: Boolean
        get() = inSeries || (recurring && period != BudgetPeriod.weekly)

    val endLocked: Boolean
        get() = inSeries || recurring

    /** Конец не позже начала: кнопка гаснет, и без подписи форма выглядит сломанной. */
    val periodInvalid: Boolean
        get() = end <= start

    // `ready` обязателен: без справочника «все категории» — не выбор пользователя, а пустой
    // список, и созданный так бюджет категорию уже не получит — её нет в `UpdateBudgetRequest`.
    val canSubmit: Boolean
        get() = name.trim().length >= MIN_NAME &&
            (amountMinor ?: 0L) > 0L &&
            !periodInvalid &&
            ready &&
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
    // Повтор `POST` мог принести id уже созданной записи: дальше форма правит именно её.
    private var savedId: UUID? = budgetId

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
                .withPeriod(period, current.start, current.recurring && period != BudgetPeriod.custom)
                .cleared(BudgetField.PERIOD)
                .cleared(BudgetField.RECURRING)
        }
    }

    fun onStartChange(start: LocalDate) {
        mutable.update { current ->
            current
                .withPeriod(current.period, start, current.recurring)
                .cleared(BudgetField.START)
        }
    }

    /** Включение подставляет календарные границы: невыровненные сервер отвергает. */
    fun onRecurringChange(recurring: Boolean) {
        mutable.update { current ->
            current
                .withPeriod(current.period, current.start, recurring && current.canRecur)
                .cleared(BudgetField.RECURRING)
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
                val unapplied = save(current, amountMinor)
                mutable.update {
                    if (unapplied == null) {
                        it.copy(submitting = false, done = true)
                    } else {
                        it.copy(submitting = false, error = unapplied)
                    }
                }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
                if (failure is ApiFailure.Api && failure.code == NOT_TAIL) reloadAfterAdvance()
            }
        }
    }

    fun onDelete() {
        val id = savedId ?: return
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

    /** Возвращает ошибку, когда сохранено не то, что на форме, — иначе `null`. */
    private suspend fun save(
        current: BudgetEditUiState,
        amountMinor: Long,
    ): UiError? {
        val id = savedId
        if (id == null) {
            val (code, created) = api.client.unwrapWithCode {
                api.budgets.createBudget(
                    CreateBudgetRequest(
                        name = current.name.trim(),
                        amountMinor = amountMinor,
                        period = current.period,
                        startDate = current.start,
                        endDate = current.end,
                        id = draftId,
                        categoryId = current.categoryId,
                        recurring = current.recurring,
                    ),
                )
            }

            return if (code == HTTP_CREATED) null else reconcile(current, created.`data`)
        }
        val changes = current.changes ?: return null
        api.client.unwrap { api.budgets.updateBudget(id, changes) }

        return null
    }

    /**
     * Серию продвинуло чужое чтение: и копия формы, и список отстали — перечитываем бюджет,
     * а список помечаем устаревшим, потому что новый хвост видно только в нём.
     */
    private fun reloadAfterAdvance() {
        val id = savedId ?: return
        mutable.update { it.copy(saved = true) }
        viewModelScope.launch {
            try {
                val actual = api.client.unwrap { api.budgets.getBudget(id) }.`data`
                mutable.update { it.filled(it.categories, actual) }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(error = failure.toUiError()) }
            }
        }
    }

    /**
     * `200` на `POST`: бюджет создала прошлая попытка, а тело повтора сервер не смотрел —
     * правки, сделанные между попытками, дошлём `PUT`-ом, иначе форма закроется успехом
     * поверх прежних значений. Период и категорию `PUT` не меняет: о них остаётся сказать.
     * Форма при этом переходит в правку созданной записи — повторный `POST` вернул бы
     * тот же отказ, и выйти из него было бы нечем.
     */
    private suspend fun reconcile(
        current: BudgetEditUiState,
        created: Budget,
    ): UiError? {
        savedId = created.id
        mutable.update { it.copy(loaded = created, editing = true, saved = true) }
        val applied = current.copy(loaded = created).changes?.let { changes ->
            api.client.unwrap { api.budgets.updateBudget(created.id, changes) }.`data`
        } ?: created
        val locked = created.period != current.period || created.categoryId != current.categoryId
        mutable.update { it.copy(loaded = applied, period = applied.period, categoryId = applied.categoryId) }

        return UiError.Resource(R.string.budget_error_recreated).takeIf { locked }
    }
}

private fun BudgetEditUiState.presetEnd(
    period: BudgetPeriod,
    start: LocalDate,
): LocalDate = if (editing) end else endOf(period, start) ?: end

/**
 * Границы под период: у повторяющегося — ровно календарный период, даже при правке, иначе
 * `PUT` с флагом упрётся в `422`; у обычного — прежний пресет конца.
 */
private fun BudgetEditUiState.withPeriod(
    period: BudgetPeriod,
    start: LocalDate,
    recurring: Boolean,
): BudgetEditUiState {
    if (!recurring) {
        return copy(period = period, recurring = false, start = start, end = presetEnd(period, start))
    }
    val aligned = alignedStart(period, start)
    return copy(period = period, recurring = true, start = aligned, end = endOf(period, aligned) ?: end)
}

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
        recurring = existing.recurring,
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
            else -> failure.toBudgetError()
        },
    )
}

/** Незнакомый `409` — тот же общий текст: он про состояние бюджета, а не про сеть или поле. */
private fun ApiFailure.toBudgetError(): UiError = if (this is ApiFailure.Api && status == HTTP_CONFLICT) {
    UiError.Resource(budgetConflicts[code] ?: R.string.budget_error_rejected)
} else {
    toUiError()
}
