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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message

/** Смена своего пароля: текущий, новый и повтор. Длина считается в байтах, как на сервере. */
@Composable
fun PasswordScreen(
    state: PasswordUiState,
    onCurrentChange: (String) -> Unit,
    onNewChange: (String) -> Unit,
    onRepeatChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val editable = !state.submitting
    val currentError = when {
        state.currentInvalid -> stringResource(R.string.settings_password_wrong_current)
        else -> state.fieldErrors[PasswordField.CURRENT]
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        SettingsHeader(R.string.settings_password, enabled = editable, onBack = onBack)

        SecretField(
            value = state.current,
            onValueChange = onCurrentChange,
            label = R.string.settings_password_current,
            enabled = editable,
            error = currentError,
        )

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

        if (state.changed) {
            Text(
                text = stringResource(R.string.settings_password_changed),
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

@Composable
private fun SecretField(
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
        visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
        modifier = Modifier.fillMaxWidth(),
    )
}
