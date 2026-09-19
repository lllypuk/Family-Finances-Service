package tech.shatrov.familyfinances.ui.overview

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.CategoryShare
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Chip
import tech.shatrov.familyfinances.ui.ChipRow
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatPercent
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.home.TotalRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import tech.shatrov.familyfinances.ui.settings.SettingsHeader
import java.time.YearMonth
import java.util.UUID

@Composable
fun OverviewScreen(
    state: OverviewUiState,
    onBack: () -> Unit,
    onSelect: (OverviewPeriod) -> Unit,
    onRetryMonthly: () -> Unit,
    onRetrySummary: () -> Unit,
    onOpenCategory: (TransactionType, UUID) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Column(
            modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
            verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        ) {
            SettingsHeader(R.string.overview_title, enabled = true, onBack = onBack)
            PeriodChips(state.period, onSelect)
        }
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = Dimens.SPACE_4),
            contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
        ) {
            item { Summary(state, onRetrySummary) }
            item { Bars(state, onRetryMonthly) { onSelect(OverviewPeriod.Month(it)) } }
            val summary = state.summary as? SummaryState.Ready ?: return@LazyColumn
            if (summary.isEmpty) {
                item {
                    Text(
                        text = stringResource(R.string.overview_empty),
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = Dimens.SPACE_6),
                    )
                }
            } else {
                categories(R.string.overview_expense_categories, summary.expenseCategories, state.currency) {
                    onOpenCategory(TransactionType.expense, it)
                }
                categories(R.string.overview_income_categories, summary.incomeCategories, state.currency) {
                    onOpenCategory(TransactionType.income, it)
                }
            }
        }
    }
}

@Composable
private fun PeriodChips(
    period: OverviewPeriod,
    onSelect: (OverviewPeriod) -> Unit,
) {
    ChipRow {
        // Месяц со столбика — состояние, а не выбор: чип с ним статичный, как `MONTH` в фильтре операций.
        if (period is OverviewPeriod.Month) {
            item { Chip(formatMonth(period.month.atDay(1)), selected = true) {} }
        }
        items(OverviewPeriod.chips.size) { index ->
            val chip = OverviewPeriod.chips[index]
            Chip(stringResource(chip.label()), selected = chip == period) { onSelect(chip) }
        }
    }
}

@StringRes
private fun OverviewPeriod.label(): Int = when (this) {
    OverviewPeriod.ThisMonth -> R.string.overview_this_month
    OverviewPeriod.PrevMonth -> R.string.overview_prev_month
    OverviewPeriod.ThreeMonths -> R.string.overview_three_months
    OverviewPeriod.Year -> R.string.overview_year
    is OverviewPeriod.Month -> error("month is not a chip")
}

@Composable
private fun Summary(
    state: OverviewUiState,
    onRetry: () -> Unit,
) {
    when (val summary = state.summary) {
        SummaryState.Loading -> Placeholder { CircularProgressIndicator() }
        is SummaryState.Failure -> Refusal(summary.error, onRetry)
        is SummaryState.Ready -> Totals(summary, state.currency)
    }
}

@Composable
private fun Totals(
    summary: SummaryState.Ready,
    currency: String,
) {
    val colors = LocalAppColors.current
    val totals = summary.totals
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
            text = formatMoney(totals.netMinor, currency, signed = true),
            style = MaterialTheme.typography.displayLarge,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            color = if (totals.netMinor < 0) colors.expense else colors.income,
        )
        TotalRow(
            label = stringResource(R.string.home_income),
            amount = formatMoney(totals.incomeMinor, currency),
            delta = summary.incomeDelta?.let { formatPercent(it, signed = true) },
            color = colors.income,
        )
        TotalRow(
            label = stringResource(R.string.home_expenses),
            amount = formatMoney(totals.expensesMinor, currency),
            delta = summary.expensesDelta?.let { formatPercent(it, signed = true) },
            color = colors.expense,
        )
        if (summary.incomeDelta != null || summary.expensesDelta != null) {
            Text(
                text = stringResource(R.string.overview_delta_caption),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun Bars(
    state: OverviewUiState,
    onRetry: () -> Unit,
    onSelectMonth: (YearMonth) -> Unit,
) {
    Column(modifier = Modifier.padding(top = Dimens.SPACE_6)) {
        when (val monthly = state.monthly) {
            MonthlyState.Loading -> Placeholder { CircularProgressIndicator() }
            is MonthlyState.Failure -> Refusal(monthly.error, onRetry)
            is MonthlyState.Ready -> MonthlyBars(monthly.months, state.range, state.currency, onSelectMonth)
        }
    }
}

private fun LazyListScope.categories(
    @StringRes title: Int,
    categories: List<CategoryShare>,
    currency: String,
    onOpen: (UUID) -> Unit,
) {
    if (categories.isEmpty()) return
    item {
        Text(
            text = stringResource(title),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = Dimens.SPACE_6, bottom = Dimens.SPACE_2),
        )
    }
    itemsIndexed(categories) { index, category ->
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .groupedRow(rowPlace(index, categories.size), LocalAppColors.current) { onOpen(category.categoryId) },
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
                text = formatMoney(category.amountMinor, currency),
                style = MaterialTheme.typography.displaySmall,
            )
            Text(
                text = formatPercent(category.share),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Icon(AppIcons.ChevronRight, contentDescription = null)
        }
    }
}

@Composable
private fun Placeholder(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = Dimens.SPACE_4),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        content()
    }
}

/** Отказ на месте блока: соседний блок и чипы остаются. */
@Composable
private fun Refusal(
    error: UiError,
    onRetry: () -> Unit,
) {
    Placeholder {
        Text(
            text = error.message(LocalContext.current.resources),
            color = MaterialTheme.colorScheme.error,
        )
        Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
            Text(stringResource(R.string.retry))
        }
    }
}
