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
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
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
) {
    ModalBottomSheet(onDismissRequest = onDismiss) {
        CategorySheetContent(categories, selected, onSelect)
    }
}

// Тело листа отдельно от `ModalBottomSheet`: тот живёт в своём окне с анимацией и в
// Robolectric ненадёжен, а проверять нужно выбор.
@Composable
internal fun CategorySheetContent(
    categories: List<Category>,
    selected: UUID?,
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
        item {
            CategoryRow(stringResource(R.string.filter_all_categories), selected == null) { onSelect(null) }
        }
        items(categories, key = { it.id }) { category ->
            CategoryRow(category.name, selected == category.id) { onSelect(category.id) }
        }
    }
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
