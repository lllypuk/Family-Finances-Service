package tech.shatrov.familyfinances.ui.settings

import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

/** Коды `409`, которые клиент переводит сам: серверный текст на экране настроек не годится. */
internal object ConflictCode {
    const val EMAIL_TAKEN = "EMAIL_TAKEN"
    const val LAST_ADMIN = "LAST_ADMIN"
    const val CANNOT_DEACTIVATE_SELF = "CANNOT_DEACTIVATE_SELF"
    const val CURRENCY_LOCKED = "CURRENCY_LOCKED"
}

/** Общая карта для `409` профиля, пользователей и семьи — `toUiError(known)` берёт её целиком. */
internal val settingsConflicts: Map<String, Int> = mapOf(
    ConflictCode.EMAIL_TAKEN to R.string.settings_error_email_taken,
    ConflictCode.LAST_ADMIN to R.string.settings_error_last_admin,
    ConflictCode.CANNOT_DEACTIVATE_SELF to R.string.settings_error_deactivate_self,
    ConflictCode.CURRENCY_LOCKED to R.string.settings_error_currency_locked,
)

/** Роль сняли с другого телефона: подраздел этому токену больше не отвечает. */
internal val ApiFailure.forbidden: Boolean get() = (this as? ApiFailure.Api)?.isForbidden == true

/** Отказ подраздела: `403` получает свой текст (серверный говорит про права, а не про что делать). */
internal fun ApiFailure.toSettingsError(): UiError =
    if (forbidden) UiError.Resource(R.string.settings_forbidden) else toUiError(settingsConflicts)
