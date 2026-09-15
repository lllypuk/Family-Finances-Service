package tech.shatrov.familyfinances.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors

/** Ряд чипов: и фильтры списка, и переключатели формы — одна и та же прокручиваемая строка. */
@Composable
internal fun ChipRow(
    modifier: Modifier = Modifier,
    content: LazyListScope.() -> Unit,
) {
    LazyRow(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        content = content,
    )
}

@Composable
internal fun Chip(
    label: String,
    selected: Boolean,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    // Заливка выбранного чипа отличается от фона на 1,2:1 — выбранность держится на обводке.
    FilterChip(
        selected = selected,
        onClick = onClick,
        enabled = enabled,
        label = { Text(label) },
        border = FilterChipDefaults.filterChipBorder(
            enabled = enabled,
            selected = selected,
            selectedBorderColor = LocalAppColors.current.action,
            selectedBorderWidth = Dimens.BORDER,
        ),
    )
}
