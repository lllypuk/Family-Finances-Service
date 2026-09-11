package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.util.UUID

/** Пользователи берутся одной страницей: их в семье двое. */
private const val USER_LIMIT = 200

/** Строка списка; подписи «это вы» и «неактивен» подставляет экран. */
data class UserRow(
    val id: UUID,
    val name: String,
    val email: String,
    val admin: Boolean,
    val active: Boolean,
    val self: Boolean,
)

sealed interface UsersUiState {
    data object Loading : UsersUiState

    data class Failure(val error: UiError) : UsersUiState

    data class Ready(
        val rows: List<UserRow>,
        val total: Int,
    ) : UsersUiState {
        /** Страница одна: если сервер знает больше, полнота списка не утверждается. */
        val truncated: Boolean get() = total > rows.size
    }
}

/** Список пользователей семьи. Правка и создание живут в отдельной странице формы. */
class UsersViewModel(
    private val api: ApiGraph,
    private val selfId: UUID,
) : ViewModel() {
    private val mutable = MutableStateFlow<UsersUiState>(UsersUiState.Loading)

    val state: StateFlow<UsersUiState> = mutable.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        mutable.value = UsersUiState.Loading
        viewModelScope.launch {
            try {
                val page = api.client.unwrap { api.users.listUsers(limit = USER_LIMIT) }
                mutable.value = UsersUiState.Ready(
                    rows = page.`data`.map { it.row() },
                    total = page.meta.pagination.total,
                )
            } catch (failure: ApiFailure) {
                mutable.value = UsersUiState.Failure(failure.toUiError())
            }
        }
    }

    private fun User.row(): UserRow = UserRow(
        id = id,
        name = "$firstName $lastName",
        email = email,
        admin = role == Role.admin,
        active = isActive,
        self = id == selfId,
    )
}
