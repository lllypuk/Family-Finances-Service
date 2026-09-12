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
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.message

/** Семья: название, валюта, таймзона. Кнопка гаснет, пока форма не отличается от сессии. */
@Composable
fun FamilyScreen(
    state: FamilyUiState,
    onNameChange: (String) -> Unit,
    onCurrencyChange: (String) -> Unit,
    onTimezoneChange: (String) -> Unit,
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
        SettingsHeader(R.string.settings_family, enabled = editable, onBack = onBack)

        OutlinedTextField(
            value = state.name,
            onValueChange = onNameChange,
            label = { Text(stringResource(R.string.settings_family_name)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(FamilyField.NAME),
            supportingText = { FieldError(state.fieldErrors[FamilyField.NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = state.currency,
            onValueChange = onCurrencyChange,
            label = { Text(stringResource(R.string.settings_family_currency)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(FamilyField.CURRENCY),
            supportingText = {
                val fieldError = state.fieldErrors[FamilyField.CURRENCY]
                if (fieldError == null) {
                    Text(stringResource(R.string.settings_family_currency_hint))
                } else {
                    FieldError(fieldError)
                }
            },
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = state.timezone,
            onValueChange = onTimezoneChange,
            label = { Text(stringResource(R.string.settings_family_timezone)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(FamilyField.TIMEZONE),
            supportingText = {
                val fieldError = state.fieldErrors[FamilyField.TIMEZONE]
                if (fieldError == null) {
                    Text(stringResource(R.string.settings_family_timezone_hint))
                } else {
                    FieldError(fieldError)
                }
            },
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
