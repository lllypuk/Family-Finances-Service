package tech.shatrov.familyfinances.ui.settings

import androidx.compose.foundation.clickable
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
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.message
import java.util.UUID

/** Пользователи семьи: строка ведёт в форму правки, «плюс» в шапке — в форму создания. */
@Composable
fun UsersScreen(
    state: UsersUiState,
    onRetry: () -> Unit,
    onAdd: () -> Unit,
    onOpen: (UUID) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
    ) {
        SettingsHeader(R.string.settings_users, enabled = true, onBack = onBack) {
            IconButton(onClick = onAdd) {
                Icon(AppIcons.Plus, contentDescription = stringResource(R.string.settings_user_add))
            }
        }

        when (state) {
            UsersUiState.Loading -> Centered { CircularProgressIndicator() }

            is UsersUiState.Failure -> Centered {
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

            is UsersUiState.Ready -> UserList(state, onOpen)
        }
    }
}

@Composable
private fun UserList(
    state: UsersUiState.Ready,
    onOpen: (UUID) -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_2),
    ) {
        items(state.rows, key = { it.id }) { row -> UserItem(row, onOpen) }

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
private fun UserItem(
    row: UserRow,
    onOpen: (UUID) -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onOpen(row.id) },
    ) {
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
                    text = row.name,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                if (row.self) {
                    Text(
                        text = stringResource(R.string.settings_user_self),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
            }
            Text(text = row.email, style = MaterialTheme.typography.bodyMedium)
            Text(
                text = stringResource(
                    if (row.admin) R.string.settings_role_admin else R.string.settings_role_member,
                ),
                style = MaterialTheme.typography.bodyMedium,
            )
            if (!row.active) {
                Text(
                    text = stringResource(R.string.settings_user_inactive),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.error,
                )
            }
        }
    }
}
