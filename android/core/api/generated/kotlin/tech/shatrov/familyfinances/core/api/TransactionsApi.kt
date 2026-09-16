package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.BulkDeleteRequest
import tech.shatrov.familyfinances.core.api.BulkDeleteTransactions200Response
import tech.shatrov.familyfinances.core.api.CreateTransactionRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListTransactions200Response
import tech.shatrov.familyfinances.core.api.RecognizeOk
import tech.shatrov.familyfinances.core.api.TransactionOk
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.core.api.UpdateTransactionRequest

import okhttp3.MultipartBody

interface TransactionsApi {
    /**
     * POST api/v1/transactions/bulk-delete
     * Удалить несколько транзакций
     * admin и member. Неизвестные id пропускаются — ответ несёт число фактически удалённых записей, повтор того же запроса возвращает &#x60;0&#x60;. Чужих id не бывает: семья одна. 
     * Responses:
     *  - 200: Транзакции удалены
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param bulkDeleteRequest 
     * @return [BulkDeleteTransactions200Response]
     */
    @POST("api/v1/transactions/bulk-delete")
    suspend fun bulkDeleteTransactions(@Body bulkDeleteRequest: BulkDeleteRequest): Response<BulkDeleteTransactions200Response>

    /**
     * POST api/v1/transactions
     * Создать транзакцию
     * admin и member. Автор берётся из токена — &#x60;user_id&#x60; в теле игнорируется. Повтор с тем же &#x60;id&#x60; возвращает &#x60;200&#x60; с существующей транзакцией; остальное тело при этом игнорируется (A-07). 
     * Responses:
     *  - 201: Транзакция
     *  - 200: Транзакция
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createTransactionRequest 
     * @return [TransactionOk]
     */
    @POST("api/v1/transactions")
    suspend fun createTransaction(@Body createTransactionRequest: CreateTransactionRequest): Response<TransactionOk>

    /**
     * DELETE api/v1/transactions/{id}
     * Удалить транзакцию
     * admin и member.
     * Responses:
     *  - 204: Транзакция удалена
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/transactions/{id}")
    suspend fun deleteTransaction(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * GET api/v1/transactions/{id}
     * Транзакция по id
     * admin и member.
     * Responses:
     *  - 200: Транзакция
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [TransactionOk]
     */
    @GET("api/v1/transactions/{id}")
    suspend fun getTransaction(@Path("id") id: java.util.UUID): Response<TransactionOk>

    /**
     * GET api/v1/transactions
     * Транзакции семьи
     * admin и member. Сортировка — по &#x60;date&#x60; убыванием, затем по &#x60;created_at&#x60;.
     * Responses:
     *  - 200: Список транзакций
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param userId  (optional)
     * @param categoryId  (optional)
     * @param type  (optional)
     * @param dateFrom Включительно, календарная дата в часовом поясе семьи (optional)
     * @param dateTo Включительно (optional)
     * @param amountFromMinor  (optional)
     * @param amountToMinor  (optional)
     * @param description Подстрока в описании, регистронезависимо (optional)
     * @return [ListTransactions200Response]
     */
    @GET("api/v1/transactions")
    suspend fun listTransactions(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("user_id") userId: java.util.UUID? = null, @Query("category_id") categoryId: java.util.UUID? = null, @Query("type") type: TransactionType? = null, @Query("date_from") dateFrom: java.time.LocalDate? = null, @Query("date_to") dateTo: java.time.LocalDate? = null, @Query("amount_from_minor") amountFromMinor: kotlin.Long? = null, @Query("amount_to_minor") amountToMinor: kotlin.Long? = null, @Query("description") description: kotlin.String? = null): Response<ListTransactions200Response>

    /**
     * POST api/v1/transactions/recognize
     * Распознать операции на скриншотах
     * admin и member. Картинки уходят модели, ответ — **кандидаты** операций: ничего не создаётся, запись идёт обычным &#x60;POST /api/v1/transactions&#x60; с клиентским &#x60;id&#x60;. Картинки на сервере не хранятся. Вызов платный и не идемпотентный: клиент не повторяет его сам, в том числе на &#x60;503&#x60;. Вызов синхронный и долгий — до ~205 с (загрузка до 60 с, модель до 130 с), таймауты клиента рассчитываются на это. Пустой &#x60;items&#x60; — успех. 
     * Responses:
     *  - 200: Кандидаты операций
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 408: Тело не пришло за срок загрузки (`REQUEST_TIMEOUT`)
     *  - 413: Тело больше 11 000 000 байт (`PAYLOAD_TOO_LARGE`)
     *  - 422: `VALIDATION_ERROR`: не картинка, больше 2 МиБ или 16 Мп, шестая часть — `field: images[i]`; ни одной картинки — `field: images`; часть с другим именем — `field` равен её имени 
     *  - 502: Модель ответила, но ответ не разобрался (`RECOGNITION_FAILED`); повторного вызова не было
     *  - 503: `RECOGNITION_UNAVAILABLE`: распознавание на сервере не настроено, либо модель не ответила (сеть, лимиты, просрочка) 
     *
     * @param images PNG или JPEG, до 2 МиБ и 16 Мп каждая; тип определяется по содержимому
     * @return [RecognizeOk]
     */
    @Multipart
    @POST("api/v1/transactions/recognize")
    suspend fun recognizeTransactions(@Part images: List<MultipartBody.Part>): Response<RecognizeOk>

    /**
     * PUT api/v1/transactions/{id}
     * Изменить транзакцию
     * admin и member; редактировать можно и чужую запись — данные общие для семьи.
     * Responses:
     *  - 200: Транзакция
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateTransactionRequest 
     * @return [TransactionOk]
     */
    @PUT("api/v1/transactions/{id}")
    suspend fun updateTransaction(@Path("id") id: java.util.UUID, @Body updateTransactionRequest: UpdateTransactionRequest): Response<TransactionOk>

}
