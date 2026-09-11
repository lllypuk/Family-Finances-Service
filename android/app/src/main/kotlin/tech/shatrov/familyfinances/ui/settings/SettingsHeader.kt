package tech.shatrov.familyfinances.ui.settings

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.AppIcons

/**
 * Шапка подраздела настроек. [enabled] выключает «назад» на время отправки: корутина умерла бы
 * вместе с моделью, а запись сервер уже мог принять.
 */
@Composable
internal fun SettingsHeader(
    @StringRes title: Int,
    enabled: Boolean,
    onBack: () -> Unit,
) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = onBack, enabled = enabled) {
            Icon(AppIcons.ArrowLeft, contentDescription = stringResource(R.string.back))
        }
        Text(text = stringResource(title), style = MaterialTheme.typography.headlineSmall)
    }
}
