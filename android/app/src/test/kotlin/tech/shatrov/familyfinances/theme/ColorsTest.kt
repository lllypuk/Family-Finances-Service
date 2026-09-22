package tech.shatrov.familyfinances.theme

import androidx.compose.ui.graphics.Color
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.pow

/** WCAG-контраст обеих палитр; падение называет тему и пару токенов. */
class ColorsTest {
    private val palettes = mapOf("dark" to DarkColors, "light" to LightColors)

    @Test
    fun text_and_accents_readable_on_every_layer() {
        assertPairs(MIN_TEXT) { c ->
            val fg = listOf(
                "textPrimary" to c.textPrimary,
                "textSecondary" to c.textSecondary,
                "action" to c.action,
                "income" to c.income,
                "warning" to c.warning,
                "expense" to c.expense,
            )
            val bg = listOf("canvas" to c.canvas, "surface" to c.surface, "elevated" to c.elevated)
            fg.flatMap { f -> bg.map { b -> f to b } }
        }
    }

    @Test
    fun inverse_text_readable_on_filled_action_and_expense() {
        assertPairs(MIN_TEXT) { c ->
            listOf(
                ("textInverse" to c.textInverse) to ("action" to c.action),
                ("textInverse" to c.textInverse) to ("expense" to c.expense),
            )
        }
    }

    @Test
    fun category_dot_ring_visible_on_canvas_and_surface() {
        assertPairs(MIN_UI) { c ->
            listOf(
                ("textSecondary" to c.textSecondary) to ("canvas" to c.canvas),
                ("textSecondary" to c.textSecondary) to ("surface" to c.surface),
            )
        }
    }

    // Тёмная оставлена как до плана 21 (1,8:1): без кольца поля и выключенный Switch в светлой не видны.
    @Test
    fun outline_visible_on_every_layer_in_light() {
        assertPairs(MIN_UI, palettes = mapOf("light" to LightColors)) { c ->
            listOf("canvas" to c.canvas, "surface" to c.surface, "elevated" to c.elevated)
                .map { b -> ("outline" to c.outline) to b }
        }
    }

    @Test
    fun contrast_formula_matches_wcag_extremes() {
        assertTrue(contrast(Color.Black, Color.White) in 20.99..21.01)
        assertTrue(contrast(Color.White, Color.White) in 0.99..1.01)
    }

    private fun assertPairs(
        min: Double,
        palettes: Map<String, AppColors> = this.palettes,
        pairs: (AppColors) -> List<Pair<Pair<String, Color>, Pair<String, Color>>>,
    ) {
        val failures = palettes.flatMap { (theme, colors) ->
            pairs(colors).mapNotNull { (fg, bg) ->
                val ratio = contrast(fg.second, bg.second)
                if (ratio >= min) null else "$theme: ${fg.first} на ${bg.first} = ${"%.2f".format(ratio)}"
            }
        }
        assertTrue("контраст ниже $min:\n" + failures.joinToString("\n"), failures.isEmpty())
    }

    private fun contrast(
        a: Color,
        b: Color,
    ): Double {
        val la = luminance(a)
        val lb = luminance(b)
        return (maxOf(la, lb) + 0.05) / (minOf(la, lb) + 0.05)
    }

    private fun luminance(c: Color): Double {
        fun channel(v: Float): Double {
            val s = v.toDouble()
            return if (s <= 0.03928) s / 12.92 else ((s + 0.055) / 1.055).pow(2.4)
        }
        return 0.2126 * channel(c.red) + 0.7152 * channel(c.green) + 0.0722 * channel(c.blue)
    }

    private companion object {
        const val MIN_TEXT = 4.5
        const val MIN_UI = 3.0
    }
}
