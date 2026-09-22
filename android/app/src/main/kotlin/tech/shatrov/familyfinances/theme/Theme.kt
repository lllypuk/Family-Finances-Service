package tech.shatrov.familyfinances.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider

/** Выбор темы пользователем; хранится строкой, неизвестная читается как [System]. */
enum class ThemeMode { System, Light, Dark }

@Composable
fun ThemeMode.isDark(): Boolean = when (this) {
    ThemeMode.System -> isSystemInDarkTheme()
    ThemeMode.Light -> false
    ThemeMode.Dark -> true
}

/**
 * Раскладка семантическая: акцент — только `action`, отказ — только `expense`. Незаданные роли
 * берутся из baseline своей темы; `surfaceTint` задан, иначе фиолетовый утекает в tonal elevation.
 */
fun appColorScheme(
    colors: AppColors,
    dark: Boolean,
): ColorScheme {
    val scheme = if (dark) darkColorScheme() else lightColorScheme()
    return scheme.copy(
        primary = colors.action,
        onPrimary = colors.textInverse,
        primaryContainer = colors.action,
        onPrimaryContainer = colors.textInverse,
        secondary = colors.warning,
        onSecondary = colors.textInverse,
        secondaryContainer = colors.elevated,
        onSecondaryContainer = colors.textPrimary,
        tertiary = colors.income,
        onTertiary = colors.textInverse,
        background = colors.canvas,
        onBackground = colors.textPrimary,
        surface = colors.canvas,
        onSurface = colors.textPrimary,
        surfaceVariant = colors.surface,
        onSurfaceVariant = colors.textSecondary,
        surfaceTint = colors.action,
        surfaceContainerLow = colors.surface,
        surfaceContainer = colors.surface,
        surfaceContainerHigh = colors.elevated,
        surfaceContainerHighest = colors.elevated,
        error = colors.expense,
        onError = colors.textInverse,
        outline = colors.border,
        outlineVariant = colors.border,
    )
}

// Схемы собираются один раз: `MaterialTheme` переписывает четыре десятка состояний при каждой новой.
private val DarkScheme = appColorScheme(DarkColors, dark = true)
private val LightScheme = appColorScheme(LightColors, dark = false)

/** Только Compose: окно (фон, иконки статус-бара) настраивает `MainActivity`. */
@Composable
fun AppTheme(
    mode: ThemeMode = ThemeMode.System,
    content: @Composable () -> Unit,
) {
    val dark = mode.isDark()
    CompositionLocalProvider(LocalAppColors provides if (dark) DarkColors else LightColors) {
        MaterialTheme(
            colorScheme = if (dark) DarkScheme else LightScheme,
            typography = AppTypography,
            content = content,
        )
    }
}
