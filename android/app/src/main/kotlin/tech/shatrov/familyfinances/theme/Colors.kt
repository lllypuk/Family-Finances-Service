package tech.shatrov.familyfinances.theme

import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * Токены цвета. Роли Material покрывают не всё: «доход/расход» и выключенный элемент приезжают
 * до экранов через [LocalAppColors].
 */
data class AppColors(
    val canvas: Color,
    val surface: Color,
    val elevated: Color,
    val border: Color,
    val textPrimary: Color,
    val textSecondary: Color,
    val textInverse: Color,
    val action: Color,
    val income: Color,
    val warning: Color,
    val expense: Color,
    val disabled: Color,
)

/** Тема одна и тёмная: светлого варианта в v1 нет, поэтому и параметра «светлая/тёмная» нет. */
val DarkColors = AppColors(
    canvas = Color(0xFF071014),
    surface = Color(0xFF121B20),
    elevated = Color(0xFF19252B),
    border = Color(0xFF334248),
    textPrimary = Color(0xFFF2F5F5),
    textSecondary = Color(0xFF9CABAF),
    textInverse = Color(0xFF06110F),
    action = Color(0xFF20D1B0),
    income = Color(0xFF20D1B0),
    warning = Color(0xFFF2A833),
    expense = Color(0xFFF25A52),
    disabled = Color(0xFF59666A),
)

// Доход и акцент — один цвет, поэтому знак суммы обязан читаться ещё и текстом: одним цветом
// «+» от «−» не отличается при нарушенном цветовосприятии.
val LocalAppColors = staticCompositionLocalOf { DarkColors }
