package tech.shatrov.familyfinances.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
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
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.message

/** Бэкапы базы: создание «плюсом» в шапке и удаление файла. Ввода на экране нет. */
@Composable
fun BackupsScreen(
    state: BackupsUiState,
    onRetry: () -> Unit,
    onCreate: () -> Unit,
    onDelete: (String) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var confirming by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
    ) {
        SettingsHeader(R.string.settings_backups, enabled = !state.busy, onBack = onBack) {
            if (state.busy) {
                CircularProgressIndicator(modifier = Modifier.size(Dimens.SPACE_6))
            } else if (state is BackupsUiState.Ready) {
                IconButton(onClick = onCreate) {
                    Icon(AppIcons.Plus, contentDescription = stringResource(R.string.settings_backup_create))
                }
            }
        }

        when (state) {
            BackupsUiState.Loading -> Centered { CircularProgressIndicator() }

            is BackupsUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(
                    onClick = onRetry,
                    modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                ) {
                    Text(stringResource(R.string.retry))
                }
            }

            is BackupsUiState.Ready -> BackupList(state) { confirming = it }
        }
    }

    val target = confirming
    if (target != null) {
        AlertDialog(
            onDismissRequest = { confirming = null },
            title = { Text(stringResource(R.string.settings_backup_delete_confirm)) },
            text = { Text(target) },
            confirmButton = {
                TextButton(onClick = {
                    confirming = null
                    onDelete(target)
                }) {
                    Text(stringResource(R.string.settings_backup_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirming = null }) { Text(stringResource(R.string.cancel)) }
            },
        )
    }
}

@Composable
private fun BackupList(
    state: BackupsUiState.Ready,
    onConfirm: (String) -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_2),
    ) {
        val error = state.error
        if (error != null) {
            item(key = "error") {
                Text(
                    text = error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }

        item(key = "hint") {
            Text(
                text = stringResource(R.string.settings_backup_hint),
                style = MaterialTheme.typography.bodySmall,
            )
        }

        items(state.rows, key = { it.name }) { row ->
            BackupItem(row, busy = state.busy, onConfirm = onConfirm)
        }

        if (state.truncated) {
            item(key = "truncated") {
                Text(
                    text = stringResource(R.string.settings_list_truncated, state.rows.size, state.total),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

@Composable
private fun BackupItem(
    row: BackupRow,
    busy: Boolean,
    onConfirm: (String) -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(Dimens.SPACE_4),
            verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
        ) {
            Text(text = row.name, style = MaterialTheme.typography.titleMedium)
            Row(
                horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = row.created,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f),
                )
                Text(text = row.size, style = MaterialTheme.typography.bodyMedium)
            }
            TextButton(
                onClick = { onConfirm(row.name) },
                enabled = !busy,
                modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(
                    text = stringResource(R.string.settings_backup_delete),
                    color = MaterialTheme.colorScheme.error,
                )
            }
        }
    }
}
