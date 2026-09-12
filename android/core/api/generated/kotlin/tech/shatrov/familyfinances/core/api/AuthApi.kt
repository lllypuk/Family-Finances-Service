package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListSessions200Response
import tech.shatrov.familyfinances.core.api.Login200Response
import tech.shatrov.familyfinances.core.api.LoginRequest

interface AuthApi {
    /**
     * GET api/v1/auth/sessions
     * Активные сессии текущего пользователя
     * Доступно любой роли; чужие сессии не видны даже админу.
     * Responses:
     *  - 200: Список сессий
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @return [ListSessions200Response]
     */
    @GET("api/v1/auth/sessions")
    suspend fun listSessions(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0): Response<ListSessions200Response>

    /**
     * POST api/v1/auth/login
     * Выдать bearer-токен
     * Публичный роут. Токен отдаётся один раз и хранится на сервере только как SHA-256 (A-01). Ответ на неизвестный email и на неверный пароль одинаков — &#x60;401 INVALID_CREDENTIALS&#x60;. Лимитер: 10 попыток / 5 минут на IP и 20 / час на email (A-09). 
     * Responses:
     *  - 200: Токен выдан
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Неверный email или пароль (`INVALID_CREDENTIALS`); ответ одинаков для обоих случаев
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *  - 429: Сработал лимитер (`RATE_LIMITED`)
     *
     * @param loginRequest 
     * @return [Login200Response]
     */
    @POST("api/v1/auth/login")
    suspend fun login(@Body loginRequest: LoginRequest): Response<Login200Response>

    /**
     * POST api/v1/auth/logout
     * Отозвать текущую сессию
     * Доступно любой роли. Токен из заголовка становится недействительным немедленно.
     * Responses:
     *  - 204: Сессия отозвана
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *
     * @return [Unit]
     */
    @POST("api/v1/auth/logout")
    suspend fun logout(): Response<Unit>

    /**
     * DELETE api/v1/auth/sessions/{id}
     * Отозвать сессию
     * Только собственная сессия текущего пользователя; чужая — &#x60;404&#x60;.
     * Responses:
     *  - 204: Сессия отозвана
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/auth/sessions/{id}")
    suspend fun revokeSession(@Path("id") id: java.util.UUID): Response<Unit>

}
