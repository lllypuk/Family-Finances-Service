package tech.shatrov.familyfinances.core.api.auth

import java.time.OffsetDateTime

/**
 * Токен сессии и его срок. Пары access/refresh нет: сервер обновления не даёт, а `401`
 * означает «сессия кончилась» и ведёт на экран входа.
 */
class SessionToken(
    val token: String,
    val expiresAt: OffsetDateTime,
) {
    // Токен в логах не нужен: в logcat он остаётся дольше, чем живёт сессия.
    override fun toString(): String = "SessionToken(expiresAt=$expiresAt)"
}

/** Шов над хранилищем: боевая реализация держит токен в Keystore, тесты подставляют свою. */
interface TokenVault {
    fun read(): SessionToken?

    fun write(token: SessionToken)

    fun clear()
}
