package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import tech.shatrov.familyfinances.core.api.ErrorDetail

// Своя форма вместо сгенерированной `Error`: там `meta` обязательна, а её нет ни у ответа
// прокси, ни у обрезанного тела — разбор кода ошибки не должен от неё зависеть.
@Serializable
internal data class ErrorEnvelope(@SerialName("error") val error: ErrorEnvelopeBody)

@Serializable
internal data class ErrorEnvelopeBody(
    @SerialName("code") val code: String,
    @SerialName("message") val message: String,
    @SerialName("details") val details: List<ErrorDetail>? = null,
)
