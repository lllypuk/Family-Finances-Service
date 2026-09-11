package tech.shatrov.familyfinances.ui

import android.content.res.Resources
import androidx.annotation.StringRes
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.net.ApiFailure

/** Почему запрос не удался. `401` сюда не доходит: интерцептор уводит на экран входа сам. */
sealed interface UiError {
    data object Network : UiError

    data object Malformed : UiError

    data class Server(val text: String) : UiError

    /** Отказ, который сервер не даёт различить по коду: текст выбирает клиент. */
    data class Resource(@StringRes val id: Int) : UiError
}

/**
 * Перевод отказа в текст. [known] отображает `error.code` в строку ресурса для кодов, чей
 * серверный текст на экране не годится (`409` семьи и пользователей); остальное идёт как есть.
 */
fun ApiFailure.toUiError(known: Map<String, Int> = emptyMap()): UiError = when (this) {
    is ApiFailure.Api -> known[code]?.let { UiError.Resource(it) } ?: UiError.Server(serverMessage)
    is ApiFailure.Network -> UiError.Network
    is ApiFailure.Malformed -> UiError.Malformed
}

fun UiError.message(res: Resources): String = when (this) {
    UiError.Network -> res.getString(R.string.error_network)
    UiError.Malformed -> res.getString(R.string.error_malformed)
    is UiError.Server -> text
    is UiError.Resource -> res.getString(id)
}
