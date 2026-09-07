package tech.shatrov.familyfinances.core.api.auth

import java.time.OffsetDateTime
import java.time.format.DateTimeParseException
import java.util.Base64

internal const val GCM_IV_BYTES = 12

private const val PAYLOAD_PARTS = 2
private const val PAYLOAD_SEPARATOR = '\n'

/** IV и тело в одной строке: SharedPreferences хранит строку, а IV у каждой записи свой. */
internal class CipherText(
    val iv: ByteArray,
    val body: ByteArray,
) {
    fun pack(): String = Base64.getEncoder().encodeToString(iv + body)

    companion object {
        /** `null` — строка непригодна; вызывающий перевыпускает ключ, а не падает. */
        fun unpack(stored: String): CipherText? {
            val raw = try {
                Base64.getDecoder().decode(stored)
            } catch (_: IllegalArgumentException) {
                return null
            }
            if (raw.size <= GCM_IV_BYTES) return null
            return CipherText(
                iv = raw.copyOfRange(0, GCM_IV_BYTES),
                body = raw.copyOfRange(GCM_IV_BYTES, raw.size),
            )
        }
    }
}

// Под шифром лежит одна строка: токен приходит в base64url, поэтому перевод строки в нём
// невозможен и годится разделителем.
internal fun SessionToken.encodePayload(): ByteArray = "$token$PAYLOAD_SEPARATOR$expiresAt".toByteArray()

/** `null` — расшифрованное не разобрано; сессия считается кончившейся. */
internal fun decodePayload(plain: ByteArray): SessionToken? {
    val parts = plain.decodeToString().split(PAYLOAD_SEPARATOR)
    if (parts.size != PAYLOAD_PARTS || parts[0].isEmpty()) return null
    val expiresAt = try {
        OffsetDateTime.parse(parts[1])
    } catch (_: DateTimeParseException) {
        return null
    }
    return SessionToken(token = parts[0], expiresAt = expiresAt)
}
