package tech.shatrov.familyfinances.ui.recognize

import android.app.Application
import android.net.Uri
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.NonCancellable
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import tech.shatrov.familyfinances.ImportStore
import tech.shatrov.familyfinances.LastAccountStore
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CreateTransactionRequest
import tech.shatrov.familyfinances.core.api.RecognizeResult
import tech.shatrov.familyfinances.core.api.RecognizedTransaction
import tech.shatrov.familyfinances.core.api.SimilarTransaction
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import tech.shatrov.familyfinances.ui.transactions.TransactionField
import java.io.File
import java.io.IOException
import java.time.LocalDate
import java.util.UUID

private const val CATEGORY_LIMIT = 200
private const val MIN_DESCRIPTION = 2
private const val HTTP_CREATED = 201
private const val HTTP_REQUEST_TIMEOUT = 408
private const val HTTP_UNPROCESSABLE = 422
private const val HTTP_SERVER_ERROR = 500

// Серверные тексты отказов распознавания английские, а `503` не различает «выключено» и «не отвечает».
private val recognizeFailures = mapOf(
    "RECOGNITION_UNAVAILABLE" to R.string.recognize_error_unavailable,
    "RECOGNITION_FAILED" to R.string.recognize_error_failed,
    "REQUEST_TIMEOUT" to R.string.recognize_error_upload_timeout,
)

private val rowFields = setOf(
    TransactionField.AMOUNT,
    TransactionField.TYPE,
    TransactionField.CATEGORY,
    TransactionField.DATE,
    TransactionField.DESCRIPTION,
)

sealed interface RowStatus {
    data object Pending : RowStatus

    data object Saving : RowStatus

    data object Saved : RowStatus

    data class Failed(val error: UiError) : RowStatus
}

/** Кандидат операции. [draft] — клиентский `id` его `POST`: повтор с ним не создаёт вторую запись. */
data class RecognizedRow(
    val draft: UUID,
    val image: Int?,
    val amountMinor: Long,
    val currency: String?,
    val currencyMismatch: Boolean,
    val type: TransactionType,
    val date: LocalDate?,
    val dateAssumed: Boolean,
    val description: String,
    val categoryId: UUID?,
    val similarTo: List<SimilarTransaction>,
    val included: Boolean = true,
    val status: RowStatus = RowStatus.Pending,
    val fieldErrors: Map<String, String> = emptyMap(),
) {
    val locked: Boolean
        get() = status == RowStatus.Saving || status == RowStatus.Saved

    val rejected: Boolean
        get() = status is RowStatus.Failed || fieldErrors.isNotEmpty()

    val savable: Boolean
        get() = !locked &&
            amountMinor > 0 &&
            date != null &&
            !dateAssumed &&
            !currencyMismatch &&
            categoryId != null &&
            description.trim().length >= MIN_DESCRIPTION
}

sealed interface RecognizePhase {
    data object Preparing : RecognizePhase

    data object Recognizing : RecognizePhase

    data class Review(
        val rows: List<RecognizedRow>,
        val saving: Boolean = false,
    ) : RecognizePhase

    data class Failure(
        val error: UiError,
        val retryable: Boolean,
    ) : RecognizePhase
}

/** [RecognizedRow.image] — индекс в [images], а не в отправленных: нечитаемые картинки не уходили. */
data class RecognizeUiState(
    val phase: RecognizePhase = RecognizePhase.Preparing,
    val images: List<ImportImage> = emptyList(),
    val dropped: Int = 0,
    val categories: List<Category> = emptyList(),
    val accounts: List<Account> = emptyList(),
    val accountId: UUID? = null,
    val incomplete: Boolean = false,
    val savedCount: Int = 0,
) {
    val saving: Boolean
        get() = (phase as? RecognizePhase.Review)?.saving == true

    val selectableAccounts: List<Account>
        get() = accounts.filter { !it.isArchived }

    /** Счёт один на пачку: после первой записанной строки остальные уходят с тем же. */
    val accountLocked: Boolean
        get() = saving || savedCount > 0

    val toSave: Int
        get() = (phase as? RecognizePhase.Review)?.rows?.count { it.included && it.savable } ?: 0
}

/**
 * Импорт [importId]: картинки в JPEG, один платный вызов распознавания и сохранение выбранных строк.
 * Распознавание повторяется только по [retry]; импорт держится в [ImportStore], пока модель занята.
 * Состояние пишется в [journals] и после смерти процесса восстанавливается без повторного вызова.
 */
class RecognizeViewModel(
    application: Application,
    private val api: ApiGraph,
    private val imports: ImportStore,
    private val journals: ImportJournalStore,
    private val importId: UUID,
    private val currency: String,
    private val lastAccount: LastAccountStore,
    private val io: CoroutineDispatcher = Dispatchers.IO,
) : AndroidViewModel(application) {
    private val mutable = MutableStateFlow(RecognizeUiState())

    val state: StateFlow<RecognizeUiState> = mutable.asStateFlow()

    private val writes = Channel<Unit>(Channel.CONFLATED)

    // Слитая запись и ожидаемая не должны разойтись: снимок и запись идут под одним замком.
    private val writeLock = Mutex()

    @Volatile private var result: RecognizeResult? = null

    @Volatile private var recognizing = false

    private var restored: ImportJournal? = null

    init {
        viewModelScope.launch(io) { for (ignored in writes) writeJournal() }
        val claimed = imports.claim(importId)
        imports.hold(importId)
        viewModelScope.launch { if (claimed != null) start(claimed.uris) else restore() }
    }

    fun retry() {
        val phase = mutable.value.phase
        if (phase !is RecognizePhase.Failure || !phase.retryable) return
        imports.hold(importId)
        mutable.update { it.copy(phase = RecognizePhase.Preparing) }
        viewModelScope.launch { recognize() }
    }

    fun onIncludedChange(
        draft: UUID,
        included: Boolean,
    ) = edit(draft, null) { it.copy(included = included) }

    /** Явный выбор даты, даже той же самой, подтверждает подставленный год. */
    fun onDateChange(
        draft: UUID,
        date: LocalDate,
    ) = edit(draft, TransactionField.DATE) { it.copy(date = date, dateAssumed = false, similarTo = emptyList()) }

    fun onCategoryChange(
        draft: UUID,
        categoryId: UUID,
    ) = edit(draft, TransactionField.CATEGORY) { it.copy(categoryId = categoryId) }

    fun onDescriptionChange(
        draft: UUID,
        description: String,
    ) = edit(draft, TransactionField.DESCRIPTION) { it.copy(description = description) }

    fun onAccountChange(accountId: UUID?) {
        mutable.update { if (it.accountLocked) it else it.copy(accountId = accountId) }
        persist()
    }

    fun save() = send(rows().filter { it.included && it.savable }.map { it.draft })

    fun retryRow(draft: UUID) = send(listOf(draft))

    /** Уход с экрана: импорт брошен, журнал и картинки удаляются. Зовётся из скоупа, переживающего модель. */
    suspend fun abandon() = withContext(NonCancellable + io) { journals.delete(importId) }

    // Не удаляет журнал: смахивание иногда доходит до `onCleared`, а журнал нужен именно тогда.
    override fun onCleared() {
        imports.release(importId)
    }

    private suspend fun start(uris: List<Uri>) {
        withContext(io) { journals.deleteOthers(keep = importId) }
        // URI читаются сразу: права на чужой контент живут не дольше задачи, которая их выдала.
        val prepared = ImportFiles.prepare(getApplication(), importId, uris)
        mutable.update { it.copy(images = prepared.images, dropped = prepared.dropped) }
        writeJournal()
        recognize()
    }

    private suspend fun restore() {
        val journal = withContext(io) { journals.read(importId) }
        if (journal == null) {
            imports.release(importId)
            fail(UiError.Resource(R.string.recognize_import_lost), retryable = false)
            return
        }
        restored = journal
        result = journal.result
        recognizing = journal.recognizing
        val dir = File(ImportFiles.filesRoot(getApplication()), importId.toString())
        mutable.update {
            it.copy(
                images = journal.images.map { image -> image.restore(dir) },
                dropped = journal.dropped,
                savedCount = journal.savedCount,
            )
        }
        if (journal.result == null && journal.recognizing) {
            // Исход прерванного платного вызова неизвестен: повторить его может только пользователь.
            imports.release(importId)
            fail(UiError.Resource(R.string.recognize_error_interrupted), retryable = true)
        } else {
            recognize()
        }
    }

    // Справочник — до платного вызова: его отказ не должен стоить распознавания.
    private suspend fun recognize() {
        var keepHold = false
        try {
            val readyAt = mutable.value.images.indices.filter { mutable.value.images[it] is ImportImage.Ready }
            if (readyAt.isEmpty()) {
                // Продолжать нечего: плашка на главной вела бы в тот же отказ.
                withContext(io) { journals.close(importId) }
                fail(UiError.Resource(R.string.recognize_no_images), retryable = false)
                return
            }
            if (mutable.value.categories.isEmpty()) loadCatalogs()
            val answer = result ?: paidCall(readyAt)
            // Ждущий share не вытесняет оплаченный ответ: импорт держится до сохранения или ухода.
            keepHold = answer.items.isNotEmpty() && imports.waiting.value
            val saved = restored?.rows.orEmpty()
            val rows = answer.items.mapIndexed { i, item ->
                item.toRow(readyAt, currency).let { row -> saved.getOrNull(i)?.let(row::restored) ?: row }
            }
            if (rows.isEmpty()) withContext(io) { journals.close(importId) } else writeJournal(rows)
            mutable.update { it.copy(phase = RecognizePhase.Review(rows), incomplete = answer.incomplete) }
        } catch (failure: ApiFailure) {
            fail(
                failure.toUiError(recognizeFailures),
                retryable = result != null ||
                    failure !is ApiFailure.Api ||
                    failure.status == HTTP_REQUEST_TIMEOUT ||
                    failure.status >= HTTP_SERVER_ERROR,
            )
        } finally {
            if (!keepHold) imports.release(importId)
        }
    }

    private suspend fun loadCatalogs() {
        val categories = api.client.unwrap { api.categories.listCategories(limit = CATEGORY_LIMIT) }.`data`
        val accounts = api.client.unwrap { api.accounts.listAccounts(limit = CATEGORY_LIMIT) }.`data`
        // Ответ в журнале — счёт уже выбирали: `null` там тоже выбор, а не повод взять прошлый.
        val journal = restored?.takeIf { it.result != null }
        val wanted = if (journal != null) journal.accountId?.let(UUID::fromString) else lastAccount.read()
        val account = wanted?.takeIf { id -> accounts.any { it.id == id } }
        mutable.update { it.copy(categories = categories, accounts = accounts, accountId = account) }
    }

    // `recognizing` остаётся взведённым и при отказе: дошёл ли запрос до модели, клиент не знает.
    private suspend fun paidCall(readyAt: List<Int>): RecognizeResult {
        mutable.update { it.copy(phase = RecognizePhase.Recognizing) }
        recognizing = true
        writeJournal()
        val files = readyAt.map { (mutable.value.images[it] as ImportImage.Ready).file }
        val answer = api.recognize(files).`data`
        result = answer
        recognizing = false
        return answer
    }

    private fun persist() {
        writes.trySend(Unit)
    }

    // Журнал — страховка: отказ диска не должен ронять импорт, который ещё жив в памяти.
    private suspend fun writeJournal(rows: List<RecognizedRow> = rows()) = withContext(io) {
        writeLock.withLock {
            try {
                journals.write(journal(rows))
            } catch (_: IOException) {
                // Следующая запись попробует снова.
            }
        }
    }

    private fun journal(rows: List<RecognizedRow>): ImportJournal {
        val state = mutable.value
        return ImportJournal(
            importId = importId.toString(),
            updatedAt = System.currentTimeMillis(),
            images = state.images.map { it.toJournal() },
            dropped = state.dropped,
            recognizing = recognizing,
            result = result,
            accountId = state.accountId?.toString(),
            rows = rows.map { it.toJournal() },
            savedCount = state.savedCount,
        )
    }

    private fun fail(
        error: UiError,
        retryable: Boolean,
    ) {
        mutable.update { it.copy(phase = RecognizePhase.Failure(error, retryable)) }
    }

    private fun send(drafts: List<UUID>) {
        val review = mutable.value.phase as? RecognizePhase.Review ?: return
        if (review.saving || drafts.isEmpty()) return
        setSaving(true)
        imports.hold(importId)
        viewModelScope.launch {
            try {
                drafts.forEach { saveRow(it) }
                val included = rows().filter { it.included }
                if (included.isNotEmpty() && included.all { it.status == RowStatus.Saved }) {
                    withContext(io) { journals.close(importId) }
                }
            } finally {
                setSaving(false)
                // Ждущий share не уносит строки с отказом (и `422` под полями), пока их не повторили или не ушли с экрана.
                val rejected = rows().any { it.included && it.rejected }
                if (!(rejected && imports.waiting.value)) imports.release(importId)
            }
        }
    }

    // Строка перечитывается на своей очереди: за время пачки её могли снять или поправить.
    private suspend fun saveRow(draft: UUID) {
        val row = rows().firstOrNull { it.draft == draft }?.takeIf { it.included && it.savable } ?: return
        val date = row.date ?: return
        val categoryId = row.categoryId ?: return
        replace(draft) { it.copy(status = RowStatus.Saving, fieldErrors = emptyMap()) }
        try {
            val (code, body) = api.client.unwrapWithCode {
                api.transactions.createTransaction(
                    CreateTransactionRequest(
                        amountMinor = row.amountMinor,
                        type = row.type,
                        description = row.description.trim(),
                        categoryId = categoryId,
                        date = date,
                        id = draft,
                        accountId = mutable.value.accountId,
                    ),
                )
            }
            lastAccount.write(mutable.value.accountId)
            // `200` — запись с этим id уже была: показывается записанное, а не правка после обрыва.
            replace(draft) { if (code == HTTP_CREATED) it.copy(status = RowStatus.Saved) else it.saved(body.`data`) }
            mutable.update { it.copy(savedCount = it.savedCount + 1) }
            persist()
        } catch (failure: ApiFailure) {
            replace(draft) { it.failed(failure) }
        }
    }

    private fun rows(): List<RecognizedRow> = (mutable.value.phase as? RecognizePhase.Review)?.rows.orEmpty()

    private fun setSaving(saving: Boolean) {
        mutable.update { state ->
            val review = state.phase as? RecognizePhase.Review ?: return@update state
            state.copy(phase = review.copy(saving = saving))
        }
    }

    private fun replace(
        draft: UUID,
        change: (RecognizedRow) -> RecognizedRow,
    ) {
        mutable.update { state ->
            val review = state.phase as? RecognizePhase.Review ?: return@update state
            state.copy(phase = review.copy(rows = review.rows.map { if (it.draft == draft) change(it) else it }))
        }
        persist()
    }

    private fun edit(
        draft: UUID,
        field: String?,
        change: (RecognizedRow) -> RecognizedRow,
    ) = replace(draft) { row ->
        if (row.locked) {
            row
        } else {
            change(row).let {
                if (field ==
                    null
                ) {
                    it
                } else {
                    it.copy(fieldErrors = it.fieldErrors - field)
                }
            }
        }
    }
}

private fun RecognizedTransaction.toRow(
    readyAt: List<Int>,
    familyCurrency: String,
): RecognizedRow = RecognizedRow(
    draft = UUID.randomUUID(),
    image = source?.let { readyAt.getOrNull(it) },
    amountMinor = amountMinor,
    currency = currency,
    currencyMismatch = currency != null && !currency.equals(familyCurrency, ignoreCase = true),
    type = type,
    date = date,
    dateAssumed = dateAssumed,
    description = description,
    categoryId = categoryId,
    similarTo = similar,
)

private fun RecognizedRow.saved(existing: Transaction): RecognizedRow = copy(
    amountMinor = existing.amountMinor,
    type = existing.type,
    date = existing.date,
    dateAssumed = false,
    description = existing.description,
    categoryId = existing.categoryId,
    similarTo = emptyList(),
    status = RowStatus.Saved,
)

// Как в форме операции: детали `422` ложатся под поля, строка остаётся доступной правке.
private fun RecognizedRow.failed(failure: ApiFailure): RecognizedRow {
    val details = (failure as? ApiFailure.Api)?.takeIf { it.status == HTTP_UNPROCESSABLE }?.details.orEmpty()
    val underFields = details.filter { it.`field` in rowFields }.associate { it.`field` to it.message }
    val allUnderFields = details.isNotEmpty() && underFields.size == details.size
    return copy(
        status = if (allUnderFields) RowStatus.Pending else RowStatus.Failed(failure.toUiError()),
        fieldErrors = underFields,
    )
}

private fun ImportImage.toJournal(): JournalImage = when (this) {
    is ImportImage.Ready -> JournalImage(uri.toString(), file = file.name)
    is ImportImage.Failed -> JournalImage(uri.toString(), failure = reason)
}

private fun JournalImage.restore(dir: File): ImportImage = if (file != null) {
    ImportImage.Ready(Uri.parse(uri), File(dir, file))
} else {
    ImportImage.Failed(Uri.parse(uri), failure ?: ImportFailure.UNREADABLE)
}

private fun RecognizedRow.toJournal(): JournalRow = JournalRow(
    draft = draft.toString(),
    included = included,
    date = date?.toString(),
    categoryId = categoryId?.toString(),
    description = description,
    status = when (status) {
        RowStatus.Pending -> JournalRowStatus.PENDING
        RowStatus.Saving -> JournalRowStatus.SAVING
        RowStatus.Saved -> JournalRowStatus.SAVED
        is RowStatus.Failed -> JournalRowStatus.FAILED
    },
    dateAssumed = dateAssumed,
    amountMinor = amountMinor,
    type = type,
)

// Прерванное сохранение повторяется тем же `draft`: `200` или `201`, дубля нет.
private fun RecognizedRow.restored(saved: JournalRow): RecognizedRow {
    val savedDate = saved.date?.let(LocalDate::parse)
    return copy(
        draft = UUID.fromString(saved.draft),
        included = saved.included,
        date = savedDate,
        dateAssumed = saved.dateAssumed,
        description = saved.description,
        categoryId = saved.categoryId?.let(UUID::fromString),
        amountMinor = saved.amountMinor ?: amountMinor,
        type = saved.type ?: type,
        similarTo = if (savedDate == date && saved.dateAssumed == dateAssumed) similarTo else emptyList(),
        status = when (saved.status) {
            JournalRowStatus.PENDING -> RowStatus.Pending

            JournalRowStatus.SAVED -> RowStatus.Saved

            JournalRowStatus.SAVING, JournalRowStatus.FAILED ->
                RowStatus.Failed(UiError.Resource(R.string.recognize_row_interrupted))
        },
    )
}
