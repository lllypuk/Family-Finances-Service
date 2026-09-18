package tech.shatrov.familyfinances.ui.transactions

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Account
import tech.shatrov.familyfinances.theme.Dimens
import java.util.UUID

/** Выбор счёта; [noneLabel] — строка для `null`: «Все счета» в фильтре, «Без счёта» в форме. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun AccountSheet(
    accounts: List<Account>,
    selected: UUID?,
    noneLabel: String,
    onSelect: (UUID?) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        AccountSheetContent(accounts, selected, noneLabel) { picked ->
            scope.launch { sheetState.hide() }.invokeOnCompletion { if (!sheetState.isVisible) onSelect(picked) }
        }
    }
}

@Composable
internal fun AccountSheetContent(
    accounts: List<Account>,
    selected: UUID?,
    noneLabel: String,
    onSelect: (UUID?) -> Unit,
) {
    LazyColumn(modifier = Modifier.fillMaxWidth()) {
        item {
            Text(
                text = stringResource(R.string.transaction_account),
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
            )
        }
        item { SheetRow(noneLabel, selected == null) { onSelect(null) } }
        items(accounts, key = { it.id }) { account ->
            SheetRow(account.label(), selected == account.id) { onSelect(account.id) }
        }
    }
}

/** Архивный счёт подписан: в форме он виден только у операции, которая уже к нему привязана. */
@Composable
internal fun Account.label(): String = if (isArchived) stringResource(R.string.account_archived_label, name) else name
