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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.message

/** Счета: активные, под ними свёрнутый архив; «плюс» в шапке — форма создания. */
@Composable
fun AccountsScreen(
    state: AccountsUiState,
    onRetry: () -> Unit,
    onAdd: () -> Unit,
    onOpen: (Account) -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
    ) {
        SettingsHeader(R.string.settings_accounts, enabled = true, onBack = onBack) {
            IconButton(onClick = onAdd) {
                Icon(AppIcons.Plus, contentDescription = stringResource(R.string.settings_account_add))
            }
        }

        when (state) {
            AccountsUiState.Loading -> Centered { CircularProgressIndicator() }

            is AccountsUiState.Failure -> Centered {
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

            is AccountsUiState.Ready ->
                if (state.isEmpty) {
                    Centered {
                        Text(stringResource(R.string.settings_accounts_empty))
                        Button(
                            onClick = onAdd,
                            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                        ) {
                            Text(stringResource(R.string.settings_account_add))
                        }
                    }
                } else {
                    AccountList(state, onOpen)
                }
        }
    }
}

@Composable
private fun AccountList(
    state: AccountsUiState.Ready,
    onOpen: (Account) -> Unit,
) {
    var archiveOpen by rememberSaveable { mutableStateOf(false) }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_2),
    ) {
        items(state.active, key = { it.id }) { AccountItem(it, MaterialTheme.colorScheme.onSurface, onOpen) }

        if (state.archived.isNotEmpty()) {
            item(key = "archive") {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { archiveOpen = !archiveOpen }
                        .heightIn(min = Dimens.TOUCH_MIN)
                        .padding(top = Dimens.SPACE_2),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = stringResource(R.string.settings_accounts_archived, state.archived.size),
                        style = MaterialTheme.typography.titleMedium,
                        modifier = Modifier.weight(1f),
                    )
                    Icon(
                        if (archiveOpen) AppIcons.ChevronUp else AppIcons.ChevronDown,
                        contentDescription = null,
                    )
                }
            }
            if (archiveOpen) {
                items(state.archived, key = { it.id }) {
                    AccountItem(it, MaterialTheme.colorScheme.onSurfaceVariant, onOpen)
                }
            }
        }
    }
}

@Composable
private fun AccountItem(
    account: Account,
    color: Color,
    onOpen: (Account) -> Unit,
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable { onOpen(account) }
                .heightIn(min = Dimens.TOUCH_MIN)
                .padding(vertical = Dimens.SPACE_3),
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = account.name,
                style = MaterialTheme.typography.bodyLarge,
                color = color,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        HorizontalDivider()
    }
}
