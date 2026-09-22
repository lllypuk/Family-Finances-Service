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
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.SegmentedChoice
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatPeriod
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.recognize.ImportLaunchers
import tech.shatrov.familyfinances.ui.recognize.ImportSourceSheet
import tech.shatrov.familyfinances.ui.rowPlace
import java.util.UUID

@Composable
fun TransactionsScreen(
    state: TransactionsUiState,
    filters: TransactionFilters,
    categories: List<Category>,
    accounts: List<Account>,
    onRetry: () -> Unit,
    onFiltersChange: (TransactionFilters) -> Unit,
    onLoadMore: () -> Unit,
    onCreate: () -> Unit,
    onOpen: (UUID) -> Unit,
    importLaunchers: ImportLaunchers,
    modifier: Modifier = Modifier,
) {
    var sheet by rememberSaveable { mutableStateOf(false) }
    var accountSheet by rememberSaveable { mutableStateOf(false) }
    var importSheet by rememberSaveable { mutableStateOf(false) }
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = Dimens.SPACE_4, end = Dimens.SPACE_2, top = Dimens.SPACE_2, bottom = Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.transactions_title),
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = { importSheet = true }) {
                Icon(AppIcons.ScanLine, contentDescription = stringResource(R.string.recognize_scan))
            }
        }

        // Фильтры вне `when`: на отказе запроса переключиться иначе некуда, а «Повторить»
        // повторяет ровно его.
        Filters(filters, categories, accounts, onFiltersChange, { sheet = true }, { accountSheet = true })

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
        CategoryFilterSheet(
            categories = categories,
            selected = filters.categoryIds,
            onDone = {
                onFiltersChange(filters.copy(categoryIds = it))
                sheet = false
            },
            onDismiss = { sheet = false },
        )
    }
    if (accountSheet) {
        AccountSheet(
            accounts = accounts,
            selected = filters.accountId,
            noneLabel = stringResource(R.string.filter_all_accounts),
            onSelect = {
                onFiltersChange(filters.withAccount(it))
                accountSheet = false
            },
            onDismiss = { accountSheet = false },
        )
    }
    if (importSheet) {
        ImportSourceSheet(importLaunchers, onDismiss = { importSheet = false })
    }
}

// Ни одной из выбранных в справочнике (удалены) — прочерк: «Все категории» на включённом фильтре
// сказали бы, что фильтра нет. Первая — по справочнику, как в запросе; удалённые в «+N» считаются.
@Composable
private fun categoryChipLabel(
    ids: Set<UUID>,
    categories: List<Category>,
): String {
    if (ids.isEmpty()) return stringResource(R.string.filter_all_categories)
    val first = categories.firstOrNull { it.id in ids } ?: return "—"
    if (ids.size == 1) return first.path(categories)
    return stringResource(R.string.filter_categories_more, first.name, ids.size - 1)
}

@Composable
private fun Filters(
    filters: TransactionFilters,
    categories: List<Category>,
    accounts: List<Account>,
    onChange: (TransactionFilters) -> Unit,
    onPickCategory: () -> Unit,
    onPickAccount: () -> Unit,
) {
    val categoryLabel = categoryChipLabel(filters.categoryIds, categories)
    val accountLabel = when {
        filters.unassigned -> stringResource(R.string.transaction_no_account)
        filters.accountId == null -> stringResource(R.string.filter_all_accounts)
        else -> accounts.firstOrNull { it.id == filters.accountId }?.label() ?: "—"
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
            // Месяц и диапазон приходят только снаружи: своего чипа у них нет, а без этого выбранный
            // период не был бы виден ни в одном из трёх.
            val month = filters.month
            if (filters.period == TransactionPeriod.MONTH && month != null) {
                item { Chip(formatMonth(month.atDay(1)), selected = true) {} }
            }
            val from = filters.from
            val to = filters.to
            if (filters.period == TransactionPeriod.RANGE && from != null && to != null) {
                item { Chip(formatPeriod(from, to), selected = true) {} }
            }
            item {
                Chip(stringResource(R.string.filter_all), filters.period == TransactionPeriod.ALL) {
                    onChange(filters.withPeriod(TransactionPeriod.ALL))
                }
            }
            item {
                Chip(
                    stringResource(R.string.filter_this_month),
                    filters.period == TransactionPeriod.THIS_MONTH,
                ) {
                    onChange(filters.withPeriod(TransactionPeriod.THIS_MONTH))
                }
            }
            item {
                Chip(
                    stringResource(R.string.filter_prev_month),
                    filters.period == TransactionPeriod.PREV_MONTH,
                ) {
                    onChange(filters.withPeriod(TransactionPeriod.PREV_MONTH))
                }
            }
            item {
                Chip(categoryLabel, filters.categoryIds.isNotEmpty(), onClick = onPickCategory)
            }
            // Без счетов у семьи чип был бы выбором из одного «Все счета».
            if (accounts.isNotEmpty() || filters.accountId != null || filters.unassigned) {
                item {
                    Chip(accountLabel, filters.accountId != null || filters.unassigned, onClick = onPickAccount)
                }
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
