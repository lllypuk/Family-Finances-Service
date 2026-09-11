package tech.shatrov.familyfinances.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.CreateUserRequest
import tech.shatrov.familyfinances.core.api.PatchUserRequest
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.UpdateUserRequest
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError
import java.util.UUID

/** Имена полей формы — те же, что в `error.details[].field`: словарь перевода не нужен. */
object UserField {
    const val EMAIL = "email"
    const val FIRST_NAME = "first_name"
    const val LAST_NAME = "last_name"
    const val PASSWORD = "password"
}

private val userFields = setOf(
    UserField.EMAIL,
    UserField.FIRST_NAME,
    UserField.LAST_NAME,
    UserField.PASSWORD,
)

/** Куда уходит форма после удачной правки; `null` — остаётся на месте. */
enum class UserEditExit {
    /** Создание закончено: назад в список. */
    List,

    /** Админ понизил себя: `/users` ему больше не отвечает. */
    Root,
}

data class UserEditUiState(
    val id: UUID?,
    val self: Boolean = false,
    val loading: Boolean = false,
    val loadError: UiError? = null,
    val loaded: User? = null,
    val email: String = "",
    val firstName: String = "",
    val lastName: String = "",
    val password: String = "",
    /** Роль формы создания; у существующей записи роль меняется отдельным `PATCH`. */
    val role: Role = Role.member,
    val submitting: Boolean = false,
    val error: UiError? = null,
    val fieldErrors: Map<String, String> = emptyMap(),
    /** Запись, которую вернул сервер: свою хост кладёт в сессию, чужой помечает список. */
    val saved: User? = null,
    val exit: UserEditExit? = null,
) {
    val creating: Boolean get() = id == null

    val currentRole: Role get() = loaded?.role ?: role

    val active: Boolean get() = loaded?.isActive != false

    val passwordLength: Int get() = password.toByteArray(Charsets.UTF_8).size

    val passwordInvalid: Boolean
        get() = password.isNotEmpty() && passwordLength !in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES

    /** Тело `PUT` из одних изменённых полей: пустого сервер не принимает (`minProperties: 1`). */
    val changes: UpdateUserRequest?
        get() {
            val user = loaded ?: return null
            val request = UpdateUserRequest(
                email = email.trim().takeIf { it != user.email },
                firstName = firstName.trim().takeIf { it != user.firstName },
                lastName = lastName.trim().takeIf { it != user.lastName },
            )
            return request.takeIf { it != UpdateUserRequest() }
        }

    val canSubmit: Boolean
        get() = !submitting &&
            email.isNotBlank() &&
            firstName.isNotBlank() &&
            lastName.isNotBlank() &&
            if (creating) passwordLength in PASSWORD_MIN_BYTES..PASSWORD_MAX_BYTES else changes != null

    /** Себя не деактивируют и себе не задают пароль: своя смена живёт в «Пароле». */
    val ownActions: Boolean get() = !creating && !self
}

/**
 * Форма пользователя: создание, правка имени и почты, роль и активность отдельными `PATCH`.
 * После каждого ответа запись перечитывается — `is_active` мог измениться другим шагом того же
 * запроса или вторым телефоном.
 */
class UserEditViewModel(
    private val api: ApiGraph,
    private val id: UUID?,
    selfId: UUID,
) : ViewModel() {
    private val mutable = MutableStateFlow(
        UserEditUiState(id = id, self = id == selfId, loading = id != null),
    )

    val state: StateFlow<UserEditUiState> = mutable.asStateFlow()

    init {
        if (id != null) load()
    }

    fun load() {
        val target = id ?: return
        mutable.update { it.copy(loading = true, loadError = null) }
        viewModelScope.launch {
            try {
                val user = api.client.unwrap { api.users.getUser(target) }.`data`
                mutable.update { it.withUser(user, notify = false).copy(loading = false) }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(loading = false, loadError = failure.toUiError(settingsConflicts)) }
            }
        }
    }

    fun onEmailChange(value: String) {
        mutable.update { it.copy(email = value).cleared(UserField.EMAIL) }
    }

    fun onFirstNameChange(value: String) {
        mutable.update { it.copy(firstName = value).cleared(UserField.FIRST_NAME) }
    }

    fun onLastNameChange(value: String) {
        mutable.update { it.copy(lastName = value).cleared(UserField.LAST_NAME) }
    }

    fun onPasswordChange(value: String) {
        mutable.update { it.copy(password = value).cleared(UserField.PASSWORD) }
    }

    fun onRoleChange(value: Role) {
        mutable.update { it.copy(role = value) }
    }

    fun onSubmit() {
        if (mutable.value.creating) create() else save()
    }

    /** Смена роли: у своей записи понижение выкидывает в корень настроек — прав на список нет. */
    fun onToggleRole() {
        val current = mutable.value
        val target = id ?: return
        if (current.submitting) return
        val next = if (current.currentRole == Role.admin) Role.member else Role.admin
        patch(target, PatchUserRequest(role = next), leaving = current.self && next == Role.member)
    }

    fun onToggleActive() {
        val current = mutable.value
        val target = id ?: return
        if (current.submitting || !current.ownActions) return
        patch(target, PatchUserRequest(isActive = !current.active), leaving = false)
    }

    private fun create() {
        val form = mutable.value
        if (!form.canSubmit) return
        val request = CreateUserRequest(
            email = form.email.trim(),
            password = form.password,
            firstName = form.firstName.trim(),
            lastName = form.lastName.trim(),
            role = form.role,
        )
        started()
        viewModelScope.launch {
            try {
                val created = api.client.unwrap { api.users.createUser(request) }.`data`
                mutable.update {
                    it.withUser(created, notify = true).copy(submitting = false, exit = UserEditExit.List)
                }
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    private fun save() {
        val form = mutable.value
        val target = id ?: return
        val changes = form.changes ?: return
        if (!form.canSubmit) return
        started()
        viewModelScope.launch {
            try {
                val saved = api.client.unwrap { api.users.updateUser(target, changes) }.`data`
                mutable.update { it.withUser(saved, notify = true).copy(submitting = false) }
                reload(target)
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
            }
        }
    }

    private fun patch(
        target: UUID,
        request: PatchUserRequest,
        leaving: Boolean,
    ) {
        started()
        viewModelScope.launch {
            try {
                val saved = api.client.unwrap { api.users.patchUser(target, request) }.`data`
                mutable.update {
                    it.withUser(saved, notify = true).copy(
                        submitting = false,
                        exit = if (leaving) UserEditExit.Root else null,
                    )
                }
                if (!leaving) reload(target)
            } catch (failure: ApiFailure) {
                mutable.value = mutable.value.failed(failure)
                // Отказ мог прийти вторым шагом запроса: роль уже записана, активность нет.
                reload(target)
            }
        }
    }

    private fun started() {
        mutable.update { it.copy(submitting = true, error = null, fieldErrors = emptyMap()) }
    }

    /** Перечитка после ответа: её собственный отказ поверх результата правки не показывается. */
    private suspend fun reload(target: UUID) {
        try {
            val fresh = api.client.unwrap { api.users.getUser(target) }.`data`
            mutable.update { it.withUser(fresh, notify = true) }
        } catch (failure: ApiFailure) {
            return
        }
    }
}

/** Правка поля гасит ошибку под ним: она была про прошлую попытку. */
private fun UserEditUiState.cleared(field: String): UserEditUiState =
    if (fieldErrors.containsKey(field)) copy(fieldErrors = fieldErrors - field) else this

/**
 * Запись с сервера становится и формой, и базой для следующего diff. [notify] отделяет ответ на
 * правку от первой загрузки: сессию и список трогает только правка.
 */
private fun UserEditUiState.withUser(
    user: User,
    notify: Boolean,
): UserEditUiState = copy(
    loaded = user,
    email = user.email,
    firstName = user.firstName,
    lastName = user.lastName,
    role = user.role,
    saved = if (notify) user else saved,
)

private fun UserEditUiState.failed(failure: ApiFailure): UserEditUiState {
    val details = (failure as? ApiFailure.Api)?.details.orEmpty()
    val underFields = details.filter { it.`field` in userFields }.associate { it.`field` to it.message }
    return copy(
        submitting = false,
        fieldErrors = underFields,
        // Деталь не про поле формы (`field: "body"`) осталась бы без текста — показываем её общим.
        error = if (underFields.size == details.size && underFields.isNotEmpty()) {
            null
        } else {
            failure.toUiError(settingsConflicts)
        },
    )
}
