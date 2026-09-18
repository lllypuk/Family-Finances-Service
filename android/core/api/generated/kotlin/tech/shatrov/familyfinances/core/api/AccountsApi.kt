package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.AccountOk
import tech.shatrov.familyfinances.core.api.CreateAccountRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListAccounts200Response
import tech.shatrov.familyfinances.core.api.ReconciliationOk
import tech.shatrov.familyfinances.core.api.ReconciliationRequest
import tech.shatrov.familyfinances.core.api.UpdateAccountRequest

interface AccountsApi {
    /**
     * POST api/v1/accounts
     * Создать счёт
     * admin и member. Повтор с тем же &#x60;id&#x60; возвращает &#x60;200&#x60; с существующим счётом; остальное тело при этом игнорируется (A-07). Имя уникально в семье без учёта регистра и пробелов по краям, архивный счёт его тоже занимает — &#x60;409 ACCOUNT_NAME_EXISTS&#x60;. 
     * Responses:
     *  - 201: Счёт
     *  - 200: Счёт
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`, `HOLDING_NAME_EXISTS` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createAccountRequest 
     * @return [AccountOk]
     */
    @POST("api/v1/accounts")
    suspend fun createAccount(@Body createAccountRequest: CreateAccountRequest): Response<AccountOk>

    /**
     * DELETE api/v1/accounts/{id}
     * Удалить счёт
     * Только admin. Счёт с операциями или сверками не удаляется — &#x60;409 ACCOUNT_IN_USE&#x60;; перевыпущенную карту архивируют. 
     * Responses:
     *  - 204: Счёт удалён
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`, `HOLDING_NAME_EXISTS` 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/accounts/{id}")
    suspend fun deleteAccount(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * DELETE api/v1/accounts/{id}/reconciliations/{month}
     * Удалить сверку за месяц
     * admin и member. &#x60;ACCOUNT_NOT_FOUND&#x60; или &#x60;RECONCILIATION_NOT_FOUND&#x60; — &#x60;404&#x60;.
     * Responses:
     *  - 204: Сверка удалена
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param month Календарный месяц &#x60;YYYY-MM&#x60;; иное — &#x60;422&#x60;
     * @return [Unit]
     */
    @DELETE("api/v1/accounts/{id}/reconciliations/{month}")
    suspend fun deleteReconciliation(@Path("id") id: java.util.UUID, @Path("month") month: kotlin.String): Response<Unit>

    /**
     * GET api/v1/accounts
     * Счета семьи
     * admin и member. Порядок — по имени без учёта регистра.
     * Responses:
     *  - 200: Список счетов
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param archived &#x60;true&#x60; добавляет в список архивные счета (optional, default to false)
     * @return [ListAccounts200Response]
     */
    @GET("api/v1/accounts")
    suspend fun listAccounts(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("archived") archived: kotlin.Boolean? = false): Response<ListAccounts200Response>

    /**
     * PUT api/v1/accounts/{id}/reconciliations/{month}
     * Записать цифру банка за месяц
     * admin и member. Полная замена: без &#x60;note&#x60; заметка очищается. Записанное не хранится — его считает &#x60;GET /stats/reconciliation&#x60;, поэтому дописанная операция меняет разницу без нового &#x60;PUT&#x60;. Месяц не блокируется. &#x60;bank_expense_minor&#x60; — &#x60;0 … 99999999999&#x60;, иначе &#x60;422&#x60;. 
     * Responses:
     *  - 200: Сверка
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param month Календарный месяц &#x60;YYYY-MM&#x60;; иное — &#x60;422&#x60;
     * @param reconciliationRequest 
     * @return [ReconciliationOk]
     */
    @PUT("api/v1/accounts/{id}/reconciliations/{month}")
    suspend fun putReconciliation(@Path("id") id: java.util.UUID, @Path("month") month: kotlin.String, @Body reconciliationRequest: ReconciliationRequest): Response<ReconciliationOk>

    /**
     * PUT api/v1/accounts/{id}
     * Переименовать, архивировать или вернуть счёт
     * admin и member. Архивный счёт скрыт из списка без &#x60;?archived&#x3D;true&#x60; и не назначается новым операциям.
     * Responses:
     *  - 200: Счёт
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 409: Состояние не допускает операцию: `SETUP_REQUIRED`, `CURRENCY_LOCKED`, `LAST_ADMIN`, `CANNOT_DEACTIVATE_SELF`, `EMAIL_TAKEN`, `BUDGET_OVERLAP`, `BUDGET_NAME_EXISTS`, `BUDGET_BELOW_SPENT`, `BUDGET_ID_EXISTS`, `BUDGET_NOT_TAIL`, `ACCOUNT_NAME_EXISTS`, `ACCOUNT_IN_USE`, `HOLDING_NAME_EXISTS` 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateAccountRequest 
     * @return [AccountOk]
     */
    @PUT("api/v1/accounts/{id}")
    suspend fun updateAccount(@Path("id") id: java.util.UUID, @Body updateAccountRequest: UpdateAccountRequest): Response<AccountOk>

}
