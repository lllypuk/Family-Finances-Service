package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import okhttp3.ResponseBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.CreateBackup201Response
import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.ListBackups200Response

interface BackupsApi {
    /**
     * POST api/v1/backups
     * Создать бэкап
     * Только admin. Синхронная операция (&#x60;VACUUM INTO&#x60;), тела запроса нет. Восстановление через API не предусмотрено — только по ssh (A-11). 
     * Responses:
     *  - 201: Бэкап создан
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 500: Бэкап не создан
     *
     * @return [CreateBackup201Response]
     */
    @POST("api/v1/backups")
    suspend fun createBackup(): Response<CreateBackup201Response>

    /**
     * DELETE api/v1/backups/{name}
     * Удалить бэкап
     * Только admin.
     * Responses:
     *  - 204: Файл удалён
     *  - 400: Имя не соответствует шаблону `backup_YYYYMMDD_HHMMSSmmm.db`
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param name Имя файла бэкапа из &#x60;GET /api/v1/backups&#x60;
     * @return [Unit]
     */
    @DELETE("api/v1/backups/{name}")
    suspend fun deleteBackup(@Path("name") name: kotlin.String): Response<Unit>

    /**
     * GET api/v1/backups/{name}/download
     * Скачать бэкап
     * Только admin. Имя проверяется на выход за пределы каталога бэкапов.
     * Responses:
     *  - 200: Файл БД SQLite
     *  - 400: Имя не соответствует шаблону `backup_YYYYMMDD_HHMMSSmmm.db`
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 404: Объект не найден: `NOT_FOUND` для неизвестного пути, `<ENTITY>_NOT_FOUND` (`USER_NOT_FOUND`, `SESSION_NOT_FOUND`, `CATEGORY_NOT_FOUND`, …) для отсутствующей записи 
     *
     * @param name Имя файла бэкапа из &#x60;GET /api/v1/backups&#x60;
     * @return [ResponseBody]
     */
    @GET("api/v1/backups/{name}/download")
    suspend fun downloadBackup(@Path("name") name: kotlin.String): Response<ResponseBody>

    /**
     * GET api/v1/backups
     * Список бэкапов
     * Только admin.
     * Responses:
     *  - 200: Список файлов бэкапа в `BACKUP_DIR`
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param limit  (optional, default to 50)
     * @param offset  (optional, default to 0)
     * @return [ListBackups200Response]
     */
    @GET("api/v1/backups")
    suspend fun listBackups(@Query("limit") limit: kotlin.Int? = 50, @Query("offset") offset: kotlin.Int? = 0): Response<ListBackups200Response>

}
