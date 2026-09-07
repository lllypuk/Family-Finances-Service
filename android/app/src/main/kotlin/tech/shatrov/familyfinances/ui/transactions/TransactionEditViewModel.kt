package tech.shatrov.familyfinances.ui.transactions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.CreateTransactionRequest
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.UpdateTransactionRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.util.UUID

/** Справочник категорий берётся одной страницей: их у семьи считанные единицы. */
private const val CATEGORY_LIMIT = 200

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object TransactionField {
    const val AMOUNT = "amount_minor"
    const val TYPE = "type"
    const val CATEGORY = "category_id"
    const val DATE = "date"
    const val DESCRIPTION = "description"
}

private val formFields = setOf(
    TransactionField.AMOUNT,
    TransactionField.TYPE,
    TransactionField.CATEGORY,
    TransactionField.DATE,
    TransactionField.DESCRIPTION,
)

data class TransactionEditUiState(
    val amount: String = "",
    val type: TransactionType = TransactionType.expense,
    val categoryId: UUID? = null,
    val date: LocalDate = LocalDate.now(),
    val description: String = "",
    val categories: List<Category> = emptyList(),
    val editing: Boolean = false,
    val loading: Boolean = true,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    val done: Boolean = false,
) {
    /** Категория показывается только своего типа: расходную в доход сервер всё равно не примет. */
    val visibleCategories: List<Category>
        get() = categories.filter { it.type == type.asCategoryType() }

    val canSubmit: Boolean
        get() = parseAmountMinor(amount) != null && categoryId != null && !loading && !submitting
}

/**
 * Форма транзакции: создание, правка и удаление. `draftId` уходит в `POST` клиентским UUID,
 * поэтому повтор после обрыва не создаёт вторую запись — сервер отвечает существующей.
 */
class TransactionEditViewModel(
    private val api: ApiGraph,
    private val transactionId: UUID? = null,
    private val draftId: UUID = UUID.randomUUID(),
    today: LocalDate = LocalDate.now(),
) : ViewModel() {
    private val mutable = MutableStateFlow(
        TransactionEditUiState(date = today, editing = transactionId != null),
    )

    val state: StateFlow<TransactionEditUiState> = mutable.asStateFlow()

    init {
        load()
    }

    fun load() {
        mutable.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            try {
                val categories = api.client.unwrap(
                    { api.categories.listCategories(limit = CATEGORY_LIMIT) },
                    { it.`data` },
                )
                val existing = transactionId?.let { id ->
                    api.client.unwrap({ api.transactions.getTransaction(id) }, { it.`data` })
                }
                mutable.update { it.filled(categories, existing) }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(loading = false, error = failure.toUiError()) }
            }
        }
    }

    fun onAmountChange(amount: String) {
        mutable.update { it.copy(amount = amount).cleared(TransactionField.AMOUNT) }
    }

    /** Смена типа снимает выбранную категорию: её больше нет в списке под этим типом. */
    fun onTypeChange(type: TransactionType) {
        mutable.update { current ->
            val keep = current.categories.any { it.id == current.categoryId && it.type == type.asCategoryType() }
            current
                .copy(type = type, categoryId = if (keep) current.categoryId else null)
                .cleared(TransactionField.TYPE)
        }
    }

    fun onCategoryChange(categoryId: UUID) {
        mutable.update { it.copy(categoryId = categoryId).cleared(TransactionField.CATEGORY) }
    }

    fun onDateChange(date: LocalDate) {
        mutable.update { it.copy(date = date).cleared(TransactionField.DATE) }
    }

    fun onDescriptionChange(description: String) {
        mutable.update { it.copy(description = description).cleared(TransactionField.DESCRIPTION) }
    }

    fun onSubmit() {
        val current = mutable.value
        val amountMinor = parseAmountMinor(current.amount) ?: return
        val categoryId = current.categoryId ?: return
        if (!current.canSubmit) return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                save(current, amountMinor, categoryId)
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    fun onDelete() {
        val id = transactionId ?: return
        mutable.update { it.copy(submitting = true, error = null) }
        viewModelScope.launch {
            try {
                api.client.send { api.transactions.deleteTransaction(id) }
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    private suspend fun save(
        current: TransactionEditUiState,
        amountMinor: Long,
        categoryId: UUID,
    ) {
        val description = current.description.trim()
        if (transactionId == null) {
            api.client.unwrap(
                {
                    api.transactions.createTransaction(
                        CreateTransactionRequest(
                            amountMinor = amountMinor,
                            type = current.type,
                            description = description,
                            categoryId = categoryId,
                            date = current.date,
                            id = draftId,
                        ),
                    )
                },
                { it.`data` },
            )
        } else {
            api.client.unwrap(
                {
                    api.transactions.updateTransaction(
                        transactionId,
                        UpdateTransactionRequest(
                            amountMinor = amountMinor,
                            type = current.type,
                            description = description,
                            categoryId = categoryId,
                            date = current.date,
                        ),
                    )
                },
                { it.`data` },
            )
        }
    }
}

private fun TransactionType.asCategoryType(): CategoryType =
    if (this == TransactionType.income) CategoryType.income else CategoryType.expense

private fun TransactionEditUiState.filled(
    categories: List<Category>,
    existing: Transaction?,
): TransactionEditUiState = if (existing == null) {
    copy(loading = false, categories = categories)
} else {
    copy(
        loading = false,
        categories = categories,
        amount = formatAmountInput(existing.amountMinor),
        type = existing.type,
        categoryId = existing.categoryId,
        date = existing.date,
        description = existing.description,
    )
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun TransactionEditUiState.cleared(field: String): TransactionEditUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

// Детали 422 ложатся под поля; общий текст остаётся только для того, что под поле не легло.
private fun TransactionEditUiState.failed(failure: ApiFailure): TransactionEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in formFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        error = if (details.isNotEmpty() && underFields.size == details.size) null else failure.toUiError(),
    )
}
