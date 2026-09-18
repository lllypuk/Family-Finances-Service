package tech.shatrov.familyfinances.ui.networth

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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.HoldingValue
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.format.formatFullDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import tech.shatrov.familyfinances.ui.settings.SettingsHeader

/** История снимков позиции: тап — лист правки снимка, карандаш — форма позиции. */
@Composable
fun HoldingHistoryScreen(
    state: HoldingHistoryUiState,
    currency: String,
    onRetry: () -> Unit,
    onLoadMore: () -> Unit,
    onOpenValue: (HoldingValue) -> Unit,
    onEdit: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
    ) {
        SettingsHeader(R.string.holding_history_title, enabled = true, onBack = onBack) {
            IconButton(onClick = onEdit, enabled = state is HoldingHistoryUiState.Ready) {
                Icon(AppIcons.Pencil, contentDescription = stringResource(R.string.net_worth_edit_holding))
            }
        }

        when (state) {
            HoldingHistoryUiState.Loading, HoldingHistoryUiState.Gone -> Centered { CircularProgressIndicator() }

            is HoldingHistoryUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                    Text(stringResource(R.string.retry))
                }
            }

            is HoldingHistoryUiState.Ready -> Values(state, currency, onLoadMore, onOpenValue)
        }
    }
}

@Composable
private fun Values(
    state: HoldingHistoryUiState.Ready,
    currency: String,
    onLoadMore: () -> Unit,
    onOpenValue: (HoldingValue) -> Unit,
) {
    val colors = LocalAppColors.current
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(top = Dimens.SPACE_2, bottom = Dimens.SPACE_4),
    ) {
        item(key = "holding") {
            Column(
                modifier = Modifier.padding(bottom = Dimens.SPACE_3),
                verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
            ) {
                Text(text = state.holding.name, style = MaterialTheme.typography.titleMedium)
                Text(
                    text = state.holding.current?.let { formatMoney(it.valueMinor, currency) }
                        ?: stringResource(R.string.net_worth_no_value),
                    style = MaterialTheme.typography.headlineSmall,
                )
            }
        }

        if (state.values.isEmpty()) {
            item(key = "empty") {
                Text(
                    text = stringResource(R.string.holding_history_empty),
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }

        itemsIndexed(state.values, key = { _, it -> it.date.toString() }) { index, value ->
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .groupedRow(rowPlace(index, state.values.size), colors, onClick = { onOpenValue(value) })
                    .heightIn(min = Dimens.TOUCH_MIN),
                horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = formatFullDay(value.date),
                    style = MaterialTheme.typography.bodyLarge,
                    modifier = Modifier.weight(1f),
                )
                Text(text = formatMoney(value.valueMinor, currency), style = MaterialTheme.typography.bodyLarge)
            }
        }

        val moreError = state.moreError
        if (moreError != null) {
            item(key = "more-error") {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = Dimens.SPACE_4),
                ) {
                    Text(
                        text = moreError.message(LocalContext.current.resources),
                        color = MaterialTheme.colorScheme.error,
                    )
                    Button(onClick = onLoadMore, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                        Text(stringResource(R.string.retry))
                    }
                }
            }
        } else if (state.hasMore) {
            item(key = "more") {
                LaunchedEffect(state.values.size) { onLoadMore() }
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
