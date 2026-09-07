package tech.shatrov.familyfinances.core.api.auth

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.GeneralSecurityException
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

private const val PREFS_NAME = "ffs_session"
private const val PREF_TOKEN = "token"
private const val KEY_ALIAS = "ffs_session_token"
private const val KEYSTORE_PROVIDER = "AndroidKeyStore"
private const val TRANSFORMATION = "AES/GCM/NoPadding"
private const val GCM_TAG_BITS = 128

/**
 * Токен под AES/GCM: ключ в Android Keystore, шифротекст — в приватном SharedPreferences.
 * Ключ пропадает при смене экрана блокировки и при восстановлении из бэкапа; это не ошибка —
 * запись выбрасывается, ключ перевыпускается, пользователь входит заново.
 */
class KeystoreTokenVault(context: Context) : TokenVault {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override fun read(): SessionToken? {
        val stored = prefs.getString(PREF_TOKEN, null) ?: return null
        val token = CipherText.unpack(stored)?.let { decrypt(it) }?.let { decodePayload(it) }
        if (token == null) {
            reissueKey()
        }
        return token
    }

    override fun write(token: SessionToken) {
        val packed = encrypt(token.encodePayload()).pack()
        // commit(), а не apply(): результат нужен здесь, иначе о потерянном токене узнает
        // только следующий запуск — уже экраном входа.
        check(prefs.edit().putString(PREF_TOKEN, packed).commit()) { "токен не сохранён" }
    }

    override fun clear() {
        prefs.edit().remove(PREF_TOKEN).commit()
    }

    private fun encrypt(plain: ByteArray): CipherText {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, secretKey())
        return CipherText(iv = cipher.iv, body = cipher.doFinal(plain))
    }

    private fun decrypt(cipherText: CipherText): ByteArray? = try {
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(GCM_TAG_BITS, cipherText.iv))
        cipher.doFinal(cipherText.body)
    } catch (_: GeneralSecurityException) {
        null
    }

    private fun secretKey(): SecretKey {
        val keystore = KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }
        (keystore.getEntry(KEY_ALIAS, null) as? KeyStore.SecretKeyEntry)?.let { return it.secretKey }

        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE_PROVIDER)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            ).setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .build(),
        )
        return generator.generateKey()
    }

    private fun reissueKey() {
        clear()
        try {
            KeyStore.getInstance(KEYSTORE_PROVIDER).apply { load(null) }.deleteEntry(KEY_ALIAS)
        } catch (_: GeneralSecurityException) {
            // Ключа уже нет — следующий secretKey() создаст новый.
        }
    }
}
