package tech.shatrov.familyfinances

import android.content.Context
import androidx.core.content.edit
import java.util.UUID

/** Счёт последней созданной операции: новая форма и пачка распознавания подставляют его. */
interface LastAccountStore {
    fun read(): UUID?

    fun write(id: UUID?)
}

/** Живёт до конца процесса: для тестов и графа, собранного без контекста. */
class MemoryLastAccountStore(private var stored: UUID? = null) : LastAccountStore {
    override fun read(): UUID? = stored

    override fun write(id: UUID?) {
        stored = id
    }
}

/**
 * Приватный SharedPreferences, а не DataStore: ключ один, а форма читает его синхронно при сборке
 * состояния — DataStore потянул бы зависимость и асинхронное чтение ради одной строки.
 */
class PrefsLastAccountStore(context: Context) : LastAccountStore {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    override fun read(): UUID? = prefs.getString(KEY, null)?.let { runCatching { UUID.fromString(it) }.getOrNull() }

    override fun write(id: UUID?) {
        prefs.edit { if (id == null) remove(KEY) else putString(KEY, id.toString()) }
    }

    private companion object {
        const val PREFS_NAME = "form_defaults"
        const val KEY = "last_account_id"
    }
}
