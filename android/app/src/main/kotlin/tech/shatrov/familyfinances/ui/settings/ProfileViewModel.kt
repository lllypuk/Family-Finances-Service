package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.UpdateUserRequest
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object ProfileField {
    const val EMAIL = "email"
    const val FIRST_NAME = "first_name"
    const val LAST_NAME = "last_name"
}

private val profileFields = setOf(ProfileField.EMAIL, ProfileField.FIRST_NAME, ProfileField.LAST_NAME)

data class ProfileUiState(
    val loaded: User,
    val email: String = loaded.email,
    val firstName: String = loaded.firstName,
    val lastName: String = loaded.lastName,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    /** Ответ `PUT /me`: сессию обновляет хост, модель о графе не знает. */
    val saved: User? = null,
) {
    /** Тело `PUT` из одних изменённых полей: пустого сервер не принимает (`minProperties: 1`). */
    val changes: UpdateUserRequest?
        get() {
            val request = UpdateUserRequest(
                email = email.trim().takeIf { it != loaded.email },
                firstName = firstName.trim().takeIf { it != loaded.firstName },
                lastName = lastName.trim().takeIf { it != loaded.lastName },
            )
            return request.takeIf { it != UpdateUserRequest() }
        }

    val canSubmit: Boolean
        get() = !submitting &&
            email.isNotBlank() &&
            firstName.isNotBlank() &&
            lastName.isNotBlank() &&
            changes != null
}

/** Профиль: имя и почта своей записи. Роль и активность этим роутом не меняются. */
class ProfileViewModel(
    private val api: ApiGraph,
    user: User,
) : ViewModel() {
    private val mutable = MutableStateFlow(ProfileUiState(loaded = user))

    val state: StateFlow<ProfileUiState> = mutable.asStateFlow()

    fun onEmailChange(email: String) {
        mutable.update { it.copy(email = email).cleared(ProfileField.EMAIL) }
    }

    fun onFirstNameChange(name: String) {
        mutable.update { it.copy(firstName = name).cleared(ProfileField.FIRST_NAME) }
    }

    fun onLastNameChange(name: String) {
        mutable.update { it.copy(lastName = name).cleared(ProfileField.LAST_NAME) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        val changes = current.changes ?: return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                val saved = api.client.unwrap { api.me.updateCurrentUser(changes) }.`data`
                mutable.update { it.copy(submitting = false, saved = saved) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun ProfileUiState.cleared(field: String): ProfileUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

private fun ProfileUiState.failed(failure: ApiFailure): ProfileUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in profileFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        // Деталь не про поле формы (`field: "body"`) осталась бы без текста — показываем её общим.
        error = if (underFields.size == details.size && underFields.isNotEmpty()) {
            null
        } else {
            failure.toUiError(settingsConflicts)
        },
    )
}
