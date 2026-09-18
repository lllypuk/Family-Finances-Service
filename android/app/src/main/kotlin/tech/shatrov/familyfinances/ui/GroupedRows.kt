package tech.shatrov.familyfinances.ui

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.RectangleShape
import tech.shatrov.familyfinances.theme.AppColors
import tech.shatrov.familyfinances.theme.Dimens

/** Место строки в группе: от него зависят скругления и разделитель сверху. */
internal enum class RowPlace { ONLY, FIRST, MIDDLE, LAST }

internal fun rowPlace(
    index: Int,
    count: Int,
): RowPlace = when {
    count == 1 -> RowPlace.ONLY
    index == 0 -> RowPlace.FIRST
    index == count - 1 -> RowPlace.LAST
    else -> RowPlace.MIDDLE
}

/**
 * Строка группового списка: соседние строки сливаются в одну панель на фоне холста, а не в
 * отдельные карточки. Заливка [AppColors.surface], края группы скруглены, между соседями —
 * волосяная линия по цвету рамки. `onClick` ставится до внутреннего отступа, чтобы нажималась
 * вся строка и волна не выходила за скругление.
 */
@OptIn(ExperimentalFoundationApi::class)
internal fun Modifier.groupedRow(
    place: RowPlace,
    colors: AppColors,
    onLongClick: (() -> Unit)? = null,
    onLongClickLabel: String? = null,
    onClick: (() -> Unit)? = null,
): Modifier {
    val radius = Dimens.RADIUS_M
    val shape = when (place) {
        RowPlace.ONLY -> RoundedCornerShape(radius)
        RowPlace.FIRST -> RoundedCornerShape(topStart = radius, topEnd = radius)
        RowPlace.LAST -> RoundedCornerShape(bottomStart = radius, bottomEnd = radius)
        RowPlace.MIDDLE -> RectangleShape
    }
    val divided = place == RowPlace.MIDDLE || place == RowPlace.LAST
    return this
        .clip(shape)
        .background(colors.surface)
        .then(
            when {
                onClick == null -> Modifier

                onLongClick == null -> Modifier.clickable(onClick = onClick)

                else -> Modifier.combinedClickable(
                    onLongClickLabel = onLongClickLabel,
                    onLongClick = onLongClick,
                    onClick = onClick,
                )
            },
        )
        .drawBehind {
            if (divided) {
                val inset = Dimens.SPACE_3.toPx()
                drawLine(
                    color = colors.border,
                    start = Offset(inset, 0f),
                    end = Offset(size.width - inset, 0f),
                    strokeWidth = Dimens.HAIRLINE.toPx(),
                )
            }
        }
        .padding(horizontal = Dimens.SPACE_3, vertical = Dimens.SPACE_2)
}
