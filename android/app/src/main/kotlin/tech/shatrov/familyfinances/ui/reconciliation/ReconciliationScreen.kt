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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.window.DialogProperties
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.ReconciliationRow
import tech.shatrov.familyfinances.core.api.ReconciliationStats
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.currencySuffix
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatMonthName
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import tech.shatrov.familyfinances.ui.settings.SettingsHeader
import java.time.YearMonth

/** Сверка остатков месяца: итог сверху, под ним счета; тап по счёту — диалог остатка на конец месяца. */
@Composable
fun ReconciliationScreen(
    month: YearMonth,
    today: YearMonth,
    state: ReconciliationUiState,
    editor: BalanceEditUiState?,
    currency: String,
    onBack: () -> Unit,
    onMonthChange: (YearMonth) -> Unit,
    onRetry: () -> Unit,
    onOpenBalance: (ReconciliationRow) -> Unit,
    onAddAccount: () -> Unit,
    onOpenTransactions: () -> Unit,
    onCloseGap: (Long) -> Unit,
    onAmountChange: (String) -> Unit,
    onToggleSign: () -> Unit,
    onSave: () -> Unit,
    onClear: () -> Unit,
    onDismissBalance: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Column(modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2)) {
            SettingsHeader(R.string.reconciliation_title, enabled = true, onBack = onBack)
            MonthSwitcher(month, today, onMonthChange)
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
                    Rows(state, currency, onMonthChange, onOpenBalance, onOpenTransactions, onCloseGap)
                }
        }
    }

    if (editor != null) {
        BalanceDialog(editor, currency, onAmountChange, onToggleSign, onSave, onClear, onDismissBalance)
    }
}

/** Вперёд не дальше [today]: будущий месяц сервер отвергает. */
@Composable
private fun MonthSwitcher(
    month: YearMonth,
    today: YearMonth,
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
        IconButton(onClick = { onMonthChange(month.plusMonths(1)) }, enabled = month < today) {
            Icon(AppIcons.ChevronRight, contentDescription = stringResource(R.string.reconciliation_next_month))
        }
    }
}

@Composable
private fun Rows(
    state: ReconciliationUiState.Ready,
    currency: String,
    onMonthChange: (YearMonth) -> Unit,
    onOpenBalance: (ReconciliationRow) -> Unit,
    onOpenTransactions: () -> Unit,
    onCloseGap: (Long) -> Unit,
) {
    val rows = state.stats.accounts
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        item { TotalCard(state.total, state.stats, currency, onMonthChange, onOpenTransactions, onCloseGap) }
        itemsIndexed(rows, key = { _, row -> row.account.id }) { index, row ->
            AccountRow(row, currency, rowPlace(index, rows.size), onOpenBalance)
        }
        item {
            Text(
                text = stringResource(R.string.reconciliation_transfer_hint),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = Dimens.SPACE_3),
            )
        }
    }
}

@Composable
private fun TotalCard(
    total: ReconciliationTotal,
    stats: ReconciliationStats,
    currency: String,
    onMonthChange: (YearMonth) -> Unit,
    onOpenTransactions: () -> Unit,
    onCloseGap: (Long) -> Unit,
) {
    Column(
        modifier = Modifier
            .padding(bottom = Dimens.SPACE_4)
            .fillMaxWidth()
            .groupedRow(RowPlace.ONLY, LocalAppColors.current),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
    ) {
        when (total) {
            is ReconciliationTotal.Complete -> {
                val muted = MaterialTheme.colorScheme.onSurfaceVariant
                Text(
                    text = stringResource(
                        R.string.reconciliation_balances,
                        formatMoney(total.openingMinor, currency),
                        formatMoney(total.closingMinor, currency),
                        formatMoney(total.closingMinor - total.openingMinor, currency, signed = true),
                    ),
                    style = MaterialTheme.typography.bodySmall,
                    color = muted,
                )
                Text(
                    text = stringResource(
                        R.string.reconciliation_operations,
                        formatMoney(stats.incomeMinor, currency),
                        formatMoney(stats.expenseMinor, currency),
                        formatMoney(stats.incomeMinor - stats.expenseMinor, currency, signed = true),
                    ),
                    style = MaterialTheme.typography.bodySmall,
                    color = muted,
                )
                GapLine(total.gapMinor, currency)
            }

            is ReconciliationTotal.MissingClosing -> Text(
                text = pluralStringResource(
                    R.plurals.reconciliation_missing_closing,
                    total.accounts,
                    total.accounts,
                ),
                style = MaterialTheme.typography.bodyMedium,
            )

            is ReconciliationTotal.MissingOpening -> {
                val prev = formatMonthName(total.prev)
                Text(
                    text = stringResource(R.string.reconciliation_missing_opening, prev),
                    style = MaterialTheme.typography.bodyMedium,
                )
                TextButton(onClick = { onMonthChange(total.prev) }) {
                    Text(stringResource(R.string.reconciliation_enter_opening, prev))
                }
            }
        }
        Row {
            TextButton(onClick = onOpenTransactions) {
                Text(stringResource(R.string.reconciliation_transactions))
            }
            if (total is ReconciliationTotal.Complete && total.gapMinor != 0L) {
                TextButton(onClick = { onCloseGap(total.gapMinor) }) {
                    Text(stringResource(R.string.reconciliation_close_gap))
                }
            }
        }
    }
}

@Composable
private fun GapLine(
    gap: Long,
    currency: String,
) {
    val verdict = gapVerdict(gap)
    if (verdict == GapVerdict.MATCHED) {
        Text(
            text = stringResource(R.string.reconciliation_matched),
            style = MaterialTheme.typography.titleMedium,
            color = LocalAppColors.current.income,
        )
        return
    }
    Text(
        text = stringResource(R.string.reconciliation_gap, formatMoney(gap, currency, signed = true)),
        style = MaterialTheme.typography.titleMedium,
        color = MaterialTheme.colorScheme.error,
    )
    Text(
        text = stringResource(
            if (verdict == GapVerdict.MISSING_INCOME) {
                R.string.reconciliation_missing_income
            } else {
                R.string.reconciliation_missing_expense
            },
        ),
        style = MaterialTheme.typography.bodySmall,
    )
}

@Composable
private fun AccountRow(
    row: ReconciliationRow,
    currency: String,
    place: RowPlace,
    onOpenBalance: (ReconciliationRow) -> Unit,
) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val none = stringResource(R.string.reconciliation_no_balance)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, LocalAppColors.current) { onOpenBalance(row) },
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = row.account.name,
                style = MaterialTheme.typography.bodyMedium,
                color = if (row.account.isArchived) muted else MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = stringResource(
                    R.string.reconciliation_opening,
                    row.openingMinor?.let { formatMoney(it, currency) } ?: none,
                ),
                style = MaterialTheme.typography.bodySmall,
                color = muted,
            )
        }
        Text(
            text = row.closingMinor?.let { formatMoney(it, currency) } ?: none,
            style = MaterialTheme.typography.bodyMedium,
        )
    }
}

@Composable
private fun BalanceDialog(
    state: BalanceEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onToggleSign: () -> Unit,
    onSave: () -> Unit,
    onClear: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(dismissOnClickOutside = !state.submitting),
        title = { Text(state.accountName) },
        text = { BalanceDialogContent(state, currency, onAmountChange, onToggleSign) },
        confirmButton = {
            TextButton(onClick = onSave, enabled = state.canSubmit) {
                Text(stringResource(R.string.reconciliation_save))
            }
        },
        dismissButton = {
            if (state.exists) {
                TextButton(onClick = onClear, enabled = !state.submitting) {
                    Text(stringResource(R.string.reconciliation_clear), color = MaterialTheme.colorScheme.error)
                }
            }
        },
    )
}

@Composable
internal fun BalanceDialogContent(
    state: BalanceEditUiState,
    currency: String,
    onAmountChange: (String) -> Unit,
    onToggleSign: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2)) {
        Text(
            text = formatMonth(state.month.atDay(1)),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        val invalid = state.amount.isNotBlank() && state.amountMinor == null
        OutlinedTextField(
            value = state.amount,
            onValueChange = onAmountChange,
            label = { Text(stringResource(R.string.reconciliation_balance_amount)) },
            singleLine = true,
            leadingIcon = {
                val label = stringResource(R.string.reconciliation_toggle_sign)
                TextButton(
                    onClick = onToggleSign,
                    enabled = !state.submitting,
                    modifier = Modifier.semantics { contentDescription = label },
                ) {
                    Text("±")
                }
            },
            suffix = currencySuffix(currency),
            isError = invalid,
            supportingText = {
                FieldError(if (invalid) stringResource(R.string.reconciliation_error_amount) else null)
            },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
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
    }
}
