package tech.shatrov.familyfinances.core.api

import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.filter
import kotlinx.coroutines.flow.receiveAsFlow
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
    // Канал, а не SharedFlow: подписчика может не быть в момент отказа — экран как раз
    // пересоздаётся, — и событие без replay пропало бы, оставив пользователя без токена
    // на экране, который его требует.
    private val sessionExpiredEvents = Channel<Unit>(Channel.CONFLATED)

    /**
     * Сервер ответил `401` на запрос с токеном: хранилище уже очищено, дальше — экран входа.
     * Хранилище перечитывается на выдаче, а не на отправке: событие ждёт подписчика, и вход,
     * прошедший за это время, не должен быть уведён на экран входа ответом прошлой сессии.
     */
    val sessionExpired: Flow<Unit> = sessionExpiredEvents.receiveAsFlow().filter { tokens.read() == null }

    val client: ApiClient = ApiClient(baseUrl, tokens) { sessionExpiredEvents.trySend(Unit) }

    val auth: AuthApi = client.create(AuthApi::class)
    val me: MeApi = client.create(MeApi::class)
    val family: FamilyApi = client.create(FamilyApi::class)
    val categories: CategoriesApi = client.create(CategoriesApi::class)
    val transactions: TransactionsApi = client.create(TransactionsApi::class)
    val budgets: BudgetsApi = client.create(BudgetsApi::class)
    val users: UsersApi = client.create(UsersApi::class)

    // `POST /backups` неидемпотентен: повтор создал бы второй файл и второй прогон retention.
    val backups: BackupsApi = client.createWithoutRetries(BackupsApi::class)
    val stats: StatsApi = client.create(StatsApi::class)
}
