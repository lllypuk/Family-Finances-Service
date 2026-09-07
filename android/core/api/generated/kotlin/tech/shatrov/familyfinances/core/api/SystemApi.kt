package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.Health

interface SystemApi {
    /**
     * GET health
     * Состояние сервиса
     * Публичный endpoint вне &#x60;/api/v1&#x60;. Клиент читает &#x60;version&#x60; (значение &#x60;git describe&#x60;, подставляется через &#x60;-ldflags&#x60;) и &#x60;setup_complete&#x60;: пока семьи нет, логин отвечает &#x60;409 SETUP_REQUIRED&#x60;. 
     * Responses:
     *  - 200: Сервис работает
     *  - 503: Сервис деградировал (проверки не прошли)
     *  - 429: Сработал лимитер (`RATE_LIMITED`)
     *
     * @return [Health]
     */
    @GET("health")
    suspend fun getHealth(): Response<Health>

}
