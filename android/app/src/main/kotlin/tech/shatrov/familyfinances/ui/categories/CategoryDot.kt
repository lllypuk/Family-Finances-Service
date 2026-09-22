package tech.shatrov.familyfinances.ui.categories

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import tech.shatrov.familyfinances.theme.LocalAppColors

/** Метка категории в списке; кольцо держит бледный серверный цвет на светлом фоне (≥ 3:1). */
@Composable
fun CategoryDot(
    hex: String,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier
            .background(parseCategoryColor(hex, MaterialTheme.colorScheme.outline), CircleShape)
            .border(1.dp, LocalAppColors.current.textSecondary, CircleShape),
    )
}
