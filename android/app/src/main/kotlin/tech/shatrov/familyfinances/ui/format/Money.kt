package tech.shatrov.familyfinances.ui.format

import kotlin.math.roundToInt

/** Валюты только с двумя знаками после запятой (A-05), поэтому делитель один на всё приложение. */
private const val MINOR_IN_UNIT = 100L
private const val FRACTION_DIGITS = 2
private const val GROUP_SIZE = 3
private const val PERCENT_IN_SHARE = 100

/** Неразрывные пробелы: сумма не должна переноситься между разрядами и знаком валюты. */
private const val NBSP = '\u00A0'

private val symbols = mapOf("RUB" to "₽", "USD" to "$", "EUR" to "€")

/**
 * Сумма из минорных единиц. Счёт целочисленный на всём пути: `Double` не появляется даже
 * промежуточно, иначе крупные суммы теряют копейки.
 */
fun formatMoney(
    minor: Long,
    currency: String,
    signed: Boolean = false,
): String {
    val absolute = if (minor < 0) -minor else minor
    val sign = when {
        minor < 0 -> "-"
        signed && minor > 0 -> "+"
        else -> ""
    }
    val fraction = (absolute % MINOR_IN_UNIT).toString().padStart(FRACTION_DIGITS, '0')
    return "$sign${groupDigits(absolute / MINOR_IN_UNIT)},$fraction$NBSP${symbolOf(currency)}"
}

/** Доли (`share`, `*_delta`, `utilization`) приходят как 0…1 — в процентах их и показываем. */
fun formatPercent(
    share: Double,
    signed: Boolean = false,
): String {
    val percent = (share * PERCENT_IN_SHARE).roundToInt()
    val sign = if (signed && percent > 0) "+" else ""
    return "$sign$percent%"
}

/** Незнакомый код показывается как есть: список валют сервер не ограничивает. */
private fun symbolOf(currency: String): String = symbols[currency] ?: currency

private fun groupDigits(units: Long): String {
    val digits = units.toString()
    val grouped = StringBuilder(digits.length + digits.length / GROUP_SIZE)
    for ((index, digit) in digits.withIndex()) {
        if (index > 0 && (digits.length - index) % GROUP_SIZE == 0) {
            grouped.append(NBSP)
        }
        grouped.append(digit)
    }
    return grouped.toString()
}
