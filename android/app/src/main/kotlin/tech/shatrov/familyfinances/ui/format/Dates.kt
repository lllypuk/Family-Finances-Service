package tech.shatrov.familyfinances.ui.format

import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.YearMonth
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.Locale

/** Интерфейс русский целиком, поэтому месяц берётся из русской локали, а не из системной. */
private val locale: Locale = Locale.forLanguageTag("ru")
private val dayMonth = DateTimeFormatter.ofPattern("d MMMM", locale)
private val dayMonthYear = DateTimeFormatter.ofPattern("d MMMM yyyy", locale)
private val dayMonthYearTime = DateTimeFormatter.ofPattern("d MMMM yyyy, HH:mm", locale)
private val monthYear = DateTimeFormatter.ofPattern("LLLL yyyy", locale)
private val monthName = DateTimeFormatter.ofPattern("LLLL", locale)

/** Дата операции: год показывается только чужой — в списке за текущий месяц он лишний шум. */
fun formatDay(
    date: LocalDate,
    today: LocalDate = LocalDate.now(),
): String = date.format(if (date.year == today.year) dayMonth else dayMonthYear)

/** Дата с годом всегда: там, где проверить нужно именно год. */
fun formatFullDay(date: LocalDate): String = date.format(dayMonthYear)

fun formatPeriod(
    from: LocalDate,
    to: LocalDate,
    today: LocalDate = LocalDate.now(),
): String = "${formatDay(from, today)} — ${formatDay(to, today)}"

/** Заголовок главной: `LLLL` — именительный «сентябрь», `MMMM` дал бы родительный «сентября». */
fun formatMonth(date: LocalDate): String = date.format(monthYear).replaceFirstChar { it.titlecase(locale) }

/** Месяц строчными посреди фразы: «Сверка · август». */
fun formatMonthName(month: YearMonth): String = month.format(monthName)

/** Служебная метка в зоне семьи (A-06): телефон в поездке иначе показал бы чужое время. */
fun formatDateTime(
    at: OffsetDateTime,
    zone: ZoneId,
): String = at.atZoneSameInstant(zone).format(dayMonthYearTime)
