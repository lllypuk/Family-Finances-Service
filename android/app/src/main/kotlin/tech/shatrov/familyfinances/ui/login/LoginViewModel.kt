package tech.shatrov.familyfinances.ui.login

import android.content.res.Resources
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.LoginRequest
import tech.shatrov.familyfinances.core.api.auth.SessionToken
import tech.shatrov.familyfinances.core.api.net.ApiErrorCode
import tech.shatrov.familyfinances.core.api.net.ApiFailure

/**
 * Почему вход не прошёл. Виды разные, потому что чинятся по-разному: пароль набирают заново,
 * лимит пережидают, ненастроенный сервис — вообще не про пользователя.
 */
sealed interface LoginError {
    data object InvalidCredentials : LoginError

    /** `Retry-After` сервер шлёт заголовком, но может и не прислать. */
    data class RateLimited(val retryAfterSeconds: Int?) : LoginError

    /** `409 SETUP_REQUIRED`: семья на сервере ещё не создана. */
    data object SetupRequired : LoginError

    data object Network : LoginError

    /** Ответ не разобран: прокси отдал страницу, тело обрезано. */
    data object Malformed : LoginError

    /** Остальные коды показываются сообщением сервера как есть. */
    data class Server(val text: String) : LoginError
}

/** Перевод причины в текст: причина живёт во ViewModel, строки — в ресурсах. */
fun LoginError.message(res: Resources): String = when (this) {
    LoginError.InvalidCredentials -> res.getString(R.string.login_error_credentials)

    is LoginError.RateLimited ->
        if (retryAfterSeconds == null) {
            res.getString(R.string.login_error_rate_limited)
        } else {
            res.getString(R.string.login_error_rate_limited_wait, retryAfterSeconds)
        }

    LoginError.SetupRequired -> res.getString(R.string.login_error_setup_required)

    LoginError.Network -> res.getString(R.string.login_error_network)

    LoginError.Malformed -> res.getString(R.string.login_error_malformed)

    is LoginError.Server -> text
}

data class LoginUiState(
    val email: String = "",
    val password: String = "",
    val submitting: Boolean = false,
    val error: LoginError? = null,
    val signedIn: Boolean = false,
) {
    val canSubmit: Boolean get() = email.isNotBlank() && password.isNotEmpty() && !submitting
}

/** Единственный публичный маршрут: `POST /auth/login` и запись токена в хранилище. */
class LoginViewModel(private val api: ApiGraph) : ViewModel() {
    private val mutable = MutableStateFlow(LoginUiState())

    val state: StateFlow<LoginUiState> = mutable.asStateFlow()

    /** Правка полей гасит прошлую ошибку: она была про прошлую попытку. */
    fun onEmailChange(email: String) {
        mutable.update { it.copy(email = email, error = null) }
    }

    fun onPasswordChange(password: String) {
        mutable.update { it.copy(password = password, error = null) }
    }

    fun onSubmit() {
        val current = mutable.value
        if (!current.canSubmit) return
        mutable.update { it.copy(submitting = true, error = null) }
        viewModelScope.launch {
            try {
                val request = LoginRequest(email = current.email.trim(), password = current.password)
                val login = api.client.unwrap({ api.auth.login(request) }, { it.`data` })
                api.tokens.write(SessionToken(login.token, login.expiresAt))
                mutable.update { it.copy(submitting = false, password = "", signedIn = true) }
            } catch (failure: ApiFailure) {
                mutable.update { it.copy(submitting = false, error = failure.toLoginError()) }
            }
        }
    }
}

// Логин уходит без заголовка Authorization, поэтому его `401` — про пароль, а не про сессию.
private fun ApiFailure.toLoginError(): LoginError = when (this) {
    is ApiFailure.Api -> when (code) {
        ApiErrorCode.INVALID_CREDENTIALS -> LoginError.InvalidCredentials
        ApiErrorCode.RATE_LIMITED -> LoginError.RateLimited(retryAfterSeconds)
        ApiErrorCode.SETUP_REQUIRED -> LoginError.SetupRequired
        else -> LoginError.Server(serverMessage)
    }

    is ApiFailure.Network -> LoginError.Network

    is ApiFailure.Malformed -> LoginError.Malformed
}
