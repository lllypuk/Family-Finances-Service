package tech.shatrov.familyfinances.core.api.auth

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.io.IOException
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

    @Synchronized
    override fun read(): SessionToken? {
        val stored = prefs.getString(PREF_TOKEN, null) ?: return null
        val token = decode(stored)
        if (token == null) {
            reissueKey()
        }
        return token
    }

    // Читают отсюда первый кадр приложения и поток OkHttp, где ловить некому: отказ Keystore
    // (ProviderException, `load(null)`) значит «токена нет», а не краш.
    private fun decode(stored: String): SessionToken? = try {
        CipherText.unpack(stored)?.let { decrypt(it) }?.let { decodePayload(it) }
    } catch (_: IOException) {
        null
    } catch (_: RuntimeException) {
        null
    }

    @Synchronized
    override fun write(token: SessionToken) {
        val packed = try {
            encrypt(token.encodePayload()).pack()
        } catch (e: GeneralSecurityException) {
            throw TokenVaultException("токен не зашифрован", e)
        } catch (e: IOException) {
            // `load(null)` объявляет IOException, а Kotlin проверяемые исключения не требует:
            // без этой ветки отказ Keystore ронял бы приложение сразу после удачного входа.
            throw TokenVaultException("токен не зашифрован", e)
        } catch (e: RuntimeException) {
            // ProviderException и родня: на части устройств Keystore отказывает именно так,
            // и без этого ветка выхода из try роняла бы приложение после удачного входа.
            throw TokenVaultException("токен не зашифрован", e)
        }
        // commit(), а не apply(): результат нужен здесь, иначе о потерянном токене узнает
        // только следующий запуск — уже экраном входа.
        if (!prefs.edit().putString(PREF_TOKEN, packed).commit()) {
            throw TokenVaultException("токен не сохранён", null)
        }
    }

    @Synchronized
    override fun clear() {
        remove()
    }

    // Сравнение и очистка под тем же монитором, что и write(): иначе между ними успевает лечь
    // токен нового входа.
    @Synchronized
    override fun clearIf(token: String): Boolean {
        if (read()?.token != token) return false
        return remove()
    }

    // Не легло на диск — токен на месте, и звать onSessionExpired нельзя: ApiGraph отфильтрует
    // событие по всё ещё читаемому токену, а запросы продолжат носить мёртвый.
    private fun remove(): Boolean = prefs.edit().remove(PREF_TOKEN).commit()

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
        } catch (_: IOException) {
            // Keystore не открылся: перевыпуск попробует следующий запуск.
        } catch (_: RuntimeException) {
            // ProviderException и родня.
        }
    }
}
