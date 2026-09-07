package tech.shatrov.familyfinances.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/** Табличные цифры: колонка сумм должна стоять ровно, иначе список транзакций «пляшет». */
const val TABULAR_FIGURES = "tnum"

/**
 * Шрифты системные: вендорить файлы ради двух семейств незачем — приложение читают с руки, а не
 * с приборной панели. Деньги набираются моноширинным с `tnum`, интерфейс — обычным.
 */
val AppTypography = Typography(
    displayLarge = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Medium,
        fontSize = 40.sp,
        lineHeight = 46.sp,
        fontFeatureSettings = TABULAR_FIGURES,
    ),
    displayMedium = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Medium,
        fontSize = 24.sp,
        lineHeight = 30.sp,
        fontFeatureSettings = TABULAR_FIGURES,
    ),
    displaySmall = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 22.sp,
        fontFeatureSettings = TABULAR_FIGURES,
    ),
)
