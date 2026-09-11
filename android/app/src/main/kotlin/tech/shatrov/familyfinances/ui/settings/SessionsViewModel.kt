package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatDateTime
import tech.shatrov.familyfinances.ui.toUiError
import java.time.ZoneId
import java.util.UUID
import tech.shatrov.familyfinances.core.api.Session as ApiSession

/** Сессии берутся одной страницей: их у пользователя единицы. */
private const val SESSION_LIMIT = 200

private const val HTTP_NOT_FOUND = 404

/** Строка списка: даты уже в зоне семьи, «Без имени» подставляет экран. */
data class SessionRow(
    val id: UUID,
    val deviceName: String?,
    val created: String,
    val lastUsed: String,
    val current: Boolean,
)

sealed interface SessionsUiState {
    data object Loading : SessionsUiState

    data class Failure(val error: UiError) : SessionsUiState

    data class Ready(
        val rows: List<SessionRow>,
        val total: Int,
        val revoking: UUID? = null,
        val error: UiError? = null,
    ) : SessionsUiState {
        /** Страница одна: если сервер знает больше, полнота списка не утверждается. */
        val truncated: Boolean get() = total > rows.size
    }

    /** Уход со страницы во время отзыва запрещён: очистка store отменила бы `DELETE`. */
    val busy: Boolean get() = this is Ready && revoking != null
}

/** Список своих сессий с отзывом чужих. Текущая отзывается только «Выйти» в корне настроек. */
class SessionsViewModel(
    private val api: ApiGraph,
    private val zone: ZoneId,
) : ViewModel() {
    private val mutable = MutableStateFlow<SessionsUiState>(SessionsUiState.Loading)

    val state: StateFlow<SessionsUiState> = mutable.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = SessionsUiState.Loading
        viewModelScope.launch {
            try {
                val page = api.client.unwrap { api.auth.listSessions(limit = SESSION_LIMIT) }
                mutable.value = SessionsUiState.Ready(
                    rows = page.`data`.map { it.row() },
                    total = page.meta.pagination.total,
                )
            } catch (failure: ApiFailure) {
                mutable.value = SessionsUiState.Failure(failure.toUiError())
            }
        }
    }

    fun onRevoke(id: UUID) {
        val current = mutable.value as? SessionsUiState.Ready ?: return
        if (current.revoking != null) return
        val row = current.rows.firstOrNull { it.id == id } ?: return
        if (row.current) return
        mutable.value = current.copy(revoking = id, error = null)
        viewModelScope.launch {
            try {
                api.client.send { api.auth.revokeSession(id) }
                refresh()
            } catch (failure: ApiFailure) {
                // Сессии уже нет — отзывать нечего, устарел список.
                if ((failure as? ApiFailure.Api)?.status == HTTP_NOT_FOUND) {
                    refresh()
                } else {
                    mutable.value = current.copy(revoking = null, error = failure.toUiError())
                }
            }
        }
    }

    private fun ApiSession.row(): SessionRow = SessionRow(
        id = id,
        deviceName = deviceName?.takeIf { it.isNotBlank() },
        created = formatDateTime(createdAt, zone),
        lastUsed = formatDateTime(lastUsedAt, zone),
        current = current,
    )
}
