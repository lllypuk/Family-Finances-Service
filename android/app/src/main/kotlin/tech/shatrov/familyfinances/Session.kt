package tech.shatrov.familyfinances

import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.User
import java.time.DateTimeException
import java.time.ZoneId

/** Кто вошёл и в какой валюте считает семья: валюты нет ни в сводке, ни в транзакции. */
data class Session(
    val user: User,
    val family: Family,
) {
    val currency: String get() = family.currency

    /**
     * Часовой пояс семьи: в нём считаются «сегодня» и границы периодов (A-06) — иначе телефон
     * в другой зоне спросит у сервера соседний месяц. Неизвестное имя — зона телефона.
     */
    val zone: ZoneId
        get() = try {
            ZoneId.of(family.timezone)
        } catch (_: DateTimeException) {
            ZoneId.systemDefault()
        }

    /** Роль решает, показывать ли админские действия (удаление категорий, пользователи). */
    val isAdmin: Boolean get() = user.role == Role.admin
}
