package tech.shatrov.familyfinances

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.auth.KeystoreTokenVault
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import java.time.OffsetDateTime

/**
 * Composition root приложения: сеть, хранилище токена и текущая сессия.
 * Hilt и Koin не подключаются — граф собирается в одном месте и целиком виден.
 */
class AppGraph(val api: ApiGraph) {
    private val mutableSession = MutableStateFlow<Session?>(null)

    /** Роль и валюта для экранов: заполняется бутстрапом, гаснет на выходе. */
    val session: StateFlow<Session?> = mutableSession.asStateFlow()

    /** Токен на месте и не просрочен — иначе стартуем с экрана входа, не тратя запрос. */
    fun hasLiveToken(now: OffsetDateTime = OffsetDateTime.now()): Boolean =
        api.tokens.read()?.expiresAt?.isAfter(now) == true

    /**
     * Бутстрап сессии: `GET /me` и `GET /family` после логина и при старте с сохранённым токеном.
     * `null` — сессия готова; отказ отдаётся вызывающему целиком: обрыв связи не то же самое,
     * что кончившийся токен, и уводить с ним на пустой экран входа нельзя.
     */
    suspend fun bootstrap(): ApiFailure? {
        val loaded = try {
            Session(
                user = api.client.unwrap { api.me.getCurrentUser() }.`data`,
                family = api.client.unwrap { api.family.getFamily() }.`data`,
            )
        } catch (failure: ApiFailure) {
            mutableSession.value = null
            return failure
        }
        mutableSession.value = loaded
        return null
    }

    /** Выход: хранилище чистится в любом случае — отказ сервера не повод оставить токен на телефоне. */
    suspend fun signOut() {
        try {
            api.client.send { api.auth.logout() }
        } catch (failure: ApiFailure) {
            // Сессия кончается на этом телефоне независимо от того, услышал ли её конец сервер.
        } finally {
            api.tokens.clear()
            mutableSession.value = null
        }
    }

    companion object {
        fun create(
            context: Context,
            baseUrl: String,
        ): AppGraph = AppGraph(ApiGraph(baseUrl, KeystoreTokenVault(context)))
    }
}
