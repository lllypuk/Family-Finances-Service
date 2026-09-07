package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.json.Json
import tech.shatrov.familyfinances.core.api.ErrorDetail
import java.io.IOException

/** Коды из `error.code`, по которым ветвится клиент. Остальные показываются как есть. */
object ApiErrorCode {
    const val UNAUTHORIZED = "UNAUTHORIZED"
    const val INVALID_CREDENTIALS = "INVALID_CREDENTIALS"
    const val FORBIDDEN = "FORBIDDEN"
    const val RATE_LIMITED = "RATE_LIMITED"
    const val VALIDATION_ERROR = "VALIDATION_ERROR"
    const val SETUP_REQUIRED = "SETUP_REQUIRED"
}

/** Единственный тип отказа, который видят вызывающие: любой другой не выходит за `ApiClient`. */
sealed class ApiFailure(
    message: String,
    cause: Throwable?,
) : Exception(message, cause) {
    /** Сервер ответил конвертом ошибки. */
    class Api(
        val status: Int,
        val code: String,
        val serverMessage: String,
        val details: List<ErrorDetail> = emptyList(),
        val retryAfterSeconds: Int? = null,
    ) : ApiFailure("HTTP $status $code: $serverMessage", null) {
        val isUnauthorized: Boolean get() = status == HTTP_UNAUTHORIZED
        val isSetupRequired: Boolean get() = code == ApiErrorCode.SETUP_REQUIRED
        val isServerError: Boolean get() = status >= HTTP_SERVER_ERROR
    }

    /** Ответ пришёл, но конверта в нём нет: страница прокси, обрезанное тело, пустой `5xx`. */
    class Malformed(
        val status: Int?,
        cause: Throwable?,
    ) : ApiFailure("ответ не разобран" + (status?.let { " (HTTP $it)" } ?: ""), cause)

    /** Соединения не было или оно оборвалось. */
    class Network(cause: IOException) : ApiFailure(cause.message ?: "сеть недоступна", cause)

    private companion object {
        const val HTTP_UNAUTHORIZED = 401
        const val HTTP_SERVER_ERROR = 500
    }
}

// Разбор тела ошибки в одном месте на все операции: `Retry-After` приходит заголовком, а не в теле.
internal fun parseApiFailure(
    status: Int,
    body: String?,
    retryAfter: String?,
    json: Json,
): ApiFailure {
    val envelope = body?.takeIf { it.isNotBlank() }?.let {
        try {
            json.decodeFromString(ErrorEnvelope.serializer(), it)
        } catch (e: IllegalArgumentException) {
            return ApiFailure.Malformed(status, e)
        }
    } ?: return ApiFailure.Malformed(status, null)

    return ApiFailure.Api(
        status = status,
        code = envelope.error.code,
        serverMessage = envelope.error.message,
        details = envelope.error.details.orEmpty(),
        retryAfterSeconds = retryAfter?.trim()?.toIntOrNull(),
    )
}
