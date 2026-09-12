package tech.shatrov.familyfinances

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Family
import tech.shatrov.familyfinances.core.api.User
import tech.shatrov.familyfinances.core.api.auth.KeystoreTokenVault
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import java.time.OffsetDateTime
import java.util.concurrent.atomic.AtomicInteger

/**
 * Composition root приложения: сеть, хранилище токена и текущая сессия.
 * Hilt и Koin не подключаются — граф собирается в одном месте и целиком виден.
 */
class AppGraph(val api: ApiGraph) {
    private val mutableSession = MutableStateFlow<Session?>(null)

    // Номер публикации сессии: перечитка, начатая до выхода или до другой публикации, не должна
    // вернуть на экран сессию, которой там уже нет.
    private val generation = AtomicInteger(0)

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
            publish(null)
            return failure
        }
        publish(loaded)
        return null
    }

    /**
     * Перечитка сессии при заходе в настройки: роль и валюту мог сменить второй телефон.
     * Прежняя сессия остаётся при любом отказе, а поздний ответ не затирает публикацию,
     * случившуюся за время запросов.
     */
    suspend fun refreshSession(): ApiFailure? {
        val started = generation.get()
        val loaded = try {
            Session(
                user = api.client.unwrap { api.me.getCurrentUser() }.`data`,
                family = api.client.unwrap { api.family.getFamily() }.`data`,
            )
        } catch (failure: ApiFailure) {
            return failure
        }
        if (generation.compareAndSet(started, started + 1)) mutableSession.value = loaded
        return null
    }

    /** Ответ `PUT /me` или правки своей записи: сессия обновляется без второго бутстрапа. */
    fun update(user: User) {
        val current = mutableSession.value ?: return
        publish(current.copy(user = user))
    }

    /** Ответ `PUT /family`: зона и валюта расходятся по экранам через ту же сессию. */
    fun update(family: Family) {
        val current = mutableSession.value ?: return
        publish(current.copy(family = family))
    }

    /** Выход: хранилище чистится в любом случае — отказ сервера не повод оставить токен на телефоне. */
    suspend fun signOut() {
        try {
            api.client.send { api.auth.logout() }
        } catch (failure: ApiFailure) {
            // Сессия кончается на этом телефоне независимо от того, услышал ли её конец сервер.
        } finally {
            api.tokens.clear()
            publish(null)
        }
    }

    private fun publish(loaded: Session?) {
        generation.incrementAndGet()
        mutableSession.value = loaded
    }

    companion object {
        fun create(
            context: Context,
            baseUrl: String,
        ): AppGraph = AppGraph(ApiGraph(baseUrl, KeystoreTokenVault(context)))
    }
}
