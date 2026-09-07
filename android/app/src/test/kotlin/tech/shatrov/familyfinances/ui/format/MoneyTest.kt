package tech.shatrov.familyfinances.ui.format

import org.junit.Assert.assertEquals
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
}
