package tech.shatrov.familyfinances.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.StatsSummary
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.recognize.ImportJournal
import tech.shatrov.familyfinances.ui.recognize.ImportJournalStore
import tech.shatrov.familyfinances.ui.toUiError
import java.time.Instant
import java.time.LocalDate
import java.time.YearMonth
import java.time.ZoneId
import java.time.ZonedDateTime
import java.util.UUID

sealed interface HomeUiState {
    data object Loading : HomeUiState

    data class Failure(val error: UiError) : HomeUiState

    data class Ready(
        val summary: StatsSummary,
        val currency: String,
    ) : HomeUiState {
        /** Ни одной операции за всё время: карточки с нулями рисовать незачем. */
        val isEmpty: Boolean get() = summary.transactionsTotal == 0
    }
}

/** Карточка сверки на главной; [Loading] и [Hidden] не рисуются. */
sealed interface ReconciliationCard {
    data object Loading : ReconciliationCard

    /** Пустая главная или отказ: сводка важнее, и карточка без неё не роняет экран. */
    data object Hidden : ReconciliationCard

    data object NoAccounts : ReconciliationCard

    /** [matched] из [total] неархивных счетов сошлись с банком. */
    data class Ready(
        val month: YearMonth,
        val matched: Int,
        val total: Int,
    ) : ReconciliationCard
}

/** Незавершённый импорт для плашки; [recognized] — оплаченный ответ уже в журнале. */
data class ImportDraft(
    val importId: UUID,
    val recognized: Boolean,
    val rows: Int,
    val saved: Int,
    val images: Int,
    val updatedAt: ZonedDateTime,
)

/** День, с которого карточка переходит на текущий месяц: до него сверяют закончившийся. */
private const val RECONCILE_CURRENT_FROM_DAY = 10

fun reconciliationMonth(today: LocalDate): YearMonth {
    val month = YearMonth.from(today)
    return if (today.dayOfMonth < RECONCILE_CURRENT_FROM_DAY) month.minusMonths(1) else month
}

/**
 * Главная — `GET /stats/summary`, за ним сверка месяца для карточки. Второй запрос идёт только
 * после сводки: пустой семье карточка не рисуется. Плашка импорта — из журнала на диске.
 */
class HomeViewModel(
    private val api: ApiGraph,
    private val currency: String,
    private val zone: ZoneId = ZoneId.systemDefault(),
    // Дата берётся на каждую сверку, а не один раз: модель живёт всю сессию приложения.
    private val today: () -> LocalDate = { LocalDate.now(zone) },
    private val journals: ImportJournalStore,
    private val io: CoroutineDispatcher = Dispatchers.IO,
) : ViewModel() {

    private val mutable = MutableStateFlow<HomeUiState>(HomeUiState.Loading)

    val state: StateFlow<HomeUiState> = mutable.asStateFlow()

    private val mutableCard = MutableStateFlow<ReconciliationCard>(ReconciliationCard.Loading)

    val card: StateFlow<ReconciliationCard> = mutableCard.asStateFlow()

    private val mutableImport = MutableStateFlow<ImportDraft?>(null)

    val import: StateFlow<ImportDraft?> = mutableImport.asStateFlow()

    private var requestedOn: LocalDate? = null
    private var job: Job? = null
    private var importJob: Job? = null

    init {
        refresh()
        loadImport()
    }

    /** На каждый заход: журнал пишет и удаляет экран распознавания, а модель живёт всю сессию. */
    fun loadImport() {
        importJob?.cancel()
        importJob = viewModelScope.launch {
            mutableImport.value = withContext(io) { journals.latest() }?.let(::draft)
        }
    }

    fun deleteImport() {
        val id = mutableImport.value?.importId ?: return
        importJob?.cancel()
        mutableImport.value = null
        // Отменённое чтение не должно вернуть плашку, а отмена — оставить журнал на диске.
        importJob = viewModelScope.launch { withContext(NonCancellable + io) { journals.delete(id) } }
    }

    private fun draft(journal: ImportJournal): ImportDraft? {
        val id = runCatching { UUID.fromString(journal.importId) }.getOrNull() ?: return null
        return ImportDraft(
            importId = id,
            recognized = journal.result != null,
            rows = journal.result?.items?.size ?: 0,
            saved = journal.savedCount,
            images = journal.images.size,
            updatedAt = Instant.ofEpochMilli(journal.updatedAt).atZone(zone),
        )
    }

    /**
     * Заход на экран и возврат из фона: сводка посчитана по «сегодня» семьи, и после полуночи
     * она уже за прошлый день, а на границе месяца — за прошлый месяц.
     * Идущий запрос сверяется по своей дате, а не по показанной сводке: полночь под ним иначе
     * осталась бы незамеченной, его ответ встал бы на экран за прошлые сутки, а второго
     * колбэка жизненного цикла не будет.
     */
    fun revalidate() {
        val ready = mutable.value as? HomeUiState.Ready
        val current = if (job?.isActive == true) requestedOn else ready?.summary?.to
        if (current == null || current == today()) return
        refresh()
    }

    fun refresh() {
        job?.cancel()
        val day = today()
        requestedOn = day
        mutable.value = HomeUiState.Loading
        mutableCard.value = ReconciliationCard.Loading
        job = viewModelScope.launch {
            val ready = try {
                // Границы периода не задаём: текущий месяц сервер считает в часовом поясе семьи
                // (A-06), а телефон может стоять в другом.
                val summary = api.client.unwrap { api.stats.getStatsSummary() }.`data`
                HomeUiState.Ready(summary, currency)
            } catch (failure: ApiFailure) {
                mutable.value = HomeUiState.Failure(failure.toUiError())
                mutableCard.value = ReconciliationCard.Hidden
                return@launch
            }
            mutable.value = ready
            mutableCard.value = if (ready.isEmpty) ReconciliationCard.Hidden else loadCard(reconciliationMonth(day))
        }
    }

    private suspend fun loadCard(month: YearMonth): ReconciliationCard = try {
        val rows = api.client
            .unwrap { api.stats.getReconciliationStats(month.toString()) }
            .`data`
            .accounts
            .filterNot { it.account.isArchived }
        if (rows.isEmpty()) {
            ReconciliationCard.NoAccounts
        } else {
            ReconciliationCard.Ready(month, matched = rows.count { it.diffMinor == 0L }, total = rows.size)
        }
    } catch (_: ApiFailure) {
        ReconciliationCard.Hidden
    }
}
