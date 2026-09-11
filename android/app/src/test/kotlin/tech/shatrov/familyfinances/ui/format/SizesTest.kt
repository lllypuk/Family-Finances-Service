package tech.shatrov.familyfinances.ui.format

import org.junit.Assert.assertEquals
import org.junit.Test

private const val NBSP = " "

/** Размер файла: целочисленный счёт, поэтому границы единиц не зависят от округления. */
class SizesTest {
    @Test
    fun bytesStayWholeBelowKilobyte() {
        assertEquals("0${NBSP}Б", formatBytes(0))
        assertEquals("1023${NBSP}Б", formatBytes(1023))
    }

    @Test
    fun kilobyteBoundarySwitchesUnit() {
        assertEquals("1,0${NBSP}КБ", formatBytes(1024))
        assertEquals("2,0${NBSP}КБ", formatBytes(2048))
        assertEquals("1023,0${NBSP}КБ", formatBytes(1023 * 1024L))
    }

    @Test
    fun megabyteBoundarySwitchesUnit() {
        assertEquals("1,0${NBSP}МБ", formatBytes(1024L * 1024))
        assertEquals("12,3${NBSP}МБ", formatBytes(12_876_906))
    }

    /** Единица выбирается до округления: 1048575 Б это ещё «КБ», но печатается уже мегабайтом. */
    @Test
    fun roundingUpCarriesTheUnit() {
        assertEquals("1023,9${NBSP}КБ", formatBytes(1_048_524))
        assertEquals("1,0${NBSP}МБ", formatBytes(1_048_525))
        assertEquals("1,0${NBSP}МБ", formatBytes(1_048_575))
    }

    @Test
    fun gigabyteIsReachedToo() {
        assertEquals("2,0${NBSP}ГБ", formatBytes(2 * 1024L * 1024 * 1024))
    }
}
