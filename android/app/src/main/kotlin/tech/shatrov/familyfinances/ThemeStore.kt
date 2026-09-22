package tech.shatrov.familyfinances

import android.content.Context
import androidx.core.content.edit
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import tech.shatrov.familyfinances.theme.ThemeMode

/** Выбор темы в настройках; `mode.value` готов сразу — активити читает его до `setContent`. */
interface ThemeStore {
    val mode: StateFlow<ThemeMode>

    fun write(mode: ThemeMode)
}

/** Живёт до конца процесса: для тестов и графа, собранного без контекста. */
class MemoryThemeStore(initial: ThemeMode = ThemeMode.System) : ThemeStore {
    private val state = MutableStateFlow(initial)

    override val mode: StateFlow<ThemeMode> = state.asStateFlow()

    override fun write(mode: ThemeMode) {
        state.value = mode
    }
}

/** SharedPreferences по той же причине, что `PrefsLastAccountStore`: синхронное чтение одной строки. */
class PrefsThemeStore(context: Context) : ThemeStore {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    private val state = MutableStateFlow(parse(prefs.getString(KEY, null)))

    override val mode: StateFlow<ThemeMode> = state.asStateFlow()

    override fun write(mode: ThemeMode) {
        prefs.edit { putString(KEY, mode.name) }
        state.value = mode
    }

    private companion object {
        const val PREFS_NAME = "ui"
        const val KEY = "theme_mode"

        fun parse(stored: String?): ThemeMode = ThemeMode.entries.firstOrNull { it.name == stored } ?: ThemeMode.System
    }
}
