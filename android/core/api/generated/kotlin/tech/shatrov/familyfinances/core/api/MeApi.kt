package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.ChangePasswordRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.UpdateUserRequest
import tech.shatrov.familyfinances.core.api.UserOk

interface MeApi {
    /**
     * PUT api/v1/me/password
     * Сменить свой пароль
     * Доступно любой роли. Успешная смена отзывает все сессии пользователя, кроме текущей (A-01). Неверный &#x60;current_password&#x60; — &#x60;401 INVALID_CREDENTIALS&#x60;; сессия при этом жива, разлогинивать клиента не нужно. 
     * Responses:
     *  - 204: Пароль изменён
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: `UNAUTHORIZED` — токен отсутствует, просрочен или отозван; `INVALID_CREDENTIALS` — неверный `current_password`. Различать по `error.code`. 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param changePasswordRequest 
     * @return [Unit]
     */
    @PUT("api/v1/me/password")
    suspend fun changePassword(@Body changePasswordRequest: ChangePasswordRequest): Response<Unit>

    /**
     * GET api/v1/me
     * Текущий пользователь
     * Доступно любой роли.
     * Responses:
     *  - 200: Пользователь
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *
     * @return [UserOk]
     */
    @GET("api/v1/me")
    suspend fun getCurrentUser(): Response<UserOk>

    /**
     * PUT api/v1/me
     * Изменить имя и email
     * Доступно любой роли. Роль и &#x60;is_active&#x60; через этот роут не меняются.
     * Responses:
     *  - 200: Пользователь
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param updateUserRequest 
     * @return [UserOk]
     */
    @PUT("api/v1/me")
    suspend fun updateCurrentUser(@Body updateUserRequest: UpdateUserRequest): Response<UserOk>

}
