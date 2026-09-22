package tech.shatrov.familyfinances.ui.categories

import android.icu.text.BreakIterator
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.unit.Dp
import tech.shatrov.familyfinances.theme.Dimens

private const val FILL_ALPHA = 0.18f

/** Значок и цвет категории картинкой: эмодзи из `icon` или первая буква имени. */
@Composable
fun CategoryAvatar(
    icon: String,
    color: String,
    name: String,
    size: Dp,
    modifier: Modifier = Modifier,
    glyph: String = categoryGlyph(icon, name),
) {
    val tint = parseCategoryColor(color, MaterialTheme.colorScheme.outline)
    Box(
        contentAlignment = Alignment.Center,
        modifier = modifier
            .size(size)
            .background(tint.copy(alpha = FILL_ALPHA), CircleShape)
            .border(Dimens.HAIRLINE, tint, CircleShape)
            .clearAndSetSemantics { contentDescription = name },
    ) {
        Text(text = glyph, color = tint, style = MaterialTheme.typography.titleMedium)
    }
}

/**
 * Первый графемный кластер `icon`, если это эмодзи, иначе первая буква имени заглавной.
 * Старые данные несут в `icon` слова (`default`, `food`) — их первая буква значком не считается.
 */
fun categoryGlyph(
    icon: String,
    name: String,
): String = categoryEmoji(icon) ?: firstGrapheme(name.trim()).uppercase().ifEmpty { "?" }

/** Первый графемный кластер `icon`, если это эмодзи; слово или пустое поле — `null`. */
fun categoryEmoji(icon: String): String? {
    val cluster = firstGrapheme(icon.trim())
    val emoji = cluster.length > 1 || (cluster.isNotEmpty() && !isAsciiLetterOrDigit(cluster.codePointAt(0)))
    return cluster.takeIf { emoji }
}

/** Подпись чипа: эмодзи перед именем; буква-заглушка в чипе только повторила бы имя. */
fun categoryChipLabel(
    icon: String,
    name: String,
): String = categoryEmoji(icon)?.let { "$it $name" } ?: name

/** Первый графемный кластер; `java.text.BreakIterator` не знает ZWJ-последовательностей, `android.icu` знает. */
fun firstGrapheme(text: String): String {
    if (text.isEmpty()) return text
    val iterator = BreakIterator.getCharacterInstance()
    iterator.setText(text)
    return text.substring(0, iterator.next())
}

private fun isAsciiLetterOrDigit(codePoint: Int): Boolean =
    codePoint in '0'.code..'9'.code || codePoint in 'a'.code..'z'.code || codePoint in 'A'.code..'Z'.code
