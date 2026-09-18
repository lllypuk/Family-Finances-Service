package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.GetStatsMonthly200Response
import tech.shatrov.familyfinances.core.api.GetStatsSummary200Response
import tech.shatrov.familyfinances.core.api.ReconciliationStatsOk

interface StatsApi {
    /**
     * GET api/v1/stats/reconciliation
     * Сверка счетов за месяц
     * admin и member. Строка на каждый неархивный счёт и на архивный, у которого в месяце есть расход или сверка. &#x60;recorded_minor&#x60; — сумма расходов (&#x60;type &#x3D; expense&#x60;) по счёту за календарный месяц; доходы её не уменьшают. &#x60;unassigned_minor&#x60; — расходы без счёта. 
     * Responses:
     *  - 200: Сверка счетов за месяц
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param month Календарный месяц &#x60;YYYY-MM&#x60;; по умолчанию текущий в часовом поясе семьи (optional)
     * @return [ReconciliationStatsOk]
     */
    @GET("api/v1/stats/reconciliation")
    suspend fun getReconciliationStats(@Query("month") month: kotlin.String? = null): Response<ReconciliationStatsOk>

    /**
     * GET api/v1/stats/monthly
     * Ряд итогов по месяцам
     * admin и member. По корзине на каждый календарный месяц, попадающий в &#x60;[from, to]&#x60;; месяц без операций — нули. Крайние месяцы не расширяются до полных: корзина покрывает только дни внутри интервала. Без параметров — двенадцать календарных месяцев по сегодняшний в часовом поясе семьи (&#x60;from&#x60; — первое число месяца одиннадцать месяцев назад). Период шире 120 месяцев — &#x60;422&#x60;. 
     * Responses:
     *  - 200: Помесячный ряд
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param from Начало периода включительно; по умолчанию первое число месяца одиннадцать месяцев назад (optional)
     * @param to Конец периода включительно; по умолчанию сегодня (optional)
     * @return [GetStatsMonthly200Response]
     */
    @GET("api/v1/stats/monthly")
    suspend fun getStatsMonthly(@Query("from") from: java.time.LocalDate? = null, @Query("to") to: java.time.LocalDate? = null): Response<GetStatsMonthly200Response>

    /**
     * GET api/v1/stats/summary
     * Сводка за период
     * admin и member. Заменяет дашборд веб-интерфейса. Без параметров — текущий месяц в часовом поясе семьи (A-06). Серии повторяющихся бюджетов достраиваются и на этом чтении: &#x60;budgets[]&#x60; может содержать инстанс, созданный самим запросом, — список бюджетов после сводки стоит считать устаревшим. 
     * Responses:
     *  - 200: Сводка
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param from Начало периода включительно; по умолчанию первое число текущего месяца (optional)
     * @param to Конец периода включительно; по умолчанию сегодня (optional)
     * @return [GetStatsSummary200Response]
     */
    @GET("api/v1/stats/summary")
    suspend fun getStatsSummary(@Query("from") from: java.time.LocalDate? = null, @Query("to") to: java.time.LocalDate? = null): Response<GetStatsSummary200Response>

}
