package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.SetPasswordRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import java.util.UUID

data class UserPasswordUiState(
    val next: String = "",
    val repeat: String = "",
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    /** Сервер ответил `204`: страница закрывается, сообщение показывает форма пользователя. */
    val done: Boolean = false,
    /** Роль сняли: страницу закрывает хост, повторять правку незачем. */
    val forbidden: Boolean = false,
) {
    val newLength: Int
        get() = next.toByteArray(Charsets.UTF_8).size

    val lengthInvalid: Boolean
        get() = next.isNotEmpty() && newLength !in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES

    val mismatch: Boolean
        get() = repeat.isNotEmpty() && repeat != next

    val canSubmit: Boolean
        get() = !submitting && newLength in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES && repeat == next
}

/**
 * Установка пароля пользователю админом: текущий не спрашивается, все сессии цели отзываются.
 * Повтора после обрыва нет — пароль мог уже смениться, и вторая попытка ничего не проверит.
 */
class UserPasswordViewModel(
    private val api: ApiGraph,
    private val id: UUID,
) : ViewModel() {
    private val mutable = MutableStateFlow(UserPasswordUiState())

    val state: StateFlow<UserPasswordUiState> = mutable.asStateFlow()

    fun onNewChange(value: String) {
        mutable.update { it.copy(next = value).cleared(PasswordField.NEW) }
    }

    fun onRepeatChange(value: String) {
        mutable.update { it.copy(repeat = value) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                api.client.send { api.users.setUserPassword(id, SetPasswordRequest(current.next)) }
                mutable.update { it.copy(submitting = false, done = true) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }
}

private fun UserPasswordUiState.cleared(field: String): UserPasswordUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

private fun UserPasswordUiState.failed(failure: ApiFailure): UserPasswordUiState {
    val rejected = failure as? ApiFailure.Api
    val details = rejected?.details.orEmpty()
    val underFields = details.filter { it.`field` == PasswordField.NEW }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        forbidden = failure.forbidden,
        error = when {
            failure.resultUnknown -> UiError.Resource(R.string.settings_user_password_unknown)
            underFields.size == details.size && underFields.isNotEmpty() -> null
            else -> failure.toSettingsError()
        },
    )
}
