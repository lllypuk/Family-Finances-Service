package tech.shatrov.familyfinances.ui.transactions

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.LastAccountStore
import tech.shatrov.familyfinances.core.api.Account
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

/** Справочники берутся одной страницей: категорий и счетов у семьи считанные единицы. */
private const val CATEGORY_LIMIT = 200

/** Нижние границы контракта: `amount_minor > 0`, `description` от двух символов. */
private const val MIN_DESCRIPTION = 2

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object TransactionField {
    const val AMOUNT = "amount_minor"
    const val TYPE = "type"
    const val CATEGORY = "category_id"
    const val DATE = "date"
    const val DESCRIPTION = "description"
    const val ACCOUNT = "account_id"
}

private val formFields = setOf(
    TransactionField.AMOUNT,
    TransactionField.TYPE,
    TransactionField.CATEGORY,
    TransactionField.DATE,
    TransactionField.DESCRIPTION,
    TransactionField.ACCOUNT,
)

data class TransactionEditUiState(
    val amount: String = "",
    val currency: String = "",
    val type: TransactionType = TransactionType.expense,
    val categoryId: UUID? = null,
    val date: LocalDate = LocalDate.now(),
    val description: String = "",
    val accountId: UUID? = null,
    val storedAccountId: UUID? = null,
    val categories: List<Category> = emptyList(),
    val accounts: List<Account> = emptyList(),
    val editing: Boolean = false,
    val loading: Boolean = true,
    val loaded: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    val done: Boolean = false,
) {
    /** Категория показывается только своего типа: расходную в доход сервер всё равно не примет. */
    val visibleCategories: List<Category>
        get() = categories.filter { it.type == type.asCategoryType() }

    /** В выборе только действующие: архивный виден лишь как уже привязанный к операции. */
    val selectableAccounts: List<Account>
        get() = accounts.filter { !it.isArchived }

    val account: Account?
        get() = accounts.firstOrNull { it.id == accountId }

    /** Счетов у семьи нет и к операции ничего не привязано — поле было бы пустым выбором. */
    val showsAccount: Boolean
        get() = selectableAccounts.isNotEmpty() || accountId != null

    val canSubmit: Boolean
        get() = (parseAmountMinor(amount) ?: 0L) > 0L &&
            description.trim().length >= MIN_DESCRIPTION &&
            categoryId != null &&
            !loading &&
            !submitting
}

/**
 * Форма транзакции: создание, правка и удаление. `draftId` уходит в `POST` клиентским UUID,
 * поэтому повтор после обрыва не создаёт вторую запись — сервер отвечает существующей.
 */
class TransactionEditViewModel(
    private val api: ApiGraph,
    private val lastAccount: LastAccountStore,
    private val transactionId: UUID? = null,
    private val draftId: UUID = UUID.randomUUID(),
    today: LocalDate = LocalDate.now(),
    currency: String = "",
) : ViewModel() {
    private val mutable = MutableStateFlow(
        TransactionEditUiState(currency = currency, date = today, editing = transactionId != null),
    )

    val state: StateFlow<TransactionEditUiState> = mutable.asStateFlow()

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
                // С архивными: привязанный к операции архивный счёт показывается по имени.
                val accounts = api.client
                    .unwrap { api.accounts.listAccounts(limit = CATEGORY_LIMIT, archived = true) }
                    .`data`
                val existing = transactionId?.let { id ->
                    api.client.unwrap { api.transactions.getTransaction(id) }.`data`
                }
                mutable.update { it.filled(categories, accounts, existing, lastAccount.read()) }
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

    /** `null` — «Без счёта»: у сохранённой операции уходит `clear_account`. */
    fun onAccountChange(accountId: UUID?) {
        mutable.update { it.copy(accountId = accountId).cleared(TransactionField.ACCOUNT) }
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
            api.client.unwrap {
                api.transactions.createTransaction(
                    CreateTransactionRequest(
                        amountMinor = amountMinor,
                        type = current.type,
                        description = description,
                        categoryId = categoryId,
                        date = current.date,
                        id = draftId,
                        accountId = current.accountId,
                    ),
                )
            }
            lastAccount.write(current.accountId)
        } else {
            val changed = current.accountId != current.storedAccountId
            api.client.unwrap {
                api.transactions.updateTransaction(
                    transactionId,
                    UpdateTransactionRequest(
                        amountMinor = amountMinor,
                        type = current.type,
                        description = description,
                        categoryId = categoryId,
                        date = current.date,
                        // Без поля счёт не трогается: `explicitNulls = false` не шлёт `null`,
                        // поэтому отвязка — отдельный флаг.
                        accountId = current.accountId.takeIf { changed },
                        clearAccount = true.takeIf { changed && current.accountId == null },
                    ),
                )
            }
        }
    }
}

internal fun TransactionType.asCategoryType(): CategoryType =
    if (this == TransactionType.income) CategoryType.income else CategoryType.expense

private fun TransactionEditUiState.filled(
    categories: List<Category>,
    accounts: List<Account>,
    existing: Transaction?,
    last: UUID?,
): TransactionEditUiState = if (existing == null) {
    val usable = accounts.any { it.id == last && !it.isArchived }
    copy(
        loading = false,
        loaded = true,
        categories = categories,
        accounts = accounts,
        accountId = last.takeIf { usable },
    )
} else {
    copy(
        loading = false,
        loaded = true,
        categories = categories,
        accounts = accounts,
        accountId = existing.accountId,
        storedAccountId = existing.accountId,
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
