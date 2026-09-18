package tech.shatrov.familyfinances.ui.networth

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.HoldingValueRequest
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.LocalDate
import java.util.UUID

/**
 * Лист снимка позиции [holdingId]: сумма и дата, дата не позже [today] семьи. [existing] — правка
 * записанного снимка: дата у него ключ, её смена записала бы второй снимок, а не перенесла этот.
 */
data class ValueEditUiState(
    val holdingId: UUID,
    val name: String,
    val today: LocalDate,
    val amount: String = "",
    val date: LocalDate = today,
    val existing: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
) {
    val amountMinor: Long? get() = parseAmountMinor(amount)

    val canSubmit: Boolean get() = amountMinor != null && !submitting
}

/** Лист снимка для экрана капитала и истории позиции; после записи или удаления зовёт [onChanged]. */
internal class ValueEditor(
    private val api: ApiGraph,
    private val scope: CoroutineScope,
    private val onChanged: () -> Unit,
) {
    private val mutable = MutableStateFlow<ValueEditUiState?>(null)

    /** `null` — лист закрыт. */
    val state: StateFlow<ValueEditUiState?> = mutable.asStateFlow()

    fun open(state: ValueEditUiState) {
        mutable.value = state
    }

    fun onAmountChange(amount: String) {
        mutable.update { it?.copy(amount = amount, error = null) }
    }

    fun onDateChange(date: LocalDate) {
        mutable.update { if (it == null || it.existing) it else it.copy(date = minOf(date, it.today), error = null) }
    }

    fun onDismiss() {
        if (mutable.value?.submitting == true) return
        mutable.value = null
    }

    fun onSave() {
        val current = mutable.value ?: return
        val amount = current.amountMinor?.takeIf { !current.submitting } ?: return
        submit(current) {
            api.client.unwrap {
                api.holdings.putHoldingValue(current.holdingId, current.date, HoldingValueRequest(amount))
            }
        }
    }

    /** `404` здесь — снимка уже нет: цель удаления достигнута. */
    fun onDelete() {
        val current = mutable.value?.takeIf { it.existing && !it.submitting } ?: return
        submit(current) {
            api.client.send { api.holdings.deleteHoldingValue(current.holdingId, current.date) }
        }
    }

    private fun submit(
        current: ValueEditUiState,
        request: suspend () -> Any?,
    ) {
        mutable.value = current.copy(submitting = true, error = null)
        scope.launch {
            try {
                request()
                mutable.value = null
                onChanged()
            } catch (failure: ApiFailure) {
                // Позицию удалили с другого телефона: листу нечего сохранять, устарел список.
                if ((failure as? ApiFailure.Api)?.isNotFound == true) {
                    mutable.value = null
                    onChanged()
                } else {
                    mutable.update { it?.copy(submitting = false, error = failure.toUiError()) }
                }
            }
        }
    }
}
