package tech.shatrov.familyfinances.ui.categories

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.CategoryType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message
import java.util.UUID

/** Форма категории. При правке тип и родитель не показываются: сервер их не меняет. */
@Composable
fun CategoryEditScreen(
    state: CategoryEditUiState,
    onNameChange: (String) -> Unit,
    onTypeChange: (CategoryType) -> Unit,
    onColorChange: (String) -> Unit,
    onIconChange: (String) -> Unit,
    onParentChange: (UUID?) -> Unit,
    onSubmit: () -> Unit,
    onDelete: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var deleteConfirmShown by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(AppIcons.ArrowLeft, contentDescription = stringResource(R.string.back))
            }
            Text(
                text = stringResource(
                    if (state.editing) R.string.category_edit_title else R.string.category_new_title,
                ),
                style = MaterialTheme.typography.headlineSmall,
            )
        }

        OutlinedTextField(
            value = state.name,
            onValueChange = onNameChange,
            label = { Text(stringResource(R.string.category_name)) },
            singleLine = true,
            isError = state.fieldErrors.containsKey(CategoryField.NAME),
            supportingText = { FieldError(state.fieldErrors[CategoryField.NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        if (!state.editing) {
            ChipRow {
                item {
                    Chip(stringResource(R.string.type_expense), state.type == CategoryType.expense) {
                        onTypeChange(CategoryType.expense)
                    }
                }
                item {
                    Chip(stringResource(R.string.type_income), state.type == CategoryType.income) {
                        onTypeChange(CategoryType.income)
                    }
                }
            }
            FieldError(state.fieldErrors[CategoryField.TYPE])

            Text(stringResource(R.string.category_parent), style = MaterialTheme.typography.bodySmall)
            ChipRow {
                item {
                    Chip(stringResource(R.string.category_no_parent), state.parentId == null) {
                        onParentChange(null)
                    }
                }
                items(state.parents) { parent: Category ->
                    Chip(parent.name, state.parentId == parent.id) { onParentChange(parent.id) }
                }
            }
            FieldError(state.fieldErrors[CategoryField.PARENT])
        }

        Text(stringResource(R.string.category_color), style = MaterialTheme.typography.bodySmall)
        Palette(state.color, onColorChange)
        FieldError(state.fieldErrors[CategoryField.COLOR])

        OutlinedTextField(
            value = state.icon,
            onValueChange = onIconChange,
            label = { Text(stringResource(R.string.category_icon)) },
            singleLine = true,
            isError = state.fieldErrors.containsKey(CategoryField.ICON),
            supportingText = { FieldError(state.fieldErrors[CategoryField.ICON]) },
            modifier = Modifier.fillMaxWidth(),
        )

        val error = state.error
        if (error != null) {
            Text(
                text = error.message(LocalContext.current.resources),
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        Button(
            onClick = onSubmit,
            enabled = state.canSubmit,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            if (state.submitting) {
                CircularProgressIndicator(modifier = Modifier.heightIn(max = Dimens.ICON_SIZE))
            } else {
                Text(stringResource(R.string.category_save))
            }
        }

        // Удаление — админское: у member кнопки нет вовсе, а не отказ по нажатию.
        if (state.editing && state.canDelete) {
            TextButton(
                onClick = { deleteConfirmShown = true },
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.category_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.category_delete_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.category_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteConfirmShown = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }
}

@Composable
private fun Palette(
    selected: String,
    onPick: (String) -> Unit,
) {
    Row(horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2)) {
        for (hex in CategoryPalette) {
            val outline = MaterialTheme.colorScheme.onSurface
            val marked = if (hex == selected) Modifier.border(Dimens.BORDER, outline, CircleShape) else Modifier
            Surface(
                color = parseCategoryColor(hex, outline),
                shape = CircleShape,
                modifier = Modifier
                    .size(Dimens.SPACE_6)
                    .then(marked)
                    .clickable { onPick(hex) }
                    .semantics { contentDescription = hex },
            ) {}
        }
    }
}
