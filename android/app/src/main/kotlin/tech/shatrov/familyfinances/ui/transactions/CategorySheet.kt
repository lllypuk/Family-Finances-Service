package tech.shatrov.familyfinances.ui.transactions

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.selection.selectable
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import java.util.UUID

/** Категорий у семьи десятки — чипами ряд не влезает, выбор уезжает в лист. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun CategorySheet(
    categories: List<Category>,
    selected: UUID?,
    onSelect: (UUID?) -> Unit,
    onDismiss: () -> Unit,
    allowAll: Boolean = true,
) {
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        CategorySheetContent(categories, selected, allowAll) { picked ->
            // Снятие с композиции убрало бы лист одним кадром: сначала анимация, потом выбор.
            scope.launch { sheetState.hide() }.invokeOnCompletion { if (!sheetState.isVisible) onSelect(picked) }
        }
    }
}

// Тело листа отдельно от `ModalBottomSheet`: тот живёт в своём окне с анимацией и в
// Robolectric ненадёжен, а проверять нужно выбор.
@Composable
internal fun CategorySheetContent(
    categories: List<Category>,
    selected: UUID?,
    allowAll: Boolean = true,
    onSelect: (UUID?) -> Unit,
) {
    LazyColumn(modifier = Modifier.fillMaxWidth()) {
        item {
            Text(
                text = stringResource(R.string.filter_category),
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
            )
        }
        if (allowAll) {
            item {
                CategoryRow(stringResource(R.string.filter_all_categories), selected == null) { onSelect(null) }
            }
        }
        items(categories, key = { it.id }) { category ->
            CategoryRow(category.path(categories), selected == category.id) { onSelect(category.id) }
        }
    }
}

/** Имя с родителем: одно имя под разными родителями законно, и без пути их не отличить. */
internal fun Category.path(categories: List<Category>): String {
    val parent = parentId?.let { id -> categories.firstOrNull { it.id == id } } ?: return name
    return "${parent.name} / $name"
}

@Composable
private fun CategoryRow(
    label: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .selectable(selected = selected, onClick = onClick)
            .heightIn(min = Dimens.TOUCH_MIN)
            .padding(horizontal = Dimens.SPACE_4),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        val colors = LocalAppColors.current
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge,
            color = if (selected) colors.action else colors.textPrimary,
        )
    }
}
