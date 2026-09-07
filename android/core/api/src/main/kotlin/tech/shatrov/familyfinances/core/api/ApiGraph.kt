package tech.shatrov.familyfinances.core.api

import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.asSharedFlow
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import tech.shatrov.familyfinances.core.api.net.ApiClient

/**
 * Composition root модуля: один клиент и типизированные интерфейсы поверх него.
 * Базовый адрес приходит параметром — `BuildConfig` у каждого модуля свой.
 */
class ApiGraph(
    baseUrl: String,
    val tokens: TokenVault,
) {
    // Буфер на одно событие: подписчика может не быть в момент отказа — экран как раз
    // пересоздаётся, — а уводить на вход всё равно нужно.
    private val sessionExpiredEvents = MutableSharedFlow<Unit>(extraBufferCapacity = 1)

    /** Сервер ответил `401` на запрос с токеном: хранилище уже очищено, дальше — экран входа. */
    val sessionExpired: SharedFlow<Unit> = sessionExpiredEvents.asSharedFlow()

    val client: ApiClient = ApiClient(baseUrl, tokens) { sessionExpiredEvents.tryEmit(Unit) }

    val auth: AuthApi = client.create(AuthApi::class)
    val me: MeApi = client.create(MeApi::class)
    val family: FamilyApi = client.create(FamilyApi::class)
    val categories: CategoriesApi = client.create(CategoriesApi::class)
    val transactions: TransactionsApi = client.create(TransactionsApi::class)
    val stats: StatsApi = client.create(StatsApi::class)
}
