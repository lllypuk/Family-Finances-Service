package tech.shatrov.familyfinances.ui.format

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

private const val NBSP = "\u00A0"

/** Форматирование денег: целочисленное, поэтому копейки не теряются на крупных суммах. */
class MoneyTest {
    @Test
    fun zeroKeepsBothFractionDigits() {
        assertEquals("0,00$NBSP₽", formatMoney(0, "RUB"))
    }

    @Test
    fun fractionIsPaddedToTwoDigits() {
        assertEquals("0,05$NBSP₽", formatMoney(5, "RUB"))
    }

    @Test
    fun thousandsAreGrouped() {
        assertEquals("1${NBSP}234${NBSP}567,89$NBSP₽", formatMoney(123456789, "RUB"))
    }

    @Test
    fun negativeCarriesMinusWithoutSignedFlag() {
        assertEquals("-4${NBSP}231,05$NBSP₽", formatMoney(-423105, "RUB"))
    }

    @Test
    fun signedShowsPlusOnlyForPositive() {
        assertEquals("+1,00$NBSP₽", formatMoney(100, "RUB", signed = true))
        assertEquals("0,00$NBSP₽", formatMoney(0, "RUB", signed = true))
        assertEquals("-1,00$NBSP₽", formatMoney(-100, "RUB", signed = true))
    }

    @Test
    fun unknownCurrencyIsShownAsCode() {
        assertEquals("10,00${NBSP}GBP", formatMoney(1000, "GBP"))
        assertEquals("10,00$NBSP$", formatMoney(1000, "USD"))
    }

    // Сумма больше, чем помещается в мантиссу Double: через `Double` копейки бы пропали.
    @Test
    fun hugeAmountKeepsEveryDigit() {
        assertEquals("999${NBSP}999${NBSP}999${NBSP}999${NBSP}999,99$NBSP₽", formatMoney(99999999999999999, "RUB"))
    }

    @Test
    fun percentRoundsHalfUp() {
        assertEquals("13%", formatPercent(0.125))
        assertEquals("12%", formatPercent(0.1249))
        assertEquals("0%", formatPercent(0.0))
    }

    @Test
    fun percentSignFollowsFlag() {
        assertEquals("+25%", formatPercent(0.25, signed = true))
        assertEquals("-15%", formatPercent(-0.15, signed = true))
        assertEquals("25%", formatPercent(0.25))
    }

    @Test
    fun amountIsParsedIntoMinorUnits() {
        assertEquals(150000L, parseAmountMinor("1500"))
        assertEquals(150050L, parseAmountMinor("1500,50"))
        assertEquals(150050L, parseAmountMinor("1500.50"))
        assertEquals(150500L, parseAmountMinor("1505,"))
        assertEquals(150500L, parseAmountMinor("1 505"))
    }

    @Test
    fun garbageAmountIsNotANumber() {
        assertNull(parseAmountMinor(""))
        assertNull(parseAmountMinor(","))
        assertNull(parseAmountMinor("-100"))
        assertNull(parseAmountMinor("1500,505"))
        assertNull(parseAmountMinor("сто"))
        assertNull(parseAmountMinor("9999999999999999"))
    }

    // Обратно в поле ввода: разряды не группируются, иначе правка строки ломает разбор.
    @Test
    fun amountGoesBackIntoTheField() {
        assertEquals("1500", formatAmountInput(150000))
        assertEquals("1500,05", formatAmountInput(150005))
        assertEquals("0", formatAmountInput(0))
    }
}
