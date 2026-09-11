package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Backup
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatBytes
import tech.shatrov.familyfinances.ui.format.formatDateTime
import tech.shatrov.familyfinances.ui.toUiError
import java.time.ZoneId

/** Бэкапов на диске десятки (`BACKUP_KEEP` 30): одной страницы хватает. */
private const val BACKUP_LIMIT = 200

private const val HTTP_NOT_FOUND = 404
private const val HTTP_SERVER_ERROR = 500

/** Строка списка: размер и дата уже отформатированы, имя — ключ для удаления. */
data class BackupRow(
    val name: String,
    val size: String,
    val created: String,
)

sealed interface BackupsUiState {
    data object Loading : BackupsUiState

    data class Failure(val error: UiError) : BackupsUiState

    data class Ready(
        val rows: List<BackupRow>,
        val total: Int,
        val creating: Boolean = false,
        val deleting: String? = null,
        val error: UiError? = null,
    ) : BackupsUiState {
        /** Страница одна: если сервер знает больше, полнота списка не утверждается. */
        val truncated: Boolean get() = total > rows.size
    }

    /** Уход со страницы во время записи запрещён: очистка store отменила бы запрос. */
    val busy: Boolean get() = this is Ready && (creating || deleting != null)
}

/**
 * Список бэкапов с созданием и удалением. Скачивания нет: файлу базы на телефоне делать нечего,
 * восстановление — по ssh (A-11).
 */
class BackupsViewModel(
    private val api: ApiGraph,
    private val zone: ZoneId,
) : ViewModel() {
    private val mutable = MutableStateFlow<BackupsUiState>(BackupsUiState.Loading)

    val state: StateFlow<BackupsUiState> = mutable.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        reload(keep = null)
    }

    fun onCreate() {
        val current = mutable.value as? BackupsUiState.Ready ?: return
        if (current.busy) return
        mutable.value = current.copy(creating = true, error = null)
        viewModelScope.launch {
            try {
                api.client.unwrap { api.backups.createBackup() }
                // Retention мог удалить старые файлы, поэтому список перечитывается целиком.
                reload(keep = null)
            } catch (failure: ApiFailure) {
                if (failure.resultUnknown) {
                    reload(keep = UiError.Resource(R.string.settings_backup_unknown))
                } else {
                    mutable.value = current.copy(creating = false, error = failure.toUiError())
                }
            }
        }
    }

    fun onDelete(name: String) {
        val current = mutable.value as? BackupsUiState.Ready ?: return
        if (current.busy) return
        mutable.value = current.copy(deleting = name, error = null)
        viewModelScope.launch {
            try {
                api.client.send { api.backups.deleteBackup(name) }
                reload(keep = null)
            } catch (failure: ApiFailure) {
                // Файла уже нет — удалять нечего, устарел список.
                if ((failure as? ApiFailure.Api)?.status == HTTP_NOT_FOUND) {
                    reload(keep = null)
                } else {
                    mutable.value = current.copy(deleting = null, error = failure.toUiError())
                }
            }
        }
    }

    private fun reload(keep: UiError?) {
        mutable.value = BackupsUiState.Loading
        viewModelScope.launch {
            try {
                val page = api.client.unwrap { api.backups.listBackups(limit = BACKUP_LIMIT) }
                mutable.value = BackupsUiState.Ready(
                    rows = page.`data`.map { it.row() },
                    total = page.meta.pagination.total,
                    error = keep,
                )
            } catch (failure: ApiFailure) {
                mutable.value = BackupsUiState.Failure(keep ?: failure.toUiError())
            }
        }
    }

    private fun Backup.row(): BackupRow = BackupRow(
        name = name,
        size = formatBytes(sizeBytes),
        created = formatDateTime(createdAt, zone),
    )
}

/**
 * Создание бэкапа не идемпотентно и не имеет клиентского ключа: после обрыва, неразобранного
 * ответа или `5xx` файл мог появиться, поэтому повтор — только руками.
 */
private val ApiFailure.resultUnknown: Boolean
    get() = this !is ApiFailure.Api || status >= HTTP_SERVER_ERROR
