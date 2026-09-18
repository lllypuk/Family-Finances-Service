package tech.shatrov.familyfinances.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message

/** Форма счёта: имя; у существующего — архив и, для админа, удаление через подтверждение. */
@Composable
fun AccountEditScreen(
    state: AccountEditUiState,
    onNameChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onToggleArchive: () -> Unit,
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
        SettingsHeader(
            if (state.editing) R.string.settings_account_edit else R.string.settings_account_new,
            enabled = !state.submitting,
            onBack = onBack,
        )

        OutlinedTextField(
            value = state.name,
            onValueChange = onNameChange,
            label = { Text(stringResource(R.string.settings_account_name)) },
            singleLine = true,
            isError = state.fieldErrors.containsKey(ACCOUNT_FIELD_NAME),
            supportingText = { FieldError(state.fieldErrors[ACCOUNT_FIELD_NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        if (state.archived) {
            Text(
                text = stringResource(R.string.settings_account_archived_hint),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

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
                Text(stringResource(R.string.settings_save))
            }
        }

        if (state.editing) {
            OutlinedButton(
                onClick = onToggleArchive,
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(
                    stringResource(
                        if (state.archived) R.string.settings_account_unarchive else R.string.settings_account_archive,
                    ),
                )
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
                Text(stringResource(R.string.settings_account_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.settings_account_delete_confirm)) },
            text = { Text(state.savedName) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.settings_account_delete))
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
