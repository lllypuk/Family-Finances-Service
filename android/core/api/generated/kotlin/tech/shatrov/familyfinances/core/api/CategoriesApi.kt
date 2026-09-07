package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.CategoryOk
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.core.api.CreateCategoryRequest
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListCategories200Response
import tech.shatrov.familyfinances.core.api.UpdateCategoryRequest

interface CategoriesApi {
    /**
     * POST api/v1/categories
     * Создать категорию
     * admin и member. Повтор с тем же &#x60;id&#x60; возвращает &#x60;200&#x60; с существующей категорией; остальное тело при этом игнорируется (A-07).
     * Responses:
     *  - 201: Категория
     *  - 200: Категория
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param createCategoryRequest 
     * @return [CategoryOk]
     */
    @POST("api/v1/categories")
    suspend fun createCategory(@Body createCategoryRequest: CreateCategoryRequest): Response<CategoryOk>

    /**
     * DELETE api/v1/categories/{id}
     * Удалить категорию
     * Только admin. Категория помечается неактивной (&#x60;is_active: false&#x60;) и перестаёт выводиться в списке, ответ &#x60;204&#x60;; её транзакции сохраняются. Подкатегории деактивируются вместе с родителем. Повторный &#x60;DELETE&#x60; уже неактивной категории — &#x60;404&#x60;. 
     * Responses:
     *  - 204: Категория удалена
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [Unit]
     */
    @DELETE("api/v1/categories/{id}")
    suspend fun deleteCategory(@Path("id") id: java.util.UUID): Response<Unit>

    /**
     * GET api/v1/categories/{id}
     * Категория по id
     * admin и member. Удалённая (&#x60;is_active: false&#x60;) категория по-прежнему отдаётся &#x60;200&#x60; — по ней разрешается имя категории у сохранённых транзакций; из списка она исчезает. 
     * Responses:
     *  - 200: Категория
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param id 
     * @return [CategoryOk]
     */
    @GET("api/v1/categories/{id}")
    suspend fun getCategory(@Path("id") id: java.util.UUID): Response<CategoryOk>

    /**
     * GET api/v1/categories
     * Категории семьи
     * admin и member.
     * Responses:
     *  - 200: Список категорий
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @param type  (optional)
     * @return [ListCategories200Response]
     */
    @GET("api/v1/categories")
    suspend fun listCategories(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0, @Query("type") type: CategoryType? = null): Response<ListCategories200Response>

    /**
     * PUT api/v1/categories/{id}
     * Изменить категорию
     * admin и member. Тип категории не меняется — у неё уже есть транзакции. Удалённая (&#x60;is_active: false&#x60;) категория не редактируется — &#x60;404&#x60;. 
     * Responses:
     *  - 200: Категория
     *  - 400: Тело или идентификатор не разобрались: `INVALID_REQUEST` (сломанный JSON или неверный тип поля), `INVALID_ID` (в пути не UUID). Ошибки валидации значений — это `422`. 
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param id 
     * @param updateCategoryRequest 
     * @return [CategoryOk]
     */
    @PUT("api/v1/categories/{id}")
    suspend fun updateCategory(@Path("id") id: java.util.UUID, @Body updateCategoryRequest: UpdateCategoryRequest): Response<CategoryOk>

}
