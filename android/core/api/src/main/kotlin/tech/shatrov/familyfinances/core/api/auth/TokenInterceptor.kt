package tech.shatrov.familyfinances.core.api.auth

import okhttp3.Interceptor
import okhttp3.Response

private const val HTTP_UNAUTHORIZED = 401
private const val LOGIN_PATH = "/api/v1/auth/login"

/**
 * Подставляет `Authorization: Bearer` и снимает сессию по `401`.
 * Логин исключён явно: его `401` — про пароль, а токен в хранилище к этому моменту может быть
 * живым (бутстрап отвалился по сети и увёл на экран входа), и стирать его нельзя.
 */
internal class TokenInterceptor(
    private val tokens: TokenVault,
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
        if (token != null && response.code == HTTP_UNAUTHORIZED) {
            tokens.clear()
            onSessionExpired()
        }
        return response
    }
}
