package tech.shatrov.familyfinances.ui.networth

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.SegmentedChoice
import tech.shatrov.familyfinances.ui.currencySuffix
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.settings.SettingsHeader

/** Форма позиции: сторона — только у новой, вид — из списка своей стороны; у существующей — архив и удаление. */
@Composable
fun HoldingEditScreen(
    state: HoldingEditUiState,
    currency: String,
    onSideChange: (HoldingSide) -> Unit,
    onNameChange: (String) -> Unit,
    onKindChange: (HoldingKind) -> Unit,
    onIncomeChange: (String) -> Unit,
    onExpenseChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onToggleArchive: (zeroFirst: Boolean) -> Unit,
    onDelete: () -> Unit,
    onRetry: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var deleteConfirmShown by remember { mutableStateOf(false) }
    var zeroOfferShown by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        SettingsHeader(
            if (state.editing) R.string.holding_edit_title else R.string.holding_new_title,
            enabled = !state.submitting,
            onBack = onBack,
        )

        val loadError = state.loadError
        when {
            state.loading -> Centered { CircularProgressIndicator() }

            loadError != null -> Centered {
                Text(text = loadError.message(LocalContext.current.resources), color = MaterialTheme.colorScheme.error)
                Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                    Text(stringResource(R.string.retry))
                }
            }

            else -> Form(
                state = state,
                currency = currency,
                onSideChange = onSideChange,
                onNameChange = onNameChange,
                onKindChange = onKindChange,
                onIncomeChange = onIncomeChange,
                onExpenseChange = onExpenseChange,
                onSubmit = onSubmit,
                onArchive = {
                    if (state.archiveNeedsZero) zeroOfferShown = true else onToggleArchive(false)
                },
                onDelete = { deleteConfirmShown = true },
            )
        }
    }

    if (zeroOfferShown) {
        AlertDialog(
            onDismissRequest = { zeroOfferShown = false },
            title = { Text(stringResource(R.string.holding_archive_zero_title)) },
            text = { Text(stringResource(R.string.holding_archive_zero_text)) },
            confirmButton = {
                TextButton(onClick = {
                    zeroOfferShown = false
                    onToggleArchive(true)
                }) {
                    Text(stringResource(R.string.holding_archive_zero_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = {
                    zeroOfferShown = false
                    onToggleArchive(false)
                }) {
                    Text(stringResource(R.string.holding_archive_only))
                }
            },
        )
    }

    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.holding_delete_confirm)) },
            text = { Text(state.saved?.name.orEmpty()) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.holding_delete))
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

@Composable
private fun Form(
    state: HoldingEditUiState,
    currency: String,
    onSideChange: (HoldingSide) -> Unit,
    onNameChange: (String) -> Unit,
    onKindChange: (HoldingKind) -> Unit,
    onIncomeChange: (String) -> Unit,
    onExpenseChange: (String) -> Unit,
    onSubmit: () -> Unit,
    onArchive: () -> Unit,
    onDelete: () -> Unit,
) {
    // Сторона после создания не меняется: сменив её, позиция перевернула бы знак всей истории.
    if (!state.editing) {
        SegmentedChoice(
            options = listOf(
                HoldingSide.asset to stringResource(R.string.holding_side_asset),
                HoldingSide.liability to stringResource(R.string.holding_side_liability),
            ),
            selected = state.side,
            onSelect = onSideChange,
        )
    }

    OutlinedTextField(
        value = state.name,
        onValueChange = onNameChange,
        label = { Text(stringResource(R.string.holding_name)) },
        singleLine = true,
        enabled = !state.submitting,
        isError = state.fieldErrors.containsKey(HOLDING_FIELD_NAME),
        supportingText = { FieldError(state.fieldErrors[HOLDING_FIELD_NAME]) },
        modifier = Modifier.fillMaxWidth(),
    )

    Text(stringResource(R.string.holding_kind), style = MaterialTheme.typography.labelLarge)
    val selected = HoldingKind.of(state.kind)
    ChipRow {
        items(HoldingKind.of(state.side), key = { it.wire }) { kind ->
            Chip(
                label = stringResource(kind.label),
                selected = kind == selected,
                enabled = !state.submitting,
                onClick = { onKindChange(kind) },
            )
        }
    }

    Text(stringResource(R.string.holding_plan), style = MaterialTheme.typography.labelLarge)
    PlanField(
        value = state.income,
        parsed = state.incomeMinor,
        serverError = state.fieldErrors[HOLDING_FIELD_INCOME],
        label = R.string.holding_plan_income,
        currency = currency,
        enabled = !state.submitting,
        onChange = onIncomeChange,
    )
    PlanField(
        value = state.expense,
        parsed = state.expenseMinor,
        serverError = state.fieldErrors[HOLDING_FIELD_EXPENSE],
        label = R.string.holding_plan_expense,
        currency = currency,
        enabled = !state.submitting,
        onChange = onExpenseChange,
    )
    Text(
        text = stringResource(R.string.holding_plan_hint),
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )

    if (state.archived) {
        Text(
            text = stringResource(R.string.holding_archived_hint),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
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
            Text(stringResource(R.string.holding_save))
        }
    }

    if (state.editing) {
        OutlinedButton(
            onClick = onArchive,
            enabled = !state.submitting,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(if (state.archived) R.string.holding_unarchive else R.string.holding_archive))
        }
    }

    // Удаление — админское: у member кнопки нет вовсе, а не отказ по нажатию.
    if (state.editing && state.canDelete) {
        TextButton(
            onClick = onDelete,
            enabled = !state.submitting,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
        ) {
            Text(stringResource(R.string.holding_delete), color = MaterialTheme.colorScheme.error)
        }
    }
}

/** Число плана; [parsed] `null` — ввод не читается, и ошибка видна до отправки. */
@Composable
private fun PlanField(
    value: String,
    parsed: Long?,
    serverError: String?,
    @StringRes label: Int,
    currency: String,
    enabled: Boolean,
    onChange: (String) -> Unit,
) {
    val error = serverError ?: stringResource(R.string.holding_plan_error_amount).takeIf { parsed == null }
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(stringResource(label)) },
        suffix = currencySuffix(currency),
        singleLine = true,
        enabled = enabled,
        isError = error != null,
        supportingText = { FieldError(error) },
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
        modifier = Modifier.fillMaxWidth(),
    )
}
