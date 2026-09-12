package tech.shatrov.familyfinances.ui.settings

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
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message

/** Профиль: имя, фамилия и почта. Кнопка гаснет, пока форма не отличается от сессии. */
@Composable
fun ProfileScreen(
    state: ProfileUiState,
    onEmailChange: (String) -> Unit,
    onFirstNameChange: (String) -> Unit,
    onLastNameChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    // Отправка уносит снимок формы: правка во время неё в запрос не попадёт.
    val editable = !state.submitting

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        SettingsHeader(R.string.settings_profile, enabled = editable, onBack = onBack)

        OutlinedTextField(
            value = state.firstName,
            onValueChange = onFirstNameChange,
            label = { Text(stringResource(R.string.settings_first_name)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(ProfileField.FIRST_NAME),
            supportingText = { FieldError(state.fieldErrors[ProfileField.FIRST_NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = state.lastName,
            onValueChange = onLastNameChange,
            label = { Text(stringResource(R.string.settings_last_name)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(ProfileField.LAST_NAME),
            supportingText = { FieldError(state.fieldErrors[ProfileField.LAST_NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = state.email,
            onValueChange = onEmailChange,
            label = { Text(stringResource(R.string.settings_email)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(ProfileField.EMAIL),
            supportingText = { FieldError(state.fieldErrors[ProfileField.EMAIL]) },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
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
                Text(stringResource(R.string.settings_save))
            }
        }
    }
}
