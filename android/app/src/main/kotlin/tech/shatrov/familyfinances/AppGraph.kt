package tech.shatrov.familyfinances

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.auth.KeystoreTokenVault
import java.time.OffsetDateTime

/**
 * Composition root приложения: сеть, хранилище токена и текущая сессия.
 * Hilt и Koin не подключаются — граф собирается в одном месте и целиком виден.
 */
class AppGraph(val api: ApiGraph) {
    private val mutableSession = MutableStateFlow<Session?>(null)

    /** Роль и валюта для экранов: заполняется бутстрапом, гаснет на выходе. */
    val session: StateFlow<Session?> = mutableSession.asStateFlow()

    fun hasLiveToken(now: OffsetDateTime = OffsetDateTime.now()): Boolean = api.tokens.hasLiveToken(now)

    /** `true` — сессия готова, `false` — на экран входа. */
    suspend fun bootstrap(): Boolean {
        val loaded = api.loadSession()
        mutableSession.value = loaded
        return loaded != null
    }

    suspend fun signOut() {
        api.signOut()
        mutableSession.value = null
    }

    companion object {
        fun create(
            context: Context,
            baseUrl: String,
        ): AppGraph = AppGraph(ApiGraph(baseUrl, KeystoreTokenVault(context)))
    }
}
