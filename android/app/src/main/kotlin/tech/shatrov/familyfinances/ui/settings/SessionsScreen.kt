package tech.shatrov.familyfinances.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
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
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.message
import java.util.UUID

/** Сессии пользователя: где он вошёл и когда оттуда обращались. Ввода на экране нет. */
@Composable
fun SessionsScreen(
    state: SessionsUiState,
    onRetry: () -> Unit,
    onRevoke: (UUID) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var confirming by remember { mutableStateOf<UUID?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
    ) {
        SettingsHeader(R.string.settings_sessions, enabled = !state.busy, onBack = onBack)

        when (state) {
            SessionsUiState.Loading -> Centered { CircularProgressIndicator() }

            is SessionsUiState.Failure -> Centered {
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

            is SessionsUiState.Ready -> SessionList(state) { confirming = it }
        }
    }

    val target = confirming
    if (target != null) {
        AlertDialog(
            onDismissRequest = { confirming = null },
            title = { Text(stringResource(R.string.settings_session_revoke_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    confirming = null
                    onRevoke(target)
                }) {
                    Text(stringResource(R.string.settings_session_revoke))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirming = null }) { Text(stringResource(R.string.cancel)) }
            },
        )
    }
}

@Composable
private fun SessionList(
    state: SessionsUiState.Ready,
    onConfirm: (UUID) -> Unit,
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

        items(state.rows, key = { it.id }) { row ->
            SessionItem(row, busy = state.revoking != null, onConfirm = onConfirm)
        }

        item(key = "hint") {
            Text(
                text = stringResource(R.string.settings_session_last_used_hint),
                style = MaterialTheme.typography.bodySmall,
            )
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
private fun SessionItem(
    row: SessionRow,
    busy: Boolean,
    onConfirm: (UUID) -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(Dimens.SPACE_4),
            verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
        ) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = row.deviceName ?: stringResource(R.string.settings_session_unnamed),
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                if (row.current) {
                    Text(
                        text = stringResource(R.string.settings_session_current),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
            }
            Text(
                text = stringResource(R.string.settings_session_created, row.created),
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                text = stringResource(R.string.settings_session_last_used, row.lastUsed),
                style = MaterialTheme.typography.bodyMedium,
            )
            // У текущей отзыва нет: это «Выйти» в корне, и он ещё чистит токен на телефоне.
            if (!row.current) {
                TextButton(
                    onClick = { onConfirm(row.id) },
                    enabled = !busy,
                    modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                ) {
                    Text(
                        text = stringResource(R.string.settings_session_revoke),
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        }
    }
}
