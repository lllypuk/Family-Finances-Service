package tech.shatrov.familyfinances.ui.categories

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertEquals
import org.junit.Test

class CategoryColorTest {
    private val fallback = Color(0xFF123456)

    @Test
    fun parses_rrggbb() {
        assertEquals(Color(0xFFF7DC6F), parseCategoryColor("#F7DC6F", fallback))
        assertEquals(Color(0xFF85C1E9), parseCategoryColor("#85c1e9", fallback))
    }

    @Test
    fun malformed_falls_back() {
        listOf("F7DC6F", "#FFF", "#F7DC6F00", "", "#GGGGGG", "#+12345", "#-12345").forEach {
            assertEquals(it, fallback, parseCategoryColor(it, fallback))
        }
    }
}
