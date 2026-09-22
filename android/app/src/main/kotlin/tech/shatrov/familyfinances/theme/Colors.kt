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

/**
 * Наборов два, третьего не будет: новый токен добавляется в оба и в тест контраста (`ColorsTest`).
 */
val DarkColors = AppColors(
    canvas = Color(0xFF071014),
    surface = Color(0xFF121B20),
    elevated = Color(0xFF19252B),
    border = Color(0xFF334248),
    textPrimary = Color(0xFFF2F5F5),
    textSecondary = Color(0xFF9CABAF),
    textInverse = Color(0xFF06110F),
    action = Color(0xFF20D1B0),
    income = Color(0xFF5FD37A),
    warning = Color(0xFFF2A833),
    expense = Color(0xFFF25A52),
    disabled = Color(0xFF59666A),
)

val LightColors = AppColors(
    canvas = Color(0xFFEDF2F2),
    surface = Color(0xFFF7FAFA),
    elevated = Color(0xFFFFFFFF),
    border = Color(0xFFC3D0D2),
    textPrimary = Color(0xFF0F1C20),
    textSecondary = Color(0xFF4F6166),
    textInverse = Color(0xFFF7FAFA),
    action = Color(0xFF087060),
    income = Color(0xFF177A35),
    warning = Color(0xFF9A5B00),
    expense = Color(0xFFC4322B),
    disabled = Color(0xFF9AA7AA),
)

// Знак суммы ставится текстом, а не только цветом: при нарушенном цветовосприятии зелёный
// «доход» от красного «расхода» не отличается.
val LocalAppColors = staticCompositionLocalOf { DarkColors }
