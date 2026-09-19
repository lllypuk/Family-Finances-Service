package tech.shatrov.familyfinances.ui.reconciliation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.AccountBalanceRequest
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.ReconciliationRow
import tech.shatrov.familyfinances.core.api.ReconciliationStats
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatAmountInput
import tech.shatrov.familyfinances.ui.format.parseSignedAmountMinor
import tech.shatrov.familyfinances.ui.toUiError
import tech.shatrov.familyfinances.ui.transactions.TransactionPrefill
import java.time.LocalDate
import java.time.YearMonth
import java.util.UUID
import kotlin.math.abs

/** Что подписать под разницей: знак `gap` словами, а не плюсом и минусом. */
enum class GapVerdict { MATCHED, MISSING_INCOME, MISSING_EXPENSE }

/** `gap > 0` — остатки выросли больше записанного: не записан приход или записан лишний расход. */
fun gapVerdict(gap: Long): GapVerdict = when {
    gap == 0L -> GapVerdict.MATCHED
    gap > 0 -> GapVerdict.MISSING_INCOME
    else -> GapVerdict.MISSING_EXPENSE
}

/**
 * «Закрыть разницу»: операция на `|gap|` последним днём [month], но не позже [today] —
 * будущую дату сервер отвергает, а текущий месяц ещё не кончился.
 */
fun gapCorrection(
    month: YearMonth,
    gap: Long,
    today: LocalDate,
    description: String,
): TransactionPrefill = TransactionPrefill(
    amountMinor = abs(gap),
    type = if (gap > 0) TransactionType.income else TransactionType.expense,
    date = minOf(month.atEndOfMonth(), today),
    description = description,
)

/** Итог месяца; пока край не заполнен, сравнивать нечего — экран зовёт его заполнить. */
sealed interface ReconciliationTotal {
    data class Complete(
        val openingMinor: Long,
        val closingMinor: Long,
        val gapMinor: Long,
    ) : ReconciliationTotal

    data class MissingClosing(val accounts: Int) : ReconciliationTotal

    data class MissingOpening(val prev: YearMonth) : ReconciliationTotal
}

sealed interface ReconciliationUiState {
    data object Loading : ReconciliationUiState

    data class Failure(val error: UiError) : ReconciliationUiState

    data class Ready(
        val month: YearMonth,
        val stats: ReconciliationStats,
    ) : ReconciliationUiState {
        val isEmpty: Boolean get() = stats.accounts.isEmpty()

        val total: ReconciliationTotal get() {
            val opening = stats.openingMinor
            val closing = stats.closingMinor
            val gap = stats.gapMinor
            return when {
                closing == null -> ReconciliationTotal.MissingClosing(stats.accounts.count { it.closingMinor == null })
                opening == null || gap == null -> ReconciliationTotal.MissingOpening(month.minusMonths(1))
                else -> ReconciliationTotal.Complete(opening, closing, gap)
            }
        }
    }
}

/** Диалог остатка счёта [accountId] на конец [month]; [exists] — остаток уже записан. */
data class BalanceEditUiState(
    val accountId: UUID,
    val accountName: String,
    val month: YearMonth,
    val amount: String = "",
    val exists: Boolean = false,
    val submitting: Boolean = false,
    val error: UiError? = null,
) {
    val amountMinor: Long? get() = parseSignedAmountMinor(amount)

    val canSubmit: Boolean get() = amountMinor != null && !submitting
}

/** Сверка остатков месяца из `GET /stats/reconciliation`, правка — `PUT`/`DELETE` остатка счёта. */
class ReconciliationViewModel(private val api: ApiGraph) : ViewModel() {
    private val mutable = MutableStateFlow<ReconciliationUiState>(ReconciliationUiState.Loading)
    private val mutableEditor = MutableStateFlow<BalanceEditUiState?>(null)

    val state: StateFlow<ReconciliationUiState> = mutable.asStateFlow()

    /** `null` — диалог закрыт. */
    val editor: StateFlow<BalanceEditUiState?> = mutableEditor.asStateFlow()

    private var month: YearMonth? = null
    private var job: Job? = null

    /** Каждый заход перечитывает месяц: операции могли поменяться. Будущий месяц сервер не примет — обрезает экран. */
    fun load(target: YearMonth) {
        this.month = target
        job?.cancel()
        mutable.value = ReconciliationUiState.Loading
        job = viewModelScope.launch {
            mutable.value = try {
                val stats = api.client.unwrap { api.stats.getReconciliationStats(target.toString()) }.`data`
                ReconciliationUiState.Ready(target, stats)
            } catch (failure: ApiFailure) {
                ReconciliationUiState.Failure(failure.toUiError())
            }
        }
    }

    fun refresh() {
        month?.let(::load)
    }

    fun onOpenBalance(row: ReconciliationRow) {
        val ready = mutable.value as? ReconciliationUiState.Ready ?: return
        mutableEditor.value = BalanceEditUiState(
            accountId = row.account.id,
            accountName = row.account.name,
            month = ready.month,
            amount = row.closingMinor?.let(::formatAmountInput).orEmpty(),
            exists = row.closingMinor != null,
        )
    }

    /**
     * Остаток на конец прошлого месяца: счёт, заведённый в этом, в прошлом не показан, а его `opening` без строки — 0.
     * Есть ли строка, по такому 0 не понять, поэтому «Очистить» здесь нет.
     */
    fun onOpenOpening(row: ReconciliationRow) {
        val ready = mutable.value as? ReconciliationUiState.Ready ?: return
        mutableEditor.value = BalanceEditUiState(
            accountId = row.account.id,
            accountName = row.account.name,
            month = ready.month.minusMonths(1),
            amount = row.openingMinor?.let(::formatAmountInput).orEmpty(),
        )
    }

    fun onAmountChange(amount: String) {
        mutableEditor.update { it?.copy(amount = amount, error = null) }
    }

    /** Минуса нет на цифровой клавиатуре многих телефонов, поэтому знак переключается кнопкой. */
    fun onToggleSign() {
        mutableEditor.update { current ->
            current?.copy(
                amount = if (current.amount.startsWith('-')) current.amount.drop(1) else "-${current.amount}",
                error = null,
            )
        }
    }

    fun onDismissBalance() {
        if (mutableEditor.value?.submitting == true) return
        mutableEditor.value = null
    }

    fun onSave() {
        val current = mutableEditor.value ?: return
        val amount = current.amountMinor?.takeIf { !current.submitting } ?: return
        mutate(current) {
            api.client.unwrap {
                api.accounts.putAccountBalance(
                    current.accountId,
                    current.month.toString(),
                    AccountBalanceRequest(balanceMinor = amount),
                )
            }
        }
    }

    fun onClear() {
        val current = mutableEditor.value?.takeIf { it.exists && !it.submitting } ?: return
        mutate(current) {
            try {
                api.client.send { api.accounts.deleteAccountBalance(current.accountId, current.month.toString()) }
            } catch (failure: ApiFailure.Api) {
                // Остаток уже удалили с другого телефона: цель достигнута.
                if (!failure.isNotFound) throw failure
            }
        }
    }

    private fun mutate(
        current: BalanceEditUiState,
        call: suspend () -> Any,
    ) {
        mutableEditor.value = current.copy(submitting = true, error = null)
        viewModelScope.launch {
            try {
                call()
                mutableEditor.value = null
                refresh()
            } catch (failure: ApiFailure) {
                mutableEditor.update { it?.copy(submitting = false, error = failure.toUiError()) }
            }
        }
    }
}
