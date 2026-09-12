package tech.shatrov.familyfinances.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.message

/** Пароль пользователя со стороны админа: текущий не спрашивается, только новый и повтор. */
@Composable
fun UserPasswordScreen(
    state: UserPasswordUiState,
    onNewChange: (String) -> Unit,
    onRepeatChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val editable = !state.submitting

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        SettingsHeader(R.string.settings_user_password, enabled = editable, onBack = onBack)

        SecretField(
            value = state.next,
            onValueChange = onNewChange,
            label = R.string.settings_password_new,
            enabled = editable,
            error = state.fieldErrors[PasswordField.NEW]
                ?: stringResource(R.string.settings_password_length).takeIf { state.lengthInvalid },
        )

        SecretField(
            value = state.repeat,
            onValueChange = onRepeatChange,
            label = R.string.settings_password_repeat,
            enabled = editable,
            error = stringResource(R.string.settings_password_mismatch).takeIf { state.mismatch },
        )

        Text(
            text = stringResource(R.string.settings_password_hint),
            style = MaterialTheme.typography.bodySmall,
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
                Text(stringResource(R.string.settings_save))
            }
        }
    }
}
