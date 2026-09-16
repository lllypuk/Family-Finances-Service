package tech.shatrov.familyfinances

import android.net.Uri
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

/** Картинки из share, галереи или камеры, ещё не взятые экраном распознавания. */
data class PendingImport(
    val id: UUID,
    val uris: List<Uri>,
)

/**
 * Владение импортом: предложенное отдаётся ровно одному экрану. Пока экран держит импорт
 * ([hold]) — готовит, распознаёт или сохраняет, — новый share ждёт и не публикуется в [pending].
 */
class ImportStore {
    private val lock = Any()
    private var offered: PendingImport? = null
    private val holders = mutableSetOf<UUID>()

    private val mutablePending = MutableStateFlow<UUID?>(null)
    private val mutableWaiting = MutableStateFlow(false)

    /** Импорт, который пора открыть; `null`, пока нечего открывать или текущий экран занят. */
    val pending: StateFlow<UUID?> = mutablePending.asStateFlow()

    /** Предложенный импорт ждёт, пока текущий экран закончит работу. */
    val waiting: StateFlow<Boolean> = mutableWaiting.asStateFlow()

    /** Новое предложение вытесняет невзятое прежнее: открывается последний share. */
    fun offer(uris: List<Uri>): UUID = synchronized(lock) {
        val id = UUID.randomUUID()
        offered = PendingImport(id, uris)
        publish()
        id
    }

    fun claim(id: UUID): PendingImport? = synchronized(lock) {
        val claimed = offered?.takeIf { it.id == id } ?: return null
        offered = null
        publish()
        claimed
    }

    fun hold(owner: UUID) = synchronized(lock) {
        holders += owner
        publish()
    }

    fun release(owner: UUID) = synchronized(lock) {
        holders -= owner
        publish()
    }

    private fun publish() {
        mutablePending.value = offered?.id?.takeIf { holders.isEmpty() }
        mutableWaiting.value = offered != null && holders.isNotEmpty()
    }
}
