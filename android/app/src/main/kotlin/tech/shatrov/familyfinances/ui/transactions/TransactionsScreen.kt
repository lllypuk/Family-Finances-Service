package tech.shatrov.familyfinances.ui.transactions

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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
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
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.SegmentedChoice
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import java.util.UUID

@Composable
fun TransactionsScreen(
    state: TransactionsUiState,
    filters: TransactionFilters,
    categories: List<Category>,
    onRetry: () -> Unit,
    onFiltersChange: (TransactionFilters) -> Unit,
    onLoadMore: () -> Unit,
    onCreate: () -> Unit,
    onOpen: (UUID) -> Unit,
    modifier: Modifier = Modifier,
) {
    var sheet by rememberSaveable { mutableStateOf(false) }
    Column(modifier = modifier.fillMaxSize()) {
        Text(
            text = stringResource(R.string.transactions_title),
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
        )

        // Фильтры вне `when`: на отказе запроса переключиться иначе некуда, а «Повторить»
        // повторяет ровно его.
        Filters(filters, categories, onFiltersChange) { sheet = true }

        when (state) {
            TransactionsUiState.Loading -> Centered { CircularProgressIndicator() }

            is TransactionsUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                RetryButton(onRetry)
            }

            is TransactionsUiState.Ready ->
                if (state.isEmpty) {
                    Empty(filters, onCreate, onFiltersChange)
                } else {
                    Days(state, onLoadMore, onOpen)
                }
        }
    }

    if (sheet) {
        CategorySheet(
            categories = categories,
            selected = filters.categoryId,
            onSelect = {
                onFiltersChange(filters.copy(categoryId = it))
                sheet = false
            },
            onDismiss = { sheet = false },
        )
    }
}

@Composable
private fun Filters(
    filters: TransactionFilters,
    categories: List<Category>,
    onChange: (TransactionFilters) -> Unit,
    onPickCategory: () -> Unit,
) {
    // Категория, которой нет в справочнике (удалена), — прочерк: «Все категории» на включённом
    // фильтре сказали бы, что фильтра нет, а список при этом остаётся пустым.
    val categoryLabel = when {
        filters.categoryId == null -> stringResource(R.string.filter_all_categories)
        else -> categories.firstOrNull { it.id == filters.categoryId }?.name ?: "—"
    }
    Column(verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1)) {
        SegmentedChoice(
            options = listOf(
                null to stringResource(R.string.filter_any_type),
                TransactionType.income to stringResource(R.string.filter_income),
                TransactionType.expense to stringResource(R.string.filter_expense),
            ),
            selected = filters.type,
            onSelect = { onChange(filters.copy(type = it)) },
            modifier = Modifier.padding(horizontal = Dimens.SPACE_4),
        )
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
            item {
                Chip(categoryLabel, filters.categoryId != null, onClick = onPickCategory)
            }
        }
    }
}

/** Пустой список: без фильтров звать создавать, с фильтрами — сбрасывать их. */
@Composable
private fun Empty(
    filters: TransactionFilters,
    onCreate: () -> Unit,
    onFiltersChange: (TransactionFilters) -> Unit,
) {
    val unfiltered = filters == TransactionFilters()
    Centered {
        Text(stringResource(R.string.transactions_empty))
        Button(
            onClick = { if (unfiltered) onCreate() else onFiltersChange(TransactionFilters()) },
            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
        ) {
            val label = if (unfiltered) R.string.transactions_add_first else R.string.transactions_reset_filters
            Text(stringResource(label))
        }
    }
}

@Composable
private fun Days(
    state: TransactionsUiState.Ready,
    onLoadMore: () -> Unit,
    onOpen: (UUID) -> Unit,
) {
    // Строки одного дня сливаются в панель, поэтому зазор между элементами нулевой: воздух
    // между днями даёт заголовок.
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        contentPadding = PaddingValues(bottom = Dimens.FAB_CLEARANCE),
    ) {
        for (group in state.groups) {
            item(key = group.date) {
                Text(
                    text = formatDay(group.date),
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(top = Dimens.SPACE_4, bottom = Dimens.SPACE_2),
                )
            }
            itemsIndexed(group.rows, key = { _, row -> row.transaction.id }) { index, row ->
                TransactionItem(row, state.currency, rowPlace(index, group.rows.size), onOpen)
            }
        }

        if (state.moreError != null) {
            item {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = Dimens.SPACE_4),
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
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = Dimens.SPACE_4),
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
    place: RowPlace,
    onOpen: (UUID) -> Unit,
) {
    val colors = LocalAppColors.current
    val income = row.transaction.type == TransactionType.income
    val author = if (row.isMine) stringResource(R.string.transactions_author_me) else row.authorName
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .heightIn(min = Dimens.TOUCH_MIN)
            .groupedRow(place, colors) { onOpen(row.transaction.id) },
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
        // Знак ставится текстом, а не только цветом: ради цветовосприятия.
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
