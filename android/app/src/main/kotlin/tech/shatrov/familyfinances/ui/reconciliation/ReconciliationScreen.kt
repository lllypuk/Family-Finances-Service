package tech.shatrov.familyfinances.ui.reconciliation

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ReconciliationRow
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.currencySuffix
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import tech.shatrov.familyfinances.ui.settings.SettingsHeader
import java.time.YearMonth
import java.util.UUID

/**
 * Сверка месяца. Тап по строке открывает её расходы (`null` — операции без счёта), тап по «в банке» —
 * лист с цифрой банка.
 */
@Composable
fun ReconciliationScreen(
    month: YearMonth,
    state: ReconciliationUiState,
    editor: BankEditUiState?,
    currency: String,
    onBack: () -> Unit,
    onMonthChange: (YearMonth) -> Unit,
    onRetry: () -> Unit,
    onOpenTransactions: (UUID?) -> Unit,
    onOpenBank: (ReconciliationRow) -> Unit,
    onAddAccount: () -> Unit,
    onAmountChange: (String) -> Unit,
    onNoteChange: (String) -> Unit,
    onSave: () -> Unit,
    onDelete: () -> Unit,
    onDismissBank: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Column(modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2)) {
            SettingsHeader(R.string.reconciliation_title, enabled = true, onBack = onBack)
            MonthSwitcher(month, onMonthChange)
        }

        when (state) {
            ReconciliationUiState.Loading -> Centered { CircularProgressIndicator() }

            is ReconciliationUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                    Text(stringResource(R.string.retry))
                }
            }

            is ReconciliationUiState.Ready ->
                if (state.isEmpty) {
                    Centered {
                        Text(stringResource(R.string.reconciliation_empty))
                        Button(onClick = onAddAccount, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                            Text(stringResource(R.string.reconciliation_add_account))
                        }
                    }
                } else {
                    Rows(state, currency, onOpenTransactions, onOpenBank)
                }
        }
    }

    if (editor != null) {
        BankSheet(editor, currency, onAmountChange, onNoteChange, onSave, onDelete, onDismissBank)
    }
}

@Composable
private fun MonthSwitcher(
    month: YearMonth,
    onMonthChange: (YearMonth) -> Unit,
) {
    Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = { onMonthChange(month.minusMonths(1)) }) {
            Icon(AppIcons.ChevronLeft, contentDescription = stringResource(R.string.reconciliation_prev_month))
        }
        Text(
            text = formatMonth(month.atDay(1)),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.weight(1f),
        )
        IconButton(onClick = { onMonthChange(month.plusMonths(1)) }) {
            Icon(AppIcons.ChevronRight, contentDescription = stringResource(R.string.reconciliation_next_month))
        }
    }
}

@Composable
private fun Rows(
    state: ReconciliationUiState.Ready,
    currency: String,
    onOpenTransactions: (UUID?) -> Unit,
    onOpenBank: (ReconciliationRow) -> Unit,
) {
    val rows = state.stats.accounts
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        itemsIndexed(rows, key = { _, row -> row.account.id }) { index, row ->
            AccountRow(row, currency, rowPlace(index, rows.size), onOpenTransactions, onOpenBank)
        }
        // Строки нет, когда без счёта ничего не потрачено: расшифровывать нечего.
        if (state.stats.unassignedMinor > 0) {
            item {
                UnassignedRow(state.stats.unassignedMinor, currency) { onOpenTransactions(null) }
            }
        }
    }
}

@Composable
private fun AccountRow(
    row: ReconciliationRow,
    currency: String,
    place: RowPlace,
    onOpenTransactions: (UUID?) -> Unit,
    onOpenBank: (ReconciliationRow) -> Unit,
) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, LocalAppColors.current) { onOpenTransactions(row.account.id) },
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = row.account.name,
                style = MaterialTheme.typography.bodyMedium,
                color = if (row.account.isArchived) muted else MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            DiffStatus(row.diffMinor, currency)
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = stringResource(R.string.reconciliation_recorded, formatMoney(row.recordedMinor, currency)),
                style = MaterialTheme.typography.bodySmall,
                color = muted,
                modifier = Modifier.weight(1f),
            )
            val bank = row.bankExpenseMinor
            TextButton(onClick = { onOpenBank(row) }) {
                Text(
                    text = if (bank == null) {
                        stringResource(R.string.reconciliation_bank_none)
                    } else {
                        stringResource(R.string.reconciliation_bank, formatMoney(bank, currency))
                    },
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
        val note = row.note
        if (!note.isNullOrBlank()) {
            Text(text = note, style = MaterialTheme.typography.bodySmall, color = muted)
        }
    }
}

/** Разница словами и без знака: «не записано» и «лишнее» читаются быстрее, чем плюс и минус. */
@Composable
private fun DiffStatus(
    diff: Long?,
    currency: String,
) {
    val (text, color) = when {
        diff == null -> stringResource(R.string.reconciliation_not_checked) to
            MaterialTheme.colorScheme.onSurfaceVariant

        diff == 0L -> stringResource(R.string.reconciliation_matched) to LocalAppColors.current.income

        diff > 0 -> stringResource(R.string.reconciliation_missing, formatMoney(diff, currency)) to
            MaterialTheme.colorScheme.error

        else -> stringResource(R.string.reconciliation_extra, formatMoney(-diff, currency)) to
            MaterialTheme.colorScheme.error
    }
    Text(text = text, style = MaterialTheme.typography.bodySmall, color = color)
}

@Composable
private fun UnassignedRow(
    amount: Long,
    currency: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .padding(top = Dimens.SPACE_4)
            .fillMaxWidth()
            .groupedRow(RowPlace.ONLY, LocalAppColors.current, onClick),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.reconciliation_unassigned),
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        Text(text = formatMoney(amount, currency), style = MaterialTheme.typography.displaySmall)
        Icon(AppIcons.ChevronRight, contentDescription = null)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BankSheet(
    state: BankEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onNoteChange: (String) -> Unit,
    onSave: () -> Unit,
    onDelete: () -> Unit,
    onDismiss: () -> Unit,
) {
    // Во время отправки лист не закрывается: модель отказ закрыть проигнорирует, а скрытый
    // жестом лист остался бы в композиции невидимым.
    val submitting by rememberUpdatedState(state.submitting)
    val sheetState = rememberModalBottomSheetState(
        skipPartiallyExpanded = true,
        confirmValueChange = { it != SheetValue.Hidden || !submitting },
    )
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        BankSheetContent(state, currency, onAmountChange, onNoteChange, onSave, onDelete)
    }
}

@Composable
internal fun BankSheetContent(
    state: BankEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onNoteChange: (String) -> Unit,
    onSave: () -> Unit,
    onDelete: () -> Unit,
) {
    var deleteConfirmShown by remember { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        Column {
            Text(text = state.accountName, style = MaterialTheme.typography.titleMedium)
            Text(
                text = formatMonth(state.month.atDay(1)),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        val invalid = state.amount.isNotBlank() && state.amountMinor == null
        OutlinedTextField(
            value = state.amount,
            onValueChange = onAmountChange,
            label = { Text(stringResource(R.string.reconciliation_bank_amount)) },
            singleLine = true,
            suffix = currencySuffix(currency),
            isError = invalid,
            supportingText = {
                FieldError(if (invalid) stringResource(R.string.reconciliation_error_amount) else null)
            },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = state.note,
            onValueChange = onNoteChange,
            label = { Text(stringResource(R.string.reconciliation_note)) },
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
            onClick = onSave,
            enabled = state.canSubmit,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            if (state.submitting) {
                CircularProgressIndicator(modifier = Modifier.heightIn(max = Dimens.ICON_SIZE))
            } else {
                Text(stringResource(R.string.reconciliation_save))
            }
        }
        if (state.exists) {
            TextButton(
                onClick = { deleteConfirmShown = true },
                enabled = !state.submitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.reconciliation_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.reconciliation_delete_confirm)) },
            text = { Text(state.accountName) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.reconciliation_delete))
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
