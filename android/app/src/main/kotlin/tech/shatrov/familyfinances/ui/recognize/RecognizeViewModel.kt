package tech.shatrov.familyfinances.ui.recognize

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.ImportStore
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CreateTransactionRequest
import tech.shatrov.familyfinances.core.api.RecognizedTransaction
import tech.shatrov.familyfinances.core.api.SimilarTransaction
import tech.shatrov.familyfinances.core.api.Transaction
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import tech.shatrov.familyfinances.ui.transactions.TransactionField
import tech.shatrov.familyfinances.ui.transactions.asCategoryType
import java.time.LocalDate
import java.util.UUID

private const val CATEGORY_LIMIT = 200
private const val MIN_DESCRIPTION = 2
private const val HTTP_CREATED = 201
private const val HTTP_UNPROCESSABLE = 422
private const val HTTP_SERVER_ERROR = 500

// Серверные тексты отказов распознавания английские, а `503` не различает «выключено» и «не отвечает».
private val recognizeFailures = mapOf(
    "RECOGNITION_UNAVAILABLE" to R.string.recognize_error_unavailable,
    "RECOGNITION_FAILED" to R.string.recognize_error_failed,
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
    val incomplete: Boolean = false,
    val savedCount: Int = 0,
) {
    val busy: Boolean
        get() = phase is RecognizePhase.Preparing ||
            phase is RecognizePhase.Recognizing ||
            saving

    val saving: Boolean
        get() = (phase as? RecognizePhase.Review)?.saving == true

    val toSave: Int
        get() = (phase as? RecognizePhase.Review)?.rows?.count { it.included && it.savable } ?: 0
}

/**
 * Импорт [importId]: картинки в JPEG, один платный вызов распознавания и сохранение выбранных строк.
 * Распознавание повторяется только по [retry]; импорт держится в [ImportStore], пока модель занята.
 */
class RecognizeViewModel(
    application: Application,
    private val api: ApiGraph,
    private val imports: ImportStore,
    private val importId: UUID,
    private val currency: String,
) : AndroidViewModel(application) {
    private val mutable = MutableStateFlow(RecognizeUiState())

    val state: StateFlow<RecognizeUiState> = mutable.asStateFlow()

    init {
        val claimed = imports.claim(importId)
        if (claimed == null) {
            mutable.value = RecognizeUiState(
                phase = RecognizePhase.Failure(UiError.Resource(R.string.recognize_import_lost), retryable = false),
            )
        } else {
            imports.hold(importId)
            viewModelScope.launch {
                // URI читаются сразу: права на чужой контент живут не дольше задачи, которая их выдала.
                val prepared = ImportFiles.prepare(getApplication(), importId, claimed.uris)
                mutable.update { it.copy(images = prepared.images, dropped = prepared.dropped) }
                recognize()
            }
        }
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

    fun onAmountChange(
        draft: UUID,
        amountMinor: Long,
    ) = edit(draft, TransactionField.AMOUNT) { it.copy(amountMinor = amountMinor, similarTo = emptyList()) }

    fun onTypeChange(
        draft: UUID,
        type: TransactionType,
    ) = edit(draft, TransactionField.TYPE) { row ->
        val keep = mutable.value.categories.any { it.id == row.categoryId && it.type == type.asCategoryType() }
        row.copy(type = type, categoryId = row.categoryId.takeIf { keep }, similarTo = emptyList())
    }

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

    fun save() = send(rows().filter { it.included && it.savable }.map { it.draft })

    fun retryRow(draft: UUID) = send(listOf(draft))

    override fun onCleared() {
        imports.release(importId)
        ImportFiles.discard(getApplication(), importId)
    }

    // Справочник — до платного вызова: его отказ не должен стоить распознавания.
    private suspend fun recognize() {
        try {
            val readyAt = mutable.value.images.indices.filter { mutable.value.images[it] is ImportImage.Ready }
            if (readyAt.isEmpty()) {
                fail(UiError.Resource(R.string.recognize_no_images), retryable = false)
                return
            }
            if (mutable.value.categories.isEmpty()) {
                val categories = api.client.unwrap { api.categories.listCategories(limit = CATEGORY_LIMIT) }.`data`
                mutable.update { it.copy(categories = categories) }
            }
            mutable.update { it.copy(phase = RecognizePhase.Recognizing) }
            val files = readyAt.map { (mutable.value.images[it] as ImportImage.Ready).file }
            val result = api.recognize(files).`data`
            mutable.update { state ->
                state.copy(
                    phase = RecognizePhase.Review(result.items.map { it.toRow(readyAt, currency) }),
                    incomplete = result.incomplete,
                )
            }
        } catch (failure: ApiFailure) {
            fail(
                failure.toUiError(recognizeFailures),
                retryable =
                failure !is ApiFailure.Api || failure.status >= HTTP_SERVER_ERROR,
            )
        } finally {
            imports.release(importId)
        }
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
            } finally {
                setSaving(false)
                imports.release(importId)
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
                    ),
                )
            }
            // `200` — запись с этим id уже была: показывается записанное, а не правка после обрыва.
            replace(draft) { if (code == HTTP_CREATED) it.copy(status = RowStatus.Saved) else it.saved(body.`data`) }
            mutable.update { it.copy(savedCount = it.savedCount + 1) }
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
