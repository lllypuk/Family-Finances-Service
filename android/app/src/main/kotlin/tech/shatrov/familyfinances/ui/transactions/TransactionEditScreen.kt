package tech.shatrov.familyfinances.ui.transactions

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.message
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.util.UUID

/**
 * Форма транзакции. Состояние приходит снаружи: модель собирается корнем, а тест экрана
 * подставляет готовое состояние вместо сети.
 */
@Composable
fun TransactionEditScreen(
    state: TransactionEditUiState,
    onAmountChange: (String) -> Unit,
    onTypeChange: (TransactionType) -> Unit,
    onCategoryChange: (UUID) -> Unit,
    onDateChange: (LocalDate) -> Unit,
    onDescriptionChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onDelete: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var datePickerShown by remember { mutableStateOf(false) }
    var deleteConfirmShown by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onBack) { Text(stringResource(R.string.back)) }
            Text(
                text = stringResource(
                    if (state.editing) R.string.transaction_edit_title else R.string.transaction_new_title,
                ),
                style = MaterialTheme.typography.headlineSmall,
            )
        }

        OutlinedTextField(
            value = state.amount,
            onValueChange = onAmountChange,
            label = { Text(stringResource(R.string.transaction_amount)) },
            singleLine = true,
            isError = state.fieldErrors.containsKey(TransactionField.AMOUNT),
            supportingText = { FieldError(state, TransactionField.AMOUNT) },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
        )

        ChipRow {
            item {
                Chip(stringResource(R.string.type_expense), state.type == TransactionType.expense) {
                    onTypeChange(TransactionType.expense)
                }
            }
            item {
                Chip(stringResource(R.string.type_income), state.type == TransactionType.income) {
                    onTypeChange(TransactionType.income)
                }
            }
        }
        FieldError(state, TransactionField.TYPE)

        Text(stringResource(R.string.transaction_category), style = MaterialTheme.typography.bodySmall)
        ChipRow {
            items(state.visibleCategories) { category: Category ->
                Chip(category.name, state.categoryId == category.id) { onCategoryChange(category.id) }
            }
        }
        FieldError(state, TransactionField.CATEGORY)

        OutlinedButton(
            onClick = { datePickerShown = true },
            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(formatDay(state.date))
        }
        FieldError(state, TransactionField.DATE)

        OutlinedTextField(
            value = state.description,
            onValueChange = onDescriptionChange,
            label = { Text(stringResource(R.string.transaction_description)) },
            singleLine = true,
            isError = state.fieldErrors.containsKey(TransactionField.DESCRIPTION),
            supportingText = { FieldError(state, TransactionField.DESCRIPTION) },
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
                Text(stringResource(R.string.transaction_save))
            }
        }

        if (state.editing) {
            TextButton(
                onClick = { deleteConfirmShown = true },
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.transaction_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    if (datePickerShown) {
        DatePickerSheet(
            date = state.date,
            onPick = {
                onDateChange(it)
                datePickerShown = false
            },
            onDismiss = { datePickerShown = false },
        )
    }

    // Удаление необратимо и подтверждения на сервере не имеет — спрашиваем здесь.
    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.transaction_delete_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.transaction_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteConfirmShown = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }
}

/** Календарь считает в UTC-полуночах, поэтому дата переводится через `ZoneOffset.UTC`. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DatePickerSheet(
    date: LocalDate,
    onPick: (LocalDate) -> Unit,
    onDismiss: () -> Unit,
) {
    val picker = rememberDatePickerState(
        initialSelectedDateMillis = date.atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli(),
    )
    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = {
                val millis = picker.selectedDateMillis
                if (millis == null) {
                    onDismiss()
                } else {
                    onPick(Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate())
                }
            }) {
                Text(stringResource(R.string.transaction_date_pick))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) }
        },
    ) {
        DatePicker(state = picker)
    }
}

@Composable
private fun FieldError(
    state: TransactionEditUiState,
    field: String,
) {
    val text = state.fieldErrors[field] ?: return
    Text(text = text, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
}
