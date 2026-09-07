package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.CreateReportRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListReports200Response
import tech.shatrov.familyfinances.core.api.ReportOk

interface ReportsApi {
    /**
     * POST api/v1/reports
     * Сгенерировать отчёт
     * admin и member. Владелец берётся из сессии — &#x60;user_id&#x60; в теле игнорируется. 
     * Responses:
     *  - 201: Отчёт
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createReportRequest 
     * @return [ReportOk]
     */
    @POST("api/v1/reports")
    suspend fun createReport(@Body createReportRequest: CreateReportRequest): Response<ReportOk>

    /**
     * DELETE api/v1/reports/{id}
     * Удалить отчёт
     * admin и member.
     * Responses:
     *  - 204: Отчёт удалён
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/reports/{id}")
    suspend fun deleteReport(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * GET api/v1/reports/{id}/export
     * Выгрузить отчёт в CSV
     * admin и member. Разделитель — запятая, кодировка UTF-8 с BOM (Excel), суммы — целые в минимальных единицах, рядом колонка &#x60;Currency&#x60;. 
     * Responses:
     *  - 200: CSV-файл
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [kotlin.String]
     */
    @GET("api/v1/reports/{id}/export")
    suspend fun exportReport(@Path("id") id: java.util.UUID): Response<kotlin.String>

    /**
     * GET api/v1/reports/{id}
     * Отчёт с данными
     * admin и member.
     * Responses:
     *  - 200: Отчёт
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [ReportOk]
     */
    @GET("api/v1/reports/{id}")
    suspend fun getReport(@Path("id") id: java.util.UUID): Response<ReportOk>

    /**
     * GET api/v1/reports
     * Сохранённые отчёты
     * admin и member.
     * Responses:
     *  - 200: Список отчётов
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param userId Только отчёты этого автора (optional)
     * @return [ListReports200Response]
     */
    @GET("api/v1/reports")
    suspend fun listReports(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("user_id") userId: java.util.UUID? = null): Response<ListReports200Response>

}
