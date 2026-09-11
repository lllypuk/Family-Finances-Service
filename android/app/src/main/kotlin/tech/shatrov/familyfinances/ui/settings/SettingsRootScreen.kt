package tech.shatrov.familyfinances.ui.settings

import androidx.annotation.StringRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
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
import tech.shatrov.familyfinances.Session
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.message

/**
 * Корень настроек: кто вошёл, какая семья, список подразделов и выход.
 * Данные берутся из сессии — перечитку заказывает хост, экран лишь показывает её ход.
 */
@Composable
fun SettingsRootScreen(
    session: Session,
    refreshing: Boolean,
    error: UiError?,
    onOpen: (SettingsPage) -> Unit,
    onRetry: () -> Unit,
    onBack: () -> Unit,
    onSignOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var signOutConfirmShown by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        SettingsHeader(R.string.settings_title, enabled = true, onBack = onBack)

        if (refreshing) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        }

        // Отказ перечитки не прячет карточки: сессия на руках старая, но верная.
        if (error != null) {
            Text(
                text = error.message(LocalContext.current.resources),
                color = MaterialTheme.colorScheme.error,
            )
            TextButton(onClick = onRetry) { Text(stringResource(R.string.retry)) }
        }

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(
                modifier = Modifier.padding(Dimens.SPACE_4),
                verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
            ) {
                Text(
                    text = "${session.user.firstName} ${session.user.lastName}",
                    style = MaterialTheme.typography.titleMedium,
                )
                Text(text = session.user.email, style = MaterialTheme.typography.bodyMedium)
                Text(
                    text = stringResource(
                        if (session.isAdmin) R.string.settings_role_admin else R.string.settings_role_member,
                    ),
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(
                modifier = Modifier.padding(Dimens.SPACE_4),
                verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
            ) {
                Text(text = session.family.name, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = stringResource(R.string.settings_currency, session.family.currency),
                    style = MaterialTheme.typography.bodyMedium,
                )
                Text(
                    text = stringResource(R.string.settings_timezone, session.family.timezone),
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
        }

        SettingsItem(R.string.settings_profile) { onOpen(SettingsPage.Profile()) }
        SettingsItem(R.string.settings_password) { onOpen(SettingsPage.Password()) }
        SettingsItem(R.string.settings_sessions) { onOpen(SettingsPage.Sessions()) }
        if (session.isAdmin) {
            SettingsItem(R.string.settings_users) { onOpen(SettingsPage.Users()) }
            SettingsItem(R.string.settings_family) { onOpen(SettingsPage.Family()) }
            SettingsItem(R.string.settings_backups) { onOpen(SettingsPage.Backups()) }
        }

        TextButton(
            onClick = { signOutConfirmShown = true },
            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(
                text = stringResource(R.string.sign_out),
                color = MaterialTheme.colorScheme.error,
            )
        }
    }

    if (signOutConfirmShown) {
        AlertDialog(
            onDismissRequest = { signOutConfirmShown = false },
            title = { Text(stringResource(R.string.settings_sign_out_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    signOutConfirmShown = false
                    onSignOut()
                }) {
                    Text(stringResource(R.string.sign_out))
                }
            },
            dismissButton = {
                TextButton(onClick = { signOutConfirmShown = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }
}

@Composable
private fun SettingsItem(
    @StringRes label: Int,
    onClick: () -> Unit,
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .heightIn(min = Dimens.TOUCH_MIN)
                .padding(vertical = Dimens.SPACE_3),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(text = stringResource(label), style = MaterialTheme.typography.bodyLarge)
        }
        HorizontalDivider()
    }
}
