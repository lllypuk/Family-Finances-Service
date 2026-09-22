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

    /** Разделитель между строками одной группы. */
    val HAIRLINE = 1.dp

    /**
     * Однострочная строка плотного списка. Ниже [TOUCH_MIN], потому что строки идут без зазора и
     * тянутся на всю ширину: промахнуться некуда, а 48 на каждой удвоило бы длину справочника.
     */
    val ROW_COMPACT = 40.dp

    /** Аватар категории в строках операций, топе главной и «Обзоре». */
    val AVATAR_S = 32.dp

    /** Аватар категории в листах и справочнике: 40 — это вся высота [ROW_COMPACT]. */
    val AVATAR_M = 36.dp

    /** Высота превью картинки над её строками на экране распознавания. */
    val PREVIEW_HEIGHT = 120.dp

    /** Нижний отступ прокручиваемого списка под плавающей кнопкой. */
    val FAB_CLEARANCE = 88.dp
}
