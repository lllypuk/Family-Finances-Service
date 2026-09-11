package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.UpdateFamilyRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

/** Имена полей формы — те же, что в `error.details[].field`. */
object FamilyField {
    const val NAME = "name"
    const val CURRENCY = "currency"
    const val TIMEZONE = "timezone"
}

private val familyFields = setOf(FamilyField.NAME, FamilyField.CURRENCY, FamilyField.TIMEZONE)

data class FamilyUiState(
    val loaded: Family,
    val name: String = loaded.name,
    val currency: String = loaded.currency,
    val timezone: String = loaded.timezone,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    /** Ответ `PUT /family`: сессию обновляет хост, модель о графе не знает. */
    val saved: Family? = null,
) {
    /** Тело `PUT` из одних изменённых полей: пустого сервер не принимает (`minProperties: 1`). */
    val changes: UpdateFamilyRequest?
        get() {
            val request = UpdateFamilyRequest(
                name = name.trim().takeIf { it != loaded.name },
                currency = currency.trim().takeIf { it != loaded.currency },
                timezone = timezone.trim().takeIf { it != loaded.timezone },
            )
            return request.takeIf { it != UpdateFamilyRequest() }
        }

    val canSubmit: Boolean
        get() = !submitting &&
            name.isNotBlank() &&
            currency.isNotBlank() &&
            timezone.isNotBlank() &&
            changes != null
}

/**
 * Семья: название, валюта и таймзона. Валюту проверяет сервер (`409 CURRENCY_LOCKED`), зону —
 * `time.LoadLocation`: клиентского списка зон и пикера здесь нет.
 */
class FamilyViewModel(
    private val api: ApiGraph,
    family: Family,
) : ViewModel() {
    private val mutable = MutableStateFlow(FamilyUiState(loaded = family))

    val state: StateFlow<FamilyUiState> = mutable.asStateFlow()

    fun onNameChange(name: String) {
        mutable.update { it.copy(name = name).cleared(FamilyField.NAME) }
    }

    fun onCurrencyChange(currency: String) {
        mutable.update { it.copy(currency = currency).cleared(FamilyField.CURRENCY) }
    }

    fun onTimezoneChange(timezone: String) {
        mutable.update { it.copy(timezone = timezone).cleared(FamilyField.TIMEZONE) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        val changes = current.changes ?: return
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            try {
                val saved = api.client.unwrap { api.family.updateFamily(changes) }.`data`
                mutable.update { it.copy(submitting = false, saved = saved) }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun FamilyUiState.cleared(field: String): FamilyUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

private fun FamilyUiState.failed(failure: ApiFailure): FamilyUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in familyFields }.associate { it.`field` to it.message }
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
