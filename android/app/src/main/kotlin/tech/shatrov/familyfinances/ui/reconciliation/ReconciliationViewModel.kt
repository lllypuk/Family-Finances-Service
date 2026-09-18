package tech.shatrov.familyfinances.ui.reconciliation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.ReconciliationRequest
import tech.shatrov.familyfinances.core.api.ReconciliationRow
import tech.shatrov.familyfinances.core.api.ReconciliationStats
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import java.time.YearMonth
import java.util.UUID

sealed interface ReconciliationUiState {
    data object Loading : ReconciliationUiState

    data class Failure(val error: UiError) : ReconciliationUiState

    data class Ready(
        val month: YearMonth,
        val stats: ReconciliationStats,
    ) : ReconciliationUiState {
        val isEmpty: Boolean get() = stats.accounts.isEmpty()
    }
}

/** Лист «в банке»: цифра банка и заметка счёта [accountId] за [month]; [exists] — сверка уже записана. */
data class BankEditUiState(
    val accountId: UUID,
    val accountName: String,
    val month: YearMonth,
    val amount: String = "",
    val note: String = "",
    val exists: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
) {
    val amountMinor: Long? get() = parseAmountMinor(amount)

    val canSubmit: Boolean get() = amountMinor != null && !submitting
}

/** Сверка месяца: одна строка на счёт из `GET /stats/reconciliation`, правка — `PUT`/`DELETE` сверки. */
class ReconciliationViewModel(private val api: ApiGraph) : ViewModel() {
    private val mutable = MutableStateFlow<ReconciliationUiState>(ReconciliationUiState.Loading)
    private val mutableEditor = MutableStateFlow<BankEditUiState?>(null)

    val state: StateFlow<ReconciliationUiState> = mutable.asStateFlow()

    /** `null` — лист закрыт. */
    val editor: StateFlow<BankEditUiState?> = mutableEditor.asStateFlow()

    private var month: YearMonth? = null
    private var job: Job? = null

    /** Каждый заход перечитывает месяц: операции могли поменяться, пока экран был закрыт. */
    fun load(month: YearMonth) {
        this.month = month
        job?.cancel()
        mutable.value = ReconciliationUiState.Loading
        job = viewModelScope.launch {
            mutable.value = try {
                val stats = api.client.unwrap { api.stats.getReconciliationStats(month.toString()) }.`data`
                ReconciliationUiState.Ready(month, stats)
            } catch (failure: ApiFailure) {
                ReconciliationUiState.Failure(failure.toUiError())
            }
        }
    }

    fun refresh() {
        month?.let(::load)
    }

    fun onOpenBank(row: ReconciliationRow) {
        val ready = mutable.value as? ReconciliationUiState.Ready ?: return
        mutableEditor.value = BankEditUiState(
            accountId = row.account.id,
            accountName = row.account.name,
            month = ready.month,
            amount = row.bankExpenseMinor?.let(::formatAmountInput).orEmpty(),
            note = row.note.orEmpty(),
            exists = row.bankExpenseMinor != null,
        )
    }

    fun onAmountChange(amount: String) {
        mutableEditor.update { it?.copy(amount = amount, error = null) }
    }

    fun onNoteChange(note: String) {
        mutableEditor.update { it?.copy(note = note, error = null) }
    }

    fun onDismissBank() {
        if (mutableEditor.value?.submitting == true) return
        mutableEditor.value = null
    }

    fun onSave() {
        val current = mutableEditor.value ?: return
        val amount = current.amountMinor?.takeIf { !current.submitting } ?: return
        // Пустая заметка не отправляется: `PUT` — полная замена, и отсутствие поля её очищает.
        val request = ReconciliationRequest(bankExpenseMinor = amount, note = current.note.trim().ifEmpty { null })
        mutate(current) {
            api.client.unwrap { api.accounts.putReconciliation(current.accountId, current.month.toString(), request) }
        }
    }

    fun onDelete() {
        val current = mutableEditor.value?.takeIf { it.exists && !it.submitting } ?: return
        mutate(current) {
            try {
                api.client.send { api.accounts.deleteReconciliation(current.accountId, current.month.toString()) }
            } catch (failure: ApiFailure.Api) {
                // Сверку уже удалили с другого телефона: цель достигнута.
                if (!failure.isNotFound) throw failure
            }
        }
    }

    private fun mutate(
        current: BankEditUiState,
        call: suspend () -> Any,
    ) {
        mutableEditor.value = current.copy(submitting = true, error = null)
        viewModelScope.launch {
            try {
                call()
                mutableEditor.value = null
                load(current.month)
            } catch (failure: ApiFailure) {
                mutableEditor.update { it?.copy(submitting = false, error = failure.toUiError()) }
            }
        }
    }
}
