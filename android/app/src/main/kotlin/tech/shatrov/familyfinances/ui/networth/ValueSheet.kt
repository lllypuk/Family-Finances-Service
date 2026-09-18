package tech.shatrov.familyfinances.ui.networth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SheetValue
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.DatePickerSheet
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.currencySuffix
import tech.shatrov.familyfinances.ui.format.formatFullDay
import tech.shatrov.familyfinances.ui.message
import java.time.LocalDate

/**
 * Лист снимка стоимости; пока снимок уходит, смахнуть его нельзя. [onHistory] — ссылка на историю
 * позиции, [onDelete] — удаление записанного снимка; `null` прячет кнопку.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ValueSheet(
    state: ValueEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onDateChange: (LocalDate) -> Unit,
    onSave: () -> Unit,
    onDismiss: () -> Unit,
    onHistory: (() -> Unit)? = null,
    onDelete: (() -> Unit)? = null,
) {
    val submitting by rememberUpdatedState(state.submitting)
    val sheetState = rememberModalBottomSheetState(
        skipPartiallyExpanded = true,
        confirmValueChange = { it != SheetValue.Hidden || !submitting },
    )
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        ValueSheetContent(state, currency, onAmountChange, onDateChange, onSave, onHistory, onDelete)
    }
}

// Тело листа отдельно от `ModalBottomSheet`: тот живёт в своём окне и в Robolectric ненадёжен.
@Composable
internal fun ValueSheetContent(
    state: ValueEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onDateChange: (LocalDate) -> Unit,
    onSave: () -> Unit,
    onHistory: (() -> Unit)? = null,
    onDelete: (() -> Unit)? = null,
) {
    var pickerShown by remember { mutableStateOf(false) }
    var deleteConfirmShown by remember { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        Text(text = state.name, style = MaterialTheme.typography.titleMedium)
        val invalid = state.amount.isNotBlank() && state.amountMinor == null
        OutlinedTextField(
            value = state.amount,
            onValueChange = onAmountChange,
            label = { Text(stringResource(R.string.holding_value_amount)) },
            singleLine = true,
            suffix = currencySuffix(currency),
            isError = invalid,
            supportingText = { FieldError(if (invalid) stringResource(R.string.holding_value_error_amount) else null) },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            enabled = !state.submitting,
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedButton(
            onClick = { pickerShown = true },
            enabled = !state.submitting && !state.existing,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(R.string.holding_value_date, formatFullDay(state.date)))
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
            onClick = onSave,
            enabled = state.canSubmit,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            if (state.submitting) {
                CircularProgressIndicator(modifier = Modifier.heightIn(max = Dimens.ICON_SIZE))
            } else {
                Text(stringResource(R.string.holding_value_save))
            }
        }
        if (onHistory != null) {
            TextButton(
                onClick = onHistory,
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.holding_value_history))
            }
        }
        if (onDelete != null && state.existing) {
            TextButton(
                onClick = { deleteConfirmShown = true },
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.holding_value_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    if (deleteConfirmShown && onDelete != null) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.holding_value_delete_confirm)) },
            text = { Text(formatFullDay(state.date)) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.holding_value_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteConfirmShown = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }

    // Потолок — «сегодня» семьи: будущий снимок сервер отвергнет `422`.
    if (pickerShown) {
        DatePickerSheet(
            date = state.date,
            latest = state.today,
            onPick = {
                pickerShown = false
                onDateChange(it)
            },
            onDismiss = { pickerShown = false },
        )
    }
}
