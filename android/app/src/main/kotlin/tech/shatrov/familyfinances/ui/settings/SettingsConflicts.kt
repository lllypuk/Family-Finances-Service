package tech.shatrov.familyfinances.ui.settings

import tech.shatrov.familyfinances.R

/** Коды `409`, которые клиент переводит сам: серверный текст на экране настроек не годится. */
internal object ConflictCode {
    const val EMAIL_TAKEN = "EMAIL_TAKEN"
}

/** Общая карта для `409` профиля, пользователей и семьи — `toUiError(known)` берёт её целиком. */
internal val settingsConflicts: Map<String, Int> = mapOf(
    ConflictCode.EMAIL_TAKEN to R.string.settings_error_email_taken,
)
