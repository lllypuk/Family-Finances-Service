package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.CreateHoldingRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.HoldingOk
import tech.shatrov.familyfinances.core.api.ListHoldings200Response
import tech.shatrov.familyfinances.core.api.UpdateHoldingRequest

interface HoldingsApi {
    /**
     * POST api/v1/holdings
     * Создать актив или пассив
     * admin и member. Повтор с тем же &#x60;id&#x60; возвращает &#x60;200&#x60; с существующей позицией; остальное тело при этом игнорируется (A-07). Имя уникально в семье без учёта регистра и пробелов по краям, архивная позиция его тоже занимает — &#x60;409 HOLDING_NAME_EXISTS&#x60;. &#x60;kind&#x60; не из списка своей стороны — &#x60;422&#x60;. 
     * Responses:
     *  - 201: Позиция
     *  - 200: Позиция
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`, `HOLDING_NAME_EXISTS` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createHoldingRequest 
     * @return [HoldingOk]
     */
    @POST("api/v1/holdings")
    suspend fun createHolding(@Body createHoldingRequest: CreateHoldingRequest): Response<HoldingOk>

    /**
     * DELETE api/v1/holdings/{id}
     * Удалить позицию
     * Только admin. Снимки удаляются вместе с позицией, и прошлые значения капитала меняются — это исправление ошибочно заведённой позиции. Проданное имущество — снимок &#x60;0&#x60; и архив. 
     * Responses:
     *  - 204: Позиция удалена
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/holdings/{id}")
    suspend fun deleteHolding(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * GET api/v1/holdings
     * Активы и пассивы семьи
     * admin и member. Порядок — по имени без учёта регистра. &#x60;current&#x60; — последний снимок с датой не позже сегодняшнего дня в часовом поясе семьи, &#x60;null&#x60; — снимков нет. 
     * Responses:
     *  - 200: Список позиций
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param archived &#x60;true&#x60; добавляет в список архивные позиции (optional, default to false)
     * @return [ListHoldings200Response]
     */
    @GET("api/v1/holdings")
    suspend fun listHoldings(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("archived") archived: kotlin.Boolean? = false): Response<ListHoldings200Response>

    /**
     * PUT api/v1/holdings/{id}
     * Переименовать, сменить вид, архивировать или вернуть позицию
     * admin и member. &#x60;side&#x60; не меняется: поля в запросе нет, присланное игнорируется. Архив только прячет позицию из списка без &#x60;?archived&#x3D;true&#x60; — в капитале она остаётся по своим снимкам; прекращение стоимости — снимок &#x60;0&#x60;. 
     * Responses:
     *  - 200: Позиция
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`, `HOLDING_NAME_EXISTS` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateHoldingRequest 
     * @return [HoldingOk]
     */
    @PUT("api/v1/holdings/{id}")
    suspend fun updateHolding(@Path("id") id: java.util.UUID, @Body updateHoldingRequest: UpdateHoldingRequest): Response<HoldingOk>

}
