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

/** Хранилище отказало: ключ Keystore недоступен или запись не легла на диск. */
class TokenVaultException(
    message: String,
    cause: Throwable?,
) : Exception(message, cause)

/** Шов над хранилищем: боевая реализация держит токен в Keystore, тесты подставляют свою. */
interface TokenVault {
    fun read(): SessionToken?

    /** Бросает [TokenVaultException]: вход прошёл, а токен сохранить не удалось. */
    fun write(token: SessionToken)

    fun clear()

    /**
     * Снимает токен, только если в хранилище лежит именно [token]; возвращает `true`, если снял.
     * Ответ давно ушедшего запроса приходит с `401` уже после нового входа, и безусловная
     * очистка стёрла бы свежий токен.
     */
    fun clearIf(token: String): Boolean
}
