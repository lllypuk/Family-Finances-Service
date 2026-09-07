package tech.shatrov.familyfinances

import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.User

/** Кто вошёл и в какой валюте считает семья: валюты нет ни в сводке, ни в транзакции. */
data class Session(
    val user: User,
    val family: Family,
) {
    val currency: String get() = family.currency

    /** Роль решает, показывать ли админские действия (удаление категорий, пользователи). */
    val isAdmin: Boolean get() = user.role == Role.admin
}
