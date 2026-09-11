package tech.shatrov.familyfinances.ui.budgets

import androidx.annotation.StringRes
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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.KeyboardType
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.BudgetPeriod
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.DatePickerSheet
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.message
import java.time.LocalDate
import java.util.UUID

/** Какая из двух дат правится календарём; `null` — календарь закрыт. */
private enum class DateField { START, END }

/**
 * Форма бюджета. Состояние приходит снаружи, как у формы операции. Период и категория при
 * правке — текст: их нет в `UpdateBudgetRequest`.
 */
@Composable
fun BudgetEditScreen(
    state: BudgetEditUiState,
    onNameChange: (String) -> Unit,
    onAmountChange: (String) -> Unit,
    onPeriodChange: (BudgetPeriod) -> Unit,
    onCategoryChange: (UUID?) -> Unit,
    onStartChange: (LocalDate) -> Unit,
    onEndChange: (LocalDate) -> Unit,
    onSubmit: () -> Unit,
    onDelete: () -> Unit,
    onRetry: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    var pickingDate by remember { mutableStateOf<DateField?>(null) }
    var deleteConfirmShown by remember { mutableStateOf(false) }

    // Отправка уносит снимок формы: правка во время неё в запрос не попадёт, а успех закроет
    // экран поверх неё.
    val editable = !state.submitting

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
    ) {
        Header(state, onBack)

        OutlinedTextField(
            value = state.name,
            onValueChange = onNameChange,
            label = { Text(stringResource(R.string.budget_name)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(BudgetField.NAME),
            supportingText = { FieldError(state.fieldErrors[BudgetField.NAME]) },
            modifier = Modifier.fillMaxWidth(),
        )

        OutlinedTextField(
            value = state.amount,
            onValueChange = onAmountChange,
            label = { Text(stringResource(R.string.budget_amount)) },
            singleLine = true,
            enabled = editable,
            isError = state.fieldErrors.containsKey(BudgetField.AMOUNT),
            supportingText = { FieldError(state.fieldErrors[BudgetField.AMOUNT]) },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            modifier = Modifier.fillMaxWidth(),
        )

        Label(R.string.budget_period)
        if (state.editing) {
            Text(stringResource(periodLabel(state.period)), style = MaterialTheme.typography.bodyMedium)
        } else {
            ChipRow {
                items(BudgetPeriod.entries) { period ->
                    Chip(stringResource(periodLabel(period)), state.period == period, editable) {
                        onPeriodChange(period)
                    }
                }
            }
        }
        FieldError(state.fieldErrors[BudgetField.PERIOD])

        Label(R.string.budget_category)
        CategoryPicker(state, editable, onCategoryChange)
        FieldError(state.fieldErrors[BudgetField.CATEGORY])

        DateButton(R.string.budget_start, formatDay(state.start), editable) { pickingDate = DateField.START }
        DateButton(R.string.budget_end, formatDay(state.end), editable) { pickingDate = DateField.END }
        FieldError(state.fieldErrors[BudgetField.START])
        FieldError(state.fieldErrors[BudgetField.END])
        if (state.periodInvalid) {
            FieldError(stringResource(R.string.budget_error_period))
        }

        FormError(state, onRetry)

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
                Text(stringResource(R.string.budget_save))
            }
        }

        if (state.editing) {
            TextButton(
                onClick = { deleteConfirmShown = true },
                enabled = !state.submitting && state.loaded != null,
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(min = Dimens.TOUCH_MIN),
            ) {
                Text(stringResource(R.string.budget_delete), color = MaterialTheme.colorScheme.error)
            }
        }
    }

    val picking = pickingDate
    if (picking != null) {
        DatePickerSheet(
            date = if (picking == DateField.START) state.start else state.end,
            onPick = {
                if (picking == DateField.START) onStartChange(it) else onEndChange(it)
                pickingDate = null
            },
            onDismiss = { pickingDate = null },
        )
    }

    // Удаление необратимо и подтверждения на сервере не имеет — спрашиваем здесь.
    if (deleteConfirmShown) {
        AlertDialog(
            onDismissRequest = { deleteConfirmShown = false },
            title = { Text(stringResource(R.string.budget_delete_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    deleteConfirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.budget_delete))
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

@StringRes
private fun periodLabel(period: BudgetPeriod): Int = when (period) {
    BudgetPeriod.weekly -> R.string.budget_period_weekly
    BudgetPeriod.monthly -> R.string.budget_period_monthly
    BudgetPeriod.yearly -> R.string.budget_period_yearly
    BudgetPeriod.custom -> R.string.budget_period_custom
}

@Composable
private fun Header(
    state: BudgetEditUiState,
    onBack: () -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        // Отправка не отменяется: её корутина умрёт вместе с моделью формы, а запись
        // сервер уже мог принять.
        IconButton(onClick = onBack, enabled = !state.submitting) {
            Icon(AppIcons.ArrowLeft, contentDescription = stringResource(R.string.back))
        }
        Text(
            text = stringResource(if (state.editing) R.string.budget_edit_title else R.string.budget_new_title),
            style = MaterialTheme.typography.headlineSmall,
        )
    }
}

/** У существующего бюджета категория показывается как есть — даже незнакомая справочнику. */
@Composable
private fun CategoryPicker(
    state: BudgetEditUiState,
    enabled: Boolean,
    onCategoryChange: (UUID?) -> Unit,
) {
    val known = state.categories.firstOrNull { it.id == state.categoryId }?.name
    if (state.editing) {
        Text(
            text = when {
                known != null -> known

                state.categoryId == null -> stringResource(R.string.budgets_all_categories)

                // Справочник формы — только расходные категории: доходная или удалённая в нём не найдётся.
                else -> "—"
            },
            style = MaterialTheme.typography.bodyMedium,
        )
        return
    }
    ChipRow {
        item {
            Chip(stringResource(R.string.budgets_all_categories), state.categoryId == null, enabled) {
                onCategoryChange(null)
            }
        }
        items(state.categories) { category: Category ->
            Chip(category.name, state.categoryId == category.id, enabled) { onCategoryChange(category.id) }
        }
    }
}

@Composable
private fun DateButton(
    @StringRes label: Int,
    day: String,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    OutlinedButton(
        onClick = onClick,
        enabled = enabled,
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = Dimens.TOUCH_MIN),
    ) {
        Text("${stringResource(label)}: $day")
    }
}

@Composable
private fun FormError(
    state: BudgetEditUiState,
    onRetry: () -> Unit,
) {
    val error = state.error ?: return
    Text(
        text = error.message(LocalContext.current.resources),
        color = MaterialTheme.colorScheme.error,
        style = MaterialTheme.typography.bodyMedium,
    )
    // Повтор только для незагрузившейся формы: на отказе сохранения он перечитал бы
    // справочник поверх введённого.
    if (!state.ready) {
        Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
            Text(stringResource(R.string.retry))
        }
    }
}

@Composable
private fun Label(@StringRes text: Int) {
    Text(stringResource(text), style = MaterialTheme.typography.bodySmall)
}
