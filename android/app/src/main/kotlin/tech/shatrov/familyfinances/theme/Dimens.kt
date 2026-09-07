package tech.shatrov.familyfinances.theme

import androidx.compose.ui.unit.dp

/** Шаг отступов и радиусов: значения вне этой таблицы в разметке не появляются. */
object Dimens {
    val SPACE_1 = 4.dp
    val SPACE_2 = 8.dp
    val SPACE_3 = 12.dp
    val SPACE_4 = 16.dp
    val SPACE_6 = 24.dp
    val SPACE_8 = 32.dp
    val RADIUS_S = 8.dp
    val RADIUS_M = 12.dp
    val RADIUS_L = 16.dp

    /** Минимальная цель нажатия — рекомендация Material; меньше неё кнопок не бывает. */
    val TOUCH_MIN = 48.dp
    val ICON_SIZE = 24.dp

    /** Обводка выбранного образца цвета. */
    val BORDER = 2.dp
}
