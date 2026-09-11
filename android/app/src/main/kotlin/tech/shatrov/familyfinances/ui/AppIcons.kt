package tech.shatrov.familyfinances.ui

import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.graphics.vector.PathParser
import androidx.compose.ui.unit.dp

private const val VIEWPORT = 24f
private const val STROKE = 2f

/**
 * Иконки интерфейса контурами Lucide. В `material-icons-core` нет ни метки, ни выхода в одном
 * стиле с домом и списком, а `material-icons-extended` тянет тысячи векторов ради шести.
 */
object AppIcons {
    val Home: ImageVector =
        icon("home", "M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z")

    val List: ImageVector =
        icon("list", "M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01")

    val Tag: ImageVector =
        icon(
            "tag",
            "M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L3 13V3h10l7.6 7.6a2 2 0 0 1 0 2.8z",
            "M7.5 6.5a1 1 0 1 0 0 2 1 1 0 1 0 0-2z",
        )

    val Target: ImageVector =
        icon(
            "target",
            "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20z",
            "M12 18a6 6 0 1 0 0-12 6 6 0 0 0 0 12z",
            "M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z",
        )

    val Plus: ImageVector = icon("plus", "M12 5v14M5 12h14")

    val ArrowLeft: ImageVector = icon("arrow-left", "m12 19-7-7 7-7M19 12H5")

    val User: ImageVector =
        icon(
            "user",
            "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2",
            "M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
        )
}

/** Цвет контура чёрный только формально: `Icon` перекрашивает вектор целиком под `tint`. */
private fun icon(
    name: String,
    vararg paths: String,
): ImageVector {
    val builder = ImageVector.Builder(
        name = name,
        defaultWidth = VIEWPORT.dp,
        defaultHeight = VIEWPORT.dp,
        viewportWidth = VIEWPORT,
        viewportHeight = VIEWPORT,
    )
    for (data in paths) {
        builder.addPath(
            pathData = PathParser().parsePathString(data).toNodes(),
            stroke = SolidColor(Color.Black),
            strokeLineWidth = STROKE,
            strokeLineCap = StrokeCap.Round,
            strokeLineJoin = StrokeJoin.Round,
        )
    }
    return builder.build()
}
