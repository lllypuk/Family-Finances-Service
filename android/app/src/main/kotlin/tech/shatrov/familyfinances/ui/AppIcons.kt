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
 * Иконки интерфейса контурами Lucide. В `material-icons-core` нет ни метки, ни профиля в одном
 * стиле с домом и списком, а `material-icons-extended` тянет тысячи векторов ради семи.
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

    val Landmark: ImageVector =
        icon("landmark", "M3 22h18", "M6 18v-7", "M10 18v-7", "M14 18v-7", "M18 18v-7", "M12 2l8 5H4z")

    val Plus: ImageVector = icon("plus", "M12 5v14M5 12h14")

    val ArrowLeft: ImageVector = icon("arrow-left", "m12 19-7-7 7-7M19 12H5")

    val ChevronDown: ImageVector = icon("chevron-down", "m6 9 6 6 6-6")

    val ChevronUp: ImageVector = icon("chevron-up", "m18 15-6-6-6 6")

    val ChevronLeft: ImageVector = icon("chevron-left", "m15 18-6-6 6-6")

    val ChevronRight: ImageVector = icon("chevron-right", "m9 18 6-6-6-6")

    val Repeat: ImageVector =
        icon("repeat", "m17 2 4 4-4 4", "M3 11v-1a4 4 0 0 1 4-4h14", "m7 22-4-4 4-4", "M21 13v1a4 4 0 0 1-4 4H3")

    val User: ImageVector =
        icon(
            "user",
            "M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2",
            "M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
        )

    val Calendar: ImageVector =
        icon(
            "calendar",
            "M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z",
            "M16 2v4",
            "M8 2v4",
            "M3 10h18",
        )

    val ScanLine: ImageVector =
        icon(
            "scan-line",
            "M3 7V5a2 2 0 0 1 2-2h2",
            "M17 3h2a2 2 0 0 1 2 2v2",
            "M21 17v2a2 2 0 0 1-2 2h-2",
            "M7 21H5a2 2 0 0 1-2-2v-2",
            "M7 12h10",
        )

    val Eye: ImageVector =
        icon(
            "eye",
            "M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7z",
            "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
        )

    val EyeOff: ImageVector =
        icon(
            "eye-off",
            "M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94",
            "M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19",
            "M9.88 9.88a3 3 0 1 0 4.24 4.24",
            "M2 2l20 20",
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
