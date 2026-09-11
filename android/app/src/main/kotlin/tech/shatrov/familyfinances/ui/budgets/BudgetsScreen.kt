package tech.shatrov.familyfinances.ui.budgets

import androidx.annotation.StringRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatPeriod
import tech.shatrov.familyfinances.ui.message
import java.util.UUID

/** `utilization` у бюджета — проценты 0…100, а индикатору нужна доля. */
private const val FULL_PERCENT = 100f

/** Бюджеты семьи: одна страница с фильтром периода и прогрессом по каждому лимиту. */
@Composable
fun BudgetsScreen(
    state: BudgetsUiState,
    currency: String,
    onRetry: () -> Unit,
    onFilterChange: (BudgetFilter) -> Unit,
    onCreate: () -> Unit,
    onOpen: (UUID) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = Dimens.SPACE_4, end = Dimens.SPACE_2, top = Dimens.SPACE_2, bottom = Dimens.SPACE_2),
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.budgets_title),
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = onCreate) {
                Icon(AppIcons.Plus, contentDescription = stringResource(R.string.budgets_add))
            }
        }

        when (state) {
            BudgetsUiState.Loading -> Centered { CircularProgressIndicator() }

            is BudgetsUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(
                    onClick = onRetry,
                    modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                ) {
                    Text(stringResource(R.string.retry))
                }
            }

            is BudgetsUiState.Ready -> {
                Filters(state.filter, onFilterChange)
                if (state.isEmpty) {
                    Centered { Text(stringResource(emptyText(state.filter))) }
                } else {
                    Rows(state, currency, onOpen)
                }
            }
        }
    }
}

@StringRes
private fun emptyText(filter: BudgetFilter): Int = when (filter) {
    // Будущий бюджет после создания не виден «на сегодня» — иначе он выглядит потерянным.
    BudgetFilter.TODAY -> R.string.budgets_empty_today

    BudgetFilter.ALL -> R.string.budgets_empty
}

@Composable
private fun Filters(
    filter: BudgetFilter,
    onChange: (BudgetFilter) -> Unit,
) {
    ChipRow(Modifier.padding(horizontal = Dimens.SPACE_4)) {
        item {
            Chip(stringResource(R.string.budgets_filter_today), filter == BudgetFilter.TODAY) {
                onChange(BudgetFilter.TODAY)
            }
        }
        item {
            Chip(stringResource(R.string.budgets_filter_all), filter == BudgetFilter.ALL) {
                onChange(BudgetFilter.ALL)
            }
        }
    }
}

@Composable
private fun Rows(
    state: BudgetsUiState.Ready,
    currency: String,
    onOpen: (UUID) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        items(state.rows, key = { it.budget.id }) { row ->
            BudgetItem(row, currency, onOpen)
        }
    }
}

@Composable
private fun BudgetItem(
    row: BudgetRow,
    currency: String,
    onOpen: (UUID) -> Unit,
) {
    val colors = LocalAppColors.current
    val budget = row.budget
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onOpen(budget.id) }
            .heightIn(min = Dimens.TOUCH_MIN),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = budget.name,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = stringResource(
                    R.string.budget_progress,
                    formatMoney(budget.spentMinor, currency),
                    formatMoney(budget.amountMinor, currency),
                ),
                style = MaterialTheme.typography.bodySmall,
            )
        }
        Text(
            text = "${row.categoryName ?: stringResource(R.string.budgets_all_categories)} · " +
                formatPeriod(budget.startDate, budget.endDate),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
        LinearProgressIndicator(
            progress = { (budget.utilization.toFloat() / FULL_PERCENT).coerceIn(0f, 1f) },
            color = when (row.level) {
                BudgetLevel.OVER -> colors.expense
                BudgetLevel.NEAR -> colors.warning
                BudgetLevel.OK -> colors.action
            },
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun Centered(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        content()
    }
}
