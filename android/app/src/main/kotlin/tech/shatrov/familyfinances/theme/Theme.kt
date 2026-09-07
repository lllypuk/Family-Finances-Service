package tech.shatrov.familyfinances.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider

/** Раскладка семантическая: акцент — только `action`, отказ — только `expense`. */
fun appColorScheme(colors: AppColors): ColorScheme = darkColorScheme(
    primary = colors.action,
    onPrimary = colors.textInverse,
    primaryContainer = colors.action,
    onPrimaryContainer = colors.textInverse,
    secondary = colors.warning,
    onSecondary = colors.textInverse,
    tertiary = colors.income,
    onTertiary = colors.textInverse,
    background = colors.canvas,
    onBackground = colors.textPrimary,
    surface = colors.canvas,
    onSurface = colors.textPrimary,
    surfaceVariant = colors.surface,
    onSurfaceVariant = colors.textSecondary,
    surfaceContainerLow = colors.surface,
    surfaceContainer = colors.surface,
    surfaceContainerHigh = colors.elevated,
    surfaceContainerHighest = colors.elevated,
    error = colors.expense,
    onError = colors.textPrimary,
    outline = colors.border,
    outlineVariant = colors.border,
)

// Схема собирается один раз: набор один, а `MaterialTheme` переписывает четыре десятка состояний.
private val DarkScheme = appColorScheme(DarkColors)

@Composable
fun AppTheme(content: @Composable () -> Unit) {
    CompositionLocalProvider(LocalAppColors provides DarkColors) {
        MaterialTheme(
            colorScheme = DarkScheme,
            typography = AppTypography,
            content = content,
        )
    }
}
