package tech.shatrov.familyfinances.ui.format

/** Размер файла БД считается по 1024: так его показывают `ls -h` и файловый менеджер телефона. */
private const val STEP = 1024L
private const val TENTHS = 10L

/** Неразрывный пробел: число и единица не разъезжаются по строкам. */
private const val NBSP = ' '

private val units = listOf("КБ", "МБ", "ГБ", "ТБ")

/**
 * Размер бэкапа. До килобайта — целые байты, дальше одна десятая; счёт целочисленный, поэтому
 * граница единицы («1024 Б» → «1,0 КБ») не зависит от округления `Double`.
 */
fun formatBytes(bytes: Long): String {
    if (bytes < STEP) return "$bytes${NBSP}Б"

    var divisor = STEP
    var unit = units.first()
    for (candidate in units) {
        unit = candidate
        if (bytes / divisor < STEP) break
        divisor *= STEP
    }

    val tenths = (bytes * TENTHS + divisor / 2) / divisor
    return "${tenths / TENTHS},${tenths % TENTHS}$NBSP$unit"
}
