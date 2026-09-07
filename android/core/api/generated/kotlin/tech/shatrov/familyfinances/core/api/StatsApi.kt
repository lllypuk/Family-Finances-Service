package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.GetStatsSummary200Response

interface StatsApi {
    /**
     * GET api/v1/stats/summary
     * Сводка за период
     * admin и member. Заменяет дашборд веб-интерфейса. Без параметров — текущий месяц в часовом поясе семьи (A-06). 
     * Responses:
     *  - 200: Сводка
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param from Начало периода включительно; по умолчанию первое число текущего месяца (optional)
     * @param to Конец периода включительно; по умолчанию сегодня (optional)
     * @return [GetStatsSummary200Response]
     */
    @GET("api/v1/stats/summary")
    suspend fun getStatsSummary(@Query("from") from: java.time.LocalDate? = null, @Query("to") to: java.time.LocalDate? = null): Response<GetStatsSummary200Response>

}
