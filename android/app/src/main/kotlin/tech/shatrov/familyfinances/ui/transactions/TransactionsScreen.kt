package tech.shatrov.familyfinances.ui.transactions

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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.message
import java.util.UUID

@Composable
fun TransactionsScreen(
    state: TransactionsUiState,
    onRetry: () -> Unit,
    onFiltersChange: (TransactionFilters) -> Unit,
    onLoadMore: () -> Unit,
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
                text = stringResource(R.string.transactions_title),
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = onCreate) {
                Icon(AppIcons.Plus, contentDescription = stringResource(R.string.transactions_add))
            }
        }

        when (state) {
            TransactionsUiState.Loading -> Centered { CircularProgressIndicator() }

            is TransactionsUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                RetryButton(onRetry)
            }

            is TransactionsUiState.Ready -> {
                Filters(state, onFiltersChange)
                if (state.isEmpty) {
                    Centered { Text(stringResource(R.string.transactions_empty)) }
                } else {
                    Days(state, onLoadMore, onOpen)
                }
            }
        }
    }
}

@Composable
private fun Filters(
    state: TransactionsUiState.Ready,
    onChange: (TransactionFilters) -> Unit,
) {
    val filters = state.filters
    Column(verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1)) {
        ChipRow(Modifier.padding(horizontal = Dimens.SPACE_4)) {
            item {
                Chip(stringResource(R.string.filter_all), filters.period == TransactionPeriod.ALL) {
                    onChange(filters.copy(period = TransactionPeriod.ALL))
                }
            }
            item {
                Chip(
                    stringResource(R.string.filter_this_month),
                    filters.period == TransactionPeriod.THIS_MONTH,
                ) {
                    onChange(filters.copy(period = TransactionPeriod.THIS_MONTH))
                }
            }
            item {
                Chip(
                    stringResource(R.string.filter_prev_month),
                    filters.period == TransactionPeriod.PREV_MONTH,
                ) {
                    onChange(filters.copy(period = TransactionPeriod.PREV_MONTH))
                }
            }
        }
        ChipRow(Modifier.padding(horizontal = Dimens.SPACE_4)) {
            item {
                Chip(stringResource(R.string.filter_any_type), filters.type == null) {
                    onChange(filters.copy(type = null))
                }
            }
            item {
                Chip(stringResource(R.string.filter_income), filters.type == TransactionType.income) {
                    onChange(filters.copy(type = TransactionType.income))
                }
            }
            item {
                Chip(stringResource(R.string.filter_expense), filters.type == TransactionType.expense) {
                    onChange(filters.copy(type = TransactionType.expense))
                }
            }
        }
        ChipRow(Modifier.padding(horizontal = Dimens.SPACE_4)) {
            item {
                Chip(stringResource(R.string.filter_all_categories), filters.categoryId == null) {
                    onChange(filters.copy(categoryId = null))
                }
            }
            items(state.categories) { category: Category ->
                Chip(category.name, filters.categoryId == category.id) {
                    onChange(filters.copy(categoryId = category.id))
                }
            }
        }
    }
}

@Composable
private fun Days(
    state: TransactionsUiState.Ready,
    onLoadMore: () -> Unit,
    onOpen: (UUID) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        for (group in state.groups) {
            item(key = group.date) {
                Text(
                    text = formatDay(group.date),
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(top = Dimens.SPACE_2),
                )
            }
            items(group.rows, key = { it.transaction.id }) { row ->
                TransactionItem(row, state.currency, onOpen)
            }
        }

        if (state.moreError != null) {
            item {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(
                        text = state.moreError.message(LocalContext.current.resources),
                        color = MaterialTheme.colorScheme.error,
                    )
                    RetryButton(onLoadMore)
                }
            }
        } else if (state.hasMore) {
            // Догрузка начинается, когда подвал доехал до экрана: отдельного «показать ещё»
            // не нужно, а повторный вызов гасит сама модель.
            item {
                LaunchedEffect(state.groups.sumOf { it.rows.size }) { onLoadMore() }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                ) {
                    CircularProgressIndicator()
                }
            }
        }
    }
}

@Composable
private fun TransactionItem(
    row: TransactionRow,
    currency: String,
    onOpen: (UUID) -> Unit,
) {
    val colors = LocalAppColors.current
    val income = row.transaction.type == TransactionType.income
    val author = if (row.isMine) stringResource(R.string.transactions_author_me) else row.authorName
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onOpen(row.transaction.id) }
            .heightIn(min = Dimens.TOUCH_MIN),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = row.transaction.description,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            val subtitle = listOfNotNull(row.categoryName, author).joinToString(" · ")
            if (subtitle.isNotEmpty()) {
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        // Знак ставится текстом, а не только цветом: доход и акцент — один цвет темы.
        Text(
            text = formatMoney(
                if (income) row.transaction.amountMinor else -row.transaction.amountMinor,
                currency,
                signed = true,
            ),
            style = MaterialTheme.typography.displaySmall,
            color = if (income) colors.income else colors.expense,
        )
    }
}

@Composable
private fun RetryButton(onClick: () -> Unit) {
    Button(
        onClick = onClick,
        modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
    ) {
        Text(stringResource(R.string.retry))
    }
}
