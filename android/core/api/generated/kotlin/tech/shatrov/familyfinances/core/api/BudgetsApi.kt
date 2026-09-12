package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.BudgetOk
import tech.shatrov.familyfinances.core.api.CreateBudgetRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListBudgets200Response
import tech.shatrov.familyfinances.core.api.UpdateBudgetRequest

interface BudgetsApi {
    /**
     * POST api/v1/budgets
     * Создать бюджет
     * admin и member. Повтор с тем же &#x60;id&#x60; возвращает &#x60;200&#x60; с существующим бюджетом; остальное тело при этом игнорируется (A-07). Бизнес-отказы отвечают &#x60;409&#x60; — &#x60;BUDGET_OVERLAP&#x60; (период пересекается с бюджетом той же области; границы включительные — общий день уже пересечение) и &#x60;BUDGET_NAME_EXISTS&#x60; (имя занято на этот период).
     * Responses:
     *  - 201: Бюджет
     *  - 200: Бюджет
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createBudgetRequest 
     * @return [BudgetOk]
     */
    @POST("api/v1/budgets")
    suspend fun createBudget(@Body createBudgetRequest: CreateBudgetRequest): Response<BudgetOk>

    /**
     * DELETE api/v1/budgets/{id}
     * Удалить бюджет
     * admin и member. Транзакции периода не затрагиваются.
     * Responses:
     *  - 204: Бюджет удалён
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/budgets/{id}")
    suspend fun deleteBudget(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * GET api/v1/budgets/{id}
     * Бюджет по id
     * admin и member. &#x60;spent_minor&#x60; считается по транзакциям периода на момент запроса.
     * Responses:
     *  - 200: Бюджет
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [BudgetOk]
     */
    @GET("api/v1/budgets/{id}")
    suspend fun getBudget(@Path("id") id: java.util.UUID): Response<BudgetOk>

    /**
     * GET api/v1/budgets
     * Бюджеты семьи
     * admin и member.
     * Responses:
     *  - 200: Список бюджетов
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param activeOnly &#x60;true&#x60; — только бюджеты, активные на сегодня (optional)
     * @return [ListBudgets200Response]
     */
    @GET("api/v1/budgets")
    suspend fun listBudgets(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("active_only") activeOnly: kotlin.Boolean? = null): Response<ListBudgets200Response>

    /**
     * PUT api/v1/budgets/{id}
     * Изменить бюджет
     * admin и member. Бизнес-отказы отвечают &#x60;409&#x60; — &#x60;BUDGET_OVERLAP&#x60;, &#x60;BUDGET_NAME_EXISTS&#x60; и &#x60;BUDGET_BELOW_SPENT&#x60; (новая сумма меньше уже потраченного за период).
     * Responses:
     *  - 200: Бюджет
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateBudgetRequest 
     * @return [BudgetOk]
     */
    @PUT("api/v1/budgets/{id}")
    suspend fun updateBudget(@Path("id") id: java.util.UUID, @Body updateBudgetRequest: UpdateBudgetRequest): Response<BudgetOk>

}
