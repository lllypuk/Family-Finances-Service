package tech.shatrov.familyfinances.ui.settings

import tech.shatrov.familyfinances.R

/** Коды `409`, которые клиент переводит сам: серверный текст на экране настроек не годится. */
internal object ConflictCode {
    const val EMAIL_TAKEN = "EMAIL_TAKEN"
    const val LAST_ADMIN = "LAST_ADMIN"
    const val CANNOT_DEACTIVATE_SELF = "CANNOT_DEACTIVATE_SELF"
}

/** Общая карта для `409` профиля, пользователей и семьи — `toUiError(known)` берёт её целиком. */
internal val settingsConflicts: Map<String, Int> = mapOf(
    ConflictCode.EMAIL_TAKEN to R.string.settings_error_email_taken,
    ConflictCode.LAST_ADMIN to R.string.settings_error_last_admin,
    ConflictCode.CANNOT_DEACTIVATE_SELF to R.string.settings_error_deactivate_self,
)
