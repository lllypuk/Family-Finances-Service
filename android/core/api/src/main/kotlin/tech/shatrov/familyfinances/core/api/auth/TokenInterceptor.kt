package tech.shatrov.familyfinances.core.api.auth

import okhttp3.Interceptor
import okhttp3.Response

private const val HTTP_UNAUTHORIZED = 401

/**
 * Подставляет `Authorization: Bearer` и снимает сессию по `401`.
 * Логин уходит без заголовка, поэтому его `401 INVALID_CREDENTIALS` хранилище не трогает.
 */
internal class TokenInterceptor(
    private val tokens: TokenVault,
    private val onSessionExpired: () -> Unit,
) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
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
