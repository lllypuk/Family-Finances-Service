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
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
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
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatPercent
import tech.shatrov.familyfinances.ui.format.formatPeriod
import tech.shatrov.familyfinances.ui.message

/** Сколько категорий помещается в карточку: остальное живёт на экране транзакций. */
private const val TOP_CATEGORIES = 5

@Composable
fun HomeScreen(
    state: HomeUiState,
    onRetry: () -> Unit,
    onSignOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = Dimens.SPACE_4, end = Dimens.SPACE_2, top = Dimens.SPACE_2, bottom = Dimens.SPACE_2),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.home_title),
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = onSignOut) {
                Icon(AppIcons.LogOut, contentDescription = stringResource(R.string.sign_out))
            }
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
                    Centered { Text(stringResource(R.string.home_empty)) }
                } else {
                    Summary(state)
                }
        }
    }
}

@Composable
private fun Summary(state: HomeUiState.Ready) {
    val summary = state.summary
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        item {
            Text(
                text = formatPeriod(summary.from, summary.to),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        item {
            TotalsCard(state)
        }

        if (summary.expenseCategories.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_top_categories)) }
            items(summary.expenseCategories.take(TOP_CATEGORIES)) { category ->
                CategoryRow(category, state.currency)
            }
        }

        if (summary.budgets.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_budgets)) }
            items(summary.budgets) { budget -> BudgetRow(budget, state.currency) }
        }

        if (summary.recent.isNotEmpty()) {
            item { SectionTitle(stringResource(R.string.home_recent)) }
            items(summary.recent) { transaction -> RecentRow(transaction, state.currency) }
        }
    }
}

@Composable
private fun TotalsCard(state: HomeUiState.Ready) {
    val colors = LocalAppColors.current
    val summary = state.summary
    // Дельты показываются только когда есть с чем сравнивать: иначе сервер шлёт нули, и «0 %»
    // читалось бы как «ничего не изменилось».
    val delta = { value: Double -> if (summary.hasPreviousData) formatPercent(value, signed = true) else null }
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(Dimens.SPACE_4),
            verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        ) {
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
            TotalRow(
                label = stringResource(R.string.home_net),
                amount = formatMoney(summary.current.netMinor, state.currency, signed = true),
                delta = null,
                color = if (summary.current.netMinor < 0) colors.expense else colors.income,
            )
            Text(
                text = stringResource(R.string.home_transactions, summary.current.transactionCount),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
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
        modifier = Modifier.padding(top = Dimens.SPACE_2),
    )
}

@Composable
private fun CategoryRow(
    category: CategoryShare,
    currency: String,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
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
) {
    val colors = LocalAppColors.current
    Column(verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1)) {
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
) {
    val colors = LocalAppColors.current
    val income = transaction.type == TransactionType.income
    Row(
        modifier = Modifier.fillMaxWidth(),
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
        // Знак ставится текстом, а не только цветом: доход и акцент — один цвет темы.
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
