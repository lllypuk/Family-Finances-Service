package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.CreateUserRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListUsers200Response
import tech.shatrov.familyfinances.core.api.PatchUserRequest
import tech.shatrov.familyfinances.core.api.SetPasswordRequest
import tech.shatrov.familyfinances.core.api.UpdateUserRequest
import tech.shatrov.familyfinances.core.api.UserOk

interface UsersApi {
    /**
     * POST api/v1/users
     * Создать пользователя
     * Только admin. Инвайтов нет: админ задаёт пароль и сообщает его лично, пользователь меняет его через &#x60;PUT /api/v1/me/password&#x60; (A-03). 
     * Responses:
     *  - 201: Пользователь
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createUserRequest 
     * @return [UserOk]
     */
    @POST("api/v1/users")
    suspend fun createUser(@Body createUserRequest: CreateUserRequest): Response<UserOk>

    /**
     * GET api/v1/users/{id}
     * Пользователь по id
     * Только admin.
     * Responses:
     *  - 200: Пользователь
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [UserOk]
     */
    @GET("api/v1/users/{id}")
    suspend fun getUser(@Path("id") id: java.util.UUID): Response<UserOk>

    /**
     * GET api/v1/users
     * Пользователи семьи
     * Только admin.
     * Responses:
     *  - 200: Список пользователей
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @return [ListUsers200Response]
     */
    @GET("api/v1/users")
    suspend fun listUsers(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0): Response<ListUsers200Response>

    /**
     * PATCH api/v1/users/{id}
     * Изменить роль или активность
     * Только admin. Деактивация (&#x60;is_active: false&#x60;) отзывает все сессии пользователя (A-04). Понижение роли или деактивация последнего активного админа — &#x60;409 LAST_ADMIN&#x60;; деактивация себя — &#x60;409 CANNOT_DEACTIVATE_SELF&#x60;. Оба поля в одном запросе применяются по очереди: сначала роль, затем активность; если второй шаг отвечает &#x60;409&#x60;, изменение роли уже сохранено — ответ описывает только не применённый шаг. 
     * Responses:
     *  - 200: Пользователь
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param patchUserRequest 
     * @return [UserOk]
     */
    @PATCH("api/v1/users/{id}")
    suspend fun patchUser(@Path("id") id: java.util.UUID, @Body patchUserRequest: PatchUserRequest): Response<UserOk>

    /**
     * PUT api/v1/users/{id}/password
     * Задать пароль пользователю
     * Только admin. Текущий пароль не требуется; все сессии пользователя отзываются.
     * Responses:
     *  - 204: Пароль задан
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param setPasswordRequest 
     * @return [Unit]
     */
    @PUT("api/v1/users/{id}/password")
    suspend fun setUserPassword(@Path("id") id: java.util.UUID, @Body setPasswordRequest: SetPasswordRequest): Response<Unit>

    /**
     * PUT api/v1/users/{id}
     * Изменить имя и email пользователя
     * Только admin.
     * Responses:
     *  - 200: Пользователь
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateUserRequest 
     * @return [UserOk]
     */
    @PUT("api/v1/users/{id}")
    suspend fun updateUser(@Path("id") id: java.util.UUID, @Body updateUserRequest: UpdateUserRequest): Response<UserOk>

}
