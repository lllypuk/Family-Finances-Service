package tech.shatrov.familyfinances.ui.settings

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.text.input.KeyboardType
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message

/** Действие, которое сервер применит сразу: у обоих спрашивается подтверждение. */
private enum class UserAction {
    Role,
    Active,
}

/**
 * Форма пользователя. Роль и активность правятся не «Сохранить», а отдельными кнопками: сервер
 * принимает их своим `PATCH`, и подтверждение спрашивается на каждую.
 *
 * @param passwordSet вернулись со страницы пароля — сообщение живёт здесь, а не в модели.
 */
@Composable
fun UserEditScreen(
    state: UserEditUiState,
    passwordSet: Boolean,
    onEmailChange: (String) -> Unit,
    onFirstNameChange: (String) -> Unit,
    onLastNameChange: (String) -> Unit,
    onPasswordChange: (String) -> Unit,
    onRoleChange: (Role) -> Unit,
    onSubmit: () -> Unit,
    onToggleRole: () -> Unit,
    onToggleActive: () -> Unit,
    onSetPassword: () -> Unit,
    onRetry: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var confirming by remember { mutableStateOf<UserAction?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
    ) {
        SettingsHeader(
            title = if (state.creating) R.string.settings_user_new else R.string.settings_user_edit,
            enabled = !state.submitting,
            onBack = onBack,
        )

        val loadError = state.loadError
        when {
            state.loading -> Centered { CircularProgressIndicator() }

            loadError != null -> Centered {
                Text(
                    text = loadError.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                    Text(stringResource(R.string.retry))
                }
            }

            else -> UserForm(
                state = state,
                passwordSet = passwordSet,
                onEmailChange = onEmailChange,
                onFirstNameChange = onFirstNameChange,
                onLastNameChange = onLastNameChange,
                onPasswordChange = onPasswordChange,
                onRoleChange = onRoleChange,
                onSubmit = onSubmit,
                onSetPassword = onSetPassword,
                onConfirm = { confirming = it },
            )
        }
    }

    val action = confirming
    if (action != null) {
        ConfirmAction(
            action = action,
            state = state,
            onDismiss = { confirming = null },
            onConfirmed = {
                confirming = null
                when (action) {
                    UserAction.Role -> onToggleRole()
                    UserAction.Active -> onToggleActive()
                }
            },
        )
    }
}

@Composable
private fun UserForm(
    state: UserEditUiState,
    passwordSet: Boolean,
    onEmailChange: (String) -> Unit,
    onFirstNameChange: (String) -> Unit,
    onLastNameChange: (String) -> Unit,
    onPasswordChange: (String) -> Unit,
    onRoleChange: (Role) -> Unit,
    onSubmit: () -> Unit,
    onSetPassword: () -> Unit,
    onConfirm: (UserAction) -> Unit,
) {
    val editable = !state.submitting

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(vertical = Dimens.SPACE_3),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        if (passwordSet) {
            Text(
                text = stringResource(R.string.settings_user_password_saved),
                style = MaterialTheme.typography.bodyMedium,
            )
        }

        NameField(
            value = state.firstName,
            onValueChange = onFirstNameChange,
            label = R.string.settings_first_name,
            enabled = editable,
            error = state.fieldErrors[UserField.FIRST_NAME],
        )

        NameField(
            value = state.lastName,
            onValueChange = onLastNameChange,
            label = R.string.settings_last_name,
            enabled = editable,
            error = state.fieldErrors[UserField.LAST_NAME],
        )

        OutlinedTextField(
            value = state.email,
            onValueChange = onEmailChange,
            label = { Text(stringResource(R.string.settings_email)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(UserField.EMAIL),
            supportingText = { FieldError(state.fieldErrors[UserField.EMAIL]) },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
            modifier = Modifier.fillMaxWidth(),
        )

        if (state.creating) {
            SecretField(
                value = state.password,
                onValueChange = onPasswordChange,
                label = R.string.settings_password,
                enabled = editable,
                error = state.fieldErrors[UserField.PASSWORD]
                    ?: stringResource(R.string.settings_password_length).takeIf { state.passwordInvalid },
            )
            Text(
                text = stringResource(R.string.settings_password_hint),
                style = MaterialTheme.typography.bodySmall,
            )

            Text(stringResource(R.string.settings_user_role), style = MaterialTheme.typography.bodySmall)
            ChipRow {
                item {
                    Chip(stringResource(R.string.settings_role_member), state.role == Role.member, editable) {
                        onRoleChange(Role.member)
                    }
                }
                item {
                    Chip(stringResource(R.string.settings_role_admin), state.role == Role.admin, editable) {
                        onRoleChange(Role.admin)
                    }
                }
            }
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
                Text(stringResource(if (state.creating) R.string.settings_user_create else R.string.settings_save))
            }
        }

        if (!state.creating) {
            HorizontalDivider()
            UserActions(state, onSetPassword, onConfirm)
        }
    }
}

/** Роль, активность и пароль: каждое действие уходит на сервер само по себе. */
@Composable
private fun UserActions(
    state: UserEditUiState,
    onSetPassword: () -> Unit,
    onConfirm: (UserAction) -> Unit,
) {
    val editable = !state.submitting

    Text(
        text = stringResource(
            R.string.settings_user_current_role,
            stringResource(
                if (state.currentRole == Role.admin) R.string.settings_role_admin else R.string.settings_role_member,
            ),
        ),
        style = MaterialTheme.typography.bodyMedium,
    )

    OutlinedButton(
        onClick = { onConfirm(UserAction.Role) },
        enabled = editable,
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = Dimens.TOUCH_MIN),
    ) {
        Text(stringResource(roleActionLabel(state.currentRole)))
    }

    // Себя не деактивируют и себе не задают пароль: своя смена живёт в «Пароле».
    if (state.ownActions) {
        OutlinedButton(
            onClick = onSetPassword,
            enabled = editable,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(R.string.settings_user_set_password))
        }

        OutlinedButton(
            onClick = { onConfirm(UserAction.Active) },
            enabled = editable,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(activeActionLabel(state.active)))
        }
    }
}

@Composable
private fun ConfirmAction(
    action: UserAction,
    state: UserEditUiState,
    onDismiss: () -> Unit,
    onConfirmed: () -> Unit,
) {
    val question = when (action) {
        UserAction.Role -> R.string.settings_user_role_confirm

        UserAction.Active -> if (state.active) {
            R.string.settings_user_deactivate_confirm
        } else {
            R.string.settings_user_activate_confirm
        }
    }
    val confirm = when (action) {
        UserAction.Role -> roleActionLabel(state.currentRole)
        UserAction.Active -> activeActionLabel(state.active)
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(question)) },
        confirmButton = { TextButton(onClick = onConfirmed) { Text(stringResource(confirm)) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) } },
    )
}

@Composable
private fun NameField(
    value: String,
    onValueChange: (String) -> Unit,
    @StringRes label: Int,
    enabled: Boolean,
    error: String?,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        label = { Text(stringResource(label)) },
        singleLine = true,
        enabled = enabled,
        isError = error != null,
        supportingText = { FieldError(error) },
        modifier = Modifier.fillMaxWidth(),
    )
}

@StringRes
private fun roleActionLabel(role: Role): Int = if (role == Role.admin) {
    R.string.settings_user_make_member
} else {
    R.string.settings_user_make_admin
}

@StringRes
private fun activeActionLabel(active: Boolean): Int = if (active) {
    R.string.settings_user_deactivate
} else {
    R.string.settings_user_activate
}
