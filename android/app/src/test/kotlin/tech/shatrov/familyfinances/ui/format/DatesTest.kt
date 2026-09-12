package tech.shatrov.familyfinances.ui.format

import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.OffsetDateTime
import java.time.ZoneId

/** Служебные метки приходят в UTC, а показываются в зоне семьи. */
class DatesTest {
    private val at = OffsetDateTime.parse("2026-09-10T23:30:00Z")

    @Test
    fun sameInstantDiffersBetweenZones() {
        assertEquals("11 сентября 2026, 02:30", formatDateTime(at, ZoneId.of("Europe/Moscow")))
        assertEquals("10 сентября 2026, 23:30", formatDateTime(at, ZoneId.of("UTC")))
    }

    @Test
    fun offsetOfTheMarkDoesNotLeakIntoTheOutput() {
        val shifted = OffsetDateTime.parse("2026-09-11T05:30:00+06:00")

        val moscow = ZoneId.of("Europe/Moscow")

        assertEquals(formatDateTime(at, moscow), formatDateTime(shifted, moscow))
    }
}
