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
import tech.shatrov.familyfinances.core.api.ChangePasswordRequest
import tech.shatrov.familyfinances.core.api.net.ApiErrorCode
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

/** Политика сервера — байты UTF-8, а не символы: «ёёёёё» это десять из десяти. */
const val PASSWORD_MIN_BYTES = 10
const val PASSWORD_MAX_BYTES = 72

private const val HTTP_UNAUTHORIZED = 401

/** Имена полей `error.details[].field` для `PUT /me/password`. */
object PasswordField {
    const val CURRENT = "current_password"
    const val NEW = "new_password"
}

private val passwordFields = setOf(PasswordField.CURRENT, PasswordField.NEW)

data class PasswordUiState(
    val current: String = "",
    val next: String = "",
    val repeat: String = "",
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    /** Сервер не принял текущий пароль: текст под полем свой, серверный говорит про вход. */
    val currentInvalid: Boolean = false,
    /** Сервер ответил `204`: поля чисты, на экране остаётся подтверждение. */
    val changed: Boolean = false,
) {
    val newLength: Int
        get() = next.toByteArray(Charsets.UTF_8).size

    val lengthInvalid: Boolean
        get() = next.isNotEmpty() && newLength !in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES

    val mismatch: Boolean
        get() = repeat.isNotEmpty() && repeat != next

    val canSubmit: Boolean
        get() = !submitting &&
            current.isNotEmpty() &&
            newLength in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES &&
            repeat == next
}

/**
 * Смена своего пароля. Текущая сессия переживает её, остальные сервер отзывает; повтора после
 * обрыва нет — запись могла пройти, и вторая попытка со старым паролем получила бы `401`.
 */
class PasswordViewModel(private val api: ApiGraph) : ViewModel() {
    private val mutable = MutableStateFlow(PasswordUiState())

    val state: StateFlow<PasswordUiState> = mutable.asStateFlow()

    fun onCurrentChange(value: String) {
        mutable.update {
            it.copy(current = value, changed = false, currentInvalid = false).cleared(PasswordField.CURRENT)
        }
    }

    fun onNewChange(value: String) {
        mutable.update { it.copy(next = value, changed = false).cleared(PasswordField.NEW) }
    }

    fun onRepeatChange(value: String) {
        mutable.update { it.copy(repeat = value, changed = false) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        mutable.update {
            it.copy(
                submitting = true,
                error = null,
                fieldErrors = emptyMap(),
                currentInvalid = false,
                changed = false,
            )
        }
        viewModelScope.launch {
            try {
                api.client.send {
                    api.me.changePassword(ChangePasswordRequest(current.current, current.next))
                }
                mutable.value = PasswordUiState(changed = true)
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }
}

private fun PasswordUiState.cleared(field: String): PasswordUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

/**
 * `401 INVALID_CREDENTIALS` — неверный текущий пароль, сессия жива (интерцептор её не гасит).
 * Обрыв, нечитаемый ответ и `5xx` оставляют результат неизвестным: сервер пишет хеш до отзыва
 * сессий, так что пароль мог уже смениться, и автоповтор тут не помогает.
 */
private fun PasswordUiState.failed(failure: ApiFailure): PasswordUiState {
    val rejected = failure as? ApiFailure.Api
    if (rejected != null &&
        rejected.status == HTTP_UNAUTHORIZED &&
        rejected.code == ApiErrorCode.INVALID_CREDENTIALS
    ) {
        return copy(submitting = false, currentInvalid = true)
    }
    val details = rejected?.details.orEmpty()
    val underFields = details.filter { it.`field` in passwordFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        error = when {
            failure.resultUnknown -> UiError.Resource(R.string.settings_password_unknown)
            underFields.size == details.size && underFields.isNotEmpty() -> null
            else -> failure.toUiError()
        },
    )
}
