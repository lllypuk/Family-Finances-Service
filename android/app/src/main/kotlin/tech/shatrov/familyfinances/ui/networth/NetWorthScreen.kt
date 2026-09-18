package tech.shatrov.familyfinances.ui.networth

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
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
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
import tech.shatrov.familyfinances.core.api.Holding
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.core.api.NetWorthMonth
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import java.time.LocalDate
import java.util.UUID

/** Капитал: итог из ряда, группы «Активы» и «Пассивы», свёрнутый архив. Тап — снимок, долгий тап — правка. */
@Composable
fun NetWorthScreen(
    state: NetWorthUiState,
    currency: String,
    today: LocalDate,
    onRetry: () -> Unit,
    onAdd: (HoldingSide) -> Unit,
    onOpenValue: (Holding) -> Unit,
    onEdit: (UUID) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Text(
            text = stringResource(R.string.net_worth_title),
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
        )

        when (state) {
            NetWorthUiState.Loading -> Centered { CircularProgressIndicator() }

            is NetWorthUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                    Text(stringResource(R.string.retry))
                }
            }

            is NetWorthUiState.Ready ->
                if (state.isEmpty) {
                    Centered {
                        Text(stringResource(R.string.net_worth_empty))
                        Button(
                            onClick = { onAdd(HoldingSide.asset) },
                            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                        ) {
                            Text(stringResource(R.string.net_worth_add_asset))
                        }
                        OutlinedButton(
                            onClick = { onAdd(HoldingSide.liability) },
                            modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                        ) {
                            Text(stringResource(R.string.net_worth_add_liability))
                        }
                    }
                } else {
                    Groups(state, currency, today, onOpenValue, onEdit)
                }
        }
    }
}

@Composable
private fun Groups(
    state: NetWorthUiState.Ready,
    currency: String,
    today: LocalDate,
    onOpenValue: (Holding) -> Unit,
    onEdit: (UUID) -> Unit,
) {
    var archiveOpen by rememberSaveable { mutableStateOf(false) }
    val row = RowActions(currency, today, onOpenValue, onEdit, stringResource(R.string.net_worth_edit_holding))

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        contentPadding = PaddingValues(top = Dimens.SPACE_2, bottom = Dimens.FAB_CLEARANCE),
    ) {
        state.latest?.let { latest -> item(key = "total") { Total(latest, currency) } }
        group("assets", R.string.net_worth_assets, state.assets, row)
        group("liabilities", R.string.net_worth_liabilities, state.liabilities, row)
        if (state.archived.isNotEmpty()) {
            item(key = "archive") {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { archiveOpen = !archiveOpen }
                        .heightIn(min = Dimens.TOUCH_MIN)
                        .padding(top = Dimens.SPACE_2),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = stringResource(R.string.net_worth_archived, state.archived.size),
                        style = MaterialTheme.typography.titleMedium,
                        modifier = Modifier.weight(1f),
                    )
                    Icon(if (archiveOpen) AppIcons.ChevronUp else AppIcons.ChevronDown, contentDescription = null)
                }
            }
            if (archiveOpen) rows("archived", state.archived, row)
        }
    }
}

@Composable
private fun Total(
    latest: NetWorthMonth,
    currency: String,
) {
    Column(
        modifier = Modifier.padding(bottom = Dimens.SPACE_3),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
    ) {
        Text(text = formatMoney(latest.netMinor, currency), style = MaterialTheme.typography.headlineMedium)
        Text(
            text = stringResource(
                R.string.net_worth_totals,
                formatMoney(latest.assetsMinor, currency),
                formatMoney(latest.liabilitiesMinor, currency),
            ),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

private class RowActions(
    val currency: String,
    val today: LocalDate,
    val onOpenValue: (Holding) -> Unit,
    val onEdit: (UUID) -> Unit,
    val editLabel: String,
)

private fun LazyListScope.group(
    key: String,
    @StringRes title: Int,
    rows: List<HoldingRow>,
    actions: RowActions,
) {
    if (rows.isEmpty()) return
    item(key = "$key-title") {
        Text(
            text = stringResource(title),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = Dimens.SPACE_3, bottom = Dimens.SPACE_2),
        )
    }
    rows(key, rows, actions)
}

private fun LazyListScope.rows(
    key: String,
    rows: List<HoldingRow>,
    actions: RowActions,
) {
    itemsIndexed(rows, key = { _, it -> "$key-${it.holding.id}" }) { index, row ->
        HoldingItem(row, rowPlace(index, rows.size), actions)
    }
}

@Composable
private fun HoldingItem(
    row: HoldingRow,
    place: RowPlace,
    actions: RowActions,
) {
    val holding = row.holding
    val current = holding.current
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(
                place,
                LocalAppColors.current,
                onClick = { actions.onOpenValue(holding) },
                onLongClick = { actions.onEdit(holding.id) },
                onLongClickLabel = actions.editLabel,
            )
            .heightIn(min = Dimens.TOUCH_MIN),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = holding.name,
                style = MaterialTheme.typography.bodyLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            Text(
                text = current?.let { formatMoney(it.valueMinor, actions.currency) }
                    ?: stringResource(R.string.net_worth_no_value),
                style = MaterialTheme.typography.bodyLarge,
                color = if (current ==
                    null
                ) {
                    MaterialTheme.colorScheme.onSurfaceVariant
                } else {
                    MaterialTheme.colorScheme.onSurface
                },
            )
        }
        val kind = stringResource(row.kind.label)
        Text(
            text = current?.let { "$kind · ${formatDay(it.date, actions.today)}" } ?: kind,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
    }
}
