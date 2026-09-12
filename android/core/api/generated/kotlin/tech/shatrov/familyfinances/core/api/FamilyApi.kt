package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.FamilyOk
import tech.shatrov.familyfinances.core.api.UpdateFamilyRequest

interface FamilyApi {
    /**
     * GET api/v1/family
     * Данные семьи
     * Доступно любой роли.
     * Responses:
     *  - 200: Семья
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *
     * @return [FamilyOk]
     */
    @GET("api/v1/family")
    suspend fun getFamily(): Response<FamilyOk>

    /**
     * PUT api/v1/family
     * Изменить семью
     * Только admin. Смена &#x60;currency&#x60; при наличии транзакций запрещена — &#x60;409 CURRENCY_LOCKED&#x60; (A-05). &#x60;timezone&#x60; — IANA-имя, в нём считаются «сегодня» и границы периодов (A-06). 
     * Responses:
     *  - 200: Семья
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param updateFamilyRequest 
     * @return [FamilyOk]
     */
    @PUT("api/v1/family")
    suspend fun updateFamily(@Body updateFamilyRequest: UpdateFamilyRequest): Response<FamilyOk>

}
