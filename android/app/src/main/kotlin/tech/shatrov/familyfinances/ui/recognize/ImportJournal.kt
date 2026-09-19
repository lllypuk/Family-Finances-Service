package tech.shatrov.familyfinances.ui.recognize

import kotlinx.serialization.Serializable
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import tech.shatrov.familyfinances.core.api.RecognizeResult
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.net.apiSerializersModule
import java.io.File
import java.io.IOException
import java.util.UUID

const val JOURNAL_VERSION = 1
const val JOURNAL_TTL_MS = 24L * 60 * 60 * 1000

/** Состояние импорта на диске: переживает смерть процесса вместе с оплаченным ответом. */
@Serializable
data class ImportJournal(
    val version: Int = JOURNAL_VERSION,
    val importId: String,
    val updatedAt: Long,
    val images: List<JournalImage>,
    val dropped: Int,
    val recognizing: Boolean,
    val result: RecognizeResult?,
    val accountId: String?,
    val rows: List<JournalRow>,
    val savedCount: Int,
)

/** Строка URI — только ключ: права на него умирают с процессом. */
@Serializable
data class JournalImage(
    val uri: String,
    val file: String? = null,
    val failure: ImportFailure? = null,
)

/** Наложение на `result.items` по индексу; сумма и тип расходятся с ответом после `200` на `POST`. */
@Serializable
data class JournalRow(
    val draft: String,
    val included: Boolean,
    val date: String?,
    val categoryId: String?,
    val description: String,
    val status: JournalRowStatus,
    val dateAssumed: Boolean = false,
    val amountMinor: Long? = null,
    val type: TransactionType? = null,
)

@Serializable
enum class JournalRowStatus { PENDING, SAVING, SAVED, FAILED }

// `RecognizeResult` помечает даты и UUID `@Contextual`: без модуля он не сериализуется.
private val journalJson = Json {
    ignoreUnknownKeys = true
    serializersModule = apiSerializersModule
}

/**
 * Журналы в `<root>/<importId>/journal.json`; блокирующий — звать вне главного потока.
 * Нечитаемый, чужой версии или старше суток журнал читается как `null`: процесс живёт дольше `sweep`.
 */
class ImportJournalStore(
    private val root: File,
    private val now: () -> Long = System::currentTimeMillis,
) {
    private val lock = Any()

    // Удалённые в этом процессе: запись, начатая до удаления, не должна вернуть журнал.
    private val closed = mutableSetOf<String>()

    // Временный файл + rename: оборванная запись оставляет прежний журнал целым.
    fun write(journal: ImportJournal): Unit = synchronized(lock) {
        if (journal.importId in closed) return
        val dir = File(root, journal.importId).apply { mkdirs() }
        val tmp = File(dir, TMP)
        tmp.writeText(journalJson.encodeToString(ImportJournal.serializer(), journal))
        if (!tmp.renameTo(File(dir, FILE))) throw IOException("rename failed in $dir")
    }

    fun read(importId: UUID): ImportJournal? = readDir(File(root, importId.toString()))

    fun latest(): ImportJournal? = root.listFiles().orEmpty().mapNotNull(::readDir).maxByOrNull { it.updatedAt }

    fun delete(importId: UUID): Unit = synchronized(lock) {
        closed += importId.toString()
        File(root, importId.toString()).deleteRecursively()
    }

    fun deleteOthers(keep: UUID): Unit = synchronized(lock) {
        root.listFiles().orEmpty().filter { it.name != keep.toString() }.forEach {
            closed += it.name
            it.deleteRecursively()
        }
    }

    fun deleteAll(): Unit = synchronized(lock) {
        closed += root.listFiles().orEmpty().map { it.name }
        root.deleteRecursively()
    }

    private fun readDir(dir: File): ImportJournal? {
        val file = File(dir, FILE)
        if (!file.isFile) return null
        val journal = try {
            journalJson.decodeFromString(ImportJournal.serializer(), file.readText())
        } catch (_: IOException) {
            return null
        } catch (_: SerializationException) {
            return null
        } catch (_: IllegalArgumentException) {
            return null
        }
        return journal.takeIf {
            it.version == JOURNAL_VERSION && it.importId == dir.name && now() - it.updatedAt <= JOURNAL_TTL_MS
        }
    }

    private companion object {
        const val FILE = "journal.json"
        const val TMP = "journal.json.tmp"
    }
}
