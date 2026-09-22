package tech.shatrov.familyfinances.ui.categories

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens

/** Подпись поля категории на форме с входом в справочник: без него за новой категорией форму пришлось бы бросить. */
@Composable
internal fun CategoryFieldHeader(
    label: String,
    enabled: Boolean,
    onManage: () -> Unit,
) {
    Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text(label, style = MaterialTheme.typography.bodySmall, modifier = Modifier.weight(1f))
        TextButton(onClick = onManage, enabled = enabled, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
            Text(stringResource(R.string.categories_manage))
        }
    }
}
