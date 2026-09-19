package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.infrastructure.CollectionFormats.*
import retrofit2.http.*
import retrofit2.Response
import okhttp3.RequestBody
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

import tech.shatrov.familyfinances.core.api.Error
import tech.shatrov.familyfinances.core.api.GetNetWorthStats200Response
import tech.shatrov.familyfinances.core.api.GetStatsMonthly200Response
import tech.shatrov.familyfinances.core.api.GetStatsSummary200Response
import tech.shatrov.familyfinances.core.api.ReconciliationStatsOk

interface StatsApi {
    /**
     * GET api/v1/stats/net-worth
     * Ряд чистого капитала по месяцам
     * admin и member. По корзине на каждый календарный месяц, попадающий в &#x60;[from, to]&#x60;; корзина — состояние на последний день месяца (для последней — на &#x60;to&#x60;). Значение позиции на дату — её последний снимок не позже этой даты, без срока давности; позиция до первого снимка не входит, архивные входят по своим снимкам. Снимок строго до &#x60;from&#x60; — начальное состояние, снимок ровно на &#x60;from&#x60; — уже изменение. Границы по умолчанию — как у &#x60;getStatsMonthly&#x60;. &#x60;to&#x60; позже сегодняшнего дня в часовом поясе семьи — &#x60;422&#x60; полем &#x60;to&#x60; (проверяется раньше потолка), период шире 120 месяцев — &#x60;422&#x60; полем &#x60;from&#x60;. 
     * Responses:
     *  - 200: Помесячный ряд капитала
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param from Начало периода включительно; по умолчанию первое число месяца одиннадцать месяцев назад (optional)
     * @param to Конец периода включительно, не позже сегодня; по умолчанию сегодня (optional)
     * @return [GetNetWorthStats200Response]
     */
    @GET("api/v1/stats/net-worth")
    suspend fun getNetWorthStats(@Query("from") from: java.time.LocalDate? = null, @Query("to") to: java.time.LocalDate? = null): Response<GetNetWorthStats200Response>

    /**
     * GET api/v1/stats/reconciliation
     * Сверка остатков с операциями за месяц
     * admin и member. &#x60;gap_minor &#x3D; (closing_minor - opening_minor) - (income_minor - expense_minor)&#x60;: больше нуля — не записан приход или записан лишний расход, меньше — наоборот. &#x60;opening_minor&#x60; — остатки на конец прошлого месяца, &#x60;closing_minor&#x60; — этого; каждая сумма &#x60;null&#x60;, пока свой край заполнен не у всех счетов списка, &#x60;gap_minor&#x60; — пока не заполнен любой. Без счетов всё &#x60;null&#x60;.  Список: неархивные счета, заведённые не позже конца месяца, и архивные, у которых есть остаток за этот месяц или ненулевой за прошлый. Остаток на начало у счёта, заведённого в этом месяце, без строки за прошлый — &#x60;0&#x60;; на конец у архивного без строки — &#x60;0&#x60;. Месяц позже текущего — &#x60;422&#x60;. 
     * Responses:
     *  - 200: Сверка остатков с операциями за месяц
     *  - 401: Токена нет, он истёк или отозван (`UNAUTHORIZED`)
     *  - 403: Роль не даёт доступа к операции (`FORBIDDEN`)
     *  - 422: Тело или параметры не прошли валидацию (`VALIDATION_ERROR`); поля — в `error.details`
     *
     * @param month Календарный месяц &#x60;YYYY-MM&#x60; не позже текущего; по умолчанию текущий в часовом поясе семьи  (optional)
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
