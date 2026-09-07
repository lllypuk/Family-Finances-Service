package tech.shatrov.familyfinances.core.api.auth

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.OffsetDateTime

class CipherTextTest {
    private val iv = ByteArray(GCM_IV_BYTES) { it.toByte() }

    @Test
    fun packedCipherTextSurvivesRoundTrip() {
        val body = byteArrayOf(9, 8, 7, 6, 5)

        val unpacked = requireNotNull(CipherText.unpack(CipherText(iv, body).pack()))

        assertArrayEquals(iv, unpacked.iv)
        assertArrayEquals(body, unpacked.body)
    }

    // Перевыпуск ключа: непригодная запись возвращает null, а не бросает — иначе приложение
    // падало бы на старте после восстановления из бэкапа.
    @Test
    fun garbageIsRejectedWithoutThrowing() {
        assertNull(CipherText.unpack("не base64!"))
    }

    @Test
    fun cipherTextWithoutBodyIsRejected() {
        assertNull(CipherText.unpack(CipherText(iv, ByteArray(0)).pack()))
    }

    @Test
    fun payloadRoundTripKeepsTokenAndExpiry() {
        val token = SessionToken("t-1", OffsetDateTime.parse("2027-03-06T10:00:00Z"))

        val decoded = requireNotNull(decodePayload(token.encodePayload()))

        assertEquals("t-1", decoded.token)
        assertEquals(token.expiresAt, decoded.expiresAt)
    }

    @Test
    fun payloadWithBrokenExpiryIsRejected() {
        assertNull(decodePayload("t-1\nвчера".toByteArray()))
    }

    @Test
    fun payloadWithoutSeparatorIsRejected() {
        assertNull(decodePayload("t-1".toByteArray()))
    }

    @Test
    fun tokenIsNotPrinted() {
        val token = SessionToken("секрет", OffsetDateTime.parse("2027-03-06T10:00:00Z"))

        assertEquals(false, token.toString().contains("секрет"))
    }
}
