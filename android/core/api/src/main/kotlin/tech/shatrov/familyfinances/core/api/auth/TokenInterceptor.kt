package tech.shatrov.familyfinances.core.api.auth

import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.Response
import tech.shatrov.familyfinances.core.api.net.ApiErrorCode
import tech.shatrov.familyfinances.core.api.net.ErrorEnvelope

private const val HTTP_UNAUTHORIZED = 401
private const val LOGIN_PATH = "/api/v1/auth/login"
private const val PASSWORD_PATH = "/api/v1/me/password"
private const val PEEK_LIMIT = 4096L

/**
 * Подставляет `Authorization: Bearer` и снимает сессию по `401`.
 * Логин исключён явно: его `401` — про пароль, а токен в хранилище к этому моменту может быть
 * живым (бутстрап отвалился по сети и увёл на экран входа), и стирать его нельзя.
 */
internal class TokenInterceptor(
    private val tokens: TokenVault,
    private val json: Json,
    private val onSessionExpired: () -> Unit,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        if (chain.request().url.encodedPath == LOGIN_PATH) {
            return chain.proceed(chain.request())
        }

        val token = tokens.read()?.token
        val request = if (token == null) {
            chain.request()
        } else {
            chain.request().newBuilder().header("Authorization", "Bearer $token").build()
        }

        val response = chain.proceed(request)
        if (response.code == HTTP_UNAUTHORIZED && !wrongCurrentPassword(response) && sessionGone(token)) {
            onSessionExpired()
        }
        return response
    }

    /**
     * `401 INVALID_CREDENTIALS` на смене своего пароля — про неверный текущий пароль, сессия жива.
     * Тело читается копией: оригинал достаётся `ApiClient`, который разбирает конверт сам.
     * Путь целиком не исключается — запрос должен уходить с токеном, а просроченный токен чиститься.
     */
    private fun wrongCurrentPassword(response: Response): Boolean {
        if (response.request.url.encodedPath != PASSWORD_PATH) return false
        val body = runCatching { response.peekBody(PEEK_LIMIT).string() }.getOrNull() ?: return false
        val envelope = runCatching { json.decodeFromString(ErrorEnvelope.serializer(), body) }.getOrNull()
        return envelope?.error?.code == ApiErrorCode.INVALID_CREDENTIALS
    }

    /**
     * Кончилась ли та сессия, с которой ушёл запрос. Ответ прошлой сессии приходит уже после
     * нового входа и не должен уводить на экран входа: с токеном это проверяет [TokenVault.clearIf],
     * без токена (Keystore не дал его прочитать) — пустое хранилище на момент ответа.
     */
    private fun sessionGone(token: String?): Boolean =
        if (token == null) tokens.read() == null else tokens.clearIf(token)
}
