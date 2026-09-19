package tech.shatrov.familyfinances.ui.home

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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.pluralStringResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.BudgetProgress
import tech.shatrov.familyfinances.core.api.CategoryShare
import tech.shatrov.familyfinances.core.api.RecentTransactionItem
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatMonthName
import tech.shatrov.familyfinances.ui.format.formatPercent
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import java.time.LocalDate
import java.time.YearMonth
import java.time.format.DateTimeFormatter
import java.util.UUID

/** Сколько категорий помещается в карточку: остальное живёт на экране транзакций. */
private const val TOP_CATEGORIES = 5

private val importTime = DateTimeFormatter.ofPattern("HH:mm")

@Composable
fun HomeScreen(
    state: HomeUiState,
    onRetry: () -> Unit,
    onAddTransaction: () -> Unit,
    onSettings: () -> Unit,
    modifier: Modifier = Modifier,
    card: ReconciliationCard = ReconciliationCard.Hidden,
    onReconciliation: (YearMonth) -> Unit = {},
    onAccounts: () -> Unit = {},
    importDraft: ImportDraft? = null,
    onResumeImport: (UUID) -> Unit = {},
    onDeleteImport: () -> Unit = {},
    today: LocalDate = LocalDate.now(),
) {
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = Dimens.SPACE_4, end = Dimens.SPACE_2, top = Dimens.SPACE_2, bottom = Dimens.SPACE_2),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(modifier = Modifier.weight(1f)) {
                val ready = state as? HomeUiState.Ready
                Text(
                    text = if (ready == null) {
                        stringResource(R.string.home_title)
                    } else {
                        formatMonth(ready.summary.from)
                    },
                    style = MaterialTheme.typography.headlineSmall,
                )
                if (ready != null) {
                    Text(
                        text = stringResource(R.string.home_through, formatDay(ready.summary.to)),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            IconButton(onClick = onSettings) {
                Icon(AppIcons.User, contentDescription = stringResource(R.string.settings_title))
            }
        }

        // Над сводкой, а не в ней: журнал локальный и нужен и тогда, когда сервер недоступен.
        if (importDraft != null) {
            ImportDraftCard(importDraft, today, onResumeImport, onDeleteImport)
        }

        when (state) {
            HomeUiState.Loading -> Centered { CircularProgressIndicator() }

            is HomeUiState.Failure -> Centered {
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

            is HomeUiState.Ready ->
                if (state.isEmpty) {
                    Centered {
                        Text(stringResource(R.string.home_empty))
                        Button(
                            onClick = onAddTransaction,
                            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                        ) {
                            Text(stringResource(R.string.home_add_transaction))
                        }
                    }
                } else {
                    Summary(state, card, onReconciliation, onAccounts)
                }
        }
    }
}

@Composable
private fun ImportDraftCard(
    draft: ImportDraft,
    today: LocalDate,
    onResume: (UUID) -> Unit,
    onDelete: () -> Unit,
) {
    var confirmShown by rememberSaveable { mutableStateOf(false) }
    Column(
        modifier = Modifier
            .padding(horizontal = Dimens.SPACE_4)
            .fillMaxWidth()
            .groupedRow(RowPlace.ONLY, LocalAppColors.current),
    ) {
        Text(stringResource(R.string.home_import_title), style = MaterialTheme.typography.bodyMedium)
        Text(
            text = importDetails(draft, today),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            TextButton(onClick = { onResume(draft.importId) }) {
                Text(stringResource(R.string.home_import_resume))
            }
            TextButton(onClick = { confirmShown = true }) {
                Text(stringResource(R.string.home_import_delete))
            }
        }
    }

    // Удалённый журнал — это оплаченный ответ распознавания: вернуть его можно только новым вызовом.
    if (confirmShown) {
        AlertDialog(
            onDismissRequest = { confirmShown = false },
            title = { Text(stringResource(R.string.home_import_delete_confirm)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmShown = false
                    onDelete()
                }) {
                    Text(stringResource(R.string.home_import_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmShown = false }) {
                    Text(stringResource(R.string.cancel))
                }
            },
        )
    }
}

@Composable
private fun importDetails(
    draft: ImportDraft,
    today: LocalDate,
): String {
    if (!draft.recognized) {
        return pluralStringResource(R.plurals.home_import_interrupted, draft.images, draft.images)
    }
    val time = draft.updatedAt.format(importTime)
    val day = draft.updatedAt.toLocalDate()
    val at = when (day) {
        today -> stringResource(R.string.home_import_today, time)
        today.minusDays(1) -> stringResource(R.string.home_import_yesterday, time)
        else -> "${formatDay(day, today)} $time"
    }
    return pluralStringResource(R.plurals.home_import_rows, draft.rows, draft.rows, draft.saved, at)
}

@Composable
private fun Summary(
    state: HomeUiState.Ready,
    card: ReconciliationCard,
    onReconciliation: (YearMonth) -> Unit,
    onAccounts: () -> Unit,
) {
    val summary = state.summary
    val topCategories = summary.expenseCategories.take(TOP_CATEGORIES)
    // Строки секции сливаются в одну панель, поэтому зазор между элементами нулевой: воздух
    // между секциями даёт заголовок.
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        item {
            Totals(state)
        }

        if (card is ReconciliationCard.Ready || card is ReconciliationCard.NoAccounts) {
            item { ReconciliationCardRow(card, onReconciliation, onAccounts) }
        }

        if (topCategories.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_top_categories)) }
            itemsIndexed(topCategories) { index, category ->
                CategoryRow(category, state.currency, rowPlace(index, topCategories.size))
            }
        }

        if (summary.budgets.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_budgets)) }
            itemsIndexed(summary.budgets) { index, budget ->
                BudgetRow(budget, state.currency, rowPlace(index, summary.budgets.size))
            }
        }

        if (summary.recent.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_recent)) }
            itemsIndexed(summary.recent) { index, transaction ->
                RecentRow(transaction, state.currency, rowPlace(index, summary.recent.size))
            }
        }
    }
}

@Composable
private fun Totals(state: HomeUiState.Ready) {
    val colors = LocalAppColors.current
    val summary = state.summary
    // Дельты показываются только когда есть с чем сравнивать: иначе сервер шлёт нули, и «0 %»
    // читалось бы как «ничего не изменилось».
    val delta = { value: Double -> if (summary.hasPreviousData) formatPercent(value, signed = true) else null }
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
    ) {
        Text(
            text = stringResource(R.string.home_net),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = formatMoney(summary.current.netMinor, state.currency, signed = true),
            style = MaterialTheme.typography.displayLarge,
            maxLines = 1,
            // Крупная сумма на узком экране не влезает: без многоточия обрезок читался бы как число.
            overflow = TextOverflow.Ellipsis,
            color = if (summary.current.netMinor < 0) colors.expense else colors.income,
        )
        TotalRow(
            label = stringResource(R.string.home_income),
            amount = formatMoney(summary.current.incomeMinor, state.currency),
            delta = delta(summary.incomeDelta),
            color = colors.income,
        )
        TotalRow(
            label = stringResource(R.string.home_expenses),
            amount = formatMoney(summary.current.expensesMinor, state.currency),
            delta = delta(summary.expensesDelta),
            color = colors.expense,
        )
        Text(
            text = stringResource(R.string.home_transactions, summary.current.transactionCount),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun ReconciliationCardRow(
    card: ReconciliationCard,
    onReconciliation: (YearMonth) -> Unit,
    onAccounts: () -> Unit,
) {
    val ready = card as? ReconciliationCard.Ready
    Row(
        modifier = Modifier
            .padding(top = Dimens.SPACE_4)
            .fillMaxWidth()
            .groupedRow(RowPlace.ONLY, LocalAppColors.current) {
                if (ready == null) onAccounts() else onReconciliation(ready.month)
            },
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = if (ready == null) {
                    stringResource(R.string.reconciliation_title)
                } else {
                    stringResource(R.string.home_reconciliation, formatMonthName(ready.month))
                },
                style = MaterialTheme.typography.bodyMedium,
            )
            if (ready == null) {
                Text(
                    text = stringResource(R.string.home_reconciliation_no_accounts),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        if (ready != null) {
            Text(
                text = stringResource(R.string.home_reconciliation_matched, ready.matched, ready.total),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Icon(AppIcons.ChevronRight, contentDescription = null)
    }
}

@Composable
private fun TotalRow(
    label: String,
    amount: String,
    delta: String?,
    color: Color,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text = label, style = MaterialTheme.typography.bodyMedium)
        Row(horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2)) {
            if (delta != null) {
                Text(
                    text = delta,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Text(text = amount, style = MaterialTheme.typography.displaySmall, color = color)
        }
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleMedium,
        modifier = Modifier.padding(top = Dimens.SPACE_6, bottom = Dimens.SPACE_2),
    )
}

@Composable
private fun CategoryRow(
    category: CategoryShare,
    currency: String,
    place: RowPlace,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, LocalAppColors.current),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = category.name,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
        Text(
            text = formatPercent(category.share),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = formatMoney(category.amountMinor, currency),
            style = MaterialTheme.typography.displaySmall,
        )
    }
}

@Composable
private fun BudgetRow(
    budget: BudgetProgress,
    currency: String,
    place: RowPlace,
) {
    val colors = LocalAppColors.current
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, colors),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
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
                    R.string.home_budget_progress,
                    formatMoney(budget.spentMinor, currency),
                    formatMoney(budget.amountMinor, currency),
                ),
                style = MaterialTheme.typography.bodySmall,
            )
        }
        LinearProgressIndicator(
            progress = { budget.utilization.toFloat().coerceIn(0f, 1f) },
            color = when {
                budget.isOverBudget -> colors.expense
                budget.isNearLimit -> colors.warning
                else -> colors.action
            },
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun RecentRow(
    transaction: RecentTransactionItem,
    currency: String,
    place: RowPlace,
) {
    val colors = LocalAppColors.current
    val income = transaction.type == TransactionType.income
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, colors),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = transaction.description,
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                text = listOfNotNull(formatDay(transaction.date), transaction.categoryName).joinToString(" · "),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        // Знак ставится текстом, а не только цветом: ради цветовосприятия.
        Text(
            text = formatMoney(
                if (income) transaction.amountMinor else -transaction.amountMinor,
                currency,
                signed = true,
            ),
            style = MaterialTheme.typography.displaySmall,
            color = if (income) colors.income else colors.expense,
        )
    }
}
