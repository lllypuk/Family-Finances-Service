package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Response
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import tech.shatrov.familyfinances.core.api.auth.TokenInterceptor
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import java.io.IOException
import java.util.concurrent.TimeUnit
import kotlin.reflect.KClass

private const val CONNECT_TIMEOUT_SECONDS = 10L
private const val READ_TIMEOUT_SECONDS = 30L

/**
 * Транспорт: Retrofit поверх OkHttp и снятие конверта ответа.
 * Базовый адрес — корень хоста со слэшем на конце: пути в сгенерированных интерфейсах полные.
 */
class ApiClient(
    baseUrl: String,
    tokens: TokenVault,
    onSessionExpired: () -> Unit,
) {
    val json: Json = Json {
        // Сервер обновляется сам с каждого мержа, телефоны — руками: новое поле в ответе
        // не должно ронять установленный APK.
        ignoreUnknownKeys = true
        explicitNulls = false
        serializersModule = apiSerializersModule
    }

    private val http: OkHttpClient = OkHttpClient.Builder()
        .addInterceptor(TokenInterceptor(tokens, onSessionExpired))
        .connectTimeout(CONNECT_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .readTimeout(READ_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .build()

    private val retrofit: Retrofit = Retrofit.Builder()
        .baseUrl(baseUrl)
        .client(http)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    fun <T : Any> create(service: KClass<T>): T = retrofit.create(service.java)

    /** Возвращает конверт ответа или бросает [ApiFailure]. */
    suspend fun <E : Any> unwrap(request: suspend () -> Response<E>): E {
        val response = execute(request)
        return response.body() ?: throw ApiFailure.Malformed(response.code(), null)
    }

    /** Операции без тела (`204`): отказ приходит тем же конвертом. */
    suspend fun send(request: suspend () -> Response<Unit>) {
        execute(request)
    }

    private suspend fun <E : Any> execute(request: suspend () -> Response<E>): Response<E> {
        val response = try {
            request()
        } catch (e: IOException) {
            throw ApiFailure.Network(e)
        } catch (e: SerializationException) {
            throw ApiFailure.Malformed(null, e)
        }
        if (!response.isSuccessful) {
            throw parseApiFailure(
                status = response.code(),
                body = response.errorBody()?.string(),
                retryAfter = response.headers()["Retry-After"],
                json = json,
            )
        }
        return response
    }
}
