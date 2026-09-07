package tech.shatrov.familyfinances

import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import java.time.OffsetDateTime

/** Кто вошёл и в какой валюте считает семья: валюты нет ни в сводке, ни в транзакции. */
data class Session(
    val user: User,
    val family: Family,
) {
    val currency: String get() = family.currency

    /** Роль решает, показывать ли админские действия (удаление категорий, пользователи). */
    val isAdmin: Boolean get() = user.role == Role.admin
}

/** Токен на месте и не просрочен — иначе стартуем с экрана входа, не тратя запрос. */
fun TokenVault.hasLiveToken(now: OffsetDateTime = OffsetDateTime.now()): Boolean =
    read()?.expiresAt?.isAfter(now) == true

/**
 * Бутстрап сессии: `GET /me` и `GET /family` после логина и при старте с сохранённым токеном.
 * Любой отказ — это `null` и возврат на вход: без роли и валюты остальным экранам нечего рисовать.
 */
suspend fun ApiGraph.loadSession(): Session? = try {
    val currentUser = client.unwrap({ me.getCurrentUser() }, { it.`data` })
    val currentFamily = client.unwrap({ family.getFamily() }, { it.`data` })
    Session(currentUser, currentFamily)
} catch (failure: ApiFailure) {
    null
}

/** Выход: хранилище чистится в любом случае — отказ сервера не повод оставить токен на телефоне. */
suspend fun ApiGraph.signOut() {
    try {
        client.send { auth.logout() }
    } catch (failure: ApiFailure) {
        // Сессия кончается на этом телефоне независимо от того, услышал ли её конец сервер.
    } finally {
        tokens.clear()
    }
}
