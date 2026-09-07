package tech.shatrov.familyfinances.ui.transactions

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import tech.shatrov.familyfinances.theme.Dimens

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
    onClick: () -> Unit,
) {
    FilterChip(selected = selected, onClick = onClick, label = { Text(label) })
}
