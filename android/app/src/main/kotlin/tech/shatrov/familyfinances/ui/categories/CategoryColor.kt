package tech.shatrov.familyfinances.ui.categories

import androidx.compose.ui.graphics.Color

private const val HEX_RADIX = 16
private const val OPAQUE = 0xFF000000

/**
 * `#RRGGBB` из ответа в цвет метки. Формат стережёт сервер (`hexcolor`), но старый APK может
 * получить и другое: цвет — не повод падать, поэтому непонятное заменяется запасным.
 */
fun parseCategoryColor(
    hex: String,
    fallback: Color,
): Color {
    val digits = hex.removePrefix("#")
    if (digits.length != "RRGGBB".length) return fallback
    val rgb = digits.toLongOrNull(HEX_RADIX) ?: return fallback
    return Color(OPAQUE or rgb)
}
