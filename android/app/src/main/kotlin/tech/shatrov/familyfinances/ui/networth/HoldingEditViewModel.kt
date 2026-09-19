package tech.shatrov.familyfinances.ui.networth

import androidx.annotation.StringRes
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.CreateHoldingRequest
import tech.shatrov.familyfinances.core.api.Holding
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.HoldingValueRequest
import tech.shatrov.familyfinances.core.api.UpdateHoldingRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.util.UUID

/** Позиций у семьи единицы: справочник берётся одной страницей вместе с архивом. */
internal const val HOLDING_LIMIT = 200

/** Верхняя граница контракта для `name`. */
const val HOLDING_NAME_MAX = 50

/** Имена полей формы — те же, что в `error.details[].field`. */
const val HOLDING_FIELD_NAME = "name"

const val HOLDING_FIELD_INCOME = "monthly_income_minor"

const val HOLDING_FIELD_EXPENSE = "monthly_expense_minor"

private val HOLDING_FORM_FIELDS = setOf(HOLDING_FIELD_NAME, HOLDING_FIELD_INCOME, HOLDING_FIELD_EXPENSE)

private const val HOLDING_NAME_EXISTS = "HOLDING_NAME_EXISTS"

/**
 * Вид позиции. В контракте это строка, а не `enum`: список растёт на сервере, и незнакомый вид
 * установленный APK показывает как [OTHER], а не падает на разборе.
 */
enum class HoldingKind(
    val wire: String,
    @StringRes val label: Int,
) {
    CASH("cash", R.string.holding_kind_cash),
    DEPOSIT("deposit", R.string.holding_kind_deposit),
    INVESTMENT("investment", R.string.holding_kind_investment),
    PROPERTY("property", R.string.holding_kind_property),
    VEHICLE("vehicle", R.string.holding_kind_vehicle),
    MORTGAGE("mortgage", R.string.holding_kind_mortgage),
    LOAN("loan", R.string.holding_kind_loan),
    CREDIT_CARD("credit_card", R.string.holding_kind_credit_card),
    OTHER("other", R.string.holding_kind_other),
    ;

    companion object {
        fun of(wire: String): HoldingKind = entries.firstOrNull { it.wire == wire } ?: OTHER

        /** Виды своей стороны: чужой сервер отвергнет `422`. */
        fun of(side: HoldingSide): List<HoldingKind> = when (side) {
            HoldingSide.asset -> listOf(CASH, DEPOSIT, INVESTMENT, PROPERTY, VEHICLE, OTHER)
            HoldingSide.liability -> listOf(MORTGAGE, LOAN, CREDIT_CARD, OTHER)
        }
    }
}

/**
 * Форма позиции. [kind] — строка с провода: незнакомый вид остаётся как есть, пока его не сменят,
 * и в `PUT` не уходит. [side] меняется только у новой.
 */
data class HoldingEditUiState(
    val id: UUID?,
    val draft: UUID,
    val side: HoldingSide,
    val name: String = "",
    val kind: String = HoldingKind.OTHER.wire,
    val income: String = "",
    val expense: String = "",
    val saved: Holding? = null,
    val loading: Boolean = false,
    val loadError: UiError? = null,
    val canDelete: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    val done: Boolean = false,
    val changed: Boolean = false,
) {
    val editing: Boolean get() = id != null

    val archived: Boolean get() = saved?.isArchived == true

    /** Архив без нулевого снимка оставил бы последнюю стоимость в капитале. */
    val archiveNeedsZero: Boolean get() = !archived && (saved?.current?.valueMinor ?: 0L) != 0L

    /** `null` — введено не число; пустое поле — `0`, то есть «плана нет». */
    val incomeMinor: Long? get() = planMinor(income)

    val expenseMinor: Long? get() = planMinor(expense)

    val canSubmit: Boolean
        get() {
            val trimmed = name.trim()
            if (trimmed.isEmpty() || trimmed.length > HOLDING_NAME_MAX || submitting || loading) return false
            val income = incomeMinor ?: return false
            val expense = expenseMinor ?: return false
            val before = saved ?: return true
            return trimmed != before.name || kind != before.kind ||
                income != before.incomeMinor || expense != before.expenseMinor
        }
}

private fun planMinor(text: String): Long? = if (text.isBlank()) 0L else parseAmountMinor(text)

/** Сервер до `v0.8.0` полей плана не шлёт: для формы это «плана нет». */
internal val Holding.incomeMinor: Long get() = monthlyIncomeMinor ?: 0L

internal val Holding.expenseMinor: Long get() = monthlyExpenseMinor ?: 0L

/** Сохранённый `0` открывается пустым полем, а не «0». */
private fun planInput(minor: Long): String = if (minor == 0L) "" else formatAmountInput(minor)

/** Создание, правка, архив и удаление позиции (удаление — только админу, снимки уходят каскадом). */
class HoldingEditViewModel(
    private val api: ApiGraph,
    id: UUID?,
    draft: UUID,
    side: HoldingSide,
    private val today: () -> LocalDate,
    private val isAdmin: Boolean,
) : ViewModel() {
    private val mutable = MutableStateFlow(
        HoldingEditUiState(
            id = id,
            draft = draft,
            side = side,
            kind = HoldingKind.of(side).first().wire,
            loading = id != null,
            canDelete = id != null && isAdmin,
        ),
    )

    val state: StateFlow<HoldingEditUiState> = mutable.asStateFlow()

    init {
        if (id != null) load()
    }

    /** Чтения по `id` у позиций нет: форма находит свою в списке, он один и короткий. */
    fun load() {
        val id = mutable.value.id ?: return
        mutable.update { it.copy(loading = true, loadError = null) }
        viewModelScope.launch {
            try {
                val page = api.client.unwrap { api.holdings.listHoldings(limit = HOLDING_LIMIT, archived = true) }
                val found = page.`data`.firstOrNull { it.id == id }
                mutable.update {
                    when {
                        found != null -> it.copy(
                            loading = false,
                            saved = found,
                            side = found.side,
                            name = found.name,
                            kind = found.kind,
                            income = planInput(found.incomeMinor),
                            expense = planInput(found.expenseMinor),
                        )

                        // Позиция может быть за первой страницей: закрыть форму значило бы соврать, что её нет.
                        page.meta.pagination.total > page.`data`.size -> it.copy(
                            loading = false,
                            loadError = UiError.Resource(R.string.holding_error_beyond_page),
                        )

                        else -> it.copy(loading = false, done = true)
                    }
                }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(loading = false, loadError = failure.toUiError()) }
            }
        }
    }

    fun onSideChange(side: HoldingSide) {
        mutable.update {
            if (it.editing || it.side == side) it else it.copy(side = side, kind = HoldingKind.of(side).first().wire)
        }
    }

    fun onNameChange(name: String) {
        mutable.update { it.copy(name = name, fieldErrors = it.fieldErrors - HOLDING_FIELD_NAME) }
    }

    fun onKindChange(kind: HoldingKind) {
        mutable.update { it.copy(kind = kind.wire) }
    }

    fun onIncomeChange(income: String) {
        mutable.update { it.copy(income = income, fieldErrors = it.fieldErrors - HOLDING_FIELD_INCOME) }
    }

    fun onExpenseChange(expense: String) {
        mutable.update { it.copy(expense = expense, fieldErrors = it.fieldErrors - HOLDING_FIELD_EXPENSE) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        val name = current.name.trim()
        val income = current.incomeMinor ?: return
        val expense = current.expenseMinor ?: return
        val saved = current.saved
        mutate {
            if (saved == null) {
                api.client.unwrap {
                    api.holdings.createHolding(
                        CreateHoldingRequest(
                            name = name,
                            side = current.side,
                            kind = current.kind,
                            id = current.draft,
                            monthlyIncomeMinor = income.takeIf { it != 0L },
                            monthlyExpenseMinor = expense.takeIf { it != 0L },
                        ),
                    )
                }
            } else {
                val request = UpdateHoldingRequest(
                    name = name.takeIf { it != saved.name },
                    kind = current.kind.takeIf { it != saved.kind },
                    monthlyIncomeMinor = income.takeIf { it != saved.incomeMinor },
                    monthlyExpenseMinor = expense.takeIf { it != saved.expenseMinor },
                )
                api.client.unwrap { api.holdings.updateHolding(saved.id, request) }
            }
        }
    }

    /**
     * Архив или возврат. [zeroFirst] — сначала снимок `0` на сегодня: архив только прячет позицию,
     * а её последняя стоимость так и переносилась бы вперёд в капитале.
     */
    fun onToggleArchive(zeroFirst: Boolean = false) {
        val saved = mutable.value.saved ?: return
        mutate {
            if (zeroFirst && !saved.isArchived) {
                api.client.unwrap { api.holdings.putHoldingValue(saved.id, today(), HoldingValueRequest(0L)) }
                // Снимок уже в капитале, даже если архив ниже не пройдёт.
                mutable.update { it.copy(changed = true) }
            }
            api.client.unwrap {
                api.holdings.updateHolding(saved.id, UpdateHoldingRequest(isArchived = !saved.isArchived))
            }
        }
    }

    fun onDelete() {
        val current = mutable.value
        val id = current.saved?.id?.takeIf { current.canDelete } ?: return
        mutate { api.client.send { api.holdings.deleteHolding(id) } }
    }

    private fun mutate(call: suspend () -> Any) {
        if (mutable.value.submitting) return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                call()
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                // Позицию удалили с другого телефона: форме нечего править, устарел список.
                if ((failure as? ApiFailure.Api)?.isNotFound == true) {
                    mutable.update { it.copy(submitting = false, done = true) }
                } else {
                    mutable.update { it.failed(failure) }
                }
            }
        }
    }
}

private fun HoldingEditUiState.failed(failure: ApiFailure): HoldingEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in HOLDING_FORM_FIELDS }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        error = if (details.isNotEmpty() && underFields.size == details.size) {
            null
        } else {
            failure.toUiError(mapOf(HOLDING_NAME_EXISTS to R.string.holding_error_name_exists))
        },
    )
}
