package tech.shatrov.familyfinances

import androidx.compose.runtime.saveable.Saver

/**
 * Экран приложения. Навигационной библиотеки нет намеренно: экранов горстка, переходов между
 * ними — единицы.
 */
sealed interface AppScreen {
    /** Токен есть, но роль и валюта ещё не приехали. */
    data object Loading : AppScreen

    data object Login : AppScreen

    data object Home : AppScreen
}

private const val KEY_LOADING = "loading"
private const val KEY_LOGIN = "login"
private const val KEY_HOME = "home"

/** Экран переживает поворот; всё остальное восстанавливается из хранилища токена. */
val AppScreenSaver: Saver<AppScreen, String> = Saver(
    save = { screen ->
        when (screen) {
            AppScreen.Loading -> KEY_LOADING
            AppScreen.Login -> KEY_LOGIN
            AppScreen.Home -> KEY_HOME
        }
    },
    restore = { key ->
        when (key) {
            KEY_LOGIN -> AppScreen.Login
            KEY_HOME -> AppScreen.Home
            else -> AppScreen.Loading
        }
    },
)
