package tech.shatrov.familyfinances.core.api

import tech.shatrov.familyfinances.core.api.net.ApiClient

/**
 * Composition root модуля: один клиент и типизированные интерфейсы поверх него.
 * Базовый адрес приходит параметром — `BuildConfig` у каждого модуля свой.
 */
class ApiGraph(baseUrl: String) {
    val client: ApiClient = ApiClient(baseUrl)

    val auth: AuthApi = client.create(AuthApi::class)
    val me: MeApi = client.create(MeApi::class)
    val family: FamilyApi = client.create(FamilyApi::class)
    val categories: CategoriesApi = client.create(CategoriesApi::class)
    val transactions: TransactionsApi = client.create(TransactionsApi::class)
    val stats: StatsApi = client.create(StatsApi::class)
}
