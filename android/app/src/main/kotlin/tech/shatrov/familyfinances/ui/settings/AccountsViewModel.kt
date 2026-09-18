package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.CreateAccountRequest
import tech.shatrov.familyfinances.core.api.UpdateAccountRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import java.util.UUID

/** Счетов у семьи единицы: справочник берётся одной страницей вместе с архивом. */
private const val ACCOUNT_LIMIT = 200

/** Верхняя граница контракта для `name`. */
const val ACCOUNT_NAME_MAX = 50

/** Имя поля формы — то же, что в `error.details[].field`. */
const val ACCOUNT_FIELD_NAME = "name"

sealed interface AccountsUiState {
    data object Loading : AccountsUiState

    /** [forbidden] — роль сняли: страницу закрывает хост. */
    data class Failure(
        val error: UiError,
        val forbidden: Boolean = false,
    ) : AccountsUiState

    data class Ready(
        val active: List<Account>,
        val archived: List<Account>,
    ) : AccountsUiState {
        val isEmpty: Boolean get() = active.isEmpty() && archived.isEmpty()
    }
}

/** Форма счёта; `draft` — `id` будущего `POST`, повтор после обрыва не создаст второй счёт. */
data class AccountEditUiState(
    val id: UUID? = null,
    val draft: UUID = UUID.randomUUID(),
    val name: String = "",
    val savedName: String = "",
    val archived: Boolean = false,
    val canDelete: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
) {
    val editing: Boolean get() = id != null

    val canSubmit: Boolean
        get() {
            val trimmed = name.trim()
            return trimmed.isNotEmpty() && trimmed.length <= ACCOUNT_NAME_MAX && trimmed != savedName && !submitting
        }
}

/** Счета: список с формой создания, переименования, архива и удаления (удаление — только админу). */
class AccountsViewModel(
    private val api: ApiGraph,
    private val isAdmin: Boolean,
) : ViewModel() {
    private val mutable = MutableStateFlow<AccountsUiState>(AccountsUiState.Loading)
    private val mutableEditor = MutableStateFlow<AccountEditUiState?>(null)

    val state: StateFlow<AccountsUiState> = mutable.asStateFlow()

    /** `null` — открыт список; иначе поверх него форма. */
    val editor: StateFlow<AccountEditUiState?> = mutableEditor.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = AccountsUiState.Loading
        viewModelScope.launch {
            try {
                val all = api.client.unwrap {
                    api.accounts.listAccounts(limit = ACCOUNT_LIMIT, archived = true)
                }.`data`
                val (archived, active) = all.partition { it.isArchived }
                mutable.value = AccountsUiState.Ready(active = active, archived = archived)
            } catch (failure: ApiFailure) {
                mutable.value = AccountsUiState.Failure(failure.toSettingsError(), failure.forbidden)
            }
        }
    }

    fun onAdd() {
        mutableEditor.value = AccountEditUiState()
    }

    fun onOpen(account: Account) {
        mutableEditor.value = AccountEditUiState(
            id = account.id,
            name = account.name,
            savedName = account.name,
            archived = account.isArchived,
            canDelete = isAdmin,
        )
    }

    fun onDismiss() {
        if (mutableEditor.value?.submitting == true) return
        mutableEditor.value = null
    }

    fun onNameChange(name: String) {
        mutableEditor.update { current ->
            current?.copy(name = name, fieldErrors = current.fieldErrors - ACCOUNT_FIELD_NAME)
        }
    }

    fun onSubmit() {
        val current = mutableEditor.value ?: return
        if (!current.canSubmit) return
        val name = current.name.trim()
        val id = current.id
        mutate(current) {
            if (id == null) {
                api.client.unwrap { api.accounts.createAccount(CreateAccountRequest(name = name, id = current.draft)) }
            } else {
                api.client.unwrap { api.accounts.updateAccount(id, UpdateAccountRequest(name = name)) }
            }
        }
    }

    /** Архив и возврат уходят сразу, без «Сохранить», и закрывают форму; несохранённое имя не отправляется. */
    fun onToggleArchive() {
        val current = mutableEditor.value ?: return
        val id = current.id ?: return
        if (current.submitting) return
        mutate(current) {
            api.client.unwrap {
                api.accounts.updateAccount(id, UpdateAccountRequest(isArchived = !current.archived))
            }
        }
    }

    fun onDelete() {
        val current = mutableEditor.value ?: return
        val id = current.id?.takeIf { current.canDelete } ?: return
        if (current.submitting) return
        mutate(current) { api.client.send { api.accounts.deleteAccount(id) } }
    }

    private fun mutate(
        current: AccountEditUiState,
        call: suspend () -> Any,
    ) {
        mutableEditor.value = current.copy(submitting = true, error = null, fieldErrors = emptyMap())
        viewModelScope.launch {
            try {
                call()
                mutableEditor.value = null
                refresh()
            } catch (failure: ApiFailure) {
                when {
                    failure.forbidden -> {
                        mutable.value = AccountsUiState.Failure(failure.toSettingsError(), forbidden = true)
                        mutableEditor.value = null
                    }

                    // Счёт удалили с другого телефона: форме нечего править, устарел список.
                    (failure as? ApiFailure.Api)?.isNotFound == true -> {
                        mutableEditor.value = null
                        refresh()
                    }

                    else -> mutableEditor.update { it?.failed(failure) }
                }
            }
        }
    }
}

// Ошибка поля `name` ложится под поле; остальное — общим текстом.
private fun AccountEditUiState.failed(failure: ApiFailure): AccountEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underName = details.filter { it.`field` == ACCOUNT_FIELD_NAME }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underName,
        error = if (details.isNotEmpty() && underName.size == details.size) null else failure.toSettingsError(),
    )
}
