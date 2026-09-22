package tech.shatrov.familyfinances.ui.transactions

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.toggleable
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.Role
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.categories.CategoryAvatar
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
    onManage: (() -> Unit)? = null,
) {
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    // Снятие с композиции убрало бы лист одним кадром: сначала анимация, потом выбор или переход.
    val afterHide = { action: () -> Unit ->
        scope.launch { sheetState.hide() }.invokeOnCompletion { if (!sheetState.isVisible) action() }
    }
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        CategorySheetContent(
            categories,
            selected,
            allowAll,
            onManage = onManage?.let { manage -> { afterHide(manage) } },
        ) { picked -> afterHide { onSelect(picked) } }
    }
}

/** Фильтр «Операций»: набор категорий уходит только по «Готово», закрытие жестом его отбрасывает. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun CategoryFilterSheet(
    categories: List<Category>,
    selected: Set<UUID>,
    onDone: (Set<UUID>) -> Unit,
    onDismiss: () -> Unit,
    onManage: (() -> Unit)? = null,
) {
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    val afterHide = { action: () -> Unit ->
        scope.launch { sheetState.hide() }.invokeOnCompletion { if (!sheetState.isVisible) action() }
    }
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        CategorySheetContent(
            categories,
            selectedIds = selected,
            onManage = onManage?.let { manage -> { afterHide(manage) } },
        ) { picked -> afterHide { onDone(picked) } }
    }
}

// Тело листа отдельно от `ModalBottomSheet`: тот живёт в своём окне с анимацией и в
// Robolectric ненадёжен, а проверять нужно выбор.
@Composable
internal fun CategorySheetContent(
    categories: List<Category>,
    selected: UUID?,
    allowAll: Boolean = true,
    onManage: (() -> Unit)? = null,
    onSelect: (UUID?) -> Unit,
) {
    LazyColumn(modifier = Modifier.fillMaxWidth()) {
        item { SheetTitle() }
        if (allowAll) {
            item {
                SheetRow(stringResource(R.string.filter_all_categories), selected == null) { onSelect(null) }
            }
        }
        items(categories, key = { it.id }) { category ->
            SheetRow(category.path(categories), selected == category.id, category) { onSelect(category.id) }
        }
        onManage?.let { item { ManageRow(it) } }
    }
}

/** Мультивыбор: «Готово» закреплена под списком, иначе при двадцати категориях до неё не долистать. */
@Composable
internal fun CategorySheetContent(
    categories: List<Category>,
    selectedIds: Set<UUID>,
    onManage: (() -> Unit)? = null,
    onDone: (Set<UUID>) -> Unit,
) {
    // Строки, а не UUID: черновик отметок должен пережить поворот, пока лист открыт.
    var draft by rememberSaveable { mutableStateOf(selectedIds.map(UUID::toString)) }
    Column(modifier = Modifier.fillMaxWidth()) {
        LazyColumn(modifier = Modifier.weight(1f, fill = false)) {
            item { SheetTitle() }
            item {
                CheckRow(stringResource(R.string.filter_all_categories), draft.isEmpty()) { draft = emptyList() }
            }
            items(categories, key = { it.id }) { category ->
                val id = category.id.toString()
                CheckRow(category.path(categories), id in draft, category) {
                    draft = if (id in draft) draft - id else draft + id
                }
            }
            onManage?.let { item { ManageRow(it) } }
        }
        Button(
            onClick = { onDone(draft.mapTo(LinkedHashSet(), UUID::fromString)) },
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2)
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(R.string.filter_done))
        }
    }
}

@Composable
private fun ManageRow(onManage: () -> Unit) {
    TextButton(
        onClick = onManage,
        modifier = Modifier
            .padding(horizontal = Dimens.SPACE_2)
            .heightIn(min = Dimens.TOUCH_MIN),
    ) {
        Text(stringResource(R.string.categories_manage))
    }
}

@Composable
private fun SheetTitle() {
    Text(
        text = stringResource(R.string.filter_category),
        style = MaterialTheme.typography.titleMedium,
        modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
    )
}

@Composable
private fun CheckRow(
    label: String,
    checked: Boolean,
    category: Category? = null,
    onToggle: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .toggleable(value = checked, role = Role.Checkbox, onValueChange = { onToggle() })
            .heightIn(min = Dimens.TOUCH_MIN)
            .padding(horizontal = Dimens.SPACE_4),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Checkbox(checked = checked, onCheckedChange = null)
        category?.let { SheetAvatar(it, Modifier.padding(start = Dimens.SPACE_2)) }
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge,
            color = LocalAppColors.current.textPrimary,
            modifier = Modifier.padding(start = Dimens.SPACE_2),
        )
    }
}

/** Имя с родителем: одно имя под разными родителями законно, и без пути их не отличить. */
internal fun Category.path(categories: List<Category>): String {
    val parent = parentId?.let { id -> categories.firstOrNull { it.id == id } } ?: return name
    return "${parent.name} / $name"
}

@Composable
internal fun SheetRow(
    label: String,
    selected: Boolean,
    category: Category? = null,
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
        category?.let { SheetAvatar(it, Modifier.padding(end = Dimens.SPACE_2)) }
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge,
            color = if (selected) colors.action else colors.textPrimary,
        )
    }
}

@Composable
private fun SheetAvatar(
    category: Category,
    modifier: Modifier = Modifier,
) {
    CategoryAvatar(category.icon, category.color, category.name, Dimens.AVATAR_M, modifier)
}
