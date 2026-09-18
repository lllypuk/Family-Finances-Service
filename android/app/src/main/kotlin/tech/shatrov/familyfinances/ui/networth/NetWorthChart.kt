package tech.shatrov.familyfinances.ui.networth

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp

private val CHART_HEIGHT = 48.dp
private val LINE_WIDTH = 2.dp
private val DOT_RADIUS = 3.dp

/**
 * Точки ряда в долях области: `x` слева направо, `y` сверху вниз. Ноль всегда внутри шкалы, так что
 * [zeroY] — ось нуля; одна корзина встаёт по центру.
 */
internal class ChartGeometry(
    val points: List<Offset>,
    val zeroY: Float,
)

internal fun chartGeometry(values: List<Long>): ChartGeometry {
    val high = maxOf(values.maxOrNull() ?: 0L, 0L).toDouble()
    val low = minOf(values.minOrNull() ?: 0L, 0L).toDouble()
    val span = (high - low).takeIf { it > 0.0 } ?: 1.0
    val step = if (values.size > 1) 1f / (values.size - 1) else 0f
    val points = values.mapIndexed { index, value ->
        Offset(if (values.size > 1) index * step else 0.5f, ((high - value) / span).toFloat())
    }
    return ChartGeometry(points, (high / span).toFloat())
}

/** Чистый капитал по месяцам без осей и подписей; ось нуля — только когда капитал уходил в минус. */
@Composable
fun NetWorthChart(
    values: List<Long>,
    modifier: Modifier = Modifier,
) {
    val line = MaterialTheme.colorScheme.primary
    val axis = MaterialTheme.colorScheme.outlineVariant
    val geometry = chartGeometry(values)
    val negative = values.any { it < 0 }
    Canvas(
        modifier = modifier
            .fillMaxWidth()
            .height(CHART_HEIGHT),
    ) {
        // Поле под толщину линии: иначе крайние значения срезаются краем холста наполовину.
        val inset = DOT_RADIUS.toPx()
        val width = size.width - 2 * inset
        val height = size.height - 2 * inset
        fun place(point: Offset) = Offset(inset + point.x * width, inset + point.y * height)

        if (negative) {
            val y = inset + geometry.zeroY * height
            drawLine(axis, Offset(inset, y), Offset(inset + width, y), strokeWidth = 1.dp.toPx())
        }
        val points = geometry.points.map(::place)
        if (points.size == 1) {
            drawCircle(line, DOT_RADIUS.toPx(), points.single())
        } else if (points.size > 1) {
            val path = Path().apply {
                moveTo(points.first().x, points.first().y)
                points.drop(1).forEach { lineTo(it.x, it.y) }
            }
            drawPath(
                path,
                line,
                style = Stroke(width = LINE_WIDTH.toPx(), cap = StrokeCap.Round, join = StrokeJoin.Round),
            )
        }
    }
}
